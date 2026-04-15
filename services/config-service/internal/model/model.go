package model

import "time"

type CreateTemplateRequest struct {
	Name         string         `json:"name" binding:"required"`
	DeviceType   string         `json:"deviceType" binding:"required"`
	Schema       map[string]any `json:"schema" binding:"required"`
	DefaultValue map[string]any `json:"default" binding:"required"`
}

type CreateVersionRequest struct {
	TemplateID      string         `json:"templateId" binding:"required"`
	Payload         map[string]any `json:"payload" binding:"required"`
	CreatedBy       string         `json:"createdBy"`
	ParentVersion   *int           `json:"parentVersion,omitempty"`
	RolloutStrategy string         `json:"rolloutStrategy,omitempty"` // full / canary
	ChangeSummary   string         `json:"changeSummary,omitempty"`
	Status          string         `json:"status,omitempty"` // draft / validated / approved
}

type UpdateVersionStatusRequest struct {
	Status string `json:"status" binding:"required"` // draft / validated / approved / deprecated
}

type TemplateMeta struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	DeviceType    string    `json:"deviceType"`
	SchemaVersion int       `json:"schemaVersion"`
	CreatedAt     time.Time `json:"createdAt"`
}

type VersionMeta struct {
	ID              string    `json:"id"`
	TemplateID      string    `json:"templateId"`
	Version         int       `json:"version"`
	Checksum        string    `json:"checksum"`
	Status          string    `json:"status"`
	CreatedBy       string    `json:"createdBy,omitempty"`
	ParentVersion   *int      `json:"parentVersion,omitempty"`
	RolloutStrategy string    `json:"rolloutStrategy,omitempty"`
	ChangeSummary   string    `json:"changeSummary,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

type VersionDetails struct {
	ID              string         `json:"id"`
	TemplateID      string         `json:"templateId"`
	Version         int            `json:"version"`
	Checksum        string         `json:"checksum"`
	Status          string         `json:"status"`
	CreatedBy       string         `json:"createdBy,omitempty"`
	ParentVersion   *int           `json:"parentVersion,omitempty"`
	RolloutStrategy string         `json:"rolloutStrategy,omitempty"`
	ChangeSummary   string         `json:"changeSummary,omitempty"`
	Diff            map[string]any `json:"diff,omitempty"`
	Payload         map[string]any `json:"payload"`
	CreatedAt       time.Time      `json:"createdAt"`
}

type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}
