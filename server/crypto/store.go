// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   ReplayStore 接口 — nonce 防重放存储抽象
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:03:00
// +----------------------------------------------------------------------

package crypto

import (
	"context"
	"time"
)

// ReplayStore 抽象 nonce 防重放存储。
// 生产实现使用 Redis SET NX EX；单测使用内存假实现。
type ReplayStore interface {
	// CheckAndSet 原子地检查并标记 nonce。
	// 返回 true 表示 nonce 已存在（重放攻击）；false 表示首次使用并已存储。
	CheckAndSet(ctx context.Context, nonce string, ttl time.Duration) (exists bool, err error)
}
