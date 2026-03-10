package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PG struct {
	Pool *pgxpool.Pool
}

func NewPG(ctx context.Context, dsn string) (*PG, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &PG{Pool: pool}, nil
}

func (p *PG) InsertApplyLogSent(ctx context.Context, deviceID, configVersionID string) error {
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO cfg.config_apply_log (device_id, config_version_id, status, created_at)
		VALUES ($1, $2, 'sent', now())
	`, deviceID, configVersionID)
	return err
}

func (p *PG) InsertApplyLogResult(ctx context.Context, deviceID, configVersionID, status, errMsg string, appliedAt time.Time) error {
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO cfg.config_apply_log (device_id, config_version_id, status, error, applied_at, created_at)
		VALUES ($1, $2, $3, $4, $5, now())
	`, deviceID, configVersionID, status, nullIfEmpty(errMsg), appliedAt)
	return err
}

func (p *PG) UpsertAssignment(ctx context.Context, targetType, targetID, configVersionID, status string) error {
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO cfg.config_assignments (target_type, target_id, config_version_id, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (target_type, target_id)
		DO UPDATE SET config_version_id = EXCLUDED.config_version_id,
		              status = EXCLUDED.status,
		              updated_at = now()
	`, targetType, targetID, configVersionID, status)
	return err
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
