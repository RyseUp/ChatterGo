package websocket

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/RyseUp/ChatterGo/internal/models"
	"github.com/RyseUp/ChatterGo/internal/repositories"
	"github.com/RyseUp/ChatterGo/utils"
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
	jwtSecret   string
	// Store authenticated user IDs by socket ID (for when OnConnect isn't called)
	authStore   map[string]uint64
	authMutex   sync.RWMutex
}

func NewSocketServer(repo repositories.Repository, jwtSecret string) (*SocketServer, error) {
	// Configure Socket.IO server options
	// Note: googollee/go-socket.io might need specific configuration
	// ⚠️ CRITICAL: googollee/go-socket.io only supports Socket.IO v1.4 protocol
	// Socket.IO v4 client sends EIO=4, but this server expects EIO=1 or EIO=2
	// This causes connect_error because the handshake protocol doesn't match
	server := socketio.NewServer(nil)
	
	// Log server initialization
	log.Printf("[WEBSOCKET] Initializing Socket.IO server with JWT secret configured")
	log.Printf("[WEBSOCKET] ⚠️ WARNING: Using googollee/go-socket.io (Socket.IO v1.4) with Socket.IO v4 client")
	log.Printf("[WEBSOCKET] ⚠️ This will cause 'connect_error' because protocol versions don't match (EIO=4 vs EIO=1/2)")

	presence := NewPresenceRegistry()
	resumeStore := NewResumeStore()

	ss := &SocketServer{
		Server:      server,
		Presence:    presence,
		repo:        repo,
		resumeStore: resumeStore,
		jwtSecret:   jwtSecret,
		authStore:   make(map[string]uint64),
	}

	// Connection auth & setup
	// Note: We allow connection to establish first, then authenticate
	// This prevents 403 errors during WebSocket upgrade handshake
	// IMPORTANT: OnConnect might only be called after WebSocket upgrade, not during polling
	log.Printf("[WEBSOCKET] Registering OnConnect handler for namespace '/'")
	server.OnConnect("/", func(s socketio.Conn) error {
		// Use defer to catch any panics and prevent server errors
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[PANIC] Recovered in OnConnect socket %s: %v", s.ID(), r)
				// Try to emit error before closing
				defer func() {
					if r2 := recover(); r2 != nil {
						log.Printf("[PANIC] Failed to emit error after panic: %v", r2)
					}
				}()
				s.Emit(EventError, map[string]interface{}{
					"error": "server error",
					"message": fmt.Sprintf("internal server error: %v", r),
				})
			}
		}()

		r := s.RemoteHeader()
		url := s.URL()
		
		// Log connection attempt for debugging
		log.Printf("[WEBSOCKET] 🎉 OnConnect CALLED! SocketID: %s, URL: %s", s.ID(), url.String())
		log.Printf("[WEBSOCKET] Query params: %v", url.Query())
		
		// Allow connection to establish first
		// Authentication will be verified immediately after
		// Create a proper request with all required fields
		req := &http.Request{
			Method: "GET",
			Header: r,
			URL:    &url,
			Host:   url.Host,
		}
		
		log.Printf("[WEBSOCKET] Starting authentication for socket %s", s.ID())
		log.Printf("[WEBSOCKET] Query params in OnConnect: %v", url.Query())
		log.Printf("[WEBSOCKET] Headers in OnConnect: %v", r)
		
		uid, err := ss.VerifySocketJWTFromConn(s, req)
		if err != nil {
			log.Printf("[WEBSOCKET] ❌ Authentication failed for socket %s: %v", s.ID(), err)
			log.Printf("[WEBSOCKET] Query params were: %v", url.Query())
			log.Printf("[WEBSOCKET] ⚠️ If this happens during WebSocket upgrade, it may cause 403")
			
			// Emit error and close connection instead of returning error
			// This prevents 403 status code during WebSocket upgrade
			s.Emit(EventError, map[string]interface{}{
				"error": "authentication failed",
				"message": err.Error(),
			})
			
			// Close connection after a brief delay to allow error message to be sent
			go func() {
				time.Sleep(100 * time.Millisecond)
				s.Close()
			}()
			return nil // Return nil to allow connection, then close it
		}
		
		log.Printf("[WEBSOCKET] ✅ Authentication successful - UserID: %d", uid)
		
		s.SetContext(map[ctxKey]any{
			ctxUserIDKey: uid,
		})

		personal := userRoom(uid)
		s.Join(personal)

		wasOnline := presence.Add(uid)
		log.Printf("[WEBSOCKET] ✅ Socket connected: %s -> user %d (wasOnline: %v)", s.ID(), uid, wasOnline)

		server.BroadcastToRoom("/", personal, EventPresenceOnline, uid)

		// Generate resume token for reconnection
		resumeToken := resumeStore.GenerateToken(uid, s.ID())
		s.Emit(EventServerAck, map[string]any{
			"message":      fmt.Sprintf("welcome user %d", uid),
			"resume_token": resumeToken,
		})
		log.Printf("[WEBSOCKET] ✅ Server acknowledgment sent to socket %s", s.ID())
		
		return nil
	})

	// Helper function to authenticate socket if not already authenticated
	authenticateIfNeeded := func(s socketio.Conn) (uint64, error) {
		uid := userIDFromConn(s)
		if uid == 0 {
			log.Printf("[WEBSOCKET] ⚠️ Socket %s not authenticated, attempting auth...", s.ID())
			r := s.RemoteHeader()
			url := s.URL()
			req := &http.Request{
				Method: "GET",
				Header: r,
				URL:    &url,
				Host:   url.Host,
			}
			authUID, err := ss.VerifySocketJWTFromConn(s, req)
			if err != nil {
				log.Printf("[WEBSOCKET] ❌ Authentication failed: %v", err)
				return 0, err
			}
			s.SetContext(map[ctxKey]any{
				ctxUserIDKey: authUID,
			})
			log.Printf("[WEBSOCKET] ✅ Authenticated socket %s - UserID: %d", s.ID(), authUID)
			return authUID, nil
		}
		return uid, nil
	}

	// Join Room
	server.OnEvent("/", EventJoinRoom, func(s socketio.Conn, p JoinRoomPayload) {
		// Authenticate on first event if not already authenticated
		_, err := authenticateIfNeeded(s)
		if err != nil {
			s.Emit(EventError, map[string]interface{}{
				"error": "authentication failed",
				"message": err.Error(),
			})
			return
		}
		
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
		// Authenticate if needed
		_, err := authenticateIfNeeded(s)
		if err != nil {
			s.Emit(EventError, map[string]interface{}{
				"error": "authentication failed",
				"message": err.Error(),
			})
			return
		}
		
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
		// Authenticate if needed
		uid, err := authenticateIfNeeded(s)
		if err != nil {
			s.Emit(EventError, map[string]interface{}{
				"error": "authentication failed",
				"message": err.Error(),
			})
			return
		}
		_ = uid // Use uid to avoid unused variable
		
		if p.RoomID == "" {
			log.Printf("Socket %s has no room id", s.ID())
			return
		}
		server.BroadcastToRoom("/", p.RoomID, EventTypingStart, s.ID())
	})
	server.OnEvent("/", EventTypingStop, func(s socketio.Conn, p TypingPayload) {
		// Authenticate if needed
		uid, err := authenticateIfNeeded(s)
		if err != nil {
			s.Emit(EventError, map[string]interface{}{
				"error": "authentication failed",
				"message": err.Error(),
			})
			return
		}
		_ = uid // Use uid to avoid unused variable
		
		if p.RoomID == "" {
			log.Printf("Socket %s has no room id", s.ID())
			return
		}
		server.BroadcastToRoom("/", p.RoomID, EventTypingStop, s.ID())
	})

	// Send Message
	server.OnEvent("/", EventMessageSend, func(s socketio.Conn, p SendMessagePayload) {
		// Authenticate if needed
		userID, err := authenticateIfNeeded(s)
		if err != nil {
			s.Emit(EventError, map[string]interface{}{
				"error": "authentication failed",
				"message": err.Error(),
			})
			return
		}
		
		if p.RoomID == "" || p.Message == "" {
			s.Emit(EventError, "missing room_id or message")
			return
		}

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
		// Authenticate if needed
		userID, err := authenticateIfNeeded(s)
		if err != nil {
			s.Emit(EventError, map[string]interface{}{
				"error": "authentication failed",
				"message": err.Error(),
			})
			return
		}
		
		if p.RoomID == "" || p.MessageID == "" || p.Status == "" {
			return
		}

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

	// Error handler - catch connection errors
	// Note: googollee/go-socket.io may not have OnError, but we'll try
	// If it doesn't work, errors will be caught in OnDisconnect with reason
	log.Printf("[WEBSOCKET] Registering error handlers")
	
	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		uid := userIDFromConn(s)
		isOffline := presence.Remove(uid)
		
		// Log disconnect with full details
		log.Printf("[WEBSOCKET] 🔴 DISCONNECT - SocketID: %s, UserID: %d, Reason: %s, Offline: %v", 
			s.ID(), uid, reason, isOffline)
		
		// If user_id is 0, it means authentication failed or context wasn't set
		if uid == 0 {
			log.Printf("[WEBSOCKET] ⚠️ WARNING: Disconnected socket %s has user_id=0", s.ID())
			log.Printf("[WEBSOCKET] ⚠️ This means OnConnect was NEVER called or authentication failed")
			log.Printf("[WEBSOCKET] ⚠️ Disconnect reason: %s", reason)
			
			// Try to get URL info if available
			url := s.URL()
			log.Printf("[WEBSOCKET] ⚠️ Socket URL was: %s", url.String())
			log.Printf("[WEBSOCKET] ⚠️ Query params were: %v", url.Query())
		}
		
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
	// Create a wrapper handler that logs all requests and attempts authentication
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a Socket.IO handshake request
		eio := r.URL.Query().Get("EIO")
		transport := r.URL.Query().Get("transport")
		isHandshake := eio != "" || transport != ""
		
		if isHandshake {
			sid := r.URL.Query().Get("sid")
			log.Printf("Socket.IO Handshake: EIO=%s, transport=%s, path=%s, sid=%s", eio, transport, r.URL.Path, sid)
			
			// WebSocket upgrade request - check if token is present
			if transport == "websocket" && sid != "" {
				log.Printf("[WEBSOCKET] 🔄 WebSocket upgrade request for session %s", sid)
				tokenInQuery := r.URL.Query().Get("token")
				if tokenInQuery == "" {
					log.Printf("[WEBSOCKET] ⚠️ WARNING: WebSocket upgrade request has NO token in query params!")
					log.Printf("[WEBSOCKET] ⚠️ This will cause authentication to fail and return 403")
					log.Printf("[WEBSOCKET] ⚠️ Query params: %v", r.URL.Query())
				} else {
					log.Printf("[WEBSOCKET] ✅ Token found in WebSocket upgrade request")
				}
			}
			
			// ⚠️ Socket.IO v4 client sends EIO=4, but googollee/go-socket.io (v1.4) expects EIO=1 or EIO=2
			// This mismatch causes connect_error on the client
			if eio == "4" {
				log.Printf("[WEBSOCKET] ⚠️ Socket.IO v4 client detected (EIO=4) - server only supports v1.4 (EIO=1/2)")
				log.Printf("[WEBSOCKET] ⚠️ This will cause 'connect_error' - protocol versions don't match!")
				log.Printf("[WEBSOCKET] ⚠️ The handshake will likely fail, and OnConnect will NOT be called")
			} else if eio == "1" || eio == "2" || eio == "3" {
				log.Printf("[WEBSOCKET] ✅ Compatible Socket.IO version (EIO=%s)", eio)
			}
		} else {
			log.Printf("[WEBSOCKET] HTTP Request: %s %s, Query: %v", r.Method, r.URL.Path, r.URL.Query())
		}
		
		// Try to extract and validate token from query params for logging
		tokenString := r.URL.Query().Get("token")
		if tokenString != "" {
			// Try to validate token to see if it's valid (for logging only)
			claims, err := utils.ValidateToken(tokenString, s.jwtSecret)
			if err != nil {
				log.Printf("[WEBSOCKET] ⚠️ Token validation failed in HTTP handler: %v", err)
			} else {
				log.Printf("[WEBSOCKET] ✅ Token is valid for user %d", claims.UserID)
			}
		} else {
			log.Printf("[WEBSOCKET] ⚠️ No token in query params")
		}
		
		// ⚠️ CRITICAL: For WebSocket upgrade requests, we need to ensure the session is authenticated
		// If OnConnect wasn't called during polling, the session won't be authenticated
		// and WebSocket upgrade will fail with 403
		if transport == "websocket" && r.URL.Query().Get("sid") != "" {
			sid := r.URL.Query().Get("sid")
			log.Printf("[WEBSOCKET] 🔄 WebSocket upgrade for session %s - checking if authenticated", sid)
			
			// Check if this session is already authenticated
			s.authMutex.RLock()
			_, isAuthenticated := s.authStore[sid]
			s.authMutex.RUnlock()
			
			if !isAuthenticated && tokenString != "" {
				// Session not authenticated yet - authenticate it now
				claims, err := utils.ValidateToken(tokenString, s.jwtSecret)
				if err == nil {
					log.Printf("[WEBSOCKET] ✅ Pre-authenticating session %s for user %d before WebSocket upgrade", sid, claims.UserID)
					s.authMutex.Lock()
					s.authStore[sid] = uint64(claims.UserID)
					s.authMutex.Unlock()
				} else {
					log.Printf("[WEBSOCKET] ⚠️ Cannot pre-authenticate session %s: %v", sid, err)
				}
			} else if isAuthenticated {
				log.Printf("[WEBSOCKET] ✅ Session %s already authenticated", sid)
			}
		}
		
		// Wrap response writer to catch errors and log response
		wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: 200}
		
		// Pass to Socket.IO server - it will handle the handshake
		// OnConnect will be called after connection is established (might be after WebSocket upgrade)
		log.Printf("[WEBSOCKET] 📤 Passing request to Socket.IO server...")
		s.Server.ServeHTTP(wrappedWriter, r)
		
		// Log response status
		if wrappedWriter.statusCode >= 400 {
			log.Printf("[WEBSOCKET] ⚠️ HTTP Error Response: %d for %s %s", wrappedWriter.statusCode, r.Method, r.URL.Path)
		} else if isHandshake {
			log.Printf("[WEBSOCKET] ✅ Handshake Response: %d (EIO=%s, transport=%s)", 
				wrappedWriter.statusCode, eio, transport)
			if eio == "4" {
				log.Printf("[WEBSOCKET] ⚠️ Even though HTTP 200, the Socket.IO protocol negotiation will fail")
				log.Printf("[WEBSOCKET] ⚠️ Client will receive 'connect_error' and disconnect")
			}
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture status codes
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// VerifySocketJWTFromConn extracts JWT token from Socket.IO connection
// This method uses the JWT secret from config
func (ss *SocketServer) VerifySocketJWTFromConn(s socketio.Conn, r *http.Request) (uint64, error) {
	var tokenString string

	// First, try URL query parameter
	tokenString = r.URL.Query().Get("token")
	
	// Check for auth.token in query
	if tokenString == "" {
		tokenString = r.URL.Query().Get("auth.token")
	}
	
	// Check Authorization header
	if tokenString == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if tokenString == "" {
		queryParams := r.URL.Query()
		return 0, fmt.Errorf("missing token - ensure token is sent in query parameter 'token' or Authorization header. Query params: %v", queryParams)
	}

	// Use ValidateToken with the correct JWT secret from config
	claims, err := utils.ValidateToken(tokenString, ss.jwtSecret)
	if err != nil {
		return 0, fmt.Errorf("failed to validate token: %w", err)
	}

	return uint64(claims.UserID), nil
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
