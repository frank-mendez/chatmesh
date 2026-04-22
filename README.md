# 🚀 chatmesh

A scalable real-time chat backend built with Go, designed to explore WebSockets, distributed systems, and cloud-native architecture.

> This project evolves from a single-node WebSocket server into a horizontally scalable system using Redis Pub/Sub, Docker, and Kubernetes.

---

## 🧠 Goals

This project is not just a chat app—it’s a learning platform for:

- Real-time communication using WebSockets
- Concurrency patterns in Go (goroutines, channels)
- Distributed system design
- Horizontal scaling and load balancing
- Event-driven architecture
- Production-ready backend practices

---

## ⚙️ Tech Stack

- **Language:** Go
- **Transport:** WebSockets
- **Cache / PubSub:** Redis
- **Database:** PostgreSQL (planned)
- **Containerization:** Docker
- **Orchestration:** Kubernetes (planned)
- **Reverse Proxy:** NGINX (planned)
- **CI/CD:** GitHub Actions (planned)

---

## 🏗️ Architecture (Phase 1)

Single-node WebSocket server:

Client ⇄ WebSocket Server ⇄ Room Manager ⇄ Clients

### Components

- **Connection Manager** — handles WebSocket connections
- **Hub (Room Manager)** — manages rooms and broadcasts
- **Client** — represents a connected user
- **Message Dispatcher** — routes messages to rooms

---

## 🔄 Architecture Evolution

### Phase 1 — Single Node

- In-memory state
- Simple room-based messaging

### Phase 2 — Multi Instance (Breaking Point)

- Multiple servers behind load balancer
- Issues: message inconsistency

### Phase 3 — Distributed System

- Redis Pub/Sub for cross-instance communication
- Horizontal scaling enabled

### Phase 4 — Cloud Native

- Dockerized services
- Kubernetes deployment
- Ingress + load balancing

---

## 📦 Project Structure

```
chatmesh/
├── cmd/
│   └── server/        # entrypoint
├── internal/
│   ├── websocket/     # websocket handling
│   ├── hub/           # room & connection manager
│   ├── models/        # data models
│   └── services/      # business logic
├── pkg/               # reusable packages
├── docker/
├── deployments/       # k8s manifests (future)
├── scripts/
└── docker-compose.yml
```

---

## 🚀 Getting Started

### Prerequisites

- Go 1.22+
- Docker (optional but recommended)

---

### Run locally (no Docker)

```bash
go run ./cmd/server
```

---

### Run with Docker (planned)

```bash
docker-compose up --build
```

---

## 🔌 WebSocket Endpoint

```
ws://localhost:8080/ws
```

### Example Message

```json
{
  "type": "message",
  "room": "general",
  "user": "frank",
  "content": "hello world"
}
```

---

## 🧪 Features

- [x] WebSocket connection handling
- [x] Room-based messaging
- [x] Concurrent message broadcasting
- [ ] User presence tracking
- [ ] Redis Pub/Sub integration
- [ ] Persistent message storage
- [ ] Authentication (JWT)
- [ ] Horizontal scaling
- [ ] Kubernetes deployment

---

## ⚠️ Known Challenges (Intentional)

This project is designed to expose real-world problems:

- WebSocket state across multiple instances
- Load balancing with persistent connections
- Message ordering and delivery guarantees
- Reconnection handling
- Distributed system consistency

---

## 🧠 Learning Focus

If you’re exploring this repo, focus on:

- How concurrency is handled in Go
- How messages flow through the system
- How scaling changes architecture decisions
- Trade-offs between simplicity and scalability

---

## 🤝 Contributing

This is a learning-driven project, but contributions and discussions are welcome.

---

## 📜 License

MIT

---

## ✨ Author

Built to explore real-time systems, backend architecture, and distributed design using Go.
