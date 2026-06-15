// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   组织架构 service 单元测试 — SQLite 内存库，docker-free
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 19:30:00
// +----------------------------------------------------------------------

package rbac

import (
	"context"
	"strconv"
	"testing"

	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/benxin_dev/benxinadminpro-server/auth"
	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var testArgon2Params = auth.Argon2idParams{
	MemoryKiB: 1024, Iterations: 1, Parallelism: 1, SaltLen: 16, KeyLen: 32,
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return setupTestDBWithPrefix(t, "")
}

func setupTestDBWithPrefix(t *testing.T, prefix string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   prefix,
			SingularTable: true,
		},
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// AutoMigrate 仅在测试中使用 SQLite（生产走 spec SQL + MySQL）
	if err := db.AutoMigrate(&SysUser{}, &SysDept{}, &SysPost{}, &SysUserPost{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func testReg(t *testing.T) *errcode.Registry {
	t.Helper()
	reg, _ := errcode.NewRegistry(11000)
	return reg
}

// ---------------------------------------------------------------------------
// UserService 测试
// ---------------------------------------------------------------------------

func TestUserCreate(t *testing.T) {
	db := setupTestDB(t)
	reg := testReg(t)
	hasher, _ := auth.NewPasswordHasher(testArgon2Params)
	svc := NewUserService(db, hasher, reg)
	ctx := context.Background()

	user, err := svc.Create(ctx, CreateUserInput{
		Username: "alice", Password: "test123",
		Nickname: "Alice", PostIDs: []uint64{},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if user.Username != "alice" {
		t.Errorf("username: got %q", user.Username)
	}
	if user.PasswordHash != "" {
		t.Error("PasswordHash should be empty in response")
	}
}

func TestUserCreateDuplicateUsername(t *testing.T) {
	db := setupTestDB(t)
	reg := testReg(t)
	hasher, _ := auth.NewPasswordHasher(testArgon2Params)
	svc := NewUserService(db, hasher, reg)
	ctx := context.Background()

	svc.Create(ctx, CreateUserInput{Username: "alice", Password: "test"})
	_, err := svc.Create(ctx, CreateUserInput{Username: "alice", Password: "test"})
	if err == nil {
		t.Fatal("expected error for duplicate username")
	}
	ecErr, ok := err.(errcode.Error)
	if !ok || ecErr.Code != reg.ErrUsernameExists.Code {
		t.Errorf("expected ErrUsernameExists, got %v", err)
	}
}

func TestUserGetNoPasswordHash(t *testing.T) {
	db := setupTestDB(t)
	reg := testReg(t)
	hasher, _ := auth.NewPasswordHasher(testArgon2Params)
	svc := NewUserService(db, hasher, reg)
	ctx := context.Background()

	created, _ := svc.Create(ctx, CreateUserInput{Username: "bob", Password: "pwd"})
	user, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if user.PasswordHash != "" {
		t.Error("Get should not return PasswordHash")
	}
}

func TestUserList(t *testing.T) {
	db := setupTestDB(t)
	reg := testReg(t)
	hasher, _ := auth.NewPasswordHasher(testArgon2Params)
	svc := NewUserService(db, hasher, reg)
	ctx := context.Background()

	svc.Create(ctx, CreateUserInput{Username: "u1", Password: "p"})
	svc.Create(ctx, CreateUserInput{Username: "u2", Password: "p"})

	result, err := svc.List(ctx, UserListQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("total: got %d, want 2", result.Total)
	}
}

func TestUserListFilterUsername(t *testing.T) {
	db := setupTestDB(t)
	reg := testReg(t)
	hasher, _ := auth.NewPasswordHasher(testArgon2Params)
	svc := NewUserService(db, hasher, reg)
	ctx := context.Background()

	svc.Create(ctx, CreateUserInput{Username: "alice", Password: "p"})
	svc.Create(ctx, CreateUserInput{Username: "bob", Password: "p"})

	result, _ := svc.List(ctx, UserListQuery{Username: "ali", Page: 1, PageSize: 10})
	if result.Total != 1 {
		t.Errorf("filtered total: got %d, want 1", result.Total)
	}
}

func TestUserDelete(t *testing.T) {
	db := setupTestDB(t)
	reg := testReg(t)
	hasher, _ := auth.NewPasswordHasher(testArgon2Params)
	svc := NewUserService(db, hasher, reg)
	ctx := context.Background()

	user, _ := svc.Create(ctx, CreateUserInput{Username: "del", Password: "p"})
	if err := svc.Delete(ctx, user.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	// 软删除后查不到
	_, err := svc.Get(ctx, user.ID)
	if err == nil {
		t.Error("expected not found after soft delete")
	}
}

func TestUserResetPassword(t *testing.T) {
	db := setupTestDB(t)
	reg := testReg(t)
	hasher, _ := auth.NewPasswordHasher(testArgon2Params)
	svc := NewUserService(db, hasher, reg)
	ctx := context.Background()

	user, _ := svc.Create(ctx, CreateUserInput{Username: "pwd", Password: "old"})
	if err := svc.ResetPassword(ctx, user.ID, "newpwd"); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}
	// 验证新密码有效
	var u SysUser
	db.First(&u, user.ID)
	ok, _ := hasher.Verify("newpwd", u.PasswordHash)
	if !ok {
		t.Error("new password should verify")
	}
}

func TestUserSetStatus(t *testing.T) {
	db := setupTestDB(t)
	reg := testReg(t)
	hasher, _ := auth.NewPasswordHasher(testArgon2Params)
	svc := NewUserService(db, hasher, reg)
	ctx := context.Background()

	user, _ := svc.Create(ctx, CreateUserInput{Username: "st", Password: "p"})
	svc.SetStatus(ctx, user.ID, 1)
	u, _ := svc.Get(ctx, user.ID)
	if u.Status != 1 {
		t.Errorf("status: got %d, want 1", u.Status)
	}
}

// ---------------------------------------------------------------------------
// DeptService 测试
// ---------------------------------------------------------------------------

func TestDeptCreateAndTree(t *testing.T) {
	db := setupTestDB(t)
	reg := testReg(t)
	svc := NewDeptService(db, reg)
	ctx := context.Background()

	root, _ := svc.Create(ctx, CreateDeptInput{Name: "总公司", ParentID: 0})
	child, _ := svc.Create(ctx, CreateDeptInput{Name: "技术部", ParentID: root.ID})

	if child.Ancestors != "0,"+uintStr(root.ID) {
		t.Errorf("ancestors: got %q", child.Ancestors)
	}

	tree, _ := svc.Tree(ctx)
	if len(tree) != 1 {
		t.Fatalf("root count: got %d, want 1", len(tree))
	}
	if len(tree[0].Children) != 1 {
		t.Errorf("children count: got %d, want 1", len(tree[0].Children))
	}
}

func TestDeptDeleteHasChildren(t *testing.T) {
	db := setupTestDB(t)
	reg := testReg(t)
	svc := NewDeptService(db, reg)
	ctx := context.Background()

	root, _ := svc.Create(ctx, CreateDeptInput{Name: "根", ParentID: 0})
	svc.Create(ctx, CreateDeptInput{Name: "子", ParentID: root.ID})

	err := svc.Delete(ctx, root.ID)
	ecErr, ok := err.(errcode.Error)
	if !ok || ecErr.Code != reg.ErrDeptHasChildren.Code {
		t.Errorf("expected ErrDeptHasChildren, got %v", err)
	}
}

func TestDeptDeleteHasUsers(t *testing.T) {
	db := setupTestDB(t)
	reg := testReg(t)
	svc := NewDeptService(db, reg)
	hasher, _ := auth.NewPasswordHasher(testArgon2Params)
	userSvc := NewUserService(db, hasher, reg)
	ctx := context.Background()

	dept, _ := svc.Create(ctx, CreateDeptInput{Name: "部门A"})
	userSvc.Create(ctx, CreateUserInput{Username: "inDept", Password: "p", DeptID: &dept.ID})

	err := svc.Delete(ctx, dept.ID)
	ecErr, ok := err.(errcode.Error)
	if !ok || ecErr.Code != reg.ErrDeptHasUsers.Code {
		t.Errorf("expected ErrDeptHasUsers, got %v", err)
	}
}

func TestDeptMovePreventCycle(t *testing.T) {
	db := setupTestDB(t)
	reg := testReg(t)
	svc := NewDeptService(db, reg)
	ctx := context.Background()

	root, _ := svc.Create(ctx, CreateDeptInput{Name: "根"})
	child, _ := svc.Create(ctx, CreateDeptInput{Name: "子", ParentID: root.ID})

	// 尝试把根节点挂到子节点下 → 防环
	err := svc.Update(ctx, root.ID, UpdateDeptInput{ParentID: &child.ID, Name: "根"})
	if err == nil {
		t.Fatal("expected error for cycle")
	}
}

// ---------------------------------------------------------------------------
// PostService 测试
// ---------------------------------------------------------------------------

func TestPostCRUD(t *testing.T) {
	db := setupTestDB(t)
	reg := testReg(t)
	svc := NewPostService(db, reg)
	ctx := context.Background()

	post, err := svc.Create(ctx, CreatePostInput{Code: "CEO", Name: "总裁"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if post.Code != "CEO" {
		t.Errorf("code: got %q", post.Code)
	}

	// 重复 code
	_, err = svc.Create(ctx, CreatePostInput{Code: "CEO", Name: "另一个"})
	ecErr, ok := err.(errcode.Error)
	if !ok || ecErr.Code != reg.ErrPostCodeExists.Code {
		t.Errorf("expected ErrPostCodeExists, got %v", err)
	}

	// 列表
	result, _ := svc.List(ctx, PostListQuery{Page: 1, PageSize: 10})
	if result.Total != 1 {
		t.Errorf("total: got %d", result.Total)
	}

	// 删除
	svc.Delete(ctx, post.ID)
	result, _ = svc.List(ctx, PostListQuery{Page: 1, PageSize: 10})
	if result.Total != 0 {
		t.Errorf("total after delete: got %d", result.Total)
	}
}

// ---------------------------------------------------------------------------
// GormUserProvider 测试
// ---------------------------------------------------------------------------

func TestGormUserProviderFindByUsername(t *testing.T) {
	db := setupTestDB(t)
	reg := testReg(t)
	hasher, _ := auth.NewPasswordHasher(testArgon2Params)
	userSvc := NewUserService(db, hasher, reg)
	ctx := context.Background()

	userSvc.Create(ctx, CreateUserInput{Username: "provider_test", Password: "secret123"})

	provider := NewGormUserProvider(db)
	authUser, err := provider.FindByUsername(ctx, "provider_test")
	if err != nil {
		t.Fatalf("FindByUsername: %v", err)
	}
	if authUser.Username != "provider_test" {
		t.Errorf("username: got %q", authUser.Username)
	}
	if authUser.PasswordHash == "" {
		t.Error("PasswordHash should be populated for auth")
	}

	// 校验密码有效
	ok, _ := hasher.Verify("secret123", authUser.PasswordHash)
	if !ok {
		t.Error("password should verify")
	}
}

func TestGormUserProviderNotFound(t *testing.T) {
	db := setupTestDB(t)
	provider := NewGormUserProvider(db)

	_, err := provider.FindByUsername(context.Background(), "nonexistent")
	if err != auth.ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// 【阻塞修复】双前缀并存测试 — 验证实例级前缀互不串表
// ---------------------------------------------------------------------------

func TestTwoPrefixesCoexist(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "shared.db")

	db1, _ := gorm.Open(sqlite.Open(tmpFile), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{TablePrefix: "a_", SingularTable: true},
	})
	db2, _ := gorm.Open(sqlite.Open(tmpFile), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{TablePrefix: "b_", SingularTable: true},
	})

	db1.AutoMigrate(&SysUser{}) // 建 a_sys_user
	db2.AutoMigrate(&SysUser{}) // 建 b_sys_user

	reg := testReg(t)
	hasher, _ := auth.NewPasswordHasher(testArgon2Params)
	svc1 := NewUserService(db1, hasher, reg)
	svc2 := NewUserService(db2, hasher, reg)
	ctx := context.Background()

	svc1.Create(ctx, CreateUserInput{Username: "alice_in_a", Password: "p"})
	svc2.Create(ctx, CreateUserInput{Username: "bob_in_b", Password: "p"})

	// a_ 只看到 alice
	r1, _ := svc1.List(ctx, UserListQuery{Page: 1, PageSize: 10})
	if r1.Total != 1 {
		t.Errorf("a_ prefix: expected 1 user, got %d", r1.Total)
	}

	// b_ 只看到 bob
	r2, _ := svc2.List(ctx, UserListQuery{Page: 1, PageSize: 10})
	if r2.Total != 1 {
		t.Errorf("b_ prefix: expected 1 user, got %d", r2.Total)
	}

	// provider 也隔离
	p1 := NewGormUserProvider(db1)
	p2 := NewGormUserProvider(db2)

	_, err := p1.FindByUsername(ctx, "bob_in_b")
	if err != auth.ErrUserNotFound {
		t.Error("a_ provider should not find bob_in_b")
	}
	_, err = p2.FindByUsername(ctx, "alice_in_a")
	if err != auth.ErrUserNotFound {
		t.Error("b_ provider should not find alice_in_a")
	}
}

// ---------------------------------------------------------------------------
// 【确认1】部门防环 — 移动节点到自己的孙子下
// ---------------------------------------------------------------------------

func TestDeptMoveToGrandchild(t *testing.T) {
	db := setupTestDB(t)
	reg := testReg(t)
	svc := NewDeptService(db, reg)
	ctx := context.Background()

	root, _ := svc.Create(ctx, CreateDeptInput{Name: "根"})
	child, _ := svc.Create(ctx, CreateDeptInput{Name: "子", ParentID: root.ID})
	grandchild, _ := svc.Create(ctx, CreateDeptInput{Name: "孙", ParentID: child.ID})

	// 尝试把根节点挂到孙节点下 → 防环必须拒绝
	err := svc.Update(ctx, root.ID, UpdateDeptInput{ParentID: &grandchild.ID, Name: "根"})
	if err == nil {
		t.Fatal("should reject moving root under its grandchild (cycle)")
	}

	// 尝试把子节点挂到孙节点下 → 也是环
	err = svc.Update(ctx, child.ID, UpdateDeptInput{ParentID: &grandchild.ID, Name: "子"})
	if err == nil {
		t.Fatal("should reject moving child under its own child (cycle)")
	}

	// 移动到自身 → 拒绝
	err = svc.Update(ctx, child.ID, UpdateDeptInput{ParentID: &child.ID, Name: "子"})
	if err == nil {
		t.Fatal("should reject moving node under itself")
	}
}

// ---------------------------------------------------------------------------
// 【确认2】password_hash JSON 序列化断言
// ---------------------------------------------------------------------------

func TestPasswordHashNotInJSON_Get(t *testing.T) {
	db := setupTestDB(t)
	reg := testReg(t)
	hasher, _ := auth.NewPasswordHasher(testArgon2Params)
	svc := NewUserService(db, hasher, reg)
	ctx := context.Background()

	created, _ := svc.Create(ctx, CreateUserInput{Username: "jsontest", Password: "secret"})

	// Get 返回的 user 序列化后不含 password_hash
	user, _ := svc.Get(ctx, created.ID)
	data, _ := json.Marshal(user)
	var m map[string]any
	json.Unmarshal(data, &m)
	if _, exists := m["password_hash"]; exists {
		t.Error("Get response JSON should NOT contain password_hash")
	}
}

func TestPasswordHashNotInJSON_List(t *testing.T) {
	db := setupTestDB(t)
	reg := testReg(t)
	hasher, _ := auth.NewPasswordHasher(testArgon2Params)
	svc := NewUserService(db, hasher, reg)
	ctx := context.Background()

	svc.Create(ctx, CreateUserInput{Username: "listjson", Password: "secret"})

	result, _ := svc.List(ctx, UserListQuery{Page: 1, PageSize: 10})
	data, _ := json.Marshal(result.List)
	jsonStr := string(data)
	if strings.Contains(jsonStr, "password_hash") {
		t.Error("List response JSON should NOT contain password_hash")
	}
	if strings.Contains(jsonStr, "$argon2id$") {
		t.Error("List response should NOT contain Argon2id hash values")
	}
}

func TestPasswordHashNotInJSON_Create(t *testing.T) {
	db := setupTestDB(t)
	reg := testReg(t)
	hasher, _ := auth.NewPasswordHasher(testArgon2Params)
	svc := NewUserService(db, hasher, reg)
	ctx := context.Background()

	user, _ := svc.Create(ctx, CreateUserInput{Username: "createjson", Password: "secret"})
	data, _ := json.Marshal(user)
	var m map[string]any
	json.Unmarshal(data, &m)
	if _, exists := m["password_hash"]; exists {
		t.Error("Create response JSON should NOT contain password_hash")
	}
}

// ---------------------------------------------------------------------------
// T-009b：SetStatus 三态 + 菜单父节点专属错误码
// ---------------------------------------------------------------------------

// codeOf 从已编码错误提取 code（service 层返回的 errcode.Error 实现 GetCode()）。
func codeOf(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		t.Fatal("expected coded error, got nil")
	}
	c, ok := err.(interface{ GetCode() int })
	if !ok {
		t.Fatalf("not a coded error: %v", err)
	}
	return c.GetCode()
}

// TestUserSetStatusNotFound 关键负例：不存在的 id 仍须返 ErrUserNotFound——
// T-009b 修复区分"无变更"vs"不存在"，绝不放宽存在性校验（否则安全/正确性倒退）。
func TestUserSetStatusNotFound(t *testing.T) {
	db := setupTestDB(t)
	reg := testReg(t)
	hasher, _ := auth.NewPasswordHasher(testArgon2Params)
	svc := NewUserService(db, hasher, reg)
	ctx := context.Background()

	err := svc.SetStatus(ctx, 999999, 1)
	if got := codeOf(t, err); got != reg.ErrUserNotFound.Code {
		t.Errorf("SetStatus on missing id: code got %d, want ErrUserNotFound(%d)", got, reg.ErrUserNotFound.Code)
	}
}

// TestUserSetStatusRealChange 真变更 → 成功且落值。
func TestUserSetStatusRealChange(t *testing.T) {
	db := setupTestDB(t)
	reg := testReg(t)
	hasher, _ := auth.NewPasswordHasher(testArgon2Params)
	svc := NewUserService(db, hasher, reg)
	ctx := context.Background()

	user, _ := svc.Create(ctx, CreateUserInput{Username: "stchg", Password: "p"})
	if err := svc.SetStatus(ctx, user.ID, 1); err != nil {
		t.Fatalf("SetStatus real change: %v", err)
	}
	u, _ := svc.Get(ctx, user.ID)
	if u.Status != 1 {
		t.Errorf("status: got %d, want 1", u.Status)
	}
}

// TestUserSetStatusSameValueIdempotent 同值（记录存在但值未变）→ 须返成功，不误返 404。
// 注：SQLite 同值 UPDATE 返 RowsAffected==1，故本用例在 SQLite 上即便旧实现也过；
// 真正区分（MySQL 同值返 RowsAffected==0、旧实现误返 404）的命门由
// org_integration_test.go:TestOrgSetStatusSameValue_MySQL（-tags=integration）坐实。
func TestUserSetStatusSameValueIdempotent(t *testing.T) {
	db := setupTestDB(t)
	reg := testReg(t)
	hasher, _ := auth.NewPasswordHasher(testArgon2Params)
	svc := NewUserService(db, hasher, reg)
	ctx := context.Background()

	user, _ := svc.Create(ctx, CreateUserInput{Username: "stsame", Password: "p"})
	if err := svc.SetStatus(ctx, user.ID, 1); err != nil {
		t.Fatalf("SetStatus to 1: %v", err)
	}
	// 再设相同值 → 应成功（幂等无变更）
	if err := svc.SetStatus(ctx, user.ID, 1); err != nil {
		t.Errorf("SetStatus same value(1→1): got %v, want nil（同值不应误返 404）", err)
	}
	u, _ := svc.Get(ctx, user.ID)
	if u.Status != 1 {
		t.Errorf("status after same-value set: got %d, want 1", u.Status)
	}
}

// setupMenuTestDB 在通用测试库基础上补 SysMenu 表。
func setupMenuTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupTestDB(t)
	if err := db.AutoMigrate(&SysMenu{}); err != nil {
		t.Fatalf("auto migrate menu: %v", err)
	}
	return db
}

// TestMenuParentUsesMenuErrorCode 菜单父节点错误改用专属 ErrInvalidParentMenu，
// 不再复用 ErrInvalidParentDept（"父部门"文案出现在菜单模块=语义错）。
func TestMenuParentUsesMenuErrorCode(t *testing.T) {
	db := setupMenuTestDB(t)
	reg := testReg(t)
	svc := NewMenuService(db, reg)
	ctx := context.Background()

	root, err := svc.Create(ctx, CreateMenuInput{MenuType: MenuTypeDir, Name: "系统管理"})
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	child, err := svc.Create(ctx, CreateMenuInput{ParentID: root.ID, MenuType: MenuTypePage, Name: "用户管理"})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	wantMenu := reg.ErrInvalidParentMenu.Code
	notDept := reg.ErrInvalidParentDept.Code
	if wantMenu == notDept {
		t.Fatalf("菜单专属码与部门码相同(%d)——未真正分离", wantMenu)
	}

	check := func(label string, err error) {
		got := codeOf(t, err)
		if got == notDept {
			t.Errorf("%s: 仍返 ErrInvalidParentDept(%d)，未改用菜单专属码", label, notDept)
		}
		if got != wantMenu {
			t.Errorf("%s: code got %d, want ErrInvalidParentMenu(%d)", label, got, wantMenu)
		}
	}

	// ① Create 父不存在
	_, err = svc.Create(ctx, CreateMenuInput{ParentID: 999999, MenuType: MenuTypePage, Name: "孤儿"})
	check("create-missing-parent", err)

	// ② Update 移动到不存在父
	missing := uint64(999999)
	err = svc.Update(ctx, child.ID, UpdateMenuInput{ParentID: &missing, MenuType: MenuTypePage, Name: "用户管理"})
	check("update-missing-parent", err)

	// ③ Update 移动成环（root 移到其子孙 child 之下）
	err = svc.Update(ctx, root.ID, UpdateMenuInput{ParentID: &child.ID, MenuType: MenuTypeDir, Name: "系统管理"})
	check("update-cycle", err)
}

// ---------------------------------------------------------------------------
// 辅助
// ---------------------------------------------------------------------------

func uintStr(id uint64) string {
	return strconv.FormatUint(id, 10)
}
