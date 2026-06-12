// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   demo 种子数据 — 超管/菜单/部门/普通用户（幂等 upsert）
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 04:10:00
// | @updated   2026-06-08 12:00:00  种子密码零明文：全部来自配置，缺失即 fail-fast
// | @updated   2026-06-09 16:10:00  T-007c：补齐既有 C 页缺失的 F 权限码（对齐后端路由 RequirePerm）
// | @updated   2026-06-10 12:02:21  T-007e：补操作/登录日志 C 菜单 + F 码（sys:operlog:* / sys:loginlog:*，对齐 handler.go RequirePerm）
// | @updated   2026-06-10 17:19:28  T-007f：补文件管理 C 菜单 + F 码（sys:file:*，对齐 handler_file.go RequirePerm）
// +----------------------------------------------------------------------

package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/benxin_dev/benxinadminpro-server/auth"
	"github.com/benxin_dev/benxinadminpro-server/rbac"
	"gorm.io/gorm"
)

// seedPassword 取指定用户的种子密码；零明文默认值，缺失即返回 error（startup fail-fast）。
func seedPassword(cfg demoConfig, username string) (string, error) {
	pwd := cfg.SeedPasswords[username]
	if pwd == "" {
		return "", fmt.Errorf("seed password for user %q is missing — set seed.%s_password in config.local.yaml "+
			"(or env DEMO_SEED_%s_PASSWORD); no plaintext default is provided", username, username, strings.ToUpper(username))
	}
	return pwd, nil
}

