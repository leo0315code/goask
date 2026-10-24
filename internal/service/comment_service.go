package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/apperr"
	"github.com/leo0315code/goask/internal/pkg/httpx"
)

type CommentService struct {
	comments  CommentRepo
	questions QuestionRepo
	answers   AnswerRepo
	users     UserRepo
}

func NewCommentService(comments CommentRepo, questions QuestionRepo, answers AnswerRepo, users UserRepo) *CommentService {
	return &CommentService{comments: comments, questions: questions, answers: answers, users: users}
}

func validateCommentContent(content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" || utf8.RuneCountInString(content) > 1000 {
		return "", apperr.With(apperr.ErrValidation, "评论不能为空且不超过 1000 字符")
	}
	return content, nil
}

// ensureTarget 校验评论目标存在且未删除。
func (s *CommentService) ensureTarget(ctx context.Context, targetType string, targetID uint64) error {
	switch targetType {
	case model.TargetQuestion:
		q, err := s.questions.GetByID(ctx, targetID)
		if err != nil {
			return err
		}
		if q.Status == model.QuestionStatusDeleted {
			return apperr.ErrNotFound
		}
		return nil
	case model.TargetAnswer:
		a, err := s.answers.GetByID(ctx, targetID)
		if err != nil {
			return err
		}
		if a.Status != model.AnswerStatusPublished {
			return apperr.With(apperr.ErrNotFound, "回答不存在或已删除")
		}
		return nil
	default:
		return apperr.With(apperr.ErrBadRequest, "未知的评论目标类型")
	}
}

func (s *CommentService) Create(ctx context.Context, userID uint64, targetType string, targetID uint64, content string) (*CommentView, error) {
	content, err := validateCommentContent(content)
	if err != nil {
		return nil, err
	}
	if err := s.ensureTarget(ctx, targetType, targetID); err != nil {
		return nil, err
	}
	c := &model.Comment{
		UserID:     userID,
		TargetType: targetType,
		TargetID:   targetID,
		Content:    content,
		Status:     model.CommentStatusPublished,
	}
	if err := s.comments.Create(ctx, c); err != nil {
		return nil, err
	}
	authors, err := buildAuthors(ctx, s.users, c.UserID)
	if err != nil {
		return nil, err
	}
	return &CommentView{Comment: *c, Author: authors[c.UserID]}, nil
}

func (s *CommentService) List(ctx context.Context, targetType string, targetID uint64, pq httpx.PageQuery) ([]CommentView, int64, error) {
	if targetType != model.TargetQuestion && targetType != model.TargetAnswer {
		return nil, 0, apperr.With(apperr.ErrBadRequest, "未知的评论目标类型")
	}
	if err := s.ensureTarget(ctx, targetType, targetID); err != nil {
		return nil, 0, err
	}
	list, total, err := s.comments.ListByTarget(ctx, targetType, targetID, pq.Page, pq.PageSize)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uint64, 0, len(list))
	for i := range list {
		ids = append(ids, list[i].UserID)
	}
	authors, err := buildAuthors(ctx, s.users, ids...)
	if err != nil {
		return nil, 0, err
	}
	views := make([]CommentView, 0, len(list))
	for i := range list {
		views = append(views, CommentView{Comment: list[i], Author: authors[list[i].UserID]})
	}
	return views, total, nil
}
