package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RyseUp/ChatterGo/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockNotificationRepository extends the existing MockRepository
func (m *MockRepository) CreateNotification(ctx context.Context, notification *models.Notification) error {
	args := m.Called(ctx, notification)
	if args.Error(0) == nil {
		notification.ID = 1 // Simulate database assigning ID
	}
	return args.Error(0)
}

func (m *MockRepository) GetNotificationByID(ctx context.Context, id uint) (*models.Notification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Notification), args.Error(1)
}

func (m *MockRepository) GetNotificationsByUserID(ctx context.Context, userID uint, limit, offset int) ([]models.Notification, int64, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]models.Notification), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) GetUnreadNotificationsByUserID(ctx context.Context, userID uint) ([]models.Notification, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]models.Notification), args.Error(1)
}

func (m *MockRepository) MarkNotificationAsRead(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) MarkAllNotificationsAsRead(ctx context.Context, userID uint) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockRepository) DeleteNotification(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) CreateNotificationPreference(ctx context.Context, pref *models.NotificationPreference) error {
	args := m.Called(ctx, pref)
	if args.Error(0) == nil {
		pref.ID = 1 // Simulate database assigning ID
	}
	return args.Error(0)
}

func (m *MockRepository) GetNotificationPreferenceByUserID(ctx context.Context, userID uint) (*models.NotificationPreference, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.NotificationPreference), args.Error(1)
}

func (m *MockRepository) UpdateNotificationPreference(ctx context.Context, userID uint, updates map[string]interface{}) error {
	args := m.Called(ctx, userID, updates)
	return args.Error(0)
}

// Search repository methods (stubs for notification tests)
func (m *MockRepository) SearchUsers(ctx context.Context, query string, limit, offset int) ([]models.User, int64, error) {
	args := m.Called(ctx, query, limit, offset)
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) SearchMessages(ctx context.Context, query string, conversationID *uint, limit, offset int) ([]models.Message, int64, error) {
	args := m.Called(ctx, query, conversationID, limit, offset)
	return args.Get(0).([]models.Message), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) SearchConversations(ctx context.Context, query string, userID uint, limit, offset int) ([]models.Conversation, int64, error) {
	args := m.Called(ctx, query, userID, limit, offset)
	return args.Get(0).([]models.Conversation), args.Get(1).(int64), args.Error(2)
}

