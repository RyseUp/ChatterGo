package postgres

import (
	"context"
	"fmt"

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
