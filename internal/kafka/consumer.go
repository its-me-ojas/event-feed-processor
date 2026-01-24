package kafka

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader     *kafka.Reader
	dlqWriter  *kafka.Writer // Dead letter queue
	maxRetries int
}

type ConsumerConfig struct {
	Brokers    []string
	Topic      string
	GroupID    string
	DLQTopic   string // dead letter topic
	MaxRetries int    // Max retries before DLQ
}

func NewConsumer(cfg ConsumerConfig) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		GroupID:        cfg.GroupID,
		Topic:          cfg.Topic,
		MinBytes:       10e3,
		MaxBytes:       10e6,
		CommitInterval: 0,
		StartOffset:    kafka.FirstOffset,
	})

	// DLQ producer
	var dlq *kafka.Writer
	if cfg.DLQTopic != "" {
		dlq = &kafka.Writer{
			Addr:  kafka.TCP(cfg.Brokers...),
			Topic: cfg.DLQTopic,
		}
	}

	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	return &Consumer{
		reader:     r,
		dlqWriter:  dlq,
		maxRetries: maxRetries,
	}
}

func (c *Consumer) FetchMessage(ctx context.Context) (kafka.Message, error) {
	return c.reader.FetchMessage(ctx)
}

func (c *Consumer) CommitMessage(ctx context.Context, msg kafka.Message) error {
	return c.reader.CommitMessages(ctx, msg)
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

// sendToDLQ sends failed message to dead letter queue
func (c *Consumer) sendToDLQ(ctx context.Context, msg kafka.Message, lastErr error) error {
	if c.dlqWriter == nil {
		log.Printf("DLQ not configured, dropping message: %s", string(msg.Key))
		return nil
	}

	dlqMsg := kafka.Message{
		Key:   msg.Key,
		Value: msg.Value,
		Headers: append(msg.Headers,
			kafka.Header{Key: "original-topic", Value: []byte(msg.Topic)},
			kafka.Header{Key: "error", Value: []byte(lastErr.Error())},
			kafka.Header{Key: "failed-at", Value: []byte(time.Now().Format(time.RFC3339))},
		),
	}
	return c.dlqWriter.WriteMessages(ctx, dlqMsg)
}

// consume loop processes messages with retry, backoff, panic recovery, and DLQ
func (c *Consumer) ConsumeLoop(ctx context.Context, handler func(msg kafka.Message) error) {
	fetchBackoff := time.Millisecond * 100
	maxFetchBackoff := time.Second * 30

	for {
		select {
		case <-ctx.Done():
			log.Println("Consumer shutting down...")
			return
		default:
		}

		// Fetch with backoff on error
		msg, err := c.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("Fetch error (backing off %v): %v", fetchBackoff, err)
			time.Sleep(fetchBackoff)
			// Exponential backoff
			fetchBackoff = min(fetchBackoff*2, maxFetchBackoff)
			continue
		}
		// Reset backoff on success
		fetchBackoff = time.Millisecond * 100

		// Process with retry * panic protection
		var lastErr error
		success := false

		for attempt := 1; attempt <= c.maxRetries; attempt++ {
			lastErr = c.safeHandle(handler, msg)
			if lastErr == nil {
				success = true
				break
			}
			log.Printf("Handler error (attempt %d/%d): %v", attempt, c.maxRetries, lastErr)
			if attempt < c.maxRetries {
				// Exponential backoff between retries
				backoff := time.Duration(attempt*attempt) * 100 * time.Millisecond
				time.Sleep(backoff)
			}
		}
		if !success {
			// Send to DLQ after max retries
			log.Printf("Max retries exceeded, sending to DLQ: %s", string(msg.Key))
			if err := c.sendToDLQ(ctx, msg, lastErr); err != nil {
				log.Printf("DLQ error: %v", err)
			}
		}
		// Always commit after processing (success or DLQ)
		// This prevents infinite retry loops
		if err := c.CommitMessage(ctx, msg); err != nil {
			log.Printf("Commit error: %v", err)
		}
	}
}

// safeHandle wraps handler with panic recovery
func (c *Consumer) safeHandle(handler func(msg kafka.Message) error, msg kafka.Message) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic recovered: %v", r)
			log.Printf("Handler panic: %v", r)
		}
	}()
	return handler(msg)
}
func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
