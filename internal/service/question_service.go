package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/apperr"
	"github.com/leo0315code/goask/internal/pkg/httpx"
	"github.com/leo0315code/goask/internal/repository"
)

type QuestionService struct {
	questions QuestionRepo
	answers   AnswerRepo
	users     UserRepo
	tags      TagRepo
	votes     VoteRepo
}

func NewQuestionService(questions QuestionRepo, answers AnswerRepo, users UserRepo, tags TagRepo, votes VoteRepo) *QuestionService {
	return &QuestionService{questions: questions, answers: answers, users: users, tags: tags, votes: votes}
}

func validateTitleContent(title, content string) (string, string, error) {
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if n := utf8.RuneCountInString(title); n < 10 || n > 150 {
		return "", "", apperr.With(apperr.ErrValidation, "标题长度需为 10-150 个字符")
	}
	if content == "" || len(content) > 65536 {
		return "", "", apperr.With(apperr.ErrValidation, "正文不能为空且不超过 65536 字节")
	}
	return title, content, nil
}

func (s *QuestionService) Create(ctx context.Context, userID uint64, title, content string, tags []string) (*QuestionView, error) {
	title, content, err := validateTitleContent(title, content)
	if err != nil {
		return nil, err
	}
	tagNames, err := normalizeTags(tags)
	if err != nil {
		return nil, err
	}
	q := &model.Question{
		UserID:         userID,
		Title:          title,
		Content:        content,
		Status:         model.QuestionStatusOpen,
		LastActivityAt: time.Now().UTC(),
	}
	if err := s.questions.Create(ctx, repository.CreateQuestionInput{Question: q, Tags: tagNames}); err != nil {
		return nil, err
	}
	// 回读以填充标签与数据库生成的时间戳
	if q, err = s.questions.GetByID(ctx, q.ID); err != nil {
		return nil, err
	}
	return s.buildView(ctx, q, 0)
}

func (s *QuestionService) buildView(ctx context.Context, q *model.Question, viewerID uint64) (*QuestionView, error) {
	authors, err := buildAuthors(ctx, s.users, q.UserID)
	if err != nil {
		return nil, err
	}
	v := &QuestionView{Question: *q, Author: authors[q.UserID]}
	if viewerID > 0 {
		vv, err := s.votes.GetViewerVote(ctx, viewerID, model.TargetQuestion, q.ID)
		if err != nil {
			return nil, err
		}
		v.ViewerVote = vv
	}
	return v, nil
}

func (s *QuestionService) assemble(ctx context.Context, list []model.Question) ([]QuestionView, error) {
	ids := make([]uint64, 0, len(list))
	for i := range list {
		ids = append(ids, list[i].UserID)
	}
	authors, err := buildAuthors(ctx, s.users, ids...)
	if err != nil {
		return nil, err
	}
	views := make([]QuestionView, 0, len(list))
	for i := range list {
		views = append(views, QuestionView{Question: list[i], Author: authors[list[i].UserID]})
	}
	return views, nil
}

func (s *QuestionService) Get(ctx context.Context, id, viewerID uint64) (*QuestionView, error) {
	// 先计数再查询，返回值包含本次浏览；计数失败不影响详情展示
	_ = s.questions.IncrementView(ctx, id)
	q, err := s.questions.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if q.Status == model.QuestionStatusDeleted {
		return nil, apperr.ErrNotFound
	}
	return s.buildView(ctx, q, viewerID)
}

// List 问题列表：sort = latest（默认）/ hot / unanswered；tag 按标签名过滤。
func (s *QuestionService) List(ctx context.Context, sort, tag string, pq httpx.PageQuery) ([]QuestionView, int64, error) {
	var tagID uint64
	if tag != "" {
		t, err := s.tags.GetByName(ctx, strings.ToLower(strings.TrimSpace(tag)))
		if err != nil {
			if errors.Is(err, apperr.ErrNotFound) {
				return nil, 0, apperr.With(apperr.ErrNotFound, "标签不存在")
			}
			return nil, 0, err
		}
		tagID = t.ID
	}
	list, total, err := s.questions.List(ctx, repository.QuestionListQuery{
		Sort: sort, TagID: tagID, Page: pq.Page, PageSize: pq.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	views, err := s.assemble(ctx, list)
	return views, total, err
}

// Search 中文全文检索（标题 + 正文），按相关度排序。
func (s *QuestionService) Search(ctx context.Context, keyword string, pq httpx.PageQuery) ([]QuestionView, int64, error) {
	keyword = strings.TrimSpace(keyword)
	if utf8.RuneCountInString(keyword) < 2 {
		return nil, 0, apperr.With(apperr.ErrValidation, "关键词至少 2 个字符")
	}
	list, total, err := s.questions.List(ctx, repository.QuestionListQuery{
		Search: keyword, Page: pq.Page, PageSize: pq.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	views, err := s.assemble(ctx, list)
	return views, total, err
}

func (s *QuestionService) Update(ctx context.Context, userID, questionID uint64, title, content string, tags []string, isAdmin bool) error {
	q, err := s.questions.GetByID(ctx, questionID)
	if err != nil {
		return err
	}
	if q.Status == model.QuestionStatusDeleted {
		return apperr.ErrNotFound
	}
	if q.UserID != userID && !isAdmin {
		return apperr.With(apperr.ErrForbidden, "仅作者可编辑问题")
	}
	title, content, err = validateTitleContent(title, content)
	if err != nil {
		return err
	}
	tagNames, err := normalizeTags(tags)
	if err != nil {
		return err
	}
	return s.questions.Update(ctx, questionID, title, content, tagNames)
}

func (s *QuestionService) Delete(ctx context.Context, userID, questionID uint64, isAdmin bool) error {
	q, err := s.questions.GetByID(ctx, questionID)
	if err != nil {
		return err
	}
	if q.Status == model.QuestionStatusDeleted {
		return apperr.ErrNotFound
	}
	if q.UserID != userID && !isAdmin {
		return apperr.With(apperr.ErrForbidden, "仅作者或管理员可删除问题")
	}
	return s.questions.Delete(ctx, questionID)
}

func (s *QuestionService) Close(ctx context.Context, userID, questionID uint64, isAdmin bool) error {
	q, err := s.questions.GetByID(ctx, questionID)
	if err != nil {
		return err
	}
	if q.Status == model.QuestionStatusDeleted {
		return apperr.ErrNotFound
	}
	if q.UserID != userID && !isAdmin {
		return apperr.With(apperr.ErrForbidden, "仅作者或管理员可关闭问题")
	}
	return s.questions.Close(ctx, questionID)
}

func (s *QuestionService) AcceptAnswer(ctx context.Context, userID, questionID, answerID uint64, isAdmin bool) error {
	q, err := s.questions.GetByID(ctx, questionID)
	if err != nil {
		return err
	}
	if q.Status == model.QuestionStatusDeleted {
		return apperr.ErrNotFound
	}
	if q.UserID != userID && !isAdmin {
		return apperr.With(apperr.ErrForbidden, "仅提问者可采纳回答")
	}
	if q.Status == model.QuestionStatusClosed {
		return apperr.With(apperr.ErrValidation, "问题已关闭")
	}
	a, err := s.answers.GetByID(ctx, answerID)
	if err != nil {
		return err
	}
	if a.QuestionID != questionID || a.Status != model.AnswerStatusPublished {
		return apperr.With(apperr.ErrNotFound, "回答不存在或不属于该问题")
	}
	return s.questions.AcceptAnswer(ctx, questionID, answerID)
}
