package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/apperr"
)

type CommentRepo struct {
	db *gorm.DB
}

// CommentRow 评论视图（联出作者与被回复人昵称）。
type CommentRow struct {
	model.Comment
	AuthorName  string `json:"author_name"`
	AuthorRole  string `json:"author_role"`
	ReplyToName string `json:"reply_to_name"`
}

// CommentNode 楼中楼节点：根评论 + 其全部回复。
type CommentNode struct {
	CommentRow
	Replies []CommentNode `json:"replies"`
}

const commentRowSelect = "c.*, u.username AS author_name, u.role AS author_role, ru.username AS reply_to_name"

func commentRowBase(db *gorm.DB) *gorm.DB {
	return db.
		Joins("LEFT JOIN users u ON u.id = c.user_id").
		Joins("LEFT JOIN users ru ON ru.id = c.reply_to_user_id")
}

func (r *CommentRepo) GetByID(ctx context.Context, id uint64) (*model.Comment, error) {
	var c model.Comment
	if err := r.db.WithContext(ctx).First(&c, id).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &c, nil
}

// GetRowByID 取单条评论视图（Create 后回显用）。
func (r *CommentRepo) GetRowByID(ctx context.Context, id uint64) (*CommentRow, error) {
	var row CommentRow
	err := commentRowBase(r.db.WithContext(ctx).Table("comments c").
		Select(commentRowSelect).Where("c.id = ?", id)).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, apperr.ErrNotFound
	}
	return &row, nil
}

func (r *CommentRepo) Create(ctx context.Context, c *model.Comment) error {
	return r.db.WithContext(ctx).Create(c).Error
}

// Delete 软删除评论。
func (r *CommentRepo) Delete(ctx context.Context, id uint64) error {
	res := r.db.WithContext(ctx).Model(&model.Comment{}).
		Where("id = ? AND status = ?", id, model.CommentStatusPublished).
		Update("status", model.CommentStatusDeleted)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperr.With(apperr.ErrNotFound, "评论不存在或已删除")
	}
	return nil
}

// ListTree 返回楼中楼结构：根评论分页（旧→新），每条根评论挂全部回复。
// total 为根评论数。
func (r *CommentRepo) ListTree(ctx context.Context, targetType string, targetID uint64, page, pageSize int) ([]CommentNode, int64, error) {
	rootBase := func() *gorm.DB {
		return r.db.WithContext(ctx).Table("comments c").
			Where("c.target_type = ? AND c.target_id = ? AND c.parent_id = 0 AND c.status = ?",
				targetType, targetID, model.CommentStatusPublished)
	}
	var total int64
	if err := rootBase().Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var roots []CommentRow
	if err := commentRowBase(rootBase().
		Select(commentRowSelect).Order("c.id ASC").
		Limit(pageSize).Offset((page - 1) * pageSize)).Scan(&roots).Error; err != nil {
		return nil, 0, err
	}
	if len(roots) == 0 {
		return []CommentNode{}, 0, nil
	}

	rootIDs := make([]uint64, 0, len(roots))
	for i := range roots {
		rootIDs = append(rootIDs, roots[i].ID)
	}

	var replies []CommentRow
	if err := commentRowBase(r.db.WithContext(ctx).Table("comments c").
		Select(commentRowSelect).
		Where("c.parent_id IN ? AND c.status = ?", rootIDs, model.CommentStatusPublished).
		Order("c.id ASC")).Scan(&replies).Error; err != nil {
		return nil, 0, err
	}

	byRoot := make(map[uint64][]CommentNode, len(roots))
	for i := range replies {
		byRoot[replies[i].ParentID] = append(byRoot[replies[i].ParentID], CommentNode{CommentRow: replies[i], Replies: []CommentNode{}})
	}

	nodes := make([]CommentNode, 0, len(roots))
	for i := range roots {
		n := CommentNode{CommentRow: roots[i], Replies: byRoot[roots[i].ID]}
		if n.Replies == nil {
			n.Replies = []CommentNode{}
		}
		nodes = append(nodes, n)
	}
	return nodes, total, nil
}
