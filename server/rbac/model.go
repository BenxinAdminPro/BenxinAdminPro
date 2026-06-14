// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   组织架构 GORM 模型 — SysUser / SysDept / SysPost / SysUserPost
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 19:02:00
// | @updated   2026-06-07 20:00:00
// | @updated   2026-06-14 15:30:00  T-008b：SysUser 加非持久化 Roles（详情 Get 预载，分配角色弹窗回填来源）
// +----------------------------------------------------------------------

package rbac

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// NewDBConfig 返回带表前缀的 GORM 配置。
// 前缀绑定在 *gorm.DB 实例上（NamingStrategy），禁止包级全局变量。
// 调用方：gorm.Open(dialector, rbac.NewDBConfig("kp_"))
func NewDBConfig(tablePrefix string) *gorm.Config {
	return &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   tablePrefix,
			SingularTable: true,
		},
	}
}

// ---------------------------------------------------------------------------
// SysUser 用户
// ---------------------------------------------------------------------------

// SysUser 用户表模型。表名由 NamingStrategy 前缀决定：{prefix}sys_user。
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

	// 非持久化，由 service 手动填充
	Posts []SysPost `gorm:"-" json:"posts,omitempty"`
	// T-008b：非持久化已授角色（仅详情 Get 预载，用于「分配角色」弹窗回填；List 不载故不污染列表出参）
	Roles []SysRole `gorm:"-" json:"roles,omitempty"`
}

// ---------------------------------------------------------------------------
// SysDept 部门（树形）
// ---------------------------------------------------------------------------

// SysDept 部门表模型。表名：{prefix}sys_dept。
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

// ---------------------------------------------------------------------------
// SysPost 岗位
// ---------------------------------------------------------------------------

// SysPost 岗位表模型。表名：{prefix}sys_post。
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

// ---------------------------------------------------------------------------
// SysUserPost 用户-岗位关联（多对多）
// ---------------------------------------------------------------------------

// SysUserPost 用户岗位关联表。表名：{prefix}sys_user_post。
type SysUserPost struct {
	UserID uint64 `gorm:"primaryKey" json:"user_id"`
	PostID uint64 `gorm:"primaryKey" json:"post_id"`
}

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
