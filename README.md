# Real-Time Chat API

A production-oriented real-time chat API built in Go. Features WebSocket-based messaging with room routing, JWT authentication, PostgreSQL persistence, and Docker deployment.

## Features

- Real-time WebSocket messaging with structured JSON protocol
- Room-based message routing — messages only reach room members
- JWT authentication protecting both REST and WebSocket endpoints
- PostgreSQL persistence with automatic schema migrations on startup
- Graceful shutdown — clean WebSocket disconnection, HTTP drain, DB cleanup
- WebSocket keepalive with periodic pings and dead-client timeout
- Slow-client detection — full send channels trigger automatic disconnect
- WebSocket origin allowlist via environment variable
- Docker multi-stage build with non-root runtime user
- Request body limits on authentication endpoints
- Context propagation through the full request path

## Tech Stack

| Component | Technology |
|---|---|
| Language | Go 1.26 |
| HTTP | `net/http` (standard library) |
| WebSocket | `gorilla/websocket` v1.5.3 |
| Database | PostgreSQL 16 |
| Driver | `pgx/v5` via `database/sql` standard interface |
| Auth | `golang-jwt/jwt/v5` (HS256), `golang.org/x/crypto` (bcrypt) |
| IDs | `google/uuid` v1.6.0 |
| Container | Docker, Docker Compose |

## Architecture

```
HTTP Request
  -> Handler (parse, validate, respond)
    -> AuthService (business logic, JWT)
      -> UserRepository (SQL queries)
        -> PostgreSQL

WebSocket Client
  -> Handler (JWT verify, upgrade)
    -> Hub (client registry, broadcast, room routing)
      -> WritePump (goroutine, outbound messages)
      -> ReadPump (synchronous, inbound messages)
```

**Key design principle:** The Hub goroutine is the sole owner of the client registry. All other goroutines communicate through channels — no mutexes required for client state.

### Responsibilities

| Layer | Responsibility |
|---|---|
| **Handler** | Parse HTTP/WebSocket requests, validate transport-level input, return HTTP responses |
| **AuthService** | Business rules, JWT generation/verification, password hashing, input validation |
| **UserRepository** | Database operations, parameterized SQL queries, persistence |
| **Hub** | Track active WebSocket clients, register/unregister, broadcast messages, room routing |
| **Client** | Own one WebSocket connection, read/write pumps, send channel for outbound messages |

## Project Structure

```
cmd/server/                  Entry point, routes, graceful shutdown
  main.go
internal/
  handler/
    auth.go                  POST /register, POST /login
    websocket.go             /ws endpoint, JWT extraction, origin check
  service/
    auth.go                  Register, Login, JWT generation/verification
  repository/
    user.go                  UserRepository interface
    postgres_user.go         PostgreSQL implementation
    mock_user.go             In-memory mock for unit tests
  websocket/
    hub.go                   Client registry, broadcast, room routing
    client.go                WritePump, ReadPump, ping/pong
    message.go               Message struct (type, content, room_id)
  database/
    postgres.go              Connect(), connection pool config
    migrate.go               RunMigrations() — reads migrations/*.sql
  model/
    user.go                  User struct (ID, Username, PasswordHash)
migrations/
  001_create_users.sql       Users table + username index
Dockerfile                   Multi-stage build (golang -> alpine)
docker-compose.yml           API + PostgreSQL services
.env.example                 Required environment variables
go.mod                       Go module definition
```

## Getting Started

### Prerequisites

- Go 1.26+
- Docker and Docker Compose
- PostgreSQL 16+ (for local development without Docker)

### Option A: Docker (Recommended)

```bash
cp .env.example .env
# Edit .env — set JWT_SECRET (min 32 bytes) and POSTGRES_PASSWORD

docker compose up --build
```

The API starts on `http://localhost:8081`. PostgreSQL data persists in a named volume.

### Option B: Local Development

```bash
# Ensure PostgreSQL is running with a 'chatdb' database
export DATABASE_URL="postgresql://postgres:password@localhost:5432/chatdb?sslmode=disable"
export JWT_SECRET="your-secret-must-be-at-least-32-bytes-long"

go run ./cmd/server
```

