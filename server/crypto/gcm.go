// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   AES-256-GCM 认证加密 — 存储场景（与 T-001 CBC 信封并存）
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 03:00:00
// +----------------------------------------------------------------------

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

// EncryptGCM 使用 AES-256-GCM 加密。
// key 必须恰好 32 字节。每次加密使用唯一随机 nonce（12 字节）。
// 返回 base64( nonce(12B) || ciphertext || tag )。
// 存储格式自描述，PHP parity 可解。
func EncryptGCM(key, plaintext []byte) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("crypto: GCM key must be 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("crypto: gcm new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: gcm new gcm: %w", err)
	}

	// 随机 nonce（12 字节），每次加密必须唯一
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("crypto: gcm generate nonce: %w", err)
	}

	// Seal: nonce || ciphertext || tag
	sealed := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// DecryptGCM 使用 AES-256-GCM 解密。
// data 为 base64( nonce(12B) || ciphertext || tag )。
// 篡改检测：认证标签校验失败返回 error，不返回脏明文。
func DecryptGCM(key []byte, data string) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("crypto: GCM key must be 32 bytes, got %d", len(key))
	}

	raw, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("crypto: gcm decode base64: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: gcm new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: gcm new gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize {
		return nil, errors.New("crypto: gcm data too short")
	}

	nonce, ciphertext := raw[:nonceSize], raw[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: gcm authentication failed (tampered?): %w", err)
	}

	return plaintext, nil
}
