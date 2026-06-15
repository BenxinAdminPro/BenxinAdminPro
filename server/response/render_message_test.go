// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   渲染层文案补全回归 — 8 键（file 6 + 菜单父节点 1 + config 2）经渲染返中文非裸 key
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-15 17:43:54
// | @updated   2026-06-15 18:10:00  T-009b 搭车（PM 裁定 A）：断言扩到 8 键（+config_decrypt_failed/migration_failed）
// +----------------------------------------------------------------------
//
// T-009b：坐实 file 错误 6 键 + sys.invalid_parent_menu + config 中心 2 键（搭车）经
// response 渲染层返回中文文案，而非裸 i18n key（修 T-007f §8-2「file 错误返裸 key」缺陷）。

package response

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/gin-gonic/gin"
)

// 期望文案表：码 offset → 渲染后中文 message（须与 render.go defaultMessages 一致）。
var renderMsgCases = []struct {
	code    int
	i18nKey string
	wantMsg string
}{
	// file 6 键（T-004b 段 70~75，此前 defaultMessages 缺失 → 返裸 key）
	{11070, "sys.file_too_large", "文件大小超出限制"},
	{11071, "sys.file_ext_not_allowed", "文件扩展名不允许"},
	{11072, "sys.file_type_mismatch", "文件类型与扩展名不匹配"},
	{11073, "sys.file_not_found", "文件不存在"},
	{11074, "sys.file_name_invalid", "无效的文件名"},
	{11075, "sys.storage_failed", "存储操作失败"},
	// 菜单父节点新键（T-009b 新增 offset 47）
	{11047, "sys.invalid_parent_menu", "无效的父菜单"},
	// config 中心 2 键（T-009b 搭车补文案，码 11080/11081 早注册、缺文案 → 返裸 key）
	{11080, "sys.config_decrypt_failed", "参数解密失败"},
	{11081, "sys.migration_failed", "数据库迁移失败"},
}

// buildProdRenderer 构建与生产同构的渲染器（errcode.Registry → response.Registry → 默认 msgFn）。
func buildProdRenderer(t *testing.T) *Renderer {
	t.Helper()
	ecReg, err := errcode.NewRegistry(11000)
	if err != nil {
		t.Fatalf("errcode.NewRegistry: %v", err)
	}
	respReg := NewRegistry()
	for _, s := range ecReg.AllSpecs() {
		if err := respReg.Register(ErrSpec{Code: s.Code, HTTP: s.HTTP, I18nKey: s.I18nKey}); err != nil {
			t.Fatalf("register %d: %v", s.Code, err)
		}
	}
	return NewRenderer(respReg, nil) // nil → defaultMessageLookup
}

// TestRenderFileAndMenuMessages 经完整渲染路径断言每个码返回中文文案、非裸 i18n key。
func TestRenderFileAndMenuMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := buildProdRenderer(t)

	for _, tc := range renderMsgCases {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		r.Fail(c, tc.code)

		var body struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("code %d: unmarshal body %q: %v", tc.code, w.Body.String(), err)
		}
		if body.Code != tc.code {
			t.Errorf("code %d: body.code got %d", tc.code, body.Code)
		}
		// 头号断言：message 不是裸 i18n key
		if body.Message == tc.i18nKey {
			t.Errorf("code %d: message is bare i18n key %q (文案缺失未修)", tc.code, tc.i18nKey)
		}
		if body.Message != tc.wantMsg {
			t.Errorf("code %d: message got %q, want %q", tc.code, body.Message, tc.wantMsg)
		}
	}
}

// TestDefaultMessageLookupNoBareKey 直测查找函数：6 file 键 + menu 键均命中、不回退裸 key。
func TestDefaultMessageLookupNoBareKey(t *testing.T) {
	for _, tc := range renderMsgCases {
		got := defaultMessageLookup(tc.i18nKey)
		if got == tc.i18nKey {
			t.Errorf("key %q: lookup returned bare key (未注册文案)", tc.i18nKey)
		}
		if got != tc.wantMsg {
			t.Errorf("key %q: got %q, want %q", tc.i18nKey, got, tc.wantMsg)
		}
	}
}
