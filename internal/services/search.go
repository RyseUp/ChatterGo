package services

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type SearchType string

const (
	SearchTypeAll           SearchType = "all"
	SearchTypeUsers         SearchType = "users"
	SearchTypeMessages      SearchType = "messages"
	SearchTypeConversations SearchType = "conversations"
)

type SearchRequest struct {
	Query          string     `form:"q" binding:"required"`
	Type           SearchType `form:"type" binding:"omitempty"`
	ConversationID *uint      `form:"conversation_id" binding:"omitempty"`
	Limit          int        `form:"limit" binding:"omitempty"`
	Offset         int        `form:"offset" binding:"omitempty"`
}

type SearchResponse struct {
	Query   string      `json:"query"`
	Type    SearchType  `json:"type"`
	Results interface{} `json:"results"`
	Total   int64       `json:"total,omitempty"`
	Limit   int         `json:"limit"`
	Offset  int         `json:"offset"`
}

// Search godoc
// @Summary Search across the platform
// @Description Search for users, messages, and conversations
// @Tags search
// @Produce json
// @Param q query string true "Search query"
// @Param type query string false "Search type (all, users, messages, conversations)" default(all)
// @Param conversation_id query int false "Conversation ID for message search"
// @Param limit query int false "Limit results" default(20)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} SearchResponse "Search results"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /search [get]
func (s *ServiceServer) Search(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var req SearchRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	// Set defaults
	if req.Type == "" {
		req.Type = SearchTypeAll
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	// Validate and sanitize query
	req.Query = strings.TrimSpace(req.Query)
	if len(req.Query) < 2 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "search query must be at least 2 characters"})
		return
	}

	var response SearchResponse
	var err error

	switch req.Type {
	case SearchTypeUsers:
		response, err = s.searchUsers(ctx, req, userID.(uint))
	case SearchTypeMessages:
		response, err = s.searchMessages(ctx, req, userID.(uint))
	case SearchTypeConversations:
		response, err = s.searchConversations(ctx, req, userID.(uint))
	case SearchTypeAll:
		response, err = s.searchAll(ctx, req, userID.(uint))
	default:
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid search type"})
		return
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "search failed", "details": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// searchUsers performs user search
func (s *ServiceServer) searchUsers(ctx *gin.Context, req SearchRequest, userID uint) (SearchResponse, error) {
	users, total, err := s.r.SearchUsers(ctx, req.Query, req.Limit, req.Offset)
	if err != nil {
		return SearchResponse{}, err
	}

	// Remove sensitive information from user results
	for i := range users {
		users[i].Password = ""
		users[i].RefreshToken = nil
	}

	return SearchResponse{
		Query:   req.Query,
		Type:    SearchTypeUsers,
		Results: users,
		Total:   total,
		Limit:   req.Limit,
		Offset:  req.Offset,
	}, nil
}

// searchMessages performs message search
func (s *ServiceServer) searchMessages(ctx *gin.Context, req SearchRequest, userID uint) (SearchResponse, error) {
	messages, _, err := s.r.SearchMessages(ctx, req.Query, req.ConversationID, req.Limit, req.Offset)
	if err != nil {
		return SearchResponse{}, err
	}

	// Filter messages to only include conversations the user is a member of
	var filteredMessages []interface{}
	for _, message := range messages {
		// Check if user is a member of the conversation
		isMember, err := s.r.IsConversationMember(ctx, message.ConversationID, userID)
		if err != nil || !isMember {
			continue
		}

		// Remove sensitive information
		if message.Sender.Password != "" {
			message.Sender.Password = ""
		}
		if message.Sender.RefreshToken != nil {
			message.Sender.RefreshToken = nil
		}

		filteredMessages = append(filteredMessages, message)
	}

	filteredTotal := int64(len(filteredMessages))
	return SearchResponse{
		Query:   req.Query,
		Type:    SearchTypeMessages,
		Results: filteredMessages,
		Total:   filteredTotal, // Adjusted total after filtering
		Limit:   req.Limit,
		Offset:  req.Offset,
	}, nil
}

// searchConversations performs conversation search
func (s *ServiceServer) searchConversations(ctx *gin.Context, req SearchRequest, userID uint) (SearchResponse, error) {
	conversations, total, err := s.r.SearchConversations(ctx, req.Query, userID, req.Limit, req.Offset)
	if err != nil {
		return SearchResponse{}, err
	}

	// Clean up sensitive information from members
	for i := range conversations {
		for j := range conversations[i].Members {
			conversations[i].Members[j].User.Password = ""
			conversations[i].Members[j].User.RefreshToken = nil
		}
	}

	return SearchResponse{
		Query:   req.Query,
		Type:    SearchTypeConversations,
		Results: conversations,
		Total:   total,
		Limit:   req.Limit,
		Offset:  req.Offset,
	}, nil
}

// searchAll performs combined search across all types
func (s *ServiceServer) searchAll(ctx *gin.Context, req SearchRequest, userID uint) (SearchResponse, error) {
	// Limit each category for combined search
	categoryLimit := 5
	if req.Limit < 15 {
		categoryLimit = req.Limit / 3
	}

	// Search users
	users, _, err := s.r.SearchUsers(ctx, req.Query, categoryLimit, 0)
	if err != nil {
		return SearchResponse{}, err
	}

	// Clean user data
	for i := range users {
		users[i].Password = ""
		users[i].RefreshToken = nil
	}

	// Search messages (only in conversations user is a member of)
	messages, _, err := s.r.SearchMessages(ctx, req.Query, nil, categoryLimit, 0)
	if err != nil {
		return SearchResponse{}, err
	}

	// Filter and clean message data
	var filteredMessages []interface{}
	for _, message := range messages {
		isMember, err := s.r.IsConversationMember(ctx, message.ConversationID, userID)
		if err != nil || !isMember {
			continue
		}

		message.Sender.Password = ""
		message.Sender.RefreshToken = nil
		filteredMessages = append(filteredMessages, message)

		if len(filteredMessages) >= categoryLimit {
			break
		}
	}

	// Search conversations
	conversations, _, err := s.r.SearchConversations(ctx, req.Query, userID, categoryLimit, 0)
	if err != nil {
		return SearchResponse{}, err
	}

	// Clean conversation data
	for i := range conversations {
		for j := range conversations[i].Members {
			conversations[i].Members[j].User.Password = ""
			conversations[i].Members[j].User.RefreshToken = nil
		}
	}

	results := map[string]interface{}{
		"users":         users,
		"messages":      filteredMessages,
		"conversations": conversations,
	}

	return SearchResponse{
		Query:   req.Query,
		Type:    SearchTypeAll,
		Results: results,
		Limit:   req.Limit,
		Offset:  req.Offset,
	}, nil
}

// SearchUsers godoc
// @Summary Search users
// @Description Search for users by username or email
// @Tags search
// @Produce json
// @Param q query string true "Search query"
// @Param limit query int false "Limit results" default(20)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} SearchResponse "User search results"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /search/users [get]
func (s *ServiceServer) SearchUsers(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	query := strings.TrimSpace(ctx.Query("q"))
	if len(query) < 2 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "search query must be at least 2 characters"})
		return
	}

	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(ctx.DefaultQuery("offset", "0"))

	req := SearchRequest{
		Query:  query,
		Type:   SearchTypeUsers,
		Limit:  limit,
		Offset: offset,
	}

	response, err := s.searchUsers(ctx, req, userID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "search failed", "details": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// SearchMessages godoc
// @Summary Search messages
// @Description Search for messages using full-text search
// @Tags search
// @Produce json
// @Param q query string true "Search query"
// @Param conversation_id query int false "Conversation ID to search within"
// @Param limit query int false "Limit results" default(20)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} SearchResponse "Message search results"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /search/messages [get]
func (s *ServiceServer) SearchMessages(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	query := strings.TrimSpace(ctx.Query("q"))
	if len(query) < 2 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "search query must be at least 2 characters"})
		return
	}

	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(ctx.DefaultQuery("offset", "0"))

	var conversationID *uint
	if convIDStr := ctx.Query("conversation_id"); convIDStr != "" {
		if convID, err := strconv.ParseUint(convIDStr, 10, 32); err == nil {
			convIDUint := uint(convID)
			conversationID = &convIDUint
		}
	}

	req := SearchRequest{
		Query:          query,
		Type:           SearchTypeMessages,
		ConversationID: conversationID,
		Limit:          limit,
		Offset:         offset,
	}

	response, err := s.searchMessages(ctx, req, userID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "search failed", "details": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}
