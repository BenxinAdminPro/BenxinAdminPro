// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   上传安全工具 — 大小/扩展名/文件名消毒/Content-Type 四件套
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 01:14:00
// +----------------------------------------------------------------------

package storage

import (
	"fmt"
	"mime"
	"path/filepath"
	"strings"
	"unicode"
)

// UploadConfig 上传安全配置，构造注入。
type UploadConfig struct {
	MaxSizeBytes int64    // 最大文件大小（字节），如 10*1024*1024
	AllowedExts  []string // 扩展名白名单（不含点），如 ["jpg","png","pdf"]
}

// ValidateSize 校验文件大小。
func (c UploadConfig) ValidateSize(size int64) error {
	if c.MaxSizeBytes > 0 && size > c.MaxSizeBytes {
		return fmt.Errorf("file too large: %d > %d", size, c.MaxSizeBytes)
	}
	return nil
}

// ValidateExt 校验扩展名在白名单内。
func (c UploadConfig) ValidateExt(ext string) error {
	ext = strings.TrimPrefix(strings.ToLower(ext), ".")
	for _, allowed := range c.AllowedExts {
		if strings.EqualFold(ext, allowed) {
			return nil
		}
	}
	return fmt.Errorf("extension not allowed: %s", ext)
}

// ValidateContentType 校验 Content-Type 与扩展名一致性。
func ValidateContentType(contentType, ext string) error {
	ext = strings.TrimPrefix(strings.ToLower(ext), ".")
	expected := mime.TypeByExtension("." + ext)
	if expected == "" {
		return nil // 未知扩展名不做 MIME 校验
	}
	// 比较 MIME 主类型（忽略参数如 charset）
	declaredMain := strings.Split(contentType, ";")[0]
	expectedMain := strings.Split(expected, ";")[0]
	if strings.TrimSpace(declaredMain) != strings.TrimSpace(expectedMain) {
		return fmt.Errorf("content-type mismatch: declared %q, expected %q for .%s", contentType, expected, ext)
	}
	return nil
}

// SanitizeFilename 消毒文件名：
//   - 去除路径分隔符和 ..
//   - 去除控制字符
//   - 限制长度（最大 200 字符）
//   - 返回安全的纯文件名（不含路径）
func SanitizeFilename(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("empty filename")
	}

	// 先检查原始名是否含路径穿越
	if strings.Contains(name, "..") {
		return "", fmt.Errorf("invalid filename: contains ..")
	}

	// 检查控制字符（含 null byte）
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", fmt.Errorf("invalid filename: contains control character")
		}
	}

	// 取纯文件名（去路径）
	name = filepath.Base(name)

	// 去除路径分隔符（双保险）
	var cleaned strings.Builder
	for _, r := range name {
		if r != '/' && r != '\\' {
			cleaned.WriteRune(r)
		}
	}
	name = cleaned.String()

	if name == "" || name == "." {
		return "", fmt.Errorf("invalid filename after sanitization")
	}

	// 限制长度
	if len(name) > 200 {
		ext := filepath.Ext(name)
		name = name[:200-len(ext)] + ext
	}

	return name, nil
}

// ExtFromFilename 从消毒后的文件名提取扩展名（不含点，小写）。
func ExtFromFilename(name string) string {
	ext := filepath.Ext(name)
	return strings.TrimPrefix(strings.ToLower(ext), ".")
}
