package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type IdempotencyRepo struct {
	db *DB
}

func NewIdempotencyRepo(db *DB) *IdempotencyRepo {
	return &IdempotencyRepo{db: db}
}

func (r *IdempotencyRepo) IsProcessed(ctx context.Context, eventID int64) (bool, error) {
	query := `SELECT 1 FROM processed_events WHERE event_id = $1`
	var exists int
	err := r.db.Pool.QueryRow(ctx, query, eventID).Scan(&exists)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *IdempotencyRepo) MarkProcessed(ctx context.Context, eventID int64) error {
	query := `INSERT INTO processed_events (event_id, processed_at) VALUES ($1, NOW()) ON CONFLICT DO NOTHING`
	_, err := r.db.Pool.Exec(ctx, query, eventID)
	return err
}