The server starts on `:8081` (configurable via `APP_PORT`). Migrations run automatically on startup.

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `JWT_SECRET` | Yes | — | Secret key for HS256 JWT signing. Minimum 32 bytes. |
| `DATABASE_URL` | Yes | — | PostgreSQL connection string. |
| `POSTGRES_PASSWORD` | Yes (Docker) | — | PostgreSQL password. Used in docker-compose.yml. |
| `APP_PORT` | No | `8081` | Server listen port. |
| `WS_ALLOWED_ORIGINS` | No | empty (reject all) | Comma-separated origins for WebSocket upgrade. Empty rejects all cross-origin connections. |

## Docker

### Services

| Service | Image | Purpose |
|---|---|---|
| `api` | Built from Dockerfile | Chat API server |
| `postgres` | `postgres:16-alpine` | PostgreSQL database |

### docker-compose.yml

- API waits for PostgreSQL healthcheck (`pg_isready`) before starting
- `JWT_SECRET` and `POSTGRES_PASSWORD` use `${VAR:?...}` syntax — Compose fails immediately if unset
- PostgreSQL data persists in named volume `pgdata`
- No database port exposed to host by default

### docker-compose.override.yml (git-ignored)

Adds PostgreSQL port `5432:5432` to the host for local debugging. This file is automatically merged by Docker Compose but excluded from version control.

### Dockerfile

Multi-stage build:

1. **Builder** (`golang:1.26-alpine`) — downloads dependencies, builds static binary with `CGO_ENABLED=0 GOOS=linux`
2. **Runtime** (`alpine:3.20`) — minimal image with `ca-certificates`, non-root `appuser`, copies binary + migrations

### Build and Run

```bash
docker compose up --build
```

### Verify

```bash
docker compose ps        # both services healthy
docker compose logs api   # "connected to PostgreSQL", "server starting on :8081"
```

## REST API

### POST /register

Register a new user. Request body limited to 1MB.

**Request:**

```http
POST /register
Content-Type: application/json

{
  "username": "alice",
  "password": "secret123"
}
```

**Responses:**

| Status | Condition | Body |
|---|---|---|
| `201 Created` | Success | `{"id":"550e8400-e29b-41d4-a716-446655440000","username":"alice"}` |
| `409 Conflict` | Username taken | `{"error":"username already taken"}` |
| `400 Bad Request` | Empty username, short password, or invalid JSON | `{"error":"..."}` |
| `405 Method Not Allowed` | Non-POST method | `{"error":"method not allowed"}` |

**Validation rules:**
- Username: required, 1-255 characters
- Password: minimum 6 characters

### POST /login

Authenticate and receive a JWT token.

**Request:**

```http
POST /login
Content-Type: application/json

{
  "username": "alice",
  "password": "secret123"
}
```

**Responses:**

| Status | Condition | Body |
|---|---|---|
| `200 OK` | Success | `{"message":"login successful","user_id":"...","username":"alice","token":"eyJhbG..."}` |
| `401 Unauthorized` | Wrong password or unknown user | `{"error":"invalid credentials"}` |
| `400 Bad Request` | Invalid JSON | `{"error":"invalid json"}` |
| `405 Method Not Allowed` | Non-POST method | `{"error":"method not allowed"}` |

## WebSocket

### Authentication

The `/ws` endpoint requires a valid JWT. Pass it in the `Authorization` header during the HTTP -> WebSocket upgrade:

```
Authorization: Bearer <jwt_token>
```

Connections without a valid token receive `401 Unauthorized`.

### Connecting

**Browser limitation:** The browser WebSocket API does not support setting custom headers. The server currently requires `Authorization: Bearer` in the HTTP header, so browser-based connections need a proxy or backend relay to inject the token.

**Command line (websocat):**

```bash
# 1. Get a token
TOKEN=$(curl -s -X POST http://localhost:8081/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"secret123"}' | jq -r '.token')

# 2. Connect with Bearer token
websocat --header="Authorization: Bearer $TOKEN" \
  "ws://localhost:8081/ws"
```

**Go client:**

```go
header := http.Header{}
header.Set("Authorization", "Bearer "+token)

dialer := websocket.Dialer{}
conn, _, err := dialer.Dial("ws://localhost:8081/ws", header)
```

### Message Protocol

All messages are JSON with this structure:

