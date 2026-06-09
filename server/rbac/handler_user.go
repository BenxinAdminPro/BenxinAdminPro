// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   用户 HTTP handler — CRUD + 密码管理 + 启用禁用 + 分配角色
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 19:22:00
// | @updated   2026-06-07 23:46:00
// | @updated   2026-06-08 02:40:00
// | @updated   2026-06-08 13:00:00  T-003d：RegisterRoutes 注入 AuthzEnforcer，路由走真 enforce
// | @updated   2026-06-09 10:40:13  T-003e：入参对外 ID 收口 — dept_id/post_ids/role_ids 收 hashid
// +----------------------------------------------------------------------

package rbac

import (
	"strconv"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/benxin_dev/benxinadminpro-server/response"
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
	svc      *UserService
	errs     *errcode.Registry
	hasher   *Hasher
	enc      *ResponseEncoder
	resolver ScopeResolver // 可选，注入后 List 自动应用数据权限
}

// NewUserHandler 创建用户 handler。
func NewUserHandler(svc *UserService, errs *errcode.Registry, hasher *Hasher) *UserHandler {
	var enc *ResponseEncoder
	if hasher != nil {
		enc = NewResponseEncoder(hasher)
	}
	return &UserHandler{svc: svc, errs: errs, hasher: hasher, enc: enc}
}

// SetScopeResolver 注入数据权限解析器（可选）。
func (h *UserHandler) SetScopeResolver(r ScopeResolver) {
	h.resolver = r
}

// RegisterRoutes 注册用户路由（每条经注入的 authz 真 enforce）。
func (h *UserHandler) RegisterRoutes(rg *gin.RouterGroup, authz *AuthzEnforcer) {
	rg.GET("/sys/users", authz.RequirePerm(PermUserList), h.List)
	rg.POST("/sys/users", authz.RequirePerm(PermUserCreate), h.Create)
	rg.GET("/sys/users/:id", authz.RequirePerm(PermUserGet), h.Get)
	rg.PUT("/sys/users/:id", authz.RequirePerm(PermUserUpdate), h.Update)
	rg.DELETE("/sys/users/:id", authz.RequirePerm(PermUserDelete), h.Delete)
	rg.PUT("/sys/users/:id/password", authz.RequirePerm(PermUserPassword), h.ResetPassword)
	rg.PUT("/sys/users/:id/status", authz.RequirePerm(PermUserStatus), h.SetStatus)
	rg.PUT("/sys/users/:id/roles", authz.RequirePerm(PermUserAssign), h.AssignRoles)
}

// --- 入参请求 DTO（对外 ID 为 hashid 字符串，handler 边界解码为内部 uint64）---

