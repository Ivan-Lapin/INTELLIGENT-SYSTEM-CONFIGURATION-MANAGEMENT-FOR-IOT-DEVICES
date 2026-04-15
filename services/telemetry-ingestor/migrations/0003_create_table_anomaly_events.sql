CREATE TABLE IF NOT EXISTS telemetry.anomaly_events (
    id BIGSERIAL PRIMARY KEY,
    device_id TEXT NOT NULL,
    ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    rollout_id TEXT,
    rollout_phase TEXT,
    event_type TEXT NOT NULL,
    severity TEXT NOT NULL,
    message TEXT,
    metrics JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_anomaly_events_device_ts
ON telemetry.anomaly_events(device_id, ts DESC);

CREATE INDEX IF NOT EXISTS idx_anomaly_events_rollout
ON telemetry.anomaly_events(rollout_id, rollout_phase, ts DESC);