// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   安全模块最小错误码契约 — offset 常量 + HTTP 映射 + Registry
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:00:00
// | @updated   2026-06-08 03:06:00
// | @updated   2026-06-15 17:43:54  T-009b：新增 ErrInvalidParentMenu（offset 47，菜单专属父节点错误码，段尾纯追加）
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
	OffsetInvalidParentMenu = 47 // T-009b：菜单专属父节点错误码（段尾纯追加，不复用 dept 的 35）

	// T-003c 数据权限错误码 offset（50~59 段）
	OffsetInvalidDataScope = 50

	// T-004a 系统管理错误码 offset（60~69 段）
	OffsetDictTypeExists   = 60
	OffsetDictTypeNotFound = 61
	OffsetConfigKeyExists  = 62
	OffsetConfigNotFound   = 63

	// T-004b 文件管理错误码 offset（70~79 段）
	OffsetFileTooLarge     = 70
	OffsetFileExtNotAllowed = 71
	OffsetFileTypeMismatch = 72
	OffsetFileNotFound     = 73
	OffsetFileNameInvalid  = 74
	OffsetStorageFailed    = 75

	// T-005 配置中心错误码 offset（80~89 段）
	OffsetConfigDecryptFailed = 80
	OffsetMigrationFailed     = 81
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
	OffsetInvalidParentMenu:     400,
	OffsetInvalidDataScope:      400,
	OffsetDictTypeExists:       409,
	OffsetDictTypeNotFound:     404,
	OffsetConfigKeyExists:      409,
	OffsetConfigNotFound:       404,
	OffsetFileTooLarge:        413,
	OffsetFileExtNotAllowed:   415,
	OffsetFileTypeMismatch:    415,
	OffsetFileNotFound:        404,
	OffsetFileNameInvalid:     400,
	OffsetStorageFailed:       500,
	OffsetConfigDecryptFailed: 500,
	OffsetMigrationFailed:     500,
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
	OffsetInvalidParentMenu:     "sys.invalid_parent_menu",
	OffsetInvalidDataScope:      "sys.invalid_data_scope",
	OffsetDictTypeExists:       "sys.dict_type_exists",
	OffsetDictTypeNotFound:     "sys.dict_type_not_found",
	OffsetConfigKeyExists:      "sys.config_key_exists",
	OffsetConfigNotFound:       "sys.config_not_found",
	OffsetFileTooLarge:        "sys.file_too_large",
	OffsetFileExtNotAllowed:   "sys.file_ext_not_allowed",
	OffsetFileTypeMismatch:    "sys.file_type_mismatch",
	OffsetFileNotFound:        "sys.file_not_found",
	OffsetFileNameInvalid:     "sys.file_name_invalid",
	OffsetStorageFailed:       "sys.storage_failed",
	OffsetConfigDecryptFailed: "sys.config_decrypt_failed",
	OffsetMigrationFailed:     "sys.migration_failed",
}

// ---------------------------------------------------------------------------
// Error 类型
// ---------------------------------------------------------------------------

// Error 是带 code 的纯错误信号。
// T-004c 降级：不再携带 HTTP 字段和自渲染逻辑。
// HTTP 状态与 message 的唯一权威 = response.Registry。
type Error struct {
	Code int // 最终码（segmentBase + offset）
}

// Error 实现 error 接口。
func (e Error) Error() string {
	return fmt.Sprintf("errcode:%d", e.Code)
}

