// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   dept/menu/auth_info 三 handler 构造器 hasher==nil fail-fast 断言（T-017 项④）
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-18 13:50:56
// +----------------------------------------------------------------------
//
// 三个对外序列化 SysDept/SysMenu 树的 handler，缺 hasher 即无编码器、出参退化为
// 裸 marshal（泄漏裸内部 id/parent_id/ancestors）。本片把该退化路径改为构造期
// fail-fast panic：nil hasher → panic；非 nil hasher → 正常构造不 panic。

package rbac

import (
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/idcodec"
)

// mustPanic 断言 fn 触发 panic（仓内既有 recover 范式，见 response/response_test.go）。
func mustPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("%s: 期望 hasher==nil 触发 panic，实际未 panic", name)
		}
	}()
	fn()
}

func testHasher(t *testing.T) *Hasher {
	t.Helper()
	h, err := idcodec.NewHasher(idcodec.HashidConfig{Salt: "test-construct-salt", MinLength: 8})
	if err != nil {
		t.Fatalf("NewHasher: %v", err)
	}
	return h
}

// TestHandlerConstructorsPanicOnNilHasher 三个构造器传 nil hasher 均须 panic（根除裸出参泄漏面）。
func TestHandlerConstructorsPanicOnNilHasher(t *testing.T) {
	mustPanic(t, "NewDeptHandler", func() { NewDeptHandler(nil, nil, nil) })
	mustPanic(t, "NewMenuHandler", func() { NewMenuHandler(nil, nil, nil) })
	mustPanic(t, "NewAuthInfoHandler", func() { NewAuthInfoHandler(nil, nil, nil) })
}

// TestHandlerConstructorsOKWithHasher 传非 nil hasher 正常构造、不 panic，且 enc 已装配。
func TestHandlerConstructorsOKWithHasher(t *testing.T) {
	h := testHasher(t)

	if dh := NewDeptHandler(nil, nil, h); dh.enc == nil {
		t.Fatal("NewDeptHandler: 非 nil hasher 下 enc 仍为 nil")
	}
	if mh := NewMenuHandler(nil, nil, h); mh.enc == nil {
		t.Fatal("NewMenuHandler: 非 nil hasher 下 enc 仍为 nil")
	}
	if ah := NewAuthInfoHandler(nil, nil, h); ah.enc == nil {
		t.Fatal("NewAuthInfoHandler: 非 nil hasher 下 enc 仍为 nil")
	}
}
