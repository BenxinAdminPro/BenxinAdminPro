// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   登录态权限下发 — 当前用户菜单树 + 权限码集合
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 21:25:00
// | @updated   2026-06-07 22:50:00
// | @updated   2026-06-08 02:40:00
// +----------------------------------------------------------------------

package rbac

import (
	"net/http"
	"strconv"

	"github.com/benxin_dev/benxinadminpro-server/response"
	"github.com/gin-gonic/gin"
)

// AuthInfoHandler 提供登录态权限下发接口。
type AuthInfoHandler struct {
	menuSvc *MenuService
	userSvc *UserService
	enc     *ResponseEncoder
}

// NewAuthInfoHandler 创建权限下发 handler。
func NewAuthInfoHandler(menuSvc *MenuService, userSvc *UserService, hasher *Hasher) *AuthInfoHandler {
	var enc *ResponseEncoder
	if hasher != nil {
		enc = NewResponseEncoder(hasher)
	}
	return &AuthInfoHandler{menuSvc: menuSvc, userSvc: userSvc, enc: enc}
}

// RegisterRoutes 注册权限下发路由（需 JWT 鉴权，无需额外权限码）。
func (h *AuthInfoHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/sys/auth/menus", h.Menus)
	rg.GET("/sys/auth/perms", h.Perms)
}

// Menus 返回当前用户可见菜单树。
func (h *AuthInfoHandler) Menus(c *gin.Context) {
	userID := h.currentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": -1, "message": "unauthorized"})
		return
	}
	tree, err := h.menuSvc.GetUserMenuTree(c.Request.Context(), userID)
	if err != nil {
		response.ErrResp(c, err)
		return
	}
	if h.enc != nil { response.OK(c, h.enc.MenuTree(tree)) } else { response.OK(c, tree) }
}

// Perms 返回当前用户权限码集合。
func (h *AuthInfoHandler) Perms(c *gin.Context) {
	userID := h.currentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": -1, "message": "unauthorized"})
		return
	}
	codes, err := h.menuSvc.GetUserPermCodes(c.Request.Context(), userID)
	if err != nil {
		response.ErrResp(c, err)
		return
	}
	if codes == nil {
		codes = []string{}
	}
	response.OK(c, codes)
}

func (h *AuthInfoHandler) currentUserID(c *gin.Context) uint64 {
	claims := GetClaims(c)
	if claims == nil {
		return 0
	}
	id, _ := strconv.ParseUint(claims.Subject, 10, 64)
	return id
}
