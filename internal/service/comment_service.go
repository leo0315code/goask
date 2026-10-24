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
	authorID   uint64 // 目标作者（通知接收人）
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

func (s *CommentService) Create(ctx context.Context, userID uint64, targetType string, targetID uint64, content string) (*CommentView, error) {
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
	if s.notifier != nil {
		s.notifier.NotifyComment(ctx, target.authorID, userID, target.questionID, targetType, content)
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
	if _, err := s.ensureTarget(ctx, targetType, targetID); err != nil {
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
