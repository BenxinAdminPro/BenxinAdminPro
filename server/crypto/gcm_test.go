// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   GCM 加解密测试 — 往返 + 篡改检测 + nonce 唯一 + fail-fast
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 03:18:00
// +----------------------------------------------------------------------

package crypto

import (
	"bytes"
	"encoding/base64"
	"testing"
)

var testGCMKey = bytes.Repeat([]byte{0x42}, 32)

func TestGCMRoundTrip(t *testing.T) {
	tests := []string{"hello", "", "你好世界", "a long string with special chars: !@#$%^&*()"}
	for _, plain := range tests {
		ct, err := EncryptGCM(testGCMKey, []byte(plain))
		if err != nil {
			t.Fatalf("EncryptGCM(%q): %v", plain, err)
		}
		pt, err := DecryptGCM(testGCMKey, ct)
		if err != nil {
			t.Fatalf("DecryptGCM: %v", err)
		}
		if string(pt) != plain {
			t.Errorf("round trip: got %q, want %q", pt, plain)
		}
	}
}

func TestGCMTamperDetection(t *testing.T) {
	ct, _ := EncryptGCM(testGCMKey, []byte("secret"))
	raw, _ := base64.StdEncoding.DecodeString(ct)
	// 篡改一字节
	raw[len(raw)-1] ^= 0xFF
	tampered := base64.StdEncoding.EncodeToString(raw)

	_, err := DecryptGCM(testGCMKey, tampered)
	if err == nil {
		t.Fatal("tampered ciphertext should fail authentication")
	}
}

func TestGCMNonceUnique(t *testing.T) {
	ct1, _ := EncryptGCM(testGCMKey, []byte("same"))
	ct2, _ := EncryptGCM(testGCMKey, []byte("same"))
	if ct1 == ct2 {
		t.Error("two encryptions of same plaintext should differ (unique nonce)")
	}
}

func TestGCMBadKeyLength(t *testing.T) {
	_, err := EncryptGCM([]byte("short"), []byte("x"))
	if err == nil {
		t.Error("short key should fail")
	}
	_, err = DecryptGCM([]byte("short"), "data")
	if err == nil {
		t.Error("short key should fail on decrypt")
	}
}
