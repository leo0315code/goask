// Package middleware 提供 gin 中间件：认证、CORS、限流、请求日志、panic 恢复。
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/apperr"
	"github.com/leo0315code/goask/internal/pkg/httpx"
	"github.com/leo0315code/goask/internal/pkg/jwtx"
)

const (
	ctxUserIDKey = "ctx_user_id"
	ctxRoleKey   = "ctx_role"
	ctxUserKey   = "ctx_user"
)

// UserLoader 必登录场景下按 ID 加载用户（用于封禁状态与最新角色校验）。
type UserLoader func(c *gin.Context, id uint64) (any, error)

// Auth 解析 Bearer JWT。
// required=true 时：token 缺失/无效 → 401；用户不存在/已删除 → 401；被封禁 → 403，
// 并从数据库读取最新角色（角色变更即时生效）。
// required=false 时：仅在有有效 token 时注入用户信息（可选登录接口，不查库）。
func Auth(secret string, required bool, loadUser UserLoader) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ""
		if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer ") {
			token = strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
		}
		if token == "" {
			if required {
				httpx.Fail(c, apperr.ErrUnauthorized)
				c.Abort()
				return
			}
			c.Next()
			return
		}
		claims, err := jwtx.Parse(token, secret)
		if err != nil {
			if required {
				httpx.Fail(c, apperr.ErrUnauthorized)
				c.Abort()
				return
			}
			c.Next()
			return
		}
		role := claims.Role
		if role != model.RoleAdmin {
			role = model.RoleUser
		}
		c.Set(ctxUserIDKey, claims.UserID)
		c.Set(ctxRoleKey, role)

		if required && loadUser != nil {
			u, lerr := loadUser(c, claims.UserID)
			if lerr != nil {
				httpx.Fail(c, apperr.With(apperr.ErrUnauthorized, "账号不存在或已被删除"))
				c.Abort()
				return
			}
			user := u.(*model.User)
			if user.Status == model.UserStatusBanned {
				httpx.Fail(c, apperr.With(apperr.ErrForbidden, "账号已被封禁"))
				c.Abort()
				return
			}
			c.Set(ctxRoleKey, user.Role)
			c.Set(ctxUserKey, user)
		}
		c.Next()
	}
}

// CurrentUser 取必登录路径上加载的用户（未经过 required 认证时返回 nil）。
func CurrentUser(c *gin.Context) *model.User {
	v, ok := c.Get(ctxUserKey)
	if !ok {
		return nil
	}
	u, _ := v.(*model.User)
	return u
}

// UserID 取当前登录用户 ID（未登录返回 false）。
func UserID(c *gin.Context) (uint64, bool) {
	v, ok := c.Get(ctxUserIDKey)
	if !ok {
		return 0, false
	}
	id, ok := v.(uint64)
	return id, ok
}

// Role 取当前登录用户角色（未登录返回空字符串）。
func Role(c *gin.Context) string {
	v, ok := c.Get(ctxRoleKey)
	if !ok {
		return ""
	}
	role, _ := v.(string)
	return role
}

// IsAdmin 当前请求者是否管理员。
func IsAdmin(c *gin.Context) bool { return Role(c) == model.RoleAdmin }
