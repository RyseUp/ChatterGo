package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/RyseUp/ChatterGo/internal/models"
	"github.com/gin-gonic/gin"
)

// MediaConfig holds configuration for media uploads
type MediaConfig struct {
	MaxFileSize      int64    `json:"max_file_size"`      // in bytes
	AllowedMimeTypes []string `json:"allowed_mime_types"` // allowed MIME types
	StorageType      string   `json:"storage_type"`       // "local" or "s3"
	LocalStoragePath string   `json:"local_storage_path"` // path for local storage
	BaseURL          string   `json:"base_url"`           // base URL for serving files
}

// Default media configuration
var DefaultMediaConfig = MediaConfig{
	MaxFileSize: 10 * 1024 * 1024, // 10MB
	AllowedMimeTypes: []string{
		"image/jpeg", "image/png", "image/gif", "image/webp",
		"application/pdf", "text/plain",
		"application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	},
	StorageType:      "local",
	LocalStoragePath: "./uploads",
	BaseURL:          "http://localhost:9090",
}

type PresignRequest struct {
	Filename string `json:"filename" binding:"required"`
	MimeType string `json:"mime_type" binding:"required"`
	Size     int64  `json:"size" binding:"required"`
}

type PresignResponse struct {
	UploadURL string `json:"upload_url"`
	MediaID   string `json:"media_id"`
	ExpiresAt int64  `json:"expires_at"`
}

type UploadResponse struct {
	MediaID  uint   `json:"media_id"`
	URL      string `json:"url"`
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
}

// ValidateFile validates file size and MIME type
func (s *ServiceServer) ValidateFile(file *multipart.FileHeader, config MediaConfig) error {
	// Check file size
	if file.Size > config.MaxFileSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d", file.Size, config.MaxFileSize)
	}

	// Check MIME type
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Read first 512 bytes to detect content type
	buffer := make([]byte, 512)
	_, err = src.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file: %w", err)
	}

	contentType := http.DetectContentType(buffer)
	
	// Check if MIME type is allowed
	allowed := false
	for _, allowedType := range config.AllowedMimeTypes {
		if contentType == allowedType {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("file type %s is not allowed", contentType)
	}

	return nil
}

// GenerateUniqueFilename generates a unique filename with timestamp and random string
func GenerateUniqueFilename(originalFilename string) string {
	ext := filepath.Ext(originalFilename)
	timestamp := time.Now().Unix()
	
	// Generate random string
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	randomStr := hex.EncodeToString(randomBytes)
	
	return fmt.Sprintf("%d_%s%s", timestamp, randomStr, ext)
}

// SaveFileLocally saves file to local filesystem
func (s *ServiceServer) SaveFileLocally(file *multipart.FileHeader, config MediaConfig) (string, error) {
	// Ensure upload directory exists
	if err := os.MkdirAll(config.LocalStoragePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Generate unique filename
	uniqueFilename := GenerateUniqueFilename(file.Filename)
	filePath := filepath.Join(config.LocalStoragePath, uniqueFilename)

	// Save file
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	// Return relative path for URL generation
	return filepath.Join("uploads", uniqueFilename), nil
}

// PresignUpload godoc
// @Summary Generate presigned upload URL
// @Description Generate a presigned URL for file upload (placeholder for S3, returns local upload endpoint for now)
// @Tags media
// @Accept json
// @Produce json
// @Param request body PresignRequest true "Presign request data"
// @Success 200 {object} PresignResponse "Presigned URL generated"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 413 {object} map[string]interface{} "File too large"
// @Failure 415 {object} map[string]interface{} "Unsupported media type"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /media/presign [post]
func (s *ServiceServer) PresignUpload(ctx *gin.Context) {
	var req PresignRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	config := DefaultMediaConfig

	// Validate file size
	if req.Size > config.MaxFileSize {
		ctx.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": "file too large",
			"max_size": config.MaxFileSize,
			"provided_size": req.Size,
		})
		return
	}

	// Validate MIME type
	allowed := false
	for _, allowedType := range config.AllowedMimeTypes {
		if req.MimeType == allowedType {
			allowed = true
			break
		}
	}

	if !allowed {
		ctx.JSON(http.StatusUnsupportedMediaType, gin.H{
			"error": "unsupported media type",
			"allowed_types": config.AllowedMimeTypes,
			"provided_type": req.MimeType,
		})
		return
	}

	// Generate media ID for tracking
	mediaID := GenerateUniqueFilename(req.Filename)
	
	// For local storage, return direct upload endpoint
	// In production with S3, this would generate actual presigned URL
	uploadURL := fmt.Sprintf("%s/api/v1/media/upload?media_id=%s", config.BaseURL, mediaID)
	
	response := PresignResponse{
		UploadURL: uploadURL,
		MediaID:   mediaID,
		ExpiresAt: time.Now().Add(15 * time.Minute).Unix(), // 15 minutes expiry
	}

	ctx.JSON(http.StatusOK, response)
}

