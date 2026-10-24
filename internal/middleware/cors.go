package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS 按白名单放行跨域请求；allowedOrigins 含 "*" 时放行所有来源。
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowAll := false
	allow := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o == "*" {
			allowAll = true
		} else {
			allow[o] = true
		}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			switch {
			case allowAll:
				c.Header("Access-Control-Allow-Origin", "*")
			case allow[origin]:
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			}
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			c.Header("Access-Control-Max-Age", "86400")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
