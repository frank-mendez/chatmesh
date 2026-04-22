package websocket

import (
	"log"
	"net/http"

	gorillaws "github.com/gorilla/websocket"

	"github.com/frank-mendez/chatmesh/internal/hub"
)

var upgrader = gorillaws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Phase 1: allow all origins. Tighten in a later phase.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Handler returns an http.HandlerFunc that upgrades connections and starts
// the per-client read/write goroutines.
//
// Query params:
//
//	room — required, the room to join
//	user — required, the display name
func Handler(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		room := r.URL.Query().Get("room")
		user := r.URL.Query().Get("user")
		if room == "" || user == "" {
			http.Error(w, "room and user query params are required", http.StatusBadRequest)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("handler: upgrade error: %v", err)
			return
		}

		c := newClient(h, conn, room, user)
		go c.WriteLoop()
		go c.ReadLoop()
	}
}
