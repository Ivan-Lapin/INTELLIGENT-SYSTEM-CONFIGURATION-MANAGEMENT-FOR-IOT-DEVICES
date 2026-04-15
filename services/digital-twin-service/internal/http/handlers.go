package http

import (
	"context"
	"net/http"
	"time"

	"digital-twin-service/internal/model"
	"digital-twin-service/internal/store"
	"digital-twin-service/internal/twin"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	pg    *store.PG
	mongo *store.Mongo
}

func NewHandlers(d Deps) *Handlers {
	return &Handlers{
		pg:    d.PG,
		mongo: d.Mongo,
	}
}

func (h *Handlers) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true, "ts": time.Now().UTC()})
}

func (h *Handlers) Validate(c *gin.Context) {
	var req model.ValidateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	latest, err := h.pg.GetLatestTelemetry(ctx, req.DeviceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "latest telemetry not found: " + err.Error()})
		return
	}

	var mongoVersionID string
	err = h.pg.Pool.QueryRow(ctx, `
		SELECT mongo_version_id
		FROM cfg.config_versions
		WHERE id = $1
	`, req.ConfigVersionID).Scan(&mongoVersionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "config version not found: " + err.Error()})
		return
	}

	cfgDoc, err := h.mongo.GetConfigVersion(ctx, mongoVersionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "mongo read failed: " + err.Error()})
		return
	}

	simReq := model.SimulateRequest{
		DeviceID:        req.DeviceID,
		CurrentConfig:   map[string]any{},
		CandidateConfig: cfgDoc.Payload,
		TelemetryWindow: model.TelemetryWindowInput{
			LatencyAvg:    latest.LatencyMs,
			LatencyP95:    latest.LatencyMs,
			PacketLossAvg: latest.Loss,
			JitterAvg:     latest.JitterMs,
			RSSIAvg:       latest.RSSI,
			BatteryLevel:  latest.Battery,
			SampleCount:   1,
			Window:        "latest",
		},
		Deployment: model.DeploymentContextInput{
			RolloutStrategy: "canary",
			CanarySize:      10,
			TargetGroupSize: 10,
		},
	}

	resp := twin.EvaluateSimulation(simReq)

	c.JSON(http.StatusOK, gin.H{
		"valid":     resp.Recommendation != "reject",
		"riskScore": resp.RiskScore,
		"reason":    joinReasons(resp.Reasons),
		"expectedDelta": model.ExpectedDelta{
			LatencyMs: resp.PredictedLatency - latest.LatencyMs,
			Loss:      resp.PredictedPacketLoss - latest.Loss,
			JitterMs:  0,
			Battery:   -resp.BatteryImpact,
		},
		"recommendation": resp.Recommendation,
		"reasons":        resp.Reasons,
		"layerScores":    resp.LayerScores,
	})
}

func (h *Handlers) Simulate(c *gin.Context) {
	var req model.SimulateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp := twin.EvaluateSimulation(req)
	c.JSON(http.StatusOK, resp)
}

func joinReasons(reasons []string) string {
	if len(reasons) == 0 {
		return "configuration is acceptable"
	}
	return reasons[0]
}
