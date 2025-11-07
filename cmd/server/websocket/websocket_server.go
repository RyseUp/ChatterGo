	package websocket

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/RyseUp/ChatterGo/internal/models"
	"github.com/RyseUp/ChatterGo/internal/repositories"
	"github.com/RyseUp/ChatterGo/utils"
	"github.com/gorilla/websocket"
)

// Message represents a WebSocket message
type Message struct {
	Type    string      `json:"type"`    // Event type (e.g., "join_room", "message.send")
	Payload interface{} `json:"payload"` // Event payload
}

// Response represents a server response
type Response struct {
	Type    string      `json:"type"`    // Response type (e.g., "server.ack", "error")
	Payload interface{} `json:"payload"` // Response payload
}

// Client represents a WebSocket client connection
type Client struct {
	ID       string
	UserID   uint64
	Conn     *websocket.Conn
	Send     chan []byte
	Server   *WebSocketServer
	Rooms    map[string]bool
	mu       sync.RWMutex
	LastPing time.Time
}

// WebSocketServer manages WebSocket connections
type WebSocketServer struct {
	clients      map[*Client]bool
	rooms        map[string]map[*Client]bool
	register     chan *Client
	unregister   chan *Client
	broadcast    chan BroadcastMessage
	upgrader     websocket.Upgrader
	presence     *PresenceRegistry
	resumeStore  *ResumeStore
	repo         repositories.Repository
	jwtSecret    string
	mu           sync.RWMutex
}

