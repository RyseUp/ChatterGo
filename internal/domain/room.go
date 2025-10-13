package domain

import "time"

type Room struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`
	CreatedBy   int64     `json:"created_by" db:"created_by"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type RoomMember struct {
	ID        int64     `json:"id" db:"id"`
	RoomID    int64     `json:"room_id" db:"room_id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	JoinedAt  time.Time `json:"joined_at" db:"joined_at"`
	IsOnline  bool      `json:"is_online" db:"is_online"`
	LastSeen  time.Time `json:"last_seen" db:"last_seen"`
}

type CreateRoomRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description,omitempty" binding:"max=500"`
}

type JoinRoomRequest struct {
	RoomID int64 `json:"room_id" binding:"required"`
}
