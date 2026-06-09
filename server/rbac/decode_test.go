// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   入参对外 ID 解码测试 — 合法/非法/空/越界/被改一位 + handler 400 接线
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-09 10:40:13
// +----------------------------------------------------------------------

package rbac

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/benxin_dev/benxinadminpro-server/idcodec"
	"github.com/gin-gonic/gin"
)

func newTestHasher(t *testing.T) *Hasher {
	t.Helper()
	h, err := idcodec.NewHasher(idcodec.HashidConfig{Salt: "t-003e-salt", MinLength: 8})
	if err != nil {
		t.Fatalf("NewHasher: %v", err)
	}
	return h
}

// --- decodeID：合法/非法/nil hasher 退化 ---

func TestDecodeID(t *testing.T) {
	h := newTestHasher(t)
	enc := h.Encode(42)

	got, err := decodeID(h, enc)
	if err != nil || got != 42 {
		t.Fatalf("decodeID(valid) = %d, %v; want 42, nil", got, err)
	}

	// 非法：含字母表外字符
	if _, err := decodeID(h, "!!!bad!!!"); err == nil {
		t.Error("decodeID(非法) 应 error")
	}

	// 被改一位：翻转末字符后不得仍解出原值 42（要么 error，要么不同值）
	tampered := enc[:len(enc)-1] + string(flipByte(enc[len(enc)-1]))
	if got, err := decodeID(h, tampered); err == nil && got == 42 {
		t.Errorf("被改一位的 hashid 仍解出原值 42: %q", tampered)
	}

	// nil hasher：退化为裸十进制
	if got, err := decodeID(nil, "7"); err != nil || got != 7 {
		t.Errorf("decodeID(nil,\"7\") = %d, %v; want 7, nil", got, err)
	}
	if _, err := decodeID(nil, "abc"); err == nil {
		t.Error("decodeID(nil, 非数字) 应 error")
	}
}

// --- decodeOptionalID：空→nil / 合法→ptr / 非法→error ---

func TestDecodeOptionalID(t *testing.T) {
	h := newTestHasher(t)
	if p, err := decodeOptionalID(h, ""); err != nil || p != nil {
		t.Errorf("空串应 → nil, nil; got %v, %v", p, err)
	}
	p, err := decodeOptionalID(h, h.Encode(9))
	if err != nil || p == nil || *p != 9 {
		t.Errorf("合法应 → &9; got %v, %v", p, err)
	}
	if _, err := decodeOptionalID(h, "###"); err == nil {
		t.Error("非法应 error")
	}
}

// --- decodeZeroableID：空→0 / 合法→id / 非法→error ---

func TestDecodeZeroableID(t *testing.T) {
	h := newTestHasher(t)
	if id, err := decodeZeroableID(h, ""); err != nil || id != 0 {
		t.Errorf("空串应 → 0; got %d, %v", id, err)
	}
	if id, err := decodeZeroableID(h, h.Encode(5)); err != nil || id != 5 {
		t.Errorf("合法应 → 5; got %d, %v", id, err)
	}
	if _, err := decodeZeroableID(h, "@@@"); err == nil {
		t.Error("非法应 error")
	}
}

// --- decodeMovableID：三态 nil/空串/合法 + 非法 ---

func TestDecodeMovableID(t *testing.T) {
	h := newTestHasher(t)
	if p, err := decodeMovableID(h, nil); err != nil || p != nil {
		t.Errorf("nil 应 → nil(不移动); got %v, %v", p, err)
	}
	empty := ""
	if p, err := decodeMovableID(h, &empty); err != nil || p == nil || *p != 0 {
		t.Errorf("空串应 → &0(移到根); got %v, %v", p, err)
	}
	enc := h.Encode(8)
	if p, err := decodeMovableID(h, &enc); err != nil || p == nil || *p != 8 {
		t.Errorf("合法应 → &8; got %v, %v", p, err)
	}
	bad := "%%%"
	if _, err := decodeMovableID(h, &bad); err == nil {
		t.Error("非法应 error")
	}
}

// --- decodeIDSlice：nil→nil / 空非nil→空非nil / 合法 / 任一非法→error ---

func TestDecodeIDSlice(t *testing.T) {
	h := newTestHasher(t)

	// nil → nil（保留“不变”语义）
	if got, err := decodeIDSlice(h, nil); err != nil || got != nil {
		t.Errorf("nil 应 → nil; got %v, %v", got, err)
	}

	// 空非 nil → 空非 nil（保留“清空”语义）
	got, err := decodeIDSlice(h, []string{})
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("[] 应 → 非 nil 空切片; got %v(nil=%v), %v", got, got == nil, err)
	}

	// 合法多元素
	got, err = decodeIDSlice(h, []string{h.Encode(1), h.Encode(2), h.Encode(3)})
	if err != nil || len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Errorf("合法列表应 → [1 2 3]; got %v, %v", got, err)
	}

	// 任一非法 → 整体 error
	if _, err := decodeIDSlice(h, []string{h.Encode(1), "!!!"}); err == nil {
		t.Error("含非法元素应整体 error")
	}
}

// --- toInput：合法 hashid 正确映射到内部 uint64 ---

