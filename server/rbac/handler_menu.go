// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   菜单 HTTP handler — 树查询 + CRUD
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 21:24:00
// | @updated   2026-06-07 22:50:00
// | @updated   2026-06-08 02:40:00
// | @updated   2026-06-08 13:00:00  T-003d：RegisterRoutes 注入 AuthzEnforcer，路由走真 enforce
// | @updated   2026-06-09 10:40:13  T-003e：parent_id 入参收 hashid
// +----------------------------------------------------------------------

package rbac

import (
	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/benxin_dev/benxinadminpro-server/response"
	"github.com/gin-gonic/gin"
)

type MenuHandler struct {
	svc    *MenuService
	errs   *errcode.Registry
	hasher *Hasher
	enc    *ResponseEncoder
}

func NewMenuHandler(svc *MenuService, errs *errcode.Registry, hasher *Hasher) *MenuHandler {
	var enc *ResponseEncoder
	if hasher != nil {
		enc = NewResponseEncoder(hasher)
	}
	return &MenuHandler{svc: svc, errs: errs, hasher: hasher, enc: enc}
}

func (h *MenuHandler) RegisterRoutes(rg *gin.RouterGroup, authz *AuthzEnforcer) {
	rg.GET("/sys/menus/tree", authz.RequirePerm("sys:menu:list"), h.Tree)
	rg.POST("/sys/menus", authz.RequirePerm("sys:menu:create"), h.Create)
	rg.PUT("/sys/menus/:id", authz.RequirePerm("sys:menu:update"), h.Update)
	rg.DELETE("/sys/menus/:id", authz.RequirePerm("sys:menu:delete"), h.Delete)
}

func (h *MenuHandler) Tree(c *gin.Context) {
	tree, err := h.svc.Tree(c.Request.Context())
	if err != nil { response.ErrResp(c, err); return }
	if h.enc != nil { response.OK(c, h.enc.MenuTree(tree)) } else { response.OK(c, tree) }
}

// --- 入参请求 DTO（parent_id 为 hashid 字符串，handler 边界解码）---

// createMenuReq 创建菜单入参。parent_id 为 hashid（空=挂根）。
type createMenuReq struct {
	ParentID  string `json:"parent_id"` // hashid，空=根
	MenuType  string `json:"menu_type" binding:"required"`
	Name      string `json:"name" binding:"required"`
	PermCode  string `json:"perm_code"`
	Path      string `json:"path"`
	Component string `json:"component"`
	Icon      string `json:"icon"`
	Sort      int    `json:"sort"`
	Visible   int8   `json:"visible"`
	Status    int8   `json:"status"`
}

func (r *createMenuReq) toInput(h *Hasher) (CreateMenuInput, error) {
	parentID, err := decodeZeroableID(h, r.ParentID)
	if err != nil {
		return CreateMenuInput{}, err
	}
	return CreateMenuInput{
		ParentID: parentID, MenuType: r.MenuType, Name: r.Name, PermCode: r.PermCode,
		Path: r.Path, Component: r.Component, Icon: r.Icon, Sort: r.Sort, Visible: r.Visible, Status: r.Status,
	}, nil
}

// updateMenuReq 更新菜单入参。parent_id 为 hashid 指针：缺省=不移动，空串=移到根。
type updateMenuReq struct {
	ParentID  *string `json:"parent_id"`
	MenuType  string  `json:"menu_type"`
	Name      string  `json:"name"`
	PermCode  string  `json:"perm_code"`
	Path      string  `json:"path"`
	Component string  `json:"component"`
	Icon      string  `json:"icon"`
	Sort      int     `json:"sort"`
	Visible   int8    `json:"visible"`
	Status    int8    `json:"status"`
}

func (r *updateMenuReq) toInput(h *Hasher) (UpdateMenuInput, error) {
	parentID, err := decodeMovableID(h, r.ParentID)
	if err != nil {
		return UpdateMenuInput{}, err
	}
	return UpdateMenuInput{
		ParentID: parentID, MenuType: r.MenuType, Name: r.Name, PermCode: r.PermCode,
		Path: r.Path, Component: r.Component, Icon: r.Icon, Sort: r.Sort, Visible: r.Visible, Status: r.Status,
	}, nil
}

func (h *MenuHandler) Create(c *gin.Context) {
	var req createMenuReq
	if err := c.ShouldBindJSON(&req); err != nil { response.BadReq(c); return }
	in, err := req.toInput(h.hasher)
	if err != nil { response.AbortErr(c, h.errs.ErrInvalidID.Code); return }
	menu, err := h.svc.Create(c.Request.Context(), in)
	if err != nil { response.ErrResp(c, err); return }
	if h.enc != nil { response.OK(c, h.enc.Menu(menu)) } else { response.OK(c, menu) }
}

func (h *MenuHandler) Update(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil { return }
	var req updateMenuReq
	if err := c.ShouldBindJSON(&req); err != nil { response.BadReq(c); return }
	in, err := req.toInput(h.hasher)
	if err != nil { response.AbortErr(c, h.errs.ErrInvalidID.Code); return }
	if err := h.svc.Update(c.Request.Context(), id, in); err != nil { response.ErrResp(c, err); return }
	response.OK(c, nil)
}

func (h *MenuHandler) Delete(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil { return }
	if err := h.svc.Delete(c.Request.Context(), id); err != nil { response.ErrResp(c, err); return }
	response.OK(c, nil)
}

func (h *MenuHandler) parseHID(c *gin.Context, param string) (uint64, error) {
	if h.hasher != nil {
		id, err := h.hasher.Decode(c.Param(param))
		if err != nil {
			response.AbortErr(c, h.errs.ErrInvalidID.Code)
			return 0, err
		}
		return id, nil
	}
	return parseID(c, param)
}
