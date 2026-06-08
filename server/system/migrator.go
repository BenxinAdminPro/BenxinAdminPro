// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   SQL 迁移执行器 — 排序 + 替换前缀 + 幂等 + 版本记录
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 03:14:00
// | @updated   2026-06-08 12:30:00  修建表静默失败 bug：先逐行剥 -- 注释再按 ; 切分
// +----------------------------------------------------------------------

package system

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/gorm"
)

// Migrator SQL 迁移执行器。
type Migrator struct {
	db     *gorm.DB
	prefix string // 表前缀（替换 {{TABLE_PREFIX}}）
	dir    string // spec/migrations 目录路径
}

// NewMigrator 创建迁移执行器。
// prefix 来自配置（非用户输入），dir 为受信的 spec/migrations 目录。
func NewMigrator(db *gorm.DB, prefix, dir string) *Migrator {
	return &Migrator{db: db, prefix: prefix, dir: dir}
}

// Up 执行全部未执行的迁移。
// 排序规则：文件名字典序（T001_ < T003a_ < T004a_ < T005_）。
// 幂等：已记录版本跳过。失败中止并报告。
func (m *Migrator) Up(ctx context.Context) error {
	// 确保 sys_migration 表存在
	if err := m.ensureMigrationTable(ctx); err != nil {
		return fmt.Errorf("migrator: ensure migration table: %w", err)
	}

	// 读取全部 SQL 文件
	files, err := m.listSQLFiles()
	if err != nil {
		return fmt.Errorf("migrator: list files: %w", err)
	}

	// 获取已执行版本
	applied, err := m.getApplied(ctx)
	if err != nil {
		return fmt.Errorf("migrator: get applied: %w", err)
	}

	// 按序执行未执行的
	for _, f := range files {
		version := f.Name()
		if applied[version] {
			continue
		}

		content, err := os.ReadFile(filepath.Join(m.dir, version))
		if err != nil {
			return fmt.Errorf("migrator: read %s: %w", version, err)
		}

		// 替换前缀
		sql := strings.ReplaceAll(string(content), "{{TABLE_PREFIX}}", m.prefix)

		// 执行（先剥整行注释再按分号分割，逐条执行）
		statements := splitStatements(sql)
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if err := m.db.WithContext(ctx).Exec(stmt).Error; err != nil {
				return fmt.Errorf("migrator: exec %s: %w", version, err)
			}
		}

		// 记录版本
		checksum := fmt.Sprintf("%x", sha256.Sum256(content))
		m.db.WithContext(ctx).Create(&SysMigration{
			Version:  version,
			Checksum: checksum,
		})
	}

	return nil
}

func (m *Migrator) ensureMigrationTable(_ context.Context) error {
	// 使用 GORM AutoMigrate 建 migration 表（此处豁免"禁 AutoMigrate"规则——
	// 因为 sys_migration 是执行器自身的基础设施表，而非业务表）。
	return m.db.AutoMigrate(&SysMigration{})
}

func (m *Migrator) listSQLFiles() ([]os.DirEntry, error) {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return nil, err
	}
	var sqlFiles []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			sqlFiles = append(sqlFiles, e)
		}
	}
	sort.Slice(sqlFiles, func(i, j int) bool {
		return sqlFiles[i].Name() < sqlFiles[j].Name()
	})
	return sqlFiles, nil
}

func (m *Migrator) getApplied(ctx context.Context) (map[string]bool, error) {
	var versions []string
	m.db.WithContext(ctx).Model(&SysMigration{}).Pluck("version", &versions)
	applied := make(map[string]bool, len(versions))
	for _, v := range versions {
		applied[v] = true
	}
	return applied, nil
}

// stripLineComments 逐行剥掉以 -- 开头的整行注释（保留语句行）。
// 注：仅处理整行注释——spec/migrations 的 DDL 不含行内 -- 注释；
// 这样既清掉文件头注释块，又不会把后面的 CREATE 语句一起吞掉。
func stripLineComments(sql string) string {
	lines := strings.Split(sql, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

// splitStatements 先逐行剥掉 -- 注释，再按分号分割（简单实现，不处理字符串内分号）。
// 旧实现直接 Split(";") 会把"文件头注释 + 首条语句"切成同一块、再被
// "整块以 -- 开头则跳过"的判断误杀，导致 CREATE 静默不执行（假装迁移成功）。
func splitStatements(sql string) []string {
	return strings.Split(stripLineComments(sql), ";")
}
