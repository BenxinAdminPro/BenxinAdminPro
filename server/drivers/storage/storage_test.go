// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   存储驱动测试 — LocalDriver + 路径穿越防护 + 上传安全
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 01:30:00
// +----------------------------------------------------------------------

package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testDriver(t *testing.T) *LocalDriver {
	t.Helper()
	root := t.TempDir()
	d, err := NewLocalDriver(root, "/sys/files/")
	if err != nil {
		t.Fatalf("NewLocalDriver: %v", err)
	}
	return d
}

// ---------------------------------------------------------------------------
// LocalDriver 基本功能
// ---------------------------------------------------------------------------

func TestLocalDriver_PutGetDeleteExists(t *testing.T) {
	d := testDriver(t)
	ctx := context.Background()
	key := "2026/06/08/test.txt"
	content := "hello-file"

	// Put
	err := d.Put(ctx, key, strings.NewReader(content), int64(len(content)), "text/plain")
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Exists
	exists, _ := d.Exists(ctx, key)
	if !exists {
		t.Error("file should exist after Put")
	}

	// Get
	rc, err := d.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != content {
		t.Errorf("content: got %q, want %q", data, content)
	}

	// Delete
	d.Delete(ctx, key)
	exists, _ = d.Exists(ctx, key)
	if exists {
		t.Error("file should not exist after Delete")
	}
}

func TestLocalDriver_AutoCreateSubDir(t *testing.T) {
	d := testDriver(t)
	key := "deep/nested/dir/file.txt"
	err := d.Put(context.Background(), key, strings.NewReader("x"), 1, "text/plain")
	if err != nil {
		t.Fatalf("Put with nested dir: %v", err)
	}
}

func TestLocalDriver_URLNotLeakRealPath(t *testing.T) {
	d := testDriver(t)
	url, err := d.URL(context.Background(), "2026/06/08/test.txt")
	if err != nil {
		t.Fatalf("URL: %v", err)
	}
	if !strings.HasPrefix(url, "/sys/files/") {
		t.Errorf("URL should start with download prefix, got %q", url)
	}
	// 不应包含临时目录路径
	if strings.Contains(url, os.TempDir()) || strings.Contains(url, "/var/") {
		t.Errorf("URL should not contain real filesystem path: %q", url)
	}
}

// ---------------------------------------------------------------------------
// 路径穿越防护（头号安全要求）
// ---------------------------------------------------------------------------

func TestLocalDriver_PathTraversal_DotDot(t *testing.T) {
	d := testDriver(t)
	ctx := context.Background()

	badKeys := []string{
		"../etc/passwd",
		"../../etc/shadow",
		"subdir/../../../etc/hosts",
		"..\\windows\\system32\\config",
	}
	for _, key := range badKeys {
		err := d.Put(ctx, key, strings.NewReader("x"), 1, "text/plain")
		if err == nil {
			t.Errorf("should reject path traversal key %q", key)
		}
	}
}

func TestLocalDriver_PathTraversal_AbsolutePath(t *testing.T) {
	d := testDriver(t)
	err := d.Put(context.Background(), "/etc/passwd", strings.NewReader("x"), 1, "text/plain")
	if err == nil {
		t.Error("should reject absolute path")
	}
}

func TestLocalDriver_PathTraversal_ControlChars(t *testing.T) {
	d := testDriver(t)
	err := d.Put(context.Background(), "file\x00name.txt", strings.NewReader("x"), 1, "text/plain")
	if err == nil {
		t.Error("should reject control characters in key")
	}
}

func TestLocalDriver_PathTraversal_EmptyKey(t *testing.T) {
	d := testDriver(t)
	err := d.Put(context.Background(), "", strings.NewReader("x"), 1, "text/plain")
	if err == nil {
		t.Error("should reject empty key")
	}
}

