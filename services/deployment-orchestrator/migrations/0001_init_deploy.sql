CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE SCHEMA IF NOT EXISTS deploy;

CREATE TABLE IF NOT EXISTS deploy.deployments (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  config_version_id UUID NOT NULL,
  strategy JSONB NOT NULL,
  status TEXT NOT NULL, -- CREATED|CANARY|EVALUATING|FULL|DONE|ROLLED_BACK|FAILED
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS deploy.deployment_targets (
  deployment_id UUID NOT NULL REFERENCES deploy.deployments(id) ON DELETE CASCADE,
  device_id UUID NOT NULL,
  phase TEXT NOT NULL, -- CANARY|FULL
  status TEXT NOT NULL DEFAULT 'PENDING', -- PENDING|SENT|APPLIED|FAILED|ROLLED_BACK
  last_error TEXT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (deployment_id, device_id)
);

CREATE INDEX IF NOT EXISTS idx_deployments_status ON deploy.deployments(status);
CREATE INDEX IF NOT EXISTS idx_targets_deployment_phase ON deploy.deployment_targets(deployment_id, phase, status);