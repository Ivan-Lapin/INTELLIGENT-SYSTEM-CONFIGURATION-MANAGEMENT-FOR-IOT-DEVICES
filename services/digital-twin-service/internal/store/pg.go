package store

import (
	"context"

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

type LatestTelemetry struct {
	DeviceID  string
	LatencyMs float64
	Loss      float64
	JitterMs  float64
	RSSI      float64
	Battery   float64
}

func (p *PG) GetLatestTelemetry(ctx context.Context, deviceID string) (*LatestTelemetry, error) {
	row := p.Pool.QueryRow(ctx, `
		SELECT device_id::text, latency_ms, loss, jitter_ms, rssi, battery
		FROM telemetry.metrics_raw
		WHERE device_id = $1
		ORDER BY ts DESC
		LIMIT 1
	`, deviceID)

	var t LatestTelemetry
	if err := row.Scan(
		&t.DeviceID,
		&t.LatencyMs,
		&t.Loss,
		&t.JitterMs,
		&t.RSSI,
		&t.Battery,
	); err != nil {
		return nil, err
	}
	return &t, nil
}
