// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   response Registry 测试 — 注册/查找/冲突检测
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 00:40:00
// +----------------------------------------------------------------------

package response

import "testing"

func TestRegistryRegisterAndLookup(t *testing.T) {
	r := NewRegistry()
	err := r.Register(ErrSpec{Code: 11001, HTTP: 400, I18nKey: "security.missing_headers"})
	if err != nil {
		t.Fatal(err)
	}

	spec, ok := r.Lookup(11001)
	if !ok {
		t.Fatal("should find registered code")
	}
	if spec.HTTP != 400 {
		t.Errorf("HTTP: got %d, want 400", spec.HTTP)
	}
	if spec.I18nKey != "security.missing_headers" {
		t.Errorf("I18nKey: got %q", spec.I18nKey)
	}
}

func TestRegistryDuplicateConflict(t *testing.T) {
	r := NewRegistry()
	r.Register(ErrSpec{Code: 11001, HTTP: 400, I18nKey: "a"})
	err := r.Register(ErrSpec{Code: 11001, HTTP: 401, I18nKey: "b"})
	if err == nil {
		t.Fatal("duplicate code should fail")
	}
}

func TestRegistryMustRegisterPanic(t *testing.T) {
	r := NewRegistry()
	r.MustRegister(ErrSpec{Code: 1, HTTP: 400, I18nKey: "a"})

	defer func() {
		if recover() == nil {
			t.Fatal("should panic on duplicate")
		}
	}()
	r.MustRegister(ErrSpec{Code: 1, HTTP: 400, I18nKey: "b"})
}

func TestRegistryLookupNotFound(t *testing.T) {
	r := NewRegistry()
	_, ok := r.Lookup(99999)
	if ok {
		t.Error("should not find unregistered code")
	}
}

func TestRegistryBusinessModuleCanRegister(t *testing.T) {
	r := NewRegistry()
	// 底座码
	r.Register(ErrSpec{Code: 11001, HTTP: 400, I18nKey: "security.x"})
	// 业务模块码（不同段）
	err := r.Register(ErrSpec{Code: 20001, HTTP: 400, I18nKey: "biz.some_error"})
	if err != nil {
		t.Fatalf("business module should be able to register: %v", err)
	}
	spec, ok := r.Lookup(20001)
	if !ok || spec.I18nKey != "biz.some_error" {
		t.Error("business code should be retrievable")
	}
}
