package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RyseUp/ChatterGo/config"
	"github.com/RyseUp/ChatterGo/internal/models"
	"github.com/RyseUp/ChatterGo/internal/repositories"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Ping() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockRepository) Transaction(ctx context.Context, txFunc func(repositories.Repository) error) error {
	args := m.Called(ctx, txFunc)
	return args.Error(0)
}

// User methods (stubs)
func (m *MockRepository) CreateUser(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockRepository) GetUserByUserID(ctx context.Context, id uint) (*models.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockRepository) GetUserByUserEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockRepository) UpdateUser(ctx context.Context, id uint, updates map[string]interface{}) error {
	args := m.Called(ctx, id, updates)
	return args.Error(0)
}

func (m *MockRepository) UpdateUserRefreshToken(ctx context.Context, id uint, refreshToken *string) error {
	args := m.Called(ctx, id, refreshToken)
	return args.Error(0)
}

func (m *MockRepository) UpdateUserLastLogin(ctx context.Context, id uint, lastLogin time.Time) error {
	args := m.Called(ctx, id, lastLogin)
	return args.Error(0)
}

// Conversation methods (stubs)
func (m *MockRepository) CreateConversation(ctx context.Context, conversation *models.Conversation) error {
	args := m.Called(ctx, conversation)
	return args.Error(0)
}

func (m *MockRepository) GetConversationByID(ctx context.Context, id uint) (*models.Conversation, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Conversation), args.Error(1)
}

func (m *MockRepository) GetConversationsByUserID(ctx context.Context, userID uint, limit, offset int) ([]models.Conversation, int64, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]models.Conversation), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) GetDirectConversationBetweenUsers(ctx context.Context, userID1, userID2 uint) (*models.Conversation, error) {
	args := m.Called(ctx, userID1, userID2)
	return args.Get(0).(*models.Conversation), args.Error(1)
}

func (m *MockRepository) UpdateConversation(ctx context.Context, id uint, updates map[string]interface{}) error {
	args := m.Called(ctx, id, updates)
	return args.Error(0)
}

func (m *MockRepository) DeleteConversation(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) AddConversationMembers(ctx context.Context, members []*models.ConversationMember) error {
	args := m.Called(ctx, members)
	return args.Error(0)
}

func (m *MockRepository) RemoveConversationMember(ctx context.Context, conversationID, userID uint) error {
	args := m.Called(ctx, conversationID, userID)
	return args.Error(0)
}

func (m *MockRepository) GetConversationMembers(ctx context.Context, conversationID uint) ([]models.ConversationMember, error) {
	args := m.Called(ctx, conversationID)
	return args.Get(0).([]models.ConversationMember), args.Error(1)
}

func (m *MockRepository) IsConversationMember(ctx context.Context, conversationID, userID uint) (bool, error) {
	args := m.Called(ctx, conversationID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) GetMemberRole(ctx context.Context, conversationID, userID uint) (*models.MemberRole, error) {
	args := m.Called(ctx, conversationID, userID)
	return args.Get(0).(*models.MemberRole), args.Error(1)
}

// Message methods (stubs)
func (m *MockRepository) CreateMessage(ctx context.Context, message *models.Message) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockRepository) GetMessageByID(ctx context.Context, id uint) (*models.Message, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Message), args.Error(1)
}

func (m *MockRepository) GetMessagesByConversationID(ctx context.Context, conversationID uint, limit, offset int) ([]models.Message, int64, error) {
	args := m.Called(ctx, conversationID, limit, offset)
	return args.Get(0).([]models.Message), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) UpdateMessage(ctx context.Context, id uint, updates map[string]interface{}) error {
	args := m.Called(ctx, id, updates)
	return args.Error(0)
}

func (m *MockRepository) DeleteMessage(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Media methods
func (m *MockRepository) CreateMedia(ctx context.Context, media *models.Media) error {
	args := m.Called(ctx, media)
	if args.Error(0) == nil {
		media.ID = 1 // Simulate database assigning ID
	}
	return args.Error(0)
}

func (m *MockRepository) GetMediaByID(ctx context.Context, id uint) (*models.Media, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Media), args.Error(1)
}

