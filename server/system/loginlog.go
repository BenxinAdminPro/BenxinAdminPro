// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   LoginLogger GORM 实现 — 登录日志落库
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 00:24:00
// +----------------------------------------------------------------------

package system

import (
	"context"

	"github.com/benxin_dev/benxinadminpro-server/auth"
	"gorm.io/gorm"
)

// GormLoginLogger 实现 auth.LoginLogger，将登录事件写入 sys_login_log。
type GormLoginLogger struct {
	DB *gorm.DB
}

// NewGormLoginLogger 创建 GORM 登录日志记录器。
func NewGormLoginLogger(db *gorm.DB) *GormLoginLogger {
	return &GormLoginLogger{DB: db}
}

// Log 记录登录事件。
func (l *GormLoginLogger) Log(_ context.Context, event auth.LoginEvent) error {
	var success int8
	if event.Success {
		success = 1
	}
	entry := SysLoginLog{
		Username:  event.Username,
		IP:        event.IP,
		UserAgent: event.UserAgent,
		Success:   success,
		Reason:    event.Reason,
	}
	return l.DB.Create(&entry).Error
}

// 接口合规
var _ auth.LoginLogger = (*GormLoginLogger)(nil)
