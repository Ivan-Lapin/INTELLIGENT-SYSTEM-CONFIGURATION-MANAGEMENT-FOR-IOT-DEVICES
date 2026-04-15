package http

import (
	"errors"
	"net/http"
	"time"

	"deployment-orchestrator/internal/model"
	"deployment-orchestrator/internal/store"
	"deployment-orchestrator/internal/worker"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Handlers struct {
	pg *store.PG
	r  *worker.Runner
}

func NewHandlers(d Deps) *Handlers {
	return &Handlers{pg: d.PG, r: d.R}
}

func (h *Handlers) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true, "ts": time.Now().UTC()})
}

func (h *Handlers) CreateDeployment(c *gin.Context) {
	var req model.CreateDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Strategy.CanaryPercent <= 0 {
		req.Strategy.CanaryPercent = 10
	}
	if req.Strategy.MaxFailRate <= 0 {
		req.Strategy.MaxFailRate = 0.10
	}
	if req.Strategy.AckWaitSec <= 0 {
		req.Strategy.AckWaitSec = 5
	}
	if req.Strategy.PollIntervalMs <= 0 {
		req.Strategy.PollIntervalMs = 500
	}
	if req.Strategy.MaxTwinRisk <= 0 {
		req.Strategy.MaxTwinRisk = 0.8
	}
	if req.Strategy.CanaryTwinRisk <= 0 {
		req.Strategy.CanaryTwinRisk = 0.5
	}
	if req.Strategy.MaxLatencyThreshold <= 0 {
		req.Strategy.MaxLatencyThreshold = 50
	}
	if req.Strategy.MaxPacketLoss <= 0 {
		req.Strategy.MaxPacketLoss = 0.05
	}
	if req.Strategy.MaxOfflineRate <= 0 {
		req.Strategy.MaxOfflineRate = 0.2
	}

	id, err := h.pg.CreateDeployment(c.Request.Context(), req.ConfigVersionId, req.Strategy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create deployment: " + err.Error()})
		return
	}

	h.r.Start(id, req)

	c.JSON(http.StatusCreated, model.CreateDeploymentResponse{
		DeploymentId: id,
		Status:       "CREATED",
	})
}

func (h *Handlers) GetDeployment(c *gin.Context) {
	id := c.Param("id")

	cv, st, startedAt, finishedAt, err := h.pg.GetDeploymentStatus(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "deployment not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "get deployment: " + err.Error()})
			return
		}
	}

	total, canary, full, byStatus, err := h.pg.GetTargetCounts(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "counts: " + err.Error()})
		return
	}

	resp := model.DeploymentStatusResponse{
		DeploymentId:    id,
		ConfigVersionId: cv,
		Status:          st,
	}
	if startedAt != nil {
		s := startedAt.UTC().Format(time.RFC3339)
		resp.StartedAt = &s
	}
	if finishedAt != nil {
		s := finishedAt.UTC().Format(time.RFC3339)
		resp.FinishedAt = &s
	}

	resp.Counts.Total = total
	resp.Counts.Canary = canary
	resp.Counts.Full = full
	resp.Counts.Applied = byStatus["APPLIED"]
	resp.Counts.Failed = byStatus["FAILED"]
	resp.Counts.Pending = byStatus["PENDING"]
	resp.Counts.Sent = byStatus["SENT"]
	resp.Counts.RolledBack = byStatus["ROLLED_BACK"]

	c.JSON(http.StatusOK, resp)
}
