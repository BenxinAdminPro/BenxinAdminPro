// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   组织架构 GORM 模型 — SysUser / SysDept / SysPost / SysUserPost
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 19:02:00
// +----------------------------------------------------------------------

package rbac

import (
	"time"

	"gorm.io/gorm"
)

// tablePrefix 由启动时注入，所有模型的 TableName 方法使用。
// 禁止硬编码；由 SetTablePrefix 在 main/bootstrap 中设置。
var tablePrefix string

// SetTablePrefix 设置表前缀，必须在任何数据库操作前调用。
func SetTablePrefix(prefix string) {
	tablePrefix = prefix
}

// GetTablePrefix 返回当前表前缀。
func GetTablePrefix() string {
	return tablePrefix
}

// ---------------------------------------------------------------------------
// SysUser 用户
// ---------------------------------------------------------------------------

// SysUser 用户表模型。
type SysUser struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"username"`
	PasswordHash string         `gorm:"type:varchar(255);not null" json:"-"` // 响应绝不暴露
	Nickname     string         `gorm:"type:varchar(64)" json:"nickname"`
	Avatar       string         `gorm:"type:varchar(255)" json:"avatar"`
	Email        string         `gorm:"type:varchar(128)" json:"email"`
	Mobile       string         `gorm:"type:varchar(32)" json:"mobile"`
	DeptID       *uint64        `gorm:"index" json:"dept_id"`
	Status       int8           `gorm:"type:tinyint;default:0;not null" json:"status"`
	Remark       string         `gorm:"type:varchar(255)" json:"remark"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Posts []SysPost `gorm:"many2many:sys_user_post;joinForeignKey:user_id;joinReferences:post_id" json:"posts,omitempty"`
}

func (SysUser) TableName() string { return tablePrefix + "sys_user" }

// ---------------------------------------------------------------------------
// SysDept 部门（树形）
// ---------------------------------------------------------------------------

// SysDept 部门表模型。
type SysDept struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	ParentID  uint64         `gorm:"default:0;not null" json:"parent_id"`
	Ancestors string         `gorm:"type:varchar(255);default:'0'" json:"ancestors"`
	Name      string         `gorm:"type:varchar(64);not null" json:"name"`
	Sort      int            `gorm:"default:0" json:"sort"`
	Leader    string         `gorm:"type:varchar(64)" json:"leader"`
	Status    int8           `gorm:"type:tinyint;default:0;not null" json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 非持久化字段，用于树构建
	Children []*SysDept `gorm:"-" json:"children,omitempty"`
}

func (SysDept) TableName() string { return tablePrefix + "sys_dept" }

// ---------------------------------------------------------------------------
// SysPost 岗位
// ---------------------------------------------------------------------------

// SysPost 岗位表模型。
type SysPost struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Code      string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"code"`
	Name      string         `gorm:"type:varchar(64);not null" json:"name"`
	Sort      int            `gorm:"default:0" json:"sort"`
	Status    int8           `gorm:"type:tinyint;default:0;not null" json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (SysPost) TableName() string { return tablePrefix + "sys_post" }

// ---------------------------------------------------------------------------
// SysUserPost 用户-岗位关联（多对多）
// ---------------------------------------------------------------------------

// SysUserPost 用户岗位关联表。
type SysUserPost struct {
	UserID uint64 `gorm:"primaryKey" json:"user_id"`
	PostID uint64 `gorm:"primaryKey" json:"post_id"`
}

func (SysUserPost) TableName() string { return tablePrefix + "sys_user_post" }

// ---------------------------------------------------------------------------
// 分页
// ---------------------------------------------------------------------------

// PageResult 分页返回结构。
type PageResult struct {
	List     any   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}
