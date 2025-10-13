package usecase

import (
	"context"
	"errors"

	"github.com/RyseUp/ChatterGo/internal/domain"
	"github.com/RyseUp/ChatterGo/internal/repository"
)

type MessageUseCase interface {
	SendMessage(ctx context.Context, userID int64, req *domain.SendMessageRequest) (*domain.Message, error)
	GetMessageHistory(ctx context.Context, userID int64, req *domain.MessageHistoryRequest) ([]*domain.Message, error)
}

type messageUseCase struct {
	messageRepo repository.MessageRepository
	roomRepo    repository.RoomRepository
	userRepo    repository.UserRepository
}

func NewMessageUseCase(
	messageRepo repository.MessageRepository,
	roomRepo repository.RoomRepository,
	userRepo repository.UserRepository,
) MessageUseCase {
	return &messageUseCase{
		messageRepo: messageRepo,
		roomRepo:    roomRepo,
		userRepo:    userRepo,
	}
}

func (uc *messageUseCase) SendMessage(ctx context.Context, userID int64, req *domain.SendMessageRequest) (*domain.Message, error) {
	// Check if room exists
	room, err := uc.roomRepo.GetByID(ctx, req.RoomID)
	if err != nil {
		return nil, errors.New("room not found")
	}
	_ = room

	// Check if user is a member of the room
	isMember, err := uc.roomRepo.IsMember(ctx, req.RoomID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("you are not a member of this room")
	}

	// Get user to include username in message
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Create message
	message := &domain.Message{
		RoomID:  req.RoomID,
		UserID:  userID,
		Content: req.Content,
	}

	if err := uc.messageRepo.Create(ctx, message); err != nil {
		return nil, err
	}

	// Add username to response
	message.Username = user.Username

	return message, nil
}

func (uc *messageUseCase) GetMessageHistory(ctx context.Context, userID int64, req *domain.MessageHistoryRequest) ([]*domain.Message, error) {
	// Check if user is a member of the room
	isMember, err := uc.roomRepo.IsMember(ctx, req.RoomID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("you are not a member of this room")
	}

	// Get messages
	messages, err := uc.messageRepo.GetByRoomID(ctx, req.RoomID, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}

	return messages, nil
}
