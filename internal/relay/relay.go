package relay

import (
	"context"
	"encoding/json"
	"log"

	"github.com/redis/go-redis/v9"

	"github.com/frank-mendez/chatmesh/internal/hub"
)

const channel = "chatmesh"

// Relay bridges the local Hub to Redis Pub/Sub so messages are delivered
// to clients on all server instances, not just the one that received the write.
type Relay struct {
	hub    *hub.Hub
	client *redis.Client
}

// New connects to Redis at redisURL and returns a ready Relay.
func New(h *hub.Hub, redisURL string) (*Relay, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return &Relay{hub: h, client: redis.NewClient(opt)}, nil
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

// Run subscribes to Redis and delivers incoming messages to the local hub.
// Blocks until ctx is cancelled — call in a goroutine.
func (r *Relay) Run(ctx context.Context) error {
	sub := r.client.Subscribe(ctx, channel)
	defer sub.Close()
	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			var m struct {
				Room string `json:"room"`
			}
			if err := json.Unmarshal([]byte(msg.Payload), &m); err != nil {
				log.Printf("relay: parse error: %v", err)
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
