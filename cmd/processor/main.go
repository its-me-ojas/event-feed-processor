package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/its-me-ojas/event-driven-feed/config"
	"github.com/its-me-ojas/event-driven-feed/internal/kafka"
	"github.com/its-me-ojas/event-driven-feed/internal/processor"
	"github.com/its-me-ojas/event-driven-feed/internal/repository"
)

func main() {
	// 1. Load Configuration
	cfg := config.Load()

	// 2. Connect to Database
	db, err := repository.NewDB(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	// 3. Initialize Repositories
	idempotencyRepo := repository.NewIdempotencyRepo(db)
	feedRepo := repository.NewFeedRepo(db)
	followersRepo := repository.NewFollowersRepo(db)

	// 4. Initialize Handler (The Business Logic)
	handler := processor.NewEventHandler(idempotencyRepo, feedRepo, followersRepo)

	// 5. Initialize Kafka Consumer (The Transport Layer)
	consumerCfg := kafka.ConsumerConfig{
		Brokers:    cfg.KafkaBrokers,
		Topic:      "post-events",
		GroupID:    "feed-processor-group",
		DLQTopic:   "dead-letter-events",
		MaxRetries: 3,
	}
	consumer := kafka.NewConsumer(consumerCfg)
	defer consumer.Close()

	// 6. Handle Graceful Shutdown
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down processor...")
		cancel()
	}()

	// 7. Start Processing Loop
	log.Println("Starting feed processor...")
	consumer.ConsumeLoop(ctx, handler.Handle)
}
