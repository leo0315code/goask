package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/apperr"
)

// 声望规则：撤销时按相反方向回退（见 docs/database.md）。
const (
	repAnswerUpvote   = 10
	repQuestionUpvote = 5
	repDownvoted      = -2
	repAccepted       = 15
)

type VoteRepo struct {
	db *gorm.DB
}

// VoteState 投票后的结果状态；value=0 表示该票已被撤销。
type VoteState struct {
	Value         int `json:"value"`
	UpvoteCount   int `json:"upvote_count"`
	DownvoteCount int `json:"downvote_count"`
}

func upvoteRep(targetType string) int {
	if targetType == model.TargetAnswer {
		return repAnswerUpvote
	}
	return repQuestionUpvote
}

// Vote 在单事务内完成「投票 / 同向撤销 / 反向切换」三种语义，
// 并同步目标表计数器与作者声望。流程：
//  1. FOR UPDATE 锁定目标行，串行化同一目标的并发投票；
//  2. 读取（或创建/更新/删除）投票人唯一票行（唯一索引 user+target）；
//  3. 计算计数器与声望增量并落库。
func (r *VoteRepo) Vote(ctx context.Context, userID uint64, targetType string, targetID uint64, value int) (*VoteState, error) {
	var state VoteState
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var authorID uint64
		var up, down int
		switch targetType {
		case model.TargetQuestion:
			var q model.Question
			if err := tx.Clauses(locking()).Select("id", "user_id", "upvote_count", "downvote_count").
				First(&q, targetID).Error; err != nil {
				return translateNotFound(err)
			}
			authorID, up, down = q.UserID, q.UpvoteCount, q.DownvoteCount
		case model.TargetAnswer:
			var a model.Answer
			if err := tx.Clauses(locking()).Select("id", "user_id", "upvote_count", "downvote_count").
				First(&a, targetID).Error; err != nil {
				return translateNotFound(err)
			}
			authorID, up, down = a.UserID, a.UpvoteCount, a.DownvoteCount
		default:
			return apperr.With(apperr.ErrBadRequest, "未知的投票目标类型")
		}

		if authorID == userID {
			return apperr.With(apperr.ErrValidation, "不能给自己的内容投票")
		}

		var existing model.Vote
		err := tx.Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
			First(&existing).Error
		has := err == nil
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		repDelta := 0
		viewerValue := value
		switch {
		case !has:
			v := model.Vote{UserID: userID, TargetType: targetType, TargetID: targetID, Value: value}
			if err := tx.Create(&v).Error; err != nil {
				return err
			}
			if value == 1 {
				up++
				repDelta += upvoteRep(targetType)
			} else {
				down++
				repDelta += repDownvoted
			}
		case existing.Value == value:
			if err := tx.Delete(&model.Vote{}, existing.ID).Error; err != nil {
				return err
			}
			viewerValue = 0
			if value == 1 {
				up--
				repDelta -= upvoteRep(targetType)
			} else {
				down--
				repDelta -= repDownvoted
			}
		default:
			if err := tx.Model(&model.Vote{}).Where("id = ?", existing.ID).Update("value", value).Error; err != nil {
				return err
			}
			if value == 1 {
				up++
				down--
				repDelta += upvoteRep(targetType) - repDownvoted
			} else {
				up--
				down++
				repDelta -= upvoteRep(targetType) - repDownvoted
			}
		}

		switch targetType {
		case model.TargetQuestion:
			if err := tx.Model(&model.Question{}).Where("id = ?", targetID).
				Updates(map[string]any{"upvote_count": up, "downvote_count": down}).Error; err != nil {
				return err
			}
		case model.TargetAnswer:
			if err := tx.Model(&model.Answer{}).Where("id = ?", targetID).
				Updates(map[string]any{"upvote_count": up, "downvote_count": down}).Error; err != nil {
				return err
			}
		}

		if repDelta != 0 {
			if err := tx.Model(&model.User{}).Where("id = ?", authorID).
				UpdateColumn("reputation", gorm.Expr("reputation + ?", repDelta)).Error; err != nil {
				return err
			}
		}

		state = VoteState{Value: viewerValue, UpvoteCount: up, DownvoteCount: down}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// GetViewerVote 返回用户对目标的当前投票值（0 表示未投票）。
func (r *VoteRepo) GetViewerVote(ctx context.Context, userID uint64, targetType string, targetID uint64) (int, error) {
	var v model.Vote
	err := r.db.WithContext(ctx).Select("value").
		Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
		First(&v).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return v.Value, nil
}
