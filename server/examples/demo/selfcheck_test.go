// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   装配自检 typed-nil 侦测测试 — 新写法抓得住、旧写法 ==nil 抓不住（T-004d）
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-09 16:09:17
// +----------------------------------------------------------------------

package main

import (
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/idcodec"
)

// TestIsNilDepCatchesTypedNil 坐实：typed-nil 指针装入 any 后，
// 旧写法 v==nil 抓不住，新写法 isNilDep 抓得住 —— 这是 T-003e 评审暴露的自检盲区。
func TestIsNilDepCatchesTypedNil(t *testing.T) {
	var typedNil *idcodec.Hasher // (*idcodec.Hasher)(nil)
	var asAny any = typedNil

	// 旧写法（自检曾用 v==nil）：跨函数边界模拟，typed-nil 在 any 中 ==nil 为 false → 漏检
	naiveIsNil := func(v any) bool { return v == nil }
	if naiveIsNil(asAny) {
		t.Fatal("前提失效：typed-nil 装入 any 后 ==nil 本应为 false")
	}

	// 新写法：isNilDep 必须侦测到 typed-nil（旧写法漏、新写法抓——对比成立）
	if !isNilDep(asAny) {
		t.Error("isNilDep 应侦测到 typed-nil (*idcodec.Hasher)(nil)，但漏检")
	}

	// 不误报：有效实例
	h, _ := idcodec.NewHasher(idcodec.HashidConfig{Salt: "x", MinLength: 8})
	if isNilDep(h) {
		t.Error("isNilDep 不应把有效 *Hasher 判为 nil")
	}

	// untyped nil 也判 nil
	if !isNilDep(nil) {
		t.Error("isNilDep 应判 untyped nil 为 nil")
	}
}

// TestIsNilDepOnVariousKinds 覆盖自检 map 可能持有的依赖类型（指针/接口/map）。
func TestIsNilDepOnVariousKinds(t *testing.T) {
	var nilMap map[string]int
	var nilIface interface{ Foo() }
	cases := []struct {
		name string
		v    any
		want bool
	}{
		{"nil map", nilMap, true},
		{"nil interface", nilIface, true},
		{"non-nil int", 5, false},
		{"non-nil string", "x", false},
	}
	for _, c := range cases {
		if got := isNilDep(c.v); got != c.want {
			t.Errorf("%s: isNilDep=%v, want %v", c.name, got, c.want)
		}
	}
}
