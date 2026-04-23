package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	gorillaws "github.com/gorilla/websocket"

	"github.com/frank-mendez/chatmesh/internal/hub"
	"github.com/frank-mendez/chatmesh/internal/models"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
	sendBufSize    = 256
)

// Broadcaster routes outbound messages to room subscribers.
// Both *hub.Hub and *relay.Relay satisfy this interface.
type Broadcaster interface {
	Broadcast(room string, payload []byte)
}

// Client represents one connected WebSocket user.
type Client struct {
	hub         *hub.Hub    // register/unregister only
	broadcaster Broadcaster // message routing (local hub or Redis relay)
	conn        *gorillaws.Conn
	send        chan []byte
	room        string
	user        string
	once        sync.Once
}

func newClient(h *hub.Hub, b Broadcaster, conn *gorillaws.Conn, room, user string) *Client {
	return &Client{
		hub:         h,
		broadcaster: b,
		conn:        conn,
		send:        make(chan []byte, sendBufSize),
		room:        room,
		user:        user,
	}
}

// Deliver implements hub.Subscriber. Non-blocking; drops the message if the buffer is full.
func (c *Client) Deliver(payload []byte) {
	select {
	case c.send <- payload:
	default:
		log.Printf("client: send buffer full, dropping message for user=%q", c.user)
	}
}

// Close implements hub.Subscriber. Idempotent — safe to call multiple times.
func (c *Client) Close() {
	c.once.Do(func() { close(c.send) })
}

// ReadLoop reads from the WebSocket and forwards validated messages to the broadcaster.
// It is the only goroutine that reads from c.conn.
func (c *Client) ReadLoop() {
	defer func() {
		c.hub.Unregister(c, c.room)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	c.hub.Register(c, c.room)

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if gorillaws.IsUnexpectedCloseError(err, gorillaws.CloseGoingAway, gorillaws.CloseAbnormalClosure) {
				log.Printf("client: unexpected close user=%q: %v", c.user, err)
			}
			return
		}

		var msg models.Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			log.Printf("client: invalid JSON user=%q: %v", c.user, err)
			continue
		}
		if !validMessage(&msg) {
			log.Printf("client: invalid message user=%q: %+v", c.user, msg)
			continue
		}

		// Stamp server-side identity so clients cannot impersonate others.
		msg.User = c.user
		msg.Room = c.room

		payload, err := json.Marshal(msg)
		if err != nil {
			log.Printf("client: marshal error: %v", err)
			continue
		}

		c.broadcaster.Broadcast(c.room, payload)
	}
}

// WriteLoop pumps outbound messages from the send channel to the WebSocket.
// It is the only goroutine that writes to c.conn.
func (c *Client) WriteLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case payload, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(gorillaws.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(gorillaws.TextMessage, payload); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(gorillaws.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func validMessage(m *models.Message) bool {
	return m.Type == "message" && m.Content != ""
}
