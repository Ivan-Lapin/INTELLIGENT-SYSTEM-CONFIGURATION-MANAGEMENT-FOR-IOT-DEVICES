package http

import (
	"context"
	"net/http"
	"time"

	"lwm2m-adapter/internal/model"
	"lwm2m-adapter/internal/store"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	pg    *store.PG
	mongo *store.Mongo
	mqtt  MQTTPublisher
}

func NewHandlers(d Deps) *Handlers {
	return &Handlers{pg: d.PG, mongo: d.Mongo, mqtt: d.MQTT}
}

func (h *Handlers) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handlers) PublishDesired(c *gin.Context) {
	var req model.PublishDesiredRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	var mongoVersionID string
	var checksum string
	err := h.pg.Pool.QueryRow(ctx, `
		SELECT mongo_version_id, checksum
		FROM cfg.config_versions
		WHERE id = $1
	`, req.ConfigVersionID).Scan(&mongoVersionID, &checksum)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "config version not found: " + err.Error()})
		return
	}

	doc, err := h.mongo.GetConfigPayloadByMongoID(ctx, mongoVersionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "mongo read: " + err.Error()})
		return
	}

	desired := model.DesiredMessage{
		DeviceID:  req.DeviceID,
		VersionID: req.ConfigVersionID,
		Checksum:  checksum,
		Payload:   doc.Payload,
		TS:        time.Now().UTC(),
		Protocol:  "lwm2m",
	}

	if err := h.pg.InsertApplyLogSent(ctx, req.DeviceID, req.ConfigVersionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "apply_log sent: " + err.Error()})
		return
	}
	_ = h.pg.UpsertAssignment(ctx, "device", req.DeviceID, req.ConfigVersionID, "desired")

	topic := "lwm2m/desired/" + req.DeviceID
	if err := h.mqtt.PublishJSON(topic, 1, true, desired); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "mqtt publish: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"published": true,
		"topic":     topic,
		"protocol":  "lwm2m",
	})
}
