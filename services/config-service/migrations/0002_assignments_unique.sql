CREATE UNIQUE INDEX IF NOT EXISTS uq_assignments_target
ON cfg.config_assignments (target_type, target_id);