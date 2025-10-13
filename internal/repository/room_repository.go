package repository

import (
	"context"

	"github.com/RyseUp/ChatterGo/internal/domain"
)

type RoomRepository interface {
	Create(ctx context.Context, room *domain.Room) error
	GetByID(ctx context.Context, id int64) (*domain.Room, error)
	List(ctx context.Context, limit, offset int) ([]*domain.Room, error)
	AddMember(ctx context.Context, roomID, userID int64) error
	RemoveMember(ctx context.Context, roomID, userID int64) error
	GetMembers(ctx context.Context, roomID int64) ([]*domain.RoomMember, error)
	IsMember(ctx context.Context, roomID, userID int64) (bool, error)
	UpdateMemberStatus(ctx context.Context, roomID, userID int64, isOnline bool) error
}
