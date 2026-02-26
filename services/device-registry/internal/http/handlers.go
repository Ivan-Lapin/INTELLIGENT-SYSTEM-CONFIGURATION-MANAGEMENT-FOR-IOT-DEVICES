package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"device-registry/internal/model"
)

type Handlers struct {
	pg *pgxpool.Pool
}

func NewHandlers(pg *pgxpool.Pool) *Handlers {
	return &Handlers{pg: pg}
}

func (h *Handlers) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handlers) CreateDevice(c *gin.Context) {
	var req model.CreateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Tags == nil {
		req.Tags = map[string]any{}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	var id string
	err := h.pg.QueryRow(ctx, `
		INSERT INTO registry.devices (external_id, device_type, protocol, tags)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, req.ExternalID, req.DeviceType, req.Protocol, req.Tags).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handlers) ListDevices(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.pg.Query(ctx, `
		SELECT id, external_id, device_type, protocol, status, tags, last_seen_at, created_at, updated_at
		FROM registry.devices
		ORDER BY created_at DESC
		LIMIT 100
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	devs := make([]model.Device, 0)
	for rows.Next() {
		var d model.Device
		if err := rows.Scan(&d.ID, &d.ExternalID, &d.DeviceType, &d.Protocol, &d.Status, &d.Tags, &d.LastSeenAt, &d.CreatedAt, &d.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		devs = append(devs, d)
	}
	c.JSON(http.StatusOK, devs)
}
