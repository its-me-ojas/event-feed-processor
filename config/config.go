package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	APIPort       string
	ProcessorPort string

	// Kafka settings
	KafkaBrokers   []string
	KafkaGroupID   string
	PostEventTopic string
	DLQTopic       string

	DatabaseURL string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	MaxRetries    int
	RetryBackoff  time.Duration
	ConsumerBatch int
}

func Load() *Config {
	return &Config{

		APIPort:       getEnv("API_PORT", "8080"),
		ProcessorPort: getEnv("PROCESSOR_PORT", "8081"),

		KafkaBrokers:   []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
		KafkaGroupID:   getEnv("KAFKA_GROUP_ID", "feed-processor"),
		PostEventTopic: getEnv("KAFKA_POST_TOPIC", "post-events"),
		DLQTopic:       getEnv("KAFKA_DLQ_TOPIC", "dead-letter-events"),

		DatabaseURL: getEnv("DATABASE_URL", "postgres://feeduser:feedpass@localhost:5432/feeddb?sslmode=disable"),

		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		MaxRetries:    getEnvInt("MAX_RETRIES", 3),
		RetryBackoff:  time.Duration(getEnvInt("RETRY_BACKOFF_MS", 100)) * time.Millisecond,
		ConsumerBatch: getEnvInt("CONSUMER_BATCH", 100),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
