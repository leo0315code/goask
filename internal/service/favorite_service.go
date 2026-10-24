package service

import (
	"context"

	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/apperr"
	"github.com/leo0315code/goask/internal/repository"
)

// FavoriteRepo 收藏仓储接口（repository.FavoriteRepo 实现）。
type FavoriteRepo interface {
	IsFavorited(ctx context.Context, userID, questionID uint64) (bool, error)
	Toggle(ctx context.Context, userID, questionID uint64) (bool, error)
	ListByUser(ctx context.Context, userID uint64, page, pageSize int) ([]repository.FavoriteItemView, int64, error)
	AnswerListByUser(ctx context.Context, userID uint64, page, pageSize int) ([]repository.AnswerUserView, int64, error)
}

type FavoriteService struct {
	favorites FavoriteRepo
	questions QuestionRepo
	users     UserRepo
}

func NewFavoriteService(favorites FavoriteRepo, questions QuestionRepo, users UserRepo) *FavoriteService {
	return &FavoriteService{favorites: favorites, questions: questions, users: users}
}

// Toggle 收藏/取消收藏问题（问题需存在且未删除）。
func (s *FavoriteService) Toggle(ctx context.Context, userID, questionID uint64) (bool, error) {
	q, err := s.questions.GetByID(ctx, questionID)
	if err != nil {
		return false, err
	}
	if q.Status == model.QuestionStatusDeleted {
		return false, apperr.ErrNotFound
	}
	return s.favorites.Toggle(ctx, userID, questionID)
}

func (s *FavoriteService) IsFavorited(ctx context.Context, userID, questionID uint64) (bool, error) {
	return s.favorites.IsFavorited(ctx, userID, questionID)
}

// ListMine 我的收藏列表。
func (s *FavoriteService) ListMine(ctx context.Context, userID uint64, page, pageSize int) ([]repository.FavoriteItemView, int64, error) {
	return s.favorites.ListByUser(ctx, userID, page, pageSize)
}

// AnswersByUser 某用户的回答列表（按用户名定位）。
func (s *FavoriteService) AnswersByUser(ctx context.Context, username string, page, pageSize int) ([]repository.AnswerUserView, int64, error) {
	u, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return nil, 0, err
	}
	return s.favorites.AnswerListByUser(ctx, u.ID, page, pageSize)
}
