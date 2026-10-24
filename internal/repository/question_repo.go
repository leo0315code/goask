package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/apperr"
)

type QuestionRepo struct {
	db *gorm.DB
}

// CreateQuestionInput 创建问题的事务输入。
type CreateQuestionInput struct {
	Question *model.Question
	Tags     []string // 已规范化的标签名
}

// Create 单事务完成：插入问题、关联/创建标签、修正标签计数、提问者计数 +1。
func (r *QuestionRepo) Create(ctx context.Context, input CreateQuestionInput) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(input.Question).Error; err != nil {
			return translateDuplicate(err)
		}
		tagIDs := make([]uint64, 0, len(input.Tags))
		tagRepo := &TagRepo{db: tx}
		for _, name := range input.Tags {
			t, err := tagRepo.getOrCreateTx(tx, name)
			if err != nil {
				return err
			}
			tagIDs = append(tagIDs, t.ID)
			if err := tx.Exec("INSERT IGNORE INTO question_tags (question_id, tag_id) VALUES (?, ?)",
				input.Question.ID, t.ID).Error; err != nil {
				return err
			}
		}
		if err := refreshTagCountsTx(tx, tagIDs); err != nil {
			return err
		}
		return tx.Model(&model.User{}).Where("id = ?", input.Question.UserID).
			UpdateColumn("question_count", gorm.Expr("question_count + 1")).Error
	})
}

func (r *QuestionRepo) GetByID(ctx context.Context, id uint64) (*model.Question, error) {
	var q model.Question
	if err := r.db.WithContext(ctx).Preload("Tags").First(&q, id).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &q, nil
}

// QuestionListQuery 列表查询条件。
type QuestionListQuery struct {
	Sort     string // latest | hot | unanswered；Search 非空时按相关度
	TagID    uint64 // 0 = 不过滤
	Since    string // 时间过滤：day | week | month | year（其他 = 全部）
	Search   string // 全文检索关键词
	UserID   uint64 // 0 = 不过滤
	Page     int
	PageSize int
}

// sinceIntervals 时间过滤取值 → SQL 区间表达式（白名单，避免拼接用户输入）。
var sinceIntervals = map[string]string{
	"day":   "INTERVAL 1 DAY",
	"week":  "INTERVAL 7 DAY",
	"month": "INTERVAL 30 DAY",
	"year":  "INTERVAL 365 DAY",
}

func (r *QuestionRepo) List(ctx context.Context, q QuestionListQuery) ([]model.Question, int64, error) {
	base := func() *gorm.DB {
		db := r.db.WithContext(ctx).Model(&model.Question{}).
			Where("status <> ?", model.QuestionStatusDeleted)
		if expr, ok := sinceIntervals[q.Since]; ok {
			db = db.Where("created_at >= NOW() - " + expr)
		}
		if q.TagID > 0 {
			db = db.Where("id IN (SELECT question_id FROM question_tags WHERE tag_id = ?)", q.TagID)
		}
		if q.Search != "" {
			db = db.Where("MATCH(title, content) AGAINST (? IN NATURAL LANGUAGE MODE)", q.Search)
		}
		if q.UserID > 0 {
			db = db.Where("user_id = ?", q.UserID)
		}
		if q.Sort == "unanswered" {
			db = db.Where("answer_count = 0")
		}
		return db
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}

	order := "last_activity_at DESC, id DESC"
	switch q.Sort {
	case "hot":
		// Reddit 式时间衰减：净赞数 + 回答数，按天数衰减
		order = "((upvote_count - downvote_count + answer_count) / POWER(DATEDIFF(NOW(), created_at) + 2, 1.5)) DESC, id DESC"
	}

	var list []model.Question
	err := base().Preload("Tags").Order(order).
		Limit(q.PageSize).Offset((q.Page - 1) * q.PageSize).Find(&list).Error
	return list, total, err
}

// IncrementView 浏览计数 +1（失败不影响主流程，由调用方忽略错误）。
func (r *QuestionRepo) IncrementView(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Model(&model.Question{}).Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
}

// Update 单事务完成：更新标题/正文、重建标签关联、修正新旧标签计数。
func (r *QuestionRepo) Update(ctx context.Context, id uint64, title, content string, tags []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var oldTagIDs []uint64
		if err := tx.Raw("SELECT tag_id FROM question_tags WHERE question_id = ?", id).Scan(&oldTagIDs).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Question{}).Where("id = ?", id).
			Updates(map[string]any{"title": title, "content": content}).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM question_tags WHERE question_id = ?", id).Error; err != nil {
			return err
		}
		tagRepo := &TagRepo{db: tx}
		tagIDs := make([]uint64, 0, len(tags))
		for _, name := range tags {
			t, err := tagRepo.getOrCreateTx(tx, name)
			if err != nil {
				return err
			}
			tagIDs = append(tagIDs, t.ID)
			if err := tx.Exec("INSERT IGNORE INTO question_tags (question_id, tag_id) VALUES (?, ?)", id, t.ID).Error; err != nil {
				return err
			}
		}
		return refreshTagCountsTx(tx, append(oldTagIDs, tagIDs...))
	})
}

// Delete 软删除问题，并回减提问者计数、修正标签计数。
func (r *QuestionRepo) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var q model.Question
		if err := tx.Clauses(locking()).Select("id", "user_id", "status").First(&q, id).Error; err != nil {
			return translateNotFound(err)
		}
		if q.Status == model.QuestionStatusDeleted {
			return apperr.ErrNotFound
		}
		res := tx.Model(&model.Question{}).Where("id = ?", id).
			Update("status", model.QuestionStatusDeleted)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return apperr.ErrConflict
		}
		if err := tx.Model(&model.User{}).Where("id = ? AND question_count > 0", q.UserID).
			UpdateColumn("question_count", gorm.Expr("question_count - 1")).Error; err != nil {
			return err
		}
		var tagIDs []uint64
		if err := tx.Raw("SELECT tag_id FROM question_tags WHERE question_id = ?", id).Scan(&tagIDs).Error; err != nil {
			return err
		}
		return refreshTagCountsTx(tx, tagIDs)
	})
}

// Close 将 open 状态的问题置为 closed；非 open 状态返回 409。
func (r *QuestionRepo) Close(ctx context.Context, id uint64) error {
	res := r.db.WithContext(ctx).Model(&model.Question{}).
		Where("id = ? AND status = ?", id, model.QuestionStatusOpen).
		Update("status", model.QuestionStatusClosed)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperr.With(apperr.ErrConflict, "问题不存在或未处于开放状态")
	}
	return nil
}

// AcceptAnswer 单事务切换采纳：取消旧采纳、标记新采纳、更新问题指针与活动时间、新作者声望 +15。
func (r *QuestionRepo) AcceptAnswer(ctx context.Context, questionID, answerID uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Answer{}).
			Where("question_id = ? AND is_accepted = 1", questionID).
			Update("is_accepted", false).Error; err != nil {
			return err
		}
		res := tx.Model(&model.Answer{}).Where("id = ?", answerID).Update("is_accepted", true)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return apperr.ErrNotFound
		}
		if err := tx.Model(&model.Question{}).Where("id = ?", questionID).
			Updates(map[string]any{
				"accepted_answer_id": answerID,
				"last_activity_at":   gorm.Expr("NOW(3)"),
			}).Error; err != nil {
			return err
		}
		return tx.Model(&model.User{}).
			Where("id = (SELECT user_id FROM answers WHERE id = ?)", answerID).
			UpdateColumn("reputation", gorm.Expr("reputation + ?", repAccepted)).Error
	})
}
