// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   Casbin 鉴权中间件骨架 — Authz Gin HandlerFunc
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:13:00
// +----------------------------------------------------------------------

package rbac

import (
	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
)

// SubjectFunc 从 Gin 上下文提取当前请求主体标识。
// 由消费方注入（如从 JWT claims 取 sub），底座不假设主体来源。
type SubjectFunc func(c *gin.Context) string

// Authz 返回 Casbin 鉴权中间件。
//   - e: Casbin enforcer 实例
//   - subjectFn: 主体提取函数
//   - errs: 错误码注册表
//
// obj = request path, act = request method。
// Enforce 失败返回 403，不泄漏策略细节。
func Authz(e *casbin.Enforcer, subjectFn SubjectFunc, errs *errcode.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		sub := subjectFn(c)
		obj := c.Request.URL.Path
		act := c.Request.Method

		allowed, err := e.Enforce(sub, obj, act)
		if err != nil || !allowed {
			c.AbortWithStatusJSON(errs.ErrForbidden.HTTP, gin.H{
				"code":    errs.ErrForbidden.Code,
				"message": errs.ErrForbidden.Message,
			})
			return
		}

		c.Next()
	}
}
