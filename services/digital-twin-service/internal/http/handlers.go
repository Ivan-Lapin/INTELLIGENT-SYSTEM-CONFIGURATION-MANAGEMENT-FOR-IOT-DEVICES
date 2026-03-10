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

	resp := twin.Evaluate(latest, cfgDoc.Payload)
	c.JSON(http.StatusOK, resp)
}
