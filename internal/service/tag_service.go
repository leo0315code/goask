package service

import (
	"context"

	"github.com/leo0315code/goask/internal/model"
)

type TagService struct {
	tags TagRepo
}

func NewTagService(tags TagRepo) *TagService {
	return &TagService{tags: tags}
}

// List 热门标签列表，按问题数降序。
func (s *TagService) List(ctx context.Context, limit int) ([]model.Tag, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.tags.List(ctx, limit)
}
