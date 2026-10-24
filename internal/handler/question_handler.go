package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/leo0315code/goask/internal/middleware"
	"github.com/leo0315code/goask/internal/pkg/httpx"
	"github.com/leo0315code/goask/internal/service"
)

type QuestionHandler struct {
	questions *service.QuestionService
}

func NewQuestionHandler(questions *service.QuestionService) *QuestionHandler {
	return &QuestionHandler{questions: questions}
}

type questionReq struct {
	Title   string   `json:"title" binding:"required"`
	Content string   `json:"content" binding:"required"`
	Tags    []string `json:"tags"`
}

// List 问题列表：sort=latest|hot|unanswered，tag=标签名，time=day|week|month|year。
func (h *QuestionHandler) List(c *gin.Context) {
	pq := pageQuery(c)
	views, total, err := h.questions.List(c.Request.Context(), c.Query("sort"), c.Query("tag"), c.Query("time"), pq)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	pageOK(c, views, total, pq)
}

func (h *QuestionHandler) Create(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	var req questionReq
	if !httpx.Bind(c, &req) {
		return
	}
	view, err := h.questions.Create(c.Request.Context(), uid, req.Title, req.Content, req.Tags)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *QuestionHandler) Get(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	viewer, _ := middleware.UserID(c)
	view, err := h.questions.Get(c.Request.Context(), id, viewer)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func (h *QuestionHandler) Update(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req questionReq
	if !httpx.Bind(c, &req) {
		return
	}
	if err := h.questions.Update(c.Request.Context(), uid, id, req.Title, req.Content, req.Tags, middleware.IsAdmin(c)); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, nil)
}

func (h *QuestionHandler) Delete(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.questions.Delete(c.Request.Context(), uid, id, middleware.IsAdmin(c)); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, nil)
}

func (h *QuestionHandler) Close(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.questions.Close(c.Request.Context(), uid, id, middleware.IsAdmin(c)); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, nil)
}

type acceptReq struct {
	AnswerID uint64 `json:"answer_id" binding:"required"`
}

func (h *QuestionHandler) Accept(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req acceptReq
	if !httpx.Bind(c, &req) {
		return
	}
	if err := h.questions.AcceptAnswer(c.Request.Context(), uid, id, req.AnswerID, middleware.IsAdmin(c)); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, nil)
}

func (h *QuestionHandler) Search(c *gin.Context) {
	pq := pageQuery(c)
	views, total, err := h.questions.Search(c.Request.Context(), c.Query("q"), pq)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	pageOK(c, views, total, pq)
}

// ListByUser 用户主页的提问列表。
func (h *QuestionHandler) ListByUser(c *gin.Context) {
	pq := pageQuery(c)
	views, total, err := h.questions.ListByUser(c.Request.Context(), c.Param("username"), pq)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	pageOK(c, views, total, pq)
}