func seed(db *gorm.DB, hasher auth.PasswordHasher, ps rbac.PolicySync, cfg demoConfig) error {
	ctx := context.Background()

	// 全部种子密码来自配置，任一缺失即 fail-fast（零明文默认值）
	adminPwd, err := seedPassword(cfg, "admin")
	if err != nil {
		return err
	}
	editorPwd, err := seedPassword(cfg, "editor")
	if err != nil {
		return err
	}
	deptMgrPwd, err := seedPassword(cfg, "dept_mgr")
	if err != nil {
		return err
	}
	bizPwd, err := seedPassword(cfg, "biz_user")
	if err != nil {
		return err
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

	// 普通用户（密码均来自配置，零明文）
	editorHash, _ := hasher.Hash(editorPwd)
	editorDeptID := techDept.ID
	editorUser := rbac.SysUser{Username: "editor", PasswordHash: editorHash, Nickname: "编辑小明", DeptID: &editorDeptID}
	db.Where("username = ?", "editor").FirstOrCreate(&editorUser)

	mgrHash, _ := hasher.Hash(deptMgrPwd)
	mgrDeptID := techDept.ID
	mgrUser := rbac.SysUser{Username: "dept_mgr", PasswordHash: mgrHash, Nickname: "技术经理", DeptID: &mgrDeptID}
	db.Where("username = ?", "dept_mgr").FirstOrCreate(&mgrUser)

	bizHash, _ := hasher.Hash(bizPwd)
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
	seedMenu(db, userPage.ID, "F", "用户详情", "sys:user:get", "", "", "", 7)
	seedMenu(db, userPage.ID, "F", "用户状态", "sys:user:status", "", "", "", 8)

	deptPage := seedMenu(db, sysDir.ID, "C", "部门管理", "", "/sys/dept", "sys/dept/index", "tree", 2)
	seedMenu(db, deptPage.ID, "F", "部门树", "sys:dept:tree", "", "", "", 1)
	seedMenu(db, deptPage.ID, "F", "部门新增", "sys:dept:create", "", "", "", 2)
	seedMenu(db, deptPage.ID, "F", "部门编辑", "sys:dept:update", "", "", "", 3)
	seedMenu(db, deptPage.ID, "F", "部门删除", "sys:dept:delete", "", "", "", 4)

	rolePage := seedMenu(db, sysDir.ID, "C", "角色管理", "", "/sys/role", "sys/role/index", "peoples", 3)
	seedMenu(db, rolePage.ID, "F", "角色列表", "sys:role:list", "", "", "", 1)
	seedMenu(db, rolePage.ID, "F", "角色新增", "sys:role:create", "", "", "", 2)
	seedMenu(db, rolePage.ID, "F", "分配菜单", "sys:role:assign", "", "", "", 3)
	seedMenu(db, rolePage.ID, "F", "角色编辑", "sys:role:update", "", "", "", 4)
	seedMenu(db, rolePage.ID, "F", "角色删除", "sys:role:delete", "", "", "", 5)

	menuPage := seedMenu(db, sysDir.ID, "C", "菜单管理", "", "/sys/menu", "sys/menu/index", "tree-table", 4)
	seedMenu(db, menuPage.ID, "F", "菜单列表", "sys:menu:list", "", "", "", 1)
	seedMenu(db, menuPage.ID, "F", "菜单新增", "sys:menu:create", "", "", "", 2)
	seedMenu(db, menuPage.ID, "F", "菜单编辑", "sys:menu:update", "", "", "", 3)
	seedMenu(db, menuPage.ID, "F", "菜单删除", "sys:menu:delete", "", "", "", 4)

	dictPage := seedMenu(db, sysDir.ID, "C", "字典管理", "", "/sys/dict", "sys/dict/index", "dict", 5)
	seedMenu(db, dictPage.ID, "F", "字典查看", "sys:dict:list", "", "", "", 1)
	seedMenu(db, dictPage.ID, "F", "字典新增", "sys:dict:create", "", "", "", 2)
	seedMenu(db, dictPage.ID, "F", "字典编辑", "sys:dict:update", "", "", "", 3)
	seedMenu(db, dictPage.ID, "F", "字典删除", "sys:dict:delete", "", "", "", 4)

	configPage := seedMenu(db, sysDir.ID, "C", "参数管理", "", "/sys/config", "sys/config/index", "edit", 6)
	seedMenu(db, configPage.ID, "F", "参数查看", "sys:config:list", "", "", "", 1)
	seedMenu(db, configPage.ID, "F", "参数新增", "sys:config:create", "", "", "", 2)
	seedMenu(db, configPage.ID, "F", "参数编辑", "sys:config:update", "", "", "", 3)
	seedMenu(db, configPage.ID, "F", "参数删除", "sys:config:delete", "", "", "", 4)
	seedMenu(db, configPage.ID, "F", "敏感值查看", "sys:secret:view", "", "", "", 5)

	// T-007e：日志页（码与 system/handler.go RegisterRoutes 的 RequirePerm 逐字一致）
	operlogPage := seedMenu(db, sysDir.ID, "C", "操作日志", "", "/sys/operlog", "sys/operlog/index", "document", 7)
	seedMenu(db, operlogPage.ID, "F", "操作日志查看", "sys:operlog:list", "", "", "", 1)
	seedMenu(db, operlogPage.ID, "F", "操作日志清理", "sys:operlog:clean", "", "", "", 2)

	loginlogPage := seedMenu(db, sysDir.ID, "C", "登录日志", "", "/sys/loginlog", "sys/loginlog/index", "monitor", 8)
	seedMenu(db, loginlogPage.ID, "F", "登录日志查看", "sys:loginlog:list", "", "", "", 1)
	seedMenu(db, loginlogPage.ID, "F", "登录日志清理", "sys:loginlog:clean", "", "", "", 2)

	// T-007f：文件管理页（码与 system/handler_file.go RegisterRoutes 的 RequirePerm 逐字一致）
	filePage := seedMenu(db, sysDir.ID, "C", "文件管理", "", "/sys/file", "sys/file/index", "folder", 9)
	seedMenu(db, filePage.ID, "F", "文件列表", "sys:file:list", "", "", "", 1)
	seedMenu(db, filePage.ID, "F", "文件上传", "sys:file:upload", "", "", "", 2)
	seedMenu(db, filePage.ID, "F", "文件下载", "sys:file:download", "", "", "", 3)
	seedMenu(db, filePage.ID, "F", "文件删除", "sys:file:delete", "", "", "", 4)

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
	return nil
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
