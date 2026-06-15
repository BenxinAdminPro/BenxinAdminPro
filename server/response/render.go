// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   统一响应渲染 — Success / Fail / FailMsg / 分页包络
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 00:02:00
// | @updated   2026-06-08 02:35:00
// | @updated   2026-06-15 17:43:54  T-009b：补全 file 错误 6 键 + 菜单父节点新键文案（消除返裸 i18n key）
// | @updated   2026-06-15 18:10:00  T-009b 搭车（PM 裁定 A）：补 config 中心 2 键文案（config_decrypt_failed/migration_failed）
// +----------------------------------------------------------------------

package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// i18n 消息默认表（中性/中文）。
// 各模块注册码时提供 i18nKey，此表做兜底查找。
var defaultMessages = map[string]string{
	"security.missing_headers":  "缺少安全头",
	"security.timestamp_expired": "时间戳过期",
	"security.sign_invalid":     "签名无效",
	"security.nonce_replay":     "重放攻击",
	"security.decrypt_failed":   "解密失败",
	"security.token_invalid":    "令牌无效",
	"security.token_expired":    "令牌已过期",
	"security.token_revoked":    "令牌已吊销",
	"security.forbidden":        "无权限",
	"auth.bad_credentials":      "用户名或密码错误",
	"auth.captcha_required":     "需要验证码",
	"auth.captcha_invalid":      "验证码错误",
	"auth.account_locked":       "账号已锁定",
	"auth.account_disabled":     "账号已禁用",
	"sys.user_not_found":        "用户不存在",
	"sys.username_exists":       "用户名已存在",
	"sys.dept_has_children":     "部门下有子部门",
	"sys.dept_has_users":        "部门下有用户",
	"sys.post_code_exists":      "岗位编码已存在",
	"sys.invalid_parent_dept":   "无效的父部门",
	"sys.role_code_exists":      "角色编码已存在",
	"sys.menu_perm_required":    "按钮权限码必填",
	"sys.menu_perm_forbidden":   "目录/菜单不能设权限码",
	"sys.menu_has_children":     "菜单下有子节点",
	"sys.role_in_use":           "角色仍在使用中",
	"sys.invalid_id":            "无效的 ID",
	"sys.perm_code_exists":      "权限码已存在",
	"sys.invalid_parent_menu":   "无效的父菜单", // T-009b：菜单专属父节点（区别于父部门）
	"sys.invalid_data_scope":    "无效的数据权限范围",
	"sys.dict_type_exists":      "字典类型已存在",
	"sys.dict_type_not_found":   "字典类型不存在",
	"sys.config_key_exists":     "参数键已存在",
	"sys.config_not_found":      "参数不存在",
	// T-009b：补全 file 错误 6 键文案（此前缺失→经渲染层返裸 i18n key，见 T-007f §8-2）
	"sys.file_too_large":        "文件大小超出限制",
	"sys.file_ext_not_allowed":  "文件扩展名不允许",
	"sys.file_type_mismatch":    "文件类型与扩展名不匹配",
	"sys.file_not_found":        "文件不存在",
	"sys.file_name_invalid":     "无效的文件名",
	"sys.storage_failed":        "存储操作失败",
	// T-009b 搭车（PM 裁定 A）：补 config 中心 2 键文案（此前缺失→返裸 key，与 file 6 键同类）
	"sys.config_decrypt_failed": "参数解密失败",
	"sys.migration_failed":      "数据库迁移失败",
}

// MessageFunc 自定义消息查找函数。
type MessageFunc func(i18nKey string) string

// Renderer 统一响应渲染器。
type Renderer struct {
	registry *Registry
	msgFn    MessageFunc
}

// NewRenderer 创建渲染器。
func NewRenderer(r *Registry, msgFn MessageFunc) *Renderer {
	if msgFn == nil {
		msgFn = defaultMessageLookup
	}
	return &Renderer{registry: r, msgFn: msgFn}
}

func defaultMessageLookup(key string) string {
	if msg, ok := defaultMessages[key]; ok {
		return msg
	}
	return key
}

// Success 成功响应。
func (r *Renderer) Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": data})
}

// Fail 按错误码渲染失败响应。
func (r *Renderer) Fail(c *gin.Context, code int) {
	spec, ok := r.registry.Lookup(code)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": code, "message": "unknown error"})
		return
	}
	c.JSON(spec.HTTP, gin.H{"code": code, "message": r.msgFn(spec.I18nKey)})
}

// FailMsg 按错误码渲染失败响应，覆盖消息。
func (r *Renderer) FailMsg(c *gin.Context, code int, msg string) {
	spec, ok := r.registry.Lookup(code)
	httpStatus := http.StatusInternalServerError
	if ok {
		httpStatus = spec.HTTP
	}
	c.JSON(httpStatus, gin.H{"code": code, "message": msg})
}

// Page 分页成功响应。
func (r *Renderer) Page(c *gin.Context, list any, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "ok",
		"data": gin.H{
			"list":      list,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// ---------------------------------------------------------------------------
// 便捷函数（T-004c 收口：handler/中间件统一调用入口）
// ---------------------------------------------------------------------------

// Coder 接口：任何带 GetCode() 的错误（errcode.Error 实现此接口）。
// response 包不 import errcode — 通过此接口解耦。
type Coder interface {
	error
	GetCode() int
}

// HandleError 从 error 提取 code → Registry Lookup → 渲染。
// 未识别错误 → 500 internal error。handler 统一入口。
func (r *Renderer) HandleError(c *gin.Context, err error) {
	if coded, ok := err.(Coder); ok {
		r.Fail(c, coded.GetCode())
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "internal error"})
}

// AbortWithError 中间件版本：提取 code → Lookup → AbortWithStatusJSON。
func (r *Renderer) AbortWithError(c *gin.Context, code int) {
	spec, ok := r.registry.Lookup(code)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": code, "message": "unknown error"})
		return
	}
	c.AbortWithStatusJSON(spec.HTTP, gin.H{"code": code, "message": r.msgFn(spec.I18nKey)})
}

// OK 成功响应便捷函数。
func (r *Renderer) OK(c *gin.Context, data any) {
	r.Success(c, data)
}

// Bad 请求参数错误。
func (r *Renderer) Bad(c *gin.Context) {
	c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "invalid request"})
}

// ---------------------------------------------------------------------------
// 包级便捷函数（渲染器初始化后全局可用）
// ---------------------------------------------------------------------------

var defaultRenderer *Renderer

// SetDefault 设置全局默认渲染器（启动时调用一次）。
func SetDefault(r *Renderer) { defaultRenderer = r }

// ErrResp 从 error 提取 code → 渲染。handler 统一入口。
func ErrResp(c *gin.Context, err error) {
	if defaultRenderer != nil {
		defaultRenderer.HandleError(c, err)
		return
	}
	// 兜底（未初始化时）
	c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "internal error"})
}

// AbortErr 中间件版本。
func AbortErr(c *gin.Context, code int) {
	if defaultRenderer != nil {
		defaultRenderer.AbortWithError(c, code)
		return
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": code, "message": "unknown error"})
}

// OK 成功响应。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": data})
}

// BadReq 请求参数错误。
func BadReq(c *gin.Context) {
	c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "invalid request"})
}
