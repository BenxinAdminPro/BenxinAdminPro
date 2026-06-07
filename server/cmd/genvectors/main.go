// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   跨语言加密测试向量生成工具 — 一次性生成冻结 golden 文件
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:25:00
// +----------------------------------------------------------------------
//
// 用法：go run ./cmd/genvectors
// 生成文件到 spec/crypto-vectors/，提交后作为冻结 golden 文件，供跨语言 parity 验证。
// 包含：
//   - aes_cbc.json: AES-256-CBC 向量（含 NIST SP 800-38A KAT + 自定义）
//   - hmac_sha256.json: HMAC-SHA256 向量（含 RFC 4231 KAT + 自定义）
//   - envelope.json: 端到端信封向量

package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ---------------------------------------------------------------------------
// 输出结构体
// ---------------------------------------------------------------------------

type AESCBCVector struct {
	Name          string `json:"name"`
	AESKeyB64     string `json:"aes_key_b64"`
	IVHex         string `json:"iv_hex"`
	PlaintextUTF8 string `json:"plaintext_utf8"`
	PlaintextHex  string `json:"plaintext_hex,omitempty"`
	CiphertextB64 string `json:"ciphertext_b64"`
}

type HMACSha256Vector struct {
	Name         string `json:"name"`
	HMACKeyB64   string `json:"hmac_key_b64"`
	HMACKeyHex   string `json:"hmac_key_hex,omitempty"`
	MessageUTF8  string `json:"message_utf8"`
	MessageHex   string `json:"message_hex,omitempty"`
	SignatureB64 string `json:"signature_b64"`
	SignatureHex string `json:"signature_hex,omitempty"`
}

type EnvelopeVector struct {
	Name                   string `json:"name"`
	AESKeyB64              string `json:"aes_key_b64"`
	HMACKeyB64             string `json:"hmac_key_b64"`
	IVHex                  string `json:"iv_hex"`
	Method                 string `json:"method"`
	Path                   string `json:"path"`
	Timestamp              string `json:"timestamp"`
	Nonce                  string `json:"nonce"`
	PlaintextUTF8          string `json:"plaintext_utf8"`
	ExpectedBodyB64        string `json:"expected_body_b64"`
	ExpectedSigningString  string `json:"expected_signing_string"`
	ExpectedSignatureB64   string `json:"expected_signature_b64"`
}

func main() {
	outDir := filepath.Join("spec", "crypto-vectors")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fatalf("mkdir %s: %v", outDir, err)
	}

	writeJSON(filepath.Join(outDir, "aes_cbc.json"), genAESCBCVectors())
	writeJSON(filepath.Join(outDir, "hmac_sha256.json"), genHMACVectors())
	writeJSON(filepath.Join(outDir, "envelope.json"), genEnvelopeVectors())

	fmt.Println("crypto-vectors generated successfully.")
}

// ---------------------------------------------------------------------------
// AES-256-CBC 向量
// ---------------------------------------------------------------------------

