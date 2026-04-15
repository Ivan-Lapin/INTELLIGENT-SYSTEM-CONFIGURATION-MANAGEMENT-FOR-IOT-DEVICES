package http

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"config-service/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Handlers struct {
	pg  *pgxpool.Pool
	mdb *mongo.Database
}

func NewHandlers(pg *pgxpool.Pool, mdb *mongo.Database) *Handlers {
	return &Handlers{pg: pg, mdb: mdb}
}

func (h *Handlers) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handlers) CreateTemplate(c *gin.Context) {
	var req model.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// 1) Mongo doc
	doc := bson.M{
		"name":       req.Name,
		"deviceType": req.DeviceType,
		"schema":     req.Schema,
		"default":    req.DefaultValue,
		"createdAt":  time.Now().UTC(),
		"schemaVer":  1,
	}
	res, err := h.mdb.Collection("config_templates").InsertOne(ctx, doc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 2) Postgres meta
	var id string
	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "mongo inserted id is not ObjectID"})
		return
	}
	mongoHex := oid.Hex()
	err = h.pg.QueryRow(ctx, `
		INSERT INTO cfg.config_templates (name, device_type, schema_version, mongo_template_id)
		VALUES ($1, $2, 1, $3)
		RETURNING id
	`, req.Name, req.DeviceType, mongoHex).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handlers) ListTemplates(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.pg.Query(ctx, `
		SELECT id, name, device_type, schema_version, created_at
		FROM cfg.config_templates
		ORDER BY created_at DESC
		LIMIT 100
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	out := make([]model.TemplateMeta, 0)
	for rows.Next() {
		var t model.TemplateMeta
		if err := rows.Scan(&t.ID, &t.Name, &t.DeviceType, &t.SchemaVersion, &t.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out = append(out, t)
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handlers) CreateVersion(c *gin.Context) {
	var req model.CreateVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Status == "" {
		req.Status = "draft"
	}
	if req.RolloutStrategy == "" {
		req.RolloutStrategy = "canary"
	}

	validation := validateBusinessRules(req.Payload)
	if !validation.Valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "business validation failed",
			"validation": validation,
		})
		return
	}

	raw, _ := json.Marshal(req.Payload)
	sum := sha256.Sum256(raw)
	checksum := hex.EncodeToString(sum[:])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	tx, err := h.pg.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pg begin: " + err.Error()})
		return
	}
	defer tx.Rollback(ctx)

	var templateExists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM cfg.config_templates WHERE id = $1 FOR UPDATE)
	`, req.TemplateID).Scan(&templateExists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pg lock template: " + err.Error()})
		return
	}
	if !templateExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "template not found"})
		return
	}

	var next int
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(version), 0) + 1
		FROM cfg.config_versions
		WHERE template_id = $1
	`, req.TemplateID).Scan(&next)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pg next version: " + err.Error()})
		return
	}

	parentVersion := req.ParentVersion
	if parentVersion == nil && next > 1 {
		prev := next - 1
		parentVersion = &prev
	}

	var oldPayload map[string]any
	if parentVersion != nil {
		parentDocID := fmt.Sprintf("%s:%d", req.TemplateID, *parentVersion)
		var prevDoc bson.M
		err := h.mdb.Collection("config_versions").FindOne(ctx, bson.M{"_id": parentDocID}).Decode(&prevDoc)
		if err == nil {
			if payload, ok := prevDoc["payload"].(map[string]any); ok {
				oldPayload = payload
			}
		}
	}
	if oldPayload == nil {
		oldPayload = map[string]any{}
	}

	diff := buildDiff(oldPayload, req.Payload)
	mongoDocID := fmt.Sprintf("%s:%d", req.TemplateID, next)

	doc := bson.M{
		"_id":           mongoDocID,
		"templateId":    req.TemplateID,
		"version":       next,
		"parentVersion": parentVersion,
		"payload":       req.Payload,
		"diff":          diff,
		"checksum":      checksum,
		"createdAt":     time.Now().UTC(),
	}

	_, mongoErr := h.mdb.Collection("config_versions").InsertOne(ctx, doc)
	if mongoErr != nil && !isMongoDuplicateKey(mongoErr) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "mongo insert: " + mongoErr.Error()})
		return
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO cfg.config_versions (
			template_id,
			version,
			checksum,
			mongo_version_id,
			parent_version,
			status,
			created_by,
			rollout_strategy,
			change_summary
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (template_id, version)
		DO UPDATE SET
			checksum = EXCLUDED.checksum,
			mongo_version_id = EXCLUDED.mongo_version_id,
			parent_version = EXCLUDED.parent_version,
			status = EXCLUDED.status,
			created_by = EXCLUDED.created_by,
			rollout_strategy = EXCLUDED.rollout_strategy,
			change_summary = EXCLUDED.change_summary
		RETURNING id
	`,
		req.TemplateID,
		next,
		checksum,
		mongoDocID,
		parentVersion,
		req.Status,
		req.CreatedBy,
		req.RolloutStrategy,
		req.ChangeSummary,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pg insert: " + err.Error()})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pg commit: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         id,
		"version":    next,
		"checksum":   checksum,
		"validation": validation,
		"diff":       diff,
	})
}

