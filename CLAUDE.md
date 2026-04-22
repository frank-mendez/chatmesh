# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

**chatmesh** is a real-time chat backend in Go, intentionally evolved through scaling phases to expose real-world distributed system problems. Some early-phase limitations are deliberate.

## Commands

```bash
go run ./cmd/server          # run the server
go build ./cmd/server        # build
go test ./...                # run all tests
go test ./internal/hub/... -run TestName   # run a single test
go vet ./...                 # lint
```

WebSocket endpoint: `ws://localhost:8080/ws`

## Architecture

### Message flow

`Client` → WebSocket connection → `internal/websocket` (connection lifecycle) → `internal/hub` (room routing + broadcast) → other `Client`s

The Hub is the central dispatcher. It owns all room state in memory (Phase 1). Each room maps to a set of connected clients.

### Concurrency model

Every WebSocket connection spawns exactly two goroutines: a **read loop** and a **write loop**. This is a hard constraint — never write to a `*websocket.Conn` from multiple goroutines. The write loop is the sole writer; messages are sent to it via a buffered channel. Goroutines must be cleaned up on disconnect.

### Message schema

All messages are JSON:

```json
{
  "type": "message",
  "room": "string",
  "user": "string",
  "content": "string"
}
```

Validate before processing. Keep transport logic (`internal/websocket`) decoupled from business logic (`internal/services`).

## Scaling phases

- **Phase 1 (current):** single instance, in-memory hub state
- **Phase 2:** multiple instances behind a load balancer — message inconsistency is an expected, intentional problem
- **Phase 3:** Redis Pub/Sub per room; all instances publish and subscribe; idempotent message handling required
- **Phase 4:** Docker + Kubernetes + Ingress

## Key constraints

- Standard library preferred; Gorilla WebSocket is the only planned external dependency before Redis
- No blocking operations inside goroutines
- `pkg/` exists for reusable utilities — avoid populating it prematurely
- JWT auth gates the WebSocket upgrade (planned); rate limiting and connection flood prevention are also planned
