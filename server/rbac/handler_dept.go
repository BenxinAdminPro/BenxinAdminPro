// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   部门 HTTP handler — 树查询 + CRUD
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 19:24:00
// | @updated   2026-06-07 21:30:00
// | @updated   2026-06-07 22:50:00
// | @updated   2026-06-08 02:40:00
// +----------------------------------------------------------------------

package rbac

import (
	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/benxin_dev/benxinadminpro-server/response"
	"github.com/gin-gonic/gin"
)

const (
	PermDeptTree   = "sys:dept:tree"
	PermDeptCreate = "sys:dept:create"
	PermDeptUpdate = "sys:dept:update"
	PermDeptDelete = "sys:dept:delete"
)

type DeptHandler struct {
	svc    *DeptService
	errs   *errcode.Registry
	hasher *Hasher
	enc    *ResponseEncoder
}

func NewDeptHandler(svc *DeptService, errs *errcode.Registry, hasher *Hasher) *DeptHandler {
	var enc *ResponseEncoder
	if hasher != nil {
		enc = NewResponseEncoder(hasher)
	}
	return &DeptHandler{svc: svc, errs: errs, hasher: hasher, enc: enc}
}

func (h *DeptHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/sys/depts/tree", RequirePerm(PermDeptTree), h.Tree)
	rg.POST("/sys/depts", RequirePerm(PermDeptCreate), h.Create)
	rg.PUT("/sys/depts/:id", RequirePerm(PermDeptUpdate), h.Update)
	rg.DELETE("/sys/depts/:id", RequirePerm(PermDeptDelete), h.Delete)
}

func (h *DeptHandler) Tree(c *gin.Context) {
	tree, err := h.svc.Tree(c.Request.Context())
	if err != nil { response.ErrResp(c, err); return }
	if h.enc != nil { response.OK(c, h.enc.DeptTree(tree)) } else { response.OK(c, tree) }
}

func (h *DeptHandler) Create(c *gin.Context) {
	var in CreateDeptInput
	if err := c.ShouldBindJSON(&in); err != nil { response.BadReq(c); return }
	dept, err := h.svc.Create(c.Request.Context(), in)
	if err != nil { response.ErrResp(c, err); return }
	if h.enc != nil { response.OK(c, h.enc.Dept(dept)) } else { response.OK(c, dept) }
}

func (h *DeptHandler) Update(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil { return }
	var in UpdateDeptInput
	if err := c.ShouldBindJSON(&in); err != nil { response.BadReq(c); return }
	if err := h.svc.Update(c.Request.Context(), id, in); err != nil { response.ErrResp(c, err); return }
	response.OK(c, nil)
}

func (h *DeptHandler) Delete(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil { return }
	if err := h.svc.Delete(c.Request.Context(), id); err != nil { response.ErrResp(c, err); return }
	response.OK(c, nil)
}

func (h *DeptHandler) parseHID(c *gin.Context, param string) (uint64, error) {
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
