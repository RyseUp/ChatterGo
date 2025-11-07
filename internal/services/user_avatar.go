package services

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// UploadAvatar godoc
// @Summary Upload user avatar
// @Description Upload and update user's avatar image
// @Tags users
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param avatar formData file true "Avatar image file"
// @Success 200 {object} map[string]interface{} "Avatar uploaded successfully"
// @Failure 400 {object} map[string]interface{} "Invalid file or request"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 413 {object} map[string]interface{} "File too large"
// @Failure 415 {object} map[string]interface{} "Unsupported media type"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /users/profile/avatar [post]
func (s *ServiceServer) UploadAvatar(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	config := DefaultMediaConfig

	// Get file from form
	file, err := ctx.FormFile("avatar")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "no file provided", "details": err.Error()})
		return
	}

	// Validate file (only images for avatar)
	imageMimeTypes := []string{"image/jpeg", "image/png", "image/gif", "image/webp"}
	isImage := false
	for _, mimeType := range imageMimeTypes {
		if strings.HasPrefix(file.Header.Get("Content-Type"), mimeType) {
			isImage = true
			break
		}
	}

	if !isImage {
		ctx.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "only image files are allowed for avatar"})
		return
	}

	// Check file size (max 5MB for avatar)
	maxAvatarSize := int64(5 * 1024 * 1024) // 5MB
	if file.Size > maxAvatarSize {
		ctx.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file size exceeds maximum allowed size (5MB)"})
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

	// Get current user to check if they have an old avatar
	user, err := s.r.GetUserByUserID(ctx, userID.(uint))
	if err != nil {
		// Clean up uploaded file
		os.Remove(filepath.Join(config.LocalStoragePath, filepath.Base(relativePath)))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}

	// Delete old avatar file if exists
	if user.AvatarURL != nil && *user.AvatarURL != "" {
		oldPath := strings.TrimPrefix(*user.AvatarURL, config.BaseURL+"/")
		oldFilePath := filepath.Join(config.LocalStoragePath, filepath.Base(oldPath))
		if _, err := os.Stat(oldFilePath); err == nil {
			os.Remove(oldFilePath)
		}
	}

	// Update user's avatar_url
	if err := s.r.UpdateUser(ctx, userID.(uint), map[string]interface{}{
		"avatar_url": fileURL,
	}); err != nil {
		// Clean up uploaded file
		os.Remove(filepath.Join(config.LocalStoragePath, filepath.Base(relativePath)))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update avatar", "details": err.Error()})
		return
	}

	// Get updated user
	updatedUser, err := s.r.GetUserByUserID(ctx, userID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get updated user"})
		return
	}

	response := UserResponse{
		ID:          updatedUser.ID,
		Email:       updatedUser.Email,
		Username:    updatedUser.Username,
		AvatarURL:   updatedUser.AvatarURL,
		IsActive:    updatedUser.IsActive,
		LastLoginAt: updatedUser.LastLoginAt,
		CreatedAt:   updatedUser.CreatedAt,
		UpdatedAt:   updatedUser.UpdatedAt,
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "avatar uploaded successfully",
		"data":    response,
	})
}

