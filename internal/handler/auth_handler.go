package handler

import (
	"crypto/rand"
	"fmt"
	"math/big"

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

// Captcha 发放图形算术验证码（注册用），并附带注册策略提示。
func (h *AuthHandler) Captcha(c *gin.Context) {
	id, question := h.auth.Captcha().New()
	httpx.OK(c, gin.H{
		"captcha_id":       id,
		"svg":              captchaSVG(question),
		"register_enabled": h.auth.RegisterEnabled(),
		"invite_required":  h.auth.InviteRequired(),
	})
}

// captchaSVG 渲染带干扰线的算术验证码 SVG。
func captchaSVG(question string) string {
	rnd := func(max int64) int {
		n, _ := rand.Int(rand.Reader, big.NewInt(max))
		return int(n.Int64())
	}
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="130" height="46" viewBox="0 0 130 46">`+
		`<rect width="130" height="46" rx="6" fill="#f1f4f9"/>`+
		`<line x1="0" y1="%d" x2="130" y2="%d" stroke="#c3cbd9" stroke-width="1"/>`+
		`<line x1="0" y1="%d" x2="130" y2="%d" stroke="#c3cbd9" stroke-width="1"/>`+
		`<text x="65" y="30" font-size="22" text-anchor="middle" fill="#334155" font-family="monospace" letter-spacing="3">%s</text>`+
		`</svg>`,
		rnd(38)+4, rnd(38)+4, rnd(38)+4, rnd(38)+4, question)
}

type registerReq struct {
	Username    string `json:"username" binding:"required,min=3,max=32"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8,max=72"`
	CaptchaID   string `json:"captcha_id" binding:"required"`
	CaptchaCode string `json:"captcha_code" binding:"required"`
	InviteCode  string `json:"invite_code"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerReq
	if !httpx.Bind(c, &req) {
		return
	}
	res, err := h.auth.Register(c.Request.Context(), service.RegisterInput{
		Username:    req.Username,
		Email:       req.Email,
		Password:    req.Password,
		CaptchaID:   req.CaptchaID,
		CaptchaCode: req.CaptchaCode,
		InviteCode:  req.InviteCode,
		IP:          c.ClientIP(),
	})
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
