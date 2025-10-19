package services

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/RyseUp/ChatterGo/internal/models"
	"github.com/gin-gonic/gin"
)

type UserRegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=6"`
}

type UserResponse struct {
	ID       uint   `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// UserRegister handles user registration
func (s *ServiceServer) UserRegister(ctx *gin.Context) {
	var req UserRegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fmt.Printf("failed to bind json: %v\n", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	// Check if user already exists
	existingUser, err := s.r.GetUserByUserEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		ctx.JSON(http.StatusConflict, gin.H{"error": "user with this email already exists"})
		return
	}

	user := &models.User{
		Email:    req.Email,
		Username: req.Username,
		Password: req.Password, // TODO: Hash password before storing
	}

	if err := s.r.CreateUser(ctx, user); err != nil {
		fmt.Printf("failed to create user: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	response := UserResponse{
		ID:       user.ID,
		Email:    user.Email,
		Username: user.Username,
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "user created successfully",
		"user":    response,
	})
}

// GetUserByID handles getting a user by ID
func (s *ServiceServer) GetUserByID(ctx *gin.Context) {
	userIDStr := ctx.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	user, err := s.r.GetUserByUserID(ctx, uint(userID))
	if err != nil {
		fmt.Printf("failed to get user by ID: %v\n", err)
		ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	response := UserResponse{
		ID:       user.ID,
		Email:    user.Email,
		Username: user.Username,
	}

	ctx.JSON(http.StatusOK, gin.H{"user": response})
}

// GetUserByEmail handles getting a user by email
func (s *ServiceServer) GetUserByEmail(ctx *gin.Context) {
	email := ctx.Query("email")
	if email == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "email parameter is required"})
		return
	}

	user, err := s.r.GetUserByUserEmail(ctx, email)
	if err != nil {
		fmt.Printf("failed to get user by email: %v\n", err)
		ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	response := UserResponse{
		ID:       user.ID,
		Email:    user.Email,
		Username: user.Username,
	}

	ctx.JSON(http.StatusOK, gin.H{"user": response})
}
