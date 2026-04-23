package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	gorillaws "github.com/gorilla/websocket"

	"github.com/frank-mendez/chatmesh/internal/models"
)

const (
	server1WS       = "ws://localhost:8081/ws"
	server2WS       = "ws://localhost:8082/ws"
	server1Instance = "http://localhost:8081/instance"
	server2Instance = "http://localhost:8082/instance"
	room            = "general"
	readTimeout     = 2 * time.Second
	connectDelay    = 200 * time.Millisecond
)

func getServerID(instanceURL string) string {
	resp, err := http.Get(instanceURL) //nolint:noctx
	if err != nil {
		log.Fatalf("GET %s: %v", instanceURL, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		ServerID string `json:"server_id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Fatalf("parse /instance response: %v", err)
	}
	return result.ServerID
}

func dialWS(wsURL, user string) *gorillaws.Conn {
	url := fmt.Sprintf("%s?room=%s&user=%s", wsURL, room, user)
	conn, _, err := gorillaws.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatalf("dial %s: %v", url, err)
	}
	return conn
}

func tryRead(conn *gorillaws.Conn, label string) {
	conn.SetReadDeadline(time.Now().Add(readTimeout))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		fmt.Printf("%-7s received:  ❌ NONE (timeout after 2s — expected)\n", label)
		return
	}
	var msg models.Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		fmt.Printf("%-7s received:  ❌ parse error: %v\n", label, err)
		return
	}
	fmt.Printf("%-7s received:  user=%q content=%q room=%q ✅\n", label, msg.User, msg.Content, msg.Room)
}

func main() {
	fmt.Println("=== Phase 2: Message Inconsistency Demo ===")
	fmt.Println()

	id1 := getServerID(server1Instance)
	id2 := getServerID(server2Instance)

	fmt.Println("[setup]")
	fmt.Printf("Alice   → %s  (ws://localhost:8081)\n", id1)
	fmt.Printf("Bob     → %s  (ws://localhost:8082)\n", id2)
	fmt.Printf("Charlie → %s  (ws://localhost:8081)\n", id1)
	fmt.Println()

	fmt.Println("[expectation]")
	fmt.Println("Charlie should receive Alice's message ✅  (same instance)")
	fmt.Println("Bob should NOT receive Alice's message ❌  (different instance)")
	fmt.Println()

	alice := dialWS(server1WS, "alice")
	defer alice.Close()
	charlie := dialWS(server1WS, "charlie")
	defer charlie.Close()
	bob := dialWS(server2WS, "bob")
	defer bob.Close()

	time.Sleep(connectDelay)

	msg := models.Message{Type: "message", Room: room, User: "alice", Content: "hello"}
	payload, _ := json.Marshal(msg)

	fmt.Println("[sending]")
	fmt.Printf("Alice sends: %s\n", payload)
	fmt.Println()

	if err := alice.WriteMessage(gorillaws.TextMessage, payload); err != nil {
		log.Fatalf("alice write: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	fmt.Println("[result]")
	tryRead(charlie, "Charlie")
	tryRead(bob, "Bob")
	fmt.Println()

	fmt.Println("[conclusion]")
	fmt.Println("Messages are NOT shared across instances.")
	fmt.Println("In-memory state is isolated per server.")
	fmt.Println("This system is not horizontally scalable as-is.")
	fmt.Println("Phase 3 will fix this with Redis Pub/Sub.")
}
