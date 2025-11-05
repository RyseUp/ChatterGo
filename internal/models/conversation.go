package models

import (
	"time"

	"gorm.io/gorm"
)

type ConversationType string

const (
	ConversationTypeDirect ConversationType = "direct"
	ConversationTypeGroup  ConversationType = "group"
)

type Conversation struct {
	ID        uint             `gorm:"primaryKey" json:"id"`
	Type      ConversationType `gorm:"type:varchar(20);not null" json:"type"`
	Name      *string          `gorm:"size:255" json:"name,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	DeletedAt gorm.DeletedAt   `gorm:"index" json:"-"`

	// Relationships
	Members  []ConversationMember `gorm:"many2many:conversation_members;foreignKey:ID;references:ID" json:"members,omitempty"`
	Messages []Message            `gorm:"many2many:conversation_messages;foreignKey:ID;references:ID" json:"messages,omitempty"`
}

type MemberRole string

const (
	MemberRoleAdmin  MemberRole = "admin"
	MemberRoleMember MemberRole = "member"
)

type ConversationMember struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	ConversationID uint       `gorm:"not null;index" json:"conversation_id"`
	UserID         uint       `gorm:"not null;index" json:"user_id"`
	Role           MemberRole `gorm:"type:varchar(20);not null;default:'member'" json:"role"`
	JoinedAt       time.Time  `json:"joined_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// Relationships
	Conversation Conversation `gorm:"foreignKey:ConversationID" json:"conversation,omitempty"`
	User         User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type Message struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	ConversationID uint           `gorm:"not null;index" json:"conversation_id"`
	SenderID       uint           `gorm:"not null;index" json:"sender_id"`
	Content        string         `gorm:"type:text;not null" json:"content"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Conversation Conversation `gorm:"foreignKey:ConversationID" json:"conversation,omitempty"`
	Sender       User         `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	Media        []Media      `gorm:"foreignKey:MessageID" json:"media,omitempty"`
}
