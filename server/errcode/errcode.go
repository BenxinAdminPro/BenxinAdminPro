// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   安全模块最小错误码契约 — offset 常量 + HTTP 映射 + Registry
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:00:00
// | @updated   2026-06-07 23:30:00
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

	// T-003a 组织架构错误码 offset（30~39 段）
	OffsetUserNotFound     = 30
	OffsetUsernameExists   = 31
	OffsetDeptHasChildren  = 32
	OffsetDeptHasUsers     = 33
	OffsetPostCodeExists   = 34
	OffsetInvalidParentDept = 35

	// T-003b RBAC 核心错误码 offset（40~49 段）
	OffsetRoleCodeExists    = 40
	OffsetMenuPermRequired  = 41
	OffsetMenuPermForbidden = 42
	OffsetMenuHasChildren   = 43
	OffsetRoleInUse         = 44
	OffsetInvalidID         = 45
	OffsetPermCodeExists    = 46

	// T-003c 数据权限错误码 offset（50~59 段）
	OffsetInvalidDataScope = 50

	// T-004a 系统管理错误码 offset（60~69 段）
	OffsetDictTypeExists   = 60
	OffsetDictTypeNotFound = 61
	OffsetConfigKeyExists  = 62
	OffsetConfigNotFound   = 63
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
	OffsetUserNotFound:           404,
	OffsetUsernameExists:         409,
	OffsetDeptHasChildren:        409,
	OffsetDeptHasUsers:           409,
	OffsetPostCodeExists:         409,
	OffsetInvalidParentDept:      400,
	OffsetRoleCodeExists:        409,
	OffsetMenuPermRequired:      400,
	OffsetMenuPermForbidden:     400,
	OffsetMenuHasChildren:       409,
	OffsetRoleInUse:             409,
	OffsetInvalidID:             400,
	OffsetPermCodeExists:        409,
	OffsetInvalidDataScope:      400,
	OffsetDictTypeExists:       409,
	OffsetDictTypeNotFound:     404,
	OffsetConfigKeyExists:      409,
	OffsetConfigNotFound:       404,
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
	OffsetUserNotFound:           "sys.user_not_found",
	OffsetUsernameExists:         "sys.username_exists",
	OffsetDeptHasChildren:        "sys.dept_has_children",
	OffsetDeptHasUsers:           "sys.dept_has_users",
	OffsetPostCodeExists:         "sys.post_code_exists",
	OffsetInvalidParentDept:      "sys.invalid_parent_dept",
	OffsetRoleCodeExists:        "sys.role_code_exists",
	OffsetMenuPermRequired:      "sys.menu_perm_required",
	OffsetMenuPermForbidden:     "sys.menu_perm_forbidden",
	OffsetMenuHasChildren:       "sys.menu_has_children",
	OffsetRoleInUse:             "sys.role_in_use",
	OffsetInvalidID:             "sys.invalid_id",
	OffsetPermCodeExists:        "sys.perm_code_exists",
	OffsetInvalidDataScope:      "sys.invalid_data_scope",
	OffsetDictTypeExists:       "sys.dict_type_exists",
	OffsetDictTypeNotFound:     "sys.dict_type_not_found",
	OffsetConfigKeyExists:      "sys.config_key_exists",
	OffsetConfigNotFound:       "sys.config_not_found",
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

	// T-003a 组织架构
	ErrUserNotFound     Error
	ErrUsernameExists   Error
	ErrDeptHasChildren  Error
	ErrDeptHasUsers     Error
	ErrPostCodeExists   Error
	ErrInvalidParentDept Error

	// T-003b RBAC 核心
	ErrRoleCodeExists    Error
	ErrMenuPermRequired  Error
	ErrMenuPermForbidden Error
	ErrMenuHasChildren   Error
	ErrRoleInUse         Error
	ErrInvalidID         Error
	ErrPermCodeExists    Error

	// T-003c 数据权限
	ErrInvalidDataScope Error

	// T-004a 系统管理
	ErrDictTypeExists   Error
	ErrDictTypeNotFound Error
	ErrConfigKeyExists  Error
	ErrConfigNotFound   Error
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
		ErrUserNotFound:           newErr(segmentBase, OffsetUserNotFound),
		ErrUsernameExists:         newErr(segmentBase, OffsetUsernameExists),
		ErrDeptHasChildren:        newErr(segmentBase, OffsetDeptHasChildren),
		ErrDeptHasUsers:           newErr(segmentBase, OffsetDeptHasUsers),
		ErrPostCodeExists:         newErr(segmentBase, OffsetPostCodeExists),
		ErrInvalidParentDept:      newErr(segmentBase, OffsetInvalidParentDept),
		ErrRoleCodeExists:        newErr(segmentBase, OffsetRoleCodeExists),
		ErrMenuPermRequired:      newErr(segmentBase, OffsetMenuPermRequired),
		ErrMenuPermForbidden:     newErr(segmentBase, OffsetMenuPermForbidden),
		ErrMenuHasChildren:       newErr(segmentBase, OffsetMenuHasChildren),
		ErrRoleInUse:             newErr(segmentBase, OffsetRoleInUse),
		ErrInvalidID:             newErr(segmentBase, OffsetInvalidID),
		ErrPermCodeExists:        newErr(segmentBase, OffsetPermCodeExists),
		ErrInvalidDataScope:     newErr(segmentBase, OffsetInvalidDataScope),
		ErrDictTypeExists:      newErr(segmentBase, OffsetDictTypeExists),
		ErrDictTypeNotFound:    newErr(segmentBase, OffsetDictTypeNotFound),
		ErrConfigKeyExists:     newErr(segmentBase, OffsetConfigKeyExists),
		ErrConfigNotFound:      newErr(segmentBase, OffsetConfigNotFound),
	}, nil
}

// AllSpecs 返回所有已注册错误码的规格列表，供 response.Registry 迁移使用。
func (r *Registry) AllSpecs() []struct{ Code, HTTP int; I18nKey string } {
	var specs []struct{ Code, HTTP int; I18nKey string }
	for _, e := range r.allErrors() {
		specs = append(specs, struct{ Code, HTTP int; I18nKey string }{e.Code, e.HTTP, e.Message})
	}
	return specs
}

func (r *Registry) allErrors() []Error {
	return []Error{
		r.ErrMissingSecurityHeaders, r.ErrTimestampExpired, r.ErrSignInvalid,
		r.ErrNonceReplay, r.ErrDecryptFailed, r.ErrTokenInvalid,
		r.ErrTokenExpired, r.ErrTokenRevoked, r.ErrForbidden,
		r.ErrBadCredentials, r.ErrCaptchaRequired, r.ErrCaptchaInvalid,
		r.ErrAccountLocked, r.ErrAccountDisabled,
		r.ErrUserNotFound, r.ErrUsernameExists, r.ErrDeptHasChildren,
		r.ErrDeptHasUsers, r.ErrPostCodeExists, r.ErrInvalidParentDept,
		r.ErrRoleCodeExists, r.ErrMenuPermRequired, r.ErrMenuPermForbidden,
		r.ErrMenuHasChildren, r.ErrRoleInUse, r.ErrInvalidID, r.ErrPermCodeExists,
		r.ErrInvalidDataScope,
		r.ErrDictTypeExists, r.ErrDictTypeNotFound,
		r.ErrConfigKeyExists, r.ErrConfigNotFound,
	}
}

func newErr(base, offset int) Error {
	return Error{
		Code:    base + offset,
		Message: i18nKeys[offset],
		HTTP:    httpStatus[offset],
	}
}
