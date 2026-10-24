package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"github.com/leo0315code/goask/internal/pkg/apperr"
	"github.com/leo0315code/goask/internal/pkg/httpx"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimit 按客户端 IP 的令牌桶限流，perMinute 为每分钟允许次数，
// 突发容量为每分钟限额的 1/10（至少 1）。后台协程定期清理不活跃 IP。
func RateLimit(perMinute int) gin.HandlerFunc {
	if perMinute <= 0 {
		perMinute = 1
	}
	r := rate.Every(time.Minute / time.Duration(perMinute))
	burst := perMinute / 10
	if burst < 1 {
		burst = 1
	}

	var visitors sync.Map // map[string]*visitor
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				now := time.Now()
				visitors.Range(func(key, value any) bool {
					if v, ok := value.(*visitor); ok && now.Sub(v.lastSeen) > 10*time.Minute {
						visitors.Delete(key)
					}
					return true
				})
			case <-stop:
				return
			}
		}
	}()

	return func(c *gin.Context) {
		key := c.ClientIP()
		stored, _ := visitors.LoadOrStore(key, &visitor{limiter: rate.NewLimiter(r, burst), lastSeen: time.Now()})
		v := stored.(*visitor)
		v.lastSeen = time.Now()
		if !v.limiter.Allow() {
			httpx.Fail(c, apperr.ErrTooMany)
			c.Abort()
			return
		}
		c.Next()
	}
}
