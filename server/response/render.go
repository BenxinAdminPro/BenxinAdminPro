// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   统一响应渲染 — Success / Fail / FailMsg / 分页包络
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 00:02:00
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
	"sys.invalid_data_scope":    "无效的数据权限范围",
	"sys.dict_type_exists":      "字典类型已存在",
	"sys.dict_type_not_found":   "字典类型不存在",
	"sys.config_key_exists":     "参数键已存在",
	"sys.config_not_found":      "参数不存在",
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
