package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/frank-mendez/chatmesh/internal/hub"
	"github.com/frank-mendez/chatmesh/internal/relay"
	ws "github.com/frank-mendez/chatmesh/internal/websocket"
)

func instanceHandler(serverID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.Marshal(map[string]string{"server_id": serverID})
		w.Write(b)
	}
}

func main() {
	serverID := os.Getenv("SERVER_ID")
	if serverID == "" {
		serverID = "server-1"
	}
	log.SetPrefix("[" + serverID + "] ")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	h := hub.New()
	go h.Run()

	var broadcaster ws.Broadcaster = h

	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		r, err := relay.New(h, redisURL)
		if err != nil {
			log.Fatalf("relay: %v", err)
		}
		defer r.Close()
		go func() {
			if err := r.Run(ctx); err != nil &&
				!errors.Is(err, context.Canceled) &&
				!errors.Is(err, context.DeadlineExceeded) {
				log.Printf("relay: %v", err)
			}
		}()
		broadcaster = r
		log.Printf("relay: starting with %s", redisURL)
	}

	http.HandleFunc("/ws", ws.Handler(h, broadcaster))
	http.HandleFunc("/instance", instanceHandler(serverID))

	log.Println("chatmesh listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
