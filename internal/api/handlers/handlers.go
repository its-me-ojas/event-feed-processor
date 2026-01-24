package handlers

import (
	"github.com/its-me-ojas/event-driven-feed/internal/kafka"
	"github.com/its-me-ojas/event-driven-feed/internal/repository"
	"github.com/its-me-ojas/event-driven-feed/internal/snowflake"
)

type Handlers struct {
	producer      *kafka.Producer
	idGen         *snowflake.Generator
	postsRepo     *repository.PostsRepo
	feedsRepo     *repository.FeedRepo
	feedCache     *repository.FeedCache
	followersRepo *repository.FollowersRepo
}

func NewHandler(
	producer *kafka.Producer, idGen *snowflake.Generator, postsRepo *repository.PostsRepo, feedsRepo *repository.FeedRepo, feedCache *repository.FeedCache, followersRepo *repository.FollowersRepo) *Handlers {
	return &Handlers{
		producer:      producer,
		idGen:         idGen,
		postsRepo:     postsRepo,
		feedsRepo:     feedsRepo,
		feedCache:     feedCache,
		followersRepo: followersRepo,
	}
}