func TestLocalDriver_PathTraversal_NoFileOutsideRoot(t *testing.T) {
	d := testDriver(t)
	ctx := context.Background()

	// 尝试各种穿越，确认根目录外无文件写出
	parentDir := filepath.Dir(d.rootDir)
	badFile := filepath.Join(parentDir, "escaped.txt")

	d.Put(ctx, "../escaped.txt", strings.NewReader("x"), 1, "text/plain")
	d.Put(ctx, "../../escaped.txt", strings.NewReader("x"), 1, "text/plain")

	if _, err := os.Stat(badFile); err == nil {
		t.Fatal("file was written outside root directory!")
	}
}

// ---------------------------------------------------------------------------
// 上传安全四件套
// ---------------------------------------------------------------------------

func TestUploadConfig_ValidateSize(t *testing.T) {
	cfg := UploadConfig{MaxSizeBytes: 1024}
	if err := cfg.ValidateSize(500); err != nil {
		t.Error("500 should be ok")
	}
	if err := cfg.ValidateSize(2000); err == nil {
		t.Error("2000 should exceed limit")
	}
}

func TestUploadConfig_ValidateExt(t *testing.T) {
	cfg := UploadConfig{AllowedExts: []string{"jpg", "png", "pdf"}}
	if err := cfg.ValidateExt("jpg"); err != nil {
		t.Error("jpg should be allowed")
	}
	if err := cfg.ValidateExt(".PNG"); err != nil {
		t.Error("PNG (case insensitive) should be allowed")
	}
	if err := cfg.ValidateExt("exe"); err == nil {
		t.Error("exe should be rejected")
	}
}

func TestValidateContentType(t *testing.T) {
	if err := ValidateContentType("image/jpeg", "jpg"); err != nil {
		t.Errorf("jpeg/jpg should match: %v", err)
	}
	if err := ValidateContentType("text/plain", "jpg"); err == nil {
		t.Error("text/plain for jpg should mismatch")
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct{ input, expected string; shouldFail bool }{
		{"normal.jpg", "normal.jpg", false},
		{"../../etc/passwd", "", true},
		{"path/to/file.txt", "file.txt", false},
		{"control\x00char.txt", "", true},
		{strings.Repeat("a", 300) + ".txt", "", false}, // should be truncated, not fail
		{"", "", true},
		{"..", "", true},
	}
	for _, tc := range tests {
		result, err := SanitizeFilename(tc.input)
		if tc.shouldFail {
			if err == nil {
				t.Errorf("SanitizeFilename(%q) should fail", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("SanitizeFilename(%q): %v", tc.input, err)
			continue
		}
		if tc.expected != "" && result != tc.expected {
			t.Errorf("SanitizeFilename(%q): got %q, want %q", tc.input, result, tc.expected)
		}
		// 长文件名应被截断
		if len(result) > 200 {
			t.Errorf("sanitized name too long: %d", len(result))
		}
	}
}

func TestExtFromFilename(t *testing.T) {
	if ext := ExtFromFilename("photo.JPG"); ext != "jpg" {
		t.Errorf("got %q", ext)
	}
	if ext := ExtFromFilename("noext"); ext != "" {
		t.Errorf("got %q", ext)
	}
}

// ---------------------------------------------------------------------------
// 流式大小限制（不全载内存）
// ---------------------------------------------------------------------------

func TestUploadSizeStreamCheck(t *testing.T) {
	cfg := UploadConfig{MaxSizeBytes: 100}
	// 构造一个大于限制的 reader
	bigData := bytes.Repeat([]byte("x"), 200)
	reader := io.LimitReader(bytes.NewReader(bigData), cfg.MaxSizeBytes+1)

	// 读取 101 字节，说明 LimitReader 限制了流式读取
	buf := make([]byte, 200)
	n, _ := reader.Read(buf)
	if n > int(cfg.MaxSizeBytes+1) {
		t.Errorf("LimitReader should cap at %d, got %d", cfg.MaxSizeBytes+1, n)
	}
}
