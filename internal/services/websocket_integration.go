package services

import (
	"context"
	"fmt"
	"log"

	"github.com/RyseUp/ChatterGo/cmd/server/websocket"
	"github.com/RyseUp/ChatterGo/internal/models"
	socketio "github.com/googollee/go-socket.io"
)

// WebSocketNotificationHub implements WebSocketHub using the existing socket server

// WebSocketNotificationHub implements WebSocketHub using the existing socket server
type WebSocketNotificationHub struct {
	socketServer *websocket.SocketServer
}

// NewWebSocketNotificationHub creates a new WebSocket notification hub
func NewWebSocketNotificationHub(socketServer *websocket.SocketServer) *WebSocketNotificationHub {
	return &WebSocketNotificationHub{
		socketServer: socketServer,
	}
}

// BroadcastToUser sends a notification to a specific user via WebSocket
func (hub *WebSocketNotificationHub) BroadcastToUser(userID uint, eventType string, data interface{}) error {
	if hub.socketServer == nil {
		return fmt.Errorf("socket server not initialized")
	}

	// Check if user is online
	if !hub.IsUserOnline(userID) {
		log.Printf("User %d is offline, skipping WebSocket notification", userID)
		return nil
	}

	// For now, broadcast to all connections (you would need to implement user-specific routing)
	// This is a simplified implementation - in production you'd want proper user-socket mapping
	hub.socketServer.Server.BroadcastToNamespace("/", eventType, data)
	log.Printf("Sent %s notification broadcast for user %d", eventType, userID)

	return nil
}

// IsUserOnline checks if a user has active WebSocket connections
func (hub *WebSocketNotificationHub) IsUserOnline(userID uint) bool {
	if hub.socketServer == nil {
		return false
	}
	// Simplified implementation - you would implement proper user presence tracking
	return true
}

// GetUserSockets returns all socket IDs for a user
func (hub *WebSocketNotificationHub) GetUserSockets(userID uint) []string {
	if hub.socketServer == nil {
		return []string{}
	}
	// Simplified implementation - you would implement proper user-socket mapping
	return []string{}
}

// NotificationWebSocketEvents defines WebSocket event types for notifications
const (
	EventNotificationReceived = "notification_received"
	EventNotificationRead     = "notification_read"
	EventNotificationCount    = "notification_count"
	EventTypingNotification   = "typing_notification"
)

// Enhanced notification service with WebSocket integration
func (s *ServiceServer) SetupWebSocketNotifications(socketServer *websocket.SocketServer) {
	s.wsHub = NewWebSocketNotificationHub(socketServer)
}

// sendWebSocketNotificationEnhanced sends a notification via WebSocket (enhanced version)
func (s *ServiceServer) sendWebSocketNotificationEnhanced(userID uint, notification *models.Notification) {
	if s.wsHub == nil {
		log.Printf("WebSocket hub not initialized, skipping notification broadcast")
		return
	}

	// Send notification received event
	err := s.wsHub.BroadcastToUser(userID, EventNotificationReceived, notification)
	if err != nil {
		log.Printf("Failed to send notification via WebSocket: %v", err)
		return
	}

	// Also send updated unread count
	s.sendUnreadNotificationCount(userID)
}

// sendUnreadNotificationCount sends the current unread notification count to user
func (s *ServiceServer) sendUnreadNotificationCount(userID uint) {
	if s.wsHub == nil {
		return
	}

	// Get unread count
	unreadNotifications, err := s.r.GetUnreadNotificationsByUserID(context.Background(), userID)
	if err != nil {
		log.Printf("Failed to get unread notification count for user %d: %v", userID, err)
		return
	}

	countData := map[string]interface{}{
		"unread_count": len(unreadNotifications),
		"user_id":      userID,
	}

	err = s.wsHub.BroadcastToUser(userID, EventNotificationCount, countData)
	if err != nil {
		log.Printf("Failed to send notification count via WebSocket: %v", err)
	}
}

// broadcastNotificationRead notifies when a notification is marked as read
func (s *ServiceServer) broadcastNotificationRead(userID uint, notificationID uint) {
	if s.wsHub == nil {
		return
	}

	readData := map[string]interface{}{
		"notification_id": notificationID,
		"user_id":         userID,
		"status":          "read",
	}

	err := s.wsHub.BroadcastToUser(userID, EventNotificationRead, readData)
	if err != nil {
		log.Printf("Failed to broadcast notification read status: %v", err)
		return
	}

	// Also update unread count
	s.sendUnreadNotificationCount(userID)
}

