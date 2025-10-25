package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Email        string         `gorm:"uniqueIndex;size:255;not null" json:"email"`
	Username     string         `gorm:"size:64;not null" json:"username"`
	Password     string         `gorm:"size:255;not null" json:"-"`
	RefreshToken *string        `gorm:"size:500" json:"-"`
	LastLoginAt  *time.Time     `gorm:"type:timestamp with time zone" json:"last_login_at,omitempty"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}
