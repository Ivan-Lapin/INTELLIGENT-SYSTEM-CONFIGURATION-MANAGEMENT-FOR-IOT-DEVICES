ALTER TABLE cfg.config_versions
ADD COLUMN IF NOT EXISTS parent_version INT,
ADD COLUMN IF NOT EXISTS status TEXT DEFAULT 'draft',
ADD COLUMN IF NOT EXISTS created_by TEXT,
ADD COLUMN IF NOT EXISTS rollout_strategy TEXT,
ADD COLUMN IF NOT EXISTS change_summary TEXT;

CREATE INDEX IF NOT EXISTS idx_config_versions_template_version
ON cfg.config_versions(template_id, version);