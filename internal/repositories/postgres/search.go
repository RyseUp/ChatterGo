package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/RyseUp/ChatterGo/internal/models"
)

// SearchUsers searches for users by name or email
func (q *Queries) SearchUsers(ctx context.Context, query string, limit, offset int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	// Sanitize query
	query = strings.TrimSpace(query)
	if query == "" {
		return users, 0, nil
	}

	searchPattern := "%" + strings.ToLower(query) + "%"

	// Count total matching users
	q.db.WithContext(ctx).Model(&models.User{}).
		Where("LOWER(username) LIKE ? OR LOWER(email) LIKE ?", searchPattern, searchPattern).
		Count(&total)

	// Get users with pagination
	err := q.db.WithContext(ctx).
		Where("LOWER(username) LIKE ? OR LOWER(email) LIKE ?", searchPattern, searchPattern).
		Order("username ASC").
		Limit(limit).
		Offset(offset).
		Find(&users).Error

	return users, total, err
}

// SearchMessages searches for messages using PostgreSQL full-text search
func (q *Queries) SearchMessages(ctx context.Context, query string, conversationID *uint, limit, offset int) ([]models.Message, int64, error) {
	var messages []models.Message
	var total int64

	// Sanitize query
	query = strings.TrimSpace(query)
	if query == "" {
		return messages, 0, nil
	}

	// Prepare full-text search query
	// Convert spaces to & for AND search
	tsQuery := strings.ReplaceAll(query, " ", " & ")
	
	db := q.db.WithContext(ctx).Model(&models.Message{})

	// Add conversation filter if specified
	if conversationID != nil {
		db = db.Where("conversation_id = ?", *conversationID)
	}

	// Use PostgreSQL full-text search
	searchCondition := "to_tsvector('english', content) @@ to_tsquery('english', ?)"
	
	// Count total matching messages
	db.Where(searchCondition, tsQuery).Count(&total)

	// Get messages with pagination, preload relationships
	err := db.
		Preload("Sender").
		Preload("Conversation").
		Preload("Media").
		Where(searchCondition, tsQuery).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error

	return messages, total, err
}

// SearchConversations searches for conversations by name
func (q *Queries) SearchConversations(ctx context.Context, query string, userID uint, limit, offset int) ([]models.Conversation, int64, error) {
	var conversations []models.Conversation
	var total int64

	// Sanitize query
	query = strings.TrimSpace(query)
	if query == "" {
		return conversations, 0, nil
	}

	searchPattern := "%" + strings.ToLower(query) + "%"

	// Subquery to get conversations where user is a member
	memberSubquery := q.db.Model(&models.ConversationMember{}).
		Select("conversation_id").
		Where("user_id = ?", userID)

	db := q.db.WithContext(ctx).Model(&models.Conversation{}).
		Where("id IN (?)", memberSubquery)

	// Add name search condition (only for group conversations that have names)
	db = db.Where("name IS NOT NULL AND LOWER(name) LIKE ?", searchPattern)

	// Count total
	db.Count(&total)

	// Get conversations with pagination
	err := db.
		Preload("Members").
		Preload("Members.User").
		Order("updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&conversations).Error

	return conversations, total, err
}

// SearchAll performs a combined search across users, messages, and conversations
func (q *Queries) SearchAll(ctx context.Context, query string, userID uint, limit int) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Search users (limit to 5 for combined search)
	users, _, err := q.SearchUsers(ctx, query, 5, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	result["users"] = users

	// Search messages (limit to 10 for combined search)
	messages, _, err := q.SearchMessages(ctx, query, nil, 10, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}
	result["messages"] = messages

	// Search conversations (limit to 5 for combined search)
	conversations, _, err := q.SearchConversations(ctx, query, userID, 5, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to search conversations: %w", err)
	}
	result["conversations"] = conversations

	return result, nil
}
