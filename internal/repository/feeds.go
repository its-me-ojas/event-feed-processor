package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type FeedRepo struct {
	db *DB
}

func NewFeedRepo(db *DB) *FeedRepo {
	return &FeedRepo{db: db}
}

func (r *FeedRepo) AddToFeed(ctx context.Context, userID string, postID int64) error {
	query := `
	INSERT INTO feeds (user_id,post_id,created_at) VALUES ($1,$2,NOW()) ON CONFLICT (user_id,post_id) DO NOTHING`
	_, err := r.db.Pool.Exec(ctx, query, userID, postID)
	return err
}

func (r *FeedRepo) AddToFeedBatch(ctx context.Context, userIDs []string, postID int64) error {
	if len(userIDs) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, userID := range userIDs {
		batch.Queue(
			`INSERT INTO feeds (user_id,post_id,created_at) VALUES ($1,$2,NOW()) ON CONFLICT DO NOTHING`, userID, postID)
	}

	results := r.db.Pool.SendBatch(ctx, batch)
	defer results.Close()

	for range userIDs {
		if _, err := results.Exec(); err != nil {
			return err
		}
	}

	return nil

}

func (r *FeedRepo) GetFeed(ctx context.Context, userID string, limit, offset int) ([]int64, error) {
	query := `SELECT post_id FROM feeds WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var postIDs []int64
	for rows.Next() {
		var postID int64
		if err := rows.Scan(&postID); err != nil {
			return nil, err
		}
		postIDs = append(postIDs, postID)
	}
	return postIDs, rows.Err()
}
