// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   JWT Claims 与 TokenPair 结构体定义
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:08:00
// | @updated   2026-06-13 15:30:00  T-005b-2：补 GetSubject() 修 uploader 恒空（鸭子类型断言此前静默失败）
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

// GetSubject 返回 Subject（JWT sub）。供不便直接 import auth 包的消费方（如 system 文件上传
// 经鸭子类型接口 interface{ GetSubject() string } 取上传人）使用。
// 修 T-007f 暴露的缺陷：system/handler_file.go 此前对 *auth.Claims 做该接口断言，但 Claims 无此方法、
// 断言静默失败返零值 → sys_file.uploader 恒空。补此方法后断言成立、uploader 真落值。
// context 存的是 *auth.Claims（指针），故用指针接收者即满足 *Claims 的方法集。
func (c *Claims) GetSubject() string {
	return c.Subject
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
