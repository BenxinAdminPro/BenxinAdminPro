// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   岗位 HTTP handler — CRUD + 分页
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 19:25:00
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
	PermPostList   = "sys:post:list"
	PermPostCreate = "sys:post:create"
	PermPostUpdate = "sys:post:update"
	PermPostDelete = "sys:post:delete"
)

type PostHandler struct {
	svc    *PostService
	errs   *errcode.Registry
	hasher *Hasher
	enc    *ResponseEncoder
}

func NewPostHandler(svc *PostService, errs *errcode.Registry, hasher *Hasher) *PostHandler {
	var enc *ResponseEncoder
	if hasher != nil {
		enc = NewResponseEncoder(hasher)
	}
	return &PostHandler{svc: svc, errs: errs, hasher: hasher, enc: enc}
}

func (h *PostHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/sys/posts", RequirePerm(PermPostList), h.List)
	rg.POST("/sys/posts", RequirePerm(PermPostCreate), h.Create)
	rg.PUT("/sys/posts/:id", RequirePerm(PermPostUpdate), h.Update)
	rg.DELETE("/sys/posts/:id", RequirePerm(PermPostDelete), h.Delete)
}

func (h *PostHandler) List(c *gin.Context) {
	var q PostListQuery
	c.ShouldBindQuery(&q)
	result, err := h.svc.List(c.Request.Context(), q)
	if err != nil { response.ErrResp(c, err); return }
	if h.enc != nil { response.OK(c, h.enc.PostList(result)) } else { response.OK(c, result) }
}

func (h *PostHandler) Create(c *gin.Context) {
	var in CreatePostInput
	if err := c.ShouldBindJSON(&in); err != nil { response.BadReq(c); return }
	post, err := h.svc.Create(c.Request.Context(), in)
	if err != nil { response.ErrResp(c, err); return }
	if h.enc != nil { response.OK(c, h.enc.Post(post)) } else { response.OK(c, post) }
}

func (h *PostHandler) Update(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil { return }
	var in UpdatePostInput
	if err := c.ShouldBindJSON(&in); err != nil { response.BadReq(c); return }
	if err := h.svc.Update(c.Request.Context(), id, in); err != nil { response.ErrResp(c, err); return }
	response.OK(c, nil)
}

func (h *PostHandler) Delete(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil { return }
	if err := h.svc.Delete(c.Request.Context(), id); err != nil { response.ErrResp(c, err); return }
	response.OK(c, nil)
}

func (h *PostHandler) parseHID(c *gin.Context, param string) (uint64, error) {
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
