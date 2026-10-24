package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/leo0315code/goask/internal/pkg/httpx"
	"github.com/leo0315code/goask/internal/service"
)

type NotificationHandler struct {
	notifications *service.NotificationService
}

func NewNotificationHandler(n *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifications: n}
}

// List 通知列表：?unread=1 只看未读。
func (h *NotificationHandler) List(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	pq := pageQuery(c)
	items, total, unread, err := h.notifications.List(c.Request.Context(), uid, c.Query("unread") == "1", pq)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"list":         items,
		"total":        total,
		"unread_count": unread,
		"page":         pq.Page,
		"page_size":    pq.PageSize,
	})
}

// Read 单条已读。
func (h *NotificationHandler) Read(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.notifications.MarkRead(c.Request.Context(), uid, id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, nil)
}

// ReadAll 全部已读。
func (h *NotificationHandler) ReadAll(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	if err := h.notifications.MarkAllRead(c.Request.Context(), uid); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, nil)
}
