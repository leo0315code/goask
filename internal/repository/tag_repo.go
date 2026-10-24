package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/leo0315code/goask/internal/model"
)

type TagRepo struct {
	db *gorm.DB
}

func (r *TagRepo) GetByName(ctx context.Context, name string) (*model.Tag, error) {
	var t model.Tag
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&t).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &t, nil
}

// List 按问题数降序返回热门标签。
func (r *TagRepo) List(ctx context.Context, limit int) ([]model.Tag, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var tags []model.Tag
	if err := r.db.WithContext(ctx).Order("question_count DESC, id ASC").Limit(limit).Find(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}

// getOrCreateTx 在事务内按名取标签，不存在则创建（并发安全：INSERT IGNORE 后重查）。
func (r *TagRepo) getOrCreateTx(tx *gorm.DB, name string) (*model.Tag, error) {
	var t model.Tag
	if err := tx.Where("name = ?", name).First(&t).Error; err == nil {
		return &t, nil
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err := tx.Exec("INSERT IGNORE INTO tags (name) VALUES (?)", name).Error; err != nil {
		return nil, err
	}
	if err := tx.Where("name = ?", name).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// refreshTagCountsTx 用关联表重算指定标签的 question_count，修正冗余计数。
func refreshTagCountsTx(tx *gorm.DB, tagIDs []uint64) error {
	if len(tagIDs) == 0 {
		return nil
	}
	return tx.Exec(`UPDATE tags SET question_count = (
		SELECT COUNT(*) FROM question_tags qt WHERE qt.tag_id = tags.id
	) WHERE id IN ?`, tagIDs).Error
}
