CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE SCHEMA IF NOT EXISTS cfg;

CREATE TABLE IF NOT EXISTS cfg.config_templates (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name TEXT NOT NULL,
  device_type TEXT NOT NULL,
  schema_version INT NOT NULL DEFAULT 1,
  mongo_template_id TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_templates_device_type ON cfg.config_templates(device_type);

CREATE TABLE IF NOT EXISTS cfg.config_versions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  template_id UUID NOT NULL REFERENCES cfg.config_templates(id) ON DELETE CASCADE,
  version INT NOT NULL,
  checksum TEXT NOT NULL,
  mongo_version_id TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (template_id, version)
);

CREATE INDEX IF NOT EXISTS idx_versions_template ON cfg.config_versions(template_id);
CREATE INDEX IF NOT EXISTS idx_versions_created_at ON cfg.config_versions(created_at);

CREATE TABLE IF NOT EXISTS cfg.config_assignments (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  target_type TEXT NOT NULL, -- device|group
  target_id UUID NOT NULL,
  config_version_id UUID NOT NULL REFERENCES cfg.config_versions(id),
  status TEXT NOT NULL DEFAULT 'desired',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_assignments_target ON cfg.config_assignments(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_assignments_status ON cfg.config_assignments(status);

CREATE TABLE IF NOT EXISTS cfg.config_apply_log (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  device_id UUID NOT NULL,
  config_version_id UUID NOT NULL REFERENCES cfg.config_versions(id),
  deployment_id UUID,
  status TEXT NOT NULL,
  error TEXT,
  applied_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_applylog_device ON cfg.config_apply_log(device_id);
CREATE INDEX IF NOT EXISTS idx_applylog_created ON cfg.config_apply_log(created_at);