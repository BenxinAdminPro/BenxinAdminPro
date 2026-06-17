// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   system 出参编码 + path :id 解码测试 — hashid 化/字段保留/脱敏保留/非法 400
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-09 16:09:17
// +----------------------------------------------------------------------

package system

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/idcodec"
	"github.com/gin-gonic/gin"
)

func newHasher(t *testing.T) *idcodec.Hasher {
	t.Helper()
	h, err := idcodec.NewHasher(idcodec.HashidConfig{Salt: "t-004d-sys", MinLength: 8})
	if err != nil {
		t.Fatalf("NewHasher: %v", err)
	}
	return h
}

// 匹配 JSON 中 "id": 裸数字 —— 出参不应再出现。
var bareIntID = regexp.MustCompile(`"id"\s*:\s*\d`)

// --- 出参编码：id → hashid，其余字段保留 ---

func TestResponseEncoderItemEncodesID(t *testing.T) {
	h := newHasher(t)
	enc := NewResponseEncoder(h)

	dt := SysDictType{ID: 42, DictType: "sys_status", Name: "状态", Status: 1, Remark: "r"}
	m := enc.Item(&dt)

	if got, ok := m["id"].(string); !ok || got != h.Encode(42) {
		t.Errorf("id 应为 hashid 字符串 %q，got %#v", h.Encode(42), m["id"])
	}
	// 其余字段保留
	if m["dict_type"] != "sys_status" || m["name"] != "状态" {
		t.Errorf("非 ID 字段应原样保留，got %#v", m)
	}
	// round-trip：hashid 能解回 42
	if id, err := h.Decode(m["id"].(string)); err != nil || id != 42 {
		t.Errorf("hashid 应解回 42，got %d err=%v", id, err)
	}
}

func TestResponseEncoderNilHasherKeepsRawID(t *testing.T) {
	enc := NewResponseEncoder(nil) // 退化
	m := enc.Item(&SysDictType{ID: 7, Name: "x"})
	// nil hasher：id 保留原始数字（json 反序列化为 float64），不编码
	if _, isStr := m["id"].(string); isStr {
		t.Errorf("nil hasher 不应把 id 编码为字符串，got %#v", m["id"])
	}
}

// 脱敏不被破坏：service 层已对 is_encrypted=1 脱敏，encoder 仅改 id 不碰 value。
func TestResponseEncoderPreservesMaskedValue(t *testing.T) {
	h := newHasher(t)
	enc := NewResponseEncoder(h)
	cfg := SysConfig{ID: 9, ConfigKey: "k", ConfigValue: "******", IsEncrypted: 1}
	m := enc.Item(&cfg)
	if m["config_value"] != "******" {
		t.Errorf("脱敏值应原样保留，got %#v", m["config_value"])
	}
	if _, isStr := m["id"].(string); !isStr {
		t.Errorf("id 应被编码为 hashid 字符串")
	}
}

// T-013 收口负向断言：SysFile 出参不应含 storage_key（相对存储 key 收口，json tag 改 "-"）。
// fixture 设了 StorageKey，若 json tag 被改回 "storage_key" 此断言会 FAIL —— 本该早拦住泄漏。
func TestResponseEncoderFileNoStorageKey(t *testing.T) {
	h := newHasher(t)
	enc := NewResponseEncoder(h)
	f := SysFile{ID: 5, OriginalName: "a.png", StorageKey: "2026/06/17/uuid.png", Mime: "image/png", Uploader: "1", UploaderName: "admin"}
	m := enc.Item(&f)
	if _, leaked := m["storage_key"]; leaked {
		t.Errorf("file 出参不应含 storage_key（收口，T-013），got %#v", m)
	}
	// T-014 双向锁：raw uploader 已收口、但展示名 uploader_name 仍在（防误删展示字段）。
	// fixture 设了 Uploader:"1"，若 json tag 被改回 "uploader" 此断言会 FAIL。
	if _, leaked := m["uploader"]; leaked {
		t.Errorf("file 出参不应含 uploader（内部用户 ID 串收口，T-014），got %#v", m)
	}
	if m["uploader_name"] != "admin" {
		t.Errorf("uploader_name 展示路径应保留（误删即 FAIL），got %#v", m["uploader_name"])
	}
	// 零回归：其余字段仍在、id 仍 hashid。
	if m["original_name"] != "a.png" {
		t.Errorf("非收口字段应原样保留，got %#v", m)
	}
	if _, isStr := m["id"].(string); !isStr {
		t.Error("id 应被编码为 hashid 字符串")
	}
}

// T-014 收口负向断言（双向锁）：SysOperLog 出参不应含 operator（内部用户 ID 串），但 operator_name 展示名仍在。
// fixture 设了 Operator，若 json tag 被改回 "operator" 此断言会 FAIL —— 本该早拦住泄漏。
func TestResponseEncoderOperLogNoOperator(t *testing.T) {
	h := newHasher(t)
	enc := NewResponseEncoder(h)
	e := SysOperLog{ID: 9, Operator: "1", OperatorName: "admin", Method: "POST", Path: "/sys/users"}
	m := enc.Item(&e)
	if _, leaked := m["operator"]; leaked {
		t.Errorf("operlog 出参不应含 operator（内部用户 ID 串收口，T-014），got %#v", m)
	}
	if m["operator_name"] != "admin" {
		t.Errorf("operator_name 展示路径应保留（误删即 FAIL），got %#v", m["operator_name"])
	}
	if _, isStr := m["id"].(string); !isStr {
		t.Error("id 应被编码为 hashid 字符串")
	}
}

// Page 编码：list 内每个 item 的 id 均为 hashid，整体无裸 id。
func TestResponseEncoderPageNoBareID(t *testing.T) {
	h := newHasher(t)
	enc := NewResponseEncoder(h)
	list := []SysOperLog{{ID: 1, Operator: "a"}, {ID: 2, Operator: "b"}}
	page := enc.Page(list, 2, 1, 20)

	// 序列化整体，断言无 "id": 数字
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.JSON(http.StatusOK, page)
	if bareIntID.MatchString(w.Body.String()) {
		t.Errorf("Page 出参不应含裸 \"id\": 数字，body=%s", w.Body.String())
	}
}

// --- path :id 解码：非法 hashid → 400 ---

func TestDecodePathIDBadHashid400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(nil, nil, nil, reg(t), newHasher(t)) // 服务 nil：非法 :id 在 decode 即 400，不触达 svc
	for _, route := range []struct {
		method, path string
		fn           gin.HandlerFunc
	}{
		{"PUT", "/sys/dict/types/:id", h.UpdateDictType},
		{"DELETE", "/sys/configs/:id", h.DeleteConfig},
	} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "id", Value: "!!!bad!!!"}}
		c.Request = httptest.NewRequest(route.method, "/x", strings.NewReader(`{}`))
		c.Request.Header.Set("Content-Type", "application/json")
		route.fn(c)
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s %s 非法 :id 应 400，got %d", route.method, route.path, w.Code)
		}
	}
}

func TestDecodePathIDValidRoundTrip(t *testing.T) {
	h := newHasher(t)
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Params = gin.Params{{Key: "id", Value: h.Encode(123)}}
	id, err := decodePathID(c, h, reg(t))
	if err != nil || id != 123 {
		t.Errorf("合法 hashid :id 应解回 123，got %d err=%v", id, err)
	}
}
