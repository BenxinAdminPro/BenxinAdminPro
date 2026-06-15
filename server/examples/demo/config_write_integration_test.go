// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-005b-3 配置加密写链路 enforce + 全栈加密 e2e（dept_mgr 200 ↔ editor 403；密文经 API 落库可解）
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-15 10:30:00
// +----------------------------------------------------------------------
//
// 运行方式：go test -tags=integration ./examples/demo/ -v -count=1 -run TestConfigWrite
// 前置：docker compose -f deploy/docker-compose.dev.yml up -d
//
// 验证：① 写路径 enforce 正向：dept_mgr POST/PUT /sys/configs 全 200 ↔ editor 全 403。
//       ② 全栈加密：经 HTTP 新建 is_encrypted=1 → DB 落真密文（≠明文、可 DecryptGCM 还原）+ 列表恒 ******。
//       ③ 全栈编辑三态：留空 PUT（省 config_value）密文不变仍可解；填值 PUT 密文换新且解出新值。
// 隔离：独立表前缀 democw_ + Redis DB 11。
// 注：解密往返三态的单元级断言见 system/config_crypto_test.go（默认 go test 闸门），本测专证 HTTP 全栈 + enforce。

//go:build integration

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/crypto"
	"github.com/benxin_dev/benxinadminpro-server/rbac"
	"github.com/benxin_dev/benxinadminpro-server/system"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	cwEnfPrefix  = "democw_"
	cwEnfRedisDB = 11
)

