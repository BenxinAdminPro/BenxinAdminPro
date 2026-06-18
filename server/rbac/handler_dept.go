// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   部门 HTTP handler — 树查询 + CRUD
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 19:24:00
// | @updated   2026-06-07 21:30:00
// | @updated   2026-06-07 22:50:00
// | @updated   2026-06-08 02:40:00
// | @updated   2026-06-08 13:00:00  T-003d：RegisterRoutes 注入 AuthzEnforcer，路由走真 enforce
// | @updated   2026-06-09 10:40:13  T-003e：parent_id 入参收 hashid
// | @updated   2026-06-18 13:50:56  T-017：构造器 hasher==nil fail-fast（panic）+ 删 enc==nil 裸出参退化分支（根除裸内部 ID/ancestors 泄漏面）
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

// NewDeptHandler 构造部门 handler。hasher 为必备依赖：缺失则编码器无法装配，
// 出参将退化为裸 marshal SysDept（泄漏裸内部 id/parent_id/ancestors）→ 故 hasher==nil
// 直接 fail-fast panic（对齐 demo 装配 self-check 与 ConfigService 缺密钥不静默精神），
// 把「静默泄漏」转为「装配即炸」的响亮编程错。
func NewDeptHandler(svc *DeptService, errs *errcode.Registry, hasher *Hasher) *DeptHandler {
	if hasher == nil {
		panic("rbac: NewDeptHandler requires a non-nil hasher (nil would leak raw internal IDs via bare-marshal fallback)")
	}
	return &DeptHandler{svc: svc, errs: errs, hasher: hasher, enc: NewResponseEncoder(hasher)}
}

func (h *DeptHandler) RegisterRoutes(rg *gin.RouterGroup, authz *AuthzEnforcer) {
	rg.GET("/sys/depts/tree", authz.RequirePerm(PermDeptTree), h.Tree)
	rg.POST("/sys/depts", authz.RequirePerm(PermDeptCreate), h.Create)
	rg.PUT("/sys/depts/:id", authz.RequirePerm(PermDeptUpdate), h.Update)
	rg.DELETE("/sys/depts/:id", authz.RequirePerm(PermDeptDelete), h.Delete)
}

func (h *DeptHandler) Tree(c *gin.Context) {
	tree, err := h.svc.Tree(c.Request.Context())
	if err != nil { response.ErrResp(c, err); return }
	response.OK(c, h.enc.DeptTree(tree))
}

// --- 入参请求 DTO（parent_id 为 hashid 字符串，handler 边界解码）---

// createDeptReq 创建部门入参。parent_id 为 hashid（空=挂根）。
type createDeptReq struct {
	ParentID string `json:"parent_id"` // hashid，空=根
	Name     string `json:"name" binding:"required"`
	Sort     int    `json:"sort"`
	Leader   string `json:"leader"`
	Status   int8   `json:"status"`
}

func (r *createDeptReq) toInput(h *Hasher) (CreateDeptInput, error) {
	parentID, err := decodeZeroableID(h, r.ParentID)
	if err != nil {
		return CreateDeptInput{}, err
	}
	return CreateDeptInput{ParentID: parentID, Name: r.Name, Sort: r.Sort, Leader: r.Leader, Status: r.Status}, nil
}

// updateDeptReq 更新部门入参。parent_id 为 hashid 指针：缺省=不移动，空串=移到根。
type updateDeptReq struct {
	ParentID *string `json:"parent_id"`
	Name     string  `json:"name"`
	Sort     int     `json:"sort"`
	Leader   string  `json:"leader"`
	Status   int8    `json:"status"`
}

func (r *updateDeptReq) toInput(h *Hasher) (UpdateDeptInput, error) {
	parentID, err := decodeMovableID(h, r.ParentID)
	if err != nil {
		return UpdateDeptInput{}, err
	}
	return UpdateDeptInput{ParentID: parentID, Name: r.Name, Sort: r.Sort, Leader: r.Leader, Status: r.Status}, nil
}

func (h *DeptHandler) Create(c *gin.Context) {
	var req createDeptReq
	if err := c.ShouldBindJSON(&req); err != nil { response.BadReq(c); return }
	in, err := req.toInput(h.hasher)
	if err != nil { response.AbortErr(c, h.errs.ErrInvalidID.Code); return }
	dept, err := h.svc.Create(c.Request.Context(), in)
	if err != nil { response.ErrResp(c, err); return }
	response.OK(c, h.enc.Dept(dept))
}

func (h *DeptHandler) Update(c *gin.Context) {
	id, err := h.parseHID(c, "id")
	if err != nil { return }
	var req updateDeptReq
	if err := c.ShouldBindJSON(&req); err != nil { response.BadReq(c); return }
	in, err := req.toInput(h.hasher)
	if err != nil { response.AbortErr(c, h.errs.ErrInvalidID.Code); return }
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
