package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/RyseUp/ChatterGo/internal/models"
	"github.com/gin-gonic/gin"
)

// NotificationService handles notification logic
type NotificationService struct {
	repo   interface {
		CreateNotification(ctx context.Context, notification *models.Notification) error
		GetNotificationPreferenceByUserID(ctx context.Context, userID uint) (*models.NotificationPreference, error)
		CreateNotificationPreference(ctx context.Context, pref *models.NotificationPreference) error
	}
	wsHub interface {
		BroadcastToUser(userID uint, message []byte)
		IsUserOnline(userID uint) bool
	}
}

// NotificationData represents additional data for notifications
type NotificationData struct {
	ConversationID uint   `json:"conversation_id,omitempty"`
	MessageID      uint   `json:"message_id,omitempty"`
	SenderID       uint   `json:"sender_id,omitempty"`
	SenderName     string `json:"sender_name,omitempty"`
	MessagePreview string `json:"message_preview,omitempty"`
}

// WebSocketNotification represents a notification sent via WebSocket
type WebSocketNotification struct {
	Type string                 `json:"type"`
	Data *models.Notification   `json:"data"`
}

// CreateMessageNotification creates a notification for a new message
func (s *ServiceServer) CreateMessageNotification(ctx context.Context, message *models.Message, excludeUserID uint) error {
	// Get conversation members
	members, err := s.r.GetConversationMembers(ctx, message.ConversationID)
	if err != nil {
		return fmt.Errorf("failed to get conversation members: %w", err)
	}

	// Get sender info
	sender, err := s.r.GetUserByUserID(ctx, message.SenderID)
	if err != nil {
		return fmt.Errorf("failed to get sender info: %w", err)
	}

	// Create notification data
	notificationData := NotificationData{
		ConversationID: message.ConversationID,
		MessageID:      message.ID,
		SenderID:       message.SenderID,
		SenderName:     sender.Username,
		MessagePreview: truncateString(message.Content, 100),
	}

	dataJSON, _ := json.Marshal(notificationData)

	// Create notifications for all members except the sender and excluded user
	for _, member := range members {
		if member.UserID == message.SenderID || member.UserID == excludeUserID {
			continue
		}

		// Check user's notification preferences
		prefs, err := s.r.GetNotificationPreferenceByUserID(ctx, member.UserID)
		if err != nil {
			// Create default preferences if not found
			defaultPrefs := &models.NotificationPreference{
				UserID:                    member.UserID,
				MessageNotifications:      true,
				MentionNotifications:      true,
				ConversationNotifications: true,
				SystemNotifications:       true,
				PushNotifications:         true,
			}
			s.r.CreateNotificationPreference(ctx, defaultPrefs)
			prefs = defaultPrefs
		}

		// Skip if user has disabled message notifications
		if !prefs.MessageNotifications {
			continue
		}

		// Check do not disturb
		if prefs.DoNotDisturb && s.isInDoNotDisturbPeriod(prefs) {
			continue
		}

		// Create notification
		notification := &models.Notification{
			UserID:         member.UserID,
			Type:           models.NotificationTypeMessage,
			Title:          fmt.Sprintf("New message from %s", sender.Username),
			Message:        truncateString(message.Content, 200),
			Data:           string(dataJSON),
			ConversationID: &message.ConversationID,
			MessageID:      &message.ID,
			Status:         models.NotificationStatusUnread,
		}

		if err := s.r.CreateNotification(ctx, notification); err != nil {
			continue // Log error but don't fail the entire operation
		}

		// Send real-time notification via WebSocket if user is online
		s.sendWebSocketNotification(member.UserID, notification)
	}

	return nil
}

// sendWebSocketNotification sends a notification via WebSocket
func (s *ServiceServer) sendWebSocketNotification(userID uint, notification *models.Notification) {
	wsNotification := WebSocketNotification{
		Type: "notification",
		Data: notification,
	}

	data, err := json.Marshal(wsNotification)
	if err != nil {
		return
	}

	// This would integrate with your existing WebSocket hub
	// For now, we'll implement a simple broadcast mechanism
	// You would replace this with your actual WebSocket hub implementation
	s.broadcastToUser(userID, data)
}

// broadcastToUser broadcasts data to a specific user via WebSocket
func (s *ServiceServer) broadcastToUser(userID uint, data []byte) {
	if s.wsHub == nil {
		fmt.Printf("WebSocket hub not available, would broadcast to user %d: %s\n", userID, string(data))
		return
	}

	var notification WebSocketNotification
	if err := json.Unmarshal(data, &notification); err != nil {
		fmt.Printf("Failed to unmarshal notification data: %v\n", err)
		return
	}

	err := s.wsHub.BroadcastToUser(userID, notification.Type, notification.Data)
	if err != nil {
		fmt.Printf("Failed to broadcast to user %d: %v\n", userID, err)
	}
}

