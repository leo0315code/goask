package service

import (
	"context"
	"time"

	"github.com/leo0315code/goask/internal/pkg/apperr"
)

// SpamChecker 防灌水检查接口（repository.SpamRepo 实现，测试用假实现替换）。
type SpamRepo interface {
	LastPostAt(ctx context.Context, table string, userID uint64) (time.Time, error)
	RecentCount(ctx context.Context, table string, userID uint64, within time.Duration) (int64, error)
	ExistsRecentDuplicate(ctx context.Context, table, column string, userID uint64, content string, within time.Duration) (bool, error)
}

// SpamChecker 对外暴露的检查能力。
type SpamChecker interface {
	// Check 频次检查：最小间隔 + 24h 数量上限。
	Check(ctx context.Context, userID uint64, kind string) error
	// CheckDuplicate 重复内容检查：24h 内不允许发布完全相同的内容。
	CheckDuplicate(ctx context.Context, userID uint64, kind, content string) error
}

// 防灌水规则：kind → 最小发布间隔 / 24 小时上限。
var spamRules = map[string]struct {
	interval time.Duration
	daily    int
}{
	"question": {interval: time.Minute, daily: 20},
	"answer":   {interval: 30 * time.Second, daily: 50},
	"comment":  {interval: 10 * time.Second, daily: 200},
}

type SpamGuard struct {
	repo SpamRepo
}

func NewSpamGuard(repo SpamRepo) *SpamGuard {
	return &SpamGuard{repo: repo}
}

func (g *SpamGuard) Check(ctx context.Context, userID uint64, kind string) error {
	rule, ok := spamRules[kind]
	if !ok || g == nil || g.repo == nil {
		return nil
	}
	last, err := g.repo.LastPostAt(ctx, kind+"s", userID)
	if err == nil && !last.IsZero() && time.Since(last) < rule.interval {
		return apperr.With(apperr.ErrTooMany, "发布太快了，请稍后再试")
	}
	n, err := g.repo.RecentCount(ctx, kind+"s", userID, 24*time.Hour)
	if err == nil && n >= int64(rule.daily) {
		return apperr.With(apperr.ErrTooMany, "您 24 小时内的发布数量已达上限")
	}
	return nil
}

func (g *SpamGuard) CheckDuplicate(ctx context.Context, userID uint64, kind, content string) error {
	if _, ok := spamRules[kind]; !ok || g == nil || g.repo == nil {
		return nil
	}
	dup, err := g.repo.ExistsRecentDuplicate(ctx, kind+"s", duplicateColumn(kind), userID, content, 24*time.Hour)
	if err == nil && dup {
		return apperr.With(apperr.ErrValidation, "请勿重复发布相同的内容")
	}
	return nil
}

func duplicateColumn(kind string) string {
	if kind == "question" {
		return "title"
	}
	return "content"
}
