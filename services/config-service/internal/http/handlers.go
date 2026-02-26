package http

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"config-service/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/bson"
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
	mongoID := res.InsertedID

	// 2) Postgres meta
	var id string
	err = h.pg.QueryRow(ctx, `
		INSERT INTO cfg.config_templates (name, device_type, schema_version, mongo_template_id)
		VALUES ($1, $2, 1, $3)
		RETURNING id
	`, req.Name, req.DeviceType, toString(mongoID)).Scan(&id)
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

	// checksum от payload (детерминированно)
	raw, _ := json.Marshal(req.Payload)
	sum := sha256.Sum256(raw)
	checksum := hex.EncodeToString(sum[:])

	ctx, cancel := context.WithTimeout(c.Request.Context(), 7*time.Second)
	defer cancel()

	// 1) вычислим следующий version для template
	var next int
	err := h.pg.QueryRow(ctx, `
		SELECT COALESCE(MAX(version), 0) + 1
		FROM cfg.config_versions
		WHERE template_id = $1
	`, req.TemplateID).Scan(&next)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 2) Mongo doc
	doc := bson.M{
		"templateId": req.TemplateID,
		"version":    next,
		"payload":    req.Payload,
		"checksum":   checksum,
		"createdAt":  time.Now().UTC(),
	}
	res, err := h.mdb.Collection("config_versions").InsertOne(ctx, doc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 3) Postgres meta
	var id string
	err = h.pg.QueryRow(ctx, `
		INSERT INTO cfg.config_versions (template_id, version, checksum, mongo_version_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, req.TemplateID, next, checksum, toString(res.InsertedID)).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":       id,
		"version":  next,
		"checksum": checksum,
	})
}

func (h *Handlers) ListVersions(c *gin.Context) {
	templateID := c.Query("templateId")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	q := `
		SELECT id, template_id, version, checksum, created_at
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
		if err := rows.Scan(&v.ID, &v.TemplateID, &v.Version, &v.Checksum, &v.CreatedAt); err != nil {
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