// UploadFile godoc
// @Summary Upload file
// @Description Upload a file and create media record
// @Tags media
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File to upload"
// @Param message_id formData int true "Message ID to associate with"
// @Success 201 {object} UploadResponse "File uploaded successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 413 {object} map[string]interface{} "File too large"
// @Failure 415 {object} map[string]interface{} "Unsupported media type"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /media/upload [post]
func (s *ServiceServer) UploadFile(ctx *gin.Context) {
	config := DefaultMediaConfig

	// Get file from form
	file, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "no file provided", "details": err.Error()})
		return
	}

	// Validate file
	if err := s.ValidateFile(file, config); err != nil {
		if strings.Contains(err.Error(), "exceeds maximum") {
			ctx.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": err.Error()})
		} else if strings.Contains(err.Error(), "not allowed") {
			ctx.JSON(http.StatusUnsupportedMediaType, gin.H{"error": err.Error()})
		} else {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	// Save file
	relativePath, err := s.SaveFileLocally(file, config)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file", "details": err.Error()})
		return
	}

	// Generate full URL
	fileURL := fmt.Sprintf("%s/%s", config.BaseURL, strings.ReplaceAll(relativePath, "\\", "/"))

	// Get MIME type
	src, _ := file.Open()
	defer src.Close()
	buffer := make([]byte, 512)
	src.Read(buffer)
	mimeType := http.DetectContentType(buffer)

	// Get and validate message_id (optional - allows uploading before message creation)
	var messageID *uint
	messageIDStr := ctx.PostForm("message_id")
	if messageIDStr != "" {
		parsedID, err := strconv.ParseUint(messageIDStr, 10, 32)
		if err != nil {
			// Clean up uploaded file
			os.Remove(filepath.Join(config.LocalStoragePath, filepath.Base(relativePath)))
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid message_id format"})
			return
		}

		// Validate that the message exists
		_, err = s.r.GetMessageByID(ctx, uint(parsedID))
		if err != nil {
			// Clean up uploaded file
			os.Remove(filepath.Join(config.LocalStoragePath, filepath.Base(relativePath)))
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "message not found", "details": "the specified message_id does not exist"})
			return
		}

		id := uint(parsedID)
		messageID = &id
	}

	// Create media record
	media := &models.Media{
		MessageID: messageID, // Can be nil if uploading before message creation
		URL:       fileURL,
		MimeType:  mimeType,
		Size:      file.Size,
		Filename:  file.Filename,
	}

	// Save to database
	if err := s.r.CreateMedia(ctx, media); err != nil {
		// Clean up uploaded file on database error
		os.Remove(filepath.Join(config.LocalStoragePath, filepath.Base(relativePath)))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save media record", "details": err.Error()})
		return
	}

	response := UploadResponse{
		MediaID:  media.ID,
		URL:      media.URL,
		Filename: media.Filename,
		MimeType: media.MimeType,
		Size:     media.Size,
	}

	ctx.JSON(http.StatusCreated, response)
}

// GetMedia godoc
// @Summary Get media by ID
// @Description Get media information by ID
// @Tags media
// @Produce json
// @Param id path int true "Media ID"
// @Success 200 {object} models.Media "Media information"
// @Failure 400 {object} map[string]interface{} "Invalid media ID"
// @Failure 404 {object} map[string]interface{} "Media not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /media/{id} [get]
func (s *ServiceServer) GetMedia(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid media ID"})
		return
	}

	media, err := s.r.GetMediaByID(ctx, uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "media not found"})
		return
	}

	ctx.JSON(http.StatusOK, media)
}

// DeleteMedia godoc
// @Summary Delete media
// @Description Delete media by ID
// @Tags media
// @Produce json
// @Param id path int true "Media ID"
// @Success 200 {object} map[string]interface{} "Media deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid media ID"
// @Failure 404 {object} map[string]interface{} "Media not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /media/{id} [delete]
func (s *ServiceServer) DeleteMedia(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid media ID"})
		return
	}

	// Get media info first to delete file
	media, err := s.r.GetMediaByID(ctx, uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "media not found"})
		return
	}

	// Delete from database
	if err := s.r.DeleteMedia(ctx, uint(id)); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete media", "details": err.Error()})
		return
	}

	// Delete physical file (for local storage)
	config := DefaultMediaConfig
	if config.StorageType == "local" {
		// Extract filename from URL
		urlParts := strings.Split(media.URL, "/")
		if len(urlParts) > 0 {
			filename := urlParts[len(urlParts)-1]
			filePath := filepath.Join(config.LocalStoragePath, filename)
			os.Remove(filePath) // Ignore error, file might not exist
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "media deleted successfully"})
}
