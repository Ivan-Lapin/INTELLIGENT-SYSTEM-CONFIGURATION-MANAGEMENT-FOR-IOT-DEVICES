package store

import (
	"context"
	"encoding/json"
	"fmt"
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
			device_id, ts, latency_ms, loss, jitter_ms, rssi, battery, config_version_id, rollout_id, rollout_phase
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`,
		ev.DeviceID,
		ev.TS,
		ev.Metrics.LatencyMs,
		ev.Metrics.Loss,
		ev.Metrics.JitterMs,
		ev.Metrics.RSSI,
		ev.Metrics.Battery,
		ev.ConfigVersionID,
		ev.RolloutID,
		ev.RolloutPhase,
	)
	return err
}

func (p *PG) GetRecentTelemetry(ctx context.Context, deviceID string, limit int) ([]model.TelemetryRow, error) {
	rows, err := p.Pool.Query(ctx, `
		SELECT id, device_id::text, ts, latency_ms, loss, jitter_ms, rssi, battery,
		       config_version_id::text, rollout_id, rollout_phase, created_at
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
			&r.RolloutID,
			&r.RolloutPhase,
			&r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}

	if err := rows.Err(); err != nil {
		return nil, err
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

func (p *PG) GetWindowAggregation(
	ctx context.Context,
	deviceID string,
	since time.Time,
	rolloutID *string,
	rolloutPhase *string,
	windowLabel string,
) (*model.TelemetryWindowAggregation, error) {
	query := `
		SELECT
			COUNT(*) AS sample_count,
			COALESCE(AVG(latency_ms), 0) AS latency_avg,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms), 0) AS latency_p95,
			COALESCE(AVG(loss), 0) AS packet_loss_avg,
			COALESCE(AVG(rssi), 0) AS rssi_avg,
			COALESCE(MAX(battery) - MIN(battery), 0) AS battery_drop_rate
		FROM telemetry.metrics_raw
		WHERE device_id = $1
		  AND ts >= $2
	`
	args := []any{deviceID, since}
	argPos := 3

	if rolloutID != nil {
		query += fmt.Sprintf(" AND rollout_id = $%d", argPos)
		args = append(args, *rolloutID)
		argPos++
	}
	if rolloutPhase != nil {
		query += fmt.Sprintf(" AND rollout_phase = $%d", argPos)
		args = append(args, *rolloutPhase)
		argPos++
	}

	var agg model.TelemetryWindowAggregation
	agg.DeviceID = deviceID
	agg.Window = windowLabel
	agg.RolloutID = rolloutID
	agg.RolloutPhase = rolloutPhase
	agg.From = since
	agg.To = time.Now().UTC()

	err := p.Pool.QueryRow(ctx, query, args...).Scan(
		&agg.SampleCount,
		&agg.LatencyAvg,
		&agg.LatencyP95,
		&agg.PacketLossAvg,
		&agg.RSSIAvg,
		&agg.BatteryDropRate,
	)
	if err != nil {
		return nil, err
	}

	return &agg, nil
}

func (p *PG) GetAggregationsForStandardWindows(
	ctx context.Context,
	deviceID string,
	rolloutID *string,
	rolloutPhase *string,
) ([]model.TelemetryWindowAggregation, error) {
	now := time.Now().UTC()

	windows := []struct {
		Label string
		Since time.Time
	}{
		{"1m", now.Add(-1 * time.Minute)},
		{"5m", now.Add(-5 * time.Minute)},
		{"15m", now.Add(-15 * time.Minute)},
	}

	out := make([]model.TelemetryWindowAggregation, 0, len(windows))
	for _, w := range windows {
		agg, err := p.GetWindowAggregation(ctx, deviceID, w.Since, rolloutID, rolloutPhase, w.Label)
		if err != nil {
			return nil, err
		}
		out = append(out, *agg)
	}

	return out, nil
}

func (p *PG) InsertAnomalyEvent(ctx context.Context, ev model.AnomalyEvent) error {
	var metricsJSON []byte
	var err error

	if ev.Metrics != nil {
		metricsJSON, err = json.Marshal(ev.Metrics)
		if err != nil {
			return err
		}
	}

	_, err = p.Pool.Exec(ctx, `
		INSERT INTO telemetry.anomaly_events (
			device_id, ts, rollout_id, rollout_phase, event_type, severity, message, metrics
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		ev.DeviceID,
		ev.TS,
		ev.RolloutID,
		ev.RolloutPhase,
		ev.EventType,
		ev.Severity,
		ev.Message,
		metricsJSON,
	)
	return err
}

func (p *PG) GetRecentAnomalies(ctx context.Context, deviceID string, limit int) ([]model.AnomalyEvent, error) {
	rows, err := p.Pool.Query(ctx, `
		SELECT id, device_id, ts, rollout_id, rollout_phase, event_type, severity, message, metrics, created_at
		FROM telemetry.anomaly_events
		WHERE device_id = $1
		ORDER BY ts DESC
		LIMIT $2
	`, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.AnomalyEvent, 0, limit)
	for rows.Next() {
		var ev model.AnomalyEvent
		var metricsRaw []byte

		if err := rows.Scan(
			&ev.ID,
			&ev.DeviceID,
			&ev.TS,
			&ev.RolloutID,
			&ev.RolloutPhase,
			&ev.EventType,
			&ev.Severity,
			&ev.Message,
			&metricsRaw,
			&ev.CreatedAt,
		); err != nil {
			return nil, err
		}

		if len(metricsRaw) > 0 {
			if err := json.Unmarshal(metricsRaw, &ev.Metrics); err != nil {
				return nil, err
			}
		}

		out = append(out, ev)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
