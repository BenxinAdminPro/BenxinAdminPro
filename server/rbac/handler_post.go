// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   岗位 HTTP handler — CRUD + 分页
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 19:25:00
// +----------------------------------------------------------------------

package rbac

import (
	"net/http"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/gin-gonic/gin"
)

const (
	PermPostList   = "sys:post:list"
	PermPostCreate = "sys:post:create"
	PermPostUpdate = "sys:post:update"
	PermPostDelete = "sys:post:delete"
)

// PostHandler 岗位 CRUD handler。
type PostHandler struct {
	svc  *PostService
	errs *errcode.Registry
}

// NewPostHandler 创建岗位 handler。
func NewPostHandler(svc *PostService, errs *errcode.Registry) *PostHandler {
	return &PostHandler{svc: svc, errs: errs}
}

// RegisterRoutes 注册岗位路由。
func (h *PostHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/sys/posts", h.List)
	rg.POST("/sys/posts", h.Create)
	rg.PUT("/sys/posts/:id", h.Update)
	rg.DELETE("/sys/posts/:id", h.Delete)
}

func (h *PostHandler) List(c *gin.Context) {
	var q PostListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		respondJSON(c, http.StatusBadRequest, -1, "invalid query", nil)
		return
	}
	result, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, result)
}

func (h *PostHandler) Create(c *gin.Context) {
	var in CreatePostInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondJSON(c, http.StatusBadRequest, -1, "invalid request", nil)
		return
	}
	post, err := h.svc.Create(c.Request.Context(), in)
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, post)
}

func (h *PostHandler) Update(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	var in UpdatePostInput
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

func (h *PostHandler) Delete(c *gin.Context) {
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