// BroadcastMessage represents a message to broadcast to a room
type BroadcastMessage struct {
	RoomID  string
	Type    string
	Payload interface{}
	Exclude *Client
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(repo repositories.Repository, jwtSecret string) *WebSocketServer {
	return &WebSocketServer{
		clients:     make(map[*Client]bool),
		rooms:       make(map[string]map[*Client]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		broadcast:   make(chan BroadcastMessage),
		presence:    NewPresenceRegistry(),
		resumeStore: NewResumeStore(),
		repo:        repo,
		jwtSecret:   jwtSecret,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins in development
				// In production, check against allowed origins
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

// HandleWebSocket handles WebSocket upgrade requests
func (s *WebSocketServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract token from query parameter
	token := r.URL.Query().Get("token")
	if token == "" {
		// Try Authorization header
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
	}

	if token == "" {
		log.Printf("[WEBSOCKET] ❌ Connection rejected: no token provided")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Validate token
	claims, err := utils.ValidateToken(token, s.jwtSecret)
	if err != nil {
		log.Printf("[WEBSOCKET] ❌ Connection rejected: invalid token: %v", err)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Upgrade connection
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WEBSOCKET] ❌ Failed to upgrade connection: %v", err)
		return
	}

	// Create client
	client := &Client{
		ID:       generateClientID(),
		UserID:   uint64(claims.UserID),
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Server:   s,
		Rooms:    make(map[string]bool),
		LastPing: time.Now(),
	}

	log.Printf("[WEBSOCKET] ✅ Client connected: %s (UserID: %d)", client.ID, client.UserID)

	// Register client
	s.register <- client

	// Start client goroutines
	go client.writePump()
	go client.readPump()
}

// Run starts the WebSocket server
func (s *WebSocketServer) Run() {
	for {
		select {
		case client := <-s.register:
			s.mu.Lock()
			s.clients[client] = true
			s.mu.Unlock()

			// Add to user's personal room
			personalRoom := getUserRoom(client.UserID)
			s.joinRoom(client, personalRoom)

			// Update presence
			wasOnline := s.presence.Add(client.UserID)
			log.Printf("[WEBSOCKET] ✅ Client registered: %s (UserID: %d, wasOnline: %v)", 
				client.ID, client.UserID, wasOnline)

			// Send welcome message
			resumeToken := s.resumeStore.GenerateToken(client.UserID, client.ID)
			client.sendResponse(EventServerAck, map[string]interface{}{
				"message":      "welcome",
				"resume_token": resumeToken,
			})

			// Broadcast presence online
			if !wasOnline {
				s.broadcastToRoom(personalRoom, EventPresenceOnline, client.UserID, nil)
			}

		case client := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client.Send)
			}
			s.mu.Unlock()

			// Remove from all rooms
			client.mu.RLock()
			for roomID := range client.Rooms {
				s.leaveRoom(client, roomID)
			}
			client.mu.RUnlock()

			// Update presence
			isOffline := s.presence.Remove(client.UserID)
			log.Printf("[WEBSOCKET] 🔴 Client disconnected: %s (UserID: %d, isOffline: %v)", 
				client.ID, client.UserID, isOffline)

			// Broadcast presence offline
			if isOffline {
				personalRoom := getUserRoom(client.UserID)
				s.broadcastToRoom(personalRoom, EventPresenceOffline, client.UserID, nil)
			}

		case msg := <-s.broadcast:
			s.broadcastToRoom(msg.RoomID, msg.Type, msg.Payload, msg.Exclude)
		}
	}
}

// joinRoom adds a client to a room
func (s *WebSocketServer) joinRoom(client *Client, roomID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.rooms[roomID] == nil {
		s.rooms[roomID] = make(map[*Client]bool)
	}
	s.rooms[roomID][client] = true

	client.mu.Lock()
	client.Rooms[roomID] = true
	client.mu.Unlock()

	log.Printf("[WEBSOCKET] Client %s joined room %s", client.ID, roomID)
}

// leaveRoom removes a client from a room
func (s *WebSocketServer) leaveRoom(client *Client, roomID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room, ok := s.rooms[roomID]; ok {
		delete(room, client)
		if len(room) == 0 {
			delete(s.rooms, roomID)
		}
	}

	client.mu.Lock()
	delete(client.Rooms, roomID)
	client.mu.Unlock()

	log.Printf("[WEBSOCKET] Client %s left room %s", client.ID, roomID)
}

// broadcastToRoom broadcasts a message to all clients in a room
func (s *WebSocketServer) broadcastToRoom(roomID, eventType string, payload interface{}, exclude *Client) {
	s.mu.RLock()
	room, exists := s.rooms[roomID]
	if !exists {
		s.mu.RUnlock()
		return
	}

	// Create response
	response := Response{
		Type:    eventType,
		Payload: payload,
	}
	data, err := json.Marshal(response)
	if err != nil {
		s.mu.RUnlock()
		log.Printf("[WEBSOCKET] ❌ Failed to marshal broadcast message: %v", err)
		return
	}

	// Broadcast to all clients in room
	for client := range room {
		if exclude != nil && client == exclude {
			continue
		}
		select {
		case client.Send <- data:
		default:
			close(client.Send)
			delete(s.clients, client)
		}
	}
	s.mu.RUnlock()
}

// readPump reads messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.Server.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.LastPing = time.Now()
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, messageBytes, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WEBSOCKET] ❌ Read error: %v", err)
			}
			break
		}

		// Log raw message for debugging
		log.Printf("[WEBSOCKET] 📥 Raw message received: %s", string(messageBytes))

		// Parse message
		var msg Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Printf("[WEBSOCKET] ❌ Failed to parse message: %v", err)
			log.Printf("[WEBSOCKET] ❌ Raw message was: %s", string(messageBytes))
			c.sendResponse(EventError, map[string]interface{}{
				"error": "invalid message format",
			})
			continue
		}

		// Handle message
		c.handleMessage(&msg)
	}
}

