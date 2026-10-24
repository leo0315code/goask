package repository

import (
	"context"

	"gorm.io/gorm"
)

// AdminRepo 管理后台专用查询。
type AdminRepo struct {
	db *gorm.DB
}

// AdminStats 概览统计。
type AdminStats struct {
	Users             int64 `json:"users"`
	Questions         int64 `json:"questions"`
	Answers           int64 `json:"answers"`
	Comments          int64 `json:"comments"`
	Votes             int64 `json:"votes"`
	NewUsersToday     int64 `json:"new_users_today"`
	NewQuestionsToday int64 `json:"new_questions_today"`
	NewAnswersToday   int64 `json:"new_answers_today"`
}

func (r *AdminRepo) Stats(ctx context.Context) (*AdminStats, error) {
	s := &AdminStats{}
	db := r.db.WithContext(ctx)
	if err := db.Table("users").Count(&s.Users).Error; err != nil {
		return nil, err
	}
	if err := db.Table("questions").Where("status <> 'deleted'").Count(&s.Questions).Error; err != nil {
		return nil, err
	}
	if err := db.Table("answers").Where("status = 'published'").Count(&s.Answers).Error; err != nil {
		return nil, err
	}
	if err := db.Table("comments").Where("status = 'published'").Count(&s.Comments).Error; err != nil {
		return nil, err
	}
	if err := db.Table("votes").Count(&s.Votes).Error; err != nil {
		return nil, err
	}
	if err := db.Table("users").Where("created_at >= CURDATE()").Count(&s.NewUsersToday).Error; err != nil {
		return nil, err
	}
	if err := db.Table("questions").Where("created_at >= CURDATE() AND status <> 'deleted'").Count(&s.NewQuestionsToday).Error; err != nil {
		return nil, err
	}
	if err := db.Table("answers").Where("created_at >= CURDATE() AND status = 'published'").Count(&s.NewAnswersToday).Error; err != nil {
		return nil, err
	}
	return s, nil
}

// AdminQuestionView 问题管理列表视图（含作者名，可含已删除）。
type AdminQuestionView struct {
	ID          uint64 `json:"id"`
	UserID      uint64 `json:"user_id"`
	Title       string `json:"title"`
	AuthorName  string `json:"author_name"`
	Status      string `json:"status"`
	AnswerCount int    `json:"answer_count"`
	ViewCount   int    `json:"view_count"`
	UpvoteCount int    `json:"upvote_count"`
	CreatedAt   string `json:"created_at"`
}

// QuestionList 问题管理列表：标题模糊搜索 + 状态过滤（all/open/closed/deleted）。
func (r *AdminRepo) QuestionList(ctx context.Context, q, status string, page, pageSize int) ([]AdminQuestionView, int64, error) {
	base := func() *gorm.DB {
		db := r.db.WithContext(ctx).Table("questions qu")
		if q != "" {
			db = db.Where("qu.title LIKE ?", "%"+q+"%")
		}
		switch status {
		case "open", "closed", "deleted":
			db = db.Where("qu.status = ?", status)
		}
		return db
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []AdminQuestionView
	err := base().
		Select("qu.id, qu.user_id, qu.title, qu.status, qu.answer_count, qu.view_count, qu.upvote_count, qu.created_at, u.username AS author_name").
		Joins("LEFT JOIN users u ON u.id = qu.user_id").
		Order("qu.id DESC").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Scan(&list).Error
	return list, total, err
}

// AdminCommentView 评论管理列表视图。
type AdminCommentView struct {
	ID         uint64 `json:"id"`
	AuthorName string `json:"author_name"`
	TargetType string `json:"target_type"`
	TargetID   uint64 `json:"target_id"`
	QuestionID uint64 `json:"question_id"`
	Content    string `json:"content"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
}

// CommentList 评论管理列表：内容模糊搜索（含已删除）。
func (r *AdminRepo) CommentList(ctx context.Context, q string, page, pageSize int) ([]AdminCommentView, int64, error) {
	base := func() *gorm.DB {
		db := r.db.WithContext(ctx).Table("comments c")
		if q != "" {
			db = db.Where("c.content LIKE ?", "%"+q+"%")
		}
		return db
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []AdminCommentView
	err := base().
		Select("c.id, c.target_type, c.target_id, c.content, c.status, c.created_at, u.username AS author_name, " +
			"CASE c.target_type WHEN 'question' THEN c.target_id ELSE IFNULL(a.question_id, 0) END AS question_id").
		Joins("LEFT JOIN users u ON u.id = c.user_id").
		Joins("LEFT JOIN answers a ON c.target_type = 'answer' AND a.id = c.target_id").
		Order("c.id DESC").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Scan(&list).Error
	return list, total, err
}
