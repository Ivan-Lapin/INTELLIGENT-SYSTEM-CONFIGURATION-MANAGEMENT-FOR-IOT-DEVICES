CREATE UNIQUE INDEX IF NOT EXISTS ux_config_assignments_target
ON cfg.config_assignments(target_type, target_id);