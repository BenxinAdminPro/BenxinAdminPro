// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-005b-3 配置加密写链路单测 — 解密往返三态 + 明文零回归 + 密钥缺失降级
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-15 10:30:00
// +----------------------------------------------------------------------
//
// 因对外 List/GetByKey 恒 mask，密文是否被破坏无法从对外路径观察，
// 唯有「ConfigService 写入 → ConfigCenter.GetConfig 解密读出」往返可证。
// 写路径逻辑（GCM 加密 + 指针三态）SQLite 内存库即可覆盖，纳入默认 go test 闸门。

package system

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
)

// newConfigSvcWithKey 造一个注入了 GCM 密钥的 ConfigService（SQLite 内存库）。
func newConfigSvcWithKey(t *testing.T) (*ConfigService, *ConfigCenter, context.Context) {
	t.Helper()
	db := setupDB(t)
	svc := NewConfigService(db, reg(t))
	svc.SetGCMKey(testGCMKey)
	// 读路径：nil cache → 恒回源 DB 并自动解密
	center := NewConfigCenter(db, nil, nil, CenterConfig{GCMKey: testGCMKey})
	return svc, center, context.Background()
}

// rawConfigValue 直接从 DB 取该 key 的原始 config_value（绕过 mask，看真落库内容）。
func rawConfigValue(t *testing.T, svc *ConfigService, key string) (string, int8) {
	t.Helper()
	var row SysConfig
	if err := svc.db.Where("config_key = ?", key).First(&row).Error; err != nil {
		t.Fatalf("raw read %q: %v", key, err)
	}
	return row.ConfigValue, row.IsEncrypted
}

// 三态①：新建加密参数 → GetConfig 解出 == 原明文，且落库非明文字面。
func TestConfigCrypto_CreateEncrypted_RoundTrip(t *testing.T) {
	svc, center, ctx := newConfigSvcWithKey(t)
	const plain = "s3cr3t-create"

	if _, err := svc.Create(ctx, CreateConfigInput{ConfigKey: "k.enc", ConfigValue: plain, IsEncrypted: 1}); err != nil {
		t.Fatalf("create encrypted: %v", err)
	}

	// 落库必是密文而非明文字面
	raw, isEnc := rawConfigValue(t, svc, "k.enc")
	if isEnc != 1 {
		t.Fatalf("is_encrypted should be 1, got %d", isEnc)
	}
	if raw == plain {
		t.Fatal("stored value is plaintext — encryption did not happen")
	}
	if _, err := base64.StdEncoding.DecodeString(raw); err != nil {
		t.Fatalf("stored value not valid base64 envelope: %v", err)
	}

	// 读路径解出原明文
	got, err := center.GetConfig(ctx, "k.enc")
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if got != plain {
		t.Fatalf("decrypted %q, want %q", got, plain)
	}
}

// 三态②：Update 带新值 → GetConfig 解出 == 新明文，落库为合法密文。
func TestConfigCrypto_UpdateEncrypted_ReplaceValue(t *testing.T) {
	svc, center, ctx := newConfigSvcWithKey(t)
	created, err := svc.Create(ctx, CreateConfigInput{ConfigKey: "k.enc", ConfigValue: "old-secret", IsEncrypted: 1})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	oldRaw, _ := rawConfigValue(t, svc, "k.enc")

	const newVal = "new-secret"
	v := newVal
	if err := svc.Update(ctx, created.ID, UpdateConfigInput{ConfigValue: &v, Name: "改名"}); err != nil {
		t.Fatalf("update with new value: %v", err)
	}

	newRaw, isEnc := rawConfigValue(t, svc, "k.enc")
	if isEnc != 1 {
		t.Fatalf("is_encrypted must remain 1, got %d", isEnc)
	}
	if newRaw == newVal {
		t.Fatal("new value stored as plaintext — re-encryption did not happen")
	}
	if newRaw == oldRaw {
		t.Fatal("ciphertext unchanged after replace")
	}

	got, err := center.GetConfig(ctx, "k.enc")
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if got != newVal {
		t.Fatalf("decrypted %q, want %q", got, newVal)
	}
}

