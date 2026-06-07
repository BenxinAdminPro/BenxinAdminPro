// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   HMAC-SHA256 签名与验签纯函数（恒定时间比较）
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:02:00
// +----------------------------------------------------------------------

package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
)

// Sign 计算 HMAC-SHA256 签名，返回 32 字节原始签名。
func Sign(key, message []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}

// Verify 使用 hmac.Equal 恒定时间比较验证 HMAC-SHA256 签名。
// 绝不使用 == 或 bytes.Equal 以防止时序攻击。
func Verify(key, message, signature []byte) bool {
	expected := Sign(key, message)
	return hmac.Equal(expected, signature)
}
