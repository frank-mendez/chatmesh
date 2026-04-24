package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/redis/go-redis/v9"

	"github.com/frank-mendez/chatmesh/internal/hub"
)

const channel = "chatmesh"

// Relay bridges the local Hub to Redis Pub/Sub so messages are delivered
// to clients on all server instances, not just the one that received the write.
type Relay struct {
	hub       *hub.Hub
	client    *redis.Client
	ready     chan struct{}
	readyOnce sync.Once
}

// New connects to Redis at redisURL and returns a ready Relay.
func New(h *hub.Hub, redisURL string) (*Relay, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return &Relay{
		hub:    h,
		client: redis.NewClient(opt),
		ready:  make(chan struct{}),
	}, nil
}

// Broadcast publishes payload to Redis. Run() picks it up on every instance
// and delivers it to the local hub.
// The room parameter is unused here; room routing is performed by the hub,
// which reads the "room" field from the JSON payload directly.
func (r *Relay) Broadcast(room string, payload []byte) {
	if err := r.client.Publish(context.Background(), channel, payload).Err(); err != nil {
		log.Printf("relay: publish room=%q: %v", room, err)
	}
}

// Ready returns a channel that is closed once the Redis subscription is confirmed.
// Callers that need to publish immediately after starting Run should wait on this.
func (r *Relay) Ready() <-chan struct{} {
	return r.ready
}

// Run subscribes to Redis and delivers incoming messages to the local hub.
// Blocks until ctx is cancelled — call in a goroutine.
func (r *Relay) Run(ctx context.Context) error {
	sub := r.client.Subscribe(ctx, channel)
	defer sub.Close()

	// Block until Redis confirms the subscription is established so that no
	// messages published immediately after Run starts are silently dropped.
	if _, err := sub.Receive(ctx); err != nil {
		return err
	}
	r.readyOnce.Do(func() { close(r.ready) })

	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				if err := ctx.Err(); err != nil {
					return err
				}
				return fmt.Errorf("relay: subscription channel closed unexpectedly")
			}
			var m struct {
				Room string `json:"room"`
			}
			if err := json.Unmarshal([]byte(msg.Payload), &m); err != nil {
				log.Printf("relay: parse error: %v", err)
				continue
			}
			if m.Room == "" {
				log.Printf("relay: dropping payload with empty room")
				continue
			}
			r.hub.Broadcast(m.Room, []byte(msg.Payload))
		}
	}
}

// Close disconnects the Redis client.
func (r *Relay) Close() {
	r.client.Close()
}