// 三态③（命门）：Update 不带值（nil）→ 只改 name/remark，密文原样保留、仍可解出原明文。
func TestConfigCrypto_UpdateEncrypted_KeepValueWhenNil(t *testing.T) {
	svc, center, ctx := newConfigSvcWithKey(t)
	const plain = "keep-me"
	created, err := svc.Create(ctx, CreateConfigInput{ConfigKey: "k.enc", ConfigValue: plain, IsEncrypted: 1, Name: "原名"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	rawBefore, _ := rawConfigValue(t, svc, "k.enc")

	// ConfigValue 为 nil（前端留空省略字段），只改 name
	if err := svc.Update(ctx, created.ID, UpdateConfigInput{ConfigValue: nil, Name: "新名", Remark: "备注"}); err != nil {
		t.Fatalf("update keep: %v", err)
	}

	rawAfter, _ := rawConfigValue(t, svc, "k.enc")
	if rawAfter != rawBefore {
		t.Fatal("ciphertext was overwritten on nil-value update — DATA CORRUPTION")
	}

	got, err := center.GetConfig(ctx, "k.enc")
	if err != nil {
		t.Fatalf("GetConfig after keep: %v", err)
	}
	if got != plain {
		t.Fatalf("decrypted %q after keep, want original %q", got, plain)
	}

	// name 确实更新了
	var row SysConfig
	svc.db.First(&row, created.ID)
	if row.Name != "新名" {
		t.Fatalf("name not updated, got %q", row.Name)
	}
}

// 明文零回归：明文 create 明文存；update 带值明文覆写；update nil 保持。
func TestConfigCrypto_PlaintextNoRegression(t *testing.T) {
	svc, _, ctx := newConfigSvcWithKey(t)
	created, err := svc.Create(ctx, CreateConfigInput{ConfigKey: "k.plain", ConfigValue: "v1", IsEncrypted: 0})
	if err != nil {
		t.Fatalf("create plaintext: %v", err)
	}
	if raw, isEnc := rawConfigValue(t, svc, "k.plain"); raw != "v1" || isEnc != 0 {
		t.Fatalf("plaintext create: raw=%q isEnc=%d", raw, isEnc)
	}

	// 带值覆写
	v := "v2"
	if err := svc.Update(ctx, created.ID, UpdateConfigInput{ConfigValue: &v}); err != nil {
		t.Fatalf("update plaintext value: %v", err)
	}
	if raw, _ := rawConfigValue(t, svc, "k.plain"); raw != "v2" {
		t.Fatalf("plaintext update value: raw=%q want v2", raw)
	}

	// nil 保持
	if err := svc.Update(ctx, created.ID, UpdateConfigInput{ConfigValue: nil, Name: "n"}); err != nil {
		t.Fatalf("update plaintext keep: %v", err)
	}
	if raw, _ := rawConfigValue(t, svc, "k.plain"); raw != "v2" {
		t.Fatalf("plaintext keep: raw=%q want v2 unchanged", raw)
	}
}

// 密钥缺失降级：is_encrypted=1 写入但未注入 gcmKey → 返 ErrConfigDecryptFailed，不 panic。
func TestConfigCrypto_NoKey_ReturnsErrNoPanic(t *testing.T) {
	db := setupDB(t)
	r := reg(t)
	svc := NewConfigService(db, r) // 未 SetGCMKey
	ctx := context.Background()

	_, err := svc.Create(ctx, CreateConfigInput{ConfigKey: "k.enc", ConfigValue: "x", IsEncrypted: 1})
	if !errors.Is(err, r.ErrConfigDecryptFailed) {
		t.Fatalf("create encrypted without key: got %v, want ErrConfigDecryptFailed", err)
	}

	// 明文参数无 key 仍正常
	if _, err := svc.Create(ctx, CreateConfigInput{ConfigKey: "k.plain", ConfigValue: "x", IsEncrypted: 0}); err != nil {
		t.Fatalf("plaintext create without key should work: %v", err)
	}

	// 加密行 Update 带新值无 key 也应返错（先造一条加密行绕过加密：直接写库）
	db.Create(&SysConfig{ConfigKey: "k.enc2", ConfigValue: "ciphertext-placeholder", IsEncrypted: 1})
	var row SysConfig
	db.Where("config_key = ?", "k.enc2").First(&row)
	v := "new"
	if err := svc.Update(ctx, row.ID, UpdateConfigInput{ConfigValue: &v}); !errors.Is(err, r.ErrConfigDecryptFailed) {
		t.Fatalf("update encrypted without key: got %v, want ErrConfigDecryptFailed", err)
	}
}
