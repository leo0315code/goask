package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/apperr"
)

// locking 返回 SELECT ... FOR UPDATE 行锁子句。
func locking() clause.Expression {
	return clause.Locking{Strength: "UPDATE"}
}

type AnswerRepo struct {
	db *gorm.DB
}

// Create 单事务完成：插入回答、问题 answer_count +1 并刷新活动时间、回答者计数 +1。
func (r *AnswerRepo) Create(ctx context.Context, a *model.Answer) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(a).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Question{}).Where("id = ?", a.QuestionID).
			Updates(map[string]any{
				"answer_count":     gorm.Expr("answer_count + 1"),
				"last_activity_at": gorm.Expr("NOW(3)"),
			}).Error; err != nil {
			return err
		}
		return tx.Model(&model.User{}).Where("id = ?", a.UserID).
			UpdateColumn("answer_count", gorm.Expr("answer_count + 1")).Error
	})
}

func (r *AnswerRepo) GetByID(ctx context.Context, id uint64) (*model.Answer, error) {
	var a model.Answer
	if err := r.db.WithContext(ctx).First(&a, id).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &a, nil
}

// AnswerListQuery 回答列表查询条件。
type AnswerListQuery struct {
	QuestionID uint64
	Sort       string // latest | votes
	Page       int
	PageSize   int
}

func (r *AnswerRepo) List(ctx context.Context, q AnswerListQuery) ([]model.Answer, int64, error) {
	base := func() *gorm.DB {
		return r.db.WithContext(ctx).Model(&model.Answer{}).
			Where("question_id = ? AND status = ?", q.QuestionID, model.AnswerStatusPublished)
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := "created_at ASC, id ASC"
	if q.Sort == "votes" {
		order = "is_accepted DESC, upvote_count DESC, id ASC"
	} else {
		order = "is_accepted DESC, " + order
	}
	var list []model.Answer
	err := base().Order(order).Limit(q.PageSize).Offset((q.Page - 1) * q.PageSize).Find(&list).Error
	return list, total, err
}

func (r *AnswerRepo) Update(ctx context.Context, id uint64, content string) error {
	res := r.db.WithContext(ctx).Model(&model.Answer{}).
		Where("id = ? AND status = ?", id, model.AnswerStatusPublished).
		Update("content", content)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperr.With(apperr.ErrNotFound, "回答不存在或已删除")
	}
	return nil
}

// Delete 软删除回答，回减问题与用户计数；若该回答已被采纳则同时清除采纳标记。
func (r *AnswerRepo) Delete(ctx context.Context, a *model.Answer) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.Answer{}).
			Where("id = ? AND status = ?", a.ID, model.AnswerStatusPublished).
			Update("status", model.AnswerStatusDeleted)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return apperr.With(apperr.ErrNotFound, "回答不存在或已删除")
		}
		if err := tx.Model(&model.Question{}).Where("id = ? AND answer_count > 0", a.QuestionID).
			UpdateColumn("answer_count", gorm.Expr("answer_count - 1")).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).Where("id = ? AND answer_count > 0", a.UserID).
			UpdateColumn("answer_count", gorm.Expr("answer_count - 1")).Error; err != nil {
			return err
		}
		if a.IsAccepted {
			return tx.Model(&model.Question{}).
				Where("id = ? AND accepted_answer_id = ?", a.QuestionID, a.ID).
				Update("accepted_answer_id", nil).Error
		}
		return nil
	})
}
