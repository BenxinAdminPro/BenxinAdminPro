// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   demo 种子数据 — 超管/菜单/部门/普通用户（幂等 upsert）
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 04:10:00
// +----------------------------------------------------------------------

package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/benxin_dev/benxinadminpro-server/auth"
	"github.com/benxin_dev/benxinadminpro-server/rbac"
	"gorm.io/gorm"
)

func seed(db *gorm.DB, hasher auth.PasswordHasher, ps rbac.PolicySync, cfg demoConfig) {
	ctx := context.Background()

	// 超管密码（来自配置，不硬编码）
	adminPwd := cfg.SeedAdminPwd
	if adminPwd == "" {
		adminPwd = "admin123" // 仅 demo 兜底；生产必须配置注入
		slog.Warn("seed: using default admin password, configure seed.admin_password for production")
	}

	// ---- 1. 超管角色 ----
	superRole := rbac.SysRole{Code: "super_admin", Name: "超级管理员", DataScope: 1} // All
	db.Where("code = ?", superRole.Code).FirstOrCreate(&superRole)

	// 普通角色
	editorRole := rbac.SysRole{Code: "editor", Name: "编辑员", DataScope: 2} // Self
	db.Where("code = ?", editorRole.Code).FirstOrCreate(&editorRole)

	deptMgrRole := rbac.SysRole{Code: "dept_mgr", Name: "部门经理", DataScope: 3} // Dept
	db.Where("code = ?", deptMgrRole.Code).FirstOrCreate(&deptMgrRole)

	// ---- 2. 部门 ----
	rootDept := rbac.SysDept{Name: "总公司", Ancestors: "0"}
	db.Where("name = ? AND parent_id = 0", rootDept.Name).FirstOrCreate(&rootDept)

	techDept := rbac.SysDept{Name: "技术部", ParentID: rootDept.ID, Ancestors: "0," + itoa(rootDept.ID)}
	db.Where("name = ? AND parent_id = ?", techDept.Name, rootDept.ID).FirstOrCreate(&techDept)

	bizDept := rbac.SysDept{Name: "业务部", ParentID: rootDept.ID, Ancestors: "0," + itoa(rootDept.ID)}
	db.Where("name = ? AND parent_id = ?", bizDept.Name, rootDept.ID).FirstOrCreate(&bizDept)

	// ---- 3. 超管用户 ----
	adminHash, _ := hasher.Hash(adminPwd)
	adminUser := rbac.SysUser{Username: "admin", PasswordHash: adminHash, Nickname: "管理员", DeptID: &rootDept.ID}
	db.Where("username = ?", "admin").FirstOrCreate(&adminUser)
	// 更新密码（如果已存在，密码可能已变）
	db.Model(&rbac.SysUser{}).Where("username = ?", "admin").Update("password_hash", adminHash)

	// 普通用户
	editorHash, _ := hasher.Hash("editor123")
	editorDeptID := techDept.ID
	editorUser := rbac.SysUser{Username: "editor", PasswordHash: editorHash, Nickname: "编辑小明", DeptID: &editorDeptID}
	db.Where("username = ?", "editor").FirstOrCreate(&editorUser)

	mgrHash, _ := hasher.Hash("manager123")
	mgrDeptID := techDept.ID
	mgrUser := rbac.SysUser{Username: "dept_mgr", PasswordHash: mgrHash, Nickname: "技术经理", DeptID: &mgrDeptID}
	db.Where("username = ?", "dept_mgr").FirstOrCreate(&mgrUser)

	bizHash, _ := hasher.Hash("bizuser123")
	bizDeptID := bizDept.ID
	bizUser := rbac.SysUser{Username: "biz_user", PasswordHash: bizHash, Nickname: "业务员", DeptID: &bizDeptID}
	db.Where("username = ?", "biz_user").FirstOrCreate(&bizUser)

	// ---- 4. 用户-角色 ----
	upsertUserRole(db, adminUser.ID, superRole.ID)
	upsertUserRole(db, editorUser.ID, editorRole.ID)
	upsertUserRole(db, mgrUser.ID, deptMgrRole.ID)

	// ---- 5. 菜单/权限点 ----
	sysDir := seedMenu(db, 0, "M", "系统管理", "", "/sys", "", "setting", 1)
	userPage := seedMenu(db, sysDir.ID, "C", "用户管理", "", "/sys/user", "sys/user/index", "user", 1)
	seedMenu(db, userPage.ID, "F", "用户列表", "sys:user:list", "", "", "", 1)
	seedMenu(db, userPage.ID, "F", "用户新增", "sys:user:create", "", "", "", 2)
	seedMenu(db, userPage.ID, "F", "用户编辑", "sys:user:update", "", "", "", 3)
	seedMenu(db, userPage.ID, "F", "用户删除", "sys:user:delete", "", "", "", 4)
	seedMenu(db, userPage.ID, "F", "重置密码", "sys:user:password", "", "", "", 5)
	seedMenu(db, userPage.ID, "F", "分配角色", "sys:user:assign", "", "", "", 6)

	deptPage := seedMenu(db, sysDir.ID, "C", "部门管理", "", "/sys/dept", "sys/dept/index", "tree", 2)
	seedMenu(db, deptPage.ID, "F", "部门树", "sys:dept:tree", "", "", "", 1)
	seedMenu(db, deptPage.ID, "F", "部门新增", "sys:dept:create", "", "", "", 2)

	rolePage := seedMenu(db, sysDir.ID, "C", "角色管理", "", "/sys/role", "sys/role/index", "peoples", 3)
	seedMenu(db, rolePage.ID, "F", "角色列表", "sys:role:list", "", "", "", 1)
	seedMenu(db, rolePage.ID, "F", "角色新增", "sys:role:create", "", "", "", 2)
	seedMenu(db, rolePage.ID, "F", "分配菜单", "sys:role:assign", "", "", "", 3)

	menuPage := seedMenu(db, sysDir.ID, "C", "菜单管理", "", "/sys/menu", "sys/menu/index", "tree-table", 4)
	seedMenu(db, menuPage.ID, "F", "菜单列表", "sys:menu:list", "", "", "", 1)
	seedMenu(db, menuPage.ID, "F", "菜单新增", "sys:menu:create", "", "", "", 2)

	dictPage := seedMenu(db, sysDir.ID, "C", "字典管理", "", "/sys/dict", "sys/dict/index", "dict", 5)
	seedMenu(db, dictPage.ID, "F", "字典查看", "sys:dict:list", "", "", "", 1)

	configPage := seedMenu(db, sysDir.ID, "C", "参数管理", "", "/sys/config", "sys/config/index", "edit", 6)
	seedMenu(db, configPage.ID, "F", "参数查看", "sys:config:list", "", "", "", 1)

	// ---- 6. 给编辑员角色分配部分权限 ----
	var allMenuIDs []uint64
	db.Model(&rbac.SysMenu{}).Pluck("id", &allMenuIDs)

	// 超管拥有全部菜单
	for _, mid := range allMenuIDs {
		upsertRoleMenu(db, superRole.ID, mid)
	}

	// 编辑员只有用户列表查看
	var userListMenuID uint64
	db.Model(&rbac.SysMenu{}).Where("perm_code = ?", "sys:user:list").Pluck("id", &userListMenuID)
	if userListMenuID > 0 {
		upsertRoleMenu(db, editorRole.ID, userListMenuID)
	}

	// 部门经理拥有全部
	for _, mid := range allMenuIDs {
		upsertRoleMenu(db, deptMgrRole.ID, mid)
	}

	// ---- 7. 联动 Casbin policy ----
	if ps != nil {
		ps.ReloadAll(ctx)
	}

	slog.Info("seed data applied")
}

