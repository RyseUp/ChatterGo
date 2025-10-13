package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/RyseUp/ChatterGo/internal/domain"
	"github.com/RyseUp/ChatterGo/internal/repository"
)

type messageRepository struct {
	db *sql.DB
}

func NewMessageRepository(db *sql.DB) repository.MessageRepository {
	return &messageRepository{db: db}
}

func (r *messageRepository) Create(ctx context.Context, message *domain.Message) error {
	query := `
		INSERT INTO messages (room_id, user_id, content, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id, created_at
	`
	return r.db.QueryRowContext(ctx, query, message.RoomID, message.UserID, message.Content).
		Scan(&message.ID, &message.CreatedAt)
}

func (r *messageRepository) GetByRoomID(ctx context.Context, roomID int64, limit, offset int) ([]*domain.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT m.id, m.room_id, m.user_id, m.content, m.created_at, u.username
		FROM messages m
		JOIN users u ON m.user_id = u.id
		WHERE m.room_id = $1
		ORDER BY m.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, query, roomID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]*domain.Message, 0)
	for rows.Next() {
		msg := &domain.Message{}
		err := rows.Scan(&msg.ID, &msg.RoomID, &msg.UserID, &msg.Content, &msg.CreatedAt, &msg.Username)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func (r *messageRepository) GetByID(ctx context.Context, id int64) (*domain.Message, error) {
	message := &domain.Message{}
	query := `
		SELECT m.id, m.room_id, m.user_id, m.content, m.created_at, u.username
		FROM messages m
		JOIN users u ON m.user_id = u.id
		WHERE m.id = $1
	`
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&message.ID, &message.RoomID, &message.UserID, &message.Content, &message.CreatedAt, &message.Username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("message not found")
		}
		return nil, err
	}
	return message, nil
}
