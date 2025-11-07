package models

import (
	"time"

	"gorm.io/gorm"
)

type Media struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	MessageID *uint          `gorm:"index" json:"message_id"` // Nullable: can be uploaded before message is created
	URL       string         `gorm:"size:500;not null" json:"url"`
	MimeType  string         `gorm:"size:100;not null" json:"mime_type"`
	Size      int64          `gorm:"not null" json:"size"`
	Filename  string         `gorm:"size:255;not null" json:"filename"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Message *Message `gorm:"foreignKey:MessageID;constraint:OnDelete:CASCADE" json:"message,omitempty"`
}
