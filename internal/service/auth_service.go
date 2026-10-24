package service

import (
	"context"
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/leo0315code/goask/internal/config"
	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/apperr"
	"github.com/leo0315code/goask/internal/pkg/captcha"
	"github.com/leo0315code/goask/internal/pkg/jwtx"
	"github.com/leo0315code/goask/internal/pkg/passwordx"
)

var usernameRe = regexp.MustCompile(`^[A-Za-z0-9_]{3,32}$`)

type AuthService struct {
	users      UserRepo
	secret     string
	ttl        time.Duration
	bcryptCost int

	registerEnabled bool
	inviteCode      string
	maxRegPerIP     int
	captcha         *captcha.Store
	regLimiter      *regLimiter
}

func NewAuthService(users UserRepo, cfg *config.Config, cap *captcha.Store) *AuthService {
	return &AuthService{
		users:           users,
		secret:          cfg.JWTSecret,
		ttl:             cfg.JWTExpire,
		bcryptCost:      cfg.BcryptCost,
		registerEnabled: cfg.RegisterEnabled,
		inviteCode:      cfg.RegisterInviteCode,
		maxRegPerIP:     cfg.RegisterMaxPerIPPerDay,
		captcha:         cap,
		regLimiter:      newRegLimiter(),
	}
}

// Captcha 暴露验证码存储，供 handler 生成图形验证码。
func (s *AuthService) Captcha() *captcha.Store { return s.captcha }

// RegisterEnabled / InviteRequired 供注册页展示提示。
func (s *AuthService) RegisterEnabled() bool { return s.registerEnabled }
func (s *AuthService) InviteRequired() bool  { return s.inviteCode != "" }

// regLimiter 同 IP 注册频次限制（内存滑动窗口，24 小时）。
type regLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func newRegLimiter() *regLimiter {
	return &regLimiter{hits: make(map[string][]time.Time)}
}

func (l *regLimiter) allow(ip string, max int) bool {
	if max <= 0 {
		return true
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	keep := l.hits[ip][:0]
	for _, t := range l.hits[ip] {
		if now.Sub(t) < 24*time.Hour {
			keep = append(keep, t)
		}
	}
	if len(keep) >= max {
		l.hits[ip] = keep
		return false
	}
	l.hits[ip] = append(keep, now)
	return true
}

// RegisterInput 注册请求参数。
type RegisterInput struct {
	Username    string
	Email       string
	Password    string
	CaptchaID   string
	CaptchaCode string
	InviteCode  string
	IP          string
}

type AuthResult struct {
	Token string      `json:"token"`
	User  *model.User `json:"user"`
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*AuthResult, error) {
	if !s.registerEnabled {
		return nil, apperr.With(apperr.ErrForbidden, "当前站点已关闭注册")
	}
	if !s.captcha.Verify(in.CaptchaID, in.CaptchaCode) {
		return nil, apperr.With(apperr.ErrValidation, "验证码错误或已过期")
	}
	if s.inviteCode != "" && in.InviteCode != s.inviteCode {
		return nil, apperr.With(apperr.ErrForbidden, "邀请码错误")
	}
	if !s.regLimiter.allow(in.IP, s.maxRegPerIP) {
		return nil, apperr.With(apperr.ErrTooMany, "该 IP 今日注册次数已达上限")
	}

	username := strings.TrimSpace(in.Username)
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if !usernameRe.MatchString(username) {
		return nil, apperr.With(apperr.ErrValidation, "用户名需为 3-32 位字母、数字或下划线")
	}
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return nil, apperr.With(apperr.ErrValidation, "邮箱格式不正确")
	}
	if len(in.Password) < 8 || len(in.Password) > 72 {
		return nil, apperr.With(apperr.ErrValidation, "密码长度需为 8-72 字节")
	}
	password := in.Password
	hash, err := passwordx.Hash(password, s.bcryptCost)
	if err != nil {
		return nil, err
	}
	u := &model.User{
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		Role:         model.RoleUser,
		Status:       model.UserStatusActive,
	}
	if err := s.users.Create(ctx, u); err != nil {
		if errors.Is(err, apperr.ErrConflict) {
			return nil, apperr.With(apperr.ErrConflict, "用户名或邮箱已被注册")
		}
		return nil, err
	}
	return s.issue(u)
}

func (s *AuthService) Login(ctx context.Context, account, password string) (*AuthResult, error) {
	account = strings.TrimSpace(account)
	if account == "" || password == "" {
		return nil, apperr.With(apperr.ErrBadRequest, "账号和密码不能为空")
	}
	u, err := s.users.GetByEmailOrUsername(ctx, account)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return nil, apperr.With(apperr.ErrValidation, "账号或密码错误")
		}
		return nil, err
	}
	if u.Status == model.UserStatusBanned {
		return nil, apperr.With(apperr.ErrForbidden, "账号已被封禁")
	}
	if !passwordx.Verify(u.PasswordHash, password) {
		return nil, apperr.With(apperr.ErrValidation, "账号或密码错误")
	}
	return s.issue(u)
}

func (s *AuthService) issue(u *model.User) (*AuthResult, error) {
	token, _, err := jwtx.Generate(u.ID, u.Role, s.secret, s.ttl)
	if err != nil {
		return nil, err
	}
	return &AuthResult{Token: token, User: u}, nil
}

func (s *AuthService) Profile(ctx context.Context, userID uint64) (*model.User, error) {
	return s.users.GetByID(ctx, userID)
}

func (s *AuthService) UpdateProfile(ctx context.Context, userID uint64, bio, avatarURL string) error {
	if len(bio) > 512 || len(avatarURL) > 512 {
		return apperr.With(apperr.ErrValidation, "bio / avatar_url 最长 512 字符")
	}
	return s.users.UpdateProfile(ctx, userID, bio, avatarURL)
}

// PublicProfile 公开主页信息，不返回邮箱。
func (s *AuthService) PublicProfile(ctx context.Context, username string) (*model.User, error) {
	u, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	cp := *u
	cp.Email = ""
	return &cp, nil
}
