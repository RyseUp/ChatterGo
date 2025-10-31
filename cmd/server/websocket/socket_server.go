package websocket

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/RyseUp/ChatterGo/internal/models"
	"github.com/RyseUp/ChatterGo/internal/repositories"
	socketio "github.com/googollee/go-socket.io"
)

type ctxKey string

const (
	ctxUserIDKey ctxKey = "user_id"
)

type SocketServer struct {
	Server      *socketio.Server
	Presence    *PresenceRegistry
	repo        repositories.Repository
	resumeStore *ResumeStore
}

func NewSocketServer(repo repositories.Repository) (*SocketServer, error) {
	server := socketio.NewServer(nil)

	presence := NewPresenceRegistry()
	resumeStore := NewResumeStore()

	ss := &SocketServer{
		Server:      server,
		Presence:    presence,
		repo:        repo,
		resumeStore: resumeStore,
	}

	// Connection auth & setup
	server.OnConnect("/", func(s socketio.Conn) error {
		r := s.RemoteHeader()
		url := s.URL()
		uid, err := VerifySocketJWT(&http.Request{
			Header: r,
			URL:    &url,
		})
		if err != nil {
			log.Printf("Authentication failed for socket %s: %v", s.ID(), err)
			return fmt.Errorf("authentication failed: %v", err)
		}
		s.SetContext(map[ctxKey]any{
			ctxUserIDKey: uid,
		})

		personal := userRoom(uid)
		s.Join(personal)

		wasOnline := presence.Add(uid)
		log.Printf("Socket connected: %s -> user %d (wasOnline: %v)", s.ID(), uid, wasOnline)

		server.BroadcastToRoom("/", personal, EventPresenceOnline, uid)

		// Generate resume token for reconnection
		resumeToken := resumeStore.GenerateToken(uid, s.ID())
		s.Emit(EventServerAck, map[string]any{
			"message":      fmt.Sprintf("welcome user %d", uid),
			"resume_token": resumeToken,
		})
		return nil
	})

	// Join Room
	server.OnEvent("/", EventJoinRoom, func(s socketio.Conn, p JoinRoomPayload) {
		if p.RoomID == "" {
			s.Emit(EventError, "room_id is required")
			return
		}
		s.Join(p.RoomID)
		s.Emit(EventServerAck, "joined room "+p.RoomID)
		log.Printf("Socket %s joined room %s", s.ID(), p.RoomID)
	})

	// Leave Room
	server.OnEvent("/", EventLeaveRoom, func(s socketio.Conn, p LeaveRoomPayload) {
		if p.RoomID == "" {
			s.Emit(EventError, "room_id is required")
			return
		}
		s.Leave(p.RoomID)
		s.Emit(EventServerAck, "left room "+p.RoomID)
		log.Printf("Socket %s left room %s", s.ID(), p.RoomID)
	})

	// Typing Start/Stop
	server.OnEvent("/", EventTypingStart, func(s socketio.Conn, p TypingPayload) {
		if p.RoomID == "" {
			log.Printf("Socket %s has no room id", s.ID())
			return
		}
		server.BroadcastToRoom("/", p.RoomID, EventTypingStart, s.ID())
	})
	server.OnEvent("/", EventTypingStop, func(s socketio.Conn, p TypingPayload) {
		if p.RoomID == "" {
			log.Printf("Socket %s has no room id", s.ID())
			return
		}
		server.BroadcastToRoom("/", p.RoomID, EventTypingStop, s.ID())
	})

	// Send Message
	server.OnEvent("/", EventMessageSend, func(s socketio.Conn, p SendMessagePayload) {
		if p.RoomID == "" || p.Message == "" {
			s.Emit(EventError, "missing room_id or message")
			return
		}
		userID := userIDFromConn(s)

		// Persist message to DB
		msg, err := ss.persistMessage(userID, p)
		if err != nil {
			log.Printf("Failed to persist message: %v", err)
			s.Emit(EventError, "failed to save message")
			return
		}

		// Broadcast to room
		server.BroadcastToRoom("/", p.RoomID, EventMessageCreated, msg)
		log.Printf("User %d sent message to room %s", userID, p.RoomID)
	})

	// Delivery Receipt
	server.OnEvent("/", EventDeliveryReceipt, func(s socketio.Conn, p DeliveryReceiptPayload) {
		if p.RoomID == "" || p.MessageID == "" || p.Status == "" {
			return
		}
		userID := userIDFromConn(s)

		// Update delivery status in DB
		if err := ss.updateDeliveryStatus(p, userID); err != nil {
			log.Printf("Failed to update delivery status: %v", err)
			return
		}

		// Broadcast delivery update to room
		server.BroadcastToRoom("/", p.RoomID, EventDeliveryUpdated, p)
		log.Printf("User %d updated delivery status for message %s: %s", userID, p.MessageID, p.Status)
	})

	// Resume connection
	server.OnEvent("/", EventResume, func(s socketio.Conn, p ResumePayload) {
		if p.ResumeToken == "" {
			s.Emit(EventError, "resume_token is required")
			return
		}

		session, ok := resumeStore.Get(p.ResumeToken)
		if !ok {
			s.Emit(EventError, "invalid or expired resume_token")
			return
		}

		userID := userIDFromConn(s)
		if session.UserID != userID {
			s.Emit(EventError, "resume_token does not match user")
			return
		}

		// Fetch missed messages since last connection
		missedMessages, err := ss.getMissedMessages(userID, session.LastMessageID)
		if err != nil {
			log.Printf("Failed to fetch missed messages: %v", err)
			s.Emit(EventError, "failed to fetch missed messages")
			return
		}

		// Send missed messages
		for _, msg := range missedMessages {
			s.Emit(EventMessageCreated, msg)
		}

		s.Emit(EventServerAck, map[string]any{
			"message":         "resume successful",
			"missed_count":    len(missedMessages),
			"last_message_id": session.LastMessageID,
		})
		log.Printf("User %d resumed connection, replayed %d messages", userID, len(missedMessages))
	})

	// Ping/Pong heartbeat
	server.OnEvent("/", EventPing, func(s socketio.Conn, timestamp int64) {
		s.Emit(EventPong, map[string]any{
			"client_timestamp": timestamp,
			"server_timestamp": time.Now().Unix(),
		})
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		uid := userIDFromConn(s)
		isOffline := presence.Remove(uid)
		log.Printf("user %d disconnected socket=%s reason=%s offline=%v", uid, s.ID(), reason, isOffline)
		if isOffline {
			server.BroadcastToRoom("/", userRoom(uid), EventPresenceOffline, uid)
		}
	})

	go func() {
		if err := server.Serve(); err != nil {
			log.Fatalf("SocketIO server error: %v", err)
		}
	}()

	return ss, nil
}

