// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   安全模块最小错误码契约 — offset 常量 + HTTP 映射 + Registry
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:00:00
// | @updated   2026-06-07 17:00:00
// +----------------------------------------------------------------------

package errcode

import "fmt"

// ---------------------------------------------------------------------------
// Offset 常量（segment_base 在运行时注入，此处只定义偏移量）
// ---------------------------------------------------------------------------

const (
	OffsetMissingSecurityHeaders = 1
	OffsetTimestampExpired       = 2
	OffsetSignInvalid            = 3
	OffsetNonceReplay            = 4
	OffsetDecryptFailed          = 5
	OffsetTokenInvalid           = 6
	OffsetTokenExpired           = 7
	OffsetTokenRevoked           = 8
	OffsetForbidden              = 9

	// T-002 认证授权错误码 offset（20~29 段）
	OffsetBadCredentials  = 20
	OffsetCaptchaRequired = 21
	OffsetCaptchaInvalid  = 22
	OffsetAccountLocked   = 23
	OffsetAccountDisabled = 24
)

// ---------------------------------------------------------------------------
// HTTP status + i18n key 静态映射
// ---------------------------------------------------------------------------

var httpStatus = map[int]int{
	OffsetMissingSecurityHeaders: 400,
	OffsetTimestampExpired:       400,
	OffsetSignInvalid:            400,
	OffsetNonceReplay:            400,
	OffsetDecryptFailed:          400,
	OffsetTokenInvalid:           401,
	OffsetTokenExpired:           401,
	OffsetTokenRevoked:           401,
	OffsetForbidden:              403,
	OffsetBadCredentials:         401,
	OffsetCaptchaRequired:        400,
	OffsetCaptchaInvalid:         400,
	OffsetAccountLocked:          423,
	OffsetAccountDisabled:        403,
}

var i18nKeys = map[int]string{
	OffsetMissingSecurityHeaders: "security.missing_headers",
	OffsetTimestampExpired:       "security.timestamp_expired",
	OffsetSignInvalid:            "security.sign_invalid",
	OffsetNonceReplay:            "security.nonce_replay",
	OffsetDecryptFailed:          "security.decrypt_failed",
	OffsetTokenInvalid:           "security.token_invalid",
	OffsetTokenExpired:           "security.token_expired",
	OffsetTokenRevoked:           "security.token_revoked",
	OffsetForbidden:              "security.forbidden",
	OffsetBadCredentials:         "auth.bad_credentials",
	OffsetCaptchaRequired:        "auth.captcha_required",
	OffsetCaptchaInvalid:         "auth.captcha_invalid",
	OffsetAccountLocked:          "auth.account_locked",
	OffsetAccountDisabled:        "auth.account_disabled",
}

// ---------------------------------------------------------------------------
// Error 类型
// ---------------------------------------------------------------------------

// Error 表示一个已解析的安全错误码（code = segmentBase + offset）。
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	HTTP    int    `json:"-"`
}

// Error 实现 error 接口。
func (e Error) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// ---------------------------------------------------------------------------
// Registry — 启动时由 segmentBase 注入，解析出全部安全错误码
// ---------------------------------------------------------------------------

// Registry 持有当前应用实例化后的完整安全错误码集。
type Registry struct {
	// T-001 安全地基
	ErrMissingSecurityHeaders Error
	ErrTimestampExpired       Error
	ErrSignInvalid            Error
	ErrNonceReplay            Error
	ErrDecryptFailed          Error
	ErrTokenInvalid           Error
	ErrTokenExpired           Error
	ErrTokenRevoked           Error
	ErrForbidden              Error

	// T-002 认证授权
	ErrBadCredentials  Error
	ErrCaptchaRequired Error
	ErrCaptchaInvalid  Error
	ErrAccountLocked   Error
	ErrAccountDisabled Error
}

// NewRegistry 用配置注入的 segmentBase 构建错误码注册表。
// segmentBase 必须 > 0，否则返回错误。
func NewRegistry(segmentBase int) (*Registry, error) {
	if segmentBase <= 0 {
		return nil, fmt.Errorf("errcode: security_segment_base must be > 0, got %d", segmentBase)
	}
	return &Registry{
		ErrMissingSecurityHeaders: newErr(segmentBase, OffsetMissingSecurityHeaders),
		ErrTimestampExpired:       newErr(segmentBase, OffsetTimestampExpired),
		ErrSignInvalid:            newErr(segmentBase, OffsetSignInvalid),
		ErrNonceReplay:            newErr(segmentBase, OffsetNonceReplay),
		ErrDecryptFailed:          newErr(segmentBase, OffsetDecryptFailed),
		ErrTokenInvalid:           newErr(segmentBase, OffsetTokenInvalid),
		ErrTokenExpired:           newErr(segmentBase, OffsetTokenExpired),
		ErrTokenRevoked:           newErr(segmentBase, OffsetTokenRevoked),
		ErrForbidden:              newErr(segmentBase, OffsetForbidden),
		ErrBadCredentials:         newErr(segmentBase, OffsetBadCredentials),
		ErrCaptchaRequired:        newErr(segmentBase, OffsetCaptchaRequired),
		ErrCaptchaInvalid:         newErr(segmentBase, OffsetCaptchaInvalid),
		ErrAccountLocked:          newErr(segmentBase, OffsetAccountLocked),
		ErrAccountDisabled:        newErr(segmentBase, OffsetAccountDisabled),
	}, nil
}

func newErr(base, offset int) Error {
	return Error{
		Code:    base + offset,
		Message: i18nKeys[offset],
		HTTP:    httpStatus[offset],
	}
}
