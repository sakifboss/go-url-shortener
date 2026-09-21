# GoShort

Production-oriented URL shortener backend built with Go, PostgreSQL, Redis, JWT authentication, and an asynchronous click-event worker pool.

GoShort demonstrates practical backend engineering patterns in a small, focused service:

- Layered HTTP, service, repository, and persistence design
- User registration, login, refresh-token rotation, and logout
- JWT-protected URL management with ownership checks
- Redis cache-aside lookups with invalidation on writes
- Bounded asynchronous click-event processing
- Retry with exponential backoff
- Per-user idempotency for URL creation
- Request-size limits, security headers, and IP-based rate limiting
- Graceful HTTP server and worker shutdown

## Status

The current implementation is functional and focused on the core URL-shortening workflow. Analytics queries, observability, and load testing remain future work.

Live deployment: https://go-url-shortener-yi2j.onrender.com

## Architecture

GoShort uses a layered architecture with explicit boundaries between transport, business logic, persistence, infrastructure, and background processing.

```text
+-----------------------------------------------------------------------+
|                         HTTP Application                              |
|                                                                       |
|  Security middleware -> Rate limiter -> Router -> Auth middleware     |
|                                                   |                   |
|                         +-------------------------+----------------+  |
|                         |                                          |  |
|                    Auth handlers                           URL handlers|
|                         |                                          |  |
|                    Auth service                         URL service   |
|                         |                              /          |   |
|                         |                             /           |   |
|                         +----------------------------+            |   |
|                                      |                              |   |
|                                Repositories                         |   |
+--------------------------------------+-------------------------------+
                                       |
                          +------------+------------+
                          |                         |
                    PostgreSQL                 Redis cache
                          ^
                          |
                  Click-event worker pool
                          ^
                          |
                    Bounded channel
```

### Layer responsibilities

| Layer | Responsibility | Main packages |
| --- | --- | --- |
| Composition root | Builds dependencies, registers routes, starts and stops the application | `main.go` |
| Transport | Validates HTTP methods and JSON, maps requests to services, writes responses | `handler/` |
| Authentication | Hashes passwords, issues JWT access tokens, rotates and revokes refresh tokens | `auth/` |
| Business logic | Creates, resolves, updates, and deletes shortened URLs | `service/` |
| Persistence | Encapsulates SQL queries and database-specific storage behavior | `repository/`, `database/` |
| Caching | Reads, writes, and invalidates URL entries in Redis | `cache/` |
| Security | Applies request limits, security headers, and client-IP rate limiting | `security/` |
| Background processing | Persists click events with bounded queuing and retry/backoff | `worker/` |
| Domain models | Defines the data exchanged between application layers | `model/` |

### URL creation flow

```text
Client
  |
  | POST /api/v1/urls + Bearer token
  v
Global security middleware
  |
  v
JWT authentication and user context
  |
  v
URL handler
  |
  +--> Idempotency repository lookup (when Idempotency-Key is present)
  |
  v
Cached URL service
  |
  v
URL repository --> PostgreSQL
  |
  +--> Store idempotent response
  +--> Invalidate related Redis entry when required
  v
JSON response
```

### Redirect and click-event flow

```text
Client
  |
  | GET /{short_code}
  v
URL service
  |
  +--> Redis cache hit --------------------------+
  |                                             |
  +--> Cache miss --> URL repository --> Redis --+
                                                |
                                                v
                                      HTTP 302 redirect
                                                |
                                                +--> Click event
                                                       |
                                                       v
                                              Bounded channel
                                                       |
                                                       v
                                                 Worker pool
                                                       |
                                                       v
                                           Retry with backoff
                                                       |
                                                       v
                                               Click repository
                                                       |
                                                       v
                                                 PostgreSQL
```

The redirect response does not wait for click-event persistence. The worker pool uses four workers, a 1,000-event bounded queue, up to five attempts, and a 100 ms initial backoff.

### Authentication flow

```text
Register -> Password hash -> PostgreSQL

Login -> Verify password -> JWT access token + refresh token

Refresh -> Hash and validate refresh token -> Rotate token pair

Logout -> Revoke refresh token

Protected request -> Validate JWT -> Attach current user to request context
```

## Technology Stack

| Area | Technology |
| --- | --- |
| Language | Go 1.26.4 |
| HTTP server | Go `net/http` |
| Database | PostgreSQL |
| Database driver | pgx via `database/sql` |
| Cache | Redis |
| Authentication | JWT access tokens and stored refresh tokens |
| Password security | `golang.org/x/crypto/bcrypt` |
| Migrations | Ordered SQL files |
| Testing | Go standard testing package |
| Concurrency | Goroutines, channels, and `sync` |

