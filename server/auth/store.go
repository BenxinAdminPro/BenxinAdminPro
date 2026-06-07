// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   BlacklistStore 接口 — JWT jti 黑名单存储抽象
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:09:00
// +----------------------------------------------------------------------

package auth

import (
	"context"
	"time"
)

// BlacklistStore 抽象 JWT jti 黑名单存储。
// 生产实现使用 Redis；单测使用内存假实现。
type BlacklistStore interface {
	// Add 将 jti 加入黑名单，TTL 到期后自动移除。
	Add(ctx context.Context, jti string, ttl time.Duration) error

	// IsBlacklisted 检查 jti 是否在黑名单中。
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}
