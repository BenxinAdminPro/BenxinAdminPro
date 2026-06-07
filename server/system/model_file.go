// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   文件元信息模型 — sys_file
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 01:18:00
// +----------------------------------------------------------------------

package system

import (
	"time"

	"gorm.io/gorm"
)

// SysFile 文件元信息。
type SysFile struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	OriginalName string         `gorm:"type:varchar(255);not null" json:"original_name"`
	StorageKey   string         `gorm:"type:varchar(512);not null" json:"storage_key"`
	StorageType  string         `gorm:"type:varchar(32);not null;default:'local'" json:"storage_type"`
	Size         int64          `gorm:"type:bigint" json:"size"`
	Mime         string         `gorm:"type:varchar(128)" json:"mime"`
	Ext          string         `gorm:"type:varchar(32)" json:"ext"`
	Uploader     string         `gorm:"type:varchar(64)" json:"uploader"`
	Status       int8           `gorm:"type:tinyint;default:0" json:"status"` // 0=正常 1=待清理
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}
