package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/leo0315code/goask/internal/middleware"
	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/apperr"
	"github.com/leo0315code/goask/internal/pkg/httpx"
	"github.com/leo0315code/goask/internal/service"
)

type AdminHandler struct {
	users *service.UserService
}

func NewAdminHandler(users *service.UserService) *AdminHandler {
	return &AdminHandler{users: users}
}

// requireAdmin 校验当前用户为管理员（required 认证路径已从数据库加载用户）。
func requireAdmin(c *gin.Context) (*model.User, bool) {
	u := middleware.CurrentUser(c)
	if u == nil || u.Role != model.RoleAdmin {
		httpx.Fail(c, apperr.ErrForbidden)
		return nil, false
	}
	return u, true
}

// ListUsers 用户列表：?q=搜索词&page=&page_size=
func (h *AdminHandler) ListUsers(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	pq := pageQuery(c)
	users, total, err := h.users.ListUsers(c.Request.Context(), c.Query("q"), pq.Page, pq.PageSize)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	pageOK(c, users, total, pq)
}

type setStatusReq struct {
	Status string `json:"status" binding:"required,oneof=active banned"`
}

// SetUserStatus 封禁 / 解封用户。
func (h *AdminHandler) SetUserStatus(c *gin.Context) {
	operator, ok := requireAdmin(c)
	if !ok {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req setStatusReq
	if !httpx.Bind(c, &req) {
		return
	}
	if err := h.users.SetStatus(c.Request.Context(), operator, id, req.Status); err != nil {
		httpx.Fail(c, err)
		return
	}
	// 管理操作审计日志
	slog.Warn("admin audit",
		"action", "set_user_status",
		"operator_id", operator.ID,
		"operator", operator.Username,
		"target_user_id", id,
		"status", req.Status,
		"ip", c.ClientIP(),
	)
	httpx.OK(c, nil)
}
