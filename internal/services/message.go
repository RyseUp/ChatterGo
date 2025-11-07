package services

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/RyseUp/ChatterGo/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Request/Response structs for messages
type SendMessageRequest struct {
	Content string `json:"content" binding:"required,min=1" example:"Hello, how are you?"`
}

type MediaResponse struct {
	ID       uint   `json:"id" example:"1"`
	URL      string `json:"url" example:"http://localhost:9090/uploads/image.jpg"`
	MimeType string `json:"mime_type" example:"image/jpeg"`
	Size     int64  `json:"size" example:"102400"`
	Filename string `json:"filename" example:"image.jpg"`
}

type MessageResponse struct {
	ID             uint            `json:"id" example:"1"`
	ConversationID uint            `json:"conversation_id" example:"1"`
	SenderID       uint            `json:"sender_id" example:"2"`
	Content        string          `json:"content" example:"Hello, how are you?"`
	Media          []MediaResponse `json:"media,omitempty" example:"[]"`
	CreatedAt      time.Time       `json:"created_at" example:"2023-10-24T10:30:00Z"`
	UpdatedAt      time.Time       `json:"updated_at" example:"2023-10-24T10:30:00Z"`
	Sender         UserResponse    `json:"sender"`
}

type MessageListResponse struct {
	Messages []MessageResponse `json:"messages"`
	Total    int64             `json:"total" example:"25"`
	Page     int               `json:"page" example:"1"`
	Limit    int               `json:"limit" example:"10"`
}

type UpdateMessageRequest struct {
	Content string `json:"content" binding:"required,min=1" example:"Updated message content"`
}

// SendMessage godoc
// @Summary Send a message to a conversation
// @Description Send a new message to a conversation (only for members)
// @Tags messages
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Conversation ID"
// @Param request body SendMessageRequest true "Message content"
// @Success 201 {object} map[string]interface{} "Message sent successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 403 {object} map[string]interface{} "Access denied"
// @Failure 404 {object} map[string]interface{} "Conversation not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /conversations/{id}/messages [post]
func (s *ServiceServer) SendMessage(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	conversationIDStr := ctx.Param("id")
	conversationID, err := strconv.ParseUint(conversationIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation ID"})
		return
	}

	var req SendMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	// Check if user is a member of the conversation
	isMember, err := s.r.IsConversationMember(ctx, uint(conversationID), userID.(uint))
	if err != nil {
		fmt.Printf("failed to check conversation membership: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if !isMember {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Create the message
	message := &models.Message{
		ConversationID: uint(conversationID),
		SenderID:       userID.(uint),
		Content:        req.Content,
	}

	if err := s.r.CreateMessage(ctx, message); err != nil {
		fmt.Printf("failed to create message: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send message"})
		return
	}

	// Fetch the complete message with sender information
	completeMessage, err := s.r.GetMessageByID(ctx, message.ID)
	if err != nil {
		fmt.Printf("failed to get complete message: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get message details"})
		return
	}

	response := s.buildMessageResponse(completeMessage)

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "message sent successfully",
		"data":    response,
	})
}

