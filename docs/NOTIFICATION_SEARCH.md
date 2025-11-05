# Notification & Search Features

This document describes the notification and search functionality for the ChatterGo chat platform.

## Overview

The notification and search features provide:
- Real-time notifications via WebSocket
- Offline notification storage
- User notification preferences
- Full-text search for messages
- User and conversation search
- PostgreSQL-powered search capabilities

## Features Implemented

### ✅ **Notification System**
- **Real-time WebSocket notifications** for new messages
- **Offline notification storage** in database
- **User notification preferences** with granular controls
- **Do Not Disturb** mode with time-based scheduling
- **Notification types**: Message, Mention, Conversation, System
- **WebSocket integration** with existing socket server

### ✅ **Search Capabilities**
- **User search** by username/email
- **Message search** using PostgreSQL full-text search
- **Conversation search** by name
- **Combined search** across all types
- **Permission-based filtering** (users only see their accessible content)

## Database Schema

### Notifications Table
```sql
CREATE TABLE notifications (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    type VARCHAR(20) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    data JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'unread',
    conversation_id INTEGER,
    message_id INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
```

### Notification Preferences Table
```sql
CREATE TABLE notification_preferences (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL UNIQUE,
    message_notifications BOOLEAN DEFAULT TRUE,
    mention_notifications BOOLEAN DEFAULT TRUE,
    conversation_notifications BOOLEAN DEFAULT TRUE,
    system_notifications BOOLEAN DEFAULT TRUE,
    email_notifications BOOLEAN DEFAULT FALSE,
    push_notifications BOOLEAN DEFAULT TRUE,
    do_not_disturb BOOLEAN DEFAULT FALSE,
    do_not_disturb_start VARCHAR(5),
    do_not_disturb_end VARCHAR(5),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### Full-Text Search Index
```sql
CREATE INDEX idx_messages_content_fulltext ON messages 
USING gin(to_tsvector('english', content));
```

## API Endpoints

### Notification Endpoints

#### Get User Notifications
```http
GET /api/v1/notifications?limit=20&offset=0
Authorization: Bearer <token>
```

**Response:**
```json
{
  "notifications": [
    {
      "id": 1,
      "user_id": 2,
      "type": "message",
      "title": "New message from Alice",
      "message": "Hello there!",
      "data": "{\"conversation_id\":1,\"sender_name\":\"Alice\"}",
      "status": "unread",
      "conversation_id": 1,
      "message_id": 5,
      "created_at": "2025-11-05T16:30:00Z"
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

#### Get Unread Notifications
```http
GET /api/v1/notifications/unread
Authorization: Bearer <token>
```

#### Mark Notification as Read
```http
PATCH /api/v1/notifications/{id}/read
Authorization: Bearer <token>
```

#### Mark All Notifications as Read
```http
PATCH /api/v1/notifications/read-all
Authorization: Bearer <token>
```

#### Get Notification Preferences
```http
GET /api/v1/notifications/preferences
Authorization: Bearer <token>
```

**Response:**
```json
{
  "id": 1,
  "user_id": 1,
  "message_notifications": true,
  "mention_notifications": true,
  "conversation_notifications": true,
  "system_notifications": true,
  "email_notifications": false,
  "push_notifications": true,
  "do_not_disturb": false,
  "do_not_disturb_start": null,
  "do_not_disturb_end": null
}
```

#### Update Notification Preferences
```http
PATCH /api/v1/notifications/preferences
Authorization: Bearer <token>
Content-Type: application/json

{
  "message_notifications": false,
  "do_not_disturb": true,
  "do_not_disturb_start": "22:00",
  "do_not_disturb_end": "06:00"
}
```

### Search Endpoints

#### Universal Search
```http
GET /api/v1/search?q=keyword&type=all&limit=20&offset=0
Authorization: Bearer <token>
```

**Response:**
```json
{
  "query": "keyword",
  "type": "all",
  "results": {
    "users": [
      {
        "id": 1,
        "username": "alice",
        "email": "alice@example.com"
      }
    ],
    "messages": [
      {
        "id": 5,
        "conversation_id": 1,
        "sender_id": 1,
        "content": "This message contains the keyword",
        "created_at": "2025-11-05T16:30:00Z",
        "sender": {
          "id": 1,
          "username": "alice"
        }
      }
    ],
    "conversations": [
      {
        "id": 1,
        "name": "Keyword Discussion",
        "type": "group"
      }
    ]
  },
  "limit": 20,
  "offset": 0
}
```

#### Search Users
```http
GET /api/v1/search/users?q=alice&limit=10&offset=0
Authorization: Bearer <token>
```

#### Search Messages
```http
GET /api/v1/search/messages?q=hello&conversation_id=1&limit=10&offset=0
Authorization: Bearer <token>
```

**Parameters:**
- `q`: Search query (required, min 2 characters)
- `type`: Search type (`all`, `users`, `messages`, `conversations`)
- `conversation_id`: Filter messages by conversation (optional)
- `limit`: Results per page (default: 20)
- `offset`: Pagination offset (default: 0)

## WebSocket Integration

### WebSocket Events

#### Notification Events
```javascript
// Notification received
socket.on('notification_received', (notification) => {
  console.log('New notification:', notification);
  updateNotificationUI(notification);
});

// Notification read
socket.on('notification_read', (data) => {
  console.log('Notification read:', data);
  updateNotificationStatus(data.notification_id, 'read');
});

// Unread count update
socket.on('notification_count', (data) => {
  console.log('Unread count:', data.unread_count);
  updateNotificationBadge(data.unread_count);
});
```

#### Client-to-Server Events
```javascript
// Acknowledge notification (mark as read)
socket.emit('notification_ack', {
  notification_id: 123
});

// Get current unread count
socket.emit('get_notification_count');

// Mark all notifications as read
socket.emit('mark_all_read');
```

### WebSocket Integration Setup

```go
// In your main server setup
func main() {
    // ... existing setup ...
    
    // Create WebSocket server
    wsServer, err := websocket.NewSocketServer(repo)
    if err != nil {
        log.Fatal("Failed to create WebSocket server:", err)
    }
    
    // Create service server
    serviceServer, err := services.NewServer(cfg, repo)
    if err != nil {
        log.Fatal("Failed to create service server:", err)
    }
    
    // Setup WebSocket notification integration
    serviceServer.SetupWebSocketNotifications(wsServer)
    
    // Setup notification WebSocket events
    services.SetupNotificationWebSocketEvents(wsServer, serviceServer)
    
    // ... start servers ...
}
```

## Usage Examples

### Frontend Integration

#### Real-time Notifications
```javascript
class NotificationManager {
  constructor(socket) {
    this.socket = socket;
    this.setupEventListeners();
  }
  
  setupEventListeners() {
    this.socket.on('notification_received', (notification) => {
      this.showNotification(notification);
      this.updateUnreadCount();
    });
    
    this.socket.on('notification_count', (data) => {
      this.updateBadge(data.unread_count);
    });
  }
  
  markAsRead(notificationId) {
    this.socket.emit('notification_ack', {
      notification_id: notificationId
    });
  }
  
  markAllAsRead() {
    this.socket.emit('mark_all_read');
  }
}
```

#### Search Implementation
```javascript
class SearchManager {
  async search(query, type = 'all') {
    const response = await fetch(`/api/v1/search?q=${encodeURIComponent(query)}&type=${type}`, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    });
    
    return await response.json();
  }
  
  async searchUsers(query) {
    const response = await fetch(`/api/v1/search/users?q=${encodeURIComponent(query)}`, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    });
    
    return await response.json();
  }
}
```

### Backend Integration

#### Creating Notifications
```go
// When a new message is sent
func (s *MessageService) SendMessage(conversationID uint, senderID uint, content string) error {
    // Create message
    message := &models.Message{
        ConversationID: conversationID,
        SenderID: senderID,
        Content: content,
    }
    
    err := s.repo.CreateMessage(ctx, message)
    if err != nil {
        return err
    }
    
    // Create notifications for conversation members
    err = s.notificationService.CreateMessageNotification(ctx, message, 0)
    if err != nil {
        log.Printf("Failed to create notifications: %v", err)
        // Don't fail message creation if notifications fail
    }
    
    return nil
}
```

#### Custom Notification Types
```go
// Create a system notification
func (s *NotificationService) CreateSystemNotification(userID uint, title, message string) error {
    notification := &models.Notification{
        UserID:  userID,
        Type:    models.NotificationTypeSystem,
        Title:   title,
        Message: message,
        Status:  models.NotificationStatusUnread,
    }
    
    err := s.repo.CreateNotification(ctx, notification)
    if err != nil {
        return err
    }
    
    // Send via WebSocket if user is online
    s.sendWebSocketNotification(userID, notification)
    
    return nil
}
```

## Configuration

### Notification Settings
```go
type NotificationConfig struct {
    EnableWebSocket     bool          `json:"enable_websocket"`
    EnableEmail         bool          `json:"enable_email"`
    BatchSize          int           `json:"batch_size"`
    RetentionDays      int           `json:"retention_days"`
    DefaultPreferences NotificationPreference `json:"default_preferences"`
}
```

### Search Settings
```go
type SearchConfig struct {
    MaxResults      int    `json:"max_results"`
    MinQueryLength  int    `json:"min_query_length"`
    EnableFullText  bool   `json:"enable_full_text"`
    SearchLanguage  string `json:"search_language"`
}
```

## Performance Considerations

### Database Optimization
- **Indexes**: Proper indexing on user_id, status, type, and created_at
- **Full-text search**: GIN index on message content
- **Pagination**: Always use limit/offset for large result sets
- **Cleanup**: Regular cleanup of old notifications

### WebSocket Optimization
- **Connection pooling**: Reuse WebSocket connections
- **Message batching**: Batch multiple notifications when possible
- **Presence tracking**: Only send to online users
- **Fallback**: Store notifications for offline users

### Search Optimization
- **Query caching**: Cache frequent search queries
- **Result limiting**: Limit search results per category
- **Permission filtering**: Filter results at database level
- **Debouncing**: Debounce search requests on frontend

## Security Considerations

### Notification Security
- **Authorization**: Users can only see their own notifications
- **Data sanitization**: Sanitize notification content
- **Rate limiting**: Prevent notification spam
- **Privacy**: Respect user privacy settings

### Search Security
- **Permission checking**: Users can only search accessible content
- **Query sanitization**: Prevent SQL injection in search queries
- **Result filtering**: Filter sensitive information from results
- **Rate limiting**: Prevent search abuse

## Testing

### Running Tests
```bash
# Run notification tests
go test ./internal/services -v -run TestNotification

# Run search tests
go test ./internal/repositories/postgres -v -run TestSearch

# Run WebSocket integration tests
go test ./internal/services -v -run TestWebSocket
```

### Test Coverage
- ✅ Notification creation and delivery
- ✅ Notification preferences management
- ✅ WebSocket event handling
- ✅ Search functionality across all types
- ✅ Permission-based filtering
- ✅ Error handling and edge cases

## Migration

### Running Migration
```bash
# Apply notification and search migration
go run ./cmd/migrator
```

### Rollback
```bash
# Rollback notification migration
go run ./cmd/migrator -down -steps 1
```

## Monitoring and Metrics

### Key Metrics to Track
- Notification delivery success rate
- WebSocket connection count
- Search query performance
- Notification read rates
- User engagement with notifications

### Logging
```go
// Notification logging
log.Printf("Notification created: user=%d, type=%s, id=%d", userID, notificationType, notificationID)
log.Printf("WebSocket notification sent: user=%d, online=%t", userID, isOnline)

// Search logging
log.Printf("Search query: user=%d, query=%s, type=%s, results=%d", userID, query, searchType, resultCount)
```

## Future Enhancements

### Notification Enhancements
1. **Email notifications** for offline users
2. **Push notifications** via FCM/APNS
3. **Notification templates** for customization
4. **Bulk operations** for admin notifications
5. **Notification scheduling** for delayed delivery

### Search Enhancements
1. **Advanced filters** (date range, file type, etc.)
2. **Search suggestions** and autocomplete
3. **Search analytics** and popular queries
4. **Elasticsearch integration** for advanced search
5. **Search result ranking** and relevance scoring

The notification and search features are now fully implemented and ready for production use! 🎉
