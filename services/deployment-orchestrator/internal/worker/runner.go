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

	mqttAdapterURL  string
	lwm2mAdapterURL string
	mlServiceURL    string
	twinServiceURL  string

	httpClient *http.Client
}

func NewRunner(pg *store.PG, mqttAdapterURL, lwm2mAdapterURL, mlServiceURL, twinServiceURL string) *Runner {
	return &Runner{
		pg:              pg,
		mqttAdapterURL:  mqttAdapterURL,
		lwm2mAdapterURL: lwm2mAdapterURL,
		mlServiceURL:    mlServiceURL,
		twinServiceURL:  twinServiceURL,
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

	total := len(req.DeviceIds)
	if total == 0 {
		return fmt.Errorf("no deviceIds")
	}

	if req.Strategy.CanaryPercent <= 0 {
		req.Strategy.CanaryPercent = 10
	}
	if req.Strategy.MaxFailRate <= 0 {
		req.Strategy.MaxFailRate = 0.1
	}
	if req.Strategy.AckWaitSec <= 0 {
		req.Strategy.AckWaitSec = 5
	}
	if req.Strategy.PollIntervalMs <= 0 {
		req.Strategy.PollIntervalMs = 500
	}
	if req.Strategy.MaxMLRisk <= 0 {
		req.Strategy.MaxMLRisk = 0.6
	}

	// defaults
	if req.Strategy.CanaryPercent <= 0 {
		req.Strategy.CanaryPercent = 10
	}
	if req.Strategy.MaxFailRate <= 0 {
		req.Strategy.MaxFailRate = 0.1
	}
	if req.Strategy.AckWaitSec <= 0 {
		req.Strategy.AckWaitSec = 10
	}
	if req.Strategy.PollIntervalMs <= 0 {
		req.Strategy.PollIntervalMs = 500
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

	if err := r.pg.InsertTargets(ctx, deploymentID, canary, "CANARY"); err != nil {
		return fmt.Errorf("insert canary targets: %w", err)
	}
	if len(full) > 0 {
		if err := r.pg.InsertTargets(ctx, deploymentID, full, "FULL"); err != nil {
			return fmt.Errorf("insert full targets: %w", err)
		}
	}

	if err := r.pg.SetDeploymentStarted(ctx, deploymentID, "CANARY"); err != nil {
		return fmt.Errorf("set started: %w", err)
	}

	canarySince := time.Now().UTC()
	if err := r.publishBatch(canary, req.ConfigVersionId, deploymentID); err != nil {
		return fmt.Errorf("publish canary: %w", err)
	}

	canApplied, canFailed, canPending, err := r.waitAndEvaluate(canary, req.ConfigVersionId, canarySince, deploymentID, req.Strategy)
	if err != nil {
		return err
	}

	canFailRate := float64(canFailed+canPending) / float64(len(canary))
	log.Printf("[deploy %s] CANARY result applied=%d failed=%d pending=%d failRate=%.3f",
		deploymentID, canApplied, canFailed, canPending, canFailRate)

	if canFailRate > req.Strategy.MaxFailRate {
		for _, d := range canary {
			_ = r.pg.MarkTargetStatus(ctx, deploymentID, d, "ROLLED_BACK", "canary fail-rate exceeded")
		}
		_ = r.pg.SetDeploymentFinished(ctx, deploymentID, "ROLLED_BACK")
		log.Printf("[deploy %s] ROLLED_BACK (canary fail-rate)", deploymentID)
		return nil
	}

	// Для MVP берём первый canary device как representative device
	repDevice := canary[0]

	if req.Strategy.EnableTwin {
		twinResp, err := r.validateWithTwin(repDevice, req.ConfigVersionId)
		if err != nil {
			return fmt.Errorf("digital twin validation failed: %w", err)
		}
		log.Printf("[deploy %s] twin result valid=%v risk=%.3f reason=%s",
			deploymentID, twinResp.Valid, twinResp.RiskScore, twinResp.Reason)

		if !twinResp.Valid {
			for _, d := range canary {
				_ = r.pg.MarkTargetStatus(ctx, deploymentID, d, "ROLLED_BACK", "digital twin rejected config")
			}
			_ = r.pg.SetDeploymentFinished(ctx, deploymentID, "FAILED_POLICY")
			log.Printf("[deploy %s] FAILED_POLICY (digital twin rejected)", deploymentID)
			return nil
		}
	}

	if req.Strategy.EnableML {
		mlResp, err := r.predictWithML(repDevice, 12)
		if err != nil {
			return fmt.Errorf("ml prediction failed: %w", err)
		}
		log.Printf("[deploy %s] ml result risk=%.3f level=%s",
			deploymentID, mlResp.RiskScore, mlResp.RiskLevel)

		if mlResp.RiskScore > req.Strategy.MaxMLRisk {
			for _, d := range canary {
				_ = r.pg.MarkTargetStatus(ctx, deploymentID, d, "ROLLED_BACK", "ml predicted high QoS degradation risk")
			}
			_ = r.pg.SetDeploymentFinished(ctx, deploymentID, "FAILED_POLICY")
			log.Printf("[deploy %s] FAILED_POLICY (ml risk too high)", deploymentID)
			return nil
		}
	}
	// ===== END NEW =====

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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	protocol, err := r.pg.GetDeviceProtocol(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("get device protocol: %w", err)
	}

	body := map[string]string{
		"deviceId":        deviceID,
		"configVersionId": configVersionID,
	}
	b, _ := json.Marshal(body)

	targetURL := r.mqttAdapterURL + "/v1/publish/desired"
	if protocol == "lwm2m" {
		targetURL = r.lwm2mAdapterURL + "/v1/publish/desired"
	}

	req, _ := http.NewRequest("POST", targetURL, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("adapter call: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("adapter status=%d", resp.StatusCode)
	}
	return nil
}

type twinValidateResponse struct {
	Valid     bool    `json:"valid"`
	RiskScore float64 `json:"riskScore"`
	Reason    string  `json:"reason"`
}

func (r *Runner) validateWithTwin(deviceID, configVersionID string) (*twinValidateResponse, error) {
	body := map[string]string{
		"deviceId":        deviceID,
		"configVersionId": configVersionID,
	}
	b, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", r.twinServiceURL+"/v1/validate", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("twin status=%d", resp.StatusCode)
	}

	var out twinValidateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

type mlPredictResponse struct {
	DeviceID   string  `json:"deviceId"`
	WindowSize int     `json:"windowSize"`
	RiskScore  float64 `json:"riskScore"`
	RiskLevel  string  `json:"riskLevel"`
}

func (r *Runner) predictWithML(deviceID string, windowSize int) (*mlPredictResponse, error) {
	body := map[string]any{
		"deviceId":   deviceID,
		"windowSize": windowSize,
	}
	b, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", r.mlServiceURL+"/predict", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ml status=%d", resp.StatusCode)
	}

	var out mlPredictResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
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
				pending++
			}
		}

		if pending == 0 {
			return applied, failed, pending, nil
		}
		if time.Now().After(deadline) {
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
