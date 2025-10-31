package websocket

import "time"

type JoinRoomPayload struct {
	RoomID string `json:"room_id"`
}

type LeaveRoomPayload struct {
	RoomID string `json:"room_id"`
}

type TypingPayload struct {
	RoomID string `json:"room_id"`
}

type SendMessagePayload struct {
	RoomID   string `json:"room_id"`
	Message  string `json:"message"`
	ClientID string `json:"client_id,omitempty"`
}

type DeliveryReceiptPayload struct {
	RoomID    string `json:"room_id"`
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
}

type ResumePayload struct {
	ResumeToken string `json:"resume_token"`
}

type MessageCreatedPayload struct {
	ID             uint      `json:"id"`
	ConversationID uint      `json:"conversation_id"`
	SenderID       uint      `json:"sender_id"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	ClientID       string    `json:"client_id,omitempty"`
}
