// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   部门 HTTP handler — 树查询 + CRUD
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 19:24:00
// +----------------------------------------------------------------------

package rbac

import (
	"net/http"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/gin-gonic/gin"
)

const (
	PermDeptTree   = "sys:dept:tree"
	PermDeptCreate = "sys:dept:create"
	PermDeptUpdate = "sys:dept:update"
	PermDeptDelete = "sys:dept:delete"
)

// DeptHandler 部门 CRUD handler。
type DeptHandler struct {
	svc  *DeptService
	errs *errcode.Registry
}

// NewDeptHandler 创建部门 handler。
func NewDeptHandler(svc *DeptService, errs *errcode.Registry) *DeptHandler {
	return &DeptHandler{svc: svc, errs: errs}
}

// RegisterRoutes 注册部门路由。
func (h *DeptHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/sys/depts/tree", h.Tree)
	rg.POST("/sys/depts", h.Create)
	rg.PUT("/sys/depts/:id", h.Update)
	rg.DELETE("/sys/depts/:id", h.Delete)
}

func (h *DeptHandler) Tree(c *gin.Context) {
	tree, err := h.svc.Tree(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, tree)
}

func (h *DeptHandler) Create(c *gin.Context) {
	var in CreateDeptInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondJSON(c, http.StatusBadRequest, -1, "invalid request", nil)
		return
	}
	dept, err := h.svc.Create(c.Request.Context(), in)
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, dept)
}

func (h *DeptHandler) Update(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	var in UpdateDeptInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondJSON(c, http.StatusBadRequest, -1, "invalid request", nil)
		return
	}
	if err := h.svc.Update(c.Request.Context(), id, in); err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, nil)
}

func (h *DeptHandler) Delete(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, nil)
}
