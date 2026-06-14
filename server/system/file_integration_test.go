// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-004b 文件管理集成测试 — 真 MySQL + 真磁盘端到端
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 02:00:00
// +----------------------------------------------------------------------
//
// 运行方式：go test -tags=integration ./system/... -v -count=1 -run TestFile
// 前置：docker compose -f deploy/docker-compose.dev.yml up -d

//go:build integration

package system

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/drivers/storage"
	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const (
	fileDSN    = "root:root@tcp(localhost:3306)/benxinadminpro?charset=utf8mb4&parseTime=true&loc=Local"
	filePrefix = "t004b_"
)

func setupFileIntegration(t *testing.T) (*FileService, *storage.LocalDriver, *gorm.DB) {
	t.Helper()

	// 真 MySQL
	db, err := gorm.Open(mysql.Open(fileDSN), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{TablePrefix: filePrefix, SingularTable: true},
	})
	if err != nil {
		t.Fatalf("connect mysql: %v", err)
	}
	db.Exec("DROP TABLE IF EXISTS `" + filePrefix + "sys_file`")

	migDir := filepath.Join("..", "spec", "migrations")
	sqlBytes, _ := os.ReadFile(filepath.Join(migDir, "T004b_sys_file.sql"))
	ddl := strings.ReplaceAll(string(sqlBytes), "{{TABLE_PREFIX}}", filePrefix)
	if err := db.Exec(ddl).Error; err != nil {
		t.Fatalf("exec DDL: %v", err)
	}

	// 真磁盘
	diskRoot := t.TempDir()
	driver, err := storage.NewLocalDriver(diskRoot, "/sys/files/")
	if err != nil {
		t.Fatalf("NewLocalDriver: %v", err)
	}

	reg, _ := errcode.NewRegistry(11000)
	uploadCfg := storage.UploadConfig{
		MaxSizeBytes: 10 * 1024 * 1024,
		AllowedExts:  []string{"txt", "jpg", "png", "pdf"},
	}
	svc := NewFileService(db, driver, uploadCfg, reg, "local")

	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS `" + filePrefix + "sys_file`")
	})

	return svc, driver, db
}

func TestFileIntegration_UploadDownloadDeleteE2E(t *testing.T) {
	svc, _, db := setupFileIntegration(t)
	ctx := context.Background()

	content := []byte("hello-integration-test-file-content-中文内容")

	// 上传
	file, err := svc.Upload(ctx, "测试文件.txt", "text/plain", int64(len(content)),
		bytes.NewReader(content), "admin")
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if file.OriginalName != "测试文件.txt" {
		t.Errorf("original_name: %q", file.OriginalName)
	}
	if file.Ext != "txt" {
		t.Errorf("ext: %q", file.Ext)
	}
	if file.StorageType != "local" {
		t.Errorf("storage_type: %q", file.StorageType)
	}

	// 验证 DB 落库
	var dbFile SysFile
	if err := db.First(&dbFile, file.ID).Error; err != nil {
		t.Fatalf("DB lookup: %v", err)
	}
	if dbFile.StorageKey == "" {
		t.Error("storage_key should not be empty in DB")
	}

	// 下载 + 字节一致性
	gotFile, reader, err := svc.Download(ctx, file.ID)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	downloaded, _ := io.ReadAll(reader)
	reader.Close()

	if !bytes.Equal(downloaded, content) {
		t.Errorf("downloaded content mismatch: got %d bytes, want %d", len(downloaded), len(content))
	}
	if gotFile.OriginalName != "测试文件.txt" {
		t.Errorf("download original_name: %q", gotFile.OriginalName)
	}

	// 删除（软删 + 物理清理）
	if err := svc.Delete(ctx, file.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// DB 软删后查不到
	var count int64
	db.Model(&SysFile{}).Where("id = ?", file.ID).Count(&count)
	if count != 0 {
		t.Error("file should be soft-deleted")
	}
}

func TestFileIntegration_MultipleUploadsNoOverwrite(t *testing.T) {
	svc, _, _ := setupFileIntegration(t)
	ctx := context.Background()

	content := []byte("same content")

	f1, _ := svc.Upload(ctx, "same.txt", "text/plain", int64(len(content)),
		bytes.NewReader(content), "admin")
	f2, _ := svc.Upload(ctx, "same.txt", "text/plain", int64(len(content)),
		bytes.NewReader(content), "admin")

	if f1.StorageKey == f2.StorageKey {
		t.Error("two uploads of same filename should get different storage keys (uuid)")
	}
}

func TestFileIntegration_ListAndFilter(t *testing.T) {
	svc, _, _ := setupFileIntegration(t)
	ctx := context.Background()

	svc.Upload(ctx, "a.txt", "text/plain", 1, strings.NewReader("x"), "alice")
	svc.Upload(ctx, "b.txt", "text/plain", 1, strings.NewReader("x"), "bob")

	// 全部
	list, total, _ := svc.List(ctx, FileListQuery{Page: 1, PageSize: 10})
	if total != 2 {
		t.Errorf("total: %d", total)
	}
	_ = list

	// 按文件名模糊（uploader 用户名过滤需 sys_user，归 query_integration_test.go 覆盖）
	_, total, _ = svc.List(ctx, FileListQuery{OriginalName: "a.txt", Page: 1, PageSize: 10})
	if total != 1 {
		t.Errorf("filtered total: %d", total)
	}
}

func TestFileIntegration_PathTraversalOnRealDisk(t *testing.T) {
	svc, driver, _ := setupFileIntegration(t)
	ctx := context.Background()

	// 尝试穿越上传 — 文件名消毒应拒绝
	_, err := svc.Upload(ctx, "../../etc/passwd", "text/plain", 5,
		strings.NewReader("evil"), "attacker")
	if err == nil {
		t.Error("path traversal filename should be rejected by upload")
	}

	// 直接对驱动尝试穿越 key
	err = driver.Put(ctx, "../escape.txt", strings.NewReader("x"), 1, "text/plain")
	if err == nil {
		t.Error("driver should reject traversal key")
	}

	// 确认根目录外无文件
	parentDir := filepath.Dir(driver.GetRootDir())
	escaped := filepath.Join(parentDir, "escape.txt")
	if _, err := os.Stat(escaped); err == nil {
		t.Fatal("file was written outside root directory on real disk!")
	}
}
