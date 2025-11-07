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

// Request/Response structs for direct conversations
type CreateDirectConversationRequest struct {
	RecipientID uint `json:"recipient_id" binding:"required" example:"2"`
}

// Request/Response structs for group conversations
type CreateGroupConversationRequest struct {
	Name      string `json:"name" binding:"required,min=1" example:"Project Discussion"`
	MemberIDs []uint `json:"member_ids" binding:"required,min=1" example:"[2,3,4]"`
}

type ConversationResponse struct {
	ID        uint                         `json:"id" example:"1"`
	Type      models.ConversationType      `json:"type" example:"direct"`
	Name      *string                      `json:"name,omitempty" example:"Project Discussion"`
	Members   []ConversationMemberResponse `json:"members"`
	CreatedAt time.Time                    `json:"created_at" example:"2023-10-24T10:30:00Z"`
	UpdatedAt time.Time                    `json:"updated_at" example:"2023-10-24T10:30:00Z"`
}

type ConversationMemberResponse struct {
	ID       uint              `json:"id" example:"1"`
	UserID   uint              `json:"user_id" example:"2"`
	Role     models.MemberRole `json:"role" example:"member"`
	JoinedAt time.Time         `json:"joined_at" example:"2023-10-24T10:30:00Z"`
	User     UserResponse      `json:"user"`
}

type ConversationListResponse struct {
	Conversations []ConversationResponse `json:"conversations"`
	Total         int64                  `json:"total" example:"5"`
	Page          int                    `json:"page" example:"1"`
	Limit         int                    `json:"limit" example:"10"`
}

// CreateDirectConversation godoc
// @Summary Create a direct conversation
// @Description Create a new direct conversation between two users
// @Tags conversations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateDirectConversationRequest true "Direct conversation data"
// @Success 201 {object} map[string]interface{} "Direct conversation created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 409 {object} map[string]interface{} "Direct conversation already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /conversations/direct [post]
func (s *ServiceServer) CreateDirectConversation(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	fmt.Println("userID", userID)
	fmt.Println("exists", exists)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var req CreateDirectConversationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	fmt.Println("req", req)
	fmt.Println("userID", userID.(uint))
	fmt.Println("req.RecipientID", req.RecipientID)
	// Check if user is trying to create conversation with themselves
	if req.RecipientID == userID.(uint) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "cannot create direct conversation with yourself"})
		return
	}

	// Check if recipient user exists
	_, err := s.r.GetUserByUserID(ctx, req.RecipientID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "recipient user not found"})
		return
	}

	// Check if direct conversation already exists between these two users
	existingConversation, err := s.r.GetDirectConversationBetweenUsers(ctx, userID.(uint), req.RecipientID)
	if err == nil && existingConversation != nil {
		// Direct conversation already exists, return the existing one
		response := s.buildConversationResponse(existingConversation)
		ctx.JSON(http.StatusOK, gin.H{
			"message": "direct conversation already exists",
			"data":    response,
		})
		return
	}

	// Create direct conversation
	conversation := &models.Conversation{
		Type: models.ConversationTypeDirect,
		Name: nil, // Direct conversations don't have names
	}

	if err := s.r.CreateConversation(ctx, conversation); err != nil {
		fmt.Printf("failed to create direct conversation: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create conversation"})
		return
	}

	// Add both users as members (creator as admin, recipient as member)
	members := []*models.ConversationMember{
		{
			ConversationID: conversation.ID,
			UserID:         userID.(uint),
			Role:           models.MemberRoleAdmin,
			JoinedAt:       time.Now(),
		},
		{
			ConversationID: conversation.ID,
			UserID:         req.RecipientID,
			Role:           models.MemberRoleMember,
			JoinedAt:       time.Now(),
		},
	}

	if err := s.r.AddConversationMembers(ctx, members); err != nil {
		fmt.Printf("failed to add conversation members: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add conversation members"})
		return
	}

	// Fetch the complete conversation with members
	completeConversation, err := s.r.GetConversationByID(ctx, conversation.ID)
	if err != nil {
		fmt.Printf("failed to get complete conversation: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get conversation details"})
		return
	}

	response := s.buildConversationResponse(completeConversation)

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "direct conversation created successfully",
		"data":    response,
	})
}

