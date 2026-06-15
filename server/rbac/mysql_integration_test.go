// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   Casbin MySQL 集成测试 — spec SQL 建表 + NewEnforcer + Enforce
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 15:36:00
// | @updated   2026-06-15 19:16:47  T-003d-fix：RoleInheritance 陈旧 URL 断言重写为 perm code + 同主体负向对照
// | @updated   2026-06-15 21:07:05  T-010a：testDSN 改读 BENXIN_TEST_MYSQL_DSN（testsupport 收口，默认不变）
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

	"github.com/benxin_dev/benxinadminpro-server/internal/testsupport"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const testTablePrefix = "inttest_"

// testDSN：优先 BENXIN_TEST_MYSQL_DSN，缺省本地默认（dup_integration_test.go 同包复用）。
var testDSN = testsupport.MySQLDSN()

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

// TestNewEnforcerMySQL_RoleInheritance 验证 g 角色继承在真实 MySQL + gorm-adapter
// 持久化路径上确实生效。
//
// T-003d-fix（HEAD dacc8a1）：原断言用 URL/keyMatch 形态（p.obj=/api/admin/*、
// Enforce obj=/api/admin/users），自 T-003b 把 model.conf 由 keyMatch2 改为
// perm code 精确匹配（r.obj == p.obj，见 spec/rbac/model.conf）后即恒红——
// 这是陈旧断言而非鉴权 bug。本次重写为 perm code 形态，并镜像
// rbac_test.go:TestEnforcerRoleInheritance 的 (sub,obj,act) 元组（仅适配器不同：
// 那条走 file adapter，本条走真 MySQL），两者平行便于对照。
//
// 命门=同主体负向对照：alice 本身不持任何直挂 p 规则（admin 才持），故
// allow 必然由 g 继承链承载。先证无链 deny、加链后 allow 真翻转——将来若有人
// 给 alice 误加直挂策略、或 g 在 gorm-adapter 持久化路径回归，本测试会真红
// 而非平凡变绿。setupMySQL 每条用例 DROP+重建 inttest_casbin_rule，table 全新、
// 不污染其它集成测试。
func TestNewEnforcerMySQL_RoleInheritance(t *testing.T) {
	db := setupMySQL(t)

	e, err := NewEnforcer(db, modelConfPath(), testTablePrefix)
	if err != nil {
		t.Fatalf("NewEnforcer: %v", err)
	}

	// admin 角色持 p 规则；alice 无直挂策略。
	_, _ = e.AddPolicy("admin", "sys:user:list", "access")

	// 负向对照（前态）：未建 g(alice,admin) 时 alice 应被拒——
	// 坐实后续 allow 来自继承而非任何直挂策略。
	allowed, _ := e.Enforce("alice", "sys:user:list", "access")
	if allowed {
		t.Error("alice should be denied before grouping policy (allow must come from inheritance, not a direct policy)")
	}

	// 建立继承链 alice → admin。
	_, _ = e.AddGroupingPolicy("alice", "admin")

	// 后态：加链后翻转为 allow——g 角色继承在真实 MySQL+gorm-adapter 路径生效。
	allowed, _ = e.Enforce("alice", "sys:user:list", "access")
	if !allowed {
		t.Error("alice should inherit admin role permissions after grouping policy added")
	}

	// act 通配维度（保留原测试 p.act="*" 覆盖，不静默丢失）：
	// admin 另持一条 act=* 的 perm，alice 经继承应对任意 act 命中。
	_, _ = e.AddPolicy("admin", "sys:user:create", "*")
	allowed, _ = e.Enforce("alice", "sys:user:create", "anyact")
	if !allowed {
		t.Error("alice should inherit admin's wildcard-act permission (act=* path)")
	}

	// bob 基线：无角色始终拒绝。
	allowed, _ = e.Enforce("bob", "sys:user:list", "access")
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
