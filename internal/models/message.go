package models

// Message is the wire format for all WebSocket communication.
type Message struct {
	Type    string `json:"type"`
	Room    string `json:"room"`
	User    string `json:"user"`
	Content string `json:"content"`
}
