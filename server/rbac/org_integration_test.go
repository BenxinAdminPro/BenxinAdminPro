// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-003a 组织架构 MySQL 集成测试 — spec SQL 建表 + CRUD + GormUserProvider
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 19:35:00
// | @updated   2026-06-15 21:07:05  T-010a：orgTestDSN 改读 BENXIN_TEST_MYSQL_DSN（testsupport 收口，默认不变）
// +----------------------------------------------------------------------
//
// 运行方式：go test -tags=integration ./rbac/... -v -count=1 -run TestOrg
// 前置：docker compose -f deploy/docker-compose.dev.yml up -d

//go:build integration

package rbac

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/auth"
	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/benxin_dev/benxinadminpro-server/internal/testsupport"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const orgTestTablePrefix = "t003a_"

// orgTestDSN：优先 BENXIN_TEST_MYSQL_DSN，缺省本地默认。
var orgTestDSN = testsupport.MySQLDSN()

func orgMigrationDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "spec", "migrations")
}

func setupOrgMySQL(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(mysql.Open(orgTestDSN), NewDBConfig(orgTestTablePrefix))
	if err != nil {
		t.Fatalf("connect mysql: %v", err)
	}

	// 清理旧表
	tables := []string{"sys_user_post", "sys_user", "sys_dept", "sys_post"}
	for _, tbl := range tables {
		db.Exec("DROP TABLE IF EXISTS `" + orgTestTablePrefix + tbl + "`")
	}

	// 用 spec SQL 建表
	sqlFiles := []string{
		"T003a_sys_dept.sql",
		"T003a_sys_post.sql",
		"T003a_sys_user.sql",
		"T003a_sys_user_post.sql",
	}
	dir := orgMigrationDir()
	for _, f := range sqlFiles {
		sqlBytes, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		ddl := strings.ReplaceAll(string(sqlBytes), "{{TABLE_PREFIX}}", orgTestTablePrefix)
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("exec %s: %v", f, err)
		}
	}

	t.Cleanup(func() {
		for _, tbl := range tables {
			db.Exec("DROP TABLE IF EXISTS `" + orgTestTablePrefix + tbl + "`")
		}
	})

	return db
}