// writePump writes messages to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage handles incoming messages from client
func (c *Client) handleMessage(msg *Message) {
	log.Printf("[WEBSOCKET] 📨 Received message from client %s: type=%s", c.ID, msg.Type)
	
	// Log raw payload for debugging
	if payloadBytes, err := json.Marshal(msg.Payload); err == nil {
		log.Printf("[WEBSOCKET] 📦 Raw payload: %s", string(payloadBytes))
	}

	switch msg.Type {
	case EventJoinRoom:
		c.handleJoinRoom(msg.Payload)
	case EventLeaveRoom:
		c.handleLeaveRoom(msg.Payload)
	case EventMessageSend:
		c.handleSendMessage(msg.Payload)
	case EventTypingStart:
		c.handleTypingStart(msg.Payload)
	case EventTypingStop:
		c.handleTypingStop(msg.Payload)
	case EventDeliveryReceipt:
		c.handleDeliveryReceipt(msg.Payload)
	case EventResume:
		c.handleResume(msg.Payload)
	case EventPing:
		c.handlePing(msg.Payload)
	default:
		log.Printf("[WEBSOCKET] ⚠️ Unknown message type: %s", msg.Type)
		c.sendResponse(EventError, map[string]interface{}{
			"error": "unknown message type",
		})
	}
}

// handleJoinRoom handles join_room event
func (c *Client) handleJoinRoom(payload interface{}) {
	var p JoinRoomPayload
	if payloadBytes, err := json.Marshal(payload); err == nil {
		log.Printf("[WEBSOCKET] 🔍 Raw join_room payload: %s", string(payloadBytes))
	}
	if err := unmarshalPayload(payload, &p); err != nil {
		c.sendResponse(EventError, map[string]interface{}{
			"error": "invalid join_room payload",
		})
		return
	}

	if p.RoomID == "" {
		c.sendResponse(EventError, map[string]interface{}{
			"error": "room_id is required",
		})
		return
	}

	c.Server.joinRoom(c, p.RoomID)
	c.sendResponse(EventServerAck, map[string]interface{}{
		"message": "joined room " + p.RoomID,
	})
}

// handleLeaveRoom handles leave_room event
func (c *Client) handleLeaveRoom(payload interface{}) {
	var p LeaveRoomPayload
	if err := unmarshalPayload(payload, &p); err != nil {
		c.sendResponse(EventError, map[string]interface{}{
			"error": "invalid leave_room payload",
		})
		return
	}

	if p.RoomID == "" {
		c.sendResponse(EventError, map[string]interface{}{
			"error": "room_id is required",
		})
		return
	}

	c.Server.leaveRoom(c, p.RoomID)
	c.sendResponse(EventServerAck, map[string]interface{}{
		"message": "left room " + p.RoomID,
	})
}

// handleSendMessage handles message.send event
func (c *Client) handleSendMessage(payload interface{}) {
	// Log raw payload before unmarshaling
	if payloadBytes, err := json.Marshal(payload); err == nil {
		log.Printf("[WEBSOCKET] 🔍 Raw message.send payload: %s", string(payloadBytes))
	}
	
	var p SendMessagePayload
	if err := unmarshalPayload(payload, &p); err != nil {
		log.Printf("[WEBSOCKET] ❌ Failed to unmarshal message.send payload: %v", err)
		log.Printf("[WEBSOCKET] ❌ Payload type: %T, value: %+v", payload, payload)
		c.sendResponse(EventError, map[string]interface{}{
			"error": "invalid message.send payload",
		})
		return
	}

	log.Printf("[WEBSOCKET] 📨 Parsed message.send: UserID=%d, RoomID=%s, Message=%s, ClientID=%s, MediaID=%d", 
		c.UserID, p.RoomID, p.Message, p.ClientID, p.MediaID)

	if p.RoomID == "" {
		log.Printf("[WEBSOCKET] ❌ Missing required field: RoomID=%s", p.RoomID)
		c.sendResponse(EventError, map[string]interface{}{
			"error": "room_id is required",
		})
		return
	}

	// Message must have either text content or media
	if p.Message == "" && p.MediaID == 0 {
		log.Printf("[WEBSOCKET] ❌ Message must have either text content or media")
		c.sendResponse(EventError, map[string]interface{}{
			"error": "message must have either text content or media",
		})
		return
	}

	// Persist message to DB
	log.Printf("[WEBSOCKET] 💾 Attempting to persist message to DB...")
	msg, err := c.Server.persistMessage(c.UserID, p)
	if err != nil {
		log.Printf("[WEBSOCKET] ❌ Failed to persist message: %v", err)
		log.Printf("[WEBSOCKET] ❌ Error details - UserID: %d, RoomID: %s, Message: %s", 
			c.UserID, p.RoomID, p.Message)
		c.sendResponse(EventError, map[string]interface{}{
			"error": "failed to save message",
		})
		return
	}

	log.Printf("[WEBSOCKET] ✅ Message persisted successfully: ID=%d, ConversationID=%d", 
		msg.ID, msg.ConversationID)

	// Broadcast to room
	c.Server.broadcastToRoom(p.RoomID, EventMessageCreated, msg, nil)
	log.Printf("[WEBSOCKET] 📤 Broadcasted message to room %s", p.RoomID)
}