func (m *MockRepository) GetMediaByMessageID(ctx context.Context, messageID uint) ([]models.Media, error) {
	args := m.Called(ctx, messageID)
	return args.Get(0).([]models.Media), args.Error(1)
}

func (m *MockRepository) UpdateMedia(ctx context.Context, id uint, updates map[string]interface{}) error {
	args := m.Called(ctx, id, updates)
	return args.Error(0)
}

func (m *MockRepository) DeleteMedia(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) DeleteMediaByMessageID(ctx context.Context, messageID uint) error {
	args := m.Called(ctx, messageID)
	return args.Error(0)
}

func setupTestServer() (*ServiceServer, *MockRepository) {
	gin.SetMode(gin.TestMode)
	
	mockRepo := &MockRepository{}
	cfg := &config.Config{
		JWT: config.JWTConfig{Secret: "test-secret"},
		Server: config.ServerConfig{Port: 8080},
	}
	
	server := &ServiceServer{
		cfg: cfg,
		r:   mockRepo,
	}
	
	return server, mockRepo
}

func TestPresignUpload(t *testing.T) {
	server, _ := setupTestServer()
	
	tests := []struct {
		name           string
		request        PresignRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name: "Valid presign request",
			request: PresignRequest{
				Filename: "test.jpg",
				MimeType: "image/jpeg",
				Size:     1024 * 1024, // 1MB
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "File too large",
			request: PresignRequest{
				Filename: "large.jpg",
				MimeType: "image/jpeg",
				Size:     20 * 1024 * 1024, // 20MB (exceeds 10MB limit)
			},
			expectedStatus: http.StatusRequestEntityTooLarge,
			expectError:    true,
		},
		{
			name: "Unsupported MIME type",
			request: PresignRequest{
				Filename: "test.exe",
				MimeType: "application/x-executable",
				Size:     1024,
			},
			expectedStatus: http.StatusUnsupportedMediaType,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			reqBody, _ := json.Marshal(tt.request)
			req := httptest.NewRequest("POST", "/api/v1/media/presign", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Request = req
			
			// Call handler
			server.PresignUpload(ctx)
			
			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if !tt.expectError {
				var response PresignResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.UploadURL)
				assert.NotEmpty(t, response.MediaID)
				assert.Greater(t, response.ExpiresAt, int64(0))
			}
		})
	}
}

