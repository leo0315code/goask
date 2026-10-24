package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/leo0315code/goask/internal/hub"
	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/apperr"
	"github.com/leo0315code/goask/internal/pkg/httpx"
	"github.com/leo0315code/goask/internal/pkg/jwtx"
	"github.com/leo0315code/goask/internal/service"
)

type NotificationHandler struct {
	notifications *service.NotificationService
	hub           *hub.Hub
	secret        string
	loadUser      func(c *gin.Context, id uint64) (any, error)
}

func NewNotificationHandler(n *service.NotificationService, h *hub.Hub, secret string, loadUser func(c *gin.Context, id uint64) (any, error)) *NotificationHandler {
	return &NotificationHandler{notifications: n, hub: h, secret: secret, loadUser: loadUser}
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

// Stream SSE 实时推送。EventSource 无法携带请求头，token 通过查询参数传递：
// GET /notifications/stream?token=<JWT>
func (h *NotificationHandler) Stream(c *gin.Context) {
	claims, err := jwtx.Parse(c.Query("token"), h.secret)
	if err != nil {
		httpx.Fail(c, apperr.ErrUnauthorized)
		return
	}
	u, err := h.loadUser(c, claims.UserID)
	if err != nil {
		httpx.Fail(c, apperr.With(apperr.ErrUnauthorized, "账号不存在或已被删除"))
		return
	}
	if user := u.(*model.User); user.Status == model.UserStatusBanned {
		httpx.Fail(c, apperr.With(apperr.ErrForbidden, "账号已被封禁"))
		return
	}
	userID := claims.UserID

	w := c.Writer
	header := w.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no") // Nginx 反代时关闭缓冲

	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.Fail(c, apperr.ErrInternal)
		return
	}

	// 客户端断开时 Context 取消，自动退出并释放订阅
	sub, cancel := h.hub.Subscribe(userID)
	defer cancel()

	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	ticker := time.NewTicker(25 * time.Second) // 心跳，防止代理断开空闲连接
	defer ticker.Stop()
	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		case msg, ok := <-sub:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		}
	}
}
