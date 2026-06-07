// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   StorageDriver 接口 — 文件存储抽象（本地/云扩展点）
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 01:10:00
// +----------------------------------------------------------------------

package storage

import (
	"context"
	"io"
)

// StorageDriver 定义文件存储的通用接口。
// 本地实现由底座提供；云驱动（OSS/S3/COS）由消费方实现注入。
// 底座不引入任何云厂商 SDK。
type StorageDriver interface {
	// Put 写入文件。key 是驱动内的相对路径（如 2026/06/08/uuid.jpg）。
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error

	// Get 读取文件，返回流。调用方负责 Close。
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete 删除文件。
	Delete(ctx context.Context, key string) error

	// Exists 检查文件是否存在。
	Exists(ctx context.Context, key string) (bool, error)

	// URL 返回文件的可访问地址。
	// 本地实现返回下载接口引用（非真实文件系统路径）。
	// 云实现返回 CDN/签名 URL。
	URL(ctx context.Context, key string) (string, error)
}
