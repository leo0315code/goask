package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/leo0315code/goask/internal/pkg/apperr"
	"github.com/leo0315code/goask/internal/pkg/httpx"
	"github.com/leo0315code/goask/internal/service"
)

type VoteHandler struct {
	votes *service.VoteService
}

func NewVoteHandler(votes *service.VoteService) *VoteHandler {
	return &VoteHandler{votes: votes}
}

type voteReq struct {
	TargetType string `json:"target_type" binding:"required,oneof=question answer"`
	TargetID   uint64 `json:"target_id" binding:"required"`
	Value      int    `json:"value" binding:"required,oneof=1 -1"`
}

func (h *VoteHandler) Vote(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	var req voteReq
	if !httpx.Bind(c, &req) {
		return
	}
	state, err := h.votes.Vote(c.Request.Context(), uid, req.TargetType, req.TargetID, req.Value)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, state)
}

type TagHandler struct {
	tags *service.TagService
}

func NewTagHandler(tags *service.TagService) *TagHandler {
	return &TagHandler{tags: tags}
}

func (h *TagHandler) List(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil || limit < 0 {
		httpx.Fail(c, apperr.With(apperr.ErrBadRequest, "limit 需为非负整数"))
		return
	}
	tags, err := h.tags.List(c.Request.Context(), limit)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, tags)
}
