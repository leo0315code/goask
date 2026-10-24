package service

import (
	"context"

	"github.com/leo0315code/goask/internal/repository"
)

// AdminRepo 管理后台仓储接口（repository.AdminRepo 实现）。
type AdminRepo interface {
	Stats(ctx context.Context) (*repository.AdminStats, error)
	QuestionList(ctx context.Context, q, status string, page, pageSize int) ([]repository.AdminQuestionView, int64, error)
	CommentList(ctx context.Context, q string, page, pageSize int) ([]repository.AdminCommentView, int64, error)
}

type AdminService struct {
	repo AdminRepo
}

func NewAdminService(repo AdminRepo) *AdminService {
	return &AdminService{repo: repo}
}

func (s *AdminService) Stats(ctx context.Context) (*repository.AdminStats, error) {
	return s.repo.Stats(ctx)
}

func (s *AdminService) Questions(ctx context.Context, q, status string, page, pageSize int) ([]repository.AdminQuestionView, int64, error) {
	return s.repo.QuestionList(ctx, q, status, page, pageSize)
}

func (s *AdminService) Comments(ctx context.Context, q string, page, pageSize int) ([]repository.AdminCommentView, int64, error) {
	return s.repo.CommentList(ctx, q, page, pageSize)
}
