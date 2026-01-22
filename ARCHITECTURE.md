# Event-Driven Feed & Processing System - Architecture

A backend system that handles writes asynchronously using Kafka, generates user feeds via fan-out strategies, and remains correct under retries, crashes, and load spikes using idempotency and backpressure.

---

## Table of Contents

1. [One-Line Definition](#one-line-definition)
2. [Why This System Exists](#why-this-system-exists)
3. [High-Level Architecture](#high-level-architecture)
4. [The Three Planes](#the-three-planes)
5. [Core Components](#core-components)
6. [Data Model](#data-model)
7. [Event Design](#event-design)
8. [Kafka Topics & Partitioning](#kafka-topics--partitioning)
9. [Write Path](#write-path)
10. [Read Path](#read-path)
11. [Fan-Out Strategy](#fan-out-strategy)
12. [Idempotency](#idempotency)
13. [Failure Scenarios](#failure-scenarios)
14. [Backpressure](#backpressure)
15. [Reliability Patterns](#reliability-patterns)
16. [Observability](#observability)
17. [Project Structure](#project-structure)
18. [Tech Stack](#tech-stack)
19. [What This Project Proves](#what-this-project-proves)

---

## One-Line Definition

> A backend system that handles writes asynchronously using Kafka, generates user feeds via fan-out strategies, and remains correct under retries, crashes, and load spikes using idempotency and backpressure.

**If you can't say this cleanly, stop and reread.**

---

## Why This System Exists

### The Naïve Approach (WRONG)

```mermaid
sequenceDiagram
    participant User
    participant Backend
    participant DB
    
    User->>Backend: Post content
    Backend->>DB: Write post
    loop For each follower
        Backend->>DB: Update follower's feed
    end
    Backend->>User: 200 OK (after ALL writes)
    
    Note over Backend,DB: ❌ Latency explodes as followers grow
```

**Problems with synchronous approach:**
- Request latency grows linearly with followers
- One slow DB write blocks everything
- Celebrity with 1M followers = 1M writes per post
- System becomes unusable at scale

### The Real Problem

| Requirement | Why It Matters |
|-------------|----------------|
| Writes should be fast | Users expect instant feedback |
| Work should be async | Heavy processing shouldn't block requests |
| Failures shouldn't lose data | Reliability is non-negotiable |
| Spikes shouldn't kill the DB | Traffic is unpredictable |

**This is why event-driven architecture exists.**

---

## High-Level Architecture

```mermaid
graph TB
    subgraph "Request Plane"
        Client[Client]
        API[API Service<br/>Go]
    end
    
    subgraph "Event Plane"
        Kafka[(Kafka<br/>Event Backbone)]
        DLQ[(Dead Letter Queue)]
    end
    
    subgraph "Processing Plane"
        FP[Feed Processor<br/>Consumer Group]
    end
    
    subgraph "Storage Layer"
        PG[(PostgreSQL<br/>Posts, Feeds)]
        Redis[(Redis<br/>Feed Cache)]
    end
    
    Client -->|POST /posts| API
    API -->|Publish Event| Kafka
    Kafka -->|Consume| FP
    FP -->|Fan-out Writes| PG
    FP -->|Cache Hot Feeds| Redis
    FP -->|Poison Messages| DLQ
    API -->|Read Feeds| Redis
    API -->|Cache Miss| PG
```

---

## The Three Planes

This architecture separates concerns into three distinct planes. **Keep these separated or the system collapses.**

```mermaid
graph LR
    subgraph "1️⃣ Request Plane"
        direction TB
        R1[Handles User Intent]
        R2[Validates Requests]
        R3[Returns Fast]
    end
    
    subgraph "2️⃣ Event Plane"
        direction TB
        E1[Guarantees Delivery]
        E2[Handles Retries]
        E3[Absorbs Spikes]
    end
    
    subgraph "3️⃣ Processing Plane"
        direction TB
        P1[Does Heavy Work]
        P2[Processes Events]
        P3[Writes to DB]
    end
    
    R1 --> E1 --> P1
```

### Why Separation Matters

| Plane | Responsibility | What It NEVER Does |
|-------|---------------|-------------------|
| **Request** | Validate, authenticate, publish event | Feed generation, heavy DB writes |
| **Event** | Persist events, guarantee delivery | Business logic |
| **Processing** | Fan-out, DB writes, complex logic | Handle user requests directly |

---

## Core Components

### 3.1 API Service (Go)

```mermaid
graph LR
    Client[Client] -->|HTTP Request| API[API Service]
    API -->|Validate| V{Valid?}
    V -->|Yes| Auth[Authenticate]
    V -->|No| Err1[400 Bad Request]
    Auth -->|OK| Kafka[(Kafka)]
    Auth -->|Fail| Err2[401 Unauthorized]
    Kafka --> Resp[200 OK]
    
    style API fill:#4CAF50,color:#fff
    style Kafka fill:#FF9800,color:#fff
```

**Responsibilities:**
- ✅ Validate requests
- ✅ Authenticate users
- ✅ Publish events to Kafka
- ✅ Return response immediately

**What it NEVER does:**
- ❌ Feed generation
- ❌ Heavy DB writes
- ❌ Fan-out logic

*That work belongs elsewhere.*

---

### 3.2 Kafka (Event Backbone)

> **Kafka is not a queue. Kafka is a durable, ordered, replayable log.**

```mermaid
graph TB
    subgraph "Kafka Cluster"
        direction LR
        subgraph "Topic: post-events"
            P0[Partition 0<br/>user_a, user_c]
            P1[Partition 1<br/>user_b, user_d]
            P2[Partition 2<br/>user_e, user_f]
        end
    end
    
    Producer[API Service] -->|partition by actor_id| P0
    Producer -->|partition by actor_id| P1
    Producer -->|partition by actor_id| P2
    
    P0 --> C1[Consumer 1]
    P1 --> C2[Consumer 2]
    P2 --> C3[Consumer 3]
    
    style Producer fill:#4CAF50,color:#fff
    style C1 fill:#2196F3,color:#fff
    style C2 fill:#2196F3,color:#fff
    style C3 fill:#2196F3,color:#fff
```

**Responsibilities:**
- ✅ Persist events durably
- ✅ Handle retries automatically
- ✅ Absorb traffic spikes
- ✅ Decouple producers from consumers

**Key Kafka concept:**
```
Kafka decouples ingestion rate from processing rate.
```
*Memorize this.*

---

### 3.3 Consumer Services (Feed Processors)

```mermaid
stateDiagram-v2
    [*] --> Consume: Poll Kafka
    Consume --> CheckIdempotency: Got Event
    CheckIdempotency --> Skip: Already Processed
    CheckIdempotency --> Process: New Event
    Skip --> CommitOffset
    Process --> FetchFollowers
    FetchFollowers --> FanOut: Write to each feed
    FanOut --> MarkProcessed: Insert event_id
    MarkProcessed --> CommitOffset
    CommitOffset --> [*]
```

**They can:**
- ✅ Crash
- ✅ Restart
- ✅ Scale horizontally

**And the system still works.**

---

### 3.4 Storage Layer

```mermaid
graph TB
    subgraph "PostgreSQL (Source of Truth)"
        Posts[(Posts Table)]
        Feeds[(Feeds Table)]
        Idem[(Idempotency Table)]
    end
    
    subgraph "Redis (Hot Cache)"
        HotFeeds[(Hot Feeds)]
        Timeline[(Timeline Cache)]
    end
    
    API[API Service] -->|Cache Hit| HotFeeds
    API -->|Cache Miss| Feeds
    Feeds -->|Populate| HotFeeds
    
    Processor[Feed Processor] -->|Write| Posts
    Processor -->|Fan-out| Feeds
    Processor -->|Check/Write| Idem
    Processor -->|Invalidate/Update| HotFeeds
```

**PostgreSQL stores:**
- Posts (permanent)
- Feed entries (per-user)
- Idempotency records

**Redis caches:**
- Hot feeds (frequently accessed)
- Timeline cache (recent posts)

*Same cache-aside logic as typical read-heavy systems.*

---

### 3.5 Dead Letter Queue (DLQ)

```mermaid
graph LR
    Kafka[(post-events)] --> Consumer[Feed Processor]
    Consumer -->|Success| DB[(PostgreSQL)]
    Consumer -->|Retry 1| Consumer
    Consumer -->|Retry 2| Consumer
    Consumer -->|Retry 3| Consumer
    Consumer -->|Retry N Failed| DLQ[(dead-letter-events)]
    
    DLQ --> Debug[Manual Debugging]
    DLQ --> Reprocess[Reprocessing Pipeline]
    
    style DLQ fill:#f44336,color:#fff
```

**Purpose:**
- ✅ Capture poison messages
- ✅ Prevent system stalls
- ✅ Enable debugging

> **If you don't have DLQ, your system is fragile.**

---

## Data Model

### Database Schema

```mermaid
erDiagram
    POSTS {
        bigint post_id PK "Snowflake ID"
        varchar author_id "User who created"
        text content "Post content"
        timestamp created_at "Creation time"
    }
    
    FEEDS {
        varchar user_id PK "Feed owner"
        bigint post_id PK "Post reference"
        timestamp created_at "When added to feed"
    }
    
    PROCESSED_EVENTS {
        bigint event_id PK "Snowflake ID"
        timestamp processed_at "Processing time"
    }
    
    USERS {
        varchar user_id PK
        varchar username
    }
    
    FOLLOWERS {
        varchar follower_id PK
        varchar followee_id PK
    }
    
    POSTS ||--o{ FEEDS : "appears in"
    USERS ||--o{ POSTS : "creates"
    USERS ||--o{ FOLLOWERS : "follows"
```

### SQL Schema

```sql
-- Posts Table: Stores all posts
CREATE TABLE posts (
    post_id BIGINT PRIMARY KEY,        -- Snowflake ID
    author_id VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Feeds Table: User's feed entries (fan-out target)
CREATE TABLE feeds (
    user_id VARCHAR(255) NOT NULL,
    post_id BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (user_id, post_id)
);

-- Idempotency Table: Prevents duplicate processing
CREATE TABLE processed_events (
    event_id BIGINT PRIMARY KEY,
    processed_at TIMESTAMP DEFAULT NOW()
);

-- Performance indexes
CREATE INDEX idx_feeds_user_created ON feeds(user_id, created_at DESC);
CREATE INDEX idx_posts_author ON posts(author_id);
```

---

## Event Design

> **This is where juniors fail.**

### Event Schema

```json
{
  "event_id": "7891234567890123456",
  "type": "POST_CREATED",
  "actor_id": "user_123",
  "payload": {
    "post_id": "7891234567890123457"
  },
  "timestamp": 1730000000
}
```

### Why This Structure?

| Property | Purpose |
|----------|---------|
| `event_id` | Unique identifier for idempotency |
| `type` | Enables routing and filtering |
| `actor_id` | Partition key for ordering |
| `payload` | Event-specific data |
| `timestamp` | Ordering and debugging |

**Key principles:**
- ✅ Immutable - events never change
- ✅ Replayable - can rebuild state from events
- ✅ Debuggable - contains all context
- ✅ Extensible - add fields without breaking

> **Never emit "just data". Emit facts.**

---

## Kafka Topics & Partitioning

### Topics

```mermaid
graph TB
    subgraph "Kafka Topics"
        PE[post-events<br/>📝 New posts]
        FE[feed-events<br/>📰 Feed updates]
        DLE[dead-letter-events<br/>☠️ Failed messages]
    end
    
    API[API Service] -->|publish| PE
    Processor[Feed Processor] -->|consume| PE
    Processor -->|publish| FE
    Processor -->|poison| DLE
```

### Partition Strategy

```go
partition_key = actor_id
```

**Why partition by actor_id?**

```mermaid
graph TB
    subgraph "Partition 0"
        E1[user_a: Post 1]
        E2[user_a: Post 2]
        E3[user_a: Post 3]
    end
    
    subgraph "Partition 1"
        E4[user_b: Post 1]
        E5[user_b: Post 2]
    end
    
    E1 --> E2 --> E3
    E4 --> E5
    
    Note1[✅ Ordering guaranteed per user]
    Note2[✅ Parallelism across users]
```

| Benefit | Explanation |
|---------|-------------|
| **Ordering** | All events from same user go to same partition |
| **Parallelism** | Different users processed in parallel |
| **No locks** | Partitions are independent |

**If asked "why not random partition?":**
> "Ordering matters for user actions."

---

## Write Path

### Complete Write Flow

```mermaid
sequenceDiagram
    participant Client
    participant API
    participant Kafka
    participant Processor
    participant DB
    participant Cache
    
    Client->>API: POST /posts {content}
    API->>API: Validate request
    API->>API: Generate event_id (Snowflake)
    API->>Kafka: Publish POST_CREATED event
    Kafka-->>API: ACK
    API->>Client: 200 OK (async processing)
    
    Note over Client,API: Request complete - low latency ✅
    
    Kafka->>Processor: Deliver event
    Processor->>DB: Check idempotency
    DB-->>Processor: Not processed
    Processor->>DB: Write post
    Processor->>DB: Fetch followers
    loop For each follower
        Processor->>DB: Insert into feeds
    end
    Processor->>DB: Mark event processed
    Processor->>Cache: Invalidate/Update feeds
    Processor->>Kafka: Commit offset
```

**Key insight:** Latency stays low because no heavy work happens in the request path.

---

## Read Path

### Feed Read Flow

```mermaid
sequenceDiagram
    participant Client
    participant API
    participant Redis
    participant PostgreSQL
    
    Client->>API: GET /feeds/user_123
    API->>Redis: GET feed:user_123
    
    alt Cache Hit
        Redis-->>API: [post_ids...]
        API->>Client: 200 OK {feed}
    else Cache Miss
        Redis-->>API: null
        API->>PostgreSQL: SELECT * FROM feeds WHERE user_id = ?
        PostgreSQL-->>API: [feed entries]
        API->>Redis: SET feed:user_123 {data} EX 300
        API->>Client: 200 OK {feed}
    end
```

**Same cache-aside pattern as typical read-heavy systems.**

---

## Fan-Out Strategy

### Strategy Comparison

```mermaid
graph TB
    subgraph "Fan-Out on Write"
        W1[User Posts] --> W2[Push to ALL followers]
        W2 --> W3[Each follower's feed updated]
        W4[✅ Fast reads]
        W5[❌ Heavy writes]
    end
    
    subgraph "Fan-Out on Read"
        R1[User Requests Feed] --> R2[Query all followees]
        R2 --> R3[Build feed on-the-fly]
        R4[✅ Cheap writes]
        R5[❌ Heavy reads]
    end
```

### Comparison Table

| Strategy | Writes | Reads | Best For |
|----------|--------|-------|----------|
| Fan-Out on Write | Heavy | Fast | Normal users (<10K followers) |
| Fan-Out on Read | Light | Heavy | Celebrities (1M+ followers) |
| **Hybrid** | Mixed | Mixed | **Production systems** |

### What We Implement

```mermaid
graph TD
    Post[New Post] --> Check{Author follower count?}
    Check -->|< 10,000| FanOutWrite[Fan-Out on Write]
    Check -->|> 10,000| FanOutRead[Fan-Out on Read]
    
    FanOutWrite --> Push[Push to follower feeds]
    FanOutRead --> Store[Store in author's posts only]
    
    style FanOutWrite fill:#4CAF50,color:#fff
    style FanOutRead fill:#2196F3,color:#fff
```

> This mirrors real Twitter/X tradeoffs.

---

## Idempotency

### The Problem

```mermaid
graph LR
    Kafka -->|Deliver| Consumer
    Consumer -->|Process| DB
    Consumer -->|Crash before commit| X[💥]
    Kafka -->|Redeliver| Consumer
    Consumer -->|Process AGAIN| DB
    
    Note[❌ Duplicate writes!]
```

Kafka provides **at-least-once** delivery:
- Messages may be redelivered after crash
- Retries may cause duplicates

### The Solution

```mermaid
graph TD
    Event[Consume Event] --> Check{event_id in processed_events?}
    Check -->|Yes| Skip[Skip - Already done]
    Check -->|No| Process[Process Event]
    Process --> Mark[INSERT INTO processed_events]
    Mark --> Commit[Commit Kafka Offset]
    Skip --> Commit
```

### Code Pattern

```go
func processEvent(event Event) error {
    // 1. Check if already processed
    exists, err := repo.EventProcessed(event.EventID)
    if err != nil {
        return err
    }
    if exists {
        return nil // Skip, already done
    }
    
    // 2. Do the actual work
    if err := doFanOut(event); err != nil {
        return err
    }
    
    // 3. Mark as processed
    if err := repo.MarkProcessed(event.EventID); err != nil {
        return err
    }
    
    return nil
}
```

> **"Consumers are stateless; correctness lives in the data."**
> 
> That's senior-level thinking.

---

## Failure Scenarios

### Scenario 1: Consumer Crashes

```mermaid
sequenceDiagram
    participant Kafka
    participant Consumer
    participant DB
    
    Kafka->>Consumer: Event 1
    Consumer->>DB: Write
    Consumer->>Consumer: 💥 CRASH
    Note over Consumer: Offset NOT committed
    
    Consumer->>Consumer: Restart
    Kafka->>Consumer: Event 1 (redelivered)
    Consumer->>DB: Check idempotency
    DB-->>Consumer: Already processed
    Consumer->>Consumer: Skip
    Consumer->>Kafka: Commit offset
```

**Result:** No data loss, no duplicates. ✅

---

### Scenario 2: Database Slow

```mermaid
graph LR
    Kafka[(Kafka<br/>Buffering events)] --> Consumer[Consumer<br/>Processing slowly]
    Consumer --> DB[(Database<br/>Under load)]
    
    Lag[Consumer Lag Increases]
    Buffer[Kafka Buffers Events]
    NoLoss[No Data Loss ✅]
    
    style Kafka fill:#FF9800,color:#fff
```

**Result:** Consumer lags behind, but Kafka buffers. No data loss. ✅

---

### Scenario 3: Poison Message

```mermaid
graph TD
    Kafka[(Kafka)] --> Consumer[Consumer]
    Consumer --> Try1[Retry 1 ❌]
    Try1 --> Try2[Retry 2 ❌]
    Try2 --> Try3[Retry 3 ❌]
    Try3 --> DLQ[(Dead Letter Queue)]
    DLQ --> Continue[Continue processing other events]
    
    style DLQ fill:#f44336,color:#fff
```

**Result:** Bad message quarantined, system continues. ✅

---

## Backpressure

### Why Kafka Exists

```mermaid
graph TB
    subgraph "Without Kafka"
        C1[Client Spike] --> API1[API] --> DB1[(Database)]
        DB1 --> X[💥 DB Overloaded]
    end
    
    subgraph "With Kafka"
        C2[Client Spike] --> API2[API] --> K[(Kafka)]
        K --> Consumer[Consumer]
        Consumer -->|Controlled Rate| DB2[(Database)]
        DB2 --> OK[✅ DB Protected]
    end
```

**Key sentence:**
> "Kafka decouples ingestion rate from processing rate."

*Memorize this.*

### Backpressure Flow

```mermaid
graph LR
    subgraph "Ingestion (Fast)"
        API[API] -->|1000 events/sec| Kafka[(Kafka)]
    end
    
    subgraph "Processing (Controlled)"
        Kafka -->|100 events/sec| Consumer[Consumer]
        Consumer --> DB[(Database)]
    end
    
    Buffer[Kafka absorbs the difference]
```

---

## Reliability Patterns

### Dead Letter Queue Implementation

```mermaid
flowchart TD
    Consume[Consume Message] --> Process[Process]
    Process -->|Success| Commit[Commit Offset]
    Process -->|Failure| Retry{Retry Count < Max?}
    Retry -->|Yes| Backoff[Exponential Backoff]
    Backoff --> Process
    Retry -->|No| DLQ[Send to DLQ]
    DLQ --> Commit
    
    style DLQ fill:#f44336,color:#fff
```

### Retry Policy

```go
type RetryPolicy struct {
    MaxRetries     int           // e.g., 3
    InitialBackoff time.Duration // e.g., 100ms
    MaxBackoff     time.Duration // e.g., 10s
    Multiplier     float64       // e.g., 2.0
}

// Backoff calculation
// Retry 1: 100ms
// Retry 2: 200ms
// Retry 3: 400ms
// After 3: Send to DLQ
```

---

## Observability

### Mandatory Metrics

```mermaid
graph TB
    subgraph "Prometheus Metrics"
        Lag[kafka_consumer_lag<br/>per partition]
        Duration[event_processing_duration_seconds<br/>histogram]
        Retries[event_retry_total<br/>counter]
        DLQSize[dlq_messages_total<br/>counter]
    end
    
    Consumer[Feed Processor] --> Lag
    Consumer --> Duration
    Consumer --> Retries
    Consumer --> DLQSize
    
    Lag --> Alert1[Alert if lag > threshold]
    DLQSize --> Alert2[Alert if DLQ growing]
```

### Metrics Table

| Metric | Type | Labels | Purpose |
|--------|------|--------|---------|
| `kafka_consumer_lag` | Gauge | topic, partition | Detect falling behind |
| `event_processing_duration_seconds` | Histogram | type | Performance tracking |
| `event_retry_total` | Counter | type | Reliability monitoring |
| `dlq_messages_total` | Counter | - | Failure rate |

> **If you don't monitor lag, you don't understand Kafka.**

---

## Project Structure

```
event-driven-feed/
├── cmd/
│   ├── api/
│   │   └── main.go           # API service entry point
│   └── processor/
│       └── main.go           # Feed processor entry point
│
├── config/
│   └── config.go             # Configuration management
│
├── internal/
│   ├── api/
│   │   ├── handlers/
│   │   │   ├── posts.go      # POST /posts handler
│   │   │   └── feeds.go      # GET /feeds/:id handler
│   │   ├── middleware/
│   │   │   └── auth.go       # Authentication
│   │   └── router.go         # HTTP routing
│   │
│   ├── cache/
│   │   └── redis.go          # Redis client wrapper
│   │
│   ├── dlq/
│   │   └── handler.go        # Dead Letter Queue
│   │
│   ├── events/
│   │   └── schema.go         # Event definitions
│   │
│   ├── kafka/
│   │   ├── producer.go       # Kafka producer
│   │   └── consumer.go       # Kafka consumer
│   │
│   ├── metrics/
│   │   └── prometheus.go     # Prometheus metrics
│   │
│   ├── processor/
│   │   ├── handler.go        # Event processing logic
│   │   └── retry.go          # Retry policies
│   │
│   ├── repository/
│   │   ├── posts.go          # Posts data access
│   │   ├── feeds.go          # Feeds data access
│   │   └── idempotency.go    # Processed events
│   │
│   └── snowflake/
│       └── generator.go      # ID generation
│
├── migrations/
│   └── 001_initial_schema.sql
│
├── docker-compose.yml
├── go.mod
├── go.sum
└── README.md
```

---

## Tech Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| Language | **Go** | High performance, great concurrency |
| Message Broker | **Kafka** | Event backbone, durability |
| Primary DB | **PostgreSQL** | ACID, reliable storage |
| Cache | **Redis** | Hot data, fast reads |
| Orchestration | **Docker Compose** | Local development |
| Monitoring | **Prometheus** | Metrics collection |

**No extras. No fluff.**

---

## What This Project Proves

After completing this project, interviewers infer:

| Skill | Evidence |
|-------|----------|
| ✅ Async systems | Kafka producer/consumer pattern |
| ✅ Failure reasoning | DLQ, idempotency, retries |
| ✅ Scale design | Fan-out strategies, partitioning |
| ✅ Non-blocking paths | Immediate API response |

### The Jump

> **Project 1 says:** "I build fast systems."
> 
> **Project 2 says:** "I build systems that don't fall apart."

That's the jump from student → engineer.

---

## What NOT to Build

| ❌ Skip | Why |
|---------|-----|
| Frontend | Backend focus |
| OAuth flows | Not the learning goal |
| Microservice explosion | Keep it simple |
| Kubernetes | Docker Compose is enough |

**Depth > Breadth**

---

## MVP vs Phase 2

### MVP (Build First)

- [x] Kafka producer
- [x] Consumer group
- [x] Fan-out logic
- [x] Idempotency
- [x] Basic metrics

### Phase 2 (Later)

- [ ] Dead Letter Queue
- [ ] Advanced retry policies
- [ ] Celebrity optimization (hybrid fan-out)

**Finish MVP first.**

---

## Implementation Order

| Phase | Component | Priority |
|-------|-----------|----------|
| 1 | Docker Compose + Go project | ⭐⭐⭐ |
| 2 | Database schema + repositories | ⭐⭐⭐ |
| 3 | Event schema + Kafka producer | ⭐⭐⭐ |
| 4 | API service (POST /posts) | ⭐⭐⭐ |
| 5 | Feed processor (consumer) | ⭐⭐⭐ |
| 6 | Caching layer | ⭐⭐ |
| 7 | DLQ + retries | ⭐⭐ |
| 8 | Prometheus metrics | ⭐⭐ |

---

## Key Concepts Summary

### Remember These

1. **Three Planes**: Request → Event → Processing
2. **Kafka is a log**, not a queue
3. **Partition by actor_id** for ordering
4. **Idempotency via event_id** for correctness
5. **DLQ for poison messages** for reliability
6. **Cache-aside for reads** for performance
7. **Kafka decouples ingestion from processing** for backpressure

### Interview One-Liners

- "Kafka decouples ingestion rate from processing rate."
- "Consumers are stateless; correctness lives in the data."
- "Ordering matters for user actions."
- "Fan-out on write for normal users, fan-out on read for celebrities."
