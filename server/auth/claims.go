// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   JWT Claims 与 TokenPair 结构体定义
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:08:00
// +----------------------------------------------------------------------

package auth

// Claims 表示 JWT 令牌中的声明。
// 底座不解释 Subject 的业务含义，由调用方自行定义。
type Claims struct {
	Issuer    string         `json:"iss"`
	Subject   string         `json:"sub"`
	TokenType string         `json:"tt"`  // "access" | "refresh"
	JTI       string         `json:"jti"` // UUIDv7
	IssuedAt  int64          `json:"iat"`
	NotBefore int64          `json:"nbf"`
	ExpiresAt int64          `json:"exp"`
	Extra     map[string]any `json:"extra,omitempty"`
}

// TokenPair 表示签发的访问令牌与刷新令牌对。
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	AccessExp    int64  `json:"access_exp"`
	RefreshExp   int64  `json:"refresh_exp"`
}

// Token 类型常量。
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)
