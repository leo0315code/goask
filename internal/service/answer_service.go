package service

import (
	"context"
	"strings"

	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/apperr"
	"github.com/leo0315code/goask/internal/pkg/httpx"
	"github.com/leo0315code/goask/internal/repository"
)

type AnswerService struct {
	answers   AnswerRepo
	questions QuestionRepo
	users     UserRepo
	spam      SpamChecker
}

func NewAnswerService(answers AnswerRepo, questions QuestionRepo, users UserRepo, spam SpamChecker) *AnswerService {
	return &AnswerService{answers: answers, questions: questions, users: users, spam: spam}
}

func validateContent(content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" || len(content) > 65536 {
		return "", apperr.With(apperr.ErrValidation, "内容不能为空且不超过 65536 字节")
	}
	return content, nil
}

func (s *AnswerService) buildView(ctx context.Context, a *model.Answer) (*AnswerView, error) {
	authors, err := buildAuthors(ctx, s.users, a.UserID)
	if err != nil {
		return nil, err
	}
	return &AnswerView{Answer: *a, Author: authors[a.UserID]}, nil
}

func (s *AnswerService) assemble(ctx context.Context, list []model.Answer) ([]AnswerView, error) {
	ids := make([]uint64, 0, len(list))
	for i := range list {
		ids = append(ids, list[i].UserID)
	}
	authors, err := buildAuthors(ctx, s.users, ids...)
	if err != nil {
		return nil, err
	}
	views := make([]AnswerView, 0, len(list))
	for i := range list {
		views = append(views, AnswerView{Answer: list[i], Author: authors[list[i].UserID]})
	}
	return views, nil
}

func (s *AnswerService) Create(ctx context.Context, userID, questionID uint64, content string) (*AnswerView, error) {
	content, err := validateContent(content)
	if err != nil {
		return nil, err
	}
	q, err := s.questions.GetByID(ctx, questionID)
	if err != nil {
		return nil, err
	}
	if q.Status == model.QuestionStatusDeleted {
		return nil, apperr.ErrNotFound
	}
	if q.Status == model.QuestionStatusClosed {
		return nil, apperr.With(apperr.ErrValidation, "问题已关闭，不能再回答")
	}
	if q.UserID == userID {
		return nil, apperr.With(apperr.ErrValidation, "不能回答自己的问题")
	}
	if err := s.spam.Check(ctx, userID, "answer"); err != nil {
		return nil, err
	}
	if err := s.spam.CheckDuplicate(ctx, userID, "answer", content); err != nil {
		return nil, err
	}
	a := &model.Answer{
		QuestionID: questionID,
		UserID:     userID,
		Content:    content,
		Status:     model.AnswerStatusPublished,
	}
	if err := s.answers.Create(ctx, a); err != nil {
		return nil, err
	}
	return s.buildView(ctx, a)
}

// List 某问题下的回答列表：sort = latest（默认，采纳置顶）/ votes。
func (s *AnswerService) List(ctx context.Context, questionID uint64, sort string, pq httpx.PageQuery) ([]AnswerView, int64, error) {
	q, err := s.questions.GetByID(ctx, questionID)
	if err != nil {
		return nil, 0, err
	}
	if q.Status == model.QuestionStatusDeleted {
		return nil, 0, apperr.ErrNotFound
	}
	list, total, err := s.answers.List(ctx, repository.AnswerListQuery{
		QuestionID: questionID, Sort: sort, Page: pq.Page, PageSize: pq.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	views, err := s.assemble(ctx, list)
	return views, total, err
}

func (s *AnswerService) Update(ctx context.Context, userID, answerID uint64, content string, isAdmin bool) error {
	content, err := validateContent(content)
	if err != nil {
		return err
	}
	a, err := s.answers.GetByID(ctx, answerID)
	if err != nil {
		return err
	}
	if a.Status != model.AnswerStatusPublished {
		return apperr.With(apperr.ErrNotFound, "回答不存在或已删除")
	}
	if a.UserID != userID && !isAdmin {
		return apperr.With(apperr.ErrForbidden, "仅作者可编辑回答")
	}
	return s.answers.Update(ctx, answerID, content)
}

func (s *AnswerService) Delete(ctx context.Context, userID, answerID uint64, isAdmin bool) error {
	a, err := s.answers.GetByID(ctx, answerID)
	if err != nil {
		return err
	}
	if a.Status != model.AnswerStatusPublished {
		return apperr.With(apperr.ErrNotFound, "回答不存在或已删除")
	}
	if a.UserID != userID && !isAdmin {
		return apperr.With(apperr.ErrForbidden, "仅作者或管理员可删除回答")
	}
	return s.answers.Delete(ctx, a)
}
