// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-011b 默认闸门单测 — mime 大类筛映射 / 批量软删幂等 / 批量删入参校验
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-16 11:55:00
// +----------------------------------------------------------------------
//
// 默认闸门（SQLite 内存 + 真磁盘 tmp 驱动），不需 MySQL；真库行为由 file_integration_test.go 兜底。

package system

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/drivers/storage"
	"github.com/glebarez/sqlite"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func setupFileSvc(t *testing.T) (*FileService, *storage.LocalDriver, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&SysFile{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	driver, err := storage.NewLocalDriver(t.TempDir(), "/sys/files/")
	if err != nil {
		t.Fatalf("NewLocalDriver: %v", err)
	}
	svc := NewFileService(db, driver, storage.UploadConfig{MaxSizeBytes: 1 << 20}, reg(t), "local")
	return svc, driver, db
}

// seedFile 直接落一行 sys_file（绕过 Upload 校验，只为构造 mime/删除场景），并在盘上写真实内容以便物理清理静默成功。
func seedFile(t *testing.T, db *gorm.DB, driver *storage.LocalDriver, name, mime, key string) uint64 {
	t.Helper()
	// uploader 留空：避开 resolveUserNames 查 sys_user（本测试不关心上传人可读化，real 解析由集成测试覆盖）。
	f := SysFile{OriginalName: name, StorageKey: key, StorageType: "local", Size: 1, Mime: mime, Ext: "bin"}
	if err := db.Create(&f).Error; err != nil {
		t.Fatalf("seed %s: %v", name, err)
	}
	if err := driver.Put(context.Background(), key, strings.NewReader("x"), 1, mime); err != nil {
		t.Fatalf("put %s: %v", key, err)
	}
	return f.ID
}

// ---------------------------------------------------------------------------
// mime 大类筛
// ---------------------------------------------------------------------------

func TestFileListMimeCategory(t *testing.T) {
	svc, driver, db := setupFileSvc(t)
	ctx := context.Background()
	seedFile(t, db, driver, "a.png", "image/png", "k/a")
	seedFile(t, db, driver, "b.jpg", "image/jpeg", "k/b")
	seedFile(t, db, driver, "c.mp4", "video/mp4", "k/c")
	seedFile(t, db, driver, "d.mp3", "audio/mpeg", "k/d")
	seedFile(t, db, driver, "e.pdf", "application/pdf", "k/e")
	seedFile(t, db, driver, "f.txt", "text/plain", "k/f")

	cases := []struct {
		category string
		want     int64
	}{
		{"image", 2}, // png + jpeg
		{"video", 1}, // mp4
		{"audio", 1}, // mpeg
		{"other", 2}, // pdf + txt（排除三大类）
		{"", 6},      // 空 → 不筛 = 全量
		{"bogus", 6}, // 未命中 token → 不筛 = 全量（防探测，不报错）
		{"image/%';DROP--", 6}, // 注入尝试 → 白名单未命中 → 不筛、不报错
	}
	for _, tc := range cases {
		_, total, _ := svc.List(ctx, FileListQuery{MimeCategory: tc.category, Page: 1, PageSize: 50})
		if total != tc.want {
			t.Errorf("mime_category=%q: total=%d, want %d", tc.category, total, tc.want)
		}
	}

	// 大类与文件名模糊可叠加：image 大类 + name 含 "a" → 仅 a.png
	_, total, _ := svc.List(ctx, FileListQuery{MimeCategory: "image", OriginalName: "a", Page: 1, PageSize: 50})
	if total != 1 {
		t.Errorf("image+name(a): total=%d, want 1", total)
	}
}

// ---------------------------------------------------------------------------
// 批量软删（service 层）
// ---------------------------------------------------------------------------

func TestBatchDeleteService(t *testing.T) {
	svc, driver, db := setupFileSvc(t)
	ctx := context.Background()
	id1 := seedFile(t, db, driver, "1.bin", "application/octet-stream", "k/1")
	id2 := seedFile(t, db, driver, "2.bin", "application/octet-stream", "k/2")
	id3 := seedFile(t, db, driver, "3.bin", "application/octet-stream", "k/3")

	// 批量软删 id1,id2 → deleted_count=2
	n, err := svc.BatchDelete(ctx, []uint64{id1, id2})
	if err != nil {
		t.Fatalf("BatchDelete: %v", err)
	}
	if n != 2 {
		t.Errorf("deleted_count: got %d, want 2", n)
	}

	// 软删行不再出现在 list（坐实软删 scope 生效，未踩 .Table 陷阱）
	_, total, _ := svc.List(ctx, FileListQuery{Page: 1, PageSize: 50})
	if total != 1 {
		t.Errorf("after batch delete, list total: got %d, want 1 (only id3)", total)
	}

	// 幂等：重复删 id1（已删）+ id2（已删）+ 不存在 id 99999 → deleted_count=0、不报错
	n, err = svc.BatchDelete(ctx, []uint64{id1, id2, 99999})
	if err != nil {
		t.Fatalf("idempotent BatchDelete: %v", err)
	}
	if n != 0 {
		t.Errorf("idempotent deleted_count: got %d, want 0", n)
	}

	// 混合：id3（存在）+ id1（已删）+ 不存在 → 仅 id3 命中 → deleted_count=1
	n, _ = svc.BatchDelete(ctx, []uint64{id3, id1, 88888})
	if n != 1 {
		t.Errorf("mixed deleted_count: got %d, want 1", n)
	}

	// 空入参 → 0、不报错
	n, err = svc.BatchDelete(ctx, nil)
	if err != nil || n != 0 {
		t.Errorf("empty BatchDelete: n=%d err=%v, want 0,nil", n, err)
	}
}

// ---------------------------------------------------------------------------
// decodeHashSlice fail-fast（nil hasher 走裸十进制）
// ---------------------------------------------------------------------------

func TestDecodeHashSlice(t *testing.T) {
	// nil hasher → 裸十进制解析
	ids, err := decodeHashSlice(nil, []string{"1", "2", "30"})
	if err != nil {
		t.Fatalf("decode decimal: %v", err)
	}
	if len(ids) != 3 || ids[0] != 1 || ids[2] != 30 {
		t.Errorf("decode decimal: %v", ids)
	}
	// 任一非法 → fail-fast 返错
	if _, err := decodeHashSlice(nil, []string{"1", "abc", "3"}); err == nil {
		t.Error("expected error on invalid id 'abc', got nil")
	}
}

// ---------------------------------------------------------------------------
// 批量删 handler 入参校验（写前 fail-fast：空/超限/非法 hashid → 400）
// ---------------------------------------------------------------------------

func TestBatchDeleteHandlerValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// nil hasher（裸解析）+ nil svc：400 路径在 service 前返回、不触达 svc
	h := NewFileHandler(nil, reg(t), nil)

	post := func(body string) int {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/sys/files/batch-delete", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		h.BatchDelete(c)
		return w.Code
	}

	if code := post(`{"ids":[]}`); code != http.StatusBadRequest {
		t.Errorf("empty ids: got %d, want 400", code)
	}
	if code := post(`{}`); code != http.StatusBadRequest {
		t.Errorf("missing ids: got %d, want 400", code)
	}
	// 超过封顶 100
	over := `{"ids":[` + strings.Repeat(`"1",`, 100) + `"1"]}` // 101 个
	if code := post(over); code != http.StatusBadRequest {
		t.Errorf("over cap(101): got %d, want 400", code)
	}
	// 非法 hashid（nil hasher 下 "abc" 非十进制 → 解码失败 → 400 ErrInvalidID）
	if code := post(`{"ids":["1","abc"]}`); code != http.StatusBadRequest {
		t.Errorf("invalid hashid: got %d, want 400", code)
	}
	// 注：合法 ≤100 的快乐路径会触达 service，由 TestBatchDeleteService 覆盖（此处 svc=nil 不重复）。
}
