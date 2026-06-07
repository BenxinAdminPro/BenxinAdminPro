// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   角色 HTTP handler — CRUD + 分配菜单
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 21:22:00
// | @updated   2026-06-07 22:50:00
// +----------------------------------------------------------------------

package rbac

import (
	"net/http"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	svc    *RoleService
	errs   *errcode.Registry
	hasher *Hasher
	enc    *ResponseEncoder
}

func NewRoleHandler(svc *RoleService, errs *errcode.Registry, hasher *Hasher) *RoleHandler {
	var enc *ResponseEncoder
	if hasher != nil {
		enc = NewResponseEncoder(hasher)
	}
	return &RoleHandler{svc: svc, errs: errs, hasher: hasher, enc: enc}
}

func (h *RoleHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/sys/roles", RequirePerm("sys:role:list"), h.List)
	rg.POST("/sys/roles", RequirePerm("sys:role:create"), h.Create)
	rg.GET("/sys/roles/:id", RequirePerm("sys:role:list"), h.Get)
	rg.PUT("/sys/roles/:id", RequirePerm("sys:role:update"), h.Update)
	rg.DELETE("/sys/roles/:id", RequirePerm("sys:role:delete"), h.Delete)
	rg.PUT("/sys/roles/:id/menus", RequirePerm("sys:role:assign"), h.AssignMenus)
}

func (h *RoleHandler) List(c *gin.Context) {
	var q RoleListQuery
	c.ShouldBindQuery(&q)
	result, err := h.svc.List(c.Request.Context(), q)
	if err != nil { respondError(c, err); return }
	if h.enc != nil { respondOK(c, h.enc.RoleList(result)) } else { respondOK(c, result) }
}

func (h *RoleHandler) Create(c *gin.Context) {
	var in CreateRoleInput
	if err := c.ShouldBindJSON(&in); err != nil { respondJSON(c, http.StatusBadRequest, -1, "invalid request", nil); return }
	role, err := h.svc.Create(c.Request.Context(), in)
	if err != nil { respondError(c, err); return }
	if h.enc != nil { respondOK(c, h.enc.Role(role)) } else { respondOK(c, role) }
}

func (h *RoleHandler) Get(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil { return }
	role, err := h.svc.Get(c.Request.Context(), id)
	if err != nil { respondError(c, err); return }
	if h.enc != nil { respondOK(c, h.enc.Role(role)) } else { respondOK(c, role) }
}

func (h *RoleHandler) Update(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil { return }
	var in UpdateRoleInput
	if err := c.ShouldBindJSON(&in); err != nil { respondJSON(c, http.StatusBadRequest, -1, "invalid request", nil); return }
	if err := h.svc.Update(c.Request.Context(), id, in); err != nil { respondError(c, err); return }
	respondOK(c, nil)
}

func (h *RoleHandler) Delete(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil { return }
	if err := h.svc.Delete(c.Request.Context(), id); err != nil { respondError(c, err); return }
	respondOK(c, nil)
}

func (h *RoleHandler) AssignMenus(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil { return }
	var req struct {
		MenuIDs []uint64 `json:"menu_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil { respondJSON(c, http.StatusBadRequest, -1, "invalid request", nil); return }
	if err := h.svc.AssignMenus(c.Request.Context(), id, req.MenuIDs); err != nil { respondError(c, err); return }
	respondOK(c, nil)
}

func (h *RoleHandler) parseHID(c *gin.Context, param string) (uint64, error) {
	if h.hasher != nil {
		id, err := h.hasher.Decode(c.Param(param))
		if err != nil {
			respondJSON(c, h.errs.ErrInvalidID.HTTP, h.errs.ErrInvalidID.Code, h.errs.ErrInvalidID.Message, nil)
			return 0, err
		}
		return id, nil
	}
	return parseID(c, param)
}