func (h *Handlers) GetVersion(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var meta model.VersionDetails
	var mongoVersionID string

	err := h.pg.QueryRow(ctx, `
		SELECT
			id,
			template_id,
			version,
			checksum,
			status,
			created_by,
			parent_version,
			rollout_strategy,
			change_summary,
			mongo_version_id,
			created_at
		FROM cfg.config_versions
		WHERE id = $1
	`, id).Scan(
		&meta.ID,
		&meta.TemplateID,
		&meta.Version,
		&meta.Checksum,
		&meta.Status,
		&meta.CreatedBy,
		&meta.ParentVersion,
		&meta.RolloutStrategy,
		&meta.ChangeSummary,
		&mongoVersionID,
		&meta.CreatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
		return
	}

	var doc bson.M
	err = h.mdb.Collection("config_versions").FindOne(ctx, bson.M{"_id": mongoVersionID}).Decode(&doc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "mongo version not found"})
		return
	}

	if payload, ok := doc["payload"].(map[string]any); ok {
		meta.Payload = payload
	}
	if diff, ok := doc["diff"].(map[string]any); ok {
		meta.Diff = diff
	}

	c.JSON(http.StatusOK, meta)
}

// isMongoDuplicateKey: проверка E11000
func isMongoDuplicateKey(err error) bool {
	var we mongo.WriteException
	if errors.As(err, &we) {
		for _, e := range we.WriteErrors {
			if e.Code == 11000 {
				return true
			}
		}
	}
	return false
}

func (h *Handlers) ListVersions(c *gin.Context) {
	templateID := c.Query("templateId")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	q := `
		SELECT
		id,
		template_id,
		version,
		checksum,
		status,
		created_by,
		parent_version,
		rollout_strategy,
		change_summary,
		created_at
		FROM cfg.config_versions
	`
	args := []any{}
	if templateID != "" {
		q += " WHERE template_id = $1"
		args = append(args, templateID)
	}
	q += " ORDER BY created_at DESC LIMIT 100"

	rows, err := h.pg.Query(ctx, q, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	out := make([]model.VersionMeta, 0)
	for rows.Next() {
		var v model.VersionMeta
		if err := rows.Scan(&v.ID,
			&v.TemplateID,
			&v.Version,
			&v.Checksum,
			&v.Status,
			&v.CreatedBy,
			&v.ParentVersion,
			&v.RolloutStrategy,
			&v.ChangeSummary,
			&v.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out = append(out, v)
	}
	c.JSON(http.StatusOK, out)
}

func toString(v any) string {
	// Mongo ObjectID печатается нормально через %v
	return fmt.Sprintf("%v", v)
}

func buildDiff(oldPayload, newPayload map[string]any) map[string]any {
	diff := make(map[string]any)

	for k, newVal := range newPayload {
		oldVal, exists := oldPayload[k]
		if !exists {
			diff[k] = map[string]any{
				"old": nil,
				"new": newVal,
			}
			continue
		}
		if fmt.Sprintf("%v", oldVal) != fmt.Sprintf("%v", newVal) {
			diff[k] = map[string]any{
				"old": oldVal,
				"new": newVal,
			}
		}
	}

	for k, oldVal := range oldPayload {
		if _, exists := newPayload[k]; !exists {
			diff[k] = map[string]any{
				"old": oldVal,
				"new": nil,
			}
		}
	}

	return diff
}

func validateBusinessRules(payload map[string]any) model.ValidationResult {
	res := model.ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	if v, ok := payload["report_interval"]; ok {
		switch val := v.(type) {
		case float64:
			if val < 1 {
				res.Valid = false
				res.Errors = append(res.Errors, "report_interval must be >= 1")
			}
			if val < 5 {
				res.Warnings = append(res.Warnings, "very low report_interval may increase network load")
			}
		case int:
			if val < 1 {
				res.Valid = false
				res.Errors = append(res.Errors, "report_interval must be >= 1")
			}
			if val < 5 {
				res.Warnings = append(res.Warnings, "very low report_interval may increase network load")
			}
		}
	}

	if v, ok := payload["retry_limit"]; ok {
		switch val := v.(type) {
		case float64:
			if val < 0 || val > 10 {
				res.Valid = false
				res.Errors = append(res.Errors, "retry_limit must be between 0 and 10")
			}
		case int:
			if val < 0 || val > 10 {
				res.Valid = false
				res.Errors = append(res.Errors, "retry_limit must be between 0 and 10")
			}
		}
	}

	return res
}

func (h *Handlers) UpdateVersionStatus(c *gin.Context) {
	id := c.Param("id")

	var req model.UpdateVersionStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	cmdTag, err := h.pg.Exec(ctx, `
		UPDATE cfg.config_versions
		SET status = $2
		WHERE id = $1
	`, id, req.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if cmdTag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"updated": true})
}

func (h *Handlers) ValidateVersion(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var mongoVersionID string
	err := h.pg.QueryRow(ctx, `
		SELECT mongo_version_id
		FROM cfg.config_versions
		WHERE id = $1
	`, id).Scan(&mongoVersionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
		return
	}

	var doc bson.M
	err = h.mdb.Collection("config_versions").FindOne(ctx, bson.M{"_id": mongoVersionID}).Decode(&doc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "mongo version not found"})
		return
	}

	payload, _ := doc["payload"].(map[string]any)
	result := validateBusinessRules(payload)

	c.JSON(http.StatusOK, result)
}
