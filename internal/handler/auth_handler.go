package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/leo0315code/goask/internal/pkg/httpx"
	"github.com/leo0315code/goask/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type registerReq struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerReq
	if !httpx.Bind(c, &req) {
		return
	}
	res, err := h.auth.Register(c.Request.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, res)
}

type loginReq struct {
	Account  string `json:"account" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if !httpx.Bind(c, &req) {
		return
	}
	res, err := h.auth.Login(c.Request.Context(), req.Account, req.Password)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, res)
}

func (h *AuthHandler) Me(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	u, err := h.auth.Profile(c.Request.Context(), uid)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, u)
}

type updateMeReq struct {
	Bio       string `json:"bio" binding:"max=512"`
	AvatarURL string `json:"avatar_url" binding:"max=512"`
}

func (h *AuthHandler) UpdateMe(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	var req updateMeReq
	if !httpx.Bind(c, &req) {
		return
	}
	if err := h.auth.UpdateProfile(c.Request.Context(), uid, req.Bio, req.AvatarURL); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, nil)
}

func (h *AuthHandler) PublicProfile(c *gin.Context) {
	u, err := h.auth.PublicProfile(c.Request.Context(), c.Param("username"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, u)
}
