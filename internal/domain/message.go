package domain

import "time"

type Message struct {
	ID        int64     `json:"id" db:"id"`
	RoomID    int64     `json:"room_id" db:"room_id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	Username  string    `json:"username,omitempty" db:"username"` // Joined from users table
}

type SendMessageRequest struct {
	RoomID  int64  `json:"room_id" binding:"required"`
	Content string `json:"content" binding:"required,min=1,max=4096"`
}

type MessageHistoryRequest struct {
	RoomID int64 `form:"room_id" binding:"required"`
	Limit  int   `form:"limit"`
	Offset int   `form:"offset"`
}
