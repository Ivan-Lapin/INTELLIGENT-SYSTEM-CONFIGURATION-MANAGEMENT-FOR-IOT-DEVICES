package model

import "time"

type TelemetryEvent struct {
	DeviceID        string           `json:"deviceId"`
	TS              time.Time        `json:"ts"`
	Metrics         TelemetryMetrics `json:"metrics"`
	ConfigVersionID *string          `json:"configVersionId,omitempty"`
	RolloutID       *string          `json:"rolloutId,omitempty"`
	RolloutPhase    *string          `json:"rolloutPhase,omitempty"` // before / canary / after
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
	RolloutID       *string   `json:"rolloutId,omitempty"`
	RolloutPhase    *string   `json:"rolloutPhase,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

type TelemetryWindowAggregation struct {
	DeviceID        string    `json:"deviceId"`
	Window          string    `json:"window"`
	RolloutID       *string   `json:"rolloutId,omitempty"`
	RolloutPhase    *string   `json:"rolloutPhase,omitempty"`
	From            time.Time `json:"from"`
	To              time.Time `json:"to"`
	SampleCount     int       `json:"sampleCount"`
	LatencyAvg      float64   `json:"latency_avg"`
	LatencyP95      float64   `json:"latency_p95"`
	PacketLossAvg   float64   `json:"packet_loss_avg"`
	BatteryDropRate float64   `json:"battery_drop_rate"`
	RSSIAvg         float64   `json:"rssi_avg"`
}

type AnomalyEvent struct {
	ID           int64          `json:"id"`
	DeviceID     string         `json:"deviceId"`
	TS           time.Time      `json:"ts"`
	RolloutID    *string        `json:"rolloutId,omitempty"`
	RolloutPhase *string        `json:"rolloutPhase,omitempty"`
	EventType    string         `json:"eventType"`
	Severity     string         `json:"severity"`
	Message      string         `json:"message,omitempty"`
	Metrics      map[string]any `json:"metrics,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
}
