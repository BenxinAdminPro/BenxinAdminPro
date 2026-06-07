// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   ApplyScope GORM 辅助 — 按 DataScope 生成查询条件（安全失败）
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 23:37:00
// +----------------------------------------------------------------------

package rbac

import "gorm.io/gorm"

// ScopeFields 由业务调用方传入，指定当前查询表的部门列名与用户列名。
// 底座不预设任何业务表名/列名，保持业务中立。
type ScopeFields struct {
	DeptColumn string // 部门 ID 列名，如 "dept_id"
	UserColumn string // 归属用户 ID 列名，如 "id" 或 "created_by"
}

// ApplyScope 返回 GORM scope 函数，按 DataScope 注入 WHERE 条件。
//
// 安全失败原则（头号红线）：
//   - nil scope → WHERE 1=0（查不到）
//   - All → 不加条件
//   - Self + UserColumn 空或 SelfID=0 → WHERE 1=0
//   - Dept + DeptColumn 空或 DeptIDs 空 → WHERE 1=0
//   - 未知 ScopeType → WHERE 1=0
//
// 任何歧义一律收紧，严禁默认放宽为全部。
func ApplyScope(scope *DataScope, fields ScopeFields) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if scope == nil {
			return db.Where("1 = 0")
		}
		if scope.All {
			return db
		}

		switch scope.Type {
		case ScopeAll:
			return db

		case ScopeSelf:
			if fields.UserColumn == "" || scope.SelfID == 0 {
				return db.Where("1 = 0")
			}
			return db.Where(fields.UserColumn+" = ?", scope.SelfID)

		case ScopeDept:
			if fields.DeptColumn == "" || len(scope.DeptIDs) == 0 {
				return db.Where("1 = 0")
			}
			return db.Where(fields.DeptColumn+" IN ?", scope.DeptIDs)

		default:
			// 未知类型 → fail-closed
			return db.Where("1 = 0")
		}
	}
}
