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
		INSERT INTO registry.devices (
			external_id,
			device_type,
			protocol,
			network_profile,
			location_zone,
			health_score,
			battery_level,
			current_config_version,
			last_successful_version,
			last_rollout_status,
			tags
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`,
		req.ExternalID,
		req.DeviceType,
		req.Protocol,
		req.NetworkProfile,
		req.LocationZone,
		req.HealthScore,
		req.BatteryLevel,
		req.CurrentConfigVersion,
		req.LastSuccessfulVersion,
		req.LastRolloutStatus,
		req.Tags,
	).Scan(&id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id": id,
	})
}

func (h *Handlers) ListDevices(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.pg.Query(ctx, `
		SELECT
			id,
			external_id,
			device_type,
			protocol,
			status,
			network_profile,
			location_zone,
			health_score,
			battery_level,
			current_config_version,
			last_successful_version,
			last_rollout_status,
			tags,
			last_seen_at,
			created_at,
			updated_at
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
		if err := rows.Scan(
			&d.ID,
			&d.ExternalID,
			&d.DeviceType,
			&d.Protocol,
			&d.Status,
			&d.NetworkProfile,
			&d.LocationZone,
			&d.HealthScore,
			&d.BatteryLevel,
			&d.CurrentConfigVersion,
			&d.LastSuccessfulVersion,
			&d.LastRolloutStatus,
			&d.Tags,
			&d.LastSeenAt,
			&d.CreatedAt,
			&d.UpdatedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		devs = append(devs, d)
	}

	c.JSON(http.StatusOK, devs)
}

func (h *Handlers) UpdateDeviceState(c *gin.Context) {
	deviceID := c.Param("id")

	var req model.UpdateDeviceStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	cmdTag, err := h.pg.Exec(ctx, `
		UPDATE registry.devices
		SET
			status = COALESCE($2, status),
			health_score = COALESCE($3, health_score),
			battery_level = COALESCE($4, battery_level),
			current_config_version = COALESCE($5, current_config_version),
			last_successful_version = COALESCE($6, last_successful_version),
			last_rollout_status = COALESCE($7, last_rollout_status),
			last_seen_at = COALESCE($8, last_seen_at),
			updated_at = NOW()
		WHERE id = $1
	`,
		deviceID,
		req.Status,
		req.HealthScore,
		req.BatteryLevel,
		req.CurrentConfigVersion,
		req.LastSuccessfulVersion,
		req.LastRolloutStatus,
		req.LastSeenAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if cmdTag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"updated": true})
}
