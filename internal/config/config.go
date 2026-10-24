// Package config 从环境变量加载运行时配置；存在 .env 文件时会先注入（不覆盖已有环境变量）。
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env      string
	HTTPPort int
	DB       DBConfig

	JWTSecret  string
	JWTExpire  time.Duration
	BcryptCost int

	CORSOrigins    []string // ["*"] 表示放行所有来源
	TrustedProxies []string // 空 = 不信任任何代理，ClientIP 取 RemoteAddr

	RateLimitGlobal int // 次/分/IP
	RateLimitAuth   int // 注册/登录 次/分/IP

	UploadDir     string // 图片上传存储目录
	MaxUploadSize int64  // 单张图片上限（字节）

	RegisterEnabled        bool   // 是否开放注册
	RegisterInviteCode     string // 非空时注册必须携带邀请码
	RegisterMaxPerIPPerDay int    // 同 IP 每日注册上限（0 = 不限）

	MigrateOnStart    bool
	MigrationsDir     string
	SeedAdminUsername string
	SeedAdminPassword string
}

type DBConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func (c *Config) IsProd() bool { return c.Env == "release" }

func (c *DBConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=UTC&multiStatements=true",
		c.User, c.Password, c.Host, c.Port, c.Name)
}

// Load 读取全部配置并做基础校验，失败返回错误由启动方终止进程（fail fast）。
func Load() (*Config, error) {
	_ = loadDotEnv(".env")

	cfg := &Config{
		Env:      getEnv("APP_ENV", "dev"),
		HTTPPort: getEnvInt("PORT", 8080),
		DB: DBConfig{
			Host:            getEnv("DB_HOST", "127.0.0.1"),
			Port:            getEnvInt("DB_PORT", 3306),
			User:            getEnv("DB_USER", "root"),
			Password:        getEnv("DB_PASSWORD", ""),
			Name:            getEnv("DB_NAME", "goask"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		JWTSecret:       os.Getenv("JWT_SECRET"),
		JWTExpire:       getEnvDuration("JWT_EXPIRE", 72*time.Hour),
		BcryptCost:      getEnvInt("BCRYPT_COST", 12),
		CORSOrigins:     splitList(getEnv("CORS_ORIGINS", "*")),
		TrustedProxies:  splitList(getEnv("TRUSTED_PROXIES", "")),
		RateLimitGlobal: getEnvInt("RATE_LIMIT_GLOBAL_RPM", 120),
		RateLimitAuth:   getEnvInt("RATE_LIMIT_AUTH_RPM", 10),

		UploadDir:     getEnv("UPLOAD_DIR", "uploads"),
		MaxUploadSize: int64(getEnvInt("MAX_UPLOAD_MB", 5)) * 1024 * 1024,

		RegisterEnabled:        getEnvBool("REGISTER_ENABLED", true),
		RegisterInviteCode:     os.Getenv("REGISTER_INVITE_CODE"),
		RegisterMaxPerIPPerDay: getEnvInt("REGISTER_MAX_PER_IP_PER_DAY", 3),
		MigrateOnStart:         getEnvBool("MIGRATE_ON_START", true),
		MigrationsDir:          getEnv("MIGRATIONS_DIR", "migrations"),
		SeedAdminUsername:      os.Getenv("SEED_ADMIN_USERNAME"),
		SeedAdminPassword:      os.Getenv("SEED_ADMIN_PASSWORD"),
	}

	if len(cfg.JWTSecret) < 16 {
		return nil, errors.New("JWT_SECRET 未设置或长度小于 16 字符，拒绝启动")
	}
	if cfg.BcryptCost < 4 || cfg.BcryptCost > 15 {
		return nil, errors.New("BCRYPT_COST 取值必须在 4-15 之间")
	}
	return cfg, nil
}

func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
			value = value[1 : len(value)-1]
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
	return sc.Err()
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func splitList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