## Project Structure

```text
.
├── main.go
├── go.mod
├── go.sum
│
├── auth/
│   ├── auth_middleware.go       # JWT validation and request user context
│   ├── auth_service.go          # Register, login, refresh, and logout
│   ├── password.go              # Password hashing and verification
│   └── *_test.go
│
├── cache/
│   └── redis.go                 # Redis client and URL cache operations
│
├── database/
│   └── postgres.go              # PostgreSQL connection-pool setup
│
├── handler/
│   ├── auth_handler.go          # Authentication HTTP endpoints
│   ├── url_handler.go           # URL CRUD, redirects, and click enqueueing
│   └── json.go                  # Shared JSON request/response helpers
│
├── migration/
│   ├── 001_init.sql             # Users and URLs
│   ├── 002_auth.sql             # Authentication fields and refresh tokens
│   ├── 003_click_events.sql     # Click-event storage and indexes
│   ├── 004_idempotency.sql      # Idempotent request records
│   ├── 005_idempotency_reservations.sql # Cross-instance reservations
│   └── 006_custom_alias.sql     # Optional user-selected aliases
│
├── model/
│   ├── click_event.go
│   ├── refresh_token.go
│   ├── url.go
│   └── user.go
│
├── repository/
│   ├── url_repository.go
│   ├── user_repository.go
│   ├── refresh_token_repository.go
│   ├── click_event_repository.go
│   └── idempotency_repository.go
│
├── security/
│   ├── middleware.go             # Request limits and security headers
│   └── rate_limiter.go           # In-memory client-IP rate limiter
│
├── service/
│   ├── url_service.go            # Core URL business logic
│   └── cached_url_service.go     # Cache-aside URL service decorator
│
└── worker/
  └── click_worker.go           # Queue, workers, retries, and shutdown
```

`main.go` acts as the composition root: it constructs infrastructure clients, repositories, services, handlers, middleware, and the worker pool. Lower layers do not create HTTP servers or read request-specific transport concerns, which keeps the core components independently testable.

## Prerequisites

- Go 1.26.4 or a compatible newer Go toolchain
- PostgreSQL
- Redis
- Git, if cloning the repository

The application expects PostgreSQL and Redis to be available before startup. Redis can be run locally or in Docker.

## Configuration

Required environment variables:

```powershell
$env:GOSHORT_DATABASE_URL = "postgres://postgres:<PASSWORD>@localhost:5432/goshort?sslmode=disable"
$env:GOSHORT_JWT_SECRET = "replace-with-a-long-development-secret"
```

Optional environment variables:

```powershell
$env:GOSHORT_REDIS_ADDR = "localhost:6379"
$env:GOSHORT_REDIS_PASSWORD = ""
$env:GOSHORT_REDIS_DB = "0"
$env:GOSHORT_PUBLIC_BASE_URL = "http://localhost:9000"
```

For hosted Redis providers, `GOSHORT_REDIS_URL` or `REDIS_URL` may be used
instead of separate Redis host and password variables. The URL must use the
provider's complete Redis connection URL.

Defaults:

- Redis address: `localhost:6379`
- Redis database: `0`
- Public base URL: `http://localhost:9000`
- HTTP server address: `:9000`
- URL cache TTL: one hour, or the remaining URL lifetime when shorter

Do not commit real credentials or production JWT secrets.

## Database Setup

Create the `goshort` PostgreSQL database, then apply migrations in order:

```text
migration/001_init.sql
migration/002_auth.sql
migration/003_click_events.sql
migration/004_idempotency.sql
migration/005_idempotency_reservations.sql
migration/006_custom_alias.sql
```

For example, with `psql`:

```powershell
psql "$env:GOSHORT_DATABASE_URL" -f migration/001_init.sql
psql "$env:GOSHORT_DATABASE_URL" -f migration/002_auth.sql
psql "$env:GOSHORT_DATABASE_URL" -f migration/003_click_events.sql
psql "$env:GOSHORT_DATABASE_URL" -f migration/004_idempotency.sql
psql "$env:GOSHORT_DATABASE_URL" -f migration/005_idempotency_reservations.sql
psql "$env:GOSHORT_DATABASE_URL" -f migration/006_custom_alias.sql
```

The schema stores users, URLs, hashed refresh tokens, click events, and idempotency records. The `(user_id, idempotency_key)` uniqueness constraint prevents duplicate idempotency records for the same user. A pending idempotency reservation is created before URL creation, preventing duplicate URL creation across multiple application instances.

## Run Locally

From the repository root:

```powershell
go mod download
go run .
```

The server listens on `http://localhost:9000`.

