// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   Casbin policy 联动 — 角色权限/用户角色双写 + 全量重建
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 21:14:00
// +----------------------------------------------------------------------

package rbac

import (
	"context"
	"fmt"

	"github.com/casbin/casbin/v2"
	"gorm.io/gorm"
)

// PolicyAct 是 Casbin policy 中的固定动作。
const PolicyAct = "access"

// PolicySync 定义 Casbin policy 联动接口。
type PolicySync interface {
	// SyncRolePerms 重写指定角色的 p 规则（先删旧 p，再批量加新 p）。
	SyncRolePerms(ctx context.Context, roleCode string, permCodes []string) error
	// SyncUserRoles 重写指定用户的 g 规则。
	SyncUserRoles(ctx context.Context, userSub string, roleCodes []string) error
	// ReloadAll 全量重建：清空后从 DB 重灌全部 p/g（兜底/启动用）。
	ReloadAll(ctx context.Context) error
}

// casbinPolicySync 实现 PolicySync。
type casbinPolicySync struct {
	enforcer *casbin.Enforcer
	db       *gorm.DB
}

// NewPolicySync 创建 Casbin 联动服务。
func NewPolicySync(enforcer *casbin.Enforcer, db *gorm.DB) PolicySync {
	return &casbinPolicySync{enforcer: enforcer, db: db}
}

// SyncRolePerms 增量双写角色权限到 Casbin p 规则。
// 一致性策略：先从 enforcer 删除该角色旧 p 规则，再添加新规则，最后 SavePolicy。
// 若 SavePolicy 失败，内存中的 policy 仍是新的（不一致），可通过 ReloadAll 兜底。
func (s *casbinPolicySync) SyncRolePerms(_ context.Context, roleCode string, permCodes []string) error {
	// 删除该角色旧 p 规则
	s.enforcer.RemoveFilteredPolicy(0, roleCode)

	// 添加新 p 规则
	var rules [][]string
	for _, code := range permCodes {
		rules = append(rules, []string{roleCode, code, PolicyAct})
	}
	if len(rules) > 0 {
		if _, err := s.enforcer.AddPolicies(rules); err != nil {
			return fmt.Errorf("rbac: add policies: %w", err)
		}
	}

	return s.enforcer.SavePolicy()
}

// SyncUserRoles 增量双写用户角色到 Casbin g 规则。
func (s *casbinPolicySync) SyncUserRoles(_ context.Context, userSub string, roleCodes []string) error {
	// 删除该用户旧 g 规则
	s.enforcer.RemoveFilteredGroupingPolicy(0, userSub)

	// 添加新 g 规则
	var rules [][]string
	for _, code := range roleCodes {
		rules = append(rules, []string{userSub, code})
	}
	if len(rules) > 0 {
		if _, err := s.enforcer.AddGroupingPolicies(rules); err != nil {
			return fmt.Errorf("rbac: add grouping policies: %w", err)
		}
	}

	return s.enforcer.SavePolicy()
}

// ReloadAll 全量重建：从 DB 查所有角色权限和用户角色，清空 enforcer 后重灌。
func (s *casbinPolicySync) ReloadAll(_ context.Context) error {
	// 清空 enforcer 内存中的所有 policy
	s.enforcer.ClearPolicy()

	// 从 DB 重建 p 规则：role → role_menu → menu.perm_code
	type rolePermRow struct {
		RoleCode string
		PermCode string
	}
	var rpRows []rolePermRow

	roleTable := resolveTable(s.db, &SysRole{})
	roleMenuTable := resolveTable(s.db, &SysRoleMenu{})
	menuTable := resolveTable(s.db, &SysMenu{})
	userRoleTable := resolveTable(s.db, &SysUserRole{})

	s.db.Table(roleMenuTable+" rm").
		Select(roleTable+".code as role_code, "+menuTable+".perm_code as perm_code").
		Joins("JOIN "+roleTable+" ON "+roleTable+".id = rm.role_id AND "+roleTable+".deleted_at IS NULL").
		Joins("JOIN "+menuTable+" ON "+menuTable+".id = rm.menu_id AND "+menuTable+".deleted_at IS NULL").
		Where(menuTable+".perm_code != ''").
		Scan(&rpRows)

	var pRules [][]string
	for _, row := range rpRows {
		pRules = append(pRules, []string{row.RoleCode, row.PermCode, PolicyAct})
	}
	if len(pRules) > 0 {
		s.enforcer.AddPolicies(pRules)
	}

	// 从 DB 重建 g 规则：user_role → user.id + role.code
	type userRoleRow struct {
		UserID   uint64
		RoleCode string
	}
	var urRows []userRoleRow

	s.db.Table(userRoleTable+" ur").
		Select("ur.user_id, "+roleTable+".code as role_code").
		Joins("JOIN "+roleTable+" ON "+roleTable+".id = ur.role_id AND "+roleTable+".deleted_at IS NULL").
		Scan(&urRows)

	var gRules [][]string
	for _, row := range urRows {
		gRules = append(gRules, []string{fmt.Sprintf("%d", row.UserID), row.RoleCode})
	}
	if len(gRules) > 0 {
		s.enforcer.AddGroupingPolicies(gRules)
	}

	return s.enforcer.SavePolicy()
}

func resolveTable(db *gorm.DB, model any) string {
	stmt := &gorm.Statement{DB: db}
	stmt.Parse(model)
	return stmt.Schema.Table
}
