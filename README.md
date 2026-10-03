# short-go 🚀

A production-grade, highly performant, and low-latency URL Shortener and Analytics API service written in idiomatic Go adhering to **Clean Architecture / Hexagonal Architecture** principles.

[![Go Report Card](https://goreportcard.com/badge/short-go)](https://goreportcard.com/report/short-go)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?style=flat&logo=postgresql)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=flat&logo=redis)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=flat&logo=docker)](https://www.docker.com/)

---

## 🏛️ System Architecture

`short-go` is designed for ultra-low latency redirection ($O(1)$ cache hits via Redis) and resilient telemetry persistence without slowing down the HTTP redirect lifecycle.

```mermaid
flowchart TD
    Client([Client / Browser])
    
    subgraph HTTP Layer [HTTP / Routing Layer]
        Router[Gin Engine & Middlewares]
        Handler[URL Handler]
    end

    subgraph Service Layer [Domain & Application Logic]
        Service[URL Service]
        Base62[Base62 Codec Engine]
    end

    subgraph Caching Layer [In-Memory Cache]
        Redis[(Redis 7 Cache)]
    end

    subgraph Persistence Layer [Relational Storage]
        Postgres[(PostgreSQL 16)]
    end

    subgraph Concurrency [Asynchronous Analytics Pipeline]
        Buffer[Buffered Event Channel]
        Workers[Bounded Worker Pool]
    end

    %% Redirection Flow
    Client -->|1. GET /:shortCode| Router
    Router --> Handler
    Handler -->|2. Resolve| Service
    Service -->|3. Cache-Aside Check| Redis
    Redis -.->|Cache Miss| Postgres
    Service -->|4. Non-blocking Dispatch| Buffer
    Buffer --> Workers
    Workers -->|Batch Write (Zero-Drop)| Postgres
    Handler -->|5. 302 Found Redirect| Client

    %% Shorten Flow
    Client -->|POST /api/v1/shorten| Router
    Service -->|Reserve ID & Base62 Encode| Base62
    Service -->|Insert Mapping| Postgres
    Service -->|Eager Warm| Redis
```

---

## ⚡ Key Features

- **Clean / Hexagonal Architecture**: Domain models and port interfaces are completely decoupled from external frameworks and database drivers.
- **Base62 Deterministic Encoding**: Generates collision-free, human-friendly short codes derived atomically from PostgreSQL identity sequences (seeded at `1,000,000` to prevent single-character codes).
- **Cache-Aside Caching**: High-throughput read path resolves URLs directly from Redis; on cache miss, it populates Redis with configurable TTL.
- **Asynchronous & Non-Blocking Analytics**: Click events are dispatched to a buffered channel without adding latency to the client's HTTP redirect.
- **Bounded Worker Pool with Batching**: Micro-batches analytics logs into multi-row SQL transactions, drastically reducing database connection contention.
- **Zero-Drop Graceful Shutdown**: Listens for `SIGINT`/`SIGTERM`, stops accepting incoming traffic, and safely drains all queued click events before shutting down database connections.
- **Strict Error Propagation**: Idiomatic Go errors with typed domain sentinels and no production panics.

---

## 📁 Project Structure

```
.
├── cmd/
│   └── api/
│       └── main.go                 # Application entrypoint & dependency injection
├── internal/
│   ├── config/                     # Environment configuration loader
│   ├── domain/                     # Core entities, value objects, domain errors, ports
│   │   ├── analytics.go
│   │   ├── click.go
│   │   ├── errors.go
│   │   ├── repository.go
│   │   ├── service.go
│   │   └── url.go
│   ├── handler/                    # Gin controllers, router setup, middlewares
│   │   ├── middleware.go
│   │   ├── router.go
│   │   └── url_handler.go
│   ├── platform/                   # Infrastructure drivers & concurrency workers
│   │   ├── postgres/               # PostgreSQL connection pool & migrations
│   │   ├── redis/                  # Redis client initialization
│   │   └── worker/                 # Bounded analytics worker pool
│   ├── repository/                 # Concrete storage implementations
│   │   ├── postgres/               # Raw SQL PostgreSQL repositories
│   │   │   ├── analytics_repository.go
│   │   │   └── url_repository.go
│   │   └── redis/                  # Redis cache repository
│   │       └── url_cache_repository.go
│   └── service/                    # Business use cases implementation
│       └── url_service.go
├── migrations/                     # Raw SQL DDL schema migrations
│   ├── 000001_init_schema.up.sql
│   └── 000001_init_schema.down.sql
├── pkg/
│   └── base62/                     # Zero-allocation Base62 codec & unit tests
│       ├── base62.go
│       └── base62_test.go
├── .env.example
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
└── README.md
```

---

## 🚀 Quickstart

### Prerequisites
- [Docker & Docker Compose](https://docs.docker.com/compose/) **OR**
- [Go 1.24+](https://go.dev/dl/) installed locally with PostgreSQL 16 & Redis running.

### 1. Run with Docker Compose (Recommended)

Start all services (PostgreSQL 16, Redis 7, and the API container) with a single command:

```bash
docker compose up --build -d
```

The database migrations will run automatically on first boot. The API will be available at `http://localhost:8080`.

To view service logs:
```bash
docker compose logs -f api
```

To stop all services:
```bash
docker compose down -v
```

---

### 2. Local Development

1. Copy `.env.example` to `.env`:
   ```bash
   cp .env.example .env
   ```

2. Start PostgreSQL and Redis:
   ```bash
   docker compose up -d postgres redis
   ```

3. Run the service:
   ```bash
   go run ./cmd/api
   ```

---

## 📡 API Reference

### 1. Shorten URL
`POST /api/v1/shorten`

Creates a shortened URL with an optional custom alias and expiration timestamp.

#### Request
```bash
curl -X POST http://localhost:8080/api/v1/shorten \
  -H "Content-Type: application/json" \
  -d '{
    "original_url": "https://github.com/golang/go/wiki/Performance",
    "custom_alias": "goperf",
    "expires_at": "2027-01-01T00:00:00Z"
  }'
```

#### Response (`201 Created`)
```json
{
  "status": "success",
  "data": {
    "id": 1000001,
    "short_code": "goperf",
    "short_url": "http://localhost:8080/goperf",
    "original_url": "https://github.com/golang/go/wiki/Performance",
    "created_at": "2026-10-03T16:45:00Z",
    "expires_at": "2027-01-01T00:00:00Z"
  }
}
```

---

### 2. Redirection
`GET /:shortCode`

High-throughput redirection. Validates URL freshness and queues an asynchronous click audit.

#### Request
```bash
curl -i http://localhost:8080/goperf
```

#### Response (`302 Found`)
```http
HTTP/1.1 302 Found
Location: https://github.com/golang/go/wiki/Performance
Content-Type: text/html; charset=utf-8
```

---

### 3. Click Analytics
`GET /api/v1/analytics/:shortCode`

Fetches real-time click statistics, top referrers, and recent access logs.

#### Request
```bash
curl -X GET http://localhost:8080/api/v1/analytics/goperf
```

#### Response (`200 OK`)
```json
{
  "status": "success",
  "data": {
    "short_code": "goperf",
    "original_url": "https://github.com/golang/go/wiki/Performance",
    "total_clicks": 142,
    "created_at": "2026-10-03T16:45:00Z",
    "last_accessed_at": "2026-10-03T16:50:23Z",
    "top_referrers": [
      {
        "referrer": "https://twitter.com",
        "clicks": 95
      },
      {
        "referrer": "Direct / Unknown",
        "clicks": 47
      }
    ],
    "recent_clicks": [
      {
        "id": 12,
        "url_id": 1000001,
        "short_code": "goperf",
        "clicked_at": "2026-10-03T16:50:23Z",
        "ip_address": "127.0.0.1",
        "user_agent": "Mozilla/5.0 ...",
        "referrer": "https://twitter.com"
      }
    ]
  }
}
```

---

### 4. Health Check
`GET /health`

Liveness and readiness probe for container orchestrators (Kubernetes / Docker).

#### Request
```bash
curl -X GET http://localhost:8080/health
```

#### Response (`200 OK`)
```json
{
  "env": "development",
  "service": "short-go",
  "status": "ok"
}
```

---

## ⚙️ Environment Variables

| Variable | Default | Description |
| :--- | :--- | :--- |
| `SERVER_PORT` | `8080` | Port for the HTTP API server |
| `SERVER_ENV` | `development` | Environment (`development` or `production`) |
| `BASE_URL` | `http://localhost:8080` | Base URL used to prefix shortened URLs |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `postgres` | PostgreSQL username |
| `DB_PASSWORD` | `postgres` | PostgreSQL password |
| `DB_NAME` | `shortgo` | Database name |
| `DB_MAX_OPEN_CONNS` | `25` | Maximum open database connections |
| `DB_MAX_IDLE_CONNS` | `10` | Maximum idle connections in pool |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_DEFAULT_TTL` | `24h` | Default cache expiration duration |
| `WORKER_POOL_SIZE` | `5` | Number of concurrent background batch workers |
| `WORKER_BUFFER_SIZE`| `10000` | Size of the asynchronous event channel |
| `WORKER_BATCH_SIZE` | `100` | Maximum events before executing batch DB write |
| `WORKER_FLUSH_INTERVAL` | `2s` | Maximum time before pending events are flushed |

---

## 🧪 Testing & Benchmarks

Run Base62 unit tests and benchmark throughput:

```bash
go test -v "short-go/pkg/base62"
go test -bench="." "short-go/pkg/base62"
```

Sample Benchmark Output:
```
BenchmarkEncode-16    21506994    59.39 ns/op
BenchmarkDecode-16    23726762    50.33 ns/op
```

---

## 🚢 Pushing to GitHub

To push this repository to GitHub:

```bash
# 1. Initialize git (if not already initialized)
git init

# 2. Stage all files
git add .

# 3. Commit changes
git commit -m "feat: initial commit of production-grade short-go service"

# 4. Link your GitHub remote repository
git remote add origin https://github.com/<your-username>/short-go.git

# 5. Rename default branch to main and push
git branch -M main
git push -u origin main
```

---

## 📄 License
This project is open-source and available under the [MIT License](LICENSE).
