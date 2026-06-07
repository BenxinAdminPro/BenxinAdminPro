// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   数据权限模型 + 解析器 — 多角色合并取最宽 + 超管短路
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 23:35:00
// +----------------------------------------------------------------------

package rbac

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// ScopeType 数据权限类型。
type ScopeType int8

const (
	ScopeAll  ScopeType = 1 // 全部数据
	ScopeSelf ScopeType = 2 // 仅本人
	ScopeDept ScopeType = 3 // 本部门
	// ScopeDeptAndChild ScopeType = 4 // 预留：本部门及子部门（本片不实现）
)

// ValidScopeType 校验 data_scope 值合法性（1/2/3）。
func ValidScopeType(v int8) bool {
	return v >= 1 && v <= 3
}

// DataScope 表示经过多角色合并后的最终数据权限范围。
// 由 ScopeResolver 解析，由 ApplyScope 应用。
type DataScope struct {
	Type    ScopeType // 合并后的有效类型（取最宽）
	All     bool      // Type==ScopeAll 时 true，调用方不加任何限制
	DeptIDs []uint64  // ScopeDept 时的部门集合（本片为单部门，预留多部门/子部门扩展）
	SelfID  uint64    // 当前用户内部 ID（ScopeSelf 时用）
}

// ScopeResolver 数据权限解析器接口。
type ScopeResolver interface {
	// Resolve 输入当前用户 ID，输出合并后的有效 DataScope。
	// 用户的 dept_id 和角色集合由解析器从 DB 查取（不接受客户端传入）。
	Resolve(ctx context.Context, userID uint64) (*DataScope, error)
}

// ---------------------------------------------------------------------------
// GormScopeResolver 实现
// ---------------------------------------------------------------------------

// GormScopeResolver 从 DB 解析用户的数据权限。
type GormScopeResolver struct {
	db       *gorm.DB
	superSet map[string]bool // 超管角色 code 集合
}

// NewGormScopeResolver 创建 GORM 数据权限解析器。
// superRoles 与 AuthzConfig.SuperAdminRoles 同源，配置注入，禁硬编码。
func NewGormScopeResolver(db *gorm.DB, superRoles []string) *GormScopeResolver {
	ss := make(map[string]bool, len(superRoles))
	for _, r := range superRoles {
		ss[r] = true
	}
	return &GormScopeResolver{db: db, superSet: ss}
}

// Resolve 解析用户有效数据权限。
// 1. 查用户角色（code + data_scope）
// 2. 超管短路 → All
// 3. 多角色合并取最宽（All > Dept > Self）
// 4. 无角色 → 默认 ScopeSelf（fail-closed）
func (r *GormScopeResolver) Resolve(ctx context.Context, userID uint64) (*DataScope, error) {
	// 查用户角色
	type roleRow struct {
		Code      string
		DataScope int8
	}

	userRoleTable := resolveTable(r.db, &SysUserRole{})
	roleTable := resolveTable(r.db, &SysRole{})

	var roles []roleRow
	err := r.db.WithContext(ctx).
		Table(userRoleTable+" ur").
		Select(roleTable+".code, "+roleTable+".data_scope").
		Joins("JOIN "+roleTable+" ON "+roleTable+".id = ur.role_id AND "+roleTable+".deleted_at IS NULL").
		Where("ur.user_id = ?", userID).
		Scan(&roles).Error
	if err != nil {
		return nil, fmt.Errorf("rbac: resolve scope: %w", err)
	}

	// 无角色 → fail-closed（默认本人）
	if len(roles) == 0 {
		return &DataScope{Type: ScopeSelf, SelfID: userID}, nil
	}

	// 超管短路
	for _, role := range roles {
		if r.superSet[role.Code] {
			return &DataScope{Type: ScopeAll, All: true, SelfID: userID}, nil
		}
	}

	// 多角色合并
	var (
		hasAll  bool
		hasDept bool
		hasSelf bool
	)
	for _, role := range roles {
		switch ScopeType(role.DataScope) {
		case ScopeAll:
			hasAll = true
		case ScopeDept:
			hasDept = true
		case ScopeSelf:
			hasSelf = true
		}
	}

	// All 短路
	if hasAll {
		return &DataScope{Type: ScopeAll, All: true, SelfID: userID}, nil
	}

	ds := &DataScope{SelfID: userID}

	// Dept 需要查 user.dept_id
	if hasDept {
		var deptID *uint64
		r.db.WithContext(ctx).Model(&SysUser{}).Select("dept_id").Where("id = ?", userID).Scan(&deptID)
		if deptID != nil && *deptID > 0 {
			ds.DeptIDs = []uint64{*deptID}
		}
		// deptID 为 nil → DeptIDs 空，ApplyScope 会安全失败
		ds.Type = ScopeDept
	} else if hasSelf {
		ds.Type = ScopeSelf
	} else {
		// 未知 scope 值 → fail-closed
		ds.Type = ScopeSelf
	}

	return ds, nil
}
