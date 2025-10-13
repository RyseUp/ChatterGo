package usecase

import (
	"context"
	"errors"

	"github.com/RyseUp/ChatterGo/internal/domain"
	"github.com/RyseUp/ChatterGo/internal/repository"
)

type RoomUseCase interface {
	CreateRoom(ctx context.Context, userID int64, req *domain.CreateRoomRequest) (*domain.Room, error)
	JoinRoom(ctx context.Context, userID int64, req *domain.JoinRoomRequest) error
	LeaveRoom(ctx context.Context, userID, roomID int64) error
	ListRooms(ctx context.Context, limit, offset int) ([]*domain.Room, error)
	GetRoom(ctx context.Context, roomID int64) (*domain.Room, error)
	GetRoomMembers(ctx context.Context, roomID int64) ([]*domain.RoomMember, error)
}

type roomUseCase struct {
	roomRepo repository.RoomRepository
}

func NewRoomUseCase(roomRepo repository.RoomRepository) RoomUseCase {
	return &roomUseCase{
		roomRepo: roomRepo,
	}
}

func (uc *roomUseCase) CreateRoom(ctx context.Context, userID int64, req *domain.CreateRoomRequest) (*domain.Room, error) {
	room := &domain.Room{
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   userID,
	}

	if err := uc.roomRepo.Create(ctx, room); err != nil {
		return nil, err
	}

	// Automatically add creator as a member
	if err := uc.roomRepo.AddMember(ctx, room.ID, userID); err != nil {
		return nil, err
	}

	return room, nil
}

func (uc *roomUseCase) JoinRoom(ctx context.Context, userID int64, req *domain.JoinRoomRequest) error {
	// Check if room exists
	_, err := uc.roomRepo.GetByID(ctx, req.RoomID)
	if err != nil {
		return errors.New("room not found")
	}

	// Add user as member
	return uc.roomRepo.AddMember(ctx, req.RoomID, userID)
}

func (uc *roomUseCase) LeaveRoom(ctx context.Context, userID, roomID int64) error {
	return uc.roomRepo.RemoveMember(ctx, roomID, userID)
}

func (uc *roomUseCase) ListRooms(ctx context.Context, limit, offset int) ([]*domain.Room, error) {
	return uc.roomRepo.List(ctx, limit, offset)
}

func (uc *roomUseCase) GetRoom(ctx context.Context, roomID int64) (*domain.Room, error) {
	return uc.roomRepo.GetByID(ctx, roomID)
}

func (uc *roomUseCase) GetRoomMembers(ctx context.Context, roomID int64) ([]*domain.RoomMember, error) {
	return uc.roomRepo.GetMembers(ctx, roomID)
}
