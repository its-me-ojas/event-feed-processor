package processor

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/its-me-ojas/event-driven-feed/internal/events"
	"github.com/its-me-ojas/event-driven-feed/internal/metrics"
	"github.com/its-me-ojas/event-driven-feed/internal/repository"
	"github.com/segmentio/kafka-go"
)

type EventHandler struct {
	idempotencyRepo *repository.IdempotencyRepo
	feedRepo        *repository.FeedRepo
	followersRepo   *repository.FollowersRepo
	postsRepo       *repository.PostsRepo
}

func NewEventHandler(idem *repository.IdempotencyRepo, feed *repository.FeedRepo, followers *repository.FollowersRepo, posts *repository.PostsRepo) *EventHandler {
	return &EventHandler{
		idempotencyRepo: idem,
		feedRepo:        feed,
		followersRepo:   followers,
		postsRepo:       posts,
	}
}

// Handle processes a single Kafka message
func (h *EventHandler) Handle(msg kafka.Message) error {
	// Start the timer
	start := time.Now()
	ctx := context.Background()

	// 1. Unmarshal the event
	event, err := events.Unmarshal(msg.Value)
	if err != nil {
		// If JSON is invalid, we cant process it
		// returning error triggers retry/DLQ loop in consumer
		return fmt.Errorf("unmarshal error: %v", err)
	}

	// 2. Check idempotency
	// if we crash after writing to DB but before commiting kafka offset
	// checking this prevents us from processing the same event twice
	processed, err := h.idempotencyRepo.IsProcessed(ctx, event.EventID)
	if err != nil {
		return fmt.Errorf("idempotency check failed: %v", err)
	}
	if processed {
		log.Printf("Event %d already processed, skipping", event.EventID)
		return nil
	}

	// 3. process based on Event type
	var processErr error // Track the error locally to decide success/failute status
	switch event.Type {
	case events.EventTypePostCreated:
		processErr = h.handlePostCreated(ctx, event)
	default:
		log.Printf("unknown event type: %s", event.Type)
	}

	// stop timer
	duration := time.Since(start).Seconds()

	// determine status for metrics
	status := "success"
	if processErr != nil {
		status = "failure"
	}

	// Record metrics
	metrics.EventsProcessed.WithLabelValues(status, event.Type).Inc()
	metrics.EventDuration.WithLabelValues(event.Type).Observe(duration)

	// 4. Mark as processed
	// This "locks" the event so it won't be processed again
	if err := h.idempotencyRepo.MarkProcessed(ctx, event.EventID); err != nil {
		return fmt.Errorf("failed to mark processed: %v", err)
	}
	log.Printf("Successfully processed event %d type %s", event.EventID, event.Type)

	if processErr != nil {
		return processErr
	}
	return nil
}

func (h *EventHandler) handlePostCreated(ctx context.Context, event *events.Event) error {
	authorID := event.ActorID
	postID := event.Payload.PostID
	content := event.Payload.Content

	post := &repository.Post{
		PostID:    postID,
		AuthorID:  authorID,
		Content:   content,
		CreatedAt: time.Unix(event.Timestamp, 0),
	}
	if err := h.postsRepo.Create(ctx, post); err != nil {
		return fmt.Errorf("Failed to persist post: %v", err)
	}

	// for follower count
	count, err := h.followersRepo.GetFollowerCount(ctx, authorID)
	if err != nil {
		return fmt.Errorf("failed to get follower count: %v", err)
	}

	if count >= 10000 {
		log.Printf("User %s is a celebrity (%d followers skipping fan out)", authorID, count)
		return nil
	}

	// 1. Fetch all followers of the author
	followers, err := h.followersRepo.GetFollowers(ctx, authorID)
	if err != nil {
		return fmt.Errorf("failed to fetch followers: %v", err)
	}

	if len(followers) == 0 {
		return nil
	}

	// 2. Add post to all followers' feed (Batch Insert)
	if err := h.feedRepo.AddToFeedBatch(ctx, followers, postID); err != nil {
		return fmt.Errorf("feed fan-out failed: %v", err)
	}
	return nil
}
