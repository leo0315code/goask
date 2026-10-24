package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/apperr"
	"github.com/leo0315code/goask/internal/pkg/httpx"
	"github.com/leo0315code/goask/internal/repository"
)

// CommentRepo 评论仓储接口（repository.CommentRepo 实现）。
type CommentRepo interface {
	Create(ctx context.Context, c *model.Comment) error
	GetByID(ctx context.Context, id uint64) (*model.Comment, error)
	GetRowByID(ctx context.Context, id uint64) (*repository.CommentRow, error)
	Delete(ctx context.Context, id uint64) error
	ListTree(ctx context.Context, targetType string, targetID uint64, page, pageSize int) ([]repository.CommentNode, int64, error)
}

type CommentService struct {
	comments  CommentRepo
	questions QuestionRepo
	answers   AnswerRepo
	users     UserRepo
	spam      SpamChecker
	notifier  Notifier
}

func NewCommentService(comments CommentRepo, questions QuestionRepo, answers AnswerRepo, users UserRepo, spam SpamChecker, notifier Notifier) *CommentService {
	return &CommentService{comments: comments, questions: questions, answers: answers, users: users, spam: spam, notifier: notifier}
}

func validateCommentContent(content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" || utf8.RuneCountInString(content) > 1000 {
		return "", apperr.With(apperr.ErrValidation, "评论不能为空且不超过 1000 字符")
	}
	return content, nil
}

// commentTarget 评论目标的归属信息。
type commentTarget struct {
	authorID   uint64 // 目标作者（根评论时通知接收人）
	questionID uint64 // 所属问题（跳转用）
}

// ensureTarget 校验评论目标存在且未删除，返回归属信息。
func (s *CommentService) ensureTarget(ctx context.Context, targetType string, targetID uint64) (commentTarget, error) {
	switch targetType {
	case model.TargetQuestion:
		q, err := s.questions.GetByID(ctx, targetID)
		if err != nil {
			return commentTarget{}, err
		}
		if q.Status == model.QuestionStatusDeleted {
			return commentTarget{}, apperr.ErrNotFound
		}
		return commentTarget{authorID: q.UserID, questionID: q.ID}, nil
	case model.TargetAnswer:
		a, err := s.answers.GetByID(ctx, targetID)
		if err != nil {
			return commentTarget{}, err
		}
		if a.Status != model.AnswerStatusPublished {
			return commentTarget{}, apperr.With(apperr.ErrNotFound, "回答不存在或已删除")
		}
		return commentTarget{authorID: a.UserID, questionID: a.QuestionID}, nil
	default:
		return commentTarget{}, apperr.With(apperr.ErrBadRequest, "未知的评论目标类型")
	}
}

// Create 发表评论；parentID > 0 时为楼中楼回复（拍平到根评论）。
// 根评论通知目标作者；回复通知被回复的评论作者。
func (s *CommentService) Create(ctx context.Context, userID uint64, targetType string, targetID uint64, content string, parentID uint64) (*repository.CommentRow, error) {
	content, err := validateCommentContent(content)
	if err != nil {
		return nil, err
	}
	target, err := s.ensureTarget(ctx, targetType, targetID)
	if err != nil {
		return nil, err
	}
	if err := s.spam.Check(ctx, userID, "comment"); err != nil {
		return nil, err
	}
	if err := s.spam.CheckDuplicate(ctx, userID, "comment", content); err != nil {
		return nil, err
	}

	var replyToUserID uint64
	if parentID > 0 {
		parent, err := s.comments.GetByID(ctx, parentID)
		if err != nil {
			return nil, apperr.With(apperr.ErrNotFound, "被回复的评论不存在")
		}
		if parent.Status != model.CommentStatusPublished {
			return nil, apperr.With(apperr.ErrNotFound, "被回复的评论不存在或已删除")
		}
		if parent.TargetType != targetType || parent.TargetID != targetID {
			return nil, apperr.With(apperr.ErrValidation, "被回复的评论不属于该内容")
		}
		if parent.ParentID > 0 {
			parentID = parent.ParentID // 只有一层嵌套：回复统一挂到根评论
		}
		replyToUserID = parent.UserID
	}

	c := &model.Comment{
		UserID:        userID,
		TargetType:    targetType,
		TargetID:      targetID,
		ParentID:      parentID,
		ReplyToUserID: replyToUserID,
		Content:       content,
		Status:        model.CommentStatusPublished,
	}
	if err := s.comments.Create(ctx, c); err != nil {
		return nil, err
	}

	if s.notifier != nil {
		if parentID > 0 {
			s.notifier.NotifyCommentReply(ctx, replyToUserID, userID, target.questionID, parentID, content)
		} else {
			s.notifier.NotifyComment(ctx, target.authorID, userID, target.questionID, targetType, content)
		}
	}

	return s.comments.GetRowByID(ctx, c.ID)
}

// Delete 删除评论（作者本人或管理员），软删除。
func (s *CommentService) Delete(ctx context.Context, userID, commentID uint64, isAdmin bool) error {
	c, err := s.comments.GetByID(ctx, commentID)
	if err != nil {
		return err
	}
	if c.Status != model.CommentStatusPublished {
		return apperr.With(apperr.ErrNotFound, "评论不存在或已删除")
	}
	if c.UserID != userID && !isAdmin {
		return apperr.With(apperr.ErrForbidden, "仅作者或管理员可删除评论")
	}
	return s.comments.Delete(ctx, commentID)
}

// List 楼中楼列表：根评论分页（旧→新），每条根评论挂全部回复。
func (s *CommentService) List(ctx context.Context, targetType string, targetID uint64, pq httpx.PageQuery) ([]repository.CommentNode, int64, error) {
	if targetType != model.TargetQuestion && targetType != model.TargetAnswer {
		return nil, 0, apperr.With(apperr.ErrBadRequest, "未知的评论目标类型")
	}
	if _, err := s.ensureTarget(ctx, targetType, targetID); err != nil {
		return nil, 0, err
	}
	return s.comments.ListTree(ctx, targetType, targetID, pq.Page, pq.PageSize)
}
