package model

import "time"

type PublishDesiredRequest struct {
	DeviceID        string `json:"deviceId" binding:"required"`
	ConfigVersionID string `json:"configVersionId" binding:"required"`
}

type DesiredMessage struct {
	Version         int            `json:"version"`
	ConfigVersionID string         `json:"configVersionId,omitempty"`
	Checksum        string         `json:"checksum"`
	Payload         map[string]any `json:"payload"`
	TS              time.Time      `json:"ts"`
	Protocol        string         `json:"protocol"`
}

type AckMessage struct {
	DeviceID        string    `json:"deviceId"`
	Version         int       `json:"version"`
	ConfigVersionID string    `json:"configVersionId,omitempty"`
	Status          string    `json:"status"`
	Error           string    `json:"error"`
	TS              time.Time `json:"ts"`
	Protocol        string    `json:"protocol"`
}

type ReportedMessage struct {
	DeviceID        string         `json:"deviceId"`
	Version         int            `json:"version"`
	ConfigVersionID string         `json:"configVersionId,omitempty"`
	State           map[string]any `json:"state"`
	TS              time.Time      `json:"ts"`
	Protocol        string         `json:"protocol"`
}

type TelemetryMessage struct {
	DeviceID        string         `json:"deviceId"`
	Version         int            `json:"version"`
	ConfigVersionID string         `json:"configVersionId,omitempty"`
	Metrics         map[string]any `json:"metrics"`
	TS              time.Time      `json:"ts"`
	Protocol        string         `json:"protocol"`
}
