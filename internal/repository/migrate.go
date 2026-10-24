package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/gorm"
)

// RunMigrations 按文件名顺序执行 dir 下未应用的 *.up.sql。
// DSN 需开启 multiStatements=true 以支持单次执行整个文件；
// 版本记录在 schema_migrations 表，保证幂等可重放。
func RunMigrations(db *gorm.DB, dir string) error {
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(64) NOT NULL,
		applied_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
		PRIMARY KEY (version)
	) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4`).Error; err != nil {
		return fmt.Errorf("创建 schema_migrations 失败: %w", err)
	}

	var applied []string
	if err := db.Raw("SELECT version FROM schema_migrations").Scan(&applied).Error; err != nil {
		return fmt.Errorf("读取迁移版本失败: %w", err)
	}
	done := make(map[string]bool, len(applied))
	for _, v := range applied {
		done[v] = true
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("读取迁移目录 %s 失败: %w", dir, err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		files = append(files, e.Name())
	}
	sort.Strings(files)

	for _, name := range files {
		version := strings.TrimSuffix(name, ".up.sql")
		if done[version] {
			continue
		}
		sqlBytes, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("读取迁移 %s 失败: %w", name, err)
		}
		if err := db.Exec(string(sqlBytes)).Error; err != nil {
			return fmt.Errorf("执行迁移 %s 失败: %w", name, err)
		}
		if err := db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version).Error; err != nil {
			return fmt.Errorf("记录迁移版本 %s 失败: %w", version, err)
		}
	}
	return nil
}
