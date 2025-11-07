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
	MediaID  uint   `json:"media_id,omitempty"` // Optional: ID of uploaded media to attach
}

type DeliveryReceiptPayload struct {
	RoomID    string `json:"room_id"`
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
}

type ResumePayload struct {
	ResumeToken string `json:"resume_token"`
}

type MediaPayload struct {
	ID       uint   `json:"id"`
	URL      string `json:"url"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
	Filename string `json:"filename"`
}

type MessageCreatedPayload struct {
	ID             uint           `json:"id"`
	ConversationID uint           `json:"conversation_id"`
	SenderID       uint           `json:"sender_id"`
	Content        string         `json:"content"`
	Media          []MediaPayload `json:"media,omitempty"` // Array of media attachments
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	ClientID       string         `json:"client_id,omitempty"`
}
