CREATE SCHEMA IF NOT EXISTS telemetry;

CREATE TABLE IF NOT EXISTS telemetry.metrics_raw (
  id BIGSERIAL PRIMARY KEY,
  device_id UUID NOT NULL,
  ts TIMESTAMPTZ NOT NULL,
  latency_ms DOUBLE PRECISION NOT NULL,
  loss DOUBLE PRECISION NOT NULL,
  jitter_ms DOUBLE PRECISION NOT NULL,
  rssi DOUBLE PRECISION NOT NULL,
  battery DOUBLE PRECISION NOT NULL,
  config_version_id UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_metrics_raw_device_ts
  ON telemetry.metrics_raw (device_id, ts DESC);

CREATE INDEX IF NOT EXISTS idx_metrics_raw_ts
  ON telemetry.metrics_raw (ts DESC);