// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   GormUserProvider — auth.UserProvider 的 GORM 真实现
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 19:10:00
// +----------------------------------------------------------------------

package rbac

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/benxin_dev/benxinadminpro-server/auth"
	"gorm.io/gorm"
)

// GormUserProvider 实现 auth.UserProvider，从 sys_user 表查找认证信息。
// 只读认证所需字段，不泄漏业务字段给 auth 包。
type GormUserProvider struct {
	db *gorm.DB
}

// NewGormUserProvider 创建 GORM UserProvider。
func NewGormUserProvider(db *gorm.DB) *GormUserProvider {
	return &GormUserProvider{db: db}
}

// FindByUsername 按用户名查找。未找到返回 auth.ErrUserNotFound。
func (p *GormUserProvider) FindByUsername(ctx context.Context, username string) (*auth.AuthUser, error) {
	var user SysUser
	err := p.db.WithContext(ctx).
		Select("id", "username", "password_hash", "status").
		Where("username = ?", username).
		First(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, auth.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("rbac: find user by username: %w", err)
	}

	return &auth.AuthUser{
		ID:           strconv.FormatUint(user.ID, 10),
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
		Status:       int(user.Status),
	}, nil
}

// 接口合规性检查
var _ auth.UserProvider = (*GormUserProvider)(nil)
