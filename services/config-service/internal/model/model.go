package model

import "time"

type CreateTemplateRequest struct {
	Name         string         `json:"name" binding:"required"`
	DeviceType   string         `json:"deviceType" binding:"required"`
	Schema       map[string]any `json:"schema" binding:"required"`  // JSON schema
	DefaultValue map[string]any `json:"default" binding:"required"` // defaults
}

type CreateVersionRequest struct {
	TemplateID string         `json:"templateId" binding:"required"`
	Payload    map[string]any `json:"payload" binding:"required"`
}

type TemplateMeta struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	DeviceType    string    `json:"deviceType"`
	SchemaVersion int       `json:"schemaVersion"`
	CreatedAt     time.Time `json:"createdAt"`
}

type VersionMeta struct {
	ID         string    `json:"id"`
	TemplateID string    `json:"templateId"`
	Version    int       `json:"version"`
	Checksum   string    `json:"checksum"`
	CreatedAt  time.Time `json:"createdAt"`
}
