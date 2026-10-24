package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/leo0315code/goask/internal/middleware"
	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/httpx"
	"github.com/leo0315code/goask/internal/service"
)

type CommentHandler struct {
	comments *service.CommentService
}

func NewCommentHandler(comments *service.CommentService) *CommentHandler {
	return &CommentHandler{comments: comments}
}

type commentReq struct {
	Content  string `json:"content" binding:"required,max=1000"`
	ParentID uint64 `json:"parent_id"`
}

func (h *CommentHandler) ListQuestionComments(c *gin.Context) {
	h.list(c, model.TargetQuestion)
}

func (h *CommentHandler) CreateQuestionComment(c *gin.Context) {
	h.create(c, model.TargetQuestion)
}

func (h *CommentHandler) ListAnswerComments(c *gin.Context) {
	h.list(c, model.TargetAnswer)
}

func (h *CommentHandler) CreateAnswerComment(c *gin.Context) {
	h.create(c, model.TargetAnswer)
}

func (h *CommentHandler) list(c *gin.Context, targetType string) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	pq := pageQuery(c)
	views, total, err := h.comments.List(c.Request.Context(), targetType, id, pq)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	pageOK(c, views, total, pq)
}

func (h *CommentHandler) create(c *gin.Context, targetType string) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req commentReq
	if !httpx.Bind(c, &req) {
		return
	}
	view, err := h.comments.Create(c.Request.Context(), uid, targetType, id, req.Content, req.ParentID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// Delete 删除评论（作者本人或管理员）。
func (h *CommentHandler) Delete(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.comments.Delete(c.Request.Context(), uid, id, middleware.IsAdmin(c)); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, nil)
}
