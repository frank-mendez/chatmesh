package relay_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/frank-mendez/chatmesh/internal/hub"
	"github.com/frank-mendez/chatmesh/internal/relay"
)

type sub struct{ got [][]byte }

func (s *sub) Deliver(p []byte) { s.got = append(s.got, p) }
func (s *sub) Close() {} // no-op: test subscriber does not need cleanup

func TestRelayDeliversToHub(t *testing.T) {
	mr := miniredis.RunT(t)

	h := hub.New()
	go h.Run()

	alice := &sub{}
	h.Register(alice, "general")
	defer h.Unregister(alice, "general")

	r, err := relay.New(h, "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("relay.New: %v", err)
	}
	defer r.Close()

	go r.Run(t.Context()) //nolint:errcheck

	time.Sleep(50 * time.Millisecond) // let subscription start

	payload := []byte(`{"type":"message","room":"general","user":"bob","content":"hi"}`)
	r.Broadcast("general", payload)

	time.Sleep(100 * time.Millisecond)

	if len(alice.got) != 1 {
		t.Fatalf("expected 1 message, got %d", len(alice.got))
	}
	var msg struct {
		User    string `json:"user"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(alice.got[0], &msg); err != nil {
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

	generalSub := &sub{}
	sportsSub := &sub{}
	h.Register(generalSub, "general")
	h.Register(sportsSub, "sports")
	defer h.Unregister(generalSub, "general")
	defer h.Unregister(sportsSub, "sports")

	r, err := relay.New(h, "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("relay.New: %v", err)
	}
	defer r.Close()

	go r.Run(t.Context()) //nolint:errcheck

	time.Sleep(50 * time.Millisecond)

	payload := []byte(`{"type":"message","room":"general","user":"alice","content":"hello"}`)
	r.Broadcast("general", payload)

	time.Sleep(100 * time.Millisecond)

	if len(generalSub.got) != 1 {
		t.Errorf("general: expected 1 message, got %d", len(generalSub.got))
	}
	if len(sportsSub.got) != 0 {
		t.Errorf("sports: expected 0 messages, got %d", len(sportsSub.got))
	}
}