// handleTypingStart handles typing.start event
func (c *Client) handleTypingStart(payload interface{}) {
	var p TypingPayload
	if err := unmarshalPayload(payload, &p); err != nil {
		return
	}

	if p.RoomID == "" {
		return
	}

	c.Server.broadcastToRoom(p.RoomID, EventTypingStart, c.UserID, c)
}

// handleTypingStop handles typing.stop event
func (c *Client) handleTypingStop(payload interface{}) {
	var p TypingPayload
	if err := unmarshalPayload(payload, &p); err != nil {
		return
	}

	if p.RoomID == "" {
		return
	}

	c.Server.broadcastToRoom(p.RoomID, EventTypingStop, c.UserID, c)
}

// handleDeliveryReceipt handles delivery.receipt event
func (c *Client) handleDeliveryReceipt(payload interface{}) {
	var p DeliveryReceiptPayload
	if err := unmarshalPayload(payload, &p); err != nil {
		return
	}

	if p.RoomID == "" || p.MessageID == "" || p.Status == "" {
		return
	}

	// Update delivery status in DB
	if err := c.Server.updateDeliveryStatus(p, c.UserID); err != nil {
		log.Printf("[WEBSOCKET] ❌ Failed to update delivery status: %v", err)
		return
	}

	// Broadcast delivery update to room
	c.Server.broadcastToRoom(p.RoomID, EventDeliveryUpdated, p, nil)
	log.Printf("[WEBSOCKET] User %d updated delivery status for message %s: %s", c.UserID, p.MessageID, p.Status)
}

// handleResume handles resume event
func (c *Client) handleResume(payload interface{}) {
	var p ResumePayload
	if err := unmarshalPayload(payload, &p); err != nil {
		c.sendResponse(EventError, map[string]interface{}{
			"error": "invalid resume payload",
		})
		return
	}

	if p.ResumeToken == "" {
		c.sendResponse(EventError, map[string]interface{}{
			"error": "resume_token is required",
		})
		return
	}

	session, ok := c.Server.resumeStore.Get(p.ResumeToken)
	if !ok {
		c.sendResponse(EventError, map[string]interface{}{
			"error": "invalid or expired resume_token",
		})
		return
	}

	if session.UserID != c.UserID {
		c.sendResponse(EventError, map[string]interface{}{
			"error": "resume_token does not match user",
		})
		return
	}

	// Fetch missed messages since last connection
	missedMessages, err := c.Server.getMissedMessages(c.UserID, session.LastMessageID)
	if err != nil {
		log.Printf("[WEBSOCKET] ❌ Failed to fetch missed messages: %v", err)
		c.sendResponse(EventError, map[string]interface{}{
			"error": "failed to fetch missed messages",
		})
		return
	}

	// Send missed messages
	for _, msg := range missedMessages {
		c.sendResponse(EventMessageCreated, msg)
	}

	c.sendResponse(EventServerAck, map[string]interface{}{
		"message":         "resume successful",
		"missed_count":    len(missedMessages),
		"last_message_id": session.LastMessageID,
	})
	log.Printf("[WEBSOCKET] User %d resumed connection, replayed %d messages", c.UserID, len(missedMessages))
}

