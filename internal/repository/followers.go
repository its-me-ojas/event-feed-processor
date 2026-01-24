package repository

import "context"

type FollowersRepo struct {
	db *DB
}

func NewFollowersRepo(db *DB) *FollowersRepo {
	return &FollowersRepo{db: db}
}

func (r *FollowersRepo) GetFollowers(ctx context.Context, userID string) ([]string, error) {
	query := `SELECT follower_id FROM followers WHERE followee_id = $1`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var followers []string
	for rows.Next() {
		var followerID string
		if err := rows.Scan(&followerID); err != nil {
			return nil, err
		}
		followers = append(followers, followerID)
	}
	return followers, rows.Err()
}

func (r *FollowersRepo) Follow(ctx context.Context, followerID, followeeID string) error {
	query := `INSERT INTO followers (follower_id, followee_id, created_at) VALUES ($1, $2, NOW()) ON CONFLICT DO NOTHING`
	_, err := r.db.Pool.Exec(ctx, query, followerID, followeeID)
	return err
}

func (r *FollowersRepo) GetFollowerCount(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM followers WHERE followee_id = $1`
	var count int
	err := r.db.Pool.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
