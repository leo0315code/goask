// Package router 组装 gin 引擎、中间件与全部路由，并完成依赖装配。
package router

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/leo0315code/goask/internal/config"
	"github.com/leo0315code/goask/internal/handler"
	"github.com/leo0315code/goask/internal/middleware"
	"github.com/leo0315code/goask/internal/pkg/httpx"
	"github.com/leo0315code/goask/internal/repository"
	"github.com/leo0315code/goask/internal/service"
)

func New(cfg *config.Config, db *gorm.DB) *gin.Engine {
	repos := repository.NewRepos(db)
	authSvc := service.NewAuthService(repos.Users, cfg)
	questionSvc := service.NewQuestionService(repos.Questions, repos.Answers, repos.Users, repos.Tags, repos.Votes)
	answerSvc := service.NewAnswerService(repos.Answers, repos.Questions, repos.Users)
	commentSvc := service.NewCommentService(repos.Comments, repos.Questions, repos.Answers, repos.Users)
	voteSvc := service.NewVoteService(repos.Votes)
	tagSvc := service.NewTagService(repos.Tags)

	logger := slog.Default()
	if cfg.IsProd() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	if len(cfg.TrustedProxies) > 0 {
		_ = r.SetTrustedProxies(cfg.TrustedProxies)
	} else {
		// 默认不信任任何代理，ClientIP 取 RemoteAddr；部署在反代后时配置 TRUSTED_PROXIES
		_ = r.SetTrustedProxies(nil)
	}

	// 请求体上限 1MB
	r.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
		c.Next()
	})
	r.Use(
		middleware.Recovery(logger),
		middleware.RequestLogger(logger),
		middleware.CORS(cfg.CORSOrigins),
		middleware.RateLimit(cfg.RateLimitGlobal),
	)

	r.GET("/healthz", func(c *gin.Context) {
		httpx.OK(c, gin.H{"status": "ok"})
	})
	r.GET("/readyz", ready(db))

	authLimiter := middleware.RateLimit(cfg.RateLimitAuth)
	authRequired := middleware.Auth(cfg.JWTSecret, true)
	authOptional := middleware.Auth(cfg.JWTSecret, false)

	authH := handler.NewAuthHandler(authSvc)
	questionH := handler.NewQuestionHandler(questionSvc)
	answerH := handler.NewAnswerHandler(answerSvc)
	commentH := handler.NewCommentHandler(commentSvc)
	voteH := handler.NewVoteHandler(voteSvc)
	tagH := handler.NewTagHandler(tagSvc)

	v1 := r.Group("/api/v1")
	v1.Use(authOptional)

	// 认证与用户
	v1.POST("/auth/register", authLimiter, authH.Register)
	v1.POST("/auth/login", authLimiter, authH.Login)
	v1.GET("/users/me", authRequired, authH.Me)
	v1.PUT("/users/me", authRequired, authH.UpdateMe)
	v1.GET("/users/:username", authH.PublicProfile)

	// 问题与检索
	v1.GET("/questions", questionH.List)
	v1.GET("/search", questionH.Search)
	v1.POST("/questions", authRequired, questionH.Create)
	v1.GET("/questions/:id", questionH.Get)
	v1.PUT("/questions/:id", authRequired, questionH.Update)
	v1.DELETE("/questions/:id", authRequired, questionH.Delete)
	v1.POST("/questions/:id/close", authRequired, questionH.Close)
	v1.POST("/questions/:id/accept", authRequired, questionH.Accept)

	// 回答
	v1.GET("/questions/:id/answers", answerH.List)
	v1.POST("/questions/:id/answers", authRequired, answerH.Create)
	v1.PUT("/answers/:id", authRequired, answerH.Update)
	v1.DELETE("/answers/:id", authRequired, answerH.Delete)

	// 评论
	v1.GET("/questions/:id/comments", commentH.ListQuestionComments)
	v1.POST("/questions/:id/comments", authRequired, commentH.CreateQuestionComment)
	v1.GET("/answers/:id/comments", commentH.ListAnswerComments)
	v1.POST("/answers/:id/comments", authRequired, commentH.CreateAnswerComment)

	// 投票与标签
	v1.POST("/votes", authRequired, voteH.Vote)
	v1.GET("/tags", tagH.List)

	return r
}

// ready 就绪探针：校验数据库可用。
func ready(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := db.WithContext(c.Request.Context()).Exec("SELECT 1").Error; err != nil {
			c.JSON(http.StatusServiceUnavailable, httpx.Response{Code: 50000, Message: "数据库不可用"})
			return
		}
		httpx.OK(c, gin.H{"status": "ready"})
	}
}
