// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   响应 ID 编码器 — 统一将对外 ID 编码为 Hashid
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 22:30:00
// | @updated   2026-06-07 23:42:00
// +----------------------------------------------------------------------

package rbac

import "github.com/gin-gonic/gin"

// ResponseEncoder 将模型中的 uint64 ID 统一编码为 hashid 字符串。
// 所有 handler 的响应必须经此编码器，禁止直接返回裸自增 ID。
type ResponseEncoder struct {
	H *Hasher
}

// NewResponseEncoder 创建响应编码器。
func NewResponseEncoder(h *Hasher) *ResponseEncoder {
	return &ResponseEncoder{H: h}
}

// User 编码用户响应。
func (e *ResponseEncoder) User(u *SysUser) gin.H {
	resp := gin.H{
		"id":         e.H.Encode(u.ID),
		"username":   u.Username,
		"nickname":   u.Nickname,
		"avatar":     u.Avatar,
		"email":      u.Email,
		"mobile":     u.Mobile,
		"status":     u.Status,
		"remark":     u.Remark,
		"created_at": u.CreatedAt,
		"updated_at": u.UpdatedAt,
	}
	if u.DeptID != nil {
		resp["dept_id"] = e.H.Encode(*u.DeptID)
	} else {
		resp["dept_id"] = nil
	}
	if len(u.Posts) > 0 {
		var posts []gin.H
		for i := range u.Posts {
			posts = append(posts, e.Post(&u.Posts[i]))
		}
		resp["posts"] = posts
	}
	return resp
}

// UserList 编码用户列表响应。
func (e *ResponseEncoder) UserList(result *PageResult) gin.H {
	users, ok := result.List.([]SysUser)
	if !ok {
		return e.RawPage(result)
	}
	var encoded []gin.H
	for i := range users {
		encoded = append(encoded, e.User(&users[i]))
	}
	return gin.H{"list": encoded, "total": result.Total, "page": result.Page, "page_size": result.PageSize}
}

// Dept 编码部门响应。
func (e *ResponseEncoder) Dept(d *SysDept) gin.H {
	resp := gin.H{
		"id":         e.H.Encode(d.ID),
		"parent_id":  e.encodeOrZero(d.ParentID),
		"ancestors":  d.Ancestors,
		"name":       d.Name,
		"sort":       d.Sort,
		"leader":     d.Leader,
		"status":     d.Status,
		"created_at": d.CreatedAt,
		"updated_at": d.UpdatedAt,
	}
	if len(d.Children) > 0 {
		var children []gin.H
		for _, c := range d.Children {
			children = append(children, e.Dept(c))
		}
		resp["children"] = children
	}
	return resp
}

// DeptTree 编码部门树。
func (e *ResponseEncoder) DeptTree(tree []*SysDept) []gin.H {
	var result []gin.H
	for _, d := range tree {
		result = append(result, e.Dept(d))
	}
	return result
}

// Post 编码岗位响应。
func (e *ResponseEncoder) Post(p *SysPost) gin.H {
	return gin.H{
		"id":         e.H.Encode(p.ID),
		"code":       p.Code,
		"name":       p.Name,
		"sort":       p.Sort,
		"status":     p.Status,
		"created_at": p.CreatedAt,
		"updated_at": p.UpdatedAt,
	}
}

// PostList 编码岗位列表。
func (e *ResponseEncoder) PostList(result *PageResult) gin.H {
	posts, ok := result.List.([]SysPost)
	if !ok {
		return e.RawPage(result)
	}
	var encoded []gin.H
	for i := range posts {
		encoded = append(encoded, e.Post(&posts[i]))
	}
	return gin.H{"list": encoded, "total": result.Total, "page": result.Page, "page_size": result.PageSize}
}

// Role 编码角色响应。
func (e *ResponseEncoder) Role(r *SysRole) gin.H {
	return gin.H{
		"id":         e.H.Encode(r.ID),
		"code":       r.Code,
		"name":       r.Name,
		"sort":       r.Sort,
		"status":     r.Status,
		"data_scope": r.DataScope,
		"remark":     r.Remark,
		"created_at": r.CreatedAt,
		"updated_at": r.UpdatedAt,
	}
}

// RoleList 编码角色列表。
func (e *ResponseEncoder) RoleList(result *PageResult) gin.H {
	roles, ok := result.List.([]SysRole)
	if !ok {
		return e.RawPage(result)
	}
	var encoded []gin.H
	for i := range roles {
		encoded = append(encoded, e.Role(&roles[i]))
	}
	return gin.H{"list": encoded, "total": result.Total, "page": result.Page, "page_size": result.PageSize}
}

// Menu 编码菜单响应。
func (e *ResponseEncoder) Menu(m *SysMenu) gin.H {
	resp := gin.H{
		"id":         e.H.Encode(m.ID),
		"parent_id":  e.encodeOrZero(m.ParentID),
		"ancestors":  m.Ancestors,
		"menu_type":  m.MenuType,
		"name":       m.Name,
		"perm_code":  m.PermCode,
		"path":       m.Path,
		"component":  m.Component,
		"icon":       m.Icon,
		"sort":       m.Sort,
		"visible":    m.Visible,
		"status":     m.Status,
		"created_at": m.CreatedAt,
		"updated_at": m.UpdatedAt,
	}
	if len(m.Children) > 0 {
		var children []gin.H
		for _, c := range m.Children {
			children = append(children, e.Menu(c))
		}
		resp["children"] = children
	}
	return resp
}

// MenuTree 编码菜单树。
func (e *ResponseEncoder) MenuTree(tree []*SysMenu) []gin.H {
	var result []gin.H
	for _, m := range tree {
		result = append(result, e.Menu(m))
	}
	return result
}

// RawPage 不编码的分页（兜底）。
func (e *ResponseEncoder) RawPage(result *PageResult) gin.H {
	return gin.H{"list": result.List, "total": result.Total, "page": result.Page, "page_size": result.PageSize}
}

func (e *ResponseEncoder) encodeOrZero(id uint64) any {
	if id == 0 {
		return nil
	}
	return e.H.Encode(id)
}
