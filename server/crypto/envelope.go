// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   C 端加密信封 — 签名串构造 + body 编解码 + 信封签名
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:03:00
// +----------------------------------------------------------------------

package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

// BuildSigningString 按协议构造规范签名串：
//
//	UPPER(method) "\n" path "\n" timestamp "\n" nonce "\n" BODY_B64
func BuildSigningString(method, path, timestamp, nonce, bodyB64 string) string {
	return strings.ToUpper(method) + "\n" + path + "\n" + timestamp + "\n" + nonce + "\n" + bodyB64
}

// EncryptBody 使用随机 IV 加密明文，返回 base64(IV || ciphertext)。
func EncryptBody(aesKey, plaintext []byte) (bodyB64 string, err error) {
	iv := make([]byte, 16)
	if _, err = rand.Read(iv); err != nil {
		return "", fmt.Errorf("crypto: generate IV: %w", err)
	}
	return EncryptBodyWithIV(aesKey, iv, plaintext)
}

// EncryptBodyWithIV 使用指定 IV 加密明文，返回 base64(IV || ciphertext)。
// 测试/向量生成用确定性 IV，生产必须用随机 IV。
func EncryptBodyWithIV(aesKey, iv, plaintext []byte) (bodyB64 string, err error) {
	ciphertext, err := Encrypt(aesKey, iv, plaintext)
	if err != nil {
		return "", err
	}
	combined := make([]byte, len(iv)+len(ciphertext))
	copy(combined, iv)
	copy(combined[len(iv):], ciphertext)
	return base64.StdEncoding.EncodeToString(combined), nil
}

// DecryptBody 解码 base64 → 分离 IV → AES-256-CBC 解密。
func DecryptBody(aesKey []byte, bodyB64 string) ([]byte, error) {
	combined, err := base64.StdEncoding.DecodeString(bodyB64)
	if err != nil {
		return nil, fmt.Errorf("crypto: invalid base64: %w", err)
	}
	if len(combined) < 17 { // 至少 16 字节 IV + 1 块密文
		return nil, fmt.Errorf("crypto: encrypted data too short (%d bytes)", len(combined))
	}
	iv := combined[:16]
	ciphertext := combined[16:]
	return Decrypt(aesKey, iv, ciphertext)
}

// SignEnvelope 对签名串计算 HMAC-SHA256，返回 base64 编码的签名。
func SignEnvelope(hmacKey []byte, signingString string) string {
	sig := Sign(hmacKey, []byte(signingString))
	return base64.StdEncoding.EncodeToString(sig)
}

// VerifyEnvelope 验证签名串的 HMAC-SHA256 签名（恒定时间比较）。
func VerifyEnvelope(hmacKey []byte, signingString, signatureB64 string) bool {
	sig, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return false
	}
	return Verify(hmacKey, []byte(signingString), sig)
}
