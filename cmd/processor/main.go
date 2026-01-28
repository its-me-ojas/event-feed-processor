package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/its-me-ojas/event-driven-feed/config"
	"github.com/its-me-ojas/event-driven-feed/internal/kafka"
	"github.com/its-me-ojas/event-driven-feed/internal/processor"
	"github.com/its-me-ojas/event-driven-feed/internal/repository"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	postsRepo := repository.NewPostsRepo(db)

	// 4. Initialize Handler (The Business Logic)
	handler := processor.NewEventHandler(idempotencyRepo, feedRepo, followersRepo, postsRepo)

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

	// Start a separate HTTP server for metrics
	// This runs in a goroutine so it doesn't block the main thread
	go func() {
		log.Printf("Starting metrics server on port %s", cfg.ProcessorPort)
		http.Handle("/metrics", promhttp.Handler())
		// We use ProcessorPort (e.g., 8081) to avoid conflict with API (8080)
		if err := http.ListenAndServe(":"+cfg.ProcessorPort, nil); err != nil {
			log.Printf("Metrics server failed: %v", err)
		}
	}()

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
