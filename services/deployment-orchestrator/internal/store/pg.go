package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PG struct{ Pool *pgxpool.Pool }

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

func (p *PG) CreateDeployment(ctx context.Context, configVersionID string, strategy any) (string, error) {
	b, _ := json.Marshal(strategy)
	var id string
	err := p.Pool.QueryRow(ctx, `
		INSERT INTO deploy.deployments (config_version_id, strategy, status)
		VALUES ($1, $2, 'CREATED')
		RETURNING id
	`, configVersionID, b).Scan(&id)
	return id, err
}

func (p *PG) SetDeploymentStarted(ctx context.Context, deploymentID, status string) error {
	_, err := p.Pool.Exec(ctx, `
		UPDATE deploy.deployments
		SET status=$2, started_at=now(), updated_at=now()
		WHERE id=$1
	`, deploymentID, status)
	return err
}

func (p *PG) SetDeploymentFinished(ctx context.Context, deploymentID, status string) error {
	_, err := p.Pool.Exec(ctx, `
		UPDATE deploy.deployments
		SET status=$2, finished_at=now(), updated_at=now()
		WHERE id=$1
	`, deploymentID, status)
	return err
}

func (p *PG) UpdateDeploymentStatus(ctx context.Context, deploymentID, status string) error {
	_, err := p.Pool.Exec(ctx, `
		UPDATE deploy.deployments
		SET status=$2, updated_at=now()
		WHERE id=$1
	`, deploymentID, status)
	return err
}

func (p *PG) InsertTargets(ctx context.Context, deploymentID string, deviceIDs []string, phase string) error {
	var b pgx.Batch
	for _, d := range deviceIDs {
		b.Queue(`
			INSERT INTO deploy.deployment_targets (deployment_id, device_id, phase, status)
			VALUES ($1, $2, $3, 'PENDING')
			ON CONFLICT (deployment_id, device_id) DO NOTHING
		`, deploymentID, d, phase)
	}
	br := p.Pool.SendBatch(ctx, &b)
	defer br.Close()
	for range deviceIDs {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (p *PG) MarkTargetStatus(ctx context.Context, deploymentID, deviceID, status, lastErr string) error {
	_, err := p.Pool.Exec(ctx, `
		UPDATE deploy.deployment_targets
		SET status=$3, last_error=$4, updated_at=now()
		WHERE deployment_id=$1 AND device_id=$2
	`, deploymentID, deviceID, status, nullIfEmpty(lastErr))
	return err
}

func (p *PG) GetLastApplyStatuses(ctx context.Context, since time.Time, deviceIDs []string, configVersionID string) (map[string]string, map[string]string, error) {
	// statusMap[deviceId] = last_status(applied|failed|sent)
	// errMap[deviceId] = last_error (if any)
	rows, err := p.Pool.Query(ctx, `
		WITH last AS (
		  SELECT device_id,
		         (ARRAY_AGG(status ORDER BY created_at DESC))[1] AS st,
		         (ARRAY_AGG(COALESCE(error,'') ORDER BY created_at DESC))[1] AS er
		  FROM cfg.config_apply_log
		  WHERE created_at >= $1
		    AND config_version_id = $2
		    AND device_id = ANY($3::uuid[])
		  GROUP BY device_id
		)
		SELECT device_id::text, st::text, er::text
		FROM last
	`, since, configVersionID, deviceIDs)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	st := map[string]string{}
	er := map[string]string{}
	for rows.Next() {
		var did, s, e string
		if err := rows.Scan(&did, &s, &e); err != nil {
			return nil, nil, err
		}
		st[did] = s
		if e != "" {
			er[did] = e
		}
	}
	return st, er, nil
}

func (p *PG) GetDeploymentStatus(ctx context.Context, deploymentID string) (configVersionID, status string, startedAt, finishedAt *time.Time, err error) {
	err = p.Pool.QueryRow(ctx, `
		SELECT config_version_id::text, status, started_at, finished_at
		FROM deploy.deployments
		WHERE id=$1
	`, deploymentID).Scan(&configVersionID, &status, &startedAt, &finishedAt)
	return
}

func (p *PG) GetTargetCounts(ctx context.Context, deploymentID string) (total, canary, full int, byStatus map[string]int, err error) {
	byStatus = map[string]int{}
	rows, err := p.Pool.Query(ctx, `
		SELECT phase, status, COUNT(*)
		FROM deploy.deployment_targets
		WHERE deployment_id=$1
		GROUP BY phase, status
	`, deploymentID)
	if err != nil {
		return 0, 0, 0, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ph, st string
		var c int
		if err := rows.Scan(&ph, &st, &c); err != nil {
			return 0, 0, 0, nil, err
		}
		total += c
		if ph == "CANARY" {
			canary += c
		} else if ph == "FULL" {
			full += c
		}
		byStatus[st] += c
	}
	return
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
