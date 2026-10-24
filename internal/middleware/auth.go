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
)

// Auth 解析 Bearer JWT。required=true 时 token 缺失/无效直接 401；
// required=false 时仅在有有效 token 时注入用户信息（可选登录接口）。
func Auth(secret string, required bool) gin.HandlerFunc {
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
		c.Next()
	}
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
