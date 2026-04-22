// Package hub manages room membership and message routing.
// All room state lives in the Run goroutine — no mutexes needed.
package hub

import "log"

// Subscriber can receive broadcast payloads and be signalled when evicted.
type Subscriber interface {
	Deliver(payload []byte)
	Close()
}

type subscribeEvent struct {
	sub  Subscriber
	room string
	done chan struct{} // closed by Run after the event is fully processed
}

type broadcastMsg struct {
	room    string
	payload []byte
}

// Hub owns all room state and routes broadcast messages.
// Run must be called in its own goroutine before any other methods.
type Hub struct {
	rooms      map[string]map[Subscriber]bool
	register   chan subscribeEvent
	unregister chan subscribeEvent
	broadcast  chan broadcastMsg
}

// New returns an initialized Hub ready to be started with Run.
func New() *Hub {
	return &Hub{
		rooms:      make(map[string]map[Subscriber]bool),
		register:   make(chan subscribeEvent),
		unregister: make(chan subscribeEvent),
		broadcast:  make(chan broadcastMsg, 256),
	}
}

// Run processes hub events. Blocks until the process exits — call in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case ev := <-h.register:
			if h.rooms[ev.room] == nil {
				h.rooms[ev.room] = make(map[Subscriber]bool)
			}
			h.rooms[ev.room][ev.sub] = true
			log.Printf("hub: +1 room=%q size=%d", ev.room, len(h.rooms[ev.room]))
			close(ev.done)

		case ev := <-h.unregister:
			subs, ok := h.rooms[ev.room]
			if ok && subs[ev.sub] {
				delete(subs, ev.sub)
				ev.sub.Close()
				if len(subs) == 0 {
					delete(h.rooms, ev.room)
				}
				log.Printf("hub: -1 room=%q", ev.room)
			}
			close(ev.done)

		case msg := <-h.broadcast:
			for sub := range h.rooms[msg.room] {
				sub.Deliver(msg.payload)
			}
		}
	}
}

// Register adds sub to room. Blocks until Run has fully processed the event.
func (h *Hub) Register(sub Subscriber, room string) {
	done := make(chan struct{})
	h.register <- subscribeEvent{sub: sub, room: room, done: done}
	<-done
}

// Unregister removes sub from room and closes it. Blocks until Run has fully processed the event.
func (h *Hub) Unregister(sub Subscriber, room string) {
	done := make(chan struct{})
	h.unregister <- subscribeEvent{sub: sub, room: room, done: done}
	<-done
}

// Broadcast enqueues payload for all subscribers in room.
func (h *Hub) Broadcast(room string, payload []byte) {
	h.broadcast <- broadcastMsg{room: room, payload: payload}
}
