package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/leo0315code/goask/internal/pkg/httpx"
	"github.com/leo0315code/goask/internal/service"
)

type FavoriteHandler struct {
	favorites *service.FavoriteService
}

func NewFavoriteHandler(f *service.FavoriteService) *FavoriteHandler {
	return &FavoriteHandler{favorites: f}
}

// Toggle 收藏/取消收藏问题，返回切换后的状态。
func (h *FavoriteHandler) Toggle(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	favorited, err := h.favorites.Toggle(c.Request.Context(), uid, id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"favorited": favorited})
}

// Get 查询当前用户是否收藏了该问题。
func (h *FavoriteHandler) Get(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	uid, ok := currentUserID(c)
	if !ok {
		httpx.OK(c, gin.H{"favorited": false})
		return
	}
	favorited, err := h.favorites.IsFavorited(c.Request.Context(), uid, id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"favorited": favorited})
}

// MyFavorites 我的收藏列表（分页）。
func (h *FavoriteHandler) MyFavorites(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	pq := pageQuery(c)
	list, total, err := h.favorites.ListMine(c.Request.Context(), uid, pq.Page, pq.PageSize)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	pageOK(c, list, total, pq)
}

// UserAnswers 某用户的回答列表（个人主页 Tab 用）。
func (h *FavoriteHandler) UserAnswers(c *gin.Context) {
	pq := pageQuery(c)
	list, total, err := h.favorites.AnswersByUser(c.Request.Context(), c.Param("username"), pq.Page, pq.PageSize)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	pageOK(c, list, total, pq)
}
