package service

import (
	"context"

	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/apperr"
	"github.com/leo0315code/goask/internal/repository"
)

type VoteService struct {
	votes    VoteRepo
	notifier Notifier
}

func NewVoteService(votes VoteRepo, notifier Notifier) *VoteService {
	return &VoteService{votes: votes, notifier: notifier}
}

// Vote 投票：value=1 赞 / -1 踩；同向再点撤销，反向切换（语义见 repository.Vote）。
func (s *VoteService) Vote(ctx context.Context, userID uint64, targetType string, targetID uint64, value int) (*repository.VoteState, error) {
	if targetType != model.TargetQuestion && targetType != model.TargetAnswer {
		return nil, apperr.With(apperr.ErrBadRequest, "未知的投票目标类型")
	}
	if value != 1 && value != -1 {
		return nil, apperr.With(apperr.ErrBadRequest, "value 取值只能为 1 或 -1")
	}
	if targetID == 0 {
		return nil, apperr.With(apperr.ErrBadRequest, "无效的目标 ID")
	}
	state, err := s.votes.Vote(ctx, userID, targetType, targetID, value)
	if err != nil {
		return nil, err
	}
	// 最终状态为赞同才通知（撤销/反对不通知）
	if state != nil && state.Value == 1 && s.notifier != nil {
		s.notifier.NotifyUpvote(ctx, userID, targetType, targetID)
	}
	return state, nil
}

// ViewerVote 查询当前用户对目标的投票值（0 表示未投票）。
func (s *VoteService) ViewerVote(ctx context.Context, userID uint64, targetType string, targetID uint64) (int, error) {
	if targetType != model.TargetQuestion && targetType != model.TargetAnswer {
		return 0, apperr.With(apperr.ErrBadRequest, "未知的投票目标类型")
	}
	return s.votes.GetViewerVote(ctx, userID, targetType, targetID)
}
