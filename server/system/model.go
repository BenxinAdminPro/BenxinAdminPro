// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   系统管理模型 — 字典/参数/操作日志/登录日志
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 00:10:00
// | @updated   2026-06-08 03:04:00
// | @updated   2026-06-14 10:35:00  T-005b-4：SysOperLog 加非持久化 OperatorName（操作人可读化出参）
// +----------------------------------------------------------------------

package system

import (
	"time"

	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// 字典
// ---------------------------------------------------------------------------

// SysDictType 字典类型。
type SysDictType struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	DictType  string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"dict_type"`
	Name      string         `gorm:"type:varchar(64);not null" json:"name"`
	Status    int8           `gorm:"type:tinyint;default:0" json:"status"`
	Remark    string         `gorm:"type:varchar(255)" json:"remark"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// SysDictData 字典项。
type SysDictData struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	DictType  string         `gorm:"type:varchar(64);not null;index" json:"dict_type"`
	Label     string         `gorm:"type:varchar(64);not null" json:"label"`
	Value     string         `gorm:"type:varchar(64);not null" json:"value"`
	Sort      int            `gorm:"default:0" json:"sort"`
	Status    int8           `gorm:"type:tinyint;default:0" json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// ---------------------------------------------------------------------------
// 参数
// ---------------------------------------------------------------------------

// SysConfig 系统参数。
// T-005：is_encrypted=1 的参数 GCM 加密存储，读自动解密。
type SysConfig struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ConfigKey   string    `gorm:"type:varchar(128);uniqueIndex;not null" json:"config_key"`
	ConfigValue string    `gorm:"type:text" json:"config_value"`
	Name        string    `gorm:"type:varchar(64)" json:"name"`
	Remark      string    `gorm:"type:varchar(255)" json:"remark"`
	IsEncrypted int8      `gorm:"type:tinyint;default:0;not null" json:"is_encrypted"` // 0=明文 1=GCM 加密
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SysMigration 迁移版本记录。
type SysMigration struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	Version   string    `gorm:"type:varchar(128);uniqueIndex;not null"`
	AppliedAt time.Time `gorm:"autoCreateTime"`
	Checksum  string    `gorm:"type:varchar(64)"`
}

// ---------------------------------------------------------------------------
// 操作日志
// ---------------------------------------------------------------------------

// SysOperLog 操作日志。
type SysOperLog struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Operator   string    `gorm:"type:varchar(64)" json:"operator"`
	Method     string    `gorm:"type:varchar(8)" json:"method"`
	Path       string    `gorm:"type:varchar(255)" json:"path"`
	PermCode   string    `gorm:"type:varchar(128)" json:"perm_code"`
	IP         string    `gorm:"type:varchar(64)" json:"ip"`
	UserAgent  string    `gorm:"type:varchar(255)" json:"user_agent"`
	ReqSummary string    `gorm:"type:text" json:"req_summary"`
	ResultCode int       `gorm:"type:int" json:"result_code"`
	LatencyMs  int       `gorm:"type:int" json:"latency_ms"`
	CreatedAt  time.Time `json:"created_at"`

	// 非持久化：操作人可读化展示名（由 service JOIN sys_user 解析填充；
	// gorm:"-" 不参与读写/迁移，仅随 json 出参，operator 内部 ID 不直接暴露）。
	OperatorName string `gorm:"-" json:"operator_name"`
}

// ---------------------------------------------------------------------------
// 登录日志
// ---------------------------------------------------------------------------

// SysLoginLog 登录日志。
type SysLoginLog struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string    `gorm:"type:varchar(64)" json:"username"`
	IP        string    `gorm:"type:varchar(64)" json:"ip"`
	UserAgent string    `gorm:"type:varchar(255)" json:"user_agent"`
	Success   int8      `gorm:"type:tinyint" json:"success"`
	Reason    string    `gorm:"type:varchar(128)" json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}