// isInDoNotDisturbPeriod checks if current time is within do not disturb period
func (s *ServiceServer) isInDoNotDisturbPeriod(prefs *models.NotificationPreference) bool {
	if prefs.DoNotDisturbStart == nil || prefs.DoNotDisturbEnd == nil {
		return false
	}

	now := time.Now()
	currentTime := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())

	start := *prefs.DoNotDisturbStart
	end := *prefs.DoNotDisturbEnd

	// Handle same day period (e.g., 09:00 to 17:00)
	if start <= end {
		return currentTime >= start && currentTime <= end
	}

	// Handle overnight period (e.g., 22:00 to 06:00)
	return currentTime >= start || currentTime <= end
}

// truncateString truncates a string to specified length
func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}

// GetNotifications godoc
// @Summary Get user notifications
// @Description Get paginated list of user notifications
// @Tags notifications
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{} "Notifications retrieved successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /notifications [get]
func (s *ServiceServer) GetNotifications(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(ctx.DefaultQuery("offset", "0"))

	notifications, total, err := s.r.GetNotificationsByUserID(ctx, userID.(uint), limit, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get notifications"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"total":         total,
		"limit":         limit,
		"offset":        offset,
	})
}

// GetUnreadNotifications godoc
// @Summary Get unread notifications
// @Description Get all unread notifications for the user
// @Tags notifications
// @Produce json
// @Success 200 {object} map[string]interface{} "Unread notifications retrieved successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /notifications/unread [get]
func (s *ServiceServer) GetUnreadNotifications(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	notifications, err := s.r.GetUnreadNotificationsByUserID(ctx, userID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get unread notifications"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"count":         len(notifications),
	})
}

// MarkNotificationAsRead godoc
// @Summary Mark notification as read
// @Description Mark a specific notification as read
// @Tags notifications
// @Produce json
// @Param id path int true "Notification ID"
// @Success 200 {object} map[string]interface{} "Notification marked as read"
// @Failure 400 {object} map[string]interface{} "Invalid notification ID"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /notifications/{id}/read [patch]
func (s *ServiceServer) MarkNotificationAsRead(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification ID"})
		return
	}

	if err := s.r.MarkNotificationAsRead(ctx, uint(id)); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark notification as read"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "notification marked as read"})
}

// MarkAllNotificationsAsRead godoc
// @Summary Mark all notifications as read
// @Description Mark all user notifications as read
// @Tags notifications
// @Produce json
// @Success 200 {object} map[string]interface{} "All notifications marked as read"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /notifications/read-all [patch]
func (s *ServiceServer) MarkAllNotificationsAsRead(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	if err := s.r.MarkAllNotificationsAsRead(ctx, userID.(uint)); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark all notifications as read"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "all notifications marked as read"})
}

// GetNotificationPreferences godoc
// @Summary Get notification preferences
// @Description Get user's notification preferences
// @Tags notifications
// @Produce json
// @Success 200 {object} models.NotificationPreference "Notification preferences"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /notifications/preferences [get]
func (s *ServiceServer) GetNotificationPreferences(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	prefs, err := s.r.GetNotificationPreferenceByUserID(ctx, userID.(uint))
	if err != nil {
		// Create default preferences if not found
		defaultPrefs := &models.NotificationPreference{
			UserID:                    userID.(uint),
			MessageNotifications:      true,
			MentionNotifications:      true,
			ConversationNotifications: true,
			SystemNotifications:       true,
			PushNotifications:         true,
		}
		if err := s.r.CreateNotificationPreference(ctx, defaultPrefs); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create notification preferences"})
			return
		}
		prefs = defaultPrefs
	}

	ctx.JSON(http.StatusOK, prefs)
}

// UpdateNotificationPreferences godoc
// @Summary Update notification preferences
// @Description Update user's notification preferences
// @Tags notifications
// @Accept json
// @Produce json
// @Param preferences body map[string]interface{} true "Notification preferences to update"
// @Success 200 {object} map[string]interface{} "Preferences updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /notifications/preferences [patch]
func (s *ServiceServer) UpdateNotificationPreferences(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var updates map[string]interface{}
	if err := ctx.ShouldBindJSON(&updates); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	if err := s.r.UpdateNotificationPreference(ctx, userID.(uint), updates); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update notification preferences"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "notification preferences updated successfully"})
}
