package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/RyseUp/ChatterGo/internal/domain"
	"github.com/RyseUp/ChatterGo/internal/repository"
)

type roomRepository struct {
	db *sql.DB
}

func NewRoomRepository(db *sql.DB) repository.RoomRepository {
	return &roomRepository{db: db}
}

func (r *roomRepository) Create(ctx context.Context, room *domain.Room) error {
	query := `
		INSERT INTO rooms (name, description, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, query, room.Name, room.Description, room.CreatedBy).
		Scan(&room.ID, &room.CreatedAt, &room.UpdatedAt)
}

func (r *roomRepository) GetByID(ctx context.Context, id int64) (*domain.Room, error) {
	room := &domain.Room{}
	query := `SELECT id, name, description, created_by, created_at, updated_at FROM rooms WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&room.ID, &room.Name, &room.Description, &room.CreatedBy, &room.CreatedAt, &room.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("room not found")
		}
		return nil, err
	}
	return room, nil
}

func (r *roomRepository) List(ctx context.Context, limit, offset int) ([]*domain.Room, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, name, description, created_by, created_at, updated_at
		FROM rooms
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rooms := make([]*domain.Room, 0)
	for rows.Next() {
		room := &domain.Room{}
		err := rows.Scan(&room.ID, &room.Name, &room.Description, &room.CreatedBy, &room.CreatedAt, &room.UpdatedAt)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	return rooms, rows.Err()
}

func (r *roomRepository) AddMember(ctx context.Context, roomID, userID int64) error {
	query := `
		INSERT INTO room_members (room_id, user_id, joined_at, is_online, last_seen)
		VALUES ($1, $2, NOW(), false, NOW())
		ON CONFLICT (room_id, user_id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query, roomID, userID)
	return err
}

func (r *roomRepository) RemoveMember(ctx context.Context, roomID, userID int64) error {
	query := `DELETE FROM room_members WHERE room_id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, roomID, userID)
	return err
}

func (r *roomRepository) GetMembers(ctx context.Context, roomID int64) ([]*domain.RoomMember, error) {
	query := `
		SELECT id, room_id, user_id, joined_at, is_online, last_seen
		FROM room_members
		WHERE room_id = $1
		ORDER BY joined_at
	`
	rows, err := r.db.QueryContext(ctx, query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]*domain.RoomMember, 0)
	for rows.Next() {
		member := &domain.RoomMember{}
		err := rows.Scan(&member.ID, &member.RoomID, &member.UserID, &member.JoinedAt, &member.IsOnline, &member.LastSeen)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

func (r *roomRepository) IsMember(ctx context.Context, roomID, userID int64) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM room_members WHERE room_id = $1 AND user_id = $2)`
	err := r.db.QueryRowContext(ctx, query, roomID, userID).Scan(&exists)
	return exists, err
}

func (r *roomRepository) UpdateMemberStatus(ctx context.Context, roomID, userID int64, isOnline bool) error {
	query := `
		UPDATE room_members
		SET is_online = $1, last_seen = NOW()
		WHERE room_id = $2 AND user_id = $3
	`
	_, err := r.db.ExecContext(ctx, query, isOnline, roomID, userID)
	return err
}
