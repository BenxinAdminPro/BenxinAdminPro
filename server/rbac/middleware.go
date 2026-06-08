// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   Casbin 鉴权中间件 — RequirePerm 内联 enforce + 超管短路
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:13:00
// | @updated   2026-06-07 21:50:00
// | @updated   2026-06-08 02:40:00
// | @updated   2026-06-08 13:00:00  T-003d：删除"假"包级 RequirePerm 与无人用的 Authz 全局中间件，
// |                                  消除"挂了像鉴权实则放行"的陷阱；只保留真 enforce 的 AuthzEnforcer.RequirePerm
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

		// 记录本路由的 perm code 供下游审计（如 OperLog）读取——
		// enforce 本身用闭包捕获的 permCode，不依赖此 context value（非旧"假 RequirePerm"模式）。
		c.Set("required_perm_code", permCode)

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

// 说明（T-003d）：原包级 RequirePerm（只设 context value 不 enforce）与依赖它的
// Authz 全局中间件已删除——它们构成"看起来在鉴权、实则放行"的陷阱，且 Gin 中
// 组中间件先于路由中间件执行，全局 Authz 读不到路由级设的 context value，本就无法生效。
// 受保护路由一律用 AuthzEnforcer.RequirePerm(code) 在路由级直接 enforce（闭包捕获 code）。
