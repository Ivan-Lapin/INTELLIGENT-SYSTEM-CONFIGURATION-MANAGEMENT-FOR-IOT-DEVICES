package model

import "time"

type PublishDesiredRequest struct {
	DeviceID        string `json:"deviceId" binding:"required"`
	ConfigVersionID string `json:"configVersionId" binding:"required"`
}

type DesiredMessage struct {
	DeviceID  string         `json:"deviceId"`
	VersionID string         `json:"versionId"`
	Checksum  string         `json:"checksum"`
	Payload   map[string]any `json:"payload"`
	TS        time.Time      `json:"ts"`
}

type AckMessage struct {
	DeviceID  string    `json:"deviceId"`
	VersionID string    `json:"versionId"`
	Status    string    `json:"status"` // APPLIED|FAILED
	Error     string    `json:"error"`
	TS        time.Time `json:"ts"`
}

type ReportedMessage struct {
	DeviceID  string         `json:"deviceId"`
	VersionID string         `json:"versionId"`
	State     map[string]any `json:"state"`
	TS        time.Time      `json:"ts"`
}