```json
{
  "type": "message",
  "content": "Hello, world!",
  "room_id": "general"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `type` | string | Yes | `"message"` for chat content, `"join"` to enter a room |
| `content` | string | Yes | Message text payload |
| `room_id` | string | No | Target room. Omit for global broadcast. |

### Joining a Room

Send a join message to route into a room. After joining, your messages only reach other members of that room.

```json
{ "type": "join", "room_id": "general" }
```

### Sending a Message

```json
{ "type": "message", "content": "Hello!", "room_id": "general" }
```

To broadcast globally (all connected clients), omit `room_id`:

```json
{ "type": "message", "content": "Hello everyone!" }
```

### Connection Lifecycle

1. Client connects with Bearer token — JWT verified — connection upgraded
2. Client registered with Hub, identity attached (user ID + username)
3. `ReadPump` reads inbound messages; `WritePump` writes outbound messages
4. `WritePump` sends pings every 54 seconds to keep the connection alive
5. If no pong received within 60 seconds, the client is considered dead and disconnected
6. On disconnect (client or server-side), the client is removed from Hub

## Database

### Schema

The `users` table is created automatically by the migration runner:

```sql
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY,
    username      VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users (username);
```

### Migrations

On startup, the server reads all `.sql` files from the `migrations/` directory, sorts them alphabetically (`001_`, `002_`, etc.), and executes each one.

Uses `CREATE IF NOT EXISTS`, so migrations are idempotent — safe to run on every startup without a separate migration tracker.

To add a new migration: create a file like `002_add_messages.sql` in `migrations/`. It will be applied automatically on the next server start.

### Connection Pool

| Setting | Value |
|---|---|
| MaxOpenConns | 25 |
| MaxIdleConns | 5 |
| ConnMaxLifetime | 5 minutes |

## Testing

### Run tests

```bash
go test ./...
```

Runs all unit tests (auth service, auth handlers, WebSocket integration). PostgreSQL integration tests in `internal/repository` require a running database and are skipped when `DATABASE_URL` is not set.

### Run with database integration

```bash
export DATABASE_URL="postgresql://postgres:password@localhost:5432/chatdb?sslmode=disable"
go test -count=1 ./...
```

With `DATABASE_URL` set, the repository tests run against a real PostgreSQL instance. Tests use the `test_` prefix for usernames and clean up after themselves.

### Static analysis

```bash
go vet ./...
```

### Test coverage

| Package | Tests | What's Covered |
|---|---|---|
| `internal/service` | 12 | Registration validation, login success/failure, JWT verification, token expiry, secret length |
| `internal/handler` | 10 | HTTP status codes, response formats, method validation, error handling |
| `internal/websocket` | 7 | Authenticated connection, missing/invalid token rejection, broadcast, room routing, clean disconnect |
| `internal/repository` | 4 | PostgreSQL CRUD operations (requires `DATABASE_URL`) |

## Security

| Feature | Implementation |
|---|---|
| JWT validation | HS256 with explicit signing method verification (prevents algorithm-switching) |
| JWT secret length | Minimum 32 bytes enforced at startup |
| Token expiry | 24-hour expiration with issued-at and expires-at claims |
| Password hashing | bcrypt with default cost |
| Origin allowlist | `WS_ALLOWED_ORIGINS` env var; empty = reject all cross-origin |
| Request body limits | 1MB on `/register` and `/login` via `MaxBytesReader` |
| Parameterized SQL | All queries use `$1, $2, $3` placeholders (SQL injection prevention) |
| WebSocket keepalive | Pings every 54s, pong timeout at 60s — dead clients detected and cleaned up |
| Slow-client protection | Non-blocking send; full channel triggers automatic disconnect |
| Server timeouts | Read: 10s, ReadHeader: 5s, Write: 10s, Idle: 120s |
| Non-root container | Docker runtime runs as `appuser` |
| Graceful shutdown | HTTP server -> Hub (close all WebSocket connections) -> Database pool |
| Context propagation | `context.Context` flows from HTTP handler through service to repository SQL queries |
| Sanitized logs | User-controlled input not logged; error messages sanitized |

## Future Improvements

- Message persistence — save and retrieve chat history from PostgreSQL
- Direct messaging — 1:1 private conversations
- Typing indicators — real-time "user is typing" signals
- Read receipts — message delivered/read status
- User presence tracking — online/offline/away status
- Message pagination — cursor-based history API
- File and image sharing
- User avatars and profile management
- Rate limiting on authentication endpoints
- HTTPS/TLS termination configuration
