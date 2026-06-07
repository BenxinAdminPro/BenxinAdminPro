// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   加密向量断言测试 — 加载 spec/crypto-vectors/*.json 验证一致性
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:35:00
// +----------------------------------------------------------------------

package crypto

import (
	stdaes "crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func specDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "spec", "crypto-vectors")
}

// ---------------------------------------------------------------------------
// AES-CBC 向量
// ---------------------------------------------------------------------------

type aesCBCVector struct {
	Name          string `json:"name"`
	AESKeyB64     string `json:"aes_key_b64"`
	IVHex         string `json:"iv_hex"`
	PlaintextUTF8 string `json:"plaintext_utf8"`
	PlaintextHex  string `json:"plaintext_hex"`
	CiphertextB64 string `json:"ciphertext_b64"`
}

func TestAESCBCVectors(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(specDir(), "aes_cbc.json"))
	if err != nil {
		t.Fatalf("read aes_cbc.json: %v", err)
	}
	var vectors []aesCBCVector
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(vectors) < 3 {
		t.Fatalf("expected at least 3 vectors, got %d", len(vectors))
	}

	for _, v := range vectors {
		t.Run(v.Name, func(t *testing.T) {
			key, _ := base64.StdEncoding.DecodeString(v.AESKeyB64)
			iv, _ := hex.DecodeString(v.IVHex)
			wantCT, _ := base64.StdEncoding.DecodeString(v.CiphertextB64)

			var plaintext []byte
			if v.PlaintextHex != "" {
				plaintext, _ = hex.DecodeString(v.PlaintextHex)
			} else {
				plaintext = []byte(v.PlaintextUTF8)
			}

			// NIST KAT 向量无 PKCS7 padding（明文已块对齐），用 raw CBC
			isNIST := v.PlaintextHex != "" && len(plaintext)%16 == 0 && v.PlaintextUTF8 == ""

			if isNIST {
				// Raw AES-CBC 加密（无 PKCS7）
				gotCT := rawAESCBCEncrypt(t, key, iv, plaintext)
				if !equalBytes(gotCT, wantCT) {
					t.Errorf("ciphertext mismatch:\n  got:  %x\n  want: %x", gotCT, wantCT)
				}
				// Raw 解密验证
				gotPT := rawAESCBCDecrypt(t, key, iv, wantCT)
				if !equalBytes(gotPT, plaintext) {
					t.Errorf("plaintext mismatch after decrypt")
				}
			} else {
				// 带 PKCS7 的加解密（使用 crypto 包的函数）
				gotCT, err := Encrypt(key, iv, plaintext)
				if err != nil {
					t.Fatalf("Encrypt: %v", err)
				}
				if !equalBytes(gotCT, wantCT) {
					t.Errorf("ciphertext mismatch:\n  got:  %s\n  want: %s",
						base64.StdEncoding.EncodeToString(gotCT), v.CiphertextB64)
				}
				gotPT, err := Decrypt(key, iv, wantCT)
				if err != nil {
					t.Fatalf("Decrypt: %v", err)
				}
				if !equalBytes(gotPT, plaintext) {
					t.Errorf("plaintext mismatch: got %q, want %q", gotPT, plaintext)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// HMAC-SHA256 向量
// ---------------------------------------------------------------------------

type hmacVector struct {
	Name         string `json:"name"`
	HMACKeyB64   string `json:"hmac_key_b64"`
	HMACKeyHex   string `json:"hmac_key_hex"`
	MessageUTF8  string `json:"message_utf8"`
	MessageHex   string `json:"message_hex"`
	SignatureB64 string `json:"signature_b64"`
	SignatureHex string `json:"signature_hex"`
}

func TestHMACSha256Vectors(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(specDir(), "hmac_sha256.json"))
	if err != nil {
		t.Fatalf("read hmac_sha256.json: %v", err)
	}
	var vectors []hmacVector
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(vectors) < 3 {
		t.Fatalf("expected at least 3 vectors, got %d", len(vectors))
	}

	for _, v := range vectors {
		t.Run(v.Name, func(t *testing.T) {
			key, _ := base64.StdEncoding.DecodeString(v.HMACKeyB64)

			var message []byte
			if v.MessageHex != "" {
				message, _ = hex.DecodeString(v.MessageHex)
			} else {
				message = []byte(v.MessageUTF8)
			}

			wantSig, _ := base64.StdEncoding.DecodeString(v.SignatureB64)

			gotSig := Sign(key, message)
			if !equalBytes(gotSig, wantSig) {
				t.Errorf("signature mismatch:\n  got:  %x\n  want: %x", gotSig, wantSig)
			}

			// Verify 也应通过
			if !Verify(key, message, wantSig) {
				t.Error("Verify returned false for correct signature")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 信封向量
// ---------------------------------------------------------------------------

type envelopeVector struct {
	Name                  string `json:"name"`
	AESKeyB64             string `json:"aes_key_b64"`
	HMACKeyB64            string `json:"hmac_key_b64"`
	IVHex                 string `json:"iv_hex"`
	Method                string `json:"method"`
	Path                  string `json:"path"`
	Timestamp             string `json:"timestamp"`
	Nonce                 string `json:"nonce"`
	PlaintextUTF8         string `json:"plaintext_utf8"`
	ExpectedBodyB64       string `json:"expected_body_b64"`
	ExpectedSigningString string `json:"expected_signing_string"`
	ExpectedSignatureB64  string `json:"expected_signature_b64"`
}

func TestEnvelopeVectors(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(specDir(), "envelope.json"))
	if err != nil {
		t.Fatalf("read envelope.json: %v", err)
	}
	var vectors []envelopeVector
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(vectors) < 3 {
		t.Fatalf("expected at least 3 vectors, got %d", len(vectors))
	}

	for _, v := range vectors {
		t.Run(v.Name, func(t *testing.T) {
			aesKey, _ := base64.StdEncoding.DecodeString(v.AESKeyB64)
			hmacKey, _ := base64.StdEncoding.DecodeString(v.HMACKeyB64)
			iv, _ := hex.DecodeString(v.IVHex)

			// 1. 验证加密 body
			gotBodyB64, err := EncryptBodyWithIV(aesKey, iv, []byte(v.PlaintextUTF8))
			if err != nil {
				t.Fatalf("EncryptBodyWithIV: %v", err)
			}
			if gotBodyB64 != v.ExpectedBodyB64 {
				t.Errorf("body_b64 mismatch:\n  got:  %s\n  want: %s", gotBodyB64, v.ExpectedBodyB64)
			}

			// 2. 验证签名串
			gotSigStr := BuildSigningString(v.Method, v.Path, v.Timestamp, v.Nonce, v.ExpectedBodyB64)
			if gotSigStr != v.ExpectedSigningString {
				t.Errorf("signing_string mismatch:\n  got:  %q\n  want: %q", gotSigStr, v.ExpectedSigningString)
			}

			// 3. 验证签名
			gotSig := SignEnvelope(hmacKey, gotSigStr)
			if gotSig != v.ExpectedSignatureB64 {
				t.Errorf("signature mismatch:\n  got:  %s\n  want: %s", gotSig, v.ExpectedSignatureB64)
			}

			// 4. 验证 VerifyEnvelope 通过
			if !VerifyEnvelope(hmacKey, gotSigStr, v.ExpectedSignatureB64) {
				t.Error("VerifyEnvelope returned false for correct signature")
			}

			// 5. 验证解密还原
			gotPT, err := DecryptBody(aesKey, v.ExpectedBodyB64)
			if err != nil {
				t.Fatalf("DecryptBody: %v", err)
			}
			if string(gotPT) != v.PlaintextUTF8 {
				t.Errorf("plaintext mismatch: got %q, want %q", gotPT, v.PlaintextUTF8)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 辅助：Raw AES-CBC（无 PKCS7，仅用于 NIST KAT）
// ---------------------------------------------------------------------------

func rawAESCBCEncrypt(t *testing.T, key, iv, plaintext []byte) []byte {
	t.Helper()
	block, err := stdaes.NewCipher(key)
	if err != nil {
		t.Fatalf("aes.NewCipher: %v", err)
	}
	ct := make([]byte, len(plaintext))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ct, plaintext)
	return ct
}

func rawAESCBCDecrypt(t *testing.T, key, iv, ciphertext []byte) []byte {
	t.Helper()
	block, err := stdaes.NewCipher(key)
	if err != nil {
		t.Fatalf("aes.NewCipher: %v", err)
	}
	pt := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(pt, ciphertext)
	return pt
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
