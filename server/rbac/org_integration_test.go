// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-003a 组织架构 MySQL 集成测试 — spec SQL 建表 + CRUD + GormUserProvider
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 19:35:00
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
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	orgTestDSN         = "root:root@tcp(localhost:3306)/benxinadminpro?charset=utf8mb4&parseTime=true&loc=Local"
	orgTestTablePrefix = "t003a_"
)

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
