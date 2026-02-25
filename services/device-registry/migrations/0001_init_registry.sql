CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE SCHEMA IF NOT EXISTS registry;

CREATE TABLE IF NOT EXISTS registry.devices (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  external_id TEXT UNIQUE,
  device_type TEXT NOT NULL,
  protocol TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  tags JSONB NOT NULL DEFAULT '{}'::jsonb,
  last_seen_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_devices_type ON registry.devices(device_type);
CREATE INDEX IF NOT EXISTS idx_devices_protocol ON registry.devices(protocol);
CREATE INDEX IF NOT EXISTS idx_devices_last_seen ON registry.devices(last_seen_at);

CREATE TABLE IF NOT EXISTS registry.device_groups (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name TEXT NOT NULL UNIQUE,
  description TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS registry.device_group_members (
  group_id UUID NOT NULL REFERENCES registry.device_groups(id) ON DELETE CASCADE,
  device_id UUID NOT NULL REFERENCES registry.devices(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (group_id, device_id)
);

CREATE INDEX IF NOT EXISTS idx_group_members_device ON registry.device_group_members(device_id);