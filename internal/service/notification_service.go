package service

import (
	"context"
	"encoding/json"

	"github.com/leo0315code/goask/internal/hub"
	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/httpx"
	"github.com/leo0315code/goask/internal/repository"
)

// NotificationRepo 通知仓储接口（repository.NotificationRepo 实现）。
type NotificationRepo interface {
	Create(ctx context.Context, n *model.Notification) error
	List(ctx context.Context, q repository.NotificationListQuery) ([]model.NotificationView, int64, int64, error)
	MarkRead(ctx context.Context, id, userID uint64) error
	MarkAllRead(ctx context.Context, userID uint64) error
	GetContentAuthor(ctx context.Context, targetType string, targetID uint64) (authorID, questionID uint64, title string, err error)
}

// Notifier 业务事件通知接口，由 NotificationService 实现；
// 各业务 service 在动作成功后调用，通知失败不影响主流程。
type Notifier interface {
	NotifyAnswerCreated(ctx context.Context, recipientID, actorID, questionID uint64, questionTitle, excerpt string)
	NotifyComment(ctx context.Context, recipientID, actorID, questionID uint64, targetType, excerpt string)
	NotifyCommentReply(ctx context.Context, recipientID, actorID, questionID, parentCommentID uint64, excerpt string)
	NotifyAccepted(ctx context.Context, recipientID, actorID, answerID, questionID uint64, questionTitle string)
	NotifyUpvote(ctx context.Context, actorID uint64, targetType string, targetID uint64)
}

type NotificationService struct {
	repo NotificationRepo
	hub  *hub.Hub // SSE 实时推送；为 nil 时退化为纯轮询
}

func NewNotificationService(repo NotificationRepo, h *hub.Hub) *NotificationService {
	return &NotificationService{repo: repo, hub: h}
}

// ---- Notifier 实现（自动跳过自己通知自己；失败不影响主流程）----

func (s *NotificationService) create(ctx context.Context, recipientID, actorID uint64, notifType string, targetType string, targetID, questionID uint64, content string) {
	if s == nil || s.repo == nil || recipientID == 0 || recipientID == actorID {
		return
	}
	n := &model.Notification{
		UserID:     recipientID,
		Type:       notifType,
		ActorID:    actorID,
		TargetType: targetType,
		TargetID:   targetID,
		QuestionID: questionID,
		Content:    content,
	}
	if err := s.repo.Create(ctx, n); err != nil {
		return
	}
	// 实时推送给在线用户（SSE）；客户端收到后刷新未读数
	if s.hub != nil {
		if payload, err := json.Marshal(map[string]any{
			"type": notifType, "content": content, "question_id": questionID,
		}); err == nil {
			s.hub.Publish(recipientID, string(payload))
		}
	}
}

func (s *NotificationService) NotifyAnswerCreated(ctx context.Context, recipientID, actorID, questionID uint64, questionTitle, _ string) {
	s.create(ctx, recipientID, actorID, model.NotifAnswer, model.TargetQuestion, questionID, questionID, firstRunes(questionTitle, 80))
}

func (s *NotificationService) NotifyComment(ctx context.Context, recipientID, actorID, questionID uint64, targetType, excerpt string) {
	notifType := model.NotifCommentQuestion
	notifTarget := model.TargetQuestion
	targetID := questionID
	if targetType == model.TargetAnswer {
		notifType = model.NotifCommentAnswer
		notifTarget = model.TargetAnswer
	}
	s.create(ctx, recipientID, actorID, notifType, notifTarget, targetID, questionID, firstRunes(excerpt, 80))
}

func (s *NotificationService) NotifyCommentReply(ctx context.Context, recipientID, actorID, questionID, parentCommentID uint64, excerpt string) {
	s.create(ctx, recipientID, actorID, model.NotifCommentReply, "comment", parentCommentID, questionID, firstRunes(excerpt, 80))
}

func (s *NotificationService) NotifyAccepted(ctx context.Context, recipientID, actorID, answerID, questionID uint64, questionTitle string) {
	s.create(ctx, recipientID, actorID, model.NotifAccept, model.TargetAnswer, answerID, questionID, firstRunes(questionTitle, 80))
}

// NotifyUpvote 赞同通知：查找内容作者并排除自己；撤销/反对不通知。
func (s *NotificationService) NotifyUpvote(ctx context.Context, actorID uint64, targetType string, targetID uint64) {
	authorID, questionID, title, err := s.repo.GetContentAuthor(ctx, targetType, targetID)
	if err != nil {
		return
	}
	notifType := model.NotifVoteUpQuestion
	if targetType == model.TargetAnswer {
		notifType = model.NotifVoteUpAnswer
	}
	s.create(ctx, authorID, actorID, notifType, targetType, targetID, questionID, firstRunes(title, 80))
}

// ---- 通知列表 ----

type NotificationItem = model.NotificationView

// List 当前用户的通知列表（unread=true 只看未读）。
func (s *NotificationService) List(ctx context.Context, userID uint64, onlyUnread bool, pq httpx.PageQuery) ([]NotificationItem, int64, int64, error) {
	return s.repo.List(ctx, repository.NotificationListQuery{
		UserID: userID, OnlyUnread: onlyUnread, Page: pq.Page, PageSize: pq.PageSize,
	})
}

func (s *NotificationService) MarkRead(ctx context.Context, userID, id uint64) error {
	return s.repo.MarkRead(ctx, id, userID)
}

func (s *NotificationService) MarkAllRead(ctx context.Context, userID uint64) error {
	return s.repo.MarkAllRead(ctx, userID)
}

// firstRunes 按字符截断摘要。
func firstRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
