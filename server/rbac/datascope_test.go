// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   数据权限测试 — 解析器合并矩阵 + ApplyScope 安全失败 + 自测样例
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 23:48:00
// +----------------------------------------------------------------------

package rbac

import (
	"context"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/auth"
	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func setupScopeDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	db.AutoMigrate(&SysUser{}, &SysRole{}, &SysUserRole{}, &SysDept{})
	return db
}

// ---------------------------------------------------------------------------
// ScopeResolver 合并矩阵
// ---------------------------------------------------------------------------

func TestScopeResolver_SingleRoleAll(t *testing.T) {
	db := setupScopeDB(t)
	db.Create(&SysUser{ID: 1, Username: "u1"})
	db.Create(&SysRole{ID: 1, Code: "admin", DataScope: 1})
	db.Create(&SysUserRole{UserID: 1, RoleID: 1})

	r := NewGormScopeResolver(db, nil)
	ds, err := r.Resolve(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if !ds.All || ds.Type != ScopeAll {
		t.Error("single All role should resolve to All")
	}
}

func TestScopeResolver_SingleRoleSelf(t *testing.T) {
	db := setupScopeDB(t)
	db.Create(&SysUser{ID: 1, Username: "u1"})
	db.Create(&SysRole{ID: 1, Code: "viewer", DataScope: 2})
	db.Create(&SysUserRole{UserID: 1, RoleID: 1})

	r := NewGormScopeResolver(db, nil)
	ds, _ := r.Resolve(context.Background(), 1)
	if ds.Type != ScopeSelf {
		t.Errorf("expected ScopeSelf, got %d", ds.Type)
	}
	if ds.SelfID != 1 {
		t.Errorf("SelfID should be 1, got %d", ds.SelfID)
	}
}

func TestScopeResolver_SingleRoleDept(t *testing.T) {
	db := setupScopeDB(t)
	deptID := uint64(10)
	db.Create(&SysUser{ID: 1, Username: "u1", DeptID: &deptID})
	db.Create(&SysRole{ID: 1, Code: "mgr", DataScope: 3})
	db.Create(&SysUserRole{UserID: 1, RoleID: 1})

	r := NewGormScopeResolver(db, nil)
	ds, _ := r.Resolve(context.Background(), 1)
	if ds.Type != ScopeDept {
		t.Errorf("expected ScopeDept, got %d", ds.Type)
	}
	if len(ds.DeptIDs) != 1 || ds.DeptIDs[0] != 10 {
		t.Errorf("DeptIDs should be [10], got %v", ds.DeptIDs)
	}
}

func TestScopeResolver_MultiRole_AnyAllShortCircuit(t *testing.T) {
	db := setupScopeDB(t)
	db.Create(&SysUser{ID: 1, Username: "u1"})
	db.Create(&SysRole{ID: 1, Code: "viewer", DataScope: 2}) // Self
	db.Create(&SysRole{ID: 2, Code: "admin", DataScope: 1})  // All
	db.Create(&SysUserRole{UserID: 1, RoleID: 1})
	db.Create(&SysUserRole{UserID: 1, RoleID: 2})

	r := NewGormScopeResolver(db, nil)
	ds, _ := r.Resolve(context.Background(), 1)
	if !ds.All {
		t.Error("any All role should short-circuit to All")
	}
}

func TestScopeResolver_MultiRole_DeptPlusSelf(t *testing.T) {
	db := setupScopeDB(t)
	deptID := uint64(5)
	db.Create(&SysUser{ID: 1, Username: "u1", DeptID: &deptID})
	db.Create(&SysRole{ID: 1, Code: "self", DataScope: 2})  // Self
	db.Create(&SysRole{ID: 2, Code: "dept", DataScope: 3})  // Dept
	db.Create(&SysUserRole{UserID: 1, RoleID: 1})
	db.Create(&SysUserRole{UserID: 1, RoleID: 2})

	r := NewGormScopeResolver(db, nil)
	ds, _ := r.Resolve(context.Background(), 1)
	// Dept > Self，取最宽
	if ds.Type != ScopeDept {
		t.Errorf("Dept+Self should resolve to Dept, got %d", ds.Type)
	}
	if ds.SelfID != 1 {
		t.Error("SelfID should still be set")
	}
}

func TestScopeResolver_SuperAdminBypass(t *testing.T) {
	db := setupScopeDB(t)
	db.Create(&SysUser{ID: 1, Username: "u1"})
	db.Create(&SysRole{ID: 1, Code: "super_admin", DataScope: 2}) // DataScope 是 Self，但超管短路
	db.Create(&SysUserRole{UserID: 1, RoleID: 1})

	r := NewGormScopeResolver(db, []string{"super_admin"})
	ds, _ := r.Resolve(context.Background(), 1)
	if !ds.All {
		t.Error("super admin should bypass to All regardless of data_scope value")
	}
}

func TestScopeResolver_NoRoles(t *testing.T) {
	db := setupScopeDB(t)
	db.Create(&SysUser{ID: 1, Username: "u1"})

	r := NewGormScopeResolver(db, nil)
	ds, _ := r.Resolve(context.Background(), 1)
	// 无角色 → fail-closed → ScopeSelf
	if ds.Type != ScopeSelf {
		t.Errorf("no roles should default to ScopeSelf, got %d", ds.Type)
	}
}

func TestScopeResolver_DeptRoleButNoDeptID(t *testing.T) {
	db := setupScopeDB(t)
	db.Create(&SysUser{ID: 1, Username: "u1"}) // 无 dept_id
	db.Create(&SysRole{ID: 1, Code: "dept", DataScope: 3})
	db.Create(&SysUserRole{UserID: 1, RoleID: 1})

	r := NewGormScopeResolver(db, nil)
	ds, _ := r.Resolve(context.Background(), 1)
	if ds.Type != ScopeDept {
		t.Errorf("expected ScopeDept, got %d", ds.Type)
	}
	// DeptIDs 应为空 → ApplyScope 会安全失败
	if len(ds.DeptIDs) != 0 {
		t.Errorf("DeptIDs should be empty when user has no dept_id, got %v", ds.DeptIDs)
	}
}

// ---------------------------------------------------------------------------
// ApplyScope 安全失败断言
// ---------------------------------------------------------------------------

func TestApplyScope_NilScope(t *testing.T) {
	db := setupScopeDB(t)
	seedUsers(db)

	var count int64
	db.Model(&SysUser{}).Scopes(ApplyScope(nil, ScopeFields{UserColumn: "id"})).Count(&count)
	if count != 0 {
		t.Errorf("nil scope should block all, got %d", count)
	}
}

func TestApplyScope_All(t *testing.T) {
	db := setupScopeDB(t)
	seedUsers(db)

	var count int64
	ds := &DataScope{Type: ScopeAll, All: true}
	db.Model(&SysUser{}).Scopes(ApplyScope(ds, ScopeFields{})).Count(&count)
	if count != 3 {
		t.Errorf("All scope should return all 3 users, got %d", count)
	}
}

func TestApplyScope_Self(t *testing.T) {
	db := setupScopeDB(t)
	seedUsers(db)

	var count int64
	ds := &DataScope{Type: ScopeSelf, SelfID: 1}
	db.Model(&SysUser{}).Scopes(ApplyScope(ds, ScopeFields{UserColumn: "id"})).Count(&count)
	if count != 1 {
		t.Errorf("Self scope should return 1 user, got %d", count)
	}
}

func TestApplyScope_Self_EmptyUserColumn(t *testing.T) {
	db := setupScopeDB(t)
	seedUsers(db)

	var count int64
	ds := &DataScope{Type: ScopeSelf, SelfID: 1}
	db.Model(&SysUser{}).Scopes(ApplyScope(ds, ScopeFields{UserColumn: ""})).Count(&count)
	if count != 0 {
		t.Errorf("empty UserColumn should block all (safety), got %d", count)
	}
}

func TestApplyScope_Self_ZeroSelfID(t *testing.T) {
	db := setupScopeDB(t)
	seedUsers(db)

	var count int64
	ds := &DataScope{Type: ScopeSelf, SelfID: 0}
	db.Model(&SysUser{}).Scopes(ApplyScope(ds, ScopeFields{UserColumn: "id"})).Count(&count)
	if count != 0 {
		t.Errorf("zero SelfID should block all (safety), got %d", count)
	}
}

func TestApplyScope_Dept(t *testing.T) {
	db := setupScopeDB(t)
	seedUsers(db)

	var count int64
	ds := &DataScope{Type: ScopeDept, DeptIDs: []uint64{10}}
	db.Model(&SysUser{}).Scopes(ApplyScope(ds, ScopeFields{DeptColumn: "dept_id"})).Count(&count)
	if count != 2 {
		t.Errorf("Dept scope with dept_id=10 should return 2 users, got %d", count)
	}
}

func TestApplyScope_Dept_EmptyDeptColumn(t *testing.T) {
	db := setupScopeDB(t)
	seedUsers(db)

	var count int64
	ds := &DataScope{Type: ScopeDept, DeptIDs: []uint64{10}}
	db.Model(&SysUser{}).Scopes(ApplyScope(ds, ScopeFields{DeptColumn: ""})).Count(&count)
	if count != 0 {
		t.Errorf("empty DeptColumn should block all (safety), got %d", count)
	}
}

func TestApplyScope_Dept_EmptyDeptIDs(t *testing.T) {
	db := setupScopeDB(t)
	seedUsers(db)

	var count int64
	ds := &DataScope{Type: ScopeDept, DeptIDs: []uint64{}}
	db.Model(&SysUser{}).Scopes(ApplyScope(ds, ScopeFields{DeptColumn: "dept_id"})).Count(&count)
	if count != 0 {
		t.Errorf("empty DeptIDs should block all (safety), got %d", count)
	}
}

func TestApplyScope_UnknownType(t *testing.T) {
	db := setupScopeDB(t)
	seedUsers(db)

	var count int64
	ds := &DataScope{Type: 99}
	db.Model(&SysUser{}).Scopes(ApplyScope(ds, ScopeFields{UserColumn: "id"})).Count(&count)
	if count != 0 {
		t.Errorf("unknown type should block all (safety), got %d", count)
	}
}

// ---------------------------------------------------------------------------
// 自测样例：UserService.List + DataScope 过滤行数
// ---------------------------------------------------------------------------

func TestUserListWithDataScope(t *testing.T) {
	db := setupScopeDB(t)
	reg, _ := errcode.NewRegistry(11000)
	hasher, _ := auth.NewPasswordHasher(testArgon2Params)
	svc := NewUserService(db, hasher, reg)
	ctx := context.Background()

	dept10 := uint64(10)
	dept20 := uint64(20)
	svc.Create(ctx, CreateUserInput{Username: "a", Password: "p", DeptID: &dept10})
	svc.Create(ctx, CreateUserInput{Username: "b", Password: "p", DeptID: &dept10})
	svc.Create(ctx, CreateUserInput{Username: "c", Password: "p", DeptID: &dept20})

	// Self: 只看自己（user id=1）
	selfScope := &DataScope{Type: ScopeSelf, SelfID: 1}
	r1, _ := svc.List(ctx, UserListQuery{Page: 1, PageSize: 10, Scope: selfScope})
	if r1.Total != 1 {
		t.Errorf("Self scope: expected 1, got %d", r1.Total)
	}

	// Dept: 看 dept_id=10 的用户
	deptScope := &DataScope{Type: ScopeDept, DeptIDs: []uint64{10}}
	r2, _ := svc.List(ctx, UserListQuery{Page: 1, PageSize: 10, Scope: deptScope})
	if r2.Total != 2 {
		t.Errorf("Dept scope: expected 2, got %d", r2.Total)
	}

	// All: 看全部
	allScope := &DataScope{Type: ScopeAll, All: true}
	r3, _ := svc.List(ctx, UserListQuery{Page: 1, PageSize: 10, Scope: allScope})
	if r3.Total != 3 {
		t.Errorf("All scope: expected 3, got %d", r3.Total)
	}
}

// ---------------------------------------------------------------------------
// 辅助
// ---------------------------------------------------------------------------

func seedUsers(db *gorm.DB) {
	dept10 := uint64(10)
	dept20 := uint64(20)
	db.Create(&SysUser{ID: 1, Username: "u1", DeptID: &dept10})
	db.Create(&SysUser{ID: 2, Username: "u2", DeptID: &dept10})
	db.Create(&SysUser{ID: 3, Username: "u3", DeptID: &dept20})
}
