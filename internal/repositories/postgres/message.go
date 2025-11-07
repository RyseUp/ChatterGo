package postgres

import (
	"context"
	"fmt"

	"github.com/RyseUp/ChatterGo/internal/models"
)

// CreateMessage creates a new message
func (r *Queries) CreateMessage(ctx context.Context, message *models.Message) error {
	return r.db.WithContext(ctx).Model(&models.Message{}).Create(message).Error
}

// GetMessageByID retrieves a message by ID with sender information
func (r *Queries) GetMessageByID(ctx context.Context, id uint) (*models.Message, error) {
	var message models.Message
	err := r.db.WithContext(ctx).
		Preload("Sender").
		Preload("Media").
		Order("created_at ASC").
		First(&message, id).Error
	if err != nil {
		return nil, fmt.Errorf("GetMessageByID: %w", err)
	}
	return &message, nil
}

// GetMessagesByConversationID retrieves messages for a conversation with pagination
func (r *Queries) GetMessagesByConversationID(ctx context.Context, conversationID uint, limit, offset int) ([]models.Message, int64, error) {
	var messages []models.Message
	var total int64

	// Get total count
	err := r.db.WithContext(ctx).
		Model(&models.Message{}).
		Where("conversation_id = ?", conversationID).
		Count(&total).Error
	if err != nil {
		return nil, 0, fmt.Errorf("GetMessagesByConversationID count: %w", err)
	}

	// Get messages with pagination, ordered by creation time (newest first)
	err = r.db.WithContext(ctx).
		Preload("Sender").
		Preload("Media").
		Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error
	if err != nil {
		return nil, 0, fmt.Errorf("GetMessagesByConversationID: %w", err)
	}

	return messages, total, nil
}

// UpdateMessage updates a message
func (r *Queries) UpdateMessage(ctx context.Context, id uint, updates map[string]interface{}) error {
	err := r.db.WithContext(ctx).Model(&models.Message{}).Where("id = ?", id).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("UpdateMessage: %w", err)
	}
	return nil
}

// DeleteMessage soft deletes a message
func (r *Queries) DeleteMessage(ctx context.Context, id uint) error {
	err := r.db.WithContext(ctx).Delete(&models.Message{}, id).Error
	if err != nil {
		return fmt.Errorf("DeleteMessage: %w", err)
	}
	return nil
}
