package models

import (
	"time"

	"gorm.io/gorm"
)

type NotificationType string

const (
	NotificationTypeMessage      NotificationType = "message"
	NotificationTypeMention      NotificationType = "mention"
	NotificationTypeConversation NotificationType = "conversation"
	NotificationTypeSystem       NotificationType = "system"
)

type NotificationStatus string

const (
	NotificationStatusUnread NotificationStatus = "unread"
	NotificationStatusRead   NotificationStatus = "read"
	NotificationStatusSent   NotificationStatus = "sent"
)

type Notification struct {
	ID             uint               `gorm:"primaryKey" json:"id"`
	UserID         uint               `gorm:"not null;index" json:"user_id"`
	Type           NotificationType   `gorm:"type:varchar(20);not null" json:"type"`
	Title          string             `gorm:"size:255;not null" json:"title"`
	Message        string             `gorm:"type:text;not null" json:"message"`
	Data           string             `gorm:"type:jsonb" json:"data,omitempty"` // JSON data for additional info
	Status         NotificationStatus `gorm:"type:varchar(20);not null;default:'unread'" json:"status"`
	ConversationID *uint              `gorm:"index" json:"conversation_id,omitempty"`
	MessageID      *uint              `gorm:"index" json:"message_id,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
	DeletedAt      gorm.DeletedAt     `gorm:"index" json:"-"`

	// Relationships
	User         User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Conversation *Conversation `gorm:"foreignKey:ConversationID" json:"conversation,omitempty"`
	RelatedMessage *Message    `gorm:"foreignKey:MessageID" json:"related_message,omitempty"`
}

type NotificationPreference struct {
	ID                    uint `gorm:"primaryKey" json:"id"`
	UserID                uint `gorm:"not null;uniqueIndex" json:"user_id"`
	MessageNotifications  bool `gorm:"default:true" json:"message_notifications"`
	MentionNotifications  bool `gorm:"default:true" json:"mention_notifications"`
	ConversationNotifications bool `gorm:"default:true" json:"conversation_notifications"`
	SystemNotifications   bool `gorm:"default:true" json:"system_notifications"`
	EmailNotifications    bool `gorm:"default:false" json:"email_notifications"`
	PushNotifications     bool `gorm:"default:true" json:"push_notifications"`
	DoNotDisturb          bool `gorm:"default:false" json:"do_not_disturb"`
	DoNotDisturbStart     *string `gorm:"size:5" json:"do_not_disturb_start,omitempty"` // HH:MM format
	DoNotDisturbEnd       *string `gorm:"size:5" json:"do_not_disturb_end,omitempty"`   // HH:MM format
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`

	// Relationships
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
