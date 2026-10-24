package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/apperr"
)

type NotificationRepo struct {
	db *gorm.DB
}

// NotificationView 列表查询条件与结果。
type NotificationListQuery struct {
	UserID     uint64
	OnlyUnread bool
	Page       int
	PageSize   int
}

// List 返回通知列表、总数与未读数（触发人昵称、问题标题联查）。
func (r *NotificationRepo) List(ctx context.Context, q NotificationListQuery) ([]model.NotificationView, int64, int64, error) {
	base := func() *gorm.DB {
		db := r.db.WithContext(ctx).Table("notifications n").Where("n.user_id = ?", q.UserID)
		if q.OnlyUnread {
			db = db.Where("n.is_read = 0")
		}
		return db
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, 0, err
	}
	var unread int64
	if err := r.db.WithContext(ctx).Table("notifications").
		Where("user_id = ? AND is_read = 0", q.UserID).Count(&unread).Error; err != nil {
		return nil, 0, 0, err
	}

	var list []model.NotificationView
	err := base().
		Select("n.*, a.username AS actor_name, IFNULL(q.title, '') AS question_title").
		Joins("LEFT JOIN users a ON a.id = n.actor_id").
		Joins("LEFT JOIN questions q ON q.id = n.question_id").
		Order("n.id DESC").
		Limit(q.PageSize).Offset((q.Page - 1) * q.PageSize).
		Scan(&list).Error
	return list, total, unread, err
}

func (r *NotificationRepo) Create(ctx context.Context, n *model.Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *NotificationRepo) MarkRead(ctx context.Context, id, userID uint64) error {
	res := r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).Update("is_read", true)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (r *NotificationRepo) MarkAllRead(ctx context.Context, userID uint64) error {
	return r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND is_read = 0", userID).Update("is_read", true).Error
}

// GetContentAuthor 查内容作者与所属问题（用于投票通知定位收件人）。
func (r *NotificationRepo) GetContentAuthor(ctx context.Context, targetType string, targetID uint64) (authorID, questionID uint64, title string, err error) {
	switch targetType {
	case model.TargetQuestion:
		var q model.Question
		if err = r.db.WithContext(ctx).Select("id", "user_id", "title").First(&q, targetID).Error; err != nil {
			return 0, 0, "", err
		}
		return q.UserID, q.ID, q.Title, nil
	case model.TargetAnswer:
		var a model.Answer
		if err = r.db.WithContext(ctx).Select("id", "user_id", "question_id").First(&a, targetID).Error; err != nil {
			return 0, 0, "", err
		}
		var q model.Question
		if err = r.db.WithContext(ctx).Select("id", "title").First(&q, a.QuestionID).Error; err != nil {
			return 0, 0, "", err
		}
		return a.UserID, a.QuestionID, q.Title, nil
	}
	return 0, 0, "", nil
}
