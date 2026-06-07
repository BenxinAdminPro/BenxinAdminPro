// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   Casbin MySQL 集成测试 — spec SQL 建表 + NewEnforcer + Enforce
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 15:36:00
// +----------------------------------------------------------------------
//
// 运行方式：go test -tags=integration ./rbac/... -v -count=1
// 前置：docker compose -f deploy/docker-compose.dev.yml up -d
//
// 说明：
// - rbac 单测（rbac_test.go）使用 file adapter + 内存策略，不依赖 DB。
// - 本集成测试使用真实 MySQL + spec/migrations/T001_casbin_rule.sql 建表，
//   验证 gorm-adapter TurnOffAutoMigrate 后通过 spec SQL 手动建表的完整路径。

//go:build integration

package rbac

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	testDSN         = "root:root@tcp(localhost:3306)/benxinadminpro?charset=utf8mb4&parseTime=true&loc=Local"
	testTablePrefix = "inttest_"
)

func specMigrationDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "spec", "migrations")
}

func setupMySQL(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(mysql.Open(testDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect mysql (is docker compose running?): %v", err)
	}

	// 用 spec SQL 建表，替换占位符
	sqlBytes, err := os.ReadFile(filepath.Join(specMigrationDir(), "T001_casbin_rule.sql"))
	if err != nil {
		t.Fatalf("read migration SQL: %v", err)
	}
	ddl := strings.ReplaceAll(string(sqlBytes), "{{TABLE_PREFIX}}", testTablePrefix)

	// 先清理可能存在的旧表
	tableName := testTablePrefix + "casbin_rule"
	db.Exec("DROP TABLE IF EXISTS `" + tableName + "`")

	// 执行 spec SQL 建表
	if err := db.Exec(ddl).Error; err != nil {
		t.Fatalf("exec migration SQL: %v", err)
	}

	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS `" + tableName + "`")
	})

	return db
}

func TestNewEnforcerMySQL(t *testing.T) {
	db := setupMySQL(t)

	e, err := NewEnforcer(db, modelConfPath(), testTablePrefix)
	if err != nil {
		t.Fatalf("NewEnforcer: %v", err)
	}

	// 无策略时应拒绝
	allowed, err := e.Enforce("alice", "/api/data", "GET")
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	if allowed {
		t.Error("should deny without policy")
	}

	// 通过 enforcer 添加策略（写入 DB）
	_, err = e.AddPolicy("alice", "/api/data", "GET")
	if err != nil {
		t.Fatalf("AddPolicy: %v", err)
	}

	// 验证命中
	allowed, err = e.Enforce("alice", "/api/data", "GET")
	if err != nil {
		t.Fatalf("Enforce after AddPolicy: %v", err)
	}
	if !allowed {
		t.Error("should allow alice GET /api/data after AddPolicy")
	}

	// 验证持久化：重新加载 enforcer，策略应仍在
	e2, err := NewEnforcer(db, modelConfPath(), testTablePrefix)
	if err != nil {
		t.Fatalf("NewEnforcer (reload): %v", err)
	}
	allowed, err = e2.Enforce("alice", "/api/data", "GET")
	if err != nil {
		t.Fatalf("Enforce (reload): %v", err)
	}
	if !allowed {
		t.Error("policy should persist across enforcer reloads")
	}
}

func TestNewEnforcerMySQL_RoleInheritance(t *testing.T) {
	db := setupMySQL(t)

	e, err := NewEnforcer(db, modelConfPath(), testTablePrefix)
	if err != nil {
		t.Fatalf("NewEnforcer: %v", err)
	}

	_, _ = e.AddPolicy("admin", "/api/admin/*", "*")
	_, _ = e.AddGroupingPolicy("alice", "admin")

	allowed, _ := e.Enforce("alice", "/api/admin/users", "POST")
	if !allowed {
		t.Error("alice should inherit admin role permissions")
	}

	allowed, _ = e.Enforce("bob", "/api/admin/users", "GET")
	if allowed {
		t.Error("bob should be denied without role")
	}
}

func TestSpecSQLNoAutoMigrate(t *testing.T) {
	db := setupMySQL(t)

	// 验证表已存在（由 spec SQL 创建，不是 AutoMigrate）
	tableName := testTablePrefix + "casbin_rule"
	var count int64
	err := db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", tableName).Scan(&count).Error
	if err != nil {
		t.Fatalf("check table: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected table %q to exist, found %d matches", tableName, count)
	}

	// 验证索引存在
	var indexCount int64
	err = db.Raw("SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? AND index_name = 'idx_casbin_rule'", tableName).Scan(&indexCount).Error
	if err != nil {
		t.Fatalf("check index: %v", err)
	}
	if indexCount == 0 {
		t.Error("expected unique index idx_casbin_rule to exist")
	}
}
