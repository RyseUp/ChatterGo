package repositories

import (
	"context"
	"time"

	"github.com/RyseUp/ChatterGo/internal/models"
)

type Repository interface {
	Ping() error
	Transaction(ctx context.Context, txFunc func(Repository) error) error
	UserRepository
}

type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByUserID(ctx context.Context, id uint) (*models.User, error)
	GetUserByUserEmail(ctx context.Context, email string) (*models.User, error)
	UpdateUser(ctx context.Context, id uint, updates map[string]interface{}) error
	UpdateUserRefreshToken(ctx context.Context, id uint, refreshToken *string) error
	UpdateUserLastLogin(ctx context.Context, id uint, lastLogin time.Time) error
}
