package postgres

import (
	"context"

	"github.com/RyseUp/ChatterGo/internal/models"
)

// Notification methods
func (q *Queries) CreateNotification(ctx context.Context, notification *models.Notification) error {
	return q.db.WithContext(ctx).Create(notification).Error
}

func (q *Queries) GetNotificationByID(ctx context.Context, id uint) (*models.Notification, error) {
	var notification models.Notification
	err := q.db.WithContext(ctx).
		Preload("User").
		Preload("Conversation").
		Preload("RelatedMessage").
		First(&notification, id).Error
	if err != nil {
		return nil, err
	}
	return &notification, nil
}

func (q *Queries) GetNotificationsByUserID(ctx context.Context, userID uint, limit, offset int) ([]models.Notification, int64, error) {
	var notifications []models.Notification
	var total int64

	// Count total
	q.db.WithContext(ctx).Model(&models.Notification{}).
		Where("user_id = ?", userID).
		Count(&total)

	// Get notifications with pagination
	err := q.db.WithContext(ctx).
		Preload("Conversation").
		Preload("RelatedMessage").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error

	return notifications, total, err
}

func (q *Queries) GetUnreadNotificationsByUserID(ctx context.Context, userID uint) ([]models.Notification, error) {
	var notifications []models.Notification
	err := q.db.WithContext(ctx).
		Preload("Conversation").
		Preload("RelatedMessage").
		Where("user_id = ? AND status = ?", userID, models.NotificationStatusUnread).
		Order("created_at DESC").
		Find(&notifications).Error
	return notifications, err
}

func (q *Queries) MarkNotificationAsRead(ctx context.Context, id uint) error {
	return q.db.WithContext(ctx).
		Model(&models.Notification{}).
		Where("id = ?", id).
		Update("status", models.NotificationStatusRead).Error
}

func (q *Queries) MarkAllNotificationsAsRead(ctx context.Context, userID uint) error {
	return q.db.WithContext(ctx).
		Model(&models.Notification{}).
		Where("user_id = ? AND status = ?", userID, models.NotificationStatusUnread).
		Update("status", models.NotificationStatusRead).Error
}

func (q *Queries) DeleteNotification(ctx context.Context, id uint) error {
	return q.db.WithContext(ctx).Delete(&models.Notification{}, id).Error
}

// Notification Preference methods
func (q *Queries) CreateNotificationPreference(ctx context.Context, pref *models.NotificationPreference) error {
	return q.db.WithContext(ctx).Create(pref).Error
}

func (q *Queries) GetNotificationPreferenceByUserID(ctx context.Context, userID uint) (*models.NotificationPreference, error) {
	var pref models.NotificationPreference
	err := q.db.WithContext(ctx).Where("user_id = ?", userID).First(&pref).Error
	if err != nil {
		return nil, err
	}
	return &pref, nil
}

func (q *Queries) UpdateNotificationPreference(ctx context.Context, userID uint, updates map[string]interface{}) error {
	return q.db.WithContext(ctx).
		Model(&models.NotificationPreference{}).
		Where("user_id = ?", userID).
		Updates(updates).Error
}
