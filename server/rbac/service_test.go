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

	"github.com/benxin_dev/benxinadminpro-server/auth"
	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var testArgon2Params = auth.Argon2idParams{
	MemoryKiB: 1024, Iterations: 1, Parallelism: 1, SaltLen: 16, KeyLen: 32,
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	SetTablePrefix("") // 测试用空前缀
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
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
// 辅助
// ---------------------------------------------------------------------------

func uintStr(id uint64) string {
	return strconv.FormatUint(id, 10)
}
