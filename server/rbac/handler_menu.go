// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   菜单 HTTP handler — 树查询 + CRUD
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 21:24:00
// +----------------------------------------------------------------------

package rbac

import (
	"net/http"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/gin-gonic/gin"
)

type MenuHandler struct {
	svc    *MenuService
	errs   *errcode.Registry
	hasher *Hasher
}

func NewMenuHandler(svc *MenuService, errs *errcode.Registry, hasher *Hasher) *MenuHandler {
	return &MenuHandler{svc: svc, errs: errs, hasher: hasher}
}

func (h *MenuHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/sys/menus/tree", RequirePerm("sys:menu:list"), h.Tree)
	rg.POST("/sys/menus", RequirePerm("sys:menu:create"), h.Create)
	rg.PUT("/sys/menus/:id", RequirePerm("sys:menu:update"), h.Update)
	rg.DELETE("/sys/menus/:id", RequirePerm("sys:menu:delete"), h.Delete)
}

func (h *MenuHandler) Tree(c *gin.Context) {
	tree, err := h.svc.Tree(c.Request.Context())
	if err != nil { respondError(c, err); return }
	respondOK(c, tree)
}

func (h *MenuHandler) Create(c *gin.Context) {
	var in CreateMenuInput
	if err := c.ShouldBindJSON(&in); err != nil { respondJSON(c, http.StatusBadRequest, -1, "invalid request", nil); return }
	menu, err := h.svc.Create(c.Request.Context(), in)
	if err != nil { respondError(c, err); return }
	respondOK(c, menu)
}

func (h *MenuHandler) Update(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil { return }
	var in UpdateMenuInput
	if err := c.ShouldBindJSON(&in); err != nil { respondJSON(c, http.StatusBadRequest, -1, "invalid request", nil); return }
	if err := h.svc.Update(c.Request.Context(), id, in); err != nil { respondError(c, err); return }
	respondOK(c, nil)
}

func (h *MenuHandler) Delete(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil { return }
	if err := h.svc.Delete(c.Request.Context(), id); err != nil { respondError(c, err); return }
	respondOK(c, nil)
}

func (h *MenuHandler) parseHID(c *gin.Context, param string) (uint64, error) {
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
