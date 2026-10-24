package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/leo0315code/goask/internal/middleware"
	"github.com/leo0315code/goask/internal/pkg/httpx"
	"github.com/leo0315code/goask/internal/service"
)

type AnswerHandler struct {
	answers *service.AnswerService
}

func NewAnswerHandler(answers *service.AnswerService) *AnswerHandler {
	return &AnswerHandler{answers: answers}
}

type answerReq struct {
	Content string `json:"content" binding:"required"`
}

// List 某问题下的回答列表：sort=latest|votes。
func (h *AnswerHandler) List(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	pq := pageQuery(c)
	views, total, err := h.answers.List(c.Request.Context(), id, c.Query("sort"), pq)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	pageOK(c, views, total, pq)
}

func (h *AnswerHandler) Create(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req answerReq
	if !httpx.Bind(c, &req) {
		return
	}
	view, err := h.answers.Create(c.Request.Context(), uid, id, req.Content)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *AnswerHandler) Update(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req answerReq
	if !httpx.Bind(c, &req) {
		return
	}
	if err := h.answers.Update(c.Request.Context(), uid, id, req.Content, middleware.IsAdmin(c)); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, nil)
}

func (h *AnswerHandler) Delete(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.answers.Delete(c.Request.Context(), uid, id, middleware.IsAdmin(c)); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, nil)
}
