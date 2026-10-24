package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/leo0315code/goask/internal/model"
)

type CommentRepo struct {
	db *gorm.DB
}

func (r *CommentRepo) Create(ctx context.Context, c *model.Comment) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *CommentRepo) ListByTarget(ctx context.Context, targetType string, targetID uint64, page, pageSize int) ([]model.Comment, int64, error) {
	base := func() *gorm.DB {
		return r.db.WithContext(ctx).Model(&model.Comment{}).
			Where("target_type = ? AND target_id = ? AND status = ?", targetType, targetID, model.CommentStatusPublished)
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Comment
	err := base().Order("id ASC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&list).Error
	return list, total, err
}
