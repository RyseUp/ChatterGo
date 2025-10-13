package repository

import (
	"context"

	"github.com/RyseUp/ChatterGo/internal/domain"
)

type MessageRepository interface {
	Create(ctx context.Context, message *domain.Message) error
	GetByRoomID(ctx context.Context, roomID int64, limit, offset int) ([]*domain.Message, error)
	GetByID(ctx context.Context, id int64) (*domain.Message, error)
}
