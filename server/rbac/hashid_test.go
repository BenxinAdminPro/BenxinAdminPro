// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   Hashid 编解码测试 — 往返/min_length/解码失败
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 21:38:00
// +----------------------------------------------------------------------

package rbac

import (
	"testing"
)

func TestHashidRoundTrip(t *testing.T) {
	h, err := NewHasher(HashidConfig{Salt: "test-salt", MinLength: 8})
	if err != nil {
		t.Fatalf("NewHasher: %v", err)
	}

	ids := []uint64{1, 42, 999, 123456789}
	for _, id := range ids {
		encoded := h.Encode(id)
		if len(encoded) < 8 {
			t.Errorf("encoded %d = %q, shorter than min_length 8", id, encoded)
		}
		decoded, err := h.Decode(encoded)
		if err != nil {
			t.Fatalf("Decode(%q): %v", encoded, err)
		}
		if decoded != id {
			t.Errorf("round trip: got %d, want %d", decoded, id)
		}
	}
}

func TestHashidDecodeFail(t *testing.T) {
	h, _ := NewHasher(HashidConfig{Salt: "test-salt", MinLength: 8})
	_, err := h.Decode("invalid!!!")
	if err == nil {
		t.Error("expected error for invalid hashid")
	}
}

func TestHashidDifferentSalts(t *testing.T) {
	h1, _ := NewHasher(HashidConfig{Salt: "salt-a", MinLength: 8})
	h2, _ := NewHasher(HashidConfig{Salt: "salt-b", MinLength: 8})

	e1 := h1.Encode(42)
	e2 := h2.Encode(42)
	if e1 == e2 {
		t.Error("different salts should produce different hashids")
	}

	// h2 不能解 h1 的 hashid
	_, err := h2.Decode(e1)
	if err == nil {
		decoded, _ := h2.Decode(e1)
		if decoded == 42 {
			t.Error("different salt should not decode to same ID")
		}
	}
}

func TestHashidEmptySalt(t *testing.T) {
	_, err := NewHasher(HashidConfig{Salt: ""})
	if err == nil {
		t.Error("expected error for empty salt")
	}
}

func TestHashidEncodeOptional(t *testing.T) {
	h, _ := NewHasher(HashidConfig{Salt: "test", MinLength: 8})
	if h.EncodeOptional(nil) != "" {
		t.Error("nil should encode to empty string")
	}
	id := uint64(42)
	if h.EncodeOptional(&id) == "" {
		t.Error("non-nil should encode to non-empty string")
	}
}
