// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   文件元信息模型 — sys_file
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 01:18:00
// | @updated   2026-06-14 10:35:00  T-005b-4：SysFile 加非持久化 UploaderName（上传人可读化出参）
// | @updated   2026-06-17 00:00:00  T-013：StorageKey json tag 改 "-"（相对存储 key 不外露；DB 列/内部 Go 字段使用原样）
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
	StorageKey   string         `gorm:"type:varchar(512);not null" json:"-"` // T-013：相对存储 key 不外露（内部 Go 字段读写原样）
	StorageType  string         `gorm:"type:varchar(32);not null;default:'local'" json:"storage_type"`
	Size         int64          `gorm:"type:bigint" json:"size"`
	Mime         string         `gorm:"type:varchar(128)" json:"mime"`
	Ext          string         `gorm:"type:varchar(32)" json:"ext"`
	Uploader     string         `gorm:"type:varchar(64)" json:"uploader"`
	Status       int8           `gorm:"type:tinyint;default:0" json:"status"` // 0=正常 1=待清理
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// 非持久化：上传人可读化展示名（由 service JOIN sys_user 解析填充；
	// gorm:"-" 不参与读写/迁移，仅随 json 出参，uploader 内部 ID 不直接暴露）。
	UploaderName string `gorm:"-" json:"uploader_name"`
}
