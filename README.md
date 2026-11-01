# Devices Management API (`tools_crud_go`)

A production-grade, containerized REST API in Go capable of persisting and managing device resources with real-time state change notifications via Server-Sent Events (SSE).

[![Go Version](https://img.shields.io/badge/Go-1.23%2B-00ADD8?style=flat&logo=go)](https://golang.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?style=flat&logo=postgresql)](https://www.postgresql.org)
[![Hurl Test](https://img.shields.io/badge/E2E%20Tests-Hurl-FF6F00?style=flat)](https://hurl.dev)
[![Container Image](https://img.shields.io/badge/Container-GHCR-24292e?style=flat&logo=github)](https://github.com/dede999/tools_crud_go/pkgs/container/tools_crud_go)

---

## 🏛️ Architecture Overview

The project adheres to Clean Architecture and Hexagonal (Ports & Adapters) principles, decoupling core business logic from database drivers and transport protocols:

```text
tools_crud_go/
├── cmd/
│   └── api/                     # Application entrypoint & HTTP server lifecycle
│       └── main.go
├── internal/
│   ├── domain/                  # Pure enterprise business rules & invariants (zero external dependencies)
│   │   ├── device.go
│   │   └── device_test.go       # Pure domain unit tests
│   ├── handler/                 # Standard library net/http REST handlers & SSE
│   │   ├── device_handler.go
│   │   └── device_handler_test.go # Fast in-memory handler tests
│   ├── notifier/                # Real-time Pub/Sub broker for Server-Sent Events
│   │   ├── notifier.go
│   │   └── notifier_test.go
│   └── repository/              # PostgreSQL persistence layer (pgx with parameterized SQL)
│       ├── repository.go        # DeviceRepository interface
│       └── postgres.go
├── migrations/                  # Self-bootstrapping database schema migrations
│   └── 001_create_devices_table.sql
├── tests/
│   └── hurl/                    # End-to-end API test specifications
│       └── devices.hurl
├── docs/
│   ├── openapi.yaml             # Complete OpenAPI 3.0 specification
│   └── architecture-decisions.md # Architecture Decision Records (ADR 001 - 006)
├── .github/workflows/           # CI/CD pipeline (Unit Tests, Race Detector, Hurl E2E, GHCR Push)
│   └── ci.yml
├── Dockerfile                   # Multi-stage build producing a minimal, secure container
└── docker-compose.yml           # Full-stack composition (API + PostgreSQL)
```

---

## ⚡ Engineering Highlights & Idiomatic Go Patterns

* **Compile-Time Interface Verification**:
  Interfaces are validated at compile-time using static assertions (e.g. `var _ Notifier = (*EventBroker)(nil)` and `var _ repository.DeviceRepository = (*PostgresRepository)(nil)`). This eliminates runtime mismatch bugs with zero memory allocation.
* **Non-Blocking Pub/Sub with Slow-Consumer Protection**:
  The Server-Sent Events broker uses buffered client channels and non-blocking writes (`select { case ch <- event: default: }`). This guarantees that slow consumers or network hiccups never block API mutations or cause head-of-line blocking for other users.
* **Defense-in-Depth against Injection & Memory Overflows**:
  * **Domain Layer**: Input sanitization enforces 255-rune bounds, validates UTF-8 integrity, and strictly rejects null bytes (`\x00`) and unprintable ASCII control characters.
  * **Persistence Layer**: Queries use positional parameters exclusively (`$1, $2, ...`) via the high-performance `pgx` driver, preventing SQL injection at the database protocol level.
* **Zero-Dependency Routing**:
  Leverages Go 1.22+ native `http.ServeMux` routing with pattern matching (`POST /devices`, `GET /devices/{id}`, `PATCH /devices/{id}`), keeping the dependency tree clean and secure.
* **Post-Persistence Notification Guarantee**:
  State change events are dispatched to subscribed clients strictly **after** the update transaction is successfully committed to PostgreSQL.

---

## 🚀 Quickstart

### Prerequisites

* [Docker](https://docs.docker.com/get-docker/) & [Docker Compose](https://docs.docker.com/compose/)
* (Optional for local dev) [Go 1.23+](https://golang.org/dl/) and [Hurl](https://hurl.dev)

### Option 1: Run with Docker Compose (Recommended)

Start the API and PostgreSQL in a single command:

```bash
docker compose up --build
```

The service will:
1. Boot PostgreSQL 16 Alpine and run schema migrations automatically.
2. Compile and launch the API container as a non-root user.
3. Expose the API on `http://localhost:8080`.

Verify health:
```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

### Option 2: Run Locally with Local PostgreSQL

```bash
# 1. Start PostgreSQL
docker compose up -d postgres

# 2. Run the application
go run cmd/api/main.go
```

---

## 🧪 Testing Strategy

We follow a three-tier testing approach:

```text
┌────────────────────────────────────────────────────────┐
│ 1. go test -race ./... (Runs in ~0.3s without Docker)  │
│  - Pure Domain Unit Tests (business rules, boundaries) │
│  - Handler Unit Tests (httptest.NewRecorder + fakes)   │
│  - Event Broker Concurrency & Subscription Tests       │
└────────────────────────────────────────────────────────┘
                           ▲
                           │ complement
                           ▼
┌────────────────────────────────────────────────────────┐
│ 2. Hurl E2E Suite (Runs against real running API + DB) │
│  - Full lifecycle: POST -> GET -> PATCH -> DELETE      │
│  - Real-time asserts on headers, status, JSON schemas  │
│  - Verifies business rule violations (409 Conflict)    │
└────────────────────────────────────────────────────────┘
```

### Running Unit Tests

```bash
go test -v -race ./...
```

### Running End-to-End Tests with Hurl

With the application running (`docker compose up` or `go run cmd/api/main.go`):

```bash
hurl --test --color tests/hurl/devices.hurl
```

---

## 📡 API Reference

| Method | Endpoint | Description | Success Status | Error Statuses |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/health` | Service health check | `200 OK` | `500` |
| `POST` | `/devices` | Create a new device | `201 Created` | `400` |
| `GET` | `/devices/{id}` | Fetch a single device by ID | `200 OK` | `400`, `404` |
| `GET` | `/devices` | List devices (`?brand=...&state=...`) | `200 OK` | `400` |
| `PUT` | `/devices/{id}` | Full update (creation_time immutable) | `200 OK` | `400`, `404`, `409` (if in-use) |
| `PATCH` | `/devices/{id}` | Partial update (state or details) | `200 OK` | `400`, `404`, `409` (if in-use) |
| `DELETE` | `/devices/{id}` | Delete device (blocked if in-use) | `204 No Content` | `404`, `409` (if in-use) |
| `GET` | `/devices/events` | Stream state changes via SSE | `200 OK` | `500` |

### Listening to Real-time Events (SSE)

Connect using `curl`:

```bash
curl -N http://localhost:8080/devices/events
```

Output:
```http
event: connected
data: {}

event: state_changed
data: {"device_id":"d1","previous_state":"available","new_state":"in-use","timestamp":"2026-11-01T20:00:00Z"}
```

Complete API schema and response samples are available in the [OpenAPI 3.0 Specification](docs/openapi.yaml).

---

## 📋 Architecture Decisions (ADR)

Detailed decision records documenting trade-offs are available in [`docs/architecture-decisions.md`](docs/architecture-decisions.md):

* **ADR 001**: Standard library `net/http` ServeMux routing over third-party routers.
* **ADR 002**: Native PostgreSQL with `pgx` and parameterized SQL over GORM.
* **ADR 003**: Defensive input validation and security-in-depth.
* **ADR 004**: Three-tier testing strategy (TDD, in-memory fakes, Hurl E2E).
* **ADR 005**: Real-time state change notifications via Server-Sent Events (SSE).
* **ADR 006 (Rejected)**: Dedicated state transition audit table & materialized views (rationale on query performance degradation vs. YAGNI).

---

## 🚢 CI/CD & Published Container Image

Every push to `main` triggers a complete GitHub Actions workflow (`.github/workflows/ci.yml`):
1. **Unit & Race Detection**: Runs `go test -race ./...`.
2. **E2E Integration Testing**: Provisions a PostgreSQL service container, starts the API, and runs the entire **Hurl** test suite.
3. **Container Publication**: Compiles the multi-stage Docker image and publishes it to the GitHub Container Registry.

Pull the published image directly:

```bash
docker pull ghcr.io/dede999/tools_crud_go:latest
```

---

## 🔮 Future Improvements & Production Considerations

* **Transactional Outbox Pattern**: In a distributed multi-replica deployment, state change events can be written to an outbox table in the same database transaction, followed by publishing to an external message broker (such as Apache Kafka, RabbitMQ, or NATS) to ensure reliable delivery across multiple API instances.
* **Authentication & Authorization**: Integration with OpenID Connect (OIDC) / OAuth2 JWT bearer tokens to secure administrative operations.
* **Rate Limiting**: Token-bucket rate limiting middleware per client IP to mitigate denial-of-service abuse.
* **Observability**: Exporting structured OpenTelemetry metrics (request latency, database connection pool statistics) and distributed traces to Prometheus/Grafana or Cloud Trace.

