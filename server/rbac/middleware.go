// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   Casbin 鉴权中间件 — RequirePerm 内联 enforce + 超管短路
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:13:00
// | @updated   2026-06-07 21:50:00
// | @updated   2026-06-08 02:40:00
// +----------------------------------------------------------------------

package rbac

import (
	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/benxin_dev/benxinadminpro-server/response"
	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
)

// SubjectFunc 从 Gin 上下文提取当前请求主体标识。
type SubjectFunc func(c *gin.Context) string

// AuthzConfig Authz 配置，由 main/bootstrap 注入。
type AuthzConfig struct {
	SuperAdminRoles []string // 超管角色 code 列表，配置注入，禁硬编码
}

// AuthzEnforcer 持有鉴权所需的依赖，供 RequirePerm 使用。
type AuthzEnforcer struct {
	enforcer  *casbin.Enforcer
	superSet  map[string]bool
	subjectFn SubjectFunc
	errs      *errcode.Registry
}

// NewAuthzEnforcer 创建 AuthzEnforcer。
func NewAuthzEnforcer(e *casbin.Enforcer, cfg AuthzConfig, subjectFn SubjectFunc, errs *errcode.Registry) *AuthzEnforcer {
	superSet := make(map[string]bool, len(cfg.SuperAdminRoles))
	for _, r := range cfg.SuperAdminRoles {
		superSet[r] = true
	}
	return &AuthzEnforcer{enforcer: e, superSet: superSet, subjectFn: subjectFn, errs: errs}
}

// RequirePerm 返回 Gin 中间件，内联执行 Casbin enforce。
// 用法：rg.GET("/sys/users", authz.RequirePerm("sys:user:list"), handler)
func (a *AuthzEnforcer) RequirePerm(permCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		sub := a.subjectFn(c)

		// 超管短路
		if len(a.superSet) > 0 {
			roles, _ := a.enforcer.GetRolesForUser(sub)
			for _, r := range roles {
				if a.superSet[r] {
					c.Next()
					return
				}
			}
		}

		// 非超管走 enforce
		allowed, err := a.enforcer.Enforce(sub, permCode, PolicyAct)
		if err != nil || !allowed {
			response.AbortErr(c, a.errs.ErrForbidden.Code)
			return
		}

		c.Next()
	}
}

// RequirePerm 包级函数（向后兼容现有路由注册）。
// 如果不需要 enforce（如 demo 阶段），直接设 context value 跳过。
func RequirePerm(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("required_perm_code", code)
		c.Next()
	}
}

// Authz 返回全局 Casbin 鉴权中间件（读 RequirePerm 设的 context value）。
// 注意：必须在 RequirePerm 之后执行（同路由链中 RequirePerm 先设值，Authz 后读取不可行）。
// 推荐用法：路由上用 authzEnforcer.RequirePerm("code")，不用全局 Authz。
// 保留此函数供简单场景和向后兼容。
func Authz(e *casbin.Enforcer, cfg AuthzConfig, subjectFn SubjectFunc, errs *errcode.Registry) gin.HandlerFunc {
	ae := NewAuthzEnforcer(e, cfg, subjectFn, errs)
	return func(c *gin.Context) {
		permCode, exists := c.Get("required_perm_code")
		if !exists {
			c.Next()
			return
		}
		code, _ := permCode.(string)
		if code == "" {
			c.Next()
			return
		}
		// 复用 AuthzEnforcer 逻辑
		sub := ae.subjectFn(c)
		if len(ae.superSet) > 0 {
			roles, _ := ae.enforcer.GetRolesForUser(sub)
			for _, r := range roles {
				if ae.superSet[r] {
					c.Next()
					return
				}
			}
		}
		allowed, err := ae.enforcer.Enforce(sub, code, PolicyAct)
		if err != nil || !allowed {
			response.AbortErr(c, ae.errs.ErrForbidden.Code)
			return
		}
		c.Next()
	}
}
