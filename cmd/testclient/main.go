package main

import (
	"encoding/json"
	"log"
	"time"

	gorillaws "github.com/gorilla/websocket"

	"github.com/frank-mendez/chatmesh/internal/models"
)

const serverURL = "ws://localhost:8080/ws"

func dial(room, user string) *gorillaws.Conn {
	url := serverURL + "?room=" + room + "&user=" + user
	conn, _, err := gorillaws.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatalf("dial %s: %v", url, err)
	}
	return conn
}

func main() {
	alice := dial("general", "alice")
	defer alice.Close()

	bob := dial("general", "bob")
	defer bob.Close()

	// Give the server time to register both clients before broadcasting.
	time.Sleep(50 * time.Millisecond)

	// Alice sends a message.
	msg := models.Message{Type: "message", Room: "general", User: "alice", Content: "hello bob"}
	if err := alice.WriteJSON(msg); err != nil {
		log.Fatalf("alice write: %v", err)
	}

	// Bob reads the broadcast with a deadline so the program always exits.
	bob.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, raw, err := bob.ReadMessage()
	if err != nil {
		log.Fatalf("bob read: %v", err)
	}

	var got models.Message
	if err := json.Unmarshal(raw, &got); err != nil {
		log.Fatalf("unmarshal: %v", err)
	}

	log.Printf("bob received: user=%q content=%q room=%q", got.User, got.Content, got.Room)

	if got.User != "alice" || got.Content != "hello bob" || got.Room != "general" {
		log.Fatalf("FAIL: unexpected message %+v", got)
	}
	log.Println("PASS")
}
