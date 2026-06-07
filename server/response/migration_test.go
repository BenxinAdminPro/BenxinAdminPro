// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   错误码回归断言 — 迁移后既有码值/HTTP/i18nKey 逐项对照
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 01:00:00
// +----------------------------------------------------------------------

package response

import (
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
)

// 快照表：T-001~T-003c + T-004a 全部既有错误码的 (code, HTTP, i18nKey)
// 迁移后必须逐项一致，否则测试失败。
var expectedCodes = []struct {
	Code    int
	HTTP    int
	I18nKey string
}{
	// T-001 安全地基 (base=11000, offset 1~9)
	{11001, 400, "security.missing_headers"},
	{11002, 400, "security.timestamp_expired"},
	{11003, 400, "security.sign_invalid"},
	{11004, 400, "security.nonce_replay"},
	{11005, 400, "security.decrypt_failed"},
	{11006, 401, "security.token_invalid"},
	{11007, 401, "security.token_expired"},
	{11008, 401, "security.token_revoked"},
	{11009, 403, "security.forbidden"},

	// T-002 认证授权 (offset 20~24)
	{11020, 401, "auth.bad_credentials"},
	{11021, 400, "auth.captcha_required"},
	{11022, 400, "auth.captcha_invalid"},
	{11023, 423, "auth.account_locked"},
	{11024, 403, "auth.account_disabled"},

	// T-003a 组织架构 (offset 30~35)
	{11030, 404, "sys.user_not_found"},
	{11031, 409, "sys.username_exists"},
	{11032, 409, "sys.dept_has_children"},
	{11033, 409, "sys.dept_has_users"},
	{11034, 409, "sys.post_code_exists"},
	{11035, 400, "sys.invalid_parent_dept"},

	// T-003b RBAC 核心 (offset 40~46)
	{11040, 409, "sys.role_code_exists"},
	{11041, 400, "sys.menu_perm_required"},
	{11042, 400, "sys.menu_perm_forbidden"},
	{11043, 409, "sys.menu_has_children"},
	{11044, 409, "sys.role_in_use"},
	{11045, 400, "sys.invalid_id"},
	{11046, 409, "sys.perm_code_exists"},

	// T-003c 数据权限 (offset 50)
	{11050, 400, "sys.invalid_data_scope"},

	// T-004a 系统管理 (offset 60~63)
	{11060, 409, "sys.dict_type_exists"},
	{11061, 404, "sys.dict_type_not_found"},
	{11062, 409, "sys.config_key_exists"},
	{11063, 404, "sys.config_not_found"},
}

// TestErrcodeRegressionViaRegistry 验证：
// 1. errcode.Registry 的所有码经 AllSpecs 注册到 response.Registry
// 2. 逐项断言 (code, HTTP, i18nKey) 与快照表一致
// 头号约束：迁移不破坏既有码。
func TestErrcodeRegressionViaRegistry(t *testing.T) {
	// 构建 errcode.Registry（段基址 11000，与生产一致）
	ecReg, err := errcode.NewRegistry(11000)
	if err != nil {
		t.Fatalf("errcode.NewRegistry: %v", err)
	}

	// 注册到 response.Registry
	respReg := NewRegistry()
	specs := ecReg.AllSpecs()
	for _, s := range specs {
		if err := respReg.Register(ErrSpec{Code: s.Code, HTTP: s.HTTP, I18nKey: s.I18nKey}); err != nil {
			t.Fatalf("Register code %d: %v", s.Code, err)
		}
	}

	// 逐项对照
	for _, expected := range expectedCodes {
		spec, ok := respReg.Lookup(expected.Code)
		if !ok {
			t.Errorf("code %d not found in registry", expected.Code)
			continue
		}
		if spec.HTTP != expected.HTTP {
			t.Errorf("code %d HTTP: got %d, want %d", expected.Code, spec.HTTP, expected.HTTP)
		}
		if spec.I18nKey != expected.I18nKey {
			t.Errorf("code %d I18nKey: got %q, want %q", expected.Code, spec.I18nKey, expected.I18nKey)
		}
	}

	// 确认注册总数与快照表一致（防止遗漏）
	if len(specs) != len(expectedCodes) {
		t.Errorf("errcode AllSpecs count: got %d, want %d", len(specs), len(expectedCodes))
	}
}
