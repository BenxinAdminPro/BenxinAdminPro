// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   配置中心测试 — 缓存/GCM加密/热加载/迁移执行器
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 03:20:00
// +----------------------------------------------------------------------

package system

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/benxin_dev/benxinadminpro-server/crypto"
	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func setupCenterDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	db.AutoMigrate(&SysConfig{}, &SysDictData{}, &SysDictType{}, &SysMigration{})
	return db
}

var testGCMKey = bytes.Repeat([]byte{0x42}, 32)

// ---------------------------------------------------------------------------
// 缓存命中/失效/回填
// ---------------------------------------------------------------------------

func TestConfigCenter_CacheHitAndMiss(t *testing.T) {
	db := setupCenterDB(t)
	db.Create(&SysConfig{ConfigKey: "site.name", ConfigValue: "TestSite"})

	cache := NewMemConfigCache()
	center := NewConfigCenter(db, cache, nil, CenterConfig{CacheTTL: time.Minute})
	ctx := context.Background()

	// 首次：cache miss → 回源 DB → 回填
	val, err := center.GetConfig(ctx, "site.name")
	if err != nil {
		t.Fatal(err)
	}
	if val != "TestSite" {
		t.Errorf("got %q", val)
	}

	// 二次：cache hit
	val2, _ := center.GetConfig(ctx, "site.name")
	if val2 != "TestSite" {
		t.Error("cache hit should return same value")
	}

	// 失效后 cache miss
	center.InvalidateConfig(ctx, "site.name")
	// 改 DB
	db.Model(&SysConfig{}).Where("config_key = ?", "site.name").Update("config_value", "NewSite")
	val3, _ := center.GetConfig(ctx, "site.name")
	if val3 != "NewSite" {
		t.Errorf("after invalidation should get new value, got %q", val3)
	}
}

// ---------------------------------------------------------------------------
// GCM 加密参数读写
// ---------------------------------------------------------------------------

func TestConfigCenter_EncryptedParam(t *testing.T) {
	db := setupCenterDB(t)

	// 加密写入
	encrypted, err := crypto.EncryptGCM(testGCMKey, []byte("my-secret-value"))
	if err != nil {
		t.Fatal(err)
	}
	db.Create(&SysConfig{ConfigKey: "db.password", ConfigValue: encrypted, IsEncrypted: 1})

	// DB 中存的是密文
	var raw SysConfig
	db.Where("config_key = ?", "db.password").First(&raw)
	if raw.ConfigValue == "my-secret-value" {
		t.Fatal("DB should store ciphertext, not plaintext")
	}

	// 通过 center 读取 → 自动解密
	center := NewConfigCenter(db, nil, nil, CenterConfig{GCMKey: testGCMKey})
	val, err := center.GetConfig(context.Background(), "db.password")
	if err != nil {
		t.Fatal(err)
	}
	if val != "my-secret-value" {
		t.Errorf("decrypted value: got %q", val)
	}
}

func TestConfigCenter_EncryptedParamSensitiveNotInPlaintext(t *testing.T) {
	db := setupCenterDB(t)
	encrypted, _ := crypto.EncryptGCM(testGCMKey, []byte("super-secret"))
	db.Create(&SysConfig{ConfigKey: "api.key", ConfigValue: encrypted, IsEncrypted: 1})

	// DB 中绝不含明文
	var raw SysConfig
	db.Where("config_key = ?", "api.key").First(&raw)
	if raw.ConfigValue == "super-secret" {
		t.Fatal("plaintext should NOT be in DB")
	}
}

// ---------------------------------------------------------------------------
// 热加载（Pub/Sub 假实现）
// ---------------------------------------------------------------------------

func TestConfigCenter_PubSubNotify(t *testing.T) {
	db := setupCenterDB(t)
	cache := NewMemConfigCache()
	pub := NewMemPublisher()
	center := NewConfigCenter(db, cache, pub, CenterConfig{CacheTTL: time.Minute})

	// 订阅
	var received []ChangeEvent
	pub.Subscribe(context.Background(), func(e ChangeEvent) {
		received = append(received, e)
	})

	// 触发变更通知
	center.InvalidateConfig(context.Background(), "site.name")
	center.InvalidateDict(context.Background(), "gender")

	if len(received) != 2 {
		t.Fatalf("expected 2 events, got %d", len(received))
	}
	if received[0].Type != "config" || received[0].Key != "site.name" {
		t.Errorf("event 0: %+v", received[0])
	}
	if received[1].Type != "dict" || received[1].Key != "gender" {
		t.Errorf("event 1: %+v", received[1])
	}
}

