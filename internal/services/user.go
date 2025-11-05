package services

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/RyseUp/ChatterGo/internal/models"
	"github.com/RyseUp/ChatterGo/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserRegisterRequest struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Username string `json:"username" binding:"required,min=3,max=64" example:"johndoe"`
	Password string `json:"password" binding:"required,min=6" example:"password123"`
}

type UserResponse struct {
	ID          uint       `json:"id" example:"1"`
	Email       string     `json:"email" example:"user@example.com"`
	Username    string     `json:"username" example:"johndoe"`
	IsActive    bool       `json:"is_active" example:"true"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty" example:"2023-10-24T10:30:00Z"`
	CreatedAt   time.Time  `json:"created_at" example:"2023-10-24T10:30:00Z"`
	UpdatedAt   time.Time  `json:"updated_at" example:"2023-10-24T10:30:00Z"`
}

type AuthResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string       `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// UserRegister godoc
// @Summary Register a new user
// @Description Create a new user account
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body UserRegisterRequest true "User registration data"
// @Success 201 {object} map[string]interface{} "User created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 409 {object} map[string]interface{} "User already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/register [post]
func (s *ServiceServer) UserRegister(ctx *gin.Context) {
	var req UserRegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fmt.Printf("failed to bind json: %v\n", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	existingUser, err := s.r.GetUserByUserEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		ctx.JSON(http.StatusConflict, gin.H{"error": "user with this email already exists"})
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		fmt.Printf("failed to hash password: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process request"})
		return
	}

	user := &models.User{
		Email:    req.Email,
		Username: req.Username,
		Password: hashedPassword,
		IsActive: true,
	}

	if err := s.r.CreateUser(ctx, user); err != nil {
		fmt.Printf("failed to create user: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	accessExpiration := time.Duration(s.cfg.JWT.ExpirationHours) * time.Hour
	refreshExpiration := 7 * 24 * time.Hour

	tokenPair, err := utils.GenerateTokenPair(user.ID, user.Email, s.cfg.JWT.Secret, accessExpiration, refreshExpiration)
	if err != nil {
		fmt.Printf("failed to generate tokens: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
		return
	}

	if err := s.r.UpdateUserRefreshToken(ctx, user.ID, &tokenPair.RefreshToken); err != nil {
		fmt.Printf("failed to store refresh token: %v\n", err)

	}

	response := AuthResponse{
		User: UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			Username:  user.Username,
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "user created successfully",
		"data":    response,
	})
}

type UserLoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required" example:"password123"`
}

// UserLogin godoc
// @Summary Login user
// @Description Authenticate user and return access token
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body UserLoginRequest true "User login credentials"
// @Success 200 {object} map[string]interface{} "Login successful"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Invalid credentials"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/login [post]
func (s *ServiceServer) UserLogin(ctx *gin.Context) {
	var req UserLoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	user, err := s.r.GetUserByUserEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		fmt.Printf("failed to get user: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process request"})
		return
	}

	if !user.IsActive {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "account is deactivated"})
		return
	}

	if err := utils.CheckPassword(req.Password, user.Password); err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	accessExpiration := time.Duration(s.cfg.JWT.ExpirationHours) * time.Hour
	refreshExpiration := 7 * 24 * time.Hour

	tokenPair, err := utils.GenerateTokenPair(user.ID, user.Email, s.cfg.JWT.Secret, accessExpiration, refreshExpiration)
	if err != nil {
		fmt.Printf("failed to generate tokens: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
		return
	}

	now := time.Now()
	if err := s.r.UpdateUserLastLogin(ctx, user.ID, now); err != nil {
		fmt.Printf("failed to update last login: %v\n", err)

	}

	if err := s.r.UpdateUserRefreshToken(ctx, user.ID, &tokenPair.RefreshToken); err != nil {
		fmt.Printf("failed to store refresh token: %v\n", err)

	}

	response := AuthResponse{
		User: UserResponse{
			ID:          user.ID,
			Email:       user.Email,
			Username:    user.Username,
			IsActive:    user.IsActive,
			LastLoginAt: &now,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
		},
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "login successful",
		"data": 	response,
	})
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Generate new access token using refresh token
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "Refresh token"
// @Success 200 {object} map[string]interface{} "Tokens refreshed successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Invalid refresh token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/refresh [post]
func (s *ServiceServer) RefreshToken(ctx *gin.Context) {
	var req RefreshTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	refreshExpiration := 7 * 24 * time.Hour
	claims, err := utils.ValidateRefreshToken(req.RefreshToken, s.cfg.JWT.Secret)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	user, err := s.r.GetUserByUserID(ctx, claims.UserID)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	if !user.IsActive {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "account is deactivated"})
		return
	}

	if user.RefreshToken == nil || *user.RefreshToken != req.RefreshToken {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	accessExpiration := time.Duration(s.cfg.JWT.ExpirationHours) * time.Hour
	tokenPair, err := utils.GenerateTokenPair(user.ID, user.Email, s.cfg.JWT.Secret, accessExpiration, refreshExpiration)
	if err != nil {
		fmt.Printf("failed to generate tokens: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
		return
	}

	if err := s.r.UpdateUserRefreshToken(ctx, user.ID, &tokenPair.RefreshToken); err != nil {
		fmt.Printf("failed to store refresh token: %v\n", err)

	}

	response := TokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "tokens refreshed successfully",
		"data":    response,
	})
}