// GetMessages godoc
// @Summary Get messages from a conversation
// @Description Get all messages from a conversation with pagination (only for members)
// @Tags messages
// @Produce json
// @Security BearerAuth
// @Param id path int true "Conversation ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{} "Messages retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid conversation ID"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 403 {object} map[string]interface{} "Access denied"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /conversations/{id}/messages [get]
func (s *ServiceServer) GetMessages(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	conversationIDStr := ctx.Param("id")
	conversationID, err := strconv.ParseUint(conversationIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation ID"})
		return
	}

	// Check if user is a member of the conversation
	isMember, err := s.r.IsConversationMember(ctx, uint(conversationID), userID.(uint))
	if err != nil {
		fmt.Printf("failed to check conversation membership: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if !isMember {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	messages, total, err := s.r.GetMessagesByConversationID(ctx, uint(conversationID), limit, offset)
	if err != nil {
		fmt.Printf("failed to get messages: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get messages"})
		return
	}

	var messageResponses []MessageResponse
	for _, msg := range messages {
		messageResponses = append(messageResponses, s.buildMessageResponse(&msg))
	}

	response := MessageListResponse{
		Messages: messageResponses,
		Total:    total,
		Page:     page,
		Limit:    limit,
	}

	ctx.JSON(http.StatusOK, gin.H{"data": response})
}

// UpdateMessage godoc
// @Summary Update a message
// @Description Update a message content (only by the sender)
// @Tags messages
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Message ID"
// @Param request body UpdateMessageRequest true "Updated message content"
// @Success 200 {object} map[string]interface{} "Message updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 403 {object} map[string]interface{} "Access denied"
// @Failure 404 {object} map[string]interface{} "Message not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /messages/{id} [patch]
func (s *ServiceServer) UpdateMessage(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	messageIDStr := ctx.Param("id")
	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid message ID"})
		return
	}

	var req UpdateMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	// Get the message to verify ownership
	message, err := s.r.GetMessageByID(ctx, uint(messageID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
			return
		}
		fmt.Printf("failed to get message: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get message"})
		return
	}

	// Check if the user is the sender of the message
	if message.SenderID != userID.(uint) {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "you can only edit your own messages"})
		return
	}

	// Update the message
	updates := map[string]interface{}{
		"content": req.Content,
	}

	if err := s.r.UpdateMessage(ctx, uint(messageID), updates); err != nil {
		fmt.Printf("failed to update message: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update message"})
		return
	}

	// Get the updated message
	updatedMessage, err := s.r.GetMessageByID(ctx, uint(messageID))
	if err != nil {
		fmt.Printf("failed to get updated message: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get updated message"})
		return
	}

	response := s.buildMessageResponse(updatedMessage)

	ctx.JSON(http.StatusOK, gin.H{
		"message": "message updated successfully",
		"data":    response,
	})
}

// DeleteMessage godoc
// @Summary Delete a message
// @Description Delete a message (only by the sender)
// @Tags messages
// @Produce json
// @Security BearerAuth
// @Param id path int true "Message ID"
// @Success 200 {object} map[string]interface{} "Message deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid message ID"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 403 {object} map[string]interface{} "Access denied"
// @Failure 404 {object} map[string]interface{} "Message not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /messages/{id} [delete]
func (s *ServiceServer) DeleteMessage(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	messageIDStr := ctx.Param("id")
	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid message ID"})
		return
	}

	// Get the message to verify ownership
	message, err := s.r.GetMessageByID(ctx, uint(messageID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
			return
		}
		fmt.Printf("failed to get message: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get message"})
		return
	}

	// Check if the user is the sender of the message
	if message.SenderID != userID.(uint) {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "you can only delete your own messages"})
		return
	}

	// Delete the message (soft delete)
	if err := s.r.DeleteMessage(ctx, uint(messageID)); err != nil {
		fmt.Printf("failed to delete message: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete message"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "message deleted successfully"})
}

// Helper function to build message response
func (s *ServiceServer) buildMessageResponse(message *models.Message) MessageResponse {
	// Build media responses
	var mediaResponses []MediaResponse
	if len(message.Media) > 0 {
		for _, media := range message.Media {
			mediaResponses = append(mediaResponses, MediaResponse{
				ID:       media.ID,
				URL:      media.URL,
				MimeType: media.MimeType,
				Size:     media.Size,
				Filename: media.Filename,
			})
		}
	}

	return MessageResponse{
		ID:             message.ID,
		ConversationID: message.ConversationID,
		SenderID:       message.SenderID,
		Content:        message.Content,
		Media:          mediaResponses,
		CreatedAt:      message.CreatedAt,
		UpdatedAt:      message.UpdatedAt,
		Sender: UserResponse{
			ID:          message.Sender.ID,
			Email:       message.Sender.Email,
			Username:    message.Sender.Username,
			AvatarURL:   message.Sender.AvatarURL,
			IsActive:    message.Sender.IsActive,
			LastLoginAt: message.Sender.LastLoginAt,
			CreatedAt:   message.Sender.CreatedAt,
			UpdatedAt:   message.Sender.UpdatedAt,
		},
	}
}