func seedMenu(db *gorm.DB, parentID uint64, menuType, name, permCode, path, component, icon string, sort int) *rbac.SysMenu {
	ancestors := "0"
	if parentID > 0 {
		var parent rbac.SysMenu
		db.First(&parent, parentID)
		ancestors = parent.Ancestors + "," + itoa(parentID)
	}
	menu := rbac.SysMenu{
		ParentID: parentID, Ancestors: ancestors, MenuType: menuType,
		Name: name, PermCode: permCode, Path: path, Component: component,
		Icon: icon, Sort: sort, Visible: 1,
	}
	if permCode != "" {
		db.Where("perm_code = ?", permCode).FirstOrCreate(&menu)
	} else {
		db.Where("name = ? AND parent_id = ? AND menu_type = ?", name, parentID, menuType).FirstOrCreate(&menu)
	}
	return &menu
}

func upsertUserRole(db *gorm.DB, userID, roleID uint64) {
	ur := rbac.SysUserRole{UserID: userID, RoleID: roleID}
	db.Where("user_id = ? AND role_id = ?", userID, roleID).FirstOrCreate(&ur)
}

func upsertRoleMenu(db *gorm.DB, roleID, menuID uint64) {
	rm := rbac.SysRoleMenu{RoleID: roleID, MenuID: menuID}
	db.Where("role_id = ? AND menu_id = ?", roleID, menuID).FirstOrCreate(&rm)
}

func itoa(id uint64) string {
	return fmt.Sprintf("%d", id)
}
