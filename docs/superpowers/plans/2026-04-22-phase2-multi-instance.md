# Phase 2: Multi-Instance WebSocket + Message Inconsistency Demo

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Run two isolated chat server instances behind an nginx round-robin load balancer and prove with a Go demo that messages do not cross instance boundaries.

**Architecture:** Two independent `cmd/server` processes each own their in-memory hub. nginx proxies WebSocket and HTTP traffic between them with no sticky sessions. A demo client connects Alice + Charlie to server-1 and Bob to server-2 directly, sends a message from Alice, and proves Bob never receives it.

**Tech Stack:** Go 1.22, gorilla/websocket, Docker, Docker Compose, nginx:alpine

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `cmd/server/main.go` | Modify | Add `SERVER_ID` env, `log.SetPrefix`, `/instance` handler |
| `cmd/server/main_test.go` | Create | Test `/instance` handler response |
| `Dockerfile` | Create | Multi-stage static binary build |
| `docker/nginx.conf` | Create | WS-aware round-robin reverse proxy |
| `docker-compose.yml` | Create | Two server instances + nginx service |
| `cmd/demo/main.go` | Create | 3-client inconsistency demo |

Files NOT modified: `internal/hub/`, `internal/websocket/`, `internal/models/`, `cmd/testclient/`

---

## Task 1: Add SERVER_ID logging and /instance endpoint

**Files:**
- Modify: `cmd/server/main.go`
- Create: `cmd/server/main_test.go`

- [ ] **Step 1: Write the failing test**

Create `cmd/server/main_test.go`:

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInstanceHandler(t *testing.T) {
	tests := []struct {
		serverID string
		wantBody string
	}{
		{"server-1", `{"server_id":"server-1"}`},
		{"server-2", `{"server_id":"server-2"}`},
	}
	for _, tt := range tests {
		t.Run(tt.serverID, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/instance", nil)
			rr := httptest.NewRecorder()
			instanceHandler(tt.serverID)(rr, req)
			if got := rr.Body.String(); got != tt.wantBody {
				t.Errorf("body got %q, want %q", got, tt.wantBody)
			}
			if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type got %q, want application/json", ct)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./cmd/server/...
```
Expected: `FAIL — undefined: instanceHandler`

- [ ] **Step 3: Replace cmd/server/main.go with updated version**

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/frank-mendez/chatmesh/internal/hub"
	ws "github.com/frank-mendez/chatmesh/internal/websocket"
)

func instanceHandler(serverID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"server_id":%q}`, serverID)
	}
}

