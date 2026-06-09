// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-004e rbac 唯一键冲突集成测试 — 真 MySQL 重复插入得友好码而非 500
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-09 17:00:00
// +----------------------------------------------------------------------
//
// 运行方式：go test -tags=integration ./rbac/... -v -count=1 -run Dup
// 前置：docker compose -f deploy/docker-compose.dev.yml up -d
//
// 策略：唯一索引不含 deleted_at，软删记录仍占位 → 预检（GORM 软删 scope）看不到旧行、
// 放行到 Create → DB 唯一索引撞 1062 → 这是非竞态的真 500 场景，正是 backstop 的用武之地。
// 复用 specMigrationDir()（mysql_integration_test.go）与 testDSN（同包）。

//go:build integration

package rbac

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/auth"
	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const dupTestPrefix = "t004e_"

func setupDupMySQL(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(mysql.Open(testDSN), NewDBConfig(dupTestPrefix))
	if err != nil {
		t.Fatalf("connect mysql (is docker compose running?): %v", err)
	}
	tables := []string{"sys_user_post", "sys_user", "sys_post", "sys_role"}
	for _, tbl := range tables {
		db.Exec("DROP TABLE IF EXISTS `" + dupTestPrefix + tbl + "`")
	}
	// T003b_sys_role 建基表，T003c ALTER 补 data_scope 列（与生产迁移序一致，否则 role.Create 写 DataScope 撞 1054）
	files := []string{
		"T003a_sys_post.sql", "T003a_sys_user.sql", "T003a_sys_user_post.sql",
		"T003b_sys_role.sql", "T003c_sys_role_data_scope.sql",
	}
	dir := specMigrationDir()
	for _, f := range files {
		b, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		ddl := strings.ReplaceAll(string(b), "{{TABLE_PREFIX}}", dupTestPrefix)
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("exec %s: %v", f, err)
		}
	}
	t.Cleanup(func() {
		for _, tbl := range tables {
			db.Exec("DROP TABLE IF EXISTS `" + dupTestPrefix + tbl + "`")
		}
	})
	return db
}

// assertDupCode 断言 err 是带预期 code 的友好业务错误（非 500/裸 error），且不泄漏 DB 细节。
func assertDupCode(t *testing.T, err error, wantCode int, label string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: 期望唯一键冲突错误，得到 nil", label)
	}
	coded, ok := err.(interface{ GetCode() int })
	if !ok {
		t.Fatalf("%s: 错误非友好业务码（会渲染 500）: %v", label, err)
	}
	if coded.GetCode() != wantCode {
		t.Errorf("%s: code=%d, want=%d", label, coded.GetCode(), wantCode)
	}
	// 中立性：干净的 errcode.Error.Error() 恒为 "errcode:N"，绝不含索引名/表名/原始 SQL。
	// 用精确等值证明"未裹 DB 错误"——子串黑名单会被合法码值误伤（如 11062 含 "1062"）。
	want := fmt.Sprintf("errcode:%d", wantCode)
	if msg := err.Error(); msg != want {
		t.Errorf("%s: 错误串应为干净 %q，实得 %q（疑似裹了 DB 细节/原始 SQL）", label, want, msg)
	}
}

func newDupUserSvc(db *gorm.DB, reg *errcode.Registry) *UserService {
	hasher, _ := auth.NewPasswordHasher(auth.Argon2idParams{
		MemoryKiB: 1024, Iterations: 1, Parallelism: 1, SaltLen: 16, KeyLen: 32,
	})
	return NewUserService(db, hasher, reg)
}

// 头号场景：软删用户名后重建（预检漏判）→ 1062 backstop → ErrUsernameExists（非 500）。
func TestDupUserSoftDeleteRecreate_MySQL(t *testing.T) {
	db := setupDupMySQL(t)
	reg, _ := errcode.NewRegistry(11000)
	svc := newDupUserSvc(db, reg)
	ctx := context.Background()

	u, err := svc.Create(ctx, CreateUserInput{Username: "dupe", Password: "pwd123"})
	if err != nil {
		t.Fatalf("首次创建: %v", err)
	}
	if err := svc.Delete(ctx, u.ID); err != nil {
		t.Fatalf("软删: %v", err)
	}
	_, err = svc.Create(ctx, CreateUserInput{Username: "dupe", Password: "pwd123"})
	assertDupCode(t, err, reg.ErrUsernameExists.GetCode(), "user 软删后重建")
}

// 简单重名（走预检路径）也应是友好码、绝不 500。
func TestDupUserSimpleCreate_MySQL(t *testing.T) {
	db := setupDupMySQL(t)
	reg, _ := errcode.NewRegistry(11000)
	svc := newDupUserSvc(db, reg)
	ctx := context.Background()

	if _, err := svc.Create(ctx, CreateUserInput{Username: "twin", Password: "pwd123"}); err != nil {
		t.Fatalf("首次创建: %v", err)
	}
	_, err := svc.Create(ctx, CreateUserInput{Username: "twin", Password: "pwd123"})
	assertDupCode(t, err, reg.ErrUsernameExists.GetCode(), "user 简单重名")
}

func TestDupRoleSoftDeleteRecreate_MySQL(t *testing.T) {
	db := setupDupMySQL(t)
	reg, _ := errcode.NewRegistry(11000)
	svc := NewRoleService(db, reg, nil)
	ctx := context.Background()

	r, err := svc.Create(ctx, CreateRoleInput{Code: "ROLE_X", Name: "X"})
	if err != nil {
		t.Fatalf("首次创建: %v", err)
	}
	// 直接软删该角色行（不经 Delete 的关联清理，仅需占位触发 1062）
	if err := db.WithContext(ctx).Where("id = ?", r.ID).Delete(&SysRole{}).Error; err != nil {
		t.Fatalf("软删: %v", err)
	}
	_, err = svc.Create(ctx, CreateRoleInput{Code: "ROLE_X", Name: "X2"})
	assertDupCode(t, err, reg.ErrRoleCodeExists.GetCode(), "role 软删后重建")
}

func TestDupPostSoftDeleteRecreate_MySQL(t *testing.T) {
	db := setupDupMySQL(t)
	reg, _ := errcode.NewRegistry(11000)
	svc := NewPostService(db, reg)
	ctx := context.Background()

	p, err := svc.Create(ctx, CreatePostInput{Code: "POST_X", Name: "X"})
	if err != nil {
		t.Fatalf("首次创建: %v", err)
	}
	if err := db.WithContext(ctx).Where("id = ?", p.ID).Delete(&SysPost{}).Error; err != nil {
		t.Fatalf("软删: %v", err)
	}
	_, err = svc.Create(ctx, CreatePostInput{Code: "POST_X", Name: "X2"})
	assertDupCode(t, err, reg.ErrPostCodeExists.GetCode(), "post 软删后重建")
}
