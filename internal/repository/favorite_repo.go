package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/apperr"
)

type FavoriteRepo struct {
	db *gorm.DB
}

func (r *FavoriteRepo) IsFavorited(ctx context.Context, userID, questionID uint64) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.Favorite{}).
		Where("user_id = ? AND question_id = ?", userID, questionID).Count(&n).Error
	return n > 0, err
}

// Toggle 收藏/取消收藏，返回切换后的状态。
func (r *FavoriteRepo) Toggle(ctx context.Context, userID, questionID uint64) (bool, error) {
	var existing model.Favorite
	err := r.db.WithContext(ctx).Where("user_id = ? AND question_id = ?", userID, questionID).First(&existing).Error
	if err == nil {
		return false, r.db.WithContext(ctx).Delete(&existing).Error
	}
	if err != gorm.ErrRecordNotFound {
		return false, err
	}
	fav := &model.Favorite{UserID: userID, QuestionID: questionID}
	if err := r.db.WithContext(ctx).Create(fav).Error; err != nil {
		return false, err
	}
	return true, nil
}

// FavoriteItemView 我的收藏列表视图（联出问题信息与作者）。
type FavoriteItemView struct {
	QuestionID  uint64 `json:"question_id"`
	Title       string `json:"title"`
	AuthorName  string `json:"author_name"`
	AnswerCount int    `json:"answer_count"`
	ViewCount   int    `json:"view_count"`
	UpvoteCount int    `json:"upvote_count"`
	FavoritedAt string `json:"favorited_at"`
}

// ListByUser 用户收藏的问题列表（排除已删除的问题）。
func (r *FavoriteRepo) ListByUser(ctx context.Context, userID uint64, page, pageSize int) ([]FavoriteItemView, int64, error) {
	base := func() *gorm.DB {
		return r.db.WithContext(ctx).Table("favorites f").
			Joins("JOIN questions q ON q.id = f.question_id AND q.status <> 'deleted'").
			Where("f.user_id = ?", userID)
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []FavoriteItemView
	err := base().
		Select("f.question_id, q.title, u.username AS author_name, q.answer_count, q.view_count, q.upvote_count, f.created_at AS favorited_at").
		Joins("LEFT JOIN users u ON u.id = q.user_id").
		Order("f.id DESC").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Scan(&list).Error
	return list, total, err
}

// AnswerUserView 用户回答列表视图（联出问题标题）。
type AnswerUserView struct {
	ID          uint64 `json:"id"`
	QuestionID  uint64 `json:"question_id"`
	QuestionTtl string `json:"question_title"`
	Content     string `json:"content"`
	UpvoteCount int    `json:"upvote_count"`
	IsAccepted  bool   `json:"is_accepted"`
	CreatedAt   string `json:"created_at"`
}

// AnswerListByUser 某用户的回答列表（含问题标题，排除已删除问题）。
func (r *FavoriteRepo) AnswerListByUser(ctx context.Context, userID uint64, page, pageSize int) ([]AnswerUserView, int64, error) {
	base := func() *gorm.DB {
		return r.db.WithContext(ctx).Table("answers a").
			Joins("JOIN questions q ON q.id = a.question_id AND q.status <> 'deleted'").
			Where("a.user_id = ? AND a.status = ?", userID, model.AnswerStatusPublished)
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []AnswerUserView
	err := base().
		Select("a.id, a.question_id, q.title AS question_ttl, a.content, a.upvote_count, a.is_accepted, a.created_at").
		Order("a.id DESC").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Scan(&list).Error
	return list, total, err
}

var _ = apperr.ErrNotFound
