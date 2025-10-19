package repositories

import (
	"context"

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
}
