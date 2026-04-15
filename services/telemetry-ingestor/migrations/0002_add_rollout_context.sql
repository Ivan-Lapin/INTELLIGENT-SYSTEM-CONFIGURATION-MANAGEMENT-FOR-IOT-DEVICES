ALTER TABLE telemetry.metrics_raw
ADD COLUMN IF NOT EXISTS rollout_id TEXT,
ADD COLUMN IF NOT EXISTS rollout_phase TEXT;

CREATE INDEX IF NOT EXISTS idx_metrics_raw_rollout
ON telemetry.metrics_raw(rollout_id, rollout_phase, ts DESC);