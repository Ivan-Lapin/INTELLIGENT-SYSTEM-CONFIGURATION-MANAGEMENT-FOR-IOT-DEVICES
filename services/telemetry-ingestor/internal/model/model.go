package model

import "time"

type TelemetryEvent struct {
	DeviceID        string           `json:"deviceId"`
	TS              time.Time        `json:"ts"`
	Metrics         TelemetryMetrics `json:"metrics"`
	ConfigVersionID *string          `json:"configVersionId,omitempty"`
}

type TelemetryMetrics struct {
	LatencyMs float64 `json:"latency_ms"`
	Loss      float64 `json:"loss"`
	JitterMs  float64 `json:"jitter_ms"`
	RSSI      float64 `json:"rssi"`
	Battery   float64 `json:"battery"`
}

type TelemetryRow struct {
	ID              int64     `json:"id"`
	DeviceID        string    `json:"deviceId"`
	TS              time.Time `json:"ts"`
	LatencyMs       float64   `json:"latencyMs"`
	Loss            float64   `json:"loss"`
	JitterMs        float64   `json:"jitterMs"`
	RSSI            float64   `json:"rssi"`
	Battery         float64   `json:"battery"`
	ConfigVersionID *string   `json:"configVersionId,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}
