// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   TokenService 接口定义 — 令牌签发/解析/校验/刷新/注销
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:08:00
// +----------------------------------------------------------------------

package auth

import "context"

// TokenService 定义 JWT 令牌服务的公共接口。
// 业务中立：不引用用户/角色表，不做密码校验。
type TokenService interface {
	// IssuePair 签发一对令牌；subject 与 extra 由调用方提供，底座不校验业务含义。
	IssuePair(ctx context.Context, subject string, extra map[string]any) (TokenPair, error)

	// Parse 解析令牌（不校验黑名单，仅验签名和格式）。
	Parse(token string) (*Claims, error)

	// Verify 完整校验：签名 + 过期 + nbf + iss + tt + 黑名单。
	Verify(ctx context.Context, token string, expectType string) (*Claims, error)

	// Refresh 校验 refresh 令牌 → 签发新 access（默认轮换 refresh 并拉黑旧 refresh jti）。
	Refresh(ctx context.Context, refreshToken string) (TokenPair, error)

	// Revoke 把 jti 写入黑名单，TTL = max(0, exp-now)。
	Revoke(ctx context.Context, jti string, exp int64) error
}