// userListReq 用户列表查询入参。dept_id 为 hashid 过滤。
type userListReq struct {
	Username string `form:"username"`
	Status   *int8  `form:"status"`
	DeptID   string `form:"dept_id"` // hashid，空=不过滤
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

func (r *userListReq) toQuery(h *Hasher) (UserListQuery, error) {
	deptID, err := decodeOptionalID(h, r.DeptID)
	if err != nil {
		return UserListQuery{}, err
	}
	return UserListQuery{
		Username: r.Username,
		Status:   r.Status,
		DeptID:   deptID,
		Page:     r.Page,
		PageSize: r.PageSize,
	}, nil
}

// createUserReq 创建用户入参。dept_id/post_ids 为 hashid。
type createUserReq struct {
	Username string   `json:"username" binding:"required"`
	Password string   `json:"password" binding:"required"`
	Nickname string   `json:"nickname"`
	Avatar   string   `json:"avatar"`
	Email    string   `json:"email"`
	Mobile   string   `json:"mobile"`
	DeptID   string   `json:"dept_id"` // hashid，空=无部门
	Status   int8     `json:"status"`
	Remark   string   `json:"remark"`
	PostIDs  []string `json:"post_ids"` // hashid[]
}

func (r *createUserReq) toInput(h *Hasher) (CreateUserInput, error) {
	deptID, err := decodeOptionalID(h, r.DeptID)
	if err != nil {
		return CreateUserInput{}, err
	}
	postIDs, err := decodeIDSlice(h, r.PostIDs)
	if err != nil {
		return CreateUserInput{}, err
	}
	return CreateUserInput{
		Username: r.Username,
		Password: r.Password,
		Nickname: r.Nickname,
		Avatar:   r.Avatar,
		Email:    r.Email,
		Mobile:   r.Mobile,
		DeptID:   deptID,
		Status:   r.Status,
		Remark:   r.Remark,
		PostIDs:  postIDs,
	}, nil
}

// updateUserReq 更新用户入参。dept_id/post_ids 为 hashid。
type updateUserReq struct {
	Nickname string   `json:"nickname"`
	Avatar   string   `json:"avatar"`
	Email    string   `json:"email"`
	Mobile   string   `json:"mobile"`
	DeptID   string   `json:"dept_id"`  // hashid，空=清空部门
	Remark   string   `json:"remark"`
	PostIDs  []string `json:"post_ids"` // hashid[]，缺省=不变，[]=清空
}

func (r *updateUserReq) toInput(h *Hasher) (UpdateUserInput, error) {
	deptID, err := decodeOptionalID(h, r.DeptID)
	if err != nil {
		return UpdateUserInput{}, err
	}
	postIDs, err := decodeIDSlice(h, r.PostIDs)
	if err != nil {
		return UpdateUserInput{}, err
	}
	return UpdateUserInput{
		Nickname: r.Nickname,
		Avatar:   r.Avatar,
		Email:    r.Email,
		Mobile:   r.Mobile,
		DeptID:   deptID,
		Remark:   r.Remark,
		PostIDs:  postIDs,
	}, nil
}

func (h *UserHandler) List(c *gin.Context) {
	var req userListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadReq(c)
		return
	}
	q, err := req.toQuery(h.hasher)
	if err != nil {
		response.AbortErr(c, h.errs.ErrInvalidID.Code)
		return
	}
	// 注入数据权限（从服务端取，不接受客户端传入）
	if h.resolver != nil {
		claims := GetClaims(c)
		if claims != nil {
			uid, _ := strconv.ParseUint(claims.Subject, 10, 64)
			if uid > 0 {
				scope, _ := h.resolver.Resolve(c.Request.Context(), uid)
				q.Scope = scope
			}
		}
	}
	result, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		response.ErrResp(c, err)
		return
	}
	if h.enc != nil {
		response.OK(c, h.enc.UserList(result))
	} else {
		response.OK(c, result)
	}
}

func (h *UserHandler) Create(c *gin.Context) {
	var req createUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadReq(c)
		return
	}
	in, err := req.toInput(h.hasher)
	if err != nil {
		response.AbortErr(c, h.errs.ErrInvalidID.Code)
		return
	}
	user, err := h.svc.Create(c.Request.Context(), in)
	if err != nil {
		response.ErrResp(c, err)
		return
	}
	if h.enc != nil {
		response.OK(c, h.enc.User(user))
	} else {
		response.OK(c, user)
	}
}

func (h *UserHandler) Get(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil {
		return
	}
	user, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		response.ErrResp(c, err)
		return
	}
	if h.enc != nil {
		response.OK(c, h.enc.User(user))
	} else {
		response.OK(c, user)
	}
}

func (h *UserHandler) Update(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil {
		return
	}
	var req updateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadReq(c)
		return
	}
	in, err := req.toInput(h.hasher)
	if err != nil {
		response.AbortErr(c, h.errs.ErrInvalidID.Code)
		return
	}
	if err := h.svc.Update(c.Request.Context(), id, in); err != nil {
		response.ErrResp(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.ErrResp(c, err)
		return
	}
	response.OK(c, nil)
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
		response.BadReq(c)
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), id, req.Password); err != nil {
		response.ErrResp(c, err)
		return
	}
	response.OK(c, nil)
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
		response.BadReq(c)
		return
	}
	if err := h.svc.SetStatus(c.Request.Context(), id, req.Status); err != nil {
		response.ErrResp(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *UserHandler) AssignRoles(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil {
		return
	}
	var req struct {
		RoleIDs []string `json:"role_ids"` // hashid[]
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadReq(c)
		return
	}
	roleIDs, err := decodeIDSlice(h.hasher, req.RoleIDs)
	if err != nil {
		response.AbortErr(c, h.errs.ErrInvalidID.Code)
		return
	}
	if err := h.svc.AssignRoles(c.Request.Context(), id, roleIDs); err != nil {
		response.ErrResp(c, err)
		return
	}
	response.OK(c, nil)
}

// ---------------------------------------------------------------------------
// 辅助
// ---------------------------------------------------------------------------

func parseID(c *gin.Context, param string) (uint64, error) {
	id, err := strconv.ParseUint(c.Param(param), 10, 64)
	if err != nil {
		response.BadReq(c)
		return 0, err
	}
	return id, nil
}

func (h *UserHandler) parseHID(c *gin.Context, param string) (uint64, error) {
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