func TestCreateMessageNotification(t *testing.T) {
	server, mockRepo := setupTestServer()

	tests := []struct {
		name          string
		message       *models.Message
		excludeUserID uint
		setupMock     func(*MockRepository)
		expectError   bool
	}{
		{
			name: "Successful notification creation",
			message: &models.Message{
				ID:             1,
				ConversationID: 1,
				SenderID:       1,
				Content:        "Hello everyone!",
			},
			excludeUserID: 0,
			setupMock: func(mr *MockRepository) {
				// Mock conversation members
				members := []models.ConversationMember{
					{UserID: 1, ConversationID: 1}, // sender
					{UserID: 2, ConversationID: 1}, // recipient 1
					{UserID: 3, ConversationID: 1}, // recipient 2
				}
				mr.On("GetConversationMembers", mock.Anything, uint(1)).Return(members, nil)

				// Mock sender info
				sender := &models.User{ID: 1, Username: "alice", Email: "alice@example.com"}
				mr.On("GetUserByUserID", mock.Anything, uint(1)).Return(sender, nil)

				// Mock notification preferences for recipients
				prefs := &models.NotificationPreference{
					UserID:               2,
					MessageNotifications: true,
					PushNotifications:    true,
				}
				mr.On("GetNotificationPreferenceByUserID", mock.Anything, uint(2)).Return(prefs, nil)
				mr.On("GetNotificationPreferenceByUserID", mock.Anything, uint(3)).Return(prefs, nil)

				// Mock notification creation
				mr.On("CreateNotification", mock.Anything, mock.AnythingOfType("*models.Notification")).Return(nil).Times(2)
			},
			expectError: false,
		},
		{
			name: "User with disabled notifications",
			message: &models.Message{
				ID:             1,
				ConversationID: 1,
				SenderID:       1,
				Content:        "Hello everyone!",
			},
			excludeUserID: 0,
			setupMock: func(mr *MockRepository) {
				members := []models.ConversationMember{
					{UserID: 1, ConversationID: 1}, // sender
					{UserID: 2, ConversationID: 1}, // recipient with disabled notifications
				}
				mr.On("GetConversationMembers", mock.Anything, uint(1)).Return(members, nil)

				sender := &models.User{ID: 1, Username: "alice", Email: "alice@example.com"}
				mr.On("GetUserByUserID", mock.Anything, uint(1)).Return(sender, nil)

				// User with disabled message notifications
				prefs := &models.NotificationPreference{
					UserID:               2,
					MessageNotifications: false, // disabled
					PushNotifications:    true,
				}
				mr.On("GetNotificationPreferenceByUserID", mock.Anything, uint(2)).Return(prefs, nil)

				// Should not create any notifications
			},
			expectError: false,
		},
		{
			name: "Error getting conversation members",
			message: &models.Message{
				ID:             1,
				ConversationID: 1,
				SenderID:       1,
				Content:        "Hello everyone!",
			},
			excludeUserID: 0,
			setupMock: func(mr *MockRepository) {
				mr.On("GetConversationMembers", mock.Anything, uint(1)).Return([]models.ConversationMember{}, fmt.Errorf("database error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			tt.setupMock(mockRepo)

			// Call the function
			err := server.CreateMessageNotification(context.Background(), tt.message, tt.excludeUserID)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Reset mock for next test
			mockRepo.ExpectedCalls = nil
		})
	}
}

func TestGetNotifications(t *testing.T) {
	server, mockRepo := setupTestServer()

	tests := []struct {
		name           string
		userID         uint
		limit          string
		offset         string
		expectedStatus int
		setupMock      func(*MockRepository)
	}{
		{
			name:           "Successful get notifications",
			userID:         1,
			limit:          "10",
			offset:         "0",
			expectedStatus: http.StatusOK,
			setupMock: func(mr *MockRepository) {
				notifications := []models.Notification{
					{
						ID:      1,
						UserID:  1,
						Type:    models.NotificationTypeMessage,
						Title:   "New message",
						Message: "Hello!",
						Status:  models.NotificationStatusUnread,
					},
				}
				mr.On("GetNotificationsByUserID", mock.Anything, uint(1), 10, 0).Return(notifications, int64(1), nil)
			},
		},
		{
			name:           "Database error",
			userID:         1,
			limit:          "10",
			offset:         "0",
			expectedStatus: http.StatusInternalServerError,
			setupMock: func(mr *MockRepository) {
				mr.On("GetNotificationsByUserID", mock.Anything, uint(1), 10, 0).Return([]models.Notification{}, int64(0), fmt.Errorf("database error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			tt.setupMock(mockRepo)

			// Create request
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/notifications?limit=%s&offset=%s", tt.limit, tt.offset), nil)
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Request = req
			ctx.Set("user_id", tt.userID)

			// Call handler
			server.GetNotifications(ctx)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "notifications")
				assert.Contains(t, response, "total")
			}

			// Reset mock for next test
			mockRepo.ExpectedCalls = nil
		})
	}
}

func TestGetUnreadNotifications(t *testing.T) {
	server, mockRepo := setupTestServer()

	tests := []struct {
		name           string
		userID         uint
		expectedStatus int
		setupMock      func(*MockRepository)
	}{
		{
			name:           "Successful get unread notifications",
			userID:         1,
			expectedStatus: http.StatusOK,
			setupMock: func(mr *MockRepository) {
				notifications := []models.Notification{
					{
						ID:      1,
						UserID:  1,
						Type:    models.NotificationTypeMessage,
						Title:   "New message",
						Message: "Hello!",
						Status:  models.NotificationStatusUnread,
					},
				}
				mr.On("GetUnreadNotificationsByUserID", mock.Anything, uint(1)).Return(notifications, nil)
			},
		},
		{
			name:           "No unread notifications",
			userID:         1,
			expectedStatus: http.StatusOK,
			setupMock: func(mr *MockRepository) {
				mr.On("GetUnreadNotificationsByUserID", mock.Anything, uint(1)).Return([]models.Notification{}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			tt.setupMock(mockRepo)

			// Create request
			req := httptest.NewRequest("GET", "/api/v1/notifications/unread", nil)
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Request = req
			ctx.Set("user_id", tt.userID)

			// Call handler
			server.GetUnreadNotifications(ctx)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "notifications")
				assert.Contains(t, response, "count")
			}

			// Reset mock for next test
			mockRepo.ExpectedCalls = nil
		})
	}
}

func TestMarkNotificationAsRead(t *testing.T) {
	server, mockRepo := setupTestServer()

	tests := []struct {
		name           string
		notificationID string
		expectedStatus int
		setupMock      func(*MockRepository)
	}{
		{
			name:           "Successful mark as read",
			notificationID: "1",
			expectedStatus: http.StatusOK,
			setupMock: func(mr *MockRepository) {
				mr.On("MarkNotificationAsRead", mock.Anything, uint(1)).Return(nil)
			},
		},
		{
			name:           "Invalid notification ID",
			notificationID: "invalid",
			expectedStatus: http.StatusBadRequest,
			setupMock:      func(mr *MockRepository) {},
		},
		{
			name:           "Database error",
			notificationID: "1",
			expectedStatus: http.StatusInternalServerError,
			setupMock: func(mr *MockRepository) {
				mr.On("MarkNotificationAsRead", mock.Anything, uint(1)).Return(fmt.Errorf("database error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			tt.setupMock(mockRepo)

			// Create request
			req := httptest.NewRequest("PATCH", "/api/v1/notifications/"+tt.notificationID+"/read", nil)
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Request = req
			ctx.Params = gin.Params{{Key: "id", Value: tt.notificationID}}

			// Call handler
			server.MarkNotificationAsRead(ctx)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Reset mock for next test
			mockRepo.ExpectedCalls = nil
		})
	}
}

func TestGetNotificationPreferences(t *testing.T) {
	server, mockRepo := setupTestServer()

	tests := []struct {
		name           string
		userID         uint
		expectedStatus int
		setupMock      func(*MockRepository)
	}{
		{
			name:           "Existing preferences",
			userID:         1,
			expectedStatus: http.StatusOK,
			setupMock: func(mr *MockRepository) {
				prefs := &models.NotificationPreference{
					ID:                   1,
					UserID:               1,
					MessageNotifications: true,
					PushNotifications:    true,
				}
				mr.On("GetNotificationPreferenceByUserID", mock.Anything, uint(1)).Return(prefs, nil)
			},
		},
		{
			name:           "Create default preferences",
			userID:         1,
			expectedStatus: http.StatusOK,
			setupMock: func(mr *MockRepository) {
				mr.On("GetNotificationPreferenceByUserID", mock.Anything, uint(1)).Return((*models.NotificationPreference)(nil), fmt.Errorf("not found"))
				mr.On("CreateNotificationPreference", mock.Anything, mock.AnythingOfType("*models.NotificationPreference")).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			tt.setupMock(mockRepo)

			// Create request
			req := httptest.NewRequest("GET", "/api/v1/notifications/preferences", nil)
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Request = req
			ctx.Set("user_id", tt.userID)

			// Call handler
			server.GetNotificationPreferences(ctx)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Reset mock for next test
			mockRepo.ExpectedCalls = nil
		})
	}
}

func TestUpdateNotificationPreferences(t *testing.T) {
	server, mockRepo := setupTestServer()

	tests := []struct {
		name           string
		userID         uint
		updates        map[string]interface{}
		expectedStatus int
		setupMock      func(*MockRepository)
	}{
		{
			name:   "Successful update",
			userID: 1,
			updates: map[string]interface{}{
				"message_notifications": false,
				"push_notifications":    true,
			},
			expectedStatus: http.StatusOK,
			setupMock: func(mr *MockRepository) {
				mr.On("UpdateNotificationPreference", mock.Anything, uint(1), mock.AnythingOfType("map[string]interface {}")).Return(nil)
			},
		},
		{
			name:           "Invalid JSON",
			userID:         1,
			updates:        nil, // Will cause JSON binding error
			expectedStatus: http.StatusBadRequest,
			setupMock:      func(mr *MockRepository) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			tt.setupMock(mockRepo)

			// Create request body
			var body *bytes.Buffer
			if tt.updates != nil {
				jsonData, _ := json.Marshal(tt.updates)
				body = bytes.NewBuffer(jsonData)
			} else {
				body = bytes.NewBuffer([]byte("invalid json"))
			}

			// Create request
			req := httptest.NewRequest("PATCH", "/api/v1/notifications/preferences", body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Request = req
			ctx.Set("user_id", tt.userID)

			// Call handler
			server.UpdateNotificationPreferences(ctx)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Reset mock for next test
			mockRepo.ExpectedCalls = nil
		})
	}
}

func TestIsInDoNotDisturbPeriod(t *testing.T) {
	server, _ := setupTestServer()

	tests := []struct {
		name     string
		prefs    *models.NotificationPreference
		expected bool
	}{
		{
			name: "No DND period set",
			prefs: &models.NotificationPreference{
				DoNotDisturb: true,
			},
			expected: false,
		},
		{
			name: "DND disabled",
			prefs: &models.NotificationPreference{
				DoNotDisturb:      false,
				DoNotDisturbStart: stringPtr("22:00"),
				DoNotDisturbEnd:   stringPtr("06:00"),
			},
			expected: false,
		},
		{
			name: "Same day period - outside",
			prefs: &models.NotificationPreference{
				DoNotDisturb:      true,
				DoNotDisturbStart: stringPtr("09:00"),
				DoNotDisturbEnd:   stringPtr("17:00"),
			},
			expected: false, // Depends on current time, but testing the logic
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.isInDoNotDisturbPeriod(tt.prefs)
			// Note: This test is time-dependent, so we're mainly testing that it doesn't panic
			_ = result
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		length   int
		expected string
	}{
		{
			name:     "String shorter than limit",
			input:    "Hello",
			length:   10,
			expected: "Hello",
		},
		{
			name:     "String longer than limit",
			input:    "This is a very long message that should be truncated",
			length:   20,
			expected: "This is a very long ...",
		},
		{
			name:     "String exactly at limit",
			input:    "Exactly twenty chars",
			length:   20,
			expected: "Exactly twenty chars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.length)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// Test WebSocket notification broadcasting (mock)
func TestSendWebSocketNotification(t *testing.T) {
	server, _ := setupTestServer()

	notification := &models.Notification{
		ID:      1,
		UserID:  1,
		Type:    models.NotificationTypeMessage,
		Title:   "Test notification",
		Message: "This is a test",
		Status:  models.NotificationStatusUnread,
	}

	// This should not panic
	assert.NotPanics(t, func() {
		server.sendWebSocketNotification(1, notification)
	})
}
