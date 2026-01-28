package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/its-me-ojas/event-driven-feed/config"
	"github.com/its-me-ojas/event-driven-feed/internal/api"
	"github.com/its-me-ojas/event-driven-feed/internal/api/handlers"
	"github.com/its-me-ojas/event-driven-feed/internal/api/middleware"
	"github.com/its-me-ojas/event-driven-feed/internal/cache"
	"github.com/its-me-ojas/event-driven-feed/internal/kafka"
	"github.com/its-me-ojas/event-driven-feed/internal/repository"
	"github.com/its-me-ojas/event-driven-feed/internal/snowflake"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg := config.Load()

	// Database
	ctx := context.Background()
	db, err := repository.NewDB(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Redis
	redisClient, err := cache.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Repos
	postsRepo := repository.NewPostsRepo(db)
	feedsRepo := repository.NewFeedRepo(db)
	followersRepo := repository.NewFollowersRepo(db)
	feedCache := repository.NewFeedCache(redisClient)

	// Kafka producer
	producer := kafka.NewProducer(cfg.KafkaBrokers, cfg.PostEventTopic)
	defer producer.Close()

	// ID generator
	idGen := snowflake.NewGenerator(1)

	// Handlers
	h := handlers.NewHandler(producer, idGen, postsRepo, feedsRepo, feedCache, followersRepo)

	// Router
	router := api.NewRouter(h)

	router.Use(middleware.MetricsMiddleware)
	router.Handle("/metrics", promhttp.Handler())

	// Server
	srv := &http.Server{
		Addr:         ":" + cfg.APIPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Printf("API server starting on port %s", cfg.APIPort)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}
	log.Println("Server stopped")
}
