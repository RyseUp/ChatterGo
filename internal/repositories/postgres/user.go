package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/RyseUp/ChatterGo/internal/models"
)

func (r *Queries) CreateUser(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Model(&models.User{}).Create(user).Error
}

func (r *Queries) GetUserByUserID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Model(&models.User{}).First(&user, id).Error
	if err != nil {
		return nil, fmt.Errorf("GetUserByUserID: %w", err)
	}
	return &user, nil
}

func (r *Queries) GetUserByUserEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, fmt.Errorf("GetUserByUserEmail: %w", err)
	}
	return &user, nil
}

func (r *Queries) UpdateUser(ctx context.Context, id uint, updates map[string]interface{}) error {
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", id).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("UpdateUser: %w", err)
	}
	return nil
}

func (r *Queries) UpdateUserRefreshToken(ctx context.Context, id uint, refreshToken *string) error {
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", id).Update("refresh_token", refreshToken).Error
	if err != nil {
		return fmt.Errorf("UpdateUserRefreshToken: %w", err)
	}
	return nil
}

func (r *Queries) UpdateUserLastLogin(ctx context.Context, id uint, lastLogin time.Time) error {
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", id).Update("last_login_at", lastLogin).Error
	if err != nil {
		return fmt.Errorf("UpdateUserLastLogin: %w", err)
	}
	return nil
}