// GetCode 返回错误码（供 response.HandleError 提取）。
func (e Error) GetCode() int { return e.Code }

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
	ErrInvalidParentMenu Error

	// T-003c 数据权限
	ErrInvalidDataScope Error

	// T-004a 系统管理
	ErrDictTypeExists   Error
	ErrDictTypeNotFound Error
	ErrConfigKeyExists  Error
	ErrConfigNotFound   Error

	// T-004b 文件管理
	ErrFileTooLarge     Error
	ErrFileExtNotAllowed Error
	ErrFileTypeMismatch Error
	ErrFileNotFound     Error
	ErrFileNameInvalid  Error
	ErrStorageFailed    Error

	// T-005 配置中心
	ErrConfigDecryptFailed Error
	ErrMigrationFailed     Error
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
		ErrInvalidParentMenu:     newErr(segmentBase, OffsetInvalidParentMenu),
		ErrInvalidDataScope:     newErr(segmentBase, OffsetInvalidDataScope),
		ErrDictTypeExists:      newErr(segmentBase, OffsetDictTypeExists),
		ErrDictTypeNotFound:    newErr(segmentBase, OffsetDictTypeNotFound),
		ErrConfigKeyExists:     newErr(segmentBase, OffsetConfigKeyExists),
		ErrConfigNotFound:      newErr(segmentBase, OffsetConfigNotFound),
		ErrFileTooLarge:        newErr(segmentBase, OffsetFileTooLarge),
		ErrFileExtNotAllowed:   newErr(segmentBase, OffsetFileExtNotAllowed),
		ErrFileTypeMismatch:    newErr(segmentBase, OffsetFileTypeMismatch),
		ErrFileNotFound:        newErr(segmentBase, OffsetFileNotFound),
		ErrFileNameInvalid:     newErr(segmentBase, OffsetFileNameInvalid),
		ErrStorageFailed:       newErr(segmentBase, OffsetStorageFailed),
		ErrConfigDecryptFailed: newErr(segmentBase, OffsetConfigDecryptFailed),
		ErrMigrationFailed:     newErr(segmentBase, OffsetMigrationFailed),
	}, nil
}

// AllSpecs 返回所有已注册错误码的规格列表，供 response.Registry 注册。
// HTTP 和 i18nKey 从静态映射取（唯一来源），不从 Error 字段取（已剥离）。
func (r *Registry) AllSpecs() []struct{ Code, HTTP int; I18nKey string } {
	var specs []struct{ Code, HTTP int; I18nKey string }
	for _, pair := range r.allOffsets() {
		code := pair[0]
		offset := pair[1]
		specs = append(specs, struct{ Code, HTTP int; I18nKey string }{
			Code: code, HTTP: httpStatus[offset], I18nKey: i18nKeys[offset],
		})
	}
	return specs
}

// allOffsets 返回 (code, offset) 对。
func (r *Registry) allOffsets() [][2]int {
	errs := r.allErrors()
	offsets := []int{
		OffsetMissingSecurityHeaders, OffsetTimestampExpired, OffsetSignInvalid,
		OffsetNonceReplay, OffsetDecryptFailed, OffsetTokenInvalid,
		OffsetTokenExpired, OffsetTokenRevoked, OffsetForbidden,
		OffsetBadCredentials, OffsetCaptchaRequired, OffsetCaptchaInvalid,
		OffsetAccountLocked, OffsetAccountDisabled,
		OffsetUserNotFound, OffsetUsernameExists, OffsetDeptHasChildren,
		OffsetDeptHasUsers, OffsetPostCodeExists, OffsetInvalidParentDept,
		OffsetRoleCodeExists, OffsetMenuPermRequired, OffsetMenuPermForbidden,
		OffsetMenuHasChildren, OffsetRoleInUse, OffsetInvalidID, OffsetPermCodeExists,
		OffsetInvalidParentMenu,
		OffsetInvalidDataScope,
		OffsetDictTypeExists, OffsetDictTypeNotFound,
		OffsetConfigKeyExists, OffsetConfigNotFound,
		OffsetFileTooLarge, OffsetFileExtNotAllowed, OffsetFileTypeMismatch,
		OffsetFileNotFound, OffsetFileNameInvalid, OffsetStorageFailed,
		OffsetConfigDecryptFailed, OffsetMigrationFailed,
	}
	var pairs [][2]int
	for i, e := range errs {
		pairs = append(pairs, [2]int{e.Code, offsets[i]})
	}
	return pairs
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
		r.ErrInvalidParentMenu,
		r.ErrInvalidDataScope,
		r.ErrDictTypeExists, r.ErrDictTypeNotFound,
		r.ErrConfigKeyExists, r.ErrConfigNotFound,
		r.ErrFileTooLarge, r.ErrFileExtNotAllowed, r.ErrFileTypeMismatch,
		r.ErrFileNotFound, r.ErrFileNameInvalid, r.ErrStorageFailed,
		r.ErrConfigDecryptFailed, r.ErrMigrationFailed,
	}
}

func newErr(base, offset int) Error {
	return Error{Code: base + offset}
}

// HTTPStatus 返回 offset 对应的 HTTP 状态码（供 AllSpecs 用）。
func HTTPStatus(offset int) int { return httpStatus[offset] }

// I18nKey 返回 offset 对应的 i18n 消息键（供 AllSpecs 用）。
func I18nKey(offset int) string { return i18nKeys[offset] }