// Enhanced message notification with WebSocket integration
func (s *ServiceServer) CreateMessageNotificationWithWebSocket(message *models.Message, excludeUserID uint) error {
	// Create notifications using existing method
	err := s.CreateMessageNotification(context.Background(), message, excludeUserID)
	if err != nil {
		return err
	}

	// Send typing stop notification to indicate message was sent
	if s.wsHub != nil {
		typingData := map[string]interface{}{
			"conversation_id": message.ConversationID,
			"user_id":         message.SenderID,
			"typing":          false,
		}
		
		// Get conversation members to notify them typing stopped
		members, err := s.r.GetConversationMembers(context.Background(), message.ConversationID)
		if err == nil {
			for _, member := range members {
				if member.UserID != message.SenderID {
					s.wsHub.BroadcastToUser(member.UserID, EventTypingNotification, typingData)
				}
			}
		}
	}

	return nil
}

// Add WebSocket hub to ServiceServer
type ServiceServerWithWebSocket struct {
	*ServiceServer
	wsHub WebSocketHub
}

// Enhanced ServiceServer methods with WebSocket integration
func (s *ServiceServer) MarkNotificationAsReadWithWebSocket(notificationID uint, userID uint) error {
	// Mark as read in database
	err := s.r.MarkNotificationAsRead(context.Background(), notificationID)
	if err != nil {
		return err
	}

	// Broadcast read status via WebSocket
	s.broadcastNotificationRead(userID, notificationID)

	return nil
}

func (s *ServiceServer) MarkAllNotificationsAsReadWithWebSocket(userID uint) error {
	// Mark all as read in database
	err := s.r.MarkAllNotificationsAsRead(context.Background(), userID)
	if err != nil {
		return err
	}

	// Send updated count (should be 0)
	s.sendUnreadNotificationCount(userID)

	// Broadcast that all notifications are read
	if s.wsHub != nil {
		readAllData := map[string]interface{}{
			"user_id": userID,
			"status":  "all_read",
		}
		s.wsHub.BroadcastToUser(userID, EventNotificationRead, readAllData)
	}

	return nil
}

// Integration with existing message service
func (s *ServiceServer) SendMessageWithNotifications(conversationID uint, senderID uint, content string) (*models.Message, error) {
	// Create message using existing logic (you would need to extract this from your message service)
	message := &models.Message{
		ConversationID: conversationID,
		SenderID:       senderID,
		Content:        content,
	}

	// Save message to database
	err := s.r.CreateMessage(context.Background(), message)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// Create notifications for conversation members
	err = s.CreateMessageNotificationWithWebSocket(message, 0)
	if err != nil {
		log.Printf("Failed to create message notifications: %v", err)
		// Don't fail the message creation if notifications fail
	}

	return message, nil
}

// WebSocket event handlers for notifications (to be added to the existing WebSocket server)
func SetupNotificationWebSocketEvents(server *websocket.SocketServer, serviceServer *ServiceServer) {
	// Handle notification acknowledgment
	server.Server.OnEvent("/", "notification_ack", func(conn socketio.Conn, data map[string]interface{}) {
		notificationIDFloat, ok := data["notification_id"].(float64)
		if !ok {
			conn.Emit("error", "invalid notification_id")
			return
		}

		notificationID := uint(notificationIDFloat)
		userID := getUserIDFromConn(conn) // You would need to implement this

		err := serviceServer.MarkNotificationAsReadWithWebSocket(notificationID, userID)
		if err != nil {
			log.Printf("Failed to mark notification as read: %v", err)
			conn.Emit("error", "failed to mark notification as read")
			return
		}

		conn.Emit("notification_ack_success", map[string]interface{}{
			"notification_id": notificationID,
		})
	})

	// Handle request for unread notification count
	server.Server.OnEvent("/", "get_notification_count", func(conn socketio.Conn) {
		userID := getUserIDFromConn(conn)
		serviceServer.sendUnreadNotificationCount(userID)
	})

	// Handle mark all notifications as read
	server.Server.OnEvent("/", "mark_all_read", func(conn socketio.Conn) {
		userID := getUserIDFromConn(conn)
		err := serviceServer.MarkAllNotificationsAsReadWithWebSocket(userID)
		if err != nil {
			log.Printf("Failed to mark all notifications as read: %v", err)
			conn.Emit("error", "failed to mark all notifications as read")
			return
		}

		conn.Emit("mark_all_read_success", map[string]interface{}{
			"user_id": userID,
		})
	})
}

// Helper function to get user ID from WebSocket connection
func getUserIDFromConn(conn socketio.Conn) uint {
	// This should match the implementation in your existing WebSocket code
	// You might need to adjust this based on how you store user ID in the connection context
	// For now, return a placeholder - implement based on your auth system
	return 1
}

// Add wsHub field to ServiceServer (you would add this to the struct definition)
// type ServiceServer struct {
//     cfg    *config.Config
//     r      repositories.Repository
//     svr    *http.Server
//     wsHub  WebSocketHub  // Add this field
// }
