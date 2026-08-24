package websocket

type ChatMessage struct {
	Type       string  `json:"type"`
	SenderID   string  `json:"sender_id"`
	ReceiverID string  `json:"receiver_id"`
	Message    string  `json:"message"`
	ImageKey   *string `json:"image_key,omitempty"`
	MessageID  string  `json:"message_id"`
}
