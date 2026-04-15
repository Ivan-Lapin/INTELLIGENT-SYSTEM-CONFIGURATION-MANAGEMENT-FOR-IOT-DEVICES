ALTER TABLE registry.devices
ADD COLUMN IF NOT EXISTS network_profile TEXT,
ADD COLUMN IF NOT EXISTS location_zone TEXT,
ADD COLUMN IF NOT EXISTS health_score DOUBLE PRECISION,
ADD COLUMN IF NOT EXISTS battery_level INT,
ADD COLUMN IF NOT EXISTS current_config_version INT,
ADD COLUMN IF NOT EXISTS last_successful_version INT,
ADD COLUMN IF NOT EXISTS last_rollout_status TEXT;