package postgres

import (
	"context"
	"fmt"

	"github.com/RyseUp/ChatterGo/internal/models"
)

// CreateConversation creates a new conversation
func (r *Queries) CreateConversation(ctx context.Context, conversation *models.Conversation) error {
	return r.db.WithContext(ctx).Model(&models.Conversation{}).Create(conversation).Error
}

// GetConversationByID retrieves a conversation by ID with members
func (r *Queries) GetConversationByID(ctx context.Context, id uint) (*models.Conversation, error) {
	var conversation models.Conversation
	err := r.db.WithContext(ctx).
		Preload("Members").
		Preload("Members.User").
		First(&conversation, id).Error
	if err != nil {
		return nil, fmt.Errorf("GetConversationByID: %w", err)
	}
	return &conversation, nil
}

// GetConversationsByUserID retrieves conversations for a specific user with pagination
func (r *Queries) GetConversationsByUserID(ctx context.Context, userID uint, limit, offset int) ([]models.Conversation, int64, error) {
	var conversations []models.Conversation
	var total int64

	// Get total count
	err := r.db.WithContext(ctx).
		Model(&models.Conversation{}).
		Joins("JOIN conversation_members ON conversations.id = conversation_members.conversation_id").
		Where("conversation_members.user_id = ?", userID).
		Count(&total).Error
	if err != nil {
		return nil, 0, fmt.Errorf("GetConversationsByUserID count: %w", err)
	}

	// Get conversations with pagination
	err = r.db.WithContext(ctx).
		Preload("Members").
		Preload("Members.User").
		Joins("JOIN conversation_members ON conversations.id = conversation_members.conversation_id").
		Where("conversation_members.user_id = ?", userID).
		Order("conversations.updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&conversations).Error
	if err != nil {
		return nil, 0, fmt.Errorf("GetConversationsByUserID: %w", err)
	}

	return conversations, total, nil
}

// GetDirectConversationBetweenUsers checks if a direct conversation exists between two users
func (r *Queries) GetDirectConversationBetweenUsers(ctx context.Context, userID1, userID2 uint) (*models.Conversation, error) {
	var conversation models.Conversation
	err := r.db.WithContext(ctx).
		Preload("Members").
		Preload("Members.User").
		Where("type = ?", models.ConversationTypeDirect).
		Joins("JOIN conversation_members cm1 ON conversations.id = cm1.conversation_id AND cm1.user_id = ?", userID1).
		Joins("JOIN conversation_members cm2 ON conversations.id = cm2.conversation_id AND cm2.user_id = ?", userID2).
		First(&conversation).Error
	if err != nil {
		return nil, fmt.Errorf("GetDirectConversationBetweenUsers: %w", err)
	}
	return &conversation, nil
}

// UpdateConversation updates a conversation
func (r *Queries) UpdateConversation(ctx context.Context, id uint, updates map[string]interface{}) error {
	err := r.db.WithContext(ctx).Model(&models.Conversation{}).Where("id = ?", id).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("UpdateConversation: %w", err)
	}
	return nil
}

// DeleteConversation soft deletes a conversation
func (r *Queries) DeleteConversation(ctx context.Context, id uint) error {
	err := r.db.WithContext(ctx).Delete(&models.Conversation{}, id).Error
	if err != nil {
		return fmt.Errorf("DeleteConversation: %w", err)
	}
	return nil
}

// AddConversationMembers adds multiple members to a conversation
func (r *Queries) AddConversationMembers(ctx context.Context, members []*models.ConversationMember) error {
	return r.db.WithContext(ctx).Model(&models.ConversationMember{}).Create(&members).Error
}

// RemoveConversationMember removes a member from a conversation
func (r *Queries) RemoveConversationMember(ctx context.Context, conversationID, userID uint) error {
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Delete(&models.ConversationMember{}).Error
	if err != nil {
		return fmt.Errorf("RemoveConversationMember: %w", err)
	}
	return nil
}

// GetConversationMembers retrieves all members of a conversation
func (r *Queries) GetConversationMembers(ctx context.Context, conversationID uint) ([]models.ConversationMember, error) {
	var members []models.ConversationMember
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("conversation_id = ?", conversationID).
		Find(&members).Error
	if err != nil {
		return nil, fmt.Errorf("GetConversationMembers: %w", err)
	}
	return members, nil
}

// IsConversationMember checks if a user is a member of a conversation
func (r *Queries) IsConversationMember(ctx context.Context, conversationID, userID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("IsConversationMember: %w", err)
	}
	return count > 0, nil
}

// GetMemberRole retrieves the role of a user in a conversation
func (r *Queries) GetMemberRole(ctx context.Context, conversationID, userID uint) (*models.MemberRole, error) {
	var member models.ConversationMember
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		First(&member).Error
	if err != nil {
		return nil, fmt.Errorf("GetMemberRole: %w", err)
	}
	return &member.Role, nil
}
