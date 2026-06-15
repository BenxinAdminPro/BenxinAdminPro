// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-005b-3 seed 加密样例幂等 — 连跑两遍 sys_config 行数稳定、加密样例落真密文
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-15 10:30:00
// +----------------------------------------------------------------------
//
// 运行方式：go test -tags=integration ./examples/demo/ -v -count=1 -run TestSeedConfigIdempotent
// 前置：docker compose -f deploy/docker-compose.dev.yml up -d
// 隔离：独立表前缀 democs_ + Redis DB 12。

//go:build integration

package main

import (
	"context"
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
	csSeedPrefix  = "democs_"
	csSeedRedisDB = 12
)

func TestSeedConfigIdempotent(t *testing.T) {
	cfg := e2eConfig(t)
	cfg.TablePrefix = csSeedPrefix
	cfg.RedisDB = csSeedRedisDB

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

	gcmKey, err := decodeB64(cfg.GCMKeyB64, 32, "gcm.key_b64")
	if err != nil {
		t.Fatalf("decode gcm key: %v", err)
	}

	countConfigs := func() int64 {
		var n int64
		db.Model(&system.SysConfig{}).Count(&n)
		return n
	}

	// 第一遍：buildApp 跑 migrate + seed
	app1, err := buildApp(cfg, db, rdb)
	if err != nil {
		t.Fatalf("buildApp #1: %v", err)
	}
	ts1 := httptest.NewServer(app1.handler)
	first := countConfigs()
	if first < 2 {
		t.Fatalf("第一遍后 sys_config 应至少 2 行（明文+加密样例），got %d", first)
	}
	app1.subCancel()
	ts1.Close()

	// 加密样例确为真密文（可解）
	var enc system.SysConfig
	if err := db.Where("config_key = ?", "demo.secret_token").First(&enc).Error; err != nil {
		t.Fatalf("加密样例未种入: %v", err)
	}
	if enc.IsEncrypted != 1 {
		t.Fatalf("加密样例 is_encrypted 应为 1, got %d", enc.IsEncrypted)
	}
	if p, e := crypto.DecryptGCM(gcmKey, enc.ConfigValue); e != nil {
		t.Fatalf("加密样例落库非合法密文: %v", e)
	} else if string(p) != "s3cr3t-demo-token" {
		t.Fatalf("加密样例解出 %q, want s3cr3t-demo-token", string(p))
	}
	firstCipher := enc.ConfigValue

	// 第二遍：同库再跑 migrate + seed，行数应稳定、加密样例不被重新加密覆盖
	app2, err := buildApp(cfg, db, rdb)
	if err != nil {
		t.Fatalf("buildApp #2: %v", err)
	}
	ts2 := httptest.NewServer(app2.handler)
	second := countConfigs()
	app2.subCancel()
	ts2.Close()

	if second != first {
		t.Fatalf("seed 非幂等：第一遍 %d 行，第二遍 %d 行", first, second)
	}
	var enc2 system.SysConfig
	db.Where("config_key = ?", "demo.secret_token").First(&enc2)
	if enc2.ConfigValue != firstCipher {
		t.Fatal("第二遍 seed 重新加密覆盖了加密样例（应跳过已存在行）")
	}
	t.Logf("seed 幂等 OK: 连跑两遍 sys_config 行数稳定 = %d；加密样例为真密文且二遍不被覆盖", second)
}
