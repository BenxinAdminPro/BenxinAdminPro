// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   RBAC 核心模型 — SysRole / SysMenu / SysRoleMenu / SysUserRole
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 21:02:00
// | @updated   2026-06-07 23:32:00
// | @updated   2026-06-14 18:10:00  T-008c：SysRole 加非持久化 MenuIDs（详情 Get 预载，授权树弹窗回填来源）
// +----------------------------------------------------------------------

package rbac

import (
	"time"

	"gorm.io/gorm"
)

// MenuType 菜单类型常量。
const (
	MenuTypeDir    = "M" // 目录
	MenuTypePage   = "C" // 菜单/页面
	MenuTypeButton = "F" // 按钮/权限点
)

// SysRole 角色。表名：{prefix}sys_role。
type SysRole struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Code      string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"code"`
	Name      string         `gorm:"type:varchar(64);not null" json:"name"`
	Sort      int            `gorm:"default:0" json:"sort"`
	Status    int8           `gorm:"type:tinyint;default:0;not null" json:"status"`
	DataScope int8          `gorm:"type:tinyint;default:2;not null;comment:数据权限(1=全部,2=本人,3=本部门)" json:"data_scope"`
	Remark    string         `gorm:"type:varchar(255)" json:"remark"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// T-008c：非持久化已授菜单 ID（全量 M/C/F 三层）。仅详情 Get 预载，供「分配菜单」授权树弹窗回填；
	// List 不载故不污染列表出参（omitempty）。enc.Role 输出时编码为 hashid 数组（同 SysUser.Roles 范式）。
	MenuIDs []uint64 `gorm:"-" json:"menu_ids,omitempty"`
}

// SysMenu 菜单/权限点。表名：{prefix}sys_menu。
// menu_type: M=目录, C=菜单页面, F=按钮权限点。
type SysMenu struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	ParentID  uint64         `gorm:"default:0;not null" json:"parent_id"`
	Ancestors string         `gorm:"type:varchar(255);default:'0'" json:"ancestors"`
	MenuType  string         `gorm:"type:char(1);not null" json:"menu_type"`
	Name      string         `gorm:"type:varchar(64);not null" json:"name"`
	PermCode  string         `gorm:"type:varchar(128)" json:"perm_code"`
	Path      string         `gorm:"type:varchar(255)" json:"path"`
	Component string         `gorm:"type:varchar(255)" json:"component"`
	Icon      string         `gorm:"type:varchar(64)" json:"icon"`
	Sort      int            `gorm:"default:0" json:"sort"`
	Visible   int8           `gorm:"type:tinyint;default:1" json:"visible"`
	Status    int8           `gorm:"type:tinyint;default:0;not null" json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Children []*SysMenu `gorm:"-" json:"children,omitempty"`
}

// SysRoleMenu 角色-菜单/权限关联。表名：{prefix}sys_role_menu。
type SysRoleMenu struct {
	RoleID uint64 `gorm:"primaryKey" json:"role_id"`
	MenuID uint64 `gorm:"primaryKey" json:"menu_id"`
}

// SysUserRole 用户-角色关联。表名：{prefix}sys_user_role。
type SysUserRole struct {
	UserID uint64 `gorm:"primaryKey" json:"user_id"`
	RoleID uint64 `gorm:"primaryKey" json:"role_id"`
}