// handlePing handles ping event
func (c *Client) handlePing(payload interface{}) {
	c.LastPing = time.Now()
	c.sendResponse(EventPong, map[string]interface{}{
		"client_timestamp": payload,
		"server_timestamp": time.Now().Unix(),
	})
}

// Helper function to unmarshal payload
func unmarshalPayload(payload interface{}, target interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// sendResponse sends a response to the client
func (c *Client) sendResponse(eventType string, payload interface{}) {
	response := Response{
		Type:    eventType,
		Payload: payload,
	}
	data, err := json.Marshal(response)
	if err != nil {
		log.Printf("[WEBSOCKET] ❌ Failed to marshal response: %v", err)
		return
	}

	select {
	case c.Send <- data:
	default:
		log.Printf("[WEBSOCKET] ⚠️ Client send buffer full, closing connection")
		close(c.Send)
	}
}

// Handler returns HTTP handler for WebSocket endpoint
func (s *WebSocketServer) Handler() http.Handler {
	return http.HandlerFunc(s.HandleWebSocket)
}

// Close closes all connections
func (s *WebSocketServer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for client := range s.clients {
		close(client.Send)
		client.Conn.Close()
	}
}

// BroadcastToUser broadcasts a message to a specific user
func (s *WebSocketServer) BroadcastToUser(userID uint, eventType string, data interface{}) error {
	roomID := getUserRoom(uint64(userID))
	s.broadcastToRoom(roomID, eventType, data, nil)
	return nil
}

// IsUserOnline checks if a user is online
func (s *WebSocketServer) IsUserOnline(userID uint) bool {
	return s.presence.IsOnline(uint64(userID))
}

// GetUserSockets returns socket IDs for a user
func (s *WebSocketServer) GetUserSockets(userID uint) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var socketIDs []string
	for client := range s.clients {
		if client.UserID == uint64(userID) {
			socketIDs = append(socketIDs, client.ID)
		}
	}
	return socketIDs
}

// Helper functions
func generateClientID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = letters[b[i]%byte(len(letters))]
	}
	return string(b)
}

// getUserRoom returns the personal room ID for a user
func getUserRoom(userID uint64) string {
	return fmt.Sprintf("user-%d", userID)
}