### Run with Docker Compose

Docker Compose starts the GoShort app, PostgreSQL, Redis, and all migrations:

```powershell
docker compose up --build
```

Health check:

```powershell
Invoke-WebRequest http://localhost:9000/health
```

Stop the stack:

```powershell
docker compose down
```

Migrations run automatically only when the PostgreSQL volume is initialized. To
recreate the local database and run all migrations from the beginning:

```powershell
docker compose down -v
docker compose up --build
```

The Compose file uses development-only credentials. Set a strong
`GOSHORT_JWT_SECRET` before using this setup outside local development.

Health check:

```powershell
Invoke-WebRequest http://localhost:9000/health
```

Expected response body:

```text
OK
```

## API Reference

### Authentication

#### Register

```http
POST /api/v1/auth/register
Content-Type: application/json
```

```json
{
  "email": "user@example.com",
  "password": "a-strong-password"
}
```

Returns the created user with `201 Created`.

#### Login

```http
POST /api/v1/auth/login
Content-Type: application/json
```

```json
{
  "email": "user@example.com",
  "password": "a-strong-password"
}
```

The response contains an access token, refresh token, and `Bearer` token type. Login is protected by a stricter rate limit of 10 requests per client IP per minute.

#### Refresh tokens

```http
POST /api/v1/auth/refresh
Content-Type: application/json
```

```json
{
  "refresh_token": "<refresh-token>"
}
```

#### Logout

```http
POST /api/v1/auth/logout
Content-Type: application/json
```

```json
{
  "refresh_token": "<refresh-token>"
}
```

### URLs

URL management endpoints require:

```http
Authorization: Bearer <access-token>
```

#### Create a short URL

```http
POST /api/v1/urls
Content-Type: application/json
Idempotency-Key: create-url-001
```

```json
{
  "url": "https://example.com/a-long-resource",
  "alias": "docs"
}
```

Response:

```json
{
  "id": 1,
  "short_code": "docs",
  "short_url": "http://localhost:9000/docs"
}
```

`Idempotency-Key` is optional. When supplied, repeating the request for the same authenticated user returns the stored response instead of creating another URL.
The `alias` field is optional. Aliases must be 3-32 characters and contain only letters, numbers, hyphens, or underscores. An alias can only be used once.

#### URL analytics

```http
GET /api/v1/urls/{id}/analytics
Authorization: Bearer <access-token>
```

The analytics response includes total clicks, device breakdown, referrer breakdown, and daily click counts. The authenticated user must own the URL.

#### Get a URL

```http
GET /api/v1/urls/{id}
```

#### Update a URL

```http
PATCH /api/v1/urls/{id}
Content-Type: application/json
```

```json
{
  "url": "https://example.com/updated-resource",
  "is_active": true
}
```

Only the owning authenticated user can update a URL. The `expires_at` field is reserved but its input format is not supported yet.

#### Delete a URL

```http
DELETE /api/v1/urls/{id}
```

Returns `204 No Content` when deletion succeeds.

#### Redirect

```http
GET /{short_code}
```

Redirects are public and return `302 Found`. A click event containing the timestamp, user agent, referrer, and detected device category is sent to the asynchronous worker queue.

### Utility endpoints

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/health` | Returns `OK` when the HTTP process is running |
| `GET` | `/hello` | Development endpoint returning `Hello, GoShort!` |

## Reliability and Security

- Passwords are hashed before persistence.
- Refresh tokens are stored as hashes, not raw tokens.
- Protected URL operations enforce ownership.
- Global rate limiting allows up to 60 requests per client IP per minute.
- Login has a separate limit of 10 requests per client IP per minute.
- The click queue is bounded at 1,000 events.
- Click persistence retries up to five times with exponential backoff, capped at five seconds.
- Click events use a unique event key, making retry persistence idempotent at the database level.
- Redis entries are invalidated when URL data changes.
- Shutdown stops the HTTP server, drains queued click work, and closes database and Redis resources.

## Testing

Run the complete test suite:

```powershell
go test ./...
```

Run with the race detector:

```powershell
go test -race ./...
```

Run with coverage:

```powershell
go test -cover ./...
```

Build all packages:

```powershell
go build ./...
```

Tests currently cover authentication middleware and services, password handling, rate limiting, URL services, and click-worker behavior.

## Roadmap

- Analytics API and reporting queries
- Structured logging, metrics, request IDs, and distributed tracing
- Integration and end-to-end tests
- Benchmarks, load testing, and broader concurrency coverage
- Docker Compose for the complete application stack
- CI/CD pipeline
- Horizontal scaling and queue-based analytics architecture

## License

No license has been specified yet.
