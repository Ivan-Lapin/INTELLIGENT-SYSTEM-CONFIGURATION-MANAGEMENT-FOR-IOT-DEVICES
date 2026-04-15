package http

import (
	"net/http"
	"strconv"
	"time"

	"telemetry-ingestor/internal/model"
	"telemetry-ingestor/internal/store"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	pg *store.PG
}

func NewHandlers(d Deps) *Handlers {
	return &Handlers{pg: d.PG}
}

func (h *Handlers) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"ts": time.Now().UTC(),
	})
}

func (h *Handlers) GetRecentTelemetry(c *gin.Context) {
	deviceID := c.Param("deviceId")
	limitStr := c.DefaultQuery("limit", "20")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
		return
	}

	rows, err := h.pg.GetRecentTelemetry(c.Request.Context(), deviceID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query telemetry: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"deviceId": deviceID,
		"count":    len(rows),
		"items":    rows,
	})
}

func (h *Handlers) GetTelemetryAggregations(c *gin.Context) {
	deviceID := c.Param("deviceId")

	var rolloutID *string
	if v := c.Query("rolloutId"); v != "" {
		rolloutID = &v
	}

	var rolloutPhase *string
	if v := c.Query("phase"); v != "" {
		switch v {
		case "before", "canary", "after":
			rolloutPhase = &v
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid phase"})
			return
		}
	}

	aggs, err := h.pg.GetAggregationsForStandardWindows(
		c.Request.Context(),
		deviceID,
		rolloutID,
		rolloutPhase,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "aggregate telemetry: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"deviceId": deviceID,
		"items":    aggs,
	})
}

func (h *Handlers) CreateAnomalyEvent(c *gin.Context) {
	var req model.AnomalyEvent
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.DeviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "deviceId is required"})
		return
	}
	if req.EventType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "eventType is required"})
		return
	}
	if req.Severity == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "severity is required"})
		return
	}

	if req.TS.IsZero() {
		req.TS = time.Now().UTC()
	}

	if err := h.pg.InsertAnomalyEvent(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "insert anomaly: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"ok": true})
}

func (h *Handlers) GetRecentAnomalies(c *gin.Context) {
	deviceID := c.Param("deviceId")
	limitStr := c.DefaultQuery("limit", "20")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
		return
	}

	rows, err := h.pg.GetRecentAnomalies(c.Request.Context(), deviceID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query anomalies: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"deviceId": deviceID,
		"count":    len(rows),
		"items":    rows,
	})
}
