package relay_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/frank-mendez/chatmesh/internal/hub"
	"github.com/frank-mendez/chatmesh/internal/relay"
)

// sub is a race-safe test subscriber that signals delivery via a buffered channel.
type sub struct {
	received chan []byte
}

func newSub() *sub {
	return &sub{received: make(chan []byte, 16)}
}

func (s *sub) Deliver(p []byte) { s.received <- p }
func (s *sub) Close()           {} // no-op: test subscriber does not need cleanup

func mustRecv(t *testing.T, s *sub) []byte {
	t.Helper()
	select {
	case p := <-s.received:
		return p
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for relay delivery")
		return nil
	}
}

func mustNotRecv(t *testing.T, s *sub) {
	t.Helper()
	select {
	case p := <-s.received:
		t.Fatalf("unexpected relay delivery: %s", p)
	case <-time.After(50 * time.Millisecond):
	}
}

func startRelay(t *testing.T, h *hub.Hub, addr string) *relay.Relay {
	t.Helper()
	r, err := relay.New(h, "redis://"+addr)
	if err != nil {
		t.Fatalf("relay.New: %v", err)
	}
	t.Cleanup(r.Close)
	go r.Run(t.Context()) //nolint:errcheck
	select {
	case <-r.Ready():
	case <-time.After(time.Second):
		t.Fatal("relay subscription timed out")
	}
	return r
}

func TestRelayDeliversToHub(t *testing.T) {
	mr := miniredis.RunT(t)

	h := hub.New()
	go h.Run()

	alice := newSub()
	h.Register(alice, "general")
	defer h.Unregister(alice, "general")

	r := startRelay(t, h, mr.Addr())

	payload := []byte(`{"type":"message","room":"general","user":"bob","content":"hi"}`)
	r.Broadcast("general", payload)

	got := mustRecv(t, alice)

	var msg struct {
		User    string `json:"user"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(got, &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.User != "bob" || msg.Content != "hi" {
		t.Errorf("got %+v, want user=bob content=hi", msg)
	}
}

func TestRelayRoomIsolation(t *testing.T) {
	mr := miniredis.RunT(t)

	h := hub.New()
	go h.Run()

	generalSub := newSub()
	sportsSub := newSub()
	h.Register(generalSub, "general")
	h.Register(sportsSub, "sports")
	defer h.Unregister(generalSub, "general")
	defer h.Unregister(sportsSub, "sports")

	r := startRelay(t, h, mr.Addr())

	payload := []byte(`{"type":"message","room":"general","user":"alice","content":"hello"}`)
	r.Broadcast("general", payload)

	mustRecv(t, generalSub)
	mustNotRecv(t, sportsSub)
}