func TestCreateUserReqToInput(t *testing.T) {
	h := newTestHasher(t)
	r := &createUserReq{
		Username: "u", Password: "p",
		DeptID:  h.Encode(10),
		PostIDs: []string{h.Encode(20), h.Encode(21)},
	}
	in, err := r.toInput(h)
	if err != nil {
		t.Fatalf("toInput: %v", err)
	}
	if in.DeptID == nil || *in.DeptID != 10 {
		t.Errorf("DeptID 应解为 10; got %v", in.DeptID)
	}
	if len(in.PostIDs) != 2 || in.PostIDs[0] != 20 || in.PostIDs[1] != 21 {
		t.Errorf("PostIDs 应解为 [20 21]; got %v", in.PostIDs)
	}
}

// --- post_ids 三态独立断言：通过 updateUserReq.toInput（nil=不变 / []=清空 / [...]=替换）---
// 这是 Go JSON 绑定 nil 切片 vs 空非 nil 切片的经典脚枪：service 层 `if in.PostIDs != nil` 据此区分
// “不动岗位”与“清空岗位”，故必须断言三态各自正确穿过 toInput，而非只测 decodeIDSlice 本身。

func TestUpdateUserReqPostIDsThreeState(t *testing.T) {
	h := newTestHasher(t)

	// 态①：缺省（JSON 不含 post_ids）→ r.PostIDs 为 nil → in.PostIDs 必须 nil（service 不动岗位）
	var rNil updateUserReq // PostIDs 零值即 nil
	inNil, err := rNil.toInput(h)
	if err != nil {
		t.Fatalf("toInput(nil): %v", err)
	}
	if inNil.PostIDs != nil {
		t.Errorf("态①缺省 post_ids 应 → nil(不变)，got %#v (nil=%v)", inNil.PostIDs, inNil.PostIDs == nil)
	}

	// 态②：显式空数组 []  → r.PostIDs 为非 nil 空切片 → in.PostIDs 必须非 nil 且 len 0（service 清空）
	rEmpty := updateUserReq{PostIDs: []string{}}
	inEmpty, err := rEmpty.toInput(h)
	if err != nil {
		t.Fatalf("toInput([]): %v", err)
	}
	if inEmpty.PostIDs == nil || len(inEmpty.PostIDs) != 0 {
		t.Errorf("态②空数组 post_ids 应 → 非 nil 空切片(清空)，got %#v (nil=%v)", inEmpty.PostIDs, inEmpty.PostIDs == nil)
	}

	// 态③：有值 → 解码替换
	rFull := updateUserReq{PostIDs: []string{h.Encode(31), h.Encode(32)}}
	inFull, err := rFull.toInput(h)
	if err != nil {
		t.Fatalf("toInput([...]): %v", err)
	}
	if len(inFull.PostIDs) != 2 || inFull.PostIDs[0] != 31 || inFull.PostIDs[1] != 32 {
		t.Errorf("态③有值 post_ids 应 → [31 32](替换)，got %v", inFull.PostIDs)
	}
}

// 对端验证：JSON 反序列化确实产生 nil vs 非 nil 空切片的区别（脚枪本身成立才有意义）。
func TestUpdateUserReqJSONNilVsEmpty(t *testing.T) {
	var rAbsent updateUserReq
	if err := json.Unmarshal([]byte(`{"nickname":"x"}`), &rAbsent); err != nil {
		t.Fatal(err)
	}
	if rAbsent.PostIDs != nil {
		t.Errorf("JSON 缺省 post_ids 反序列化应为 nil 切片，got %#v", rAbsent.PostIDs)
	}
	var rEmpty updateUserReq
	if err := json.Unmarshal([]byte(`{"post_ids":[]}`), &rEmpty); err != nil {
		t.Fatal(err)
	}
	if rEmpty.PostIDs == nil {
		t.Errorf("JSON 空数组 post_ids 反序列化应为非 nil 空切片，got nil")
	}
}

// --- handler 接线：body 内非法 hashid → 400（ErrInvalidID），不触达 svc ---

func TestUserCreateRejectsBadHashid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg, _ := errcode.NewRegistry(11000)
	h := NewUserHandler(nil, reg, newTestHasher(t)) // svc nil：非法 hashid 在 decode 即 400，不触达 svc

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/sys/users",
		strings.NewReader(`{"username":"x","password":"y","dept_id":"!!!notvalid!!!"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("非法 dept_id hashid 应 400，got %d (%s)", w.Code, w.Body.String())
	}
}

func TestUserCreateRejectsBadPostID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg, _ := errcode.NewRegistry(11000)
	h := NewUserHandler(nil, reg, newTestHasher(t))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/sys/users",
		strings.NewReader(`{"username":"x","password":"y","post_ids":["@@@bad@@@"]}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("非法 post_ids hashid 应 400，got %d", w.Code)
	}
}

func TestRoleAssignMenusRejectsBadHashid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg, _ := errcode.NewRegistry(11000)
	hasher := newTestHasher(t)
	h := NewRoleHandler(nil, reg, hasher)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: hasher.Encode(1)}} // 路径 id 合法，body menu_ids 非法
	c.Request = httptest.NewRequest("PUT", "/sys/roles/x/menus",
		strings.NewReader(`{"menu_ids":["bad!!!"]}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.AssignMenus(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("非法 menu_ids hashid 应 400，got %d", w.Code)
	}
}

// flipByte 把字符在 hashid 默认字母表内换成另一个，制造“被改一位”样本。
func flipByte(b byte) byte {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	for i := range len(alphabet) {
		if alphabet[i] == b {
			return alphabet[(i+1)%len(alphabet)]
		}
	}
	return alphabet[0]
}
