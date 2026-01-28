package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/its-me-ojas/event-driven-feed/config"
	"github.com/segmentio/kafka-go"
)

func main() {
	mode := flag.String("mode", "inspect", "Mode: inspect or replay")
	limit := flag.Int("limit", 10, "Number of messages to process")
	flag.Parse()

	cfg := config.Load()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Connect to DLQ
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.KafkaBrokers,
		Topic:       cfg.DLQTopic,
		GroupID:     "dlq-inspector-" + time.Now().Format("20060102150405"), // Unique group to see all msgs
		StartOffset: kafka.FirstOffset,
	})
	defer reader.Close()

	// Producer for Replay
	var writer *kafka.Writer
	if *mode == "replay" {
		writer = &kafka.Writer{
			Addr:     kafka.TCP(cfg.KafkaBrokers...),
			Balancer: &kafka.LeastBytes{},
		}
		defer writer.Close()
	}

	fmt.Printf("Starting DLQ Inspector (Mode: %s, Limit: %d)\n", *mode, *limit)
	fmt.Println("---------------------------------------------------")

	count := 0
	for count < *limit {
		// Set a read deadline so we don't block forever if empty
		readCtx, readCancel := context.WithTimeout(ctx, 5*time.Second)
		m, err := reader.FetchMessage(readCtx)
		readCancel()

		if err != nil {
			if ctx.Err() != nil {
				break
			}
			if err == context.DeadlineExceeded {
				fmt.Println("No more messages found (timeout).")
				break
			}
			log.Printf("Error fetching message: %v", err)
			continue
		}

		if *mode == "inspect" {
			printMessage(m)
		} else if *mode == "replay" {
			if err := replayMessage(ctx, writer, m); err != nil {
				log.Printf("Failed to replay message: %v", err)
			} else {
				fmt.Printf("Replayed message: %s\n", string(m.Key))
				// Only commit if replayed successfully
				reader.CommitMessages(ctx, m)
			}
		}

		count++
	}
}

func printMessage(m kafka.Message) {
	fmt.Printf("Key: %s\n", string(m.Key))
	fmt.Printf("Value: %s\n", string(m.Value))
	fmt.Println("Headers:")
	for _, h := range m.Headers {
		fmt.Printf("  - %s: %s\n", h.Key, string(h.Value))
	}
	fmt.Println("---------------------------------------------------")
}

func replayMessage(ctx context.Context, w *kafka.Writer, m kafka.Message) error {
	var originalTopic string

	// Find original topic from headers
	// We filter out the error headers so we don't pollute the replayed message
	var cleanHeaders []kafka.Header
	for _, h := range m.Headers {
		if h.Key == "original-topic" {
			originalTopic = string(h.Value)
		} else if h.Key != "error" && h.Key != "failed-at" {
			cleanHeaders = append(cleanHeaders, h)
		}
	}

	if originalTopic == "" {
		return fmt.Errorf("could not find original-topic header")
	}

	msg := kafka.Message{
		Topic:   originalTopic,
		Key:     m.Key,
		Value:   m.Value,
		Headers: cleanHeaders,
	}

	return w.WriteMessages(ctx, msg)
}