// persistMessage persists a message to the database
func (s *WebSocketServer) persistMessage(userID uint64, p SendMessagePayload) (*MessageCreatedPayload, error) {
	// Parse conversation ID from room_id (assuming format: "conv-{id}")
	var conversationID uint
	n, err := fmt.Sscanf(p.RoomID, "conv-%d", &conversationID)

	if err != nil || conversationID == 0 || n != 1 {
		log.Printf("[WEBSOCKET] ❌ Failed to parse room_id: %s, error: %v, parsed: %d", 
			p.RoomID, err, conversationID)
		return nil, fmt.Errorf("invalid room_id format: %s (expected format: conv-{id})", p.RoomID)
	}

	log.Printf("[WEBSOCKET] 📝 Parsed conversation ID: %d from room_id: %s", conversationID, p.RoomID)

	// Create message in DB
	ctx := context.Background()
	msg := &models.Message{
		ConversationID: conversationID,
		SenderID:       uint(userID),
		Content:        p.Message,
	}

	log.Printf("[WEBSOCKET] 💾 Creating message in DB: ConversationID=%d, SenderID=%d, Content=%s", 
		msg.ConversationID, msg.SenderID, msg.Content)

	if err := s.repo.CreateMessage(ctx, msg); err != nil {
		log.Printf("[WEBSOCKET] ❌ Database error creating message: %v", err)
		log.Printf("[WEBSOCKET] ❌ Message details: ConversationID=%d, SenderID=%d, Content length=%d", 
			msg.ConversationID, msg.SenderID, len(msg.Content))
		return nil, fmt.Errorf("database error: %w", err)
	}

	log.Printf("[WEBSOCKET] ✅ Message created successfully in DB: ID=%d, CreatedAt=%v", 
		msg.ID, msg.CreatedAt)

	// Handle media attachment if provided
	var mediaPayloads []MediaPayload
	if p.MediaID > 0 {
		// Get the media record
		media, err := s.repo.GetMediaByID(ctx, p.MediaID)
		if err != nil {
			log.Printf("[WEBSOCKET] ⚠️ Failed to get media by ID %d: %v", p.MediaID, err)
			// Continue without media rather than failing the entire message
		} else {
			// Verify the media is not already linked to another message (security check)
			if media.MessageID == nil || *media.MessageID == 0 || *media.MessageID == msg.ID {
				// Update media to link it to this message
				messageID := msg.ID
				if err := s.repo.UpdateMedia(ctx, p.MediaID, map[string]interface{}{
					"message_id": messageID,
				}); err != nil {
					log.Printf("[WEBSOCKET] ⚠️ Failed to link media to message: %v", err)
				} else {
					mediaPayloads = append(mediaPayloads, MediaPayload{
						ID:       media.ID,
						URL:      media.URL,
						MimeType: media.MimeType,
						Size:     media.Size,
						Filename: media.Filename,
					})
					log.Printf("[WEBSOCKET] ✅ Linked media %d to message %d", p.MediaID, msg.ID)
				}
			} else {
				log.Printf("[WEBSOCKET] ⚠️ Media %d is already linked to another message (%d)", p.MediaID, *media.MessageID)
			}
		}
	}

	return &MessageCreatedPayload{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		SenderID:       msg.SenderID,
		Content:        msg.Content,
		Media:          mediaPayloads,
		CreatedAt:      msg.CreatedAt,
		UpdatedAt:      msg.UpdatedAt,
		ClientID:       p.ClientID,
	}, nil
}

// updateDeliveryStatus updates message delivery/read status
func (s *WebSocketServer) updateDeliveryStatus(p DeliveryReceiptPayload, userID uint64) error {
	// Parse message ID
	var messageID uint
	_, err := fmt.Sscanf(p.MessageID, "%d", &messageID)

	if err != nil || messageID == 0 {
		return fmt.Errorf("invalid message_id: %s", p.MessageID)
	}

	// TODO: Implement delivery status tracking in your database schema
	log.Printf("[WEBSOCKET] Delivery receipt: message=%d, user=%d, status=%s", messageID, userID, p.Status)
	return nil
}

// getMissedMessages fetches messages after a given message ID for resume
func (s *WebSocketServer) getMissedMessages(userID uint64, lastMessageID uint64) ([]*MessageCreatedPayload, error) {
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
			log.Printf("[WEBSOCKET] Failed to get messages for conversation %d: %v", conv.ID, err)
			continue
		}

		// Filter messages after lastMessageID
		for _, msg := range messages {
			if uint64(msg.ID) > lastMessageID {
				// Get media for this message
				mediaList, err := s.repo.GetMediaByMessageID(ctx, msg.ID)
				var mediaPayloads []MediaPayload
				if err == nil {
					for _, media := range mediaList {
						mediaPayloads = append(mediaPayloads, MediaPayload{
							ID:       media.ID,
							URL:      media.URL,
							MimeType: media.MimeType,
							Size:     media.Size,
							Filename: media.Filename,
						})
					}
				}

				allMissedMessages = append(allMissedMessages, &MessageCreatedPayload{
					ID:             msg.ID,
					ConversationID: msg.ConversationID,
					SenderID:       msg.SenderID,
					Content:        msg.Content,
					Media:          mediaPayloads,
					CreatedAt:      msg.CreatedAt,
					UpdatedAt:      msg.UpdatedAt,
				})
			}
		}
	}

	return allMissedMessages, nil
}

