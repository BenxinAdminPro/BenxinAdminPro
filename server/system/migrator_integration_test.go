// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-005 迁移执行器集成测试 — 真库验证建表真实落地（堵假阳性）
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 12:30:00
// +----------------------------------------------------------------------
//
// 运行方式：
//   docker compose -f deploy/docker-compose.dev.yml up -d   # 起 MySQL
//   go test -tags=integration ./system/... -v -count=1 -run TestMigrator
//
// DSN 可经环境变量覆盖（端口冲突时用）：
//   BENXIN_TEST_MYSQL_DSN  默认 root:root@tcp(localhost:3306)/benxinadminpro?...
//
// 本测试专为堵死 #5「迁移器静默不建表却报 applied」的回归：
// 不止断言「Up 不报错」，而是断言【建表数 > 0 且目标表真实存在】——
// 因为旧 bug 正是 Up 返回 nil（假装成功）但一张表都没建。

//go:build integration

package system

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const migTestPrefix = "migtest_"

func migTestDSN() string {
	if v := os.Getenv("BENXIN_TEST_MYSQL_DSN"); v != "" {
		return v
	}
	return "root:root@tcp(localhost:3306)/benxinadminpro?charset=utf8mb4&parseTime=true&loc=Local"
}

func migrationsDirForTest() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "spec", "migrations")
}

// dropMigTestTables 清掉本前缀的所有残留表，保证可重复跑。
func dropMigTestTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	var names []string
	db.Raw("SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name LIKE ?",
		migTestPrefix+"%").Scan(&names)
	if len(names) == 0 {
		return
	}
	db.Exec("SET FOREIGN_KEY_CHECKS=0")
	for _, n := range names {
		db.Exec("DROP TABLE IF EXISTS `" + n + "`")
	}
	db.Exec("SET FOREIGN_KEY_CHECKS=1")
}

func TestMigratorCreatesTablesForReal(t *testing.T) {
	db, err := gorm.Open(mysql.Open(migTestDSN()), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{TablePrefix: migTestPrefix, SingularTable: true},
	})
	if err != nil {
		t.Fatalf("connect mysql (%s): %v", migTestDSN(), err)
	}
	dropMigTestTables(t, db)
	t.Cleanup(func() { dropMigTestTables(t, db) })

	dir := migrationsDirForTest()
	ctx := context.Background()

	// ---- 执行迁移 ----
	if err := NewMigrator(db, migTestPrefix, dir).Up(ctx); err != nil {
		t.Fatalf("migrator Up: %v", err)
	}

	// ---- 断言 1：建表数 > 0（旧 bug 下这里会是 0，但 Up 仍返回 nil）----
	var tableCount int64
	db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name LIKE ?",
		migTestPrefix+"%").Scan(&tableCount)
	if tableCount == 0 {
		t.Fatal("migrator reported success but created ZERO tables — 假阳性回归 (#5)")
	}
	t.Logf("建表数 = %d", tableCount)

	// ---- 断言 2：关键目标表真实存在（不是只看不报错）----
	// 覆盖 CREATE TABLE 路径的代表表，跨 T001~T004b 各切片。
	wantTables := []string{
		"casbin_rule",   // T001
		"sys_user",      // T003a
		"sys_dept",      // T003a
		"sys_post",      // T003a
		"sys_role",      // T003b
		"sys_menu",      // T003b
		"sys_role_menu", // T003b
		"sys_user_role", // T003b
		"sys_config",    // T004a
		"sys_dict_type", // T004a
		"sys_dict_data", // T004a
		"sys_oper_log",  // T004a
		"sys_login_log", // T004a
		"sys_file",      // T004b
		"sys_migration", // 执行器自身基础设施表
	}
	for _, tbl := range wantTables {
		full := migTestPrefix + tbl
		if !db.Migrator().HasTable(full) {
			t.Errorf("expected table %q to exist after migration, but it does not", full)
		}
	}

	// ---- 断言 3：ALTER 语句也真的执行了（不只 CREATE）----
	// T005 给 sys_config 加 is_encrypted 列；T003c 给 sys_role 加 data_scope 列。
	if !db.Migrator().HasColumn(&SysConfig{}, "is_encrypted") {
		t.Error("expected sys_config.is_encrypted column from T005 ALTER migration, but it is missing")
	}
	if !hasColumn(db, migTestPrefix+"sys_role", "data_scope") {
		t.Error("expected sys_role.data_scope column from T003c ALTER migration, but it is missing")
	}

	// ---- 断言 4：sys_migration 记录条数 == spec 下 .sql 文件数（无遗漏、无假记录）----
	wantVersions := countSQLFiles(t, dir)
	var recorded int64
	db.Model(&SysMigration{}).Count(&recorded)
	if recorded != int64(wantVersions) {
		t.Errorf("sys_migration recorded %d versions, want %d (== number of *.sql files)", recorded, wantVersions)
	}
	t.Logf("sys_migration 记录版本数 = %d (== .sql 文件数)", recorded)
}

// TestMigratorStripsLeadingCommentBlock 直接验修复点：带文件头注释块的 SQL 能正确建表。
// 这是 #5 的最小复现：旧实现把"注释头 + CREATE"切成一块当纯注释跳过。
func TestMigratorStripsLeadingCommentBlock(t *testing.T) {
	sql := `-- +----------------------------------------------------------------------
-- | 这是文件头注释块，旧实现会把它和下面的 CREATE 一起当注释跳过
-- +----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS ` + "`migtest_comment_probe`" + ` (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (id)
);`
	stmts := splitStatements(sql)
	var hasCreate bool
	for _, s := range stmts {
		if strings.Contains(strings.ToUpper(strings.TrimSpace(s)), "CREATE TABLE") {
			hasCreate = true
		}
	}
	if !hasCreate {
		t.Fatal("splitStatements 把 CREATE 语句弄丢了 — 注释剥离失败 (#5)")
	}
}

// hasColumn 用 information_schema 直接验某表某列存在（不依赖跨包模型）。
func hasColumn(db *gorm.DB, table, column string) bool {
	var n int64
	db.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?",
		table, column).Scan(&n)
	return n > 0
}

func countSQLFiles(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			n++
		}
	}
	return n
}