// CreateGroupConversation godoc
// @Summary Create a group conversation
// @Description Create a new group conversation with multiple users
// @Tags conversations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateGroupConversationRequest true "Group conversation data"
// @Success 201 {object} map[string]interface{} "Group conversation created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /conversations/group [post]
func (s *ServiceServer) CreateGroupConversation(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var req CreateGroupConversationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	// Validate minimum group size (creator + at least 2 other members)
	if len(req.MemberIDs) < 2 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "group conversations must have at least 2 other members"})
		return
	}

	// Remove duplicates and creator from member list
	uniqueMembers := make(map[uint]bool)
	var validMemberIDs []uint
	for _, memberID := range req.MemberIDs {
		if memberID != userID.(uint) && !uniqueMembers[memberID] {
			uniqueMembers[memberID] = true
			validMemberIDs = append(validMemberIDs, memberID)
		}
	}

	// Validate that all member users exist
	for _, memberID := range validMemberIDs {
		_, err := s.r.GetUserByUserID(ctx, memberID)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("user with ID %d not found", memberID)})
			return
		}
	}

	// Create group conversation
	conversation := &models.Conversation{
		Type: models.ConversationTypeGroup,
		Name: &req.Name,
	}

	if err := s.r.CreateConversation(ctx, conversation); err != nil {
		fmt.Printf("failed to create group conversation: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create conversation"})
		return
	}

	// Add the creator as admin
	var members []*models.ConversationMember
	members = append(members, &models.ConversationMember{
		ConversationID: conversation.ID,
		UserID:         userID.(uint),
		Role:           models.MemberRoleAdmin,
		JoinedAt:       time.Now(),
	})

	// Add other members
	for _, memberID := range validMemberIDs {
		members = append(members, &models.ConversationMember{
			ConversationID: conversation.ID,
			UserID:         memberID,
			Role:           models.MemberRoleMember,
			JoinedAt:       time.Now(),
		})
	}

	if err := s.r.AddConversationMembers(ctx, members); err != nil {
		fmt.Printf("failed to add conversation members: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add conversation members"})
		return
	}

	// Fetch the complete conversation with members
	completeConversation, err := s.r.GetConversationByID(ctx, conversation.ID)
	if err != nil {
		fmt.Printf("failed to get complete conversation: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get conversation details"})
		return
	}

	response := s.buildConversationResponse(completeConversation)

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "group conversation created successfully",
		"data":    response,
	})
}

// GetConversations godoc
// @Summary List conversations for current user
// @Description Get all conversations for the authenticated user with pagination
// @Tags conversations
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} map[string]interface{} "Conversations retrieved successfully"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /conversations [get]
func (s *ServiceServer) GetConversations(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	conversations, total, err := s.r.GetConversationsByUserID(ctx, userID.(uint), limit, offset)
	if err != nil {
		fmt.Printf("failed to get conversations: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get conversations"})
		return
	}

	var conversationResponses []ConversationResponse
	for _, conv := range conversations {
		conversationResponses = append(conversationResponses, s.buildConversationResponse(&conv))
	}

	response := ConversationListResponse{
		Conversations: conversationResponses,
		Total:         total,
		Page:          page,
		Limit:         limit,
	}

	ctx.JSON(http.StatusOK, gin.H{"data": response})
}

// GetConversation godoc
// @Summary Get conversation by ID
// @Description Get a specific conversation by ID (only for members)
// @Tags conversations
// @Produce json
// @Security BearerAuth
// @Param id path int true "Conversation ID"
// @Success 200 {object} map[string]interface{} "Conversation retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid conversation ID"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 403 {object} map[string]interface{} "Access denied"
// @Failure 404 {object} map[string]interface{} "Conversation not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /conversations/{id} [get]
func (s *ServiceServer) GetConversation(ctx *gin.Context) {
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

	conversation, err := s.r.GetConversationByID(ctx, uint(conversationID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
			return
		}
		fmt.Printf("failed to get conversation: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get conversation"})
		return
	}

	response := s.buildConversationResponse(conversation)
	ctx.JSON(http.StatusOK, gin.H{"data": response})
}

// Helper function to build conversation response
func (s *ServiceServer) buildConversationResponse(conversation *models.Conversation) ConversationResponse {
	var members []ConversationMemberResponse
	for _, member := range conversation.Members {
		memberResponse := ConversationMemberResponse{
			ID:       member.ID,
			UserID:   member.UserID,
			Role:     member.Role,
			JoinedAt: member.JoinedAt,
			User: UserResponse{
				ID:          member.User.ID,
				Email:       member.User.Email,
				Username:    member.User.Username,
				AvatarURL:   member.User.AvatarURL,
				IsActive:    member.User.IsActive,
				LastLoginAt: member.User.LastLoginAt,
				CreatedAt:   member.User.CreatedAt,
				UpdatedAt:   member.User.UpdatedAt,
			},
		}
		members = append(members, memberResponse)
	}

	return ConversationResponse{
		ID:        conversation.ID,
		Type:      conversation.Type,
		Name:      conversation.Name,
		Members:   members,
		CreatedAt: conversation.CreatedAt,
		UpdatedAt: conversation.UpdatedAt,
	}
}