func main() {
	serverID := os.Getenv("SERVER_ID")
	if serverID == "" {
		serverID = "server-1"
	}
	log.SetPrefix("[" + serverID + "] ")

	h := hub.New()
	go h.Run()

	http.HandleFunc("/ws", ws.Handler(h))
	http.HandleFunc("/instance", instanceHandler(serverID))

	log.Println("chatmesh listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./cmd/server/...
```
Expected:
```
--- PASS: TestInstanceHandler/server-1 (0.00s)
--- PASS: TestInstanceHandler/server-2 (0.00s)
PASS
ok  	github.com/frank-mendez/chatmesh/cmd/server
```

- [ ] **Step 5: Run full test suite to confirm nothing regressed**

```bash
go test ./...
```
Expected: all packages PASS.

- [ ] **Step 6: Commit**

```bash
git add cmd/server/main.go cmd/server/main_test.go
git commit -m "feat: add SERVER_ID logging and /instance endpoint"
```

---

## Task 2: Create Dockerfile

**Files:**
- Create: `Dockerfile`

- [ ] **Step 1: Create Dockerfile at repo root**

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

- [ ] **Step 2: Build the image locally**

```bash
docker build -t chatmesh:dev .
```
Expected: `Successfully built ...` with no errors.

- [ ] **Step 3: Smoke-test the binary starts**

```bash
docker run --rm -e SERVER_ID=smoke-test -p 8888:8080 chatmesh:dev &
sleep 1
curl -s http://localhost:8888/instance
kill %1
```
Expected output: `{"server_id":"smoke-test"}`

- [ ] **Step 4: Commit**

```bash
git add Dockerfile
git commit -m "feat: add multi-stage Dockerfile with static binary"
```

---

## Task 3: Create nginx config

**Files:**
- Create: `docker/nginx.conf`

- [ ] **Step 1: Create docker/ directory and nginx.conf**

```bash
mkdir -p docker
```

Write `docker/nginx.conf`:

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

- [ ] **Step 2: Validate nginx config syntax**

```bash
docker run --rm -v $(pwd)/docker/nginx.conf:/etc/nginx/nginx.conf:ro nginx:alpine nginx -t
```
Expected: `nginx: configuration file /etc/nginx/nginx.conf test is successful`

- [ ] **Step 3: Commit**

```bash
git add docker/nginx.conf
git commit -m "feat: add nginx WebSocket reverse proxy config"
```

---

## Task 4: Create docker-compose.yml

**Files:**
- Create: `docker-compose.yml`

- [ ] **Step 1: Create docker-compose.yml at repo root**

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

- [ ] **Step 2: Build and start the stack**

```bash
docker compose up --build -d
```
Expected: three containers start (`server1`, `server2`, `nginx`).

- [ ] **Step 3: Verify each instance responds with correct ID**

```bash
curl -s http://localhost:8081/instance
curl -s http://localhost:8082/instance
```
Expected:
```
{"server_id":"server-1"}
{"server_id":"server-2"}
```

- [ ] **Step 4: Verify nginx is proxying WebSocket connections**

```bash
curl -s http://localhost:8080/instance
```
Expected: either `{"server_id":"server-1"}` or `{"server_id":"server-2"}` (round-robin).

- [ ] **Step 5: Commit**

```bash
git add docker-compose.yml
git commit -m "feat: add docker-compose with two server instances and nginx"
```

---

## Task 5: Create the inconsistency demo

**Files:**
- Create: `cmd/demo/main.go`

- [ ] **Step 1: Create cmd/demo/main.go**

```go
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	gorillaws "github.com/gorilla/websocket"

	"github.com/frank-mendez/chatmesh/internal/models"
)

const (
	server1WS       = "ws://localhost:8081/ws"
	server2WS       = "ws://localhost:8082/ws"
	server1Instance = "http://localhost:8081/instance"
	server2Instance = "http://localhost:8082/instance"
	room            = "general"
	readTimeout     = 2 * time.Second
	connectDelay    = 200 * time.Millisecond
)

func getServerID(instanceURL string) string {
	resp, err := http.Get(instanceURL) //nolint:noctx
	if err != nil {
		log.Fatalf("GET %s: %v", instanceURL, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		ServerID string `json:"server_id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Fatalf("parse /instance response: %v", err)
	}
	return result.ServerID
}

func dialWS(wsURL, user string) *gorillaws.Conn {
	url := fmt.Sprintf("%s?room=%s&user=%s", wsURL, room, user)
	conn, _, err := gorillaws.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatalf("dial %s: %v", url, err)
	}
	return conn
}

func tryRead(conn *gorillaws.Conn, label string) {
	conn.SetReadDeadline(time.Now().Add(readTimeout))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		fmt.Printf("%-7s received:  ❌ NONE (timeout after 2s — expected)\n", label)
		return
	}
	var msg models.Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		fmt.Printf("%-7s received:  ❌ parse error: %v\n", label, err)
		return
	}
	fmt.Printf("%-7s received:  user=%q content=%q room=%q ✅\n", label, msg.User, msg.Content, msg.Room)
}

func main() {
	fmt.Println("=== Phase 2: Message Inconsistency Demo ===")
	fmt.Println()

	id1 := getServerID(server1Instance)
	id2 := getServerID(server2Instance)

	fmt.Println("[setup]")
	fmt.Printf("Alice   → %s  (ws://localhost:8081)\n", id1)
	fmt.Printf("Bob     → %s  (ws://localhost:8082)\n", id2)
	fmt.Printf("Charlie → %s  (ws://localhost:8081)\n", id1)
	fmt.Println()

	fmt.Println("[expectation]")
	fmt.Println("Charlie should receive Alice's message ✅  (same instance)")
	fmt.Println("Bob should NOT receive Alice's message ❌  (different instance)")
	fmt.Println()

	alice := dialWS(server1WS, "alice")
	defer alice.Close()
	charlie := dialWS(server1WS, "charlie")
	defer charlie.Close()
	bob := dialWS(server2WS, "bob")
	defer bob.Close()

	time.Sleep(connectDelay)

	msg := models.Message{Type: "message", Room: room, User: "alice", Content: "hello"}
	payload, _ := json.Marshal(msg)

	fmt.Println("[sending]")
	fmt.Printf("Alice sends: %s\n", payload)
	fmt.Println()

	if err := alice.WriteMessage(gorillaws.TextMessage, payload); err != nil {
		log.Fatalf("alice write: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	fmt.Println("[result]")
	tryRead(charlie, "Charlie")
	tryRead(bob, "Bob")
	fmt.Println()

	fmt.Println("[conclusion]")
	fmt.Println("Messages are NOT shared across instances.")
	fmt.Println("In-memory state is isolated per server.")
	fmt.Println("This system is not horizontally scalable as-is.")
	fmt.Println("Phase 3 will fix this with Redis Pub/Sub.")
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./cmd/demo/...
```
Expected: no output (success).

- [ ] **Step 3: Ensure Docker stack is running, then run the demo**

```bash
docker compose up -d   # skip if already running
go run ./cmd/demo
```

Expected output:
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
Charlie received:  user="alice" content="hello" room="general" ✅
Bob     received:  ❌ NONE (timeout after 2s — expected)

[conclusion]
Messages are NOT shared across instances.
In-memory state is isolated per server.
This system is not horizontally scalable as-is.
Phase 3 will fix this with Redis Pub/Sub.
```

- [ ] **Step 4: Check server logs to confirm split routing**

```bash
docker compose logs server1 | grep -E "connected|received"
docker compose logs server2 | grep -E "connected|received"
```
Expected: server1 logs show alice + charlie connections; server2 logs show bob's connection.

- [ ] **Step 5: Commit**

```bash
git add cmd/demo/main.go
git commit -m "feat: add Phase 2 inconsistency demo client"
```

---

## Task 6: Push and final verification

- [ ] **Step 1: Run full test suite one last time**

```bash
go test ./...
```
Expected: all PASS.

- [ ] **Step 2: Push**

```bash
git push origin main
```

- [ ] **Step 3: Tear down Docker stack**

```bash
docker compose down
```
