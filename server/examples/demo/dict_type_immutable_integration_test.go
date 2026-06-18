// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-017 项① dict_type 真禁改 HTTP e2e — PUT 带变更 dict_type 应被忽略（库内不变、name 已更新）
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-18 13:50:56
// +----------------------------------------------------------------------
//
// 运行方式：go test -tags=integration ./examples/demo/ -v -count=1 -run TestDictTypeImmutable
// 前置：docker compose -f deploy/docker-compose.dev.yml up -d
//
// 验证：dict_type 为唯一键禁改——POST 建类型后 PUT 一个「变更后的 dict_type + 新 name」，
//       断言 200，且库内该行 dict_type 原样不变、name 已更新、不存在以新 dict_type 命名的行
//       （即后端物理忽略 dict_type，杜绝 curl 绕过前端 editable:false 改键 → 孤儿化既有 dict_data）。
// 隔离：独立表前缀 demodt_ + Redis DB 12。

//go:build integration

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/rbac"
	"github.com/benxin_dev/benxinadminpro-server/system"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	dtPrefix  = "demodt_"
	dtRedisDB = 12
)

func TestDictTypeImmutableE2E(t *testing.T) {
	cfg := e2eConfig(t)
	cfg.TablePrefix = dtPrefix
	cfg.RedisDB = dtRedisDB

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

	adminTok := loginTok(t, ts.URL, "admin", e2eAdminPwd)

	const origType = "t017_orig"
	const changedType = "t017_changed"

	// ① 建字典类型（取回 hashid id 供 PUT）
	create := doJSON(t, http.MethodPost, ts.URL+"/sys/dict/types", adminTok, map[string]any{
		"dict_type": origType, "name": "原名", "status": 0,
	})
	if create.status != http.StatusOK {
		t.Fatalf("create dict type: got %d %+v", create.status, create.body)
	}
	id, _ := dataMap(t, create)["id"].(string)
	if id == "" {
		t.Fatalf("create dict type: 出参缺 hashid id，body=%+v", create.body)
	}

	// ② PUT 带「变更后的 dict_type + 新 name」——dict_type 应被后端忽略，name 应更新
	upd := doJSON(t, http.MethodPut, ts.URL+"/sys/dict/types/"+id, adminTok, map[string]any{
		"dict_type": changedType, "name": "新名", "status": 1, "remark": "r",
	})
	if upd.status != http.StatusOK {
		t.Fatalf("update dict type: got %d %+v", upd.status, upd.body)
	}

	// ③ 库内断言：原 dict_type 行仍在、name 已更新；不存在以变更后 dict_type 命名的行
	var row system.SysDictType
	if err := db.Model(&system.SysDictType{}).Where("dict_type = ?", origType).First(&row).Error; err != nil {
		t.Fatalf("原 dict_type=%q 行应仍在（dict_type 禁改），却查不到：%v", origType, err)
	}
	if row.Name != "新名" {
		t.Errorf("name 应已更新为「新名」，实际 %q（非 dict_type 字段更新被误吞）", row.Name)
	}

	var changedCount int64
	db.Model(&system.SysDictType{}).Where("dict_type = ?", changedType).Count(&changedCount)
	if changedCount != 0 {
		t.Errorf("dict_type 被改成 %q（禁改失守）：命中 %d 行，应为 0", changedType, changedCount)
	}

	t.Logf("ALL PASSED: dict_type 真禁改 e2e — PUT 带变更 dict_type 被忽略（库内 %q 不变、name 已更新、无 %q 行）", origType, changedType)
}
