// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   LocalDriver — 本地磁盘存储实现 + 路径穿越防护
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 01:12:00
// +----------------------------------------------------------------------

package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalDriver 本地磁盘存储实现。
// 根目录和 URL 前缀经构造注入，禁止硬编码。
type LocalDriver struct {
	rootDir   string // 存储根目录（绝对路径）
	urlPrefix string // 下载接口 URL 前缀（如 /sys/files/）
}

// NewLocalDriver 创建本地存储驱动。
// rootDir 必须是已存在的绝对路径；urlPrefix 指向下载接口（非文件系统路径）。
func NewLocalDriver(rootDir, urlPrefix string) (*LocalDriver, error) {
	abs, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("storage: resolve root dir: %w", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("storage: create root dir: %w", err)
	}
	return &LocalDriver{rootDir: abs, urlPrefix: urlPrefix}, nil
}

// safePath 解析 key 为根目录内的安全绝对路径。
// 路径穿越防护（头号安全要求）：
//   - filepath.Clean 消除 . 和 ..
//   - 校验结果仍在 rootDir 内（前缀检查）
//   - 拒绝包含 .. 的 key、绝对路径、控制字符
func (d *LocalDriver) safePath(key string) (string, error) {
	// 拒绝明显恶意输入
	if key == "" {
		return "", fmt.Errorf("storage: empty key")
	}
	if filepath.IsAbs(key) {
		return "", fmt.Errorf("storage: absolute path not allowed")
	}
	if strings.Contains(key, "..") {
		return "", fmt.Errorf("storage: path traversal detected")
	}
	for _, c := range key {
		if c < 32 { // 控制字符
			return "", fmt.Errorf("storage: control character in key")
		}
	}

	full := filepath.Join(d.rootDir, filepath.Clean(key))
	full, _ = filepath.Abs(full)

	// 最终路径必须在 rootDir 内
	if !strings.HasPrefix(full, d.rootDir+string(filepath.Separator)) && full != d.rootDir {
		return "", fmt.Errorf("storage: path traversal: resolved path outside root")
	}

	return full, nil
}

func (d *LocalDriver) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	path, err := d.safePath(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("storage: create dir: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("storage: create file: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("storage: write file: %w", err)
	}
	return nil
}

func (d *LocalDriver) Get(_ context.Context, key string) (io.ReadCloser, error) {
	path, err := d.safePath(key)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

// GetRootDir 返回存储根目录（仅供测试断言路径穿越防护，生产不应使用）。
func (d *LocalDriver) GetRootDir() string { return d.rootDir }

func (d *LocalDriver) Delete(_ context.Context, key string) error {
	path, err := d.safePath(key)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

func (d *LocalDriver) Exists(_ context.Context, key string) (bool, error) {
	path, err := d.safePath(key)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// URL 返回下载接口引用，不泄漏真实文件系统路径。
func (d *LocalDriver) URL(_ context.Context, key string) (string, error) {
	if _, err := d.safePath(key); err != nil {
		return "", err
	}
	return d.urlPrefix + key, nil
}

// 接口合规
var _ StorageDriver = (*LocalDriver)(nil)
