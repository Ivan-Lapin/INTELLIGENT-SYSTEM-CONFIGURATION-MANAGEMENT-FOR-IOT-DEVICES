package store

import (
	"context"
	"time"

	"telemetry-ingestor/internal/model"

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

func (p *PG) InsertTelemetry(ctx context.Context, ev model.TelemetryEvent) error {
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO telemetry.metrics_raw (
			device_id, ts, latency_ms, loss, jitter_ms, rssi, battery, config_version_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		ev.DeviceID,
		ev.TS,
		ev.Metrics.LatencyMs,
		ev.Metrics.Loss,
		ev.Metrics.JitterMs,
		ev.Metrics.RSSI,
		ev.Metrics.Battery,
		ev.ConfigVersionID,
	)
	return err
}

func (p *PG) GetRecentTelemetry(ctx context.Context, deviceID string, limit int) ([]model.TelemetryRow, error) {
	rows, err := p.Pool.Query(ctx, `
		SELECT id, device_id::text, ts, latency_ms, loss, jitter_ms, rssi, battery, config_version_id::text, created_at
		FROM telemetry.metrics_raw
		WHERE device_id = $1
		ORDER BY ts DESC
		LIMIT $2
	`, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.TelemetryRow, 0, limit)
	for rows.Next() {
		var r model.TelemetryRow
		if err := rows.Scan(
			&r.ID,
			&r.DeviceID,
			&r.TS,
			&r.LatencyMs,
			&r.Loss,
			&r.JitterMs,
			&r.RSSI,
			&r.Battery,
			&r.ConfigVersionID,
			&r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (p *PG) GetTelemetryCountSince(ctx context.Context, since time.Time) (int, error) {
	var n int
	err := p.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM telemetry.metrics_raw
		WHERE ts >= $1
	`, since).Scan(&n)
	return n, err
}