func TestOrgUserCRUD_MySQL(t *testing.T) {
	db := setupOrgMySQL(t)
	reg, _ := errcode.NewRegistry(11000)
	hasher, _ := auth.NewPasswordHasher(auth.Argon2idParams{
		MemoryKiB: 1024, Iterations: 1, Parallelism: 1, SaltLen: 16, KeyLen: 32,
	})
	svc := NewUserService(db, hasher, reg)
	ctx := context.Background()

	// Create
	user, err := svc.Create(ctx, CreateUserInput{
		Username: "inttest_user", Password: "pwd123",
		Nickname: "集成测试",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if user.PasswordHash != "" {
		t.Error("response should not contain password_hash")
	}

	// Get
	got, err := svc.Get(ctx, user.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Nickname != "集成测试" {
		t.Errorf("nickname: %q", got.Nickname)
	}

	// List
	result, _ := svc.List(ctx, UserListQuery{Page: 1, PageSize: 10})
	if result.Total != 1 {
		t.Errorf("total: %d", result.Total)
	}

	// Delete
	svc.Delete(ctx, user.ID)
	_, err = svc.Get(ctx, user.ID)
	if err == nil {
		t.Error("should not find after delete")
	}
}

func TestOrgDeptTree_MySQL(t *testing.T) {
	db := setupOrgMySQL(t)
	reg, _ := errcode.NewRegistry(11000)
	svc := NewDeptService(db, reg)
	ctx := context.Background()

	root, _ := svc.Create(ctx, CreateDeptInput{Name: "总公司"})
	child, _ := svc.Create(ctx, CreateDeptInput{Name: "技术部", ParentID: root.ID})

	if !strings.Contains(child.Ancestors, uintStr(root.ID)) {
		t.Errorf("ancestors should contain root ID, got %q", child.Ancestors)
	}

	tree, _ := svc.Tree(ctx)
	if len(tree) != 1 || len(tree[0].Children) != 1 {
		t.Errorf("tree structure unexpected: roots=%d", len(tree))
	}
}

func TestOrgGormUserProvider_MySQL(t *testing.T) {
	db := setupOrgMySQL(t)
	reg, _ := errcode.NewRegistry(11000)
	hasher, _ := auth.NewPasswordHasher(auth.Argon2idParams{
		MemoryKiB: 1024, Iterations: 1, Parallelism: 1, SaltLen: 16, KeyLen: 32,
	})
	userSvc := NewUserService(db, hasher, reg)
	ctx := context.Background()

	userSvc.Create(ctx, CreateUserInput{Username: "gorm_provider", Password: "secret"})

	provider := NewGormUserProvider(db)
	authUser, err := provider.FindByUsername(ctx, "gorm_provider")
	if err != nil {
		t.Fatalf("FindByUsername: %v", err)
	}
	if authUser.Username != "gorm_provider" {
		t.Errorf("username: %q", authUser.Username)
	}

	// 验证密码
	ok, _ := hasher.Verify("secret", authUser.PasswordHash)
	if !ok {
		t.Error("password should verify via GormUserProvider")
	}

	// Not found
	_, err = provider.FindByUsername(ctx, "no_such_user")
	if err != auth.ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestOrgSpecSQLNoAutoMigrate_MySQL(t *testing.T) {
	db := setupOrgMySQL(t)

	// 验证表已存在（由 spec SQL 创建）
	tables := []string{"sys_user", "sys_dept", "sys_post", "sys_user_post"}
	for _, tbl := range tables {
		var count int64
		fullName := orgTestTablePrefix + tbl
		db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", fullName).Scan(&count)
		if count != 1 {
			t.Errorf("expected table %q to exist", fullName)
		}
	}
}

// TestOrgSetStatusSameValue_MySQL T-009b 命门：在真实 MySQL 上坐实 SetStatus 三态。
// SQLite 同值 UPDATE 返 RowsAffected==1（无法暴露缺陷），唯有 MySQL（默认不带
// CLIENT_FOUND_ROWS）同值返 RowsAffected==0——旧实现据此误返 404。本测试三态断言：
// ① 同值 → 成功（不 404，修复核心）② 不存在 id → 仍 ErrUserNotFound（存在性未放宽，关键负例）
// ③ 真变更 → 成功且落值。
func TestOrgSetStatusSameValue_MySQL(t *testing.T) {
	db := setupOrgMySQL(t)
	reg, _ := errcode.NewRegistry(11000)
	hasher, _ := auth.NewPasswordHasher(auth.Argon2idParams{
		MemoryKiB: 1024, Iterations: 1, Parallelism: 1, SaltLen: 16, KeyLen: 32,
	})
	svc := NewUserService(db, hasher, reg)
	ctx := context.Background()

	user, err := svc.Create(ctx, CreateUserInput{Username: "st_same", Password: "p"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// 先把 status 设到已知值 1，建立"已是该值"前提
	if err := svc.SetStatus(ctx, user.ID, 1); err != nil {
		t.Fatalf("SetStatus to 1: %v", err)
	}

	// ① 同值（1→1）→ MySQL RowsAffected==0，修复后须返成功（旧实现在此误返 404）
	if err := svc.SetStatus(ctx, user.ID, 1); err != nil {
		t.Errorf("SetStatus same value(1→1): got %v, want nil（MySQL 同值误返 404 缺陷未修）", err)
	}

	// ② 不存在 id → 仍 ErrUserNotFound（关键负例：存在性校验未放宽）
	err = svc.SetStatus(ctx, 99999999, 1)
	if err == nil {
		t.Fatal("SetStatus on missing id: expected ErrUserNotFound, got nil（存在性被放宽=安全倒退）")
	}
	if c, ok := err.(interface{ GetCode() int }); !ok || c.GetCode() != reg.ErrUserNotFound.Code {
		t.Errorf("SetStatus missing id: got %v, want ErrUserNotFound(%d)", err, reg.ErrUserNotFound.Code)
	}

	// ③ 真变更（1→0）→ 成功且落值
	if err := svc.SetStatus(ctx, user.ID, 0); err != nil {
		t.Errorf("SetStatus real change(1→0): %v", err)
	}
	got, _ := svc.Get(ctx, user.ID)
	if got.Status != 0 {
		t.Errorf("status after change: got %d, want 0", got.Status)
	}
}
