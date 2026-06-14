// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-008b 增量 — UserService.List 批量回填角色：正确性 + 批量非 N+1 查询计数证明
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-14 16:45:00
// +----------------------------------------------------------------------
//
// 运行方式：go test -tags=integration ./rbac/... -v -count=1 -run TestListUserRoles
// 前置：docker compose -f deploy/docker-compose.dev.yml up -d
//
// 验证：① List 出参每行 roles 正确（一对多分组：A=2角色 B=1 C=0）
//       ② 批量非 N+1：用查询计数器证明 List 的 SQL 次数固定（2行与5行用户页查询次数相同）
//       ③ password_hash 不泄漏（List 用户行不含哈希）
//       ④ 软删角色不出现在 roles（model 查询 deleted_at scope 生效）

//go:build integration

package rbac

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/auth"
	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const lrPrefix = "t008b_"

func setupListRolesMySQL(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(mysql.Open(orgTestDSN), NewDBConfig(lrPrefix))
	if err != nil {
		t.Fatalf("connect mysql: %v", err)
	}
	tables := []string{"sys_user_role", "sys_role", "sys_user"}
	drop := func() {
		for _, tbl := range tables {
			db.Exec("DROP TABLE IF EXISTS `" + lrPrefix + tbl + "`")
		}
	}
	drop()
	dir := orgMigrationDir()
	for _, f := range []string{"T003a_sys_user.sql", "T003b_sys_role.sql", "T003c_sys_role_data_scope.sql", "T003b_sys_user_role.sql"} {
		b, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if err := db.Exec(strings.ReplaceAll(string(b), "{{TABLE_PREFIX}}", lrPrefix)).Error; err != nil {
			t.Fatalf("exec %s: %v", f, err)
		}
	}
	t.Cleanup(drop)
	return db
}

func TestListUserRolesBackfillAndQueryCount(t *testing.T) {
	db := setupListRolesMySQL(t)
	reg, _ := errcode.NewRegistry(11000)
	hasher, _ := auth.NewPasswordHasher(auth.Argon2idParams{
		MemoryKiB: 1024, Iterations: 1, Parallelism: 1, SaltLen: 16, KeyLen: 32,
	})
	svc := NewUserService(db, hasher, reg)
	ctx := context.Background()

	// 角色：r1 r2 + 一个软删角色 rDel
	r1 := SysRole{Code: "r1", Name: "角色一"}
	r2 := SysRole{Code: "r2", Name: "角色二"}
	rDel := SysRole{Code: "rdel", Name: "已删角色"}
	db.Create(&r1)
	db.Create(&r2)
	db.Create(&rDel)
	db.Delete(&rDel) // 软删

	mkUser := func(name string, roleIDs ...uint64) uint64 {
		u, err := svc.Create(ctx, CreateUserInput{Username: name, Password: "pwd123"})
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		for _, rid := range roleIDs {
			db.Create(&SysUserRole{UserID: u.ID, RoleID: rid})
		}
		return u.ID
	}
	// A=2角色(含软删角色 rDel→应被剔除，仍只显 2) B=1 C=0
	mkUser("ua", r1.ID, r2.ID, rDel.ID)
	mkUser("ub", r1.ID)
	mkUser("uc")

	// ① 正确性：List 回填
	res, err := svc.List(ctx, UserListQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	users := res.List.([]SysUser)
	byName := map[string]SysUser{}
	for _, u := range users {
		byName[u.Username] = u
		if u.PasswordHash != "" {
			t.Errorf("③ password_hash 泄漏于 List：user=%s", u.Username) // ③
		}
	}
	if got := len(byName["ua"].Roles); got != 2 {
		t.Errorf("① ua 应 2 角色（软删 rDel 剔除），got %d", got) // ① + ④
	}
	if got := len(byName["ub"].Roles); got != 1 {
		t.Errorf("① ub 应 1 角色，got %d", got)
	}
	if got := len(byName["uc"].Roles); got != 0 {
		t.Errorf("① uc 应 0 角色，got %d", got)
	}
	// ④ 软删角色不在 ua.roles
	for _, r := range byName["ua"].Roles {
		if r.Code == "rdel" {
			t.Error("④ 软删角色 rdel 不应出现在 roles（model 查询 scope 失效）")
		}
	}
	t.Log("① 回填正确（ua=2/ub=1/uc=0）+ ③ 无 password_hash + ④ 软删角色剔除")

	// ② 批量非 N+1：查询计数器证明 List 的 SQL 次数与本页行数无关
	var qCount int
	db.Callback().Query().After("gorm:query").Register("t008b:count", func(d *gorm.DB) { qCount++ })
	defer db.Callback().Query().Remove("t008b:count")

	qCount = 0
	svc.List(ctx, UserListQuery{Page: 1, PageSize: 2}) // 本页 2 行
	q2 := qCount

	// 再加 3 个有角色的用户 → 本页 5 行
	for _, n := range []string{"ud", "ue", "uf"} {
		mkUser(n, r1.ID)
	}
	qCount = 0
	svc.List(ctx, UserListQuery{Page: 1, PageSize: 10}) // 本页 6 行
	q6 := qCount

	// 固定 4 次：Count + 用户 + junction(③) + 角色(④)，与行数 N 解耦
	if q2 != 4 {
		t.Errorf("② 2 行页查询次数应为 4（Count+用户+junction+角色），got %d", q2)
	}
	if q2 != q6 {
		t.Errorf("② 批量非 N+1 被破坏：2 行页 %d 次 ≠ 6 行页 %d 次（应固定不随 N 增长）", q2, q6)
	}
	t.Logf("② 批量非 N+1 证明 OK：2 行页=%d 次查询 == 6 行页=%d 次（固定，不随 N 增长）", q2, q6)
}