func TestUploadFile(t *testing.T) {
	server, mockRepo := setupTestServer()
	
	// Create test directory
	testDir := "./test_uploads"
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)
	
	// Override config for testing
	DefaultMediaConfig.LocalStoragePath = testDir
	
	tests := []struct {
		name           string
		filename       string
		content        string
		mimeType       string
		messageID      string
		expectedStatus int
		setupMock      func(*MockRepository)
	}{
		{
			name:           "Valid image upload",
			filename:       "test.jpg",
			content:        "\xFF\xD8\xFF\xE0\x00\x10JFIF\x00\x01\x01\x01\x00H\x00H\x00\x00\xFF\xDB\x00C\x00\x08\x06\x06\x07\x06\x05\x08\x07\x07\x07\t\t\x08\n\x0C\x14\r\x0C\x0B\x0B\x0C\x19\x12\x13\x0F\x14\x1D\x1A\x1F\x1E\x1D\x1A\x1C\x1C $.' \",#\x1C\x1C(7),01444\x1F'9=82<.342\xFF\xC0\x00\x11\x08\x00\x01\x00\x01\x01\x01\x11\x00\x02\x11\x01\x03\x11\x01\xFF\xC4\x00\x14\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x08\xFF\xC4\x00\x14\x10\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xFF\xDA\x00\x0C\x03\x01\x00\x02\x11\x03\x11\x00\x3F\x00\xAA\xFF\xD9", // Minimal valid JPEG
			mimeType:       "image/jpeg",
			messageID:      "1",
			expectedStatus: http.StatusCreated,
			setupMock: func(mr *MockRepository) {
				// Mock message exists
				message := &models.Message{ID: 1}
				mr.On("GetMessageByID", mock.Anything, uint(1)).Return(message, nil)
				mr.On("CreateMedia", mock.Anything, mock.AnythingOfType("*models.Media")).Return(nil)
			},
		},
		{
			name:           "Database error",
			filename:       "test2.jpg",
			content:        "\xFF\xD8\xFF\xE0\x00\x10JFIF\x00\x01\x01\x01\x00H\x00H\x00\x00\xFF\xDB\x00C\x00\x08\x06\x06\x07\x06\x05\x08\x07\x07\x07\t\t\x08\n\x0C\x14\r\x0C\x0B\x0B\x0C\x19\x12\x13\x0F\x14\x1D\x1A\x1F\x1E\x1D\x1A\x1C\x1C $.' \",#\x1C\x1C(7),01444\x1F'9=82<.342\xFF\xC0\x00\x11\x08\x00\x01\x00\x01\x01\x01\x11\x00\x02\x11\x01\x03\x11\x01\xFF\xC4\x00\x14\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x08\xFF\xC4\x00\x14\x10\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xFF\xDA\x00\x0C\x03\x01\x00\x02\x11\x03\x11\x00\x3F\x00\xAA\xFF\xD9", // Minimal valid JPEG
			mimeType:       "image/jpeg",
			messageID:      "1",
			expectedStatus: http.StatusInternalServerError,
			setupMock: func(mr *MockRepository) {
				// Mock message exists
				message := &models.Message{ID: 1}
				mr.On("GetMessageByID", mock.Anything, uint(1)).Return(message, nil)
				mr.On("CreateMedia", mock.Anything, mock.AnythingOfType("*models.Media")).Return(fmt.Errorf("database error"))
			},
		},
		{
			name:           "Missing message_id",
			filename:       "test3.jpg",
			content:        "\xFF\xD8\xFF\xE0\x00\x10JFIF\x00\x01\x01\x01\x00H\x00H\x00\x00\xFF\xDB\x00C\x00\x08\x06\x06\x07\x06\x05\x08\x07\x07\x07\t\t\x08\n\x0C\x14\r\x0C\x0B\x0B\x0C\x19\x12\x13\x0F\x14\x1D\x1A\x1F\x1E\x1D\x1A\x1C\x1C $.' \",#\x1C\x1C(7),01444\x1F'9=82<.342\xFF\xC0\x00\x11\x08\x00\x01\x00\x01\x01\x01\x11\x00\x02\x11\x01\x03\x11\x01\xFF\xC4\x00\x14\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x08\xFF\xC4\x00\x14\x10\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xFF\xDA\x00\x0C\x03\x01\x00\x02\x11\x03\x11\x00\x3F\x00\xAA\xFF\xD9", // Minimal valid JPEG
			mimeType:       "image/jpeg",
			messageID:      "",
			expectedStatus: http.StatusBadRequest,
			setupMock:      func(mr *MockRepository) {},
		},
		{
			name:           "Invalid message_id format",
			filename:       "test4.jpg",
			content:        "\xFF\xD8\xFF\xE0\x00\x10JFIF\x00\x01\x01\x01\x00H\x00H\x00\x00\xFF\xDB\x00C\x00\x08\x06\x06\x07\x06\x05\x08\x07\x07\x07\t\t\x08\n\x0C\x14\r\x0C\x0B\x0B\x0C\x19\x12\x13\x0F\x14\x1D\x1A\x1F\x1E\x1D\x1A\x1C\x1C $.' \",#\x1C\x1C(7),01444\x1F'9=82<.342\xFF\xC0\x00\x11\x08\x00\x01\x00\x01\x01\x01\x11\x00\x02\x11\x01\x03\x11\x01\xFF\xC4\x00\x14\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x08\xFF\xC4\x00\x14\x10\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xFF\xDA\x00\x0C\x03\x01\x00\x02\x11\x03\x11\x00\x3F\x00\xAA\xFF\xD9", // Minimal valid JPEG
			mimeType:       "image/jpeg",
			messageID:      "invalid",
			expectedStatus: http.StatusBadRequest,
			setupMock:      func(mr *MockRepository) {},
		},
		{
			name:           "Message not found",
			filename:       "test5.jpg",
			content:        "\xFF\xD8\xFF\xE0\x00\x10JFIF\x00\x01\x01\x01\x00H\x00H\x00\x00\xFF\xDB\x00C\x00\x08\x06\x06\x07\x06\x05\x08\x07\x07\x07\t\t\x08\n\x0C\x14\r\x0C\x0B\x0B\x0C\x19\x12\x13\x0F\x14\x1D\x1A\x1F\x1E\x1D\x1A\x1C\x1C $.' \",#\x1C\x1C(7),01444\x1F'9=82<.342\xFF\xC0\x00\x11\x08\x00\x01\x00\x01\x01\x01\x11\x00\x02\x11\x01\x03\x11\x01\xFF\xC4\x00\x14\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x08\xFF\xC4\x00\x14\x10\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xFF\xDA\x00\x0C\x03\x01\x00\x02\x11\x03\x11\x00\x3F\x00\xAA\xFF\xD9", // Minimal valid JPEG
			mimeType:       "image/jpeg",
			messageID:      "999",
			expectedStatus: http.StatusBadRequest,
			setupMock: func(mr *MockRepository) {
				mr.On("GetMessageByID", mock.Anything, uint(999)).Return((*models.Message)(nil), fmt.Errorf("not found"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			tt.setupMock(mockRepo)
			
			// Create multipart form
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			
			// Add file part
			part, err := writer.CreateFormFile("file", tt.filename)
			assert.NoError(t, err)
			
			_, err = io.WriteString(part, tt.content)
			assert.NoError(t, err)
			
			// Add message_id field if provided
			if tt.messageID != "" {
				err = writer.WriteField("message_id", tt.messageID)
				assert.NoError(t, err)
			}
			
			err = writer.Close()
			assert.NoError(t, err)
			
			// Create request
			req := httptest.NewRequest("POST", "/api/v1/media/upload", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Request = req
			
			// Call handler
			server.UploadFile(ctx)
			
			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedStatus == http.StatusCreated {
				var response UploadResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, uint(1), response.MediaID)
				assert.Contains(t, response.URL, ".jpg") // URL contains file extension
				assert.Equal(t, tt.filename, response.Filename)
			}
			
			// Reset mock for next test
			mockRepo.ExpectedCalls = nil
		})
	}
}

func TestGetMedia(t *testing.T) {
	server, mockRepo := setupTestServer()
	
	tests := []struct {
		name           string
		mediaID        string
		expectedStatus int
		setupMock      func(*MockRepository)
	}{
		{
			name:           "Valid media ID",
			mediaID:        "1",
			expectedStatus: http.StatusOK,
			setupMock: func(mr *MockRepository) {
				media := &models.Media{
					ID:       1,
					URL:      "http://localhost:9090/uploads/test.jpg",
					MimeType: "image/jpeg",
					Size:     1024,
					Filename: "test.jpg",
				}
				mr.On("GetMediaByID", mock.Anything, uint(1)).Return(media, nil)
			},
		},
		{
			name:           "Invalid media ID",
			mediaID:        "invalid",
			expectedStatus: http.StatusBadRequest,
			setupMock:      func(mr *MockRepository) {},
		},
		{
			name:           "Media not found",
			mediaID:        "999",
			expectedStatus: http.StatusNotFound,
			setupMock: func(mr *MockRepository) {
				mr.On("GetMediaByID", mock.Anything, uint(999)).Return((*models.Media)(nil), fmt.Errorf("not found"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			tt.setupMock(mockRepo)
			
			// Create request
			req := httptest.NewRequest("GET", "/api/v1/media/"+tt.mediaID, nil)
			
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Request = req
			ctx.Params = gin.Params{{Key: "id", Value: tt.mediaID}}
			
			// Call handler
			server.GetMedia(ctx)
			
			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedStatus == http.StatusOK {
				var media models.Media
				err := json.Unmarshal(w.Body.Bytes(), &media)
				assert.NoError(t, err)
				assert.Equal(t, uint(1), media.ID)
			}
			
			// Reset mock for next test
			mockRepo.ExpectedCalls = nil
		})
	}
}

func TestDeleteMedia(t *testing.T) {
	server, mockRepo := setupTestServer()
	
	// Create test file
	testDir := "./test_uploads"
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)
	
	testFile := filepath.Join(testDir, "test.jpg")
	os.WriteFile(testFile, []byte("test content"), 0644)
	
	// Override config for testing
	DefaultMediaConfig.LocalStoragePath = testDir
	
	tests := []struct {
		name           string
		mediaID        string
		expectedStatus int
		setupMock      func(*MockRepository)
	}{
		{
			name:           "Valid deletion",
			mediaID:        "1",
			expectedStatus: http.StatusOK,
			setupMock: func(mr *MockRepository) {
				media := &models.Media{
					ID:       1,
					URL:      "http://localhost:9090/uploads/test.jpg",
					MimeType: "image/jpeg",
					Size:     1024,
					Filename: "test.jpg",
				}
				mr.On("GetMediaByID", mock.Anything, uint(1)).Return(media, nil)
				mr.On("DeleteMedia", mock.Anything, uint(1)).Return(nil)
			},
		},
		{
			name:           "Media not found",
			mediaID:        "999",
			expectedStatus: http.StatusNotFound,
			setupMock: func(mr *MockRepository) {
				mr.On("GetMediaByID", mock.Anything, uint(999)).Return((*models.Media)(nil), fmt.Errorf("not found"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			tt.setupMock(mockRepo)
			
			// Create request
			req := httptest.NewRequest("DELETE", "/api/v1/media/"+tt.mediaID, nil)
			
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Request = req
			ctx.Params = gin.Params{{Key: "id", Value: tt.mediaID}}
			
			// Call handler
			server.DeleteMedia(ctx)
			
			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			// Reset mock for next test
			mockRepo.ExpectedCalls = nil
		})
	}
}

func TestValidateFile(t *testing.T) {
	server, _ := setupTestServer()
	config := DefaultMediaConfig
	
	tests := []struct {
		name        string
		filename    string
		content     string
		size        int64
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid JPEG file",
			filename:    "test.jpg",
			content:     "\xff\xd8\xff\xe0", // JPEG magic bytes
			size:        1024,
			expectError: false,
		},
		{
			name:        "File too large",
			filename:    "large.jpg",
			content:     "\xff\xd8\xff\xe0",
			size:        config.MaxFileSize + 1,
			expectError: true,
			errorMsg:    "exceeds maximum",
		},
		{
			name:        "Invalid file type",
			filename:    "test.exe",
			content:     "MZ", // PE executable magic bytes
			size:        1024,
			expectError: true,
			errorMsg:    "not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile, err := os.CreateTemp("", tt.filename)
			assert.NoError(t, err)
			defer os.Remove(tmpFile.Name())
			
			_, err = tmpFile.WriteString(tt.content)
			assert.NoError(t, err)
			tmpFile.Close()
			
			// Create multipart file header
			fileHeader := &multipart.FileHeader{
				Filename: tt.filename,
				Size:     tt.size,
			}
			
			// Mock the file opening
			if tt.size <= config.MaxFileSize {
				// For size validation, we need to create the actual file content
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, _ := writer.CreateFormFile("file", tt.filename)
				io.WriteString(part, tt.content)
				writer.Close()
				
				// Parse the multipart form to get a real FileHeader
				req := httptest.NewRequest("POST", "/", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				req.ParseMultipartForm(32 << 20)
				
				if req.MultipartForm != nil && len(req.MultipartForm.File["file"]) > 0 {
					fileHeader = req.MultipartForm.File["file"][0]
				}
			}
			
			// Test validation
			err = server.ValidateFile(fileHeader, config)
			
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerateUniqueFilename(t *testing.T) {
	filename1 := GenerateUniqueFilename("test.jpg")
	filename2 := GenerateUniqueFilename("test.jpg")
	
	// Should be different
	assert.NotEqual(t, filename1, filename2)
	
	// Should preserve extension
	assert.True(t, strings.HasSuffix(filename1, ".jpg"))
	assert.True(t, strings.HasSuffix(filename2, ".jpg"))
	
	// Should contain timestamp and random string
	assert.Contains(t, filename1, "_")
	assert.Contains(t, filename2, "_")
}