// Logout godoc
// @Summary Logout user
// @Description Logout user by revoking refresh token
// @Tags authentication
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Logout successful"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/logout [post]
func (s *ServiceServer) Logout(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	if err := s.r.UpdateUserRefreshToken(ctx, userID.(uint), nil); err != nil {
		fmt.Printf("failed to clear refresh token: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to logout"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "logout successful"})
}

// GetUserByID godoc
// @Summary Get user by ID
// @Description Get user information by user ID
// @Tags users
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{} "User data"
// @Failure 400 {object} map[string]interface{} "Invalid user ID"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /users/{id} [get]
func (s *ServiceServer) GetUserByID(ctx *gin.Context) {
	userIDStr := ctx.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	user, err := s.r.GetUserByUserID(ctx, uint(userID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		fmt.Printf("failed to get user by ID: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}

	response := UserResponse{
		ID:          user.ID,
		Email:       user.Email,
		Username:    user.Username,
		IsActive:    user.IsActive,
		LastLoginAt: user.LastLoginAt,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}

	ctx.JSON(http.StatusOK, gin.H{"data": response})
}

// GetUserProfile godoc
// @Summary Get current user profile
// @Description Get authenticated user's profile information
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "User profile data"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /users/profile [get]
func (s *ServiceServer) GetUserProfile(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	user, err := s.r.GetUserByUserID(ctx, userID.(uint))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		fmt.Printf("failed to get user profile: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user profile"})
		return
	}

	response := UserResponse{
		ID:          user.ID,
		Email:       user.Email,
		Username:    user.Username,
		IsActive:    user.IsActive,
		LastLoginAt: user.LastLoginAt,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}

	ctx.JSON(http.StatusOK, gin.H{"data": response})
}

type UserUpdateRequest struct {
	Username string `json:"username,omitempty" binding:"omitempty,min=3,max=64" example:"johndoe_updated"`
	Email    string `json:"email,omitempty" binding:"omitempty,email" example:"newemail@example.com"`
}

// UpdateUserProfile godoc
// @Summary Update user profile
// @Description Update current user's profile information
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UserUpdateRequest true "User update data"
// @Success 200 {object} map[string]interface{} "User updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 409 {object} map[string]interface{} "Email already taken"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /users/profile [patch]
func (s *ServiceServer) UpdateUserProfile(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var req UserUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if req.Username != "" {
		updates["username"] = req.Username
	}
	if req.Email != "" {

		existingUser, err := s.r.GetUserByUserEmail(ctx, req.Email)
		if err == nil && existingUser != nil && existingUser.ID != userID.(uint) {
			ctx.JSON(http.StatusConflict, gin.H{"error": "email is already taken"})
			return
		}
		updates["email"] = req.Email
	}

	if len(updates) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	if err := s.r.UpdateUser(ctx, userID.(uint), updates); err != nil {
		fmt.Printf("failed to update user: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}

	user, err := s.r.GetUserByUserID(ctx, userID.(uint))
	if err != nil {
		fmt.Printf("failed to get updated user: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get updated user"})
		return
	}

	response := UserResponse{
		ID:          user.ID,
		Email:       user.Email,
		Username:    user.Username,
		IsActive:    user.IsActive,
		LastLoginAt: user.LastLoginAt,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "user updated successfully",
		"data":    response,
	})
}

func (s *ServiceServer) GetUserByEmail(ctx *gin.Context) {
	email := ctx.Query("email")
	if email == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "email parameter is required"})
		return
	}

	user, err := s.r.GetUserByUserEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		fmt.Printf("failed to get user by email: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}

	response := UserResponse{
		ID:          user.ID,
		Email:       user.Email,
		Username:    user.Username,
		IsActive:    user.IsActive,
		LastLoginAt: user.LastLoginAt,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}

	ctx.JSON(http.StatusOK, gin.H{"data": response})
}
