// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-003e 入参 hashid 收口集成测试 — 经 HTTP handler 以 hashid 建用户落库 + 裸 uint64 被拒
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-09 10:40:13
// +----------------------------------------------------------------------
//
// 运行方式：go test -tags=integration ./rbac/... -v -count=1 -run TestDecodeHashidInput
// 前置：docker compose -f deploy/docker-compose.dev.yml up -d

//go:build integration

package rbac

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/auth"
	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/gin-gonic/gin"
)

// postJSON 直接调 handler（绕过 authz 中间件，本测试聚焦入参解码与落库）。
func postJSON(h *UserHandler, method, target, body string, params gin.Params) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = params
	c.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Create(c)
	return w
}

func TestDecodeHashidInput_MySQL(t *testing.T) {
	db := setupOrgMySQL(t)
	reg, _ := errcode.NewRegistry(11000)
	pwHasher, _ := auth.NewPasswordHasher(auth.Argon2idParams{
		MemoryKiB: 1024, Iterations: 1, Parallelism: 1, SaltLen: 16, KeyLen: 32,
	})
	hashidHasher, _ := NewHasher(HashidConfig{Salt: "t-003e-int", MinLength: 8})
	svc := NewUserService(db, pwHasher, reg)
	h := NewUserHandler(svc, reg, hashidHasher)

	// 预置一个部门与岗位，取其内部自增 id
	dept := SysDept{Name: "技术部", Ancestors: "0"}
	if err := db.Create(&dept).Error; err != nil {
		t.Fatalf("create dept: %v", err)
	}
	post := SysPost{Code: "dev", Name: "开发"}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}

	deptHID := hashidHasher.Encode(dept.ID)
	postHID := hashidHasher.Encode(post.ID)

	// 1) 以 hashid 入参建用户 → 200，落库内部 id 正确
	body := `{"username":"hid_user","password":"pwd123","dept_id":"` + deptHID + `","post_ids":["` + postHID + `"]}`
	w := postJSON(h, "POST", "/sys/users", body, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("以 hashid 建用户应 200，got %d (%s)", w.Code, w.Body.String())
	}

	var created SysUser
	if err := db.Where("username = ?", "hid_user").First(&created).Error; err != nil {
		t.Fatalf("查落库用户: %v", err)
	}
	if created.DeptID == nil || *created.DeptID != dept.ID {
		t.Errorf("落库 dept_id 应为内部 %d，got %v", dept.ID, created.DeptID)
	}
	var upCount int64
	db.Model(&SysUserPost{}).Where("user_id = ? AND post_id = ?", created.ID, post.ID).Count(&upCount)
	if upCount != 1 {
		t.Errorf("user_post 关联应落库内部 post_id=%d，count=%d", post.ID, upCount)
	}

	// 2) 裸 uint64 入参（JSON 数字 dept_id）→ 绑定失败 400，不落库
	rawBody := `{"username":"raw_user","password":"pwd123","dept_id":` + itoa(dept.ID) + `}`
	w = postJSON(h, "POST", "/sys/users", rawBody, nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("裸 uint64 dept_id 应被 400 拒，got %d (%s)", w.Code, w.Body.String())
	}
	var rawCount int64
	db.Model(&SysUser{}).Where("username = ?", "raw_user").Count(&rawCount)
	if rawCount != 0 {
		t.Errorf("被拒的裸 uint64 入参不应落库，count=%d", rawCount)
	}

	// 3) 伪造/非法 hashid → 400，不落库（不泄漏内部 id 是否存在）
	forgedBody := `{"username":"forged_user","password":"pwd123","dept_id":"!!!forged!!!"}`
	w = postJSON(h, "POST", "/sys/users", forgedBody, nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("伪造 hashid 应 400，got %d", w.Code)
	}
}

// TestUpdateUserPostIDsThreeState_MySQL 端到端验证 post_ids 三态经 HTTP Update → DB：
// 缺省=不动岗位 / []=清空 / [...]=替换。这是 nil 切片 vs 空非 nil 切片脚枪的真后端兜底。
func TestUpdateUserPostIDsThreeState_MySQL(t *testing.T) {
	db := setupOrgMySQL(t)
	reg, _ := errcode.NewRegistry(11000)
	pwHasher, _ := auth.NewPasswordHasher(auth.Argon2idParams{
		MemoryKiB: 1024, Iterations: 1, Parallelism: 1, SaltLen: 16, KeyLen: 32,
	})
	hh, _ := NewHasher(HashidConfig{Salt: "t-003e-3state", MinLength: 8})
	svc := NewUserService(db, pwHasher, reg)
	h := NewUserHandler(svc, reg, hh)

	p1 := SysPost{Code: "p1", Name: "岗一"}
	p2 := SysPost{Code: "p2", Name: "岗二"}
	db.Create(&p1)
	db.Create(&p2)

	// 建用户，初始带 p1
	created, err := svc.Create(context.Background(), CreateUserInput{
		Username: "three_state", Password: "pwd123", PostIDs: []uint64{p1.ID},
	})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	uidHID := hh.Encode(created.ID)
	postCount := func() int64 {
		var n int64
		db.Model(&SysUserPost{}).Where("user_id = ?", created.ID).Count(&n)
		return n
	}
	if postCount() != 1 {
		t.Fatalf("前置：初始应 1 个岗位，got %d", postCount())
	}

	doUpdate := func(body string) *httptest.ResponseRecorder {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "id", Value: uidHID}}
		c.Request = httptest.NewRequest("PUT", "/sys/users/x", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		h.Update(c)
		return w
	}

	// 态①：缺省 post_ids（只改昵称）→ 岗位不动，仍 1 个
	if w := doUpdate(`{"nickname":"改名但不动岗位"}`); w.Code != http.StatusOK {
		t.Fatalf("态①update 应 200，got %d (%s)", w.Code, w.Body.String())
	}
	if postCount() != 1 {
		t.Errorf("态①缺省 post_ids 应保留岗位(1)，got %d —— nil 被错当清空", postCount())
	}

	// 态②：显式空数组 → 清空，0 个
	if w := doUpdate(`{"nickname":"清空岗位","post_ids":[]}`); w.Code != http.StatusOK {
		t.Fatalf("态②update 应 200，got %d", w.Code)
	}
	if postCount() != 0 {
		t.Errorf("态②空数组 post_ids 应清空岗位(0)，got %d", postCount())
	}

	// 态③：有值 → 替换为 p2，1 个且为 p2
	if w := doUpdate(`{"nickname":"换岗","post_ids":["` + hh.Encode(p2.ID) + `"]}`); w.Code != http.StatusOK {
		t.Fatalf("态③update 应 200，got %d", w.Code)
	}
	var got int64
	db.Model(&SysUserPost{}).Where("user_id = ? AND post_id = ?", created.ID, p2.ID).Count(&got)
	if postCount() != 1 || got != 1 {
		t.Errorf("态③应替换为 p2(1)，总数=%d p2=%d", postCount(), got)
	}
}

// itoa 本测试用最小整数转字符串（避免引入 strconv 仅此一处）。
func itoa(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
