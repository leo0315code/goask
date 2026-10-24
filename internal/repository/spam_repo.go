package repository

import (
	"context"
	"database/sql"
	"time"

	"gorm.io/gorm"
)

// SpamRepo 为防灌水检查提供查询（表/列名仅允许白名单，杜绝拼接注入）。
type SpamRepo struct {
	db *gorm.DB
}

// spamColumns 白名单：表名 → 可做重复检测的列。
var spamColumns = map[string]string{
	"questions": "title",
	"answers":   "content",
	"comments":  "content",
}

func spamValid(table, column string) bool {
	c, ok := spamColumns[table]
	return ok && c == column
}

// LastPostAt 用户在该表最后一次发布时间；从未发布返回零值。
func (r *SpamRepo) LastPostAt(ctx context.Context, table string, userID uint64) (time.Time, error) {
	if _, ok := spamColumns[table]; !ok {
		return time.Time{}, nil
	}
	var t sql.NullTime
	err := r.db.WithContext(ctx).Table(table).
		Where("user_id = ?", userID).
		Select("MAX(created_at)").
		Scan(&t).Error
	if err != nil || !t.Valid {
		return time.Time{}, err
	}
	return t.Time, nil
}

// RecentCount 用户在 within 时间窗口内的发布数量。
func (r *SpamRepo) RecentCount(ctx context.Context, table string, userID uint64, within time.Duration) (int64, error) {
	if _, ok := spamColumns[table]; !ok {
		return 0, nil
	}
	var n int64
	err := r.db.WithContext(ctx).Table(table).
		Where("user_id = ? AND created_at >= ?", userID, time.Now().Add(-within)).
		Count(&n).Error
	return n, err
}

// ExistsRecentDuplicate 用户在窗口内是否已发布过完全相同的内容。
func (r *SpamRepo) ExistsRecentDuplicate(ctx context.Context, table, column string, userID uint64, content string, within time.Duration) (bool, error) {
	if !spamValid(table, column) {
		return false, nil
	}
	var one int64
	err := r.db.WithContext(ctx).Table(table).
		Where("user_id = ? AND "+column+" = ? AND created_at >= ?", userID, content, time.Now().Add(-within)).
		Limit(1).Count(&one).Error
	return one > 0, err
}
