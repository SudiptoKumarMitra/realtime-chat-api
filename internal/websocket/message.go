package websocket

// Message is the structured protocol for all WebSocket communication.
// Type identifies the kind of message (e.g., "message", "system").
// Content carries the payload text.
// IMPORTANT: Content must never be written to logs — it contains user data.
// Later milestones can add fields like Room, Sender, and Timestamp.
type Message struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	RoomID  string `json:"room_id,omitempty"`
}
