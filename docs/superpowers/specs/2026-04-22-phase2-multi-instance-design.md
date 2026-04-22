# Phase 2: Multi-Instance WebSocket + Message Inconsistency Demo

**Date:** 2026-04-22
**Status:** Approved
**Scope:** Docker + nginx multi-instance setup; intentional message inconsistency demo. No Redis, no state sharing.

---

## Context

Phase 1 delivered a single-node WebSocket chat server with an in-memory hub. Phase 2 scales it horizontally to expose a fundamental distributed systems limitation: in-memory state does not cross process boundaries. The inconsistency is intentional and educational — Phase 3 will fix it with Redis Pub/Sub.

---

## Architecture

```
                    ┌─────────────────────────────────┐
                    │         nginx :8080              │
                    │   round-robin upstream           │
                    └────────┬──────────────┬──────────┘
                             │              │
                    ┌────────▼───┐    ┌─────▼──────┐
                    │  server-1  │    │  server-2  │
                    │  :8081     │    │  :8082     │
                    │  hub (A)   │    │  hub (B)   │
                    └────────────┘    └────────────┘
                         ▲                  ▲
                    Alice (WS)         Bob (WS)
                    Charlie (WS)
                    [same room, different hubs — messages do not cross]
```

Two completely isolated Go server processes, each with its own in-memory hub. nginx round-robins WebSocket connections between them. The demo connects clients directly to exposed ports to guarantee the split.

---

## Components

### 1. Go Server — `cmd/server/main.go`

Two additions only. All other files (`internal/hub`, `internal/websocket`, `internal/models`) are untouched.

**Instance-prefixed logging**
```go
serverID := os.Getenv("SERVER_ID")
if serverID == "" {
    serverID = "server-1"
}
log.SetPrefix("[" + serverID + "] ")
```
`log.SetPrefix` applies globally — every existing log line in `hub.go` and `client.go` gets the prefix automatically.

**`/instance` endpoint**
```go
http.HandleFunc("/instance", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintf(w, `{"server_id":%q}`, serverID)
})
```
Returns `{"server_id":"server-1"}`. Used by the demo client to log which instance each connection landed on.

---

### 2. Dockerfile

Static binary into `scratch` — no alpine runtime issues.

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates wget
COPY --from=builder /app/app /app
EXPOSE 8080
ENTRYPOINT ["/app"]
```

---

### 3. docker-compose.yml

```yaml
services:
  server1:
    build: .
    environment:
      - SERVER_ID=server-1
    ports:
      - "8081:8080"
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/instance"]
      interval: 10s
      timeout: 3s
      retries: 3

  server2:
    build: .
    environment:
      - SERVER_ID=server-2
    ports:
      - "8082:8080"
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/instance"]
      interval: 10s
      timeout: 3s
      retries: 3

  nginx:
    image: nginx:alpine
    ports:
      - "8080:80"
    volumes:
      - ./docker/nginx.conf:/etc/nginx/nginx.conf:ro
    depends_on:
      - server1
      - server2
    restart: unless-stopped
```

---

### 4. docker/nginx.conf

```nginx
events {}

http {
    upstream chatmesh {
        # default: round-robin
        server server1:8080;
        server server2:8080;
    }

    server {
        listen 80;

        location /ws {
            proxy_pass         http://chatmesh;
            proxy_http_version 1.1;
            proxy_set_header   Upgrade         $http_upgrade;
            proxy_set_header   Connection      "upgrade";
            proxy_set_header   Host            $host;
            proxy_set_header   X-Forwarded-For $remote_addr;
            proxy_buffering    off;
        }

        location /instance {
            proxy_pass http://chatmesh;
        }
    }
}
```

---

### 5. Demo Client — `cmd/demo/main.go`

Three participants, direct-port connections to guarantee instance split:

| Client  | Port  | Instance  | Role |
|---------|-------|-----------|------|
| Alice   | 8081  | server-1  | sender |
| Charlie | 8081  | server-1  | should receive (same instance) |
| Bob     | 8082  | server-2  | should NOT receive (different instance) |

**Flow:**
1. Call `GET http://localhost:808{1,2}/instance` — log instance mapping
2. Open WebSocket connections for all three clients
3. `time.Sleep(200ms)` — let connections fully register
4. Alice sends `{"type":"message","room":"general","user":"alice","content":"hello"}`
5. Charlie reads with 2s deadline → logs received ✅
6. Bob reads with 2s deadline → times out → logs `❌ NONE (timeout after 2s — expected)`
7. Print conclusion block

**Expected output:**
```
=== Phase 2: Message Inconsistency Demo ===

[setup]
Alice   → server-1  (ws://localhost:8081)
Bob     → server-2  (ws://localhost:8082)
Charlie → server-1  (ws://localhost:8081)

[expectation]
Charlie should receive Alice's message ✅  (same instance)
Bob should NOT receive Alice's message ❌  (different instance)

[sending]
Alice sends: {"type":"message","room":"general","user":"alice","content":"hello"}

[result]
Charlie received: user="alice" content="hello" room="general" ✅
Bob received:     ❌ NONE (timeout after 2s — expected)

[conclusion]
Messages are NOT shared across instances.
In-memory state is isolated per server.
This system is not horizontally scalable as-is.
Phase 3 will fix this with Redis Pub/Sub.
```

---

## File Manifest

| File | Action |
|------|--------|
| `cmd/server/main.go` | Modify — add `SERVER_ID` env, `log.SetPrefix`, `/instance` handler |
| `Dockerfile` | Create |
| `docker-compose.yml` | Create |
| `docker/nginx.conf` | Create |
| `cmd/demo/main.go` | Create |

Files NOT modified: `internal/hub/`, `internal/websocket/`, `internal/models/`, `cmd/testclient/`

---

## Verification

```bash
# 1. Build and start
docker compose up --build

# 2. Confirm instances are up
curl http://localhost:8081/instance   # {"server_id":"server-1"}
curl http://localhost:8082/instance   # {"server_id":"server-2"}

# 3. Run the demo
go run ./cmd/demo

# 4. Check instance logs
docker compose logs server1
docker compose logs server2
```

---

## Out of Scope

- Redis / Pub/Sub (Phase 3)
- Sticky sessions
- Authentication
- Kubernetes
- Message synchronization of any kind
