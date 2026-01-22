# Event-Driven Feed & Processing System - Tasks

## 📋 Project Overview

Building a backend system that handles writes asynchronously using Kafka, generates user feeds via fan-out strategies, and remains correct under retries, crashes, and load spikes.

---

## Phase 1: Project Setup & Infrastructure ✅

- [x] Initialize Go module with `go mod init`
- [x] Create project directory structure
- [x] Create `docker-compose.yml` with:
  - [x] Zookeeper container
  - [x] Kafka container
  - [x] PostgreSQL container
  - [x] Redis container
  - [x] Prometheus container
- [x] Create `config/config.go` for environment configuration
- [x] Create `prometheus.yml` for metrics scraping
- [ ] Verify all containers start correctly

---

## Phase 2: Data Layer ✅

- [x] Create `migrations/001_initial_schema.sql`
  - [x] posts table
  - [x] feeds table
  - [x] processed_events table
  - [x] followers table
  - [x] indexes
- [ ] Run migrations against PostgreSQL
- [x] Create `internal/repository/db.go` (connection pool)
- [x] Create `internal/repository/posts.go`
- [x] Create `internal/repository/feeds.go`
- [x] Create `internal/repository/idempotency.go`
- [x] Create `internal/repository/followers.go`
- [ ] Write unit tests for repositories

---

## Phase 3: Event System

- [ ] Create `internal/events/schema.go` (Event struct)
- [ ] Create `internal/snowflake/generator.go`
- [ ] Create `internal/kafka/producer.go`
  - [ ] Connection setup
  - [ ] Partition key logic
  - [ ] Error handling
- [ ] Create `internal/kafka/consumer.go`
  - [ ] Consumer group setup
  - [ ] Manual offset commit
  - [ ] Graceful shutdown
- [ ] Test producer/consumer locally

---

## Phase 4: API Service

- [ ] Create `cmd/api/main.go`
- [ ] Create `internal/api/router.go`
- [ ] Create `internal/api/handlers/posts.go`
  - [ ] POST /posts endpoint
  - [ ] Validation
  - [ ] Event publishing
- [ ] Create `internal/api/handlers/feeds.go`
  - [ ] GET /feeds/:user_id endpoint
  - [ ] Cache-aside pattern
- [ ] Create `internal/api/middleware/auth.go`
- [ ] Test API endpoints with curl/httpie

---

## Phase 5: Feed Processor (Consumer)

- [ ] Create `cmd/processor/main.go`
- [ ] Create `internal/processor/handler.go`
  - [ ] Event consumption loop
  - [ ] Idempotency check
  - [ ] Fan-out logic
  - [ ] Offset commit
- [ ] Create `internal/processor/retry.go`
- [ ] Test consumer with manual events
- [ ] Verify idempotency works

---

## Phase 6: Caching Layer

- [ ] Create `internal/cache/redis.go`
- [ ] Implement cache-aside in feeds handler
- [ ] Implement cache invalidation in processor
- [ ] Test cache hit/miss scenarios

---

## Phase 7: Reliability (DLQ & Retries)

- [ ] Create `internal/dlq/handler.go`
- [ ] Implement exponential backoff in retry logic
- [ ] Create `dead-letter-events` topic
- [ ] Test with poison messages
- [ ] Verify DLQ captures failures

---

## Phase 8: Observability

- [ ] Create `internal/metrics/prometheus.go`
- [ ] Add consumer lag metric
- [ ] Add processing duration histogram
- [ ] Add retry counter
- [ ] Add DLQ size counter
- [ ] Verify metrics in Prometheus UI

---

## 🎯 Current Focus

> **Phase 3: Event System**

---

## ✅ Completed

- Phase 1: Infrastructure (Docker Compose, config, prometheus.yml)
- Phase 2: Data Layer (migrations, repository layer)

---

## 📝 Notes

- Follow Go project layout conventions (cmd/, internal/, etc.)
- Use bun instead of npm for any JS tooling
- Test each phase before moving to the next
- Commit frequently with descriptive messages

---

## 🔗 Quick Commands

```bash
# Start infrastructure
docker-compose up -d

# Run API service
go run cmd/api/main.go

# Run Feed Processor
go run cmd/processor/main.go

# Run tests
go test ./...

# Check Kafka topics
docker exec -it kafka kafka-topics.sh --list --bootstrap-server localhost:9092
```
