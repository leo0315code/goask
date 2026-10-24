package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/apperr"
	"github.com/leo0315code/goask/internal/repository"
)

// 本文件定义 service 依赖的仓储接口（消费方接口，repository 结构体隐式实现），
// 单元测试时可用内存假实现替换。

type UserRepo interface {
	Create(ctx context.Context, u *model.User) error
	GetByID(ctx context.Context, id uint64) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByEmailOrUsername(ctx context.Context, account string) (*model.User, error)
	UpdateProfile(ctx context.Context, id uint64, bio, avatarURL string) error
	UpdatePassword(ctx context.Context, id uint64, hash string) error
	SetStatus(ctx context.Context, id uint64, status string) error
	List(ctx context.Context, q string, page, pageSize int) ([]model.User, int64, error)
	GetByIDs(ctx context.Context, ids []uint64) ([]model.User, error)
}

type QuestionRepo interface {
	Create(ctx context.Context, input repository.CreateQuestionInput) error
	GetByID(ctx context.Context, id uint64) (*model.Question, error)
	List(ctx context.Context, query repository.QuestionListQuery) ([]model.Question, int64, error)
	Update(ctx context.Context, id uint64, title, content string, tags []string) error
	Delete(ctx context.Context, id uint64) error
	Close(ctx context.Context, id uint64) error
	IncrementView(ctx context.Context, id uint64) error
	AcceptAnswer(ctx context.Context, questionID, answerID uint64) error
}

type AnswerRepo interface {
	Create(ctx context.Context, a *model.Answer) error
	GetByID(ctx context.Context, id uint64) (*model.Answer, error)
	List(ctx context.Context, query repository.AnswerListQuery) ([]model.Answer, int64, error)
	Update(ctx context.Context, id uint64, content string) error
	Delete(ctx context.Context, a *model.Answer) error
}

type VoteRepo interface {
	Vote(ctx context.Context, userID uint64, targetType string, targetID uint64, value int) (*repository.VoteState, error)
	GetViewerVote(ctx context.Context, userID uint64, targetType string, targetID uint64) (int, error)
}

type CommentRepo interface {
	Create(ctx context.Context, c *model.Comment) error
	GetByID(ctx context.Context, id uint64) (*model.Comment, error)
	Delete(ctx context.Context, id uint64) error
	ListByTarget(ctx context.Context, targetType string, targetID uint64, page, pageSize int) ([]model.Comment, int64, error)
}

type TagRepo interface {
	GetByName(ctx context.Context, name string) (*model.Tag, error)
	List(ctx context.Context, limit int) ([]model.Tag, error)
}

// ---- 视图对象（service 输出的业务视图，handler 直接序列化）----

type QuestionView struct {
	model.Question
	Author     model.Author `json:"author"`
	ViewerVote int          `json:"viewer_vote"`
}

type AnswerView struct {
	model.Answer
	Author model.Author `json:"author"`
}

type CommentView struct {
	model.Comment
	Author model.Author `json:"author"`
}

// buildAuthors 批量组装作者摘要，自动去重、忽略零值。
func buildAuthors(ctx context.Context, users UserRepo, ids ...uint64) (map[uint64]model.Author, error) {
	out := make(map[uint64]model.Author)
	uniq := make([]uint64, 0, len(ids))
	seen := make(map[uint64]bool, len(ids))
	for _, id := range ids {
		if id != 0 && !seen[id] {
			seen[id] = true
			uniq = append(uniq, id)
		}
	}
	if len(uniq) == 0 {
		return out, nil
	}
	list, err := users.GetByIDs(ctx, uniq)
	if err != nil {
		return nil, err
	}
	for _, u := range list {
		out[u.ID] = model.Author{ID: u.ID, Username: u.Username, AvatarURL: u.AvatarURL, Role: u.Role}
	}
	return out, nil
}

// normalizeTags 标签规范化：去空白、转小写、去重；最多 5 个、每个不超过 32 字符。
func normalizeTags(tags []string) ([]string, error) {
	out := make([]string, 0, len(tags))
	seen := make(map[string]bool, len(tags))
	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if utf8.RuneCountInString(t) > 32 {
			return nil, apperr.With(apperr.ErrValidation, "标签过长（最多 32 字符）")
		}
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	if len(out) > 5 {
		return nil, apperr.With(apperr.ErrValidation, "标签最多 5 个")
	}
	return out, nil
}
