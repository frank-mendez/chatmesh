package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/frank-mendez/chatmesh/internal/hub"
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

	h := hub.New()
	go h.Run()

	http.HandleFunc("/ws", ws.Handler(h))
	http.HandleFunc("/instance", instanceHandler(serverID))

	log.Println("chatmesh listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