func genAESCBCVectors() []AESCBCVector {
	var vectors []AESCBCVector

	// ---- NIST SP 800-38A F.2.5: CBC-AES256.Encrypt ----
	// https://csrc.nist.gov/publications/detail/sp/800-38a/final
	nistKey := mustHex("603deb1015ca71be2b73aef0857d77811f352c073b6108d72d9810a30914dff4")
	nistIV := mustHex("000102030405060708090a0b0c0d0e0f")
	nistPlaintext := mustHex("6bc1bee22e409f96e93d7e117393172a") // Block 1 only
	nistCiphertext := encryptAESCBC(nistKey, nistIV, nistPlaintext)
	vectors = append(vectors, AESCBCVector{
		Name:          "nist_sp800_38a_f25_block1",
		AESKeyB64:     base64.StdEncoding.EncodeToString(nistKey),
		IVHex:         hex.EncodeToString(nistIV),
		PlaintextHex:  hex.EncodeToString(nistPlaintext),
		PlaintextUTF8: "",
		CiphertextB64: base64.StdEncoding.EncodeToString(nistCiphertext),
	})

	// NIST 4-block
	nistPlaintext4 := mustHex(
		"6bc1bee22e409f96e93d7e117393172a" +
			"ae2d8a571e03ac9c9eb76fac45af8e51" +
			"30c81c46a35ce411e5fbc1191a0a52ef" +
			"f69f2445df4f9b17ad2b417be66c3710")
	nistCiphertext4 := encryptAESCBC(nistKey, nistIV, nistPlaintext4)
	vectors = append(vectors, AESCBCVector{
		Name:          "nist_sp800_38a_f25_4blocks",
		AESKeyB64:     base64.StdEncoding.EncodeToString(nistKey),
		IVHex:         hex.EncodeToString(nistIV),
		PlaintextHex:  hex.EncodeToString(nistPlaintext4),
		PlaintextUTF8: "",
		CiphertextB64: base64.StdEncoding.EncodeToString(nistCiphertext4),
	})

	// ---- 自定义：ASCII 短文本 ----
	customKey := mustHex("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	customIV := mustHex("00112233445566778899aabbccddeeff")
	asciiPlain := []byte("hello-benxin")
	asciiCipher := encryptAESCBCPKCS7(customKey, customIV, asciiPlain)
	vectors = append(vectors, AESCBCVector{
		Name:          "ascii_short",
		AESKeyB64:     base64.StdEncoding.EncodeToString(customKey),
		IVHex:         hex.EncodeToString(customIV),
		PlaintextUTF8: "hello-benxin",
		CiphertextB64: base64.StdEncoding.EncodeToString(asciiCipher),
	})

	// ---- 自定义：多字节中文 ----
	chinesePlain := []byte("你好，本心")
	chineseCipher := encryptAESCBCPKCS7(customKey, customIV, chinesePlain)
	vectors = append(vectors, AESCBCVector{
		Name:          "chinese_multibyte",
		AESKeyB64:     base64.StdEncoding.EncodeToString(customKey),
		IVHex:         hex.EncodeToString(customIV),
		PlaintextUTF8: "你好，本心",
		CiphertextB64: base64.StdEncoding.EncodeToString(chineseCipher),
	})

	// ---- 自定义：空体 ----
	emptyPlain := []byte("")
	emptyCipher := encryptAESCBCPKCS7(customKey, customIV, emptyPlain)
	vectors = append(vectors, AESCBCVector{
		Name:          "empty_body",
		AESKeyB64:     base64.StdEncoding.EncodeToString(customKey),
		IVHex:         hex.EncodeToString(customIV),
		PlaintextUTF8: "",
		CiphertextB64: base64.StdEncoding.EncodeToString(emptyCipher),
	})

	return vectors
}

// ---------------------------------------------------------------------------
// HMAC-SHA256 向量
// ---------------------------------------------------------------------------

func genHMACVectors() []HMACSha256Vector {
	var vectors []HMACSha256Vector

	// ---- RFC 4231 Test Case 1 ----
	rfc1Key := mustHex("0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b")
	rfc1Msg := []byte("Hi There")
	rfc1Sig := computeHMAC(rfc1Key, rfc1Msg)
	vectors = append(vectors, HMACSha256Vector{
		Name:         "rfc4231_test1",
		HMACKeyB64:   base64.StdEncoding.EncodeToString(rfc1Key),
		HMACKeyHex:   hex.EncodeToString(rfc1Key),
		MessageUTF8:  "Hi There",
		SignatureB64: base64.StdEncoding.EncodeToString(rfc1Sig),
		SignatureHex: hex.EncodeToString(rfc1Sig),
	})

	// ---- RFC 4231 Test Case 2 ----
	rfc2Key := []byte("Jefe")
	rfc2Msg := []byte("what do ya want for nothing?")
	rfc2Sig := computeHMAC(rfc2Key, rfc2Msg)
	vectors = append(vectors, HMACSha256Vector{
		Name:         "rfc4231_test2",
		HMACKeyB64:   base64.StdEncoding.EncodeToString(rfc2Key),
		HMACKeyHex:   hex.EncodeToString(rfc2Key),
		MessageUTF8:  "what do ya want for nothing?",
		SignatureB64: base64.StdEncoding.EncodeToString(rfc2Sig),
		SignatureHex: hex.EncodeToString(rfc2Sig),
	})

	// ---- RFC 4231 Test Case 3 ----
	rfc3Key := mustHex("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	rfc3Data := mustHex("dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")
	rfc3Sig := computeHMAC(rfc3Key, rfc3Data)
	vectors = append(vectors, HMACSha256Vector{
		Name:         "rfc4231_test3",
		HMACKeyB64:   base64.StdEncoding.EncodeToString(rfc3Key),
		HMACKeyHex:   hex.EncodeToString(rfc3Key),
		MessageHex:   hex.EncodeToString(rfc3Data),
		MessageUTF8:  "",
		SignatureB64: base64.StdEncoding.EncodeToString(rfc3Sig),
		SignatureHex: hex.EncodeToString(rfc3Sig),
	})

	// ---- 自定义：短消息 ----
	customKey := mustHex("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	customMsg := []byte("abc")
	customSig := computeHMAC(customKey, customMsg)
	vectors = append(vectors, HMACSha256Vector{
		Name:         "custom_basic",
		HMACKeyB64:   base64.StdEncoding.EncodeToString(customKey),
		MessageUTF8:  "abc",
		SignatureB64: base64.StdEncoding.EncodeToString(customSig),
	})

	// ---- 自定义：中文 ----
	chineseMsg := []byte("你好世界")
	chineseSig := computeHMAC(customKey, chineseMsg)
	vectors = append(vectors, HMACSha256Vector{
		Name:         "custom_chinese",
		HMACKeyB64:   base64.StdEncoding.EncodeToString(customKey),
		MessageUTF8:  "你好世界",
		SignatureB64: base64.StdEncoding.EncodeToString(chineseSig),
	})

	// ---- 自定义：空消息 ----
	emptySig := computeHMAC(customKey, []byte(""))
	vectors = append(vectors, HMACSha256Vector{
		Name:         "custom_empty",
		HMACKeyB64:   base64.StdEncoding.EncodeToString(customKey),
		MessageUTF8:  "",
		SignatureB64: base64.StdEncoding.EncodeToString(emptySig),
	})

	return vectors
}

// ---------------------------------------------------------------------------
// 信封向量
// ---------------------------------------------------------------------------

func genEnvelopeVectors() []EnvelopeVector {
	aesKey := mustHex("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	hmacKey := mustHex("fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210")
	iv := mustHex("00112233445566778899aabbccddeeff")

	var vectors []EnvelopeVector

	// ---- POST JSON ----
	{
		plaintext := []byte(`{"msg":"hi"}`)
		bodyB64 := encryptToBodyB64(aesKey, iv, plaintext)
		sigStr := buildSigningString("POST", "/api/c/echo", "1750000000", "abc123nonce", bodyB64)
		sig := computeHMAC(hmacKey, []byte(sigStr))
		vectors = append(vectors, EnvelopeVector{
			Name:                  "post_json",
			AESKeyB64:             base64.StdEncoding.EncodeToString(aesKey),
			HMACKeyB64:            base64.StdEncoding.EncodeToString(hmacKey),
			IVHex:                 hex.EncodeToString(iv),
			Method:                "POST",
			Path:                  "/api/c/echo",
			Timestamp:             "1750000000",
			Nonce:                 "abc123nonce",
			PlaintextUTF8:         `{"msg":"hi"}`,
			ExpectedBodyB64:       bodyB64,
			ExpectedSigningString: sigStr,
			ExpectedSignatureB64:  base64.StdEncoding.EncodeToString(sig),
		})
	}

	// ---- 中文 body ----
	{
		plaintext := []byte(`{"name":"本心管理后台","version":"1.0"}`)
		bodyB64 := encryptToBodyB64(aesKey, iv, plaintext)
		sigStr := buildSigningString("PUT", "/api/c/config", "1750001000", "nonce_zh_cn_001", bodyB64)
		sig := computeHMAC(hmacKey, []byte(sigStr))
		vectors = append(vectors, EnvelopeVector{
			Name:                  "put_chinese",
			AESKeyB64:             base64.StdEncoding.EncodeToString(aesKey),
			HMACKeyB64:            base64.StdEncoding.EncodeToString(hmacKey),
			IVHex:                 hex.EncodeToString(iv),
			Method:                "PUT",
			Path:                  "/api/c/config",
			Timestamp:             "1750001000",
			Nonce:                 "nonce_zh_cn_001",
			PlaintextUTF8:         `{"name":"本心管理后台","version":"1.0"}`,
			ExpectedBodyB64:       bodyB64,
			ExpectedSigningString: sigStr,
			ExpectedSignatureB64:  base64.StdEncoding.EncodeToString(sig),
		})
	}

	// ---- 空 body ----
	{
		plaintext := []byte("")
		bodyB64 := encryptToBodyB64(aesKey, iv, plaintext)
		sigStr := buildSigningString("DELETE", "/api/c/item/42", "1750002000", "nonce_empty_body", bodyB64)
		sig := computeHMAC(hmacKey, []byte(sigStr))
		vectors = append(vectors, EnvelopeVector{
			Name:                  "delete_empty_body",
			AESKeyB64:             base64.StdEncoding.EncodeToString(aesKey),
			HMACKeyB64:            base64.StdEncoding.EncodeToString(hmacKey),
			IVHex:                 hex.EncodeToString(iv),
			Method:                "DELETE",
			Path:                  "/api/c/item/42",
			Timestamp:             "1750002000",
			Nonce:                 "nonce_empty_body",
			PlaintextUTF8:         "",
			ExpectedBodyB64:       bodyB64,
			ExpectedSigningString: sigStr,
			ExpectedSignatureB64:  base64.StdEncoding.EncodeToString(sig),
		})
	}

	// ---- GET（小写 method 测试 → 签名串应 UPPER）----
	{
		plaintext := []byte(`{"q":"search"}`)
		bodyB64 := encryptToBodyB64(aesKey, iv, plaintext)
		sigStr := buildSigningString("get", "/api/c/search", "1750003000", "nonce_get_test", bodyB64)
		sig := computeHMAC(hmacKey, []byte(sigStr))
		vectors = append(vectors, EnvelopeVector{
			Name:                  "get_method_case",
			AESKeyB64:             base64.StdEncoding.EncodeToString(aesKey),
			HMACKeyB64:            base64.StdEncoding.EncodeToString(hmacKey),
			IVHex:                 hex.EncodeToString(iv),
			Method:                "get",
			Path:                  "/api/c/search",
			Timestamp:             "1750003000",
			Nonce:                 "nonce_get_test",
			PlaintextUTF8:         `{"q":"search"}`,
			ExpectedBodyB64:       bodyB64,
			ExpectedSigningString: sigStr,
			ExpectedSignatureB64:  base64.StdEncoding.EncodeToString(sig),
		})
	}

	return vectors
}

// ---------------------------------------------------------------------------
// 加密/签名辅助（独立于 crypto 包，避免循环依赖）
// ---------------------------------------------------------------------------

func encryptAESCBC(key, iv, plaintext []byte) []byte {
	// 无 PKCS7 padding —— 仅用于 NIST KAT（明文已是块对齐）
	block, err := aes.NewCipher(key)
	if err != nil {
		fatalf("aes.NewCipher: %v", err)
	}
	ct := make([]byte, len(plaintext))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ct, plaintext)
	return ct
}

func encryptAESCBCPKCS7(key, iv, plaintext []byte) []byte {
	padded := pkcs7Pad(plaintext, aes.BlockSize)
	return encryptAESCBC(key, iv, padded)
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	pad := make([]byte, padding)
	for i := range pad {
		pad[i] = byte(padding)
	}
	return append(data, pad...)
}

func computeHMAC(key, message []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}

func encryptToBodyB64(aesKey, iv, plaintext []byte) string {
	ct := encryptAESCBCPKCS7(aesKey, iv, plaintext)
	combined := append(append([]byte{}, iv...), ct...)
	return base64.StdEncoding.EncodeToString(combined)
}

func buildSigningString(method, path, timestamp, nonce, bodyB64 string) string {
	return strings.ToUpper(method) + "\n" + path + "\n" + timestamp + "\n" + nonce + "\n" + bodyB64
}

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		fatalf("hex decode %q: %v", s, err)
	}
	return b
}

func writeJSON(path string, v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fatalf("marshal json: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fatalf("write %s: %v", path, err)
	}
	fmt.Printf("  wrote %s\n", path)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "genvectors: "+format+"\n", args...)
	os.Exit(1)
}