func TestConfigWriteEnforceAndEncryptE2E(t *testing.T) {
	cfg := e2eConfig(t)
	cfg.TablePrefix = cwEnfPrefix
	cfg.RedisDB = cwEnfRedisDB

	ctx := context.Background()
	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), rbac.NewDBConfig(cfg.TablePrefix))
	if err != nil {
		t.Fatalf("connect mysql: %v", err)
	}
	dropPrefixedTables(t, db, cfg.TablePrefix)
	t.Cleanup(func() { dropPrefixedTables(t, db, cfg.TablePrefix) })

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, DB: cfg.RedisDB})
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("connect redis: %v", err)
	}
	rdb.FlushDB(ctx)
	t.Cleanup(func() { rdb.FlushDB(ctx); rdb.Close() })

	app, err := buildApp(cfg, db, rdb)
	if err != nil {
		t.Fatalf("buildApp: %v", err)
	}
	t.Cleanup(app.subCancel)
	ts := httptest.NewServer(app.handler)
	t.Cleanup(ts.Close)

	gcmKey, err := decodeB64(cfg.GCMKeyB64, 32, "gcm.key_b64")
	if err != nil {
		t.Fatalf("decode gcm key: %v", err)
	}

	mgrTok := loginTok(t, ts.URL, "dept_mgr", e2eDeptMgrPwd)
	editorTok := loginTok(t, ts.URL, "editor", e2eEditorPwd)

	rawValue := func(key string) system.SysConfig {
		var row system.SysConfig
		if err := db.Where("config_key = ?", key).First(&row).Error; err != nil {
			t.Fatalf("raw read %q: %v", key, err)
		}
		return row
	}

	// ---- ① 写路径 enforce 正向对照 ----
	// editor 无 sys:config:create → POST 403
	edC := doJSON(t, http.MethodPost, ts.URL+"/sys/configs", editorTok, map[string]any{
		"config_key": "cw.denied", "config_value": "x",
	})
	if edC.status != http.StatusForbidden {
		t.Errorf("editor POST /sys/configs: got %d, want 403", edC.status)
	}
	// dept_mgr 有全量菜单 → POST 200
	mgrC := doJSON(t, http.MethodPost, ts.URL+"/sys/configs", mgrTok, map[string]any{
		"config_key": "cw.plain", "config_value": "v1", "name": "明文",
	})
	if mgrC.status != http.StatusOK {
		t.Fatalf("dept_mgr POST /sys/configs: got %d %+v", mgrC.status, mgrC.body)
	}
	plainID, _ := dataMap(t, mgrC)["id"].(string)
	if plainID == "" {
		t.Fatal("create response missing id")
	}
	// dept_mgr PUT 200
	mgrU := doJSON(t, http.MethodPut, ts.URL+"/sys/configs/"+plainID, mgrTok, map[string]any{"name": "改名"})
	if mgrU.status != http.StatusOK {
		t.Errorf("dept_mgr PUT /sys/configs/%s: got %d %+v", plainID, mgrU.status, mgrU.body)
	}
	// editor PUT 403
	edU := doJSON(t, http.MethodPut, ts.URL+"/sys/configs/"+plainID, editorTok, map[string]any{"name": "y"})
	if edU.status != http.StatusForbidden {
		t.Errorf("editor PUT /sys/configs/%s: got %d, want 403", plainID, edU.status)
	}
	t.Log("enforce OK: 写路径 dept_mgr POST/PUT 200 ↔ editor POST/PUT 403")

	// ---- ② 全栈加密：经 HTTP 建加密参数 → DB 真密文 + 列表脱敏 ----
	const secret = "top-secret-v1"
	encC := doJSON(t, http.MethodPost, ts.URL+"/sys/configs", mgrTok, map[string]any{
		"config_key": "cw.secret", "config_value": secret, "name": "密钥", "is_encrypted": 1,
	})
	if encC.status != http.StatusOK {
		t.Fatalf("create encrypted via API: got %d %+v", encC.status, encC.body)
	}
	encID, _ := dataMap(t, encC)["id"].(string)

	row := rawValue("cw.secret")
	if row.IsEncrypted != 1 {
		t.Fatalf("is_encrypted should be 1, got %d", row.IsEncrypted)
	}
	if row.ConfigValue == secret {
		t.Fatal("API stored plaintext — encryption did not happen on write path")
	}
	plain, derr := crypto.DecryptGCM(gcmKey, row.ConfigValue)
	if derr != nil || string(plain) != secret {
		t.Fatalf("DB ciphertext not decryptable to original: err=%v got=%q", derr, string(plain))
	}
	// 列表恒脱敏 ******
	list := doJSON(t, http.MethodGet, ts.URL+"/sys/configs?config_key=cw.secret", mgrTok, nil)
	rows, _ := dataMap(t, list)["list"].([]any)
	foundMask := false
	for _, r := range rows {
		m, _ := r.(map[string]any)
		if m["config_key"] == "cw.secret" {
			if m["config_value"] != "******" {
				t.Errorf("encrypted row not masked in list: got %v", m["config_value"])
			}
			foundMask = true
		}
	}
	if !foundMask {
		t.Fatal("encrypted row not found in list")
	}
	t.Log("全栈加密 OK: HTTP 新建 is_encrypted=1 → DB 真密文（可解回原值）+ 列表恒 ******")

	// ---- ③ 全栈编辑三态 ----
	beforeKeep := rawValue("cw.secret").ConfigValue
	// 留空（省 config_value）→ 改 name，密文不动
	keepU := doJSON(t, http.MethodPut, ts.URL+"/sys/configs/"+encID, mgrTok, map[string]any{"name": "改名保持"})
	if keepU.status != http.StatusOK {
		t.Fatalf("keep update: got %d %+v", keepU.status, keepU.body)
	}
	afterKeep := rawValue("cw.secret")
	if afterKeep.ConfigValue != beforeKeep {
		t.Fatal("留空 PUT 破坏了密文 — DATA CORRUPTION")
	}
	if p, e := crypto.DecryptGCM(gcmKey, afterKeep.ConfigValue); e != nil || string(p) != secret {
		t.Fatalf("keep 后密文不可解回原值: err=%v got=%q", e, string(p))
	}
	if afterKeep.Name != "改名保持" {
		t.Fatalf("keep 后 name 未更新: %q", afterKeep.Name)
	}

	// 填新值 → 重新加密替换
	const secret2 = "top-secret-v2"
	repU := doJSON(t, http.MethodPut, ts.URL+"/sys/configs/"+encID, mgrTok, map[string]any{"config_value": secret2})
	if repU.status != http.StatusOK {
		t.Fatalf("replace update: got %d %+v", repU.status, repU.body)
	}
	afterRep := rawValue("cw.secret")
	if afterRep.ConfigValue == beforeKeep {
		t.Fatal("填值 PUT 未更换密文")
	}
	if p, e := crypto.DecryptGCM(gcmKey, afterRep.ConfigValue); e != nil || string(p) != secret2 {
		t.Fatalf("replace 后解出 %q, want %q (err=%v)", string(p), secret2, e)
	}
	t.Log("全栈编辑三态 OK: 留空保持密文(可解原值) / 填值重新加密(可解新值)")

	t.Log("ALL PASSED: T-005b-3 写路径 enforce + 全栈加密 + 编辑三态端到端")
}
