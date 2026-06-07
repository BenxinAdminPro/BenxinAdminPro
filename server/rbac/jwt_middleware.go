// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   JWT 鉴权中间件 — admin 端路由守卫
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 19:20:00
// | @updated   2026-06-08 02:40:00
// +----------------------------------------------------------------------

package rbac

import (
	"strings"

	"github.com/benxin_dev/benxinadminpro-server/auth"
	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/benxin_dev/benxinadminpro-server/response"
	"github.com/gin-gonic/gin"
)

// CtxKeyClaims 是 JWT claims 在 gin.Context 中的存储键。
const CtxKeyClaims = "jwt_claims"

// JWTAuth 返回 JWT 鉴权中间件。
// 从 Authorization: Bearer <token> 取 access token，校验后存入 context。
func JWTAuth(tokenSvc auth.TokenService, errs *errcode.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if len(header) <= 7 || !strings.EqualFold(header[:7], "bearer ") {
			response.AbortErr(c, errs.ErrTokenInvalid.Code)
			return
		}
		token := header[7:]

		claims, err := tokenSvc.Verify(c.Request.Context(), token, auth.TokenTypeAccess)
		if err != nil {
			if coded, ok := err.(interface{ GetCode() int }); ok {
				response.AbortErr(c, coded.GetCode())
			} else {
				response.AbortErr(c, errs.ErrTokenInvalid.Code)
			}
			return
		}

		c.Set(CtxKeyClaims, claims)
		c.Next()
	}
}

// GetClaims 从 gin.Context 取 JWT claims。
func GetClaims(c *gin.Context) *auth.Claims {
	v, ok := c.Get(CtxKeyClaims)
	if !ok {
		return nil
	}
	claims, _ := v.(*auth.Claims)
	return claims
}
