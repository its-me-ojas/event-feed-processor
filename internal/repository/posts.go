package repository

import (
	"context"
	"time"
)

type Post struct {
	PostID    int64
	AuthorID  string
	Content   string
	CreatedAt time.Time
}

type PostsRepo struct {
	db *DB
}

func NewPostsRepo(db *DB) *PostsRepo {
	return &PostsRepo{db: db}
}

func (r *PostsRepo) create(ctx context.Context, post *Post) error {
	query := `
	INSERT INTO posts (post_id,author_id,content,created_at) VALUES ($1,$2,$3,$4)`

	_, err := r.db.Pool.Exec(ctx, query, post.PostID, post.AuthorID, post.Content, post.CreatedAt)
	return err
}

func (r *PostsRepo) GetByID(ctx context.Context, postId int64) (*Post, error) {
	query := `SELECT post_id, author_id, content, created_at FROM posts WHERE post_id=$1`
	var post Post
	err := r.db.Pool.QueryRow(ctx, query, postId).Scan(&postId, &post.AuthorID, &post.Content, &post.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &post, nil
}
