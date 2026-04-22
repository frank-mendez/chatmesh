package main

import (
	"log"
	"net/http"

	"github.com/frank-mendez/chatmesh/internal/hub"
	ws "github.com/frank-mendez/chatmesh/internal/websocket"
)

func main() {
	h := hub.New()
	go h.Run()

	http.HandleFunc("/ws", ws.Handler(h))

	log.Println("chatmesh listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
