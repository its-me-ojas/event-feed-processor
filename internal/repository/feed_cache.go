package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/its-me-ojas/event-driven-feed/internal/cache"
	"github.com/redis/go-redis/v9"
)

type FeedCache struct {
	client *cache.RedisClient
}

func NewFeedCache(client *cache.RedisClient) *FeedCache {
	return &FeedCache{
		client: client,
	}
}

// GetFeed attempts to fetch the feed from Redis
func (c *FeedCache) GetFeed(ctx context.Context, userID string) ([]int64, error) {
	key := fmt.Sprintf("feed:%s", userID)
	val, err := c.client.Client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, err
	}
	var postIDs []int64
	if err := json.Unmarshal([]byte(val), &postIDs); err != nil {
		return nil, err
	}
	return postIDs, nil
}

// SetFeed caches the feed with an expiration
func (c *FeedCache) SetFeed(ctx context.Context, userID string, postIDs []int64) error {
	key := fmt.Sprintf("feed:%s", userID)
	data, err := json.Marshal(postIDs)
	if err != nil {
		return err
	}
	// Cache for 5 minutes
	return c.client.Client.Set(ctx, key, data, 5*time.Minute).Err()
}

// InvalidateFeed removes a user's feed from cache (used when new posts arrive)
func (c *FeedCache) InvalidateFeed(ctx context.Context, userID string) error {
	key := fmt.Sprintf("feed:%s", userID)
	return c.client.Client.Del(ctx, key).Err()
}
