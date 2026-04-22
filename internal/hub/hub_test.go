package hub_test

import (
	"sync"
	"testing"
	"time"

	"github.com/frank-mendez/chatmesh/internal/hub"
)

// mockSub is a test double for hub.Subscriber.
type mockSub struct {
	received chan []byte
	once     sync.Once
	mu       sync.Mutex
	closed   bool
}

func newMock() *mockSub {
	return &mockSub{received: make(chan []byte, 16)}
}

func (m *mockSub) Deliver(payload []byte) { m.received <- payload }

func (m *mockSub) Close() {
	m.once.Do(func() {
		m.mu.Lock()
		m.closed = true
		m.mu.Unlock()
	})
}

func (m *mockSub) isClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func startHub(t *testing.T) *hub.Hub {
	t.Helper()
	h := hub.New()
	go h.Run()
	return h
}

func mustRecv(t *testing.T, sub *mockSub) []byte {
	t.Helper()
	select {
	case p := <-sub.received:
		return p
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for delivery")
		return nil
	}
}

func mustNotRecv(t *testing.T, sub *mockSub) {
	t.Helper()
	select {
	case p := <-sub.received:
		t.Fatalf("unexpected delivery: %s", p)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestHub_Broadcast(t *testing.T) {
	t.Run("single subscriber receives message", func(t *testing.T) {
		h := startHub(t)
		s := newMock()
		h.Register(s, "general")
		h.Broadcast("general", []byte("hello"))
		got := mustRecv(t, s)
		if string(got) != "hello" {
			t.Errorf("got %q, want %q", got, "hello")
		}
	})

	t.Run("multiple subscribers all receive", func(t *testing.T) {
		h := startHub(t)
		a, b, c := newMock(), newMock(), newMock()
		h.Register(a, "room1")
		h.Register(b, "room1")
		h.Register(c, "room1")
		h.Broadcast("room1", []byte("everyone"))
		mustRecv(t, a)
		mustRecv(t, b)
		mustRecv(t, c)
	})

	t.Run("broadcast to wrong room delivers nothing", func(t *testing.T) {
		h := startHub(t)
		s := newMock()
		h.Register(s, "room-a")
		h.Broadcast("room-b", []byte("misrouted"))
		mustNotRecv(t, s)
	})

	t.Run("broadcast to empty room does not panic", func(t *testing.T) {
		h := startHub(t)
		h.Broadcast("ghost-room", []byte("noop"))
		// No assertion needed — just must not panic.
		time.Sleep(20 * time.Millisecond)
	})
}

func TestHub_RoomIsolation(t *testing.T) {
	h := startHub(t)
	a := newMock()
	b := newMock()
	h.Register(a, "room-a")
	h.Register(b, "room-b")

	h.Broadcast("room-a", []byte("for-a"))

	got := mustRecv(t, a)
	if string(got) != "for-a" {
		t.Errorf("got %q, want %q", got, "for-a")
	}
	mustNotRecv(t, b)
}

func TestHub_Unregister(t *testing.T) {
	t.Run("closes subscriber", func(t *testing.T) {
		h := startHub(t)
		s := newMock()
		h.Register(s, "room")
		h.Unregister(s, "room")
		if !s.isClosed() {
			t.Error("want subscriber closed after Unregister")
		}
	})

	t.Run("removes from room so no further delivery", func(t *testing.T) {
		h := startHub(t)
		s := newMock()
		h.Register(s, "room")
		h.Unregister(s, "room")
		h.Broadcast("room", []byte("after-leave"))
		mustNotRecv(t, s)
	})

	t.Run("double unregister is safe", func(t *testing.T) {
		h := startHub(t)
		s := newMock()
		h.Register(s, "room")
		h.Unregister(s, "room")
		h.Unregister(s, "room") // must not panic or block
	})
}
