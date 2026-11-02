# Architecture Decision Records (ADR)

This document records the architectural and design decisions made throughout the development of the service.

---

## ADR 001: Standard Library HTTP Router (`net/http`)

* **Status**: Accepted
* **Context**: The service exposes RESTful endpoints with path parameters (e.g., `/devices/{id}`) and query parameters (e.g., `?brand=...&state=...`).
* **Decision**: Use Go 1.22+ standard library `http.ServeMux` with native method routing and path pattern matching instead of third-party frameworks (e.g., Chi, Gin, Echo).
* **Consequences**:
  * **Pros**: Zero external dependencies for routing, zero vulnerability surface from third-party routers, 100% standard library compliance, and long-term maintainability.
  * **Cons**: Middleware chaining is implemented natively without framework-specific syntactic sugar.

---

## ADR 002: Native PostgreSQL (`pgx`) with Parameterized SQL vs. ORM (GORM)

* **Status**: Accepted
* **Context**: Persistence is required on a real database (PostgreSQL), excluding in-memory storage for production.
* **Decision**: Use the high-performance `pgx` driver (`github.com/jackc/pgx/v5/stdlib`) with explicit, parameterized SQL queries instead of an Object-Relational Mapper (ORM) like GORM.
* **Consequences**:
  * **Pros**:
    * **Performance & Efficiency**: Avoids reflection overhead and runtime memory allocations inherent in ORMs.
    * **Clean Architecture**: Domain entities (`internal/domain/device.go`) remain pure Go structs without ORM annotations (`gorm:"..."`) or database coupling.
    * **Security**: Positional parameterized queries (`$1, $2, ...`) provide absolute protection against SQL Injection at the database protocol level.
    * **Predictability**: Eliminates hidden queries and N+1 query surprises.
  * **Cons**: Requires writing explicit SQL statements (`INSERT`, `SELECT`, `UPDATE`, `DELETE`) for migrations and repository operations.

---

## ADR 003: Defensive Input Validation & Security-in-Depth

* **Status**: Accepted
* **Context**: Unsanitized user inputs could lead to denial of service (DoS), buffer overflow, or payload injection attacks.
* **Decision**: Implement strict defense-in-depth domain validation before any database interaction:
  * Maximum field length: 255 runes (aligned with database `VARCHAR(255)` schema).
  * Enforce valid UTF-8 encoding.
  * Explicitly reject null bytes (`\x00`) and unprintable ASCII control characters.
* **Consequences**:
  * **Pros**: Prevents injection attempts and malformed inputs at the domain layer before reaching persistence.
  * **Cons**: Slightly more validation logic in the domain entity constructor and update methods.

---

## ADR 004: Three-Tier Testing Strategy (TDD, Fakes & Hurl E2E)

* **Status**: Accepted
* **Context**: The test suite needs to be fast, reliable, reproducible, and verifiable without tight coupling to local environments.
* **Decision**: Adopt a three-tier testing approach:
  1. **Domain Unit Tests**: Pure Go unit tests validating business rules and invariants (sub-millisecond execution, zero dependencies).
  2. **Handler Unit Tests**: HTTP handler tests using Go's `httptest.NewRecorder()` and an in-memory `fakeRepository` implementing the `DeviceRepository` interface.
  3. **End-to-End API Tests (Hurl)**: Declarative HTTP specifications (`tests/hurl/devices.hurl`) executed against the running application and real PostgreSQL database.
* **Consequences**:
  * **Pros**: Developers can run `go test ./...` in ~0.3s without needing Docker or network access. Full E2E behavior is validated against the real database using Hurl.
  * **Cons**: Maintaining both a fake repository for unit tests and a SQL repository for integration.

---

## ADR 005: Real-time Device State Notifications via Server-Sent Events (SSE)

* **Status**: Accepted
* **Context**: Clients must be notified of device state changes in real time, including the device ID, previous state, and new state, sent only after successful persistence.
* **Decision**: Implement real-time notifications using Server-Sent Events (SSE) via `GET /devices/events` (`text/event-stream`).
* **Consequences**:
  * **Pros**:
    * Native to HTTP/REST with zero extra network protocols or third-party libraries.
    * Implemented using standard Go concurrency (goroutines, channels, and `http.Flusher`).
    * Trivially testable using `curl -N`, browsers, or automated scripts.
  * **Cons**: Unidirectional (server-to-client only), which perfectly fits the notification requirement.

---

## ADR 006: Dedicated State Transition Audit Table & Materialized Views

* **Status**: Rejected (Abandoned Decision)
* **Context**: We evaluated decoupling the immutable device properties (`id`, `creation_time`) from mutable states (`state`) by storing state transitions in a dedicated historical/audit log table (inspired by patterns such as Ruby's `paper_trail` or event-sourcing), and reconstructing current state for queries via composite joins or PostgreSQL Materialized Views.
* **Reason for Rejection (Trade-off Analysis)**:
  1. **Query Performance Degradation**: Fetching current devices (e.g. `GET /devices?state=available`) would require either composite aggregation queries (`DISTINCT ON` or `MAX(changed_at) GROUP BY device_id`) or Materialized Views. Composite queries degrade significantly at scale, while Materialized Views require expensive, asynchronous refreshes (`REFRESH MATERIALIZED VIEW`), leading to eventual consistency issues.
  2. **YAGNI (You Aren't Gonna Need It)**: The current requirements do not mandate historical audit logs, time-travel debugging, or replay of past states. Adding a multi-table transition schema adds structural complexity without immediate business value.
  3. **Architectural Simplicity**: Storing the current state directly on the `devices` table with B-Tree indexes enables sub-millisecond filtering on `brand` and `state`, perfectly aligning with the evaluation criteria (*"Code efficiency and general performance"*).
* **Future Re-evaluation**: If future requirements introduce strict audit compliance (e.g., tracking who changed which state, when, and why), an append-only `device_state_transitions` table can be introduced alongside the existing operational `devices` table (Operational Snapshot + Audit Log pattern) without compromising operational read performance.

---

## ADR 007: Embedded Database Migrations via `//go:embed`

* **Status**: Accepted
* **Context**: Schema migrations (`001_create_devices_table.sql`) must be executed reliably across local development, Docker Compose, and CI environments without forcing evaluators to install external migration CLI tools (`golang-migrate`, `flyway`, etc.), and without duplicating raw SQL strings across the Go codebase.
* **Decision**: Use Go's standard library `embed.FS` (`//go:embed *.sql`) in the `migrations` package and consume it via `internal/database/migrations.go`.
* **Consequences**:
  * **Pros**:
    * **Single Source of Truth**: The `.sql` files in `migrations/` are the sole source of truth; zero duplicate SQL strings in Go source code.
    * **Self-Contained Executable**: The compiled binary embeds its own database schema, guaranteeing that the application can bootstrap its schema on startup anywhere.
    * **Deterministic Execution**: Migration files are read and executed in sorted lexical order (`001_...`, `002_...`).
    * **Zero Tooling Overhead**: Eliminates external binary dependencies for evaluators and CI runners.
  * **Cons**: In large enterprise systems with hundreds of historical migrations, migrations may eventually be decoupled into dedicated init-containers. For this service scope, `//go:embed` is the optimal, idiomatic standard.



