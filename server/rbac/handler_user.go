// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   用户 HTTP handler — CRUD + 密码管理 + 启用禁用 + 分配角色
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 19:22:00
// | @updated   2026-06-07 21:28:00
// +----------------------------------------------------------------------

package rbac

import (
	"net/http"
	"strconv"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/gin-gonic/gin"
)

// 权限码常量（统一命名 模块:资源:动作）。
const (
	PermUserList     = "sys:user:list"
	PermUserCreate   = "sys:user:create"
	PermUserGet      = "sys:user:get"
	PermUserUpdate   = "sys:user:update"
	PermUserDelete   = "sys:user:delete"
	PermUserPassword = "sys:user:password"
	PermUserStatus   = "sys:user:status"
	PermUserAssign   = "sys:user:assign"
)

// UserHandler 用户 CRUD handler。
type UserHandler struct {
	svc    *UserService
	errs   *errcode.Registry
	hasher *Hasher
}

// NewUserHandler 创建用户 handler。
func NewUserHandler(svc *UserService, errs *errcode.Registry, hasher *Hasher) *UserHandler {
	return &UserHandler{svc: svc, errs: errs, hasher: hasher}
}

// RegisterRoutes 注册用户路由（全部挂 RequirePerm）。
func (h *UserHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/sys/users", RequirePerm(PermUserList), h.List)
	rg.POST("/sys/users", RequirePerm(PermUserCreate), h.Create)
	rg.GET("/sys/users/:id", RequirePerm(PermUserGet), h.Get)
	rg.PUT("/sys/users/:id", RequirePerm(PermUserUpdate), h.Update)
	rg.DELETE("/sys/users/:id", RequirePerm(PermUserDelete), h.Delete)
	rg.PUT("/sys/users/:id/password", RequirePerm(PermUserPassword), h.ResetPassword)
	rg.PUT("/sys/users/:id/status", RequirePerm(PermUserStatus), h.SetStatus)
	rg.PUT("/sys/users/:id/roles", RequirePerm(PermUserAssign), h.AssignRoles)
}

func (h *UserHandler) List(c *gin.Context) {
	var q UserListQuery
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

func (h *UserHandler) Create(c *gin.Context) {
	var in CreateUserInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondJSON(c, http.StatusBadRequest, -1, "invalid request", nil)
		return
	}
	user, err := h.svc.Create(c.Request.Context(), in)
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, user)
}

func (h *UserHandler) Get(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil {
		return
	}
	user, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, user)
}

func (h *UserHandler) Update(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil {
		return
	}
	var in UpdateUserInput
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

func (h *UserHandler) Delete(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, nil)
}

func (h *UserHandler) ResetPassword(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil {
		return
	}
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondJSON(c, http.StatusBadRequest, -1, "invalid request", nil)
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), id, req.Password); err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, nil)
}

func (h *UserHandler) SetStatus(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil {
		return
	}
	var req struct {
		Status int8 `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondJSON(c, http.StatusBadRequest, -1, "invalid request", nil)
		return
	}
	if err := h.svc.SetStatus(c.Request.Context(), id, req.Status); err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, nil)
}

func (h *UserHandler) AssignRoles(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil {
		return
	}
	var req struct {
		RoleIDs []uint64 `json:"role_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondJSON(c, http.StatusBadRequest, -1, "invalid request", nil)
		return
	}
	if err := h.svc.AssignRoles(c.Request.Context(), id, req.RoleIDs); err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, nil)
}

// ---------------------------------------------------------------------------
// 响应辅助（包内共用）
// ---------------------------------------------------------------------------

func respondOK(c *gin.Context, data any) {
	respondJSON(c, http.StatusOK, 0, "ok", data)
}

func respondJSON(c *gin.Context, httpStatus, code int, message string, data any) {
	c.JSON(httpStatus, gin.H{"code": code, "message": message, "data": data})
}

func respondError(c *gin.Context, err error) {
	if ecErr, ok := err.(errcode.Error); ok {
		respondJSON(c, ecErr.HTTP, ecErr.Code, ecErr.Message, nil)
		return
	}
	respondJSON(c, http.StatusInternalServerError, -1, "internal error", nil)
}

func parseID(c *gin.Context, param string) (uint64, error) {
	id, err := strconv.ParseUint(c.Param(param), 10, 64)
	if err != nil {
		respondJSON(c, http.StatusBadRequest, -1, "invalid id", nil)
		return 0, err
	}
	return id, nil
}

func (h *UserHandler) parseHID(c *gin.Context, param string) (uint64, error) {
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
