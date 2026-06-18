// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   上传安全工具 — 大小/扩展名/文件名消毒/Content-Type 四件套
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 01:14:00
// | @updated   2026-06-18 08:51:21  T-016：Content-Type 交叉校验改确定性大类级（去 OS mime db 依赖 + alias 免疫）
// | @updated   2026-06-18 13:50:56  T-017：二义容器扩展名补充容忍表（ogg 容忍 audio|video，主表 extCategory 不动）
// +----------------------------------------------------------------------

package storage

import (
	"fmt"
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

// extCategory 是底座自持的「扩展名 → 顶层大类」确定性映射。
//
// 设计意图（T-016）：ValidateContentType 原先依赖 mime.TypeByExtension 读宿主机
// OS MIME 数据库，macOS/Linux/最小化容器返回值不一致（甚至为空），导致同一上传
// 在不同部署环境结果漂移，且裸子类型相等比较不认 audio/x-m4a 等合法 alias。
// 此表把校验收敛到「顶层大类（category）级」，去 OS 依赖、对 alias 免疫。
//
// 该校验为「组织性非安全边界」（仅防跨大类粗暴错配，非安全闸）；表外扩展名走
// 「未知放行」兜底（见 ValidateContentType），故无需穷举，仅覆盖底座默认白名单 +
// 常见项即可。key 为小写无点扩展名，value 为顶层大类。
var extCategory = map[string]string{
	// image
	"jpg":  "image", // image/jpeg
	"jpeg": "image", // image/jpeg
	"png":  "image", // image/png
	"gif":  "image", // image/gif
	// video
	"mp4":  "video", // video/mp4
	"webm": "video", // video/webm
	"mov":  "video", // video/quicktime
	// audio
	"mp3": "audio", // audio/mpeg
	"wav": "audio", // audio/wav, audio/x-wav
	"ogg": "audio", // audio/ogg
	"m4a": "audio", // audio/mp4, audio/x-m4a, audio/m4a（freedesktop alias）
	// application
	"pdf":  "application", // application/pdf
	"docx": "application", // application/vnd.openxmlformats-officedocument.wordprocessingml.document
	"xlsx": "application", // application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
	"zip":  "application", // application/zip
	// text
	"txt": "text", // text/plain
}

// extExtraCategories 是「二义容器格式」的额外容忍大类表（T-017）。
//
// 部分容器扩展名在 audio/video 两大类间天然二义（如 ogg 既可装音频流亦可装视频流），
// extCategory 只能取其一为「期望大类」，对另一合法大类会误拒（如 .ogg 被声明 video/ogg）。
// 此表给这类扩展名登记「主表大类之外仍应放行」的额外大类，ValidateContentType 在主表
// 大类相等之外再查此表。key 为小写无点扩展名，value 为额外容忍的顶层大类集合。
var extExtraCategories = map[string]map[string]bool{
	"ogg": {"video": true}, // extCategory 主表 ogg=audio，此处补容忍 video/ogg
}

// ValidateContentType 校验 Content-Type 与扩展名在「顶层大类」层一致。
//
// 确定性比较（不依赖 OS mime db）：
//   - 扩展名归一化后取期望大类；未知扩展名 → 放行（维持「未知不校验」既有契约）。
//   - 声明类型为空 → 放行（无声明不强拦，沿用宽容口径）。
//   - 声明类型为 application/octet-stream → 放行（浏览器通用兜底，非安全闸）。
//   - 否则比较声明类型的顶层大类与期望大类，跨大类才拒绝。
func ValidateContentType(contentType, ext string) error {
	ext = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(ext)), ".")
	wantCat, ok := extCategory[ext]
	if !ok {
		return nil // 未知扩展名不做大类校验
	}
	declared := strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0])
	if declared == "" {
		return nil // 无声明不强拦
	}
	if strings.EqualFold(declared, "application/octet-stream") {
		return nil // 浏览器「不认识」的通用兜底，放行
	}
	gotCat := strings.SplitN(strings.ToLower(declared), "/", 2)[0]
	if gotCat == wantCat || extExtraCategories[ext][gotCat] {
		return nil // 主表期望大类，或二义容器格式登记的额外容忍大类
	}
	return fmt.Errorf("content-type category mismatch: declared %q (category %q), expected category %q for .%s", declared, gotCat, wantCat, ext)
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
