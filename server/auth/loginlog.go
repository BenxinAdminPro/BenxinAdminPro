// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   LoginLogger 接口 — 登录事件记录（auth 不依赖 DB，接口注入）
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 00:22:00
// +----------------------------------------------------------------------

package auth

import "context"

// LoginEvent 表示一次登录尝试。
type LoginEvent struct {
	Username  string
	IP        string
	UserAgent string
	Success   bool
	Reason    string // 失败原因码（如 bad_credentials/locked），成功为空
}

// LoginLogger 记录登录事件。
// auth 包只定义接口，GORM 实现放 system 侧，由装配注入。
// 为 nil 时登录流程不受影响（向后兼容）。
type LoginLogger interface {
	Log(ctx context.Context, event LoginEvent) error
}
