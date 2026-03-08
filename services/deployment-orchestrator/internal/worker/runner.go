package worker

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"time"

	"deployment-orchestrator/internal/model"
	"deployment-orchestrator/internal/store"
)

type Runner struct {
	pg *store.PG

	mqttAdapterURL string
	httpClient     *http.Client
}

func NewRunner(pg *store.PG, mqttAdapterURL string) *Runner {
	return &Runner{
		pg:             pg,
		mqttAdapterURL: mqttAdapterURL,
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (r *Runner) Start(deploymentID string, req model.CreateDeploymentRequest) {
	go func() {
		if err := r.run(deploymentID, req); err != nil {
			log.Printf("[deploy %s] FAILED: %v", deploymentID, err)
			_ = r.pg.SetDeploymentFinished(context.Background(), deploymentID, "FAILED")
		}
	}()
}

func (r *Runner) run(deploymentID string, req model.CreateDeploymentRequest) error {
	ctx := context.Background()

	// 0) split canary / full
	total := len(req.DeviceIds)
	if total == 0 {
		return fmt.Errorf("no deviceIds")
	}

	canaryN := int(math.Ceil(float64(total) * float64(req.Strategy.CanaryPercent) / 100.0))
	if canaryN < 1 {
		canaryN = 1
	}
	if canaryN > total {
		canaryN = total
	}

	devs := make([]string, 0, total)
	devs = append(devs, req.DeviceIds...)

	shuffleDeterministic(devs, deploymentID)

	canary := devs[:canaryN]
	full := devs[canaryN:]

	// 1) persist targets
	if err := r.pg.InsertTargets(ctx, deploymentID, canary, "CANARY"); err != nil {
		return fmt.Errorf("insert canary targets: %w", err)
	}
	if len(full) > 0 {
		if err := r.pg.InsertTargets(ctx, deploymentID, full, "FULL"); err != nil {
			return fmt.Errorf("insert full targets: %w", err)
		}
	}

	// 2) start canary
	if err := r.pg.SetDeploymentStarted(ctx, deploymentID, "CANARY"); err != nil {
		return fmt.Errorf("set started: %w", err)
	}

	canarySince := time.Now().UTC()
	if err := r.publishBatch(canary, req.ConfigVersionId, deploymentID); err != nil {
		return fmt.Errorf("publish canary: %w", err)
	}

	// 3) wait canary ACK
	canApplied, canFailed, canPending, err := r.waitAndEvaluate(canary, req.ConfigVersionId, canarySince, deploymentID, req.Strategy)
	if err != nil {
		return err
	}

	canFailRate := float64(canFailed+canPending) / float64(len(canary))
	log.Printf("[deploy %s] CANARY result applied=%d failed=%d pending=%d failRate=%.3f",
		deploymentID, canApplied, canFailed, canPending, canFailRate)

	if canFailRate > req.Strategy.MaxFailRate {
		// soft rollback
		for _, d := range canary {
			_ = r.pg.MarkTargetStatus(ctx, deploymentID, d, "ROLLED_BACK", "canary fail-rate exceeded")
		}
		_ = r.pg.SetDeploymentFinished(ctx, deploymentID, "ROLLED_BACK")
		log.Printf("[deploy %s] ROLLED_BACK (soft)", deploymentID)
		return nil
	}

	// 4) full rollout
	if err := r.pg.UpdateDeploymentStatus(ctx, deploymentID, "FULL"); err != nil {
		return fmt.Errorf("set FULL: %w", err)
	}
	if len(full) == 0 {
		_ = r.pg.SetDeploymentFinished(ctx, deploymentID, "DONE")
		log.Printf("[deploy %s] DONE (only canary)", deploymentID)
		return nil
	}

	fullSince := time.Now().UTC()
	if err := r.publishBatch(full, req.ConfigVersionId, deploymentID); err != nil {
		return fmt.Errorf("publish full: %w", err)
	}

	fApplied, fFailed, fPending, err := r.waitAndEvaluate(full, req.ConfigVersionId, fullSince, deploymentID, req.Strategy)
	if err != nil {
		return err
	}

	fullFailRate := float64(fFailed+fPending) / float64(len(full))
	log.Printf("[deploy %s] FULL result applied=%d failed=%d pending=%d failRate=%.3f",
		deploymentID, fApplied, fFailed, fPending, fullFailRate)

	// policy: если full failRate превышает порог — FAILED (можно тоже soft rollback)
	if fullFailRate > req.Strategy.MaxFailRate {
		_ = r.pg.SetDeploymentFinished(ctx, deploymentID, "FAILED")
		return nil
	}

	_ = r.pg.SetDeploymentFinished(ctx, deploymentID, "DONE")
	log.Printf("[deploy %s] DONE", deploymentID)
	return nil
}

func (r *Runner) publishBatch(deviceIDs []string, configVersionID, deploymentID string) error {
	for _, d := range deviceIDs {
		if err := r.publishDesired(d, configVersionID); err != nil {
			_ = r.pg.MarkTargetStatus(context.Background(), deploymentID, d, "FAILED", err.Error())
			continue
		}
		_ = r.pg.MarkTargetStatus(context.Background(), deploymentID, d, "SENT", "")
	}
	return nil
}

func (r *Runner) publishDesired(deviceID, configVersionID string) error {
	body := map[string]string{
		"deviceId":        deviceID,
		"configVersionId": configVersionID,
	}
	b, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", r.mqttAdapterURL+"/v1/publish/desired", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("mqtt-adapter call: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("mqtt-adapter status=%d", resp.StatusCode)
	}
	return nil
}

func (r *Runner) waitAndEvaluate(deviceIDs []string, configVersionID string, since time.Time, deploymentID string, strat model.Strategy) (applied, failed, pending int, err error) {
	ctx := context.Background()

	deadline := time.Now().Add(time.Duration(strat.AckWaitSec) * time.Second)
	poll := time.Duration(strat.PollIntervalMs) * time.Millisecond

	for {
		stMap, errMap, e := r.pg.GetLastApplyStatuses(ctx, since, deviceIDs, configVersionID)
		if e != nil {
			return 0, 0, 0, fmt.Errorf("read apply statuses: %w", e)
		}

		applied = 0
		failed = 0
		pending = 0

		for _, d := range deviceIDs {
			st, ok := stMap[d]
			if !ok {
				pending++
				continue
			}
			switch st {
			case "applied":
				applied++
				_ = r.pg.MarkTargetStatus(ctx, deploymentID, d, "APPLIED", "")
			case "failed":
				failed++
				_ = r.pg.MarkTargetStatus(ctx, deploymentID, d, "FAILED", errMap[d])
			case "sent":
				pending++
				_ = r.pg.MarkTargetStatus(ctx, deploymentID, d, "SENT", "")
			default:
				// unknown -> pending
				pending++
			}
		}

		if pending == 0 {
			return applied, failed, pending, nil
		}
		if time.Now().After(deadline) {
			// timeout: pending считаем как pending (в политике fail-rate они учитываются)
			return applied, failed, pending, nil
		}
		time.Sleep(poll)
	}
}

func shuffleDeterministic(a []string, seedStr string) {
	h := sha1.Sum([]byte(seedStr))
	seed := int64(0)
	for i := 0; i < 8; i++ {
		seed = (seed << 8) | int64(h[i])
	}
	r := rand.New(rand.NewSource(seed))
	r.Shuffle(len(a), func(i, j int) { a[i], a[j] = a[j], a[i] })
}