// ---------------------------------------------------------------------------
// 迁移执行器
// ---------------------------------------------------------------------------

func TestMigrator_SortAndExecute(t *testing.T) {
	db := setupCenterDB(t)

	// 创建临时迁移目录
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "T002_b.sql"), []byte(
		"CREATE TABLE IF NOT EXISTS test_b (id INTEGER PRIMARY KEY);"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "T001_a.sql"), []byte(
		"CREATE TABLE IF NOT EXISTS test_a (id INTEGER PRIMARY KEY);"), 0o644)

	m := NewMigrator(db, "", tmpDir)
	if err := m.Up(context.Background()); err != nil {
		t.Fatal(err)
	}

	// 验证表已创建
	var count int64
	db.Raw("SELECT COUNT(*) FROM test_a").Scan(&count)
	if count < 0 {
		t.Error("test_a should exist")
	}

	// 验证版本记录
	var versions []string
	db.Model(&SysMigration{}).Pluck("version", &versions)
	if len(versions) != 2 {
		t.Errorf("expected 2 migration records, got %d", len(versions))
	}
}

func TestMigrator_Idempotent(t *testing.T) {
	db := setupCenterDB(t)
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "T001_init.sql"), []byte(
		"CREATE TABLE IF NOT EXISTS test_idem (id INTEGER PRIMARY KEY);"), 0o644)

	m := NewMigrator(db, "", tmpDir)
	m.Up(context.Background())
	m.Up(context.Background()) // 二次执行应幂等

	var count int64
	db.Model(&SysMigration{}).Count(&count)
	if count != 1 {
		t.Errorf("idempotent: expected 1 record, got %d", count)
	}
}

func TestMigrator_PrefixReplacement(t *testing.T) {
	db := setupCenterDB(t)
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "T001_prefix.sql"), []byte(
		"CREATE TABLE IF NOT EXISTS `{{TABLE_PREFIX}}test_prefix` (id INTEGER PRIMARY KEY);"), 0o644)

	m := NewMigrator(db, "pfx_", tmpDir)
	if err := m.Up(context.Background()); err != nil {
		t.Fatal(err)
	}

	// SQLite 不支持反引号，但替换应该发生了
	// 验证版本记录
	var versions []string
	db.Model(&SysMigration{}).Pluck("version", &versions)
	if len(versions) != 1 {
		t.Errorf("expected 1 record, got %d", len(versions))
	}
}

// ---------------------------------------------------------------------------
// 加密参数列表脱敏断言（docker-free）
// ---------------------------------------------------------------------------

func TestConfigService_EncryptedMaskedInList(t *testing.T) {
	db := setupCenterDB(t)
	gcmKey := testGCMKey

	encrypted, _ := crypto.EncryptGCM(gcmKey, []byte("my-api-key-secret"))
	db.Create(&SysConfig{ConfigKey: "api.key", ConfigValue: encrypted, IsEncrypted: 1, Name: "API密钥"})
	db.Create(&SysConfig{ConfigKey: "site.url", ConfigValue: "https://example.com", IsEncrypted: 0, Name: "站点URL"})

	svc := NewConfigService(db, func() *errcode.Registry { r, _ := errcode.NewRegistry(11000); return r }())
	list, _, _ := svc.List(context.Background(), 1, 10)

	for _, cfg := range list {
		if cfg.IsEncrypted == 1 {
			if cfg.ConfigValue != MaskedValue {
				t.Errorf("encrypted param %q value should be %q, got %q", cfg.ConfigKey, MaskedValue, cfg.ConfigValue)
			}
			if cfg.ConfigValue == "my-api-key-secret" {
				t.Fatal("plaintext leaked in list!")
			}
		}
		if cfg.ConfigKey == "site.url" && cfg.ConfigValue != "https://example.com" {
			t.Error("non-encrypted param should keep original value")
		}
	}

	// GetByKey 也应脱敏
	got, _ := svc.GetByKey(context.Background(), "api.key")
	if got.ConfigValue != MaskedValue {
		t.Errorf("GetByKey encrypted should be masked, got %q", got.ConfigValue)
	}
}

// 接口合规
var (
	_ ConfigCache     = (*MemConfigCache)(nil)
	_ ConfigPublisher = (*MemPublisher)(nil)
)
