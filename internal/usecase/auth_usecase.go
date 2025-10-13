package usecase

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
	"github.com/RyseUp/ChatterGo/internal/domain"
	"github.com/RyseUp/ChatterGo/internal/repository"
	"github.com/RyseUp/ChatterGo/pkg/jwt"
)

type AuthUseCase interface {
	Register(ctx context.Context, req *domain.RegisterRequest) (*domain.AuthResponse, error)
	Login(ctx context.Context, req *domain.LoginRequest) (*domain.AuthResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*domain.AuthResponse, error)
}

type authUseCase struct {
	userRepo     repository.UserRepository
	tokenManager *jwt.TokenManager
}

func NewAuthUseCase(userRepo repository.UserRepository, tokenManager *jwt.TokenManager) AuthUseCase {
	return &authUseCase{
		userRepo:     userRepo,
		tokenManager: tokenManager,
	}
}

func (uc *authUseCase) Register(ctx context.Context, req *domain.RegisterRequest) (*domain.AuthResponse, error) {
	// Check if user already exists
	existingUser, _ := uc.userRepo.GetByEmail(ctx, req.Email)
	if existingUser != nil {
		return nil, errors.New("email already registered")
	}

	existingUser, _ = uc.userRepo.GetByUsername(ctx, req.Username)
	if existingUser != nil {
		return nil, errors.New("username already taken")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create user
	user := &domain.User{
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword),
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Generate tokens
	accessToken, err := uc.tokenManager.GenerateAccessToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := uc.tokenManager.GenerateRefreshToken(user)
	if err != nil {
		return nil, err
	}

	// Clear password before returning
	user.Password = ""

	return &domain.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (uc *authUseCase) Login(ctx context.Context, req *domain.LoginRequest) (*domain.AuthResponse, error) {
	// Get user by email
	user, err := uc.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Generate tokens
	accessToken, err := uc.tokenManager.GenerateAccessToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := uc.tokenManager.GenerateRefreshToken(user)
	if err != nil {
		return nil, err
	}

	// Clear password before returning
	user.Password = ""

	return &domain.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (uc *authUseCase) RefreshToken(ctx context.Context, refreshToken string) (*domain.AuthResponse, error) {
	// Validate refresh token
	claims, err := uc.tokenManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// Get user
	user, err := uc.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Generate new tokens
	newAccessToken, err := uc.tokenManager.GenerateAccessToken(user)
	if err != nil {
		return nil, err
	}

	newRefreshToken, err := uc.tokenManager.GenerateRefreshToken(user)
	if err != nil {
		return nil, err
	}

	// Clear password before returning
	user.Password = ""

	return &domain.AuthResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		User:         user,
	}, nil
}