func (s *SocketServer) Close() {
	if s.resumeStore != nil {
		s.resumeStore.Stop()
	}
	s.Server.Close()
}

func (s *SocketServer) Handler() http.Handler {
	return s.Server
}

func userRoom(userId uint64) string {
	return fmt.Sprintf("user-%d", userId)
}

func userIDFromConn(c socketio.Conn) uint64 {
	ctx, _ := c.Context().(map[ctxKey]any)
	if ctx == nil {
		return 0
	}
	if val, ok := ctx[ctxUserIDKey].(uint64); ok {
		return val
	}
	return 0
}

// persistMessage saves message to database and returns the created message
func (s *SocketServer) persistMessage(userID uint64, p SendMessagePayload) (*MessageCreatedPayload, error) {
	// Parse conversation ID from room_id (assuming format: "conv-{id}")
	var conversationID uint
	_, err := fmt.Sscanf(p.RoomID, "conv-%d", &conversationID)

	if err != nil || conversationID == 0 {
		return nil, fmt.Errorf("invalid room_id format: %s", p.RoomID)
	}

	// Create message in DB
	ctx := context.Background()
	msg := &models.Message{
		ConversationID: conversationID,
		SenderID:       uint(userID),
		Content:        p.Message,
	}

	if err := s.repo.CreateMessage(ctx, msg); err != nil {
		return nil, err
	}

	return &MessageCreatedPayload{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		SenderID:       msg.SenderID,
		Content:        msg.Content,
		CreatedAt:      msg.CreatedAt,
		UpdatedAt:      msg.UpdatedAt,
		ClientID:       p.ClientID,
	}, nil
}

// updateDeliveryStatus updates message delivery/read status
func (s *SocketServer) updateDeliveryStatus(p DeliveryReceiptPayload, userID uint64) error {
	// Parse message ID
	var messageID uint
	_, err := fmt.Sscanf(p.MessageID, "%d", &messageID)

	if err != nil || messageID == 0 {
		return fmt.Errorf("invalid message_id: %s", p.MessageID)
	}

	// TODO: Implement delivery status tracking in your database schema
	// For now, we just log it. You'll need to add a delivery_receipts table
	// or update the messages table with read_by/delivered_to fields
	log.Printf("Delivery receipt: message=%d, user=%d, status=%s", messageID, userID, p.Status)
	return nil
}

// getMissedMessages fetches messages after a given message ID for resume
func (s *SocketServer) getMissedMessages(userID uint64, lastMessageID uint64) ([]*MessageCreatedPayload, error) {
	ctx := context.Background()

	// Get user's conversations
	conversations, _, err := s.repo.GetConversationsByUserID(ctx, uint(userID), 50, 0)
	if err != nil {
		return nil, err
	}

	var allMissedMessages []*MessageCreatedPayload

	// For each conversation, get messages after lastMessageID
	for _, conv := range conversations {
		messages, _, err := s.repo.GetMessagesByConversationID(ctx, conv.ID, 100, 0)
		if err != nil {
			log.Printf("Failed to get messages for conversation %d: %v", conv.ID, err)
			continue
		}

		// Filter messages after lastMessageID
		for _, msg := range messages {
			if uint64(msg.ID) > lastMessageID {
				allMissedMessages = append(allMissedMessages, &MessageCreatedPayload{
					ID:             msg.ID,
					ConversationID: msg.ConversationID,
					SenderID:       msg.SenderID,
					Content:        msg.Content,
					CreatedAt:      msg.CreatedAt,
					UpdatedAt:      msg.UpdatedAt,
				})
			}
		}
	}

	return allMissedMessages, nil
}
