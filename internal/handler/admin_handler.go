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
	admin *service.AdminService
}

func NewAdminHandler(users *service.UserService, admin *service.AdminService) *AdminHandler {
	return &AdminHandler{users: users, admin: admin}
}

// Stats 概览统计。
func (h *AdminHandler) Stats(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	stats, err := h.admin.Stats(c.Request.Context())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, stats)
}

// Questions 问题管理列表：?q=&status=all|open|closed|deleted&page=
func (h *AdminHandler) Questions(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	pq := pageQuery(c)
	list, total, err := h.admin.Questions(c.Request.Context(), c.Query("q"), c.DefaultQuery("status", "all"), pq.Page, pq.PageSize)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	pageOK(c, list, total, pq)
}

// Comments 评论管理列表：?q=&page=
func (h *AdminHandler) Comments(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	pq := pageQuery(c)
	list, total, err := h.admin.Comments(c.Request.Context(), c.Query("q"), pq.Page, pq.PageSize)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	pageOK(c, list, total, pq)
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
