-- Migration: 001_initial_schema.sql

-- Posts table
CREATE TABLE IF NOT EXISTS posts(
    post_id BIGINT PRIMARY KEY,
    author_id VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_posts_author ON posts(author_id,created_at DESC);

-- Feeds table
CREATE TABLE IF NOT EXISTS feeds(
    user_id VARCHAR(255) NOT NULL,
    post_id BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (user_id,post_id)
);

CREATE INDEX IF NOT EXISTS idx_feeds_user_created ON feeds(user_id,created_at DESC);

-- Idempotency table (prevents duplicate processing)
CREATE TABLE IF NOT EXISTS processed_events(
    event_id BIGINT PRIMARY KEY,
    processed_at TIMESTAMP DEFAULT NOW()
);

-- Follower table (for fan-out queries)
CREATE TABLE IF NOT EXISTS followers(
    follower_id VARCHAR(255) NOT NULL,
    followee_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (follower_id,followee_id)
);

CREATE INDEX IF NOT EXISTS idx_followers_followee ON followers(followee_id);
