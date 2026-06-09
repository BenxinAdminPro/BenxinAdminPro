// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   响应 Hashid 编码断言测试 — 确认无裸自增整数泄漏
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 23:05:00
// +----------------------------------------------------------------------

package rbac

import (
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/benxin_dev/benxinadminpro-server/idcodec"
	"github.com/gin-gonic/gin"
)

func testEncoder(t *testing.T) *ResponseEncoder {
	t.Helper()
	h, err := idcodec.NewHasher(idcodec.HashidConfig{Salt: "test-response-salt", MinLength: 8})
	if err != nil {
		t.Fatalf("NewHasher: %v", err)
	}
	return NewResponseEncoder(h)
}

// bareIntID 匹配 JSON 中 "id": 数字 或 "dept_id": 数字 等裸整数 ID 模式
var bareIntID = regexp.MustCompile(`"(?:id|dept_id|parent_id|user_id|post_id|role_id)":\s*\d+`)

func assertNoBareIntID(t *testing.T, data []byte) {
	t.Helper()
	if bareIntID.Match(data) {
		t.Errorf("response contains bare integer ID:\n%s", data)
	}
}

func TestResponseEncoder_User(t *testing.T) {
	enc := testEncoder(t)
	deptID := uint64(5)
	user := &SysUser{
		ID: 42, Username: "alice", Nickname: "Alice", DeptID: &deptID,
		Status: 0, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		Posts: []SysPost{{ID: 10, Code: "CEO", Name: "总裁"}},
	}

	resp := enc.User(user)
	data, _ := json.Marshal(resp)

	assertNoBareIntID(t, data)

	// id 应为 hashid 字符串
	id, ok := resp["id"].(string)
	if !ok || id == "" {
		t.Error("user id should be a non-empty hashid string")
	}
	// dept_id 应为 hashid 字符串
	did, ok := resp["dept_id"].(string)
	if !ok || did == "" {
		t.Error("user dept_id should be a non-empty hashid string")
	}
	// posts[0].id 应为 hashid 字符串
	posts := resp["posts"].([]gin.H)
	pid, ok := posts[0]["id"].(string)
	if !ok || pid == "" {
		t.Error("post id in user response should be hashid string")
	}
}

func TestResponseEncoder_UserList(t *testing.T) {
	enc := testEncoder(t)
	result := &PageResult{
		List:     []SysUser{{ID: 1, Username: "a"}, {ID: 2, Username: "b"}},
		Total:    2,
		Page:     1,
		PageSize: 10,
	}

	resp := enc.UserList(result)
	data, _ := json.Marshal(resp)
	assertNoBareIntID(t, data)
}

func TestResponseEncoder_Dept(t *testing.T) {
	enc := testEncoder(t)
	dept := &SysDept{ID: 3, ParentID: 1, Name: "技术部", Ancestors: "0,1"}

	resp := enc.Dept(dept)
	data, _ := json.Marshal(resp)
	assertNoBareIntID(t, data)

	id, ok := resp["id"].(string)
	if !ok || id == "" {
		t.Error("dept id should be hashid string")
	}
	pid, ok := resp["parent_id"].(string)
	if !ok || pid == "" {
		t.Error("dept parent_id should be hashid string")
	}
}

func TestResponseEncoder_DeptTree(t *testing.T) {
	enc := testEncoder(t)
	tree := []*SysDept{
		{ID: 1, Name: "根", Children: []*SysDept{{ID: 2, ParentID: 1, Name: "子"}}},
	}

	resp := enc.DeptTree(tree)
	data, _ := json.Marshal(resp)
	assertNoBareIntID(t, data)
}

func TestResponseEncoder_Role(t *testing.T) {
	enc := testEncoder(t)
	role := &SysRole{ID: 7, Code: "admin", Name: "管理员"}

	resp := enc.Role(role)
	data, _ := json.Marshal(resp)
	assertNoBareIntID(t, data)
}

func TestResponseEncoder_Menu(t *testing.T) {
	enc := testEncoder(t)
	menu := &SysMenu{ID: 15, ParentID: 3, MenuType: "F", Name: "查看", PermCode: "sys:user:list"}

	resp := enc.Menu(menu)
	data, _ := json.Marshal(resp)
	assertNoBareIntID(t, data)
}

func TestResponseEncoder_PostList(t *testing.T) {
	enc := testEncoder(t)
	result := &PageResult{
		List:     []SysPost{{ID: 1, Code: "CEO"}, {ID: 2, Code: "CTO"}},
		Total:    2,
		Page:     1,
		PageSize: 10,
	}

	resp := enc.PostList(result)
	data, _ := json.Marshal(resp)
	assertNoBareIntID(t, data)
}
