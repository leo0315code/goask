// GoAsk 服务入口：加载配置 → 连接数据库 → 执行迁移 → 装配路由 → 优雅停机。
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/leo0315code/goask/internal/config"
	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/passwordx"
	"github.com/leo0315code/goask/internal/repository"
	"github.com/leo0315code/goask/internal/router"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "配置错误:", err)
		os.Exit(1)
	}

	// 确保上传目录存在
	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "创建上传目录失败:", err)
		os.Exit(1)
	}

	level := slog.LevelInfo
	if !cfg.IsProd() {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	db, err := connectDB(cfg)
	if err != nil {
		logger.Error("数据库连接失败", "error", err)
		os.Exit(1)
	}
	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("获取底层连接失败", "error", err)
		os.Exit(1)
	}
	defer func() { _ = sqlDB.Close() }()

	if cfg.MigrateOnStart {
		if err := repository.RunMigrations(db, cfg.MigrationsDir); err != nil {
			logger.Error("数据库迁移失败", "error", err)
			os.Exit(1)
		}
		logger.Info("数据库迁移完成")
	}

	seedAdmin(db, cfg, logger)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           router.New(cfg, db),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		logger.Info("HTTP 服务启动", "addr", srv.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP 服务异常退出", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("收到退出信号，开始优雅停机")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("优雅停机超时", "error", err)
	}
	logger.Info("服务已退出")
}

// connectDB 带重试地建立数据库连接（容器编排下 MySQL 可能晚于应用就绪）。
func connectDB(cfg *config.Config) (*gorm.DB, error) {
	var db *gorm.DB
	var err error
	for i := 0; i < 30; i++ {
		db, err = gorm.Open(mysql.Open(cfg.DB.DSN()), &gorm.Config{
			Logger:         gormlogger.Default.LogMode(gormlogger.Warn),
			TranslateError: true, // 把 MySQL 1062 等错误翻译为 gorm.ErrDuplicatedKey
		})
		if err == nil {
			var sqlDB *sql.DB
			sqlDB, err = db.DB()
			if err == nil {
				if err = sqlDB.PingContext(context.Background()); err == nil {
					sqlDB.SetMaxOpenConns(cfg.DB.MaxOpenConns)
					sqlDB.SetMaxIdleConns(cfg.DB.MaxIdleConns)
					sqlDB.SetConnMaxLifetime(cfg.DB.ConnMaxLifetime)
					return db, nil
				}
			}
		}
		time.Sleep(time.Second)
	}
	return nil, err
}

// seedAdmin 按环境变量确保管理员账号存在（仅设置 SEED_ADMIN_* 时执行）。
func seedAdmin(db *gorm.DB, cfg *config.Config, logger *slog.Logger) {
	if cfg.SeedAdminUsername == "" || cfg.SeedAdminPassword == "" {
		return
	}
	var count int64
	if err := db.Model(&model.User{}).Where("username = ?", cfg.SeedAdminUsername).Count(&count).Error; err != nil {
		logger.Error("查询管理员账号失败", "error", err)
		return
	}
	if count > 0 {
		return
	}
	hash, err := passwordx.Hash(cfg.SeedAdminPassword, cfg.BcryptCost)
	if err != nil {
		logger.Error("生成管理员密码哈希失败", "error", err)
		return
	}
	u := &model.User{
		Username:     cfg.SeedAdminUsername,
		Email:        cfg.SeedAdminUsername + "@admin.local",
		PasswordHash: hash,
		Role:         model.RoleAdmin,
		Status:       model.UserStatusActive,
	}
	if err := db.Create(u).Error; err != nil {
		logger.Error("创建管理员账号失败", "error", err)
		return
	}
	logger.Info("已创建管理员账号", "username", cfg.SeedAdminUsername)
}
