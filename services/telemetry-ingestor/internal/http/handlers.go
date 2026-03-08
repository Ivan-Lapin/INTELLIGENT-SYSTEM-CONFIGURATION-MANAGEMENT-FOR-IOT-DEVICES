package http

import (
	"net/http"
	"strconv"
	"time"

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
