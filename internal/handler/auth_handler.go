package handler

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/leo0315code/goask/internal/middleware"
	"github.com/leo0315code/goask/internal/pkg/apperr"
	"github.com/leo0315code/goask/internal/pkg/httpx"
	"github.com/leo0315code/goask/internal/service"
)

type AuthHandler struct {
	auth  *service.AuthService
	users *service.UserService
}

func NewAuthHandler(auth *service.AuthService, users *service.UserService) *AuthHandler {
	return &AuthHandler{auth: auth, users: users}
}

type changePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=72"`
}

// ChangePassword 修改当前用户密码。
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	var req changePasswordReq
	if !httpx.Bind(c, &req) {
		return
	}
	user, err := h.users.ChangePassword(c.Request.Context(), uid, req.OldPassword, req.NewPassword)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	// 旧 token 已随版本号失效，签发并返回新 token 保持登录
	res, err := h.auth.IssueFor(user)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, res)
}

// Refresh 刷新当前会话的 token。
func (h *AuthHandler) Refresh(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		httpx.Fail(c, apperr.ErrUnauthorized)
		return
	}
	res, err := h.auth.IssueFor(user)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, res)
}

// LogoutEverywhere 使所有设备的登录态失效，返回新 token（当前设备保持登录）。
func (h *AuthHandler) LogoutEverywhere(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	user, err := h.users.LogoutEverywhere(c.Request.Context(), uid)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	res, err := h.auth.IssueFor(user)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, res)
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

// captchaSVG 渲染算术验证码 SVG：逐字符着色旋转 + 噪点浅弧线，保证可读性。
func captchaSVG(question string) string {
	rnd := func(n int) int {
		v, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
		return int(v.Int64())
	}
	palette := []string{"#1e293b", "#1d4ed8", "#0f766e", "#6d28d9", "#b45309"}

	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="170" height="52" viewBox="0 0 170 52">`)
	b.WriteString(`<rect width="170" height="52" rx="8" fill="#eef2f9"/>`)
	for i := 0; i < 24; i++ {
		fmt.Fprintf(&b, `<circle cx="%d" cy="%d" r="1.2" fill="#94a3b8" opacity="0.3"/>`, rnd(166)+2, rnd(48)+2)
	}
	// 文字下层的浅色弧线，不横穿字形
	fmt.Fprintf(&b, `<path d="M0,%d Q85,%d 170,%d" stroke="#c7d2e2" stroke-width="1.5" fill="none" opacity="0.8"/>`,
		rnd(12)+6, rnd(18)+18, rnd(12)+6)
	// 每个字符独立着色与轻微旋转
	x := 16
	for _, ch := range question {
		if ch == ' ' {
			x += 9
			continue
		}
		fmt.Fprintf(&b, `<text x="%d" y="35" font-size="27" font-weight="bold" fill="%s" `+
			`font-family="Menlo, Consolas, 'Courier New', monospace" text-anchor="middle" `+
			`transform="rotate(%d %d 35)">%c</text>`,
			x, palette[rnd(len(palette))], rnd(17)-8, x, ch)
		x += 26
	}
	b.WriteString(`</svg>`)
	return b.String()
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
