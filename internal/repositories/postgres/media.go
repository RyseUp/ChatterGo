package postgres

import (
	"context"

	"github.com/RyseUp/ChatterGo/internal/models"
)

func (q *Queries) CreateMedia(ctx context.Context, media *models.Media) error {
	return q.db.WithContext(ctx).Create(media).Error
}

func (q *Queries) GetMediaByID(ctx context.Context, id uint) (*models.Media, error) {
	var media models.Media
	err := q.db.WithContext(ctx).First(&media, id).Error
	if err != nil {
		return nil, err
	}
	return &media, nil
}

func (q *Queries) GetMediaByMessageID(ctx context.Context, messageID uint) ([]models.Media, error) {
	var media []models.Media
	err := q.db.WithContext(ctx).Where("message_id = ?", messageID).Find(&media).Error
	return media, err
}

func (q *Queries) UpdateMedia(ctx context.Context, id uint, updates map[string]interface{}) error {
	return q.db.WithContext(ctx).Model(&models.Media{}).Where("id = ?", id).Updates(updates).Error
}

func (q *Queries) DeleteMedia(ctx context.Context, id uint) error {
	return q.db.WithContext(ctx).Delete(&models.Media{}, id).Error
}

func (q *Queries) DeleteMediaByMessageID(ctx context.Context, messageID uint) error {
	return q.db.WithContext(ctx).Where("message_id = ?", messageID).Delete(&models.Media{}).Error
}
