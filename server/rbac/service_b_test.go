// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-003b 角色/菜单/联动 service 单元测试 — SQLite 内存
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 21:40:00
// +----------------------------------------------------------------------

package rbac

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/auth"
	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/casbin/casbin/v2"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func setupTestDBb(t *testing.T) *gorm.DB {
	t.Helper()
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	db.AutoMigrate(&SysUser{}, &SysDept{}, &SysPost{}, &SysUserPost{},
		&SysRole{}, &SysMenu{}, &SysRoleMenu{}, &SysUserRole{})
	return db
}

func testEnforcerWithFile(t *testing.T) *casbin.Enforcer {
	t.Helper()
	tmpDir := t.TempDir()
	policyFile := filepath.Join(tmpDir, "policy.csv")
	os.WriteFile(policyFile, []byte(""), 0o644)
	e, _ := casbin.NewEnforcer(modelConfPath(), fileadapter.NewAdapter(policyFile))
	return e
}

// ---------------------------------------------------------------------------
// 角色 CRUD
// ---------------------------------------------------------------------------

func TestRoleCRUD(t *testing.T) {
	db := setupTestDBb(t)
	reg, _ := errcode.NewRegistry(11000)
	svc := NewRoleService(db, reg, nil)
	ctx := context.Background()

	role, err := svc.Create(ctx, CreateRoleInput{Code: "admin", Name: "管理员"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if role.Code != "admin" {
		t.Errorf("code: %q", role.Code)
	}

	// 重复 code
	_, err = svc.Create(ctx, CreateRoleInput{Code: "admin", Name: "另一个"})
	if err == nil {
		t.Error("expected ErrRoleCodeExists")
	}

	// 列表
	result, _ := svc.List(ctx, RoleListQuery{Page: 1, PageSize: 10})
	if result.Total != 1 {
		t.Errorf("total: %d", result.Total)
	}
}

func TestRoleDeleteInUse(t *testing.T) {
	db := setupTestDBb(t)
	reg, _ := errcode.NewRegistry(11000)
	svc := NewRoleService(db, reg, nil)
	ctx := context.Background()

	role, _ := svc.Create(ctx, CreateRoleInput{Code: "editor", Name: "编辑"})
	db.Create(&SysUserRole{UserID: 1, RoleID: role.ID})

	err := svc.Delete(ctx, role.ID)
	ecErr, ok := err.(errcode.Error)
	if !ok || ecErr.Code != reg.ErrRoleInUse.Code {
		t.Errorf("expected ErrRoleInUse, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// 菜单 CRUD + menu_type 校验
// ---------------------------------------------------------------------------

func TestMenuCreateTypeValidation(t *testing.T) {
	db := setupTestDBb(t)
	reg, _ := errcode.NewRegistry(11000)
	svc := NewMenuService(db, reg)
	ctx := context.Background()

	// F 类型必带 perm_code
	_, err := svc.Create(ctx, CreateMenuInput{MenuType: "F", Name: "查看按钮"})
	if err == nil {
		t.Error("F type without perm_code should fail")
	}

	// F 类型带 perm_code → 成功
	btn, err := svc.Create(ctx, CreateMenuInput{MenuType: "F", Name: "查看按钮", PermCode: "sys:user:list"})
	if err != nil {
		t.Fatalf("Create F: %v", err)
	}
	if btn.PermCode != "sys:user:list" {
		t.Error("perm_code should be set")
	}

	// M 类型不能带 perm_code
	_, err = svc.Create(ctx, CreateMenuInput{MenuType: "M", Name: "系统管理", PermCode: "sys:manage"})
	if err == nil {
		t.Error("M type with perm_code should fail")
	}

	// C 类型不能带 perm_code
	_, err = svc.Create(ctx, CreateMenuInput{MenuType: "C", Name: "用户管理", PermCode: "sys:user"})
	if err == nil {
		t.Error("C type with perm_code should fail")
	}
}

func TestMenuPermCodeUnique(t *testing.T) {
	db := setupTestDBb(t)
	reg, _ := errcode.NewRegistry(11000)
	svc := NewMenuService(db, reg)
	ctx := context.Background()

	svc.Create(ctx, CreateMenuInput{MenuType: "F", Name: "按钮1", PermCode: "sys:user:list"})
	_, err := svc.Create(ctx, CreateMenuInput{MenuType: "F", Name: "按钮2", PermCode: "sys:user:list"})
	ecErr, ok := err.(errcode.Error)
	if !ok || ecErr.Code != reg.ErrPermCodeExists.Code {
		t.Errorf("expected ErrPermCodeExists, got %v", err)
	}
}

func TestMenuTree(t *testing.T) {
	db := setupTestDBb(t)
	reg, _ := errcode.NewRegistry(11000)
	svc := NewMenuService(db, reg)
	ctx := context.Background()

	dir, _ := svc.Create(ctx, CreateMenuInput{MenuType: "M", Name: "系统管理"})
	svc.Create(ctx, CreateMenuInput{MenuType: "C", Name: "用户管理", ParentID: dir.ID, Path: "/sys/user"})

	tree, _ := svc.Tree(ctx)
	if len(tree) != 1 || len(tree[0].Children) != 1 {
		t.Error("tree structure unexpected")
	}
}

func TestMenuDeleteHasChildren(t *testing.T) {
	db := setupTestDBb(t)
	reg, _ := errcode.NewRegistry(11000)
	svc := NewMenuService(db, reg)
	ctx := context.Background()

	dir, _ := svc.Create(ctx, CreateMenuInput{MenuType: "M", Name: "根"})
	svc.Create(ctx, CreateMenuInput{MenuType: "C", Name: "子", ParentID: dir.ID})

	err := svc.Delete(ctx, dir.ID)
	ecErr, ok := err.(errcode.Error)
	if !ok || ecErr.Code != reg.ErrMenuHasChildren.Code {
		t.Errorf("expected ErrMenuHasChildren, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// 用户分配角色
// ---------------------------------------------------------------------------

func TestUserAssignRoles(t *testing.T) {
	db := setupTestDBb(t)
	reg, _ := errcode.NewRegistry(11000)
	hasher, _ := auth.NewPasswordHasher(testArgon2Params)
	userSvc := NewUserService(db, hasher, reg)
	ctx := context.Background()

	user, _ := userSvc.Create(ctx, CreateUserInput{Username: "assigntest", Password: "p"})
	db.Create(&SysRole{Code: "admin", Name: "管理员"})
	db.Create(&SysRole{Code: "editor", Name: "编辑"})

	var roles []SysRole
	db.Find(&roles)

	var roleIDs []uint64
	for _, r := range roles {
		roleIDs = append(roleIDs, r.ID)
	}

	if err := userSvc.AssignRoles(ctx, user.ID, roleIDs); err != nil {
		t.Fatalf("AssignRoles: %v", err)
	}

	// 验证关联
	var urs []SysUserRole
	db.Where("user_id = ?", user.ID).Find(&urs)
	if len(urs) != 2 {
		t.Errorf("expected 2 user-role records, got %d", len(urs))
	}
}

// ---------------------------------------------------------------------------
// Casbin 联动（PolicySync + enforcer）
// ---------------------------------------------------------------------------

func TestPolicySyncRolePerms(t *testing.T) {
	e := testEnforcerWithFile(t)
	db := setupTestDBb(t)
	ps := NewPolicySync(e, db)
	ctx := context.Background()

	// 添加角色权限
	ps.SyncRolePerms(ctx, "admin", []string{"sys:user:list", "sys:user:create"})

	allowed, _ := e.Enforce("admin", "sys:user:list", "access")
	if !allowed {
		t.Error("should allow after SyncRolePerms")
	}

	// 撤销一个权限
	ps.SyncRolePerms(ctx, "admin", []string{"sys:user:list"})
	allowed, _ = e.Enforce("admin", "sys:user:create", "access")
	if allowed {
		t.Error("should deny removed perm after re-sync")
	}
	allowed, _ = e.Enforce("admin", "sys:user:list", "access")
	if !allowed {
		t.Error("should still allow retained perm")
	}
}

func TestPolicySyncUserRoles(t *testing.T) {
	e := testEnforcerWithFile(t)
	db := setupTestDBb(t)
	ps := NewPolicySync(e, db)
	ctx := context.Background()

	ps.SyncRolePerms(ctx, "admin", []string{"sys:user:list"})
	ps.SyncUserRoles(ctx, "1", []string{"admin"})

	allowed, _ := e.Enforce("1", "sys:user:list", "access")
	if !allowed {
		t.Error("user 1 should inherit admin perms")
	}

	// 移除角色
	ps.SyncUserRoles(ctx, "1", []string{})
	allowed, _ = e.Enforce("1", "sys:user:list", "access")
	if allowed {
		t.Error("user 1 should be denied after role removal")
	}
}

// ---------------------------------------------------------------------------
// 权限下发同源
// ---------------------------------------------------------------------------

func TestUserMenusAndPerms(t *testing.T) {
	db := setupTestDBb(t)
	reg, _ := errcode.NewRegistry(11000)
	menuSvc := NewMenuService(db, reg)
	ctx := context.Background()

	// 建菜单树：目录 → 页面 → 按钮
	dir, _ := menuSvc.Create(ctx, CreateMenuInput{MenuType: "M", Name: "系统管理", Visible: 1})
	page, _ := menuSvc.Create(ctx, CreateMenuInput{MenuType: "C", Name: "用户管理", ParentID: dir.ID, Path: "/sys/user", Visible: 1})
	menuSvc.Create(ctx, CreateMenuInput{MenuType: "F", Name: "查看", ParentID: page.ID, PermCode: "sys:user:list"})
	menuSvc.Create(ctx, CreateMenuInput{MenuType: "F", Name: "新建", ParentID: page.ID, PermCode: "sys:user:create"})

	// 建角色并分配菜单
	role := SysRole{Code: "admin", Name: "管理员"}
	db.Create(&role)

	allMenuIDs := []uint64{dir.ID, page.ID}
	// 加上按钮
	var btnIDs []uint64
	db.Model(&SysMenu{}).Where("menu_type = 'F'").Pluck("id", &btnIDs)
	allMenuIDs = append(allMenuIDs, btnIDs...)

	db.Create([]SysRoleMenu{
		{RoleID: role.ID, MenuID: dir.ID},
		{RoleID: role.ID, MenuID: page.ID},
	})
	for _, bid := range btnIDs {
		db.Create(&SysRoleMenu{RoleID: role.ID, MenuID: bid})
	}

	// 用户分配角色
	db.Create(&SysUserRole{UserID: 1, RoleID: role.ID})

	// 获取菜单树（只含 M/C）
	tree, _ := menuSvc.GetUserMenuTree(ctx, 1)
	if len(tree) != 1 {
		t.Fatalf("expected 1 root menu, got %d", len(tree))
	}

	// 获取权限码集合
	perms, _ := menuSvc.GetUserPermCodes(ctx, 1)
	if len(perms) != 2 {
		t.Errorf("expected 2 perm codes, got %d: %v", len(perms), perms)
	}
}
