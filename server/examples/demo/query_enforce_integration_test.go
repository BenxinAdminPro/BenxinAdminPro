// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-005b-4 列表查询能力 enforce 正向证据 + operator 可读化 e2e（dept_mgr 200 ↔ editor 403）
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-14 11:30:00
// +----------------------------------------------------------------------
//
// 运行方式：go test -tags=integration ./examples/demo/ -v -count=1 -run TestQueryEnforce
// 前置：docker compose -f deploy/docker-compose.dev.yml up -d
//
// 验证：① 带新查询参数（模糊/排序/时间范围）各列表端点 dept_mgr（非超管/真 Casbin）全 200 ↔ editor 全 403
//       ② operator 可读化端到端（dept_mgr 写动作产生的 oper_log operator_name = "dept_mgr"）
//       ③ 时间范围非法 → 400；排序注入 → 被忽略不报 500（HTTP 层负例）
// 隔离：独立表前缀 demoqe_ + Redis DB 8。

//go:build integration

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/benxin_dev/benxinadminpro-server/rbac"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	qeEnfPrefix  = "demoqe_"
	qeEnfRedisDB = 8
)

func TestQueryEnforceE2E(t *testing.T) {
	cfg := e2eConfig(t)
	cfg.TablePrefix = qeEnfPrefix
	cfg.RedisDB = qeEnfRedisDB

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

	mgrTok := loginTok(t, ts.URL, "dept_mgr", e2eDeptMgrPwd)
	editorTok := loginTok(t, ts.URL, "editor", e2eEditorPwd)

	// 带新查询参数的各列表端点（覆盖模糊/排序/时间范围/用户名过滤/分页）
	endpoints := []string{
		"/sys/dict/types?keyword=sys&sort=created_at&order=desc",
		"/sys/dict/data?dict_type=sys_user_status&page=1&page_size=10",
		"/sys/configs?keyword=site&sort=created_at&order=asc",
		"/sys/logs/oper?operator_name=admin&path=/sys&sort=latency_ms&order=desc&start_time=2020-01-01",
		"/sys/logs/login?username=a&ip=1&sort=created_at&order=desc",
		"/sys/files?uploader_name=admin&original_name=x&sort=created_at&order=desc",
	}

	// ① enforce 正向对照：dept_mgr 全 200 ↔ editor 全 403
	for _, ep := range endpoints {
		mgr := doJSON(t, http.MethodGet, ts.URL+ep, mgrTok, nil)
		if mgr.status != http.StatusOK {
			t.Errorf("dept_mgr GET %s: got %d, want 200 (body=%+v)", ep, mgr.status, mgr.body)
		}
		ed := doJSON(t, http.MethodGet, ts.URL+ep, editorTok, nil)
		if ed.status != http.StatusForbidden {
			t.Errorf("editor GET %s: got %d, want 403", ep, ed.status)
		}
	}
	t.Log("enforce OK: 6 列表端点（带新查询参数）dept_mgr 全 200 ↔ editor 全 403")

	// ② operator 可读化端到端：dept_mgr 做一次写动作（POST 被 oper_log 采集），
	//    再列操作日志，断言该行 operator_name == "dept_mgr"（内部 ID → 用户名）。
	create := doJSON(t, http.MethodPost, ts.URL+"/sys/dict/types", mgrTok, map[string]any{
		"dict_type": "t005b4_probe", "name": "可读化探针",
	})
	if create.status != http.StatusOK {
		t.Fatalf("dept_mgr 创建字典类型: got %d %+v", create.status, create.body)
	}
	// oper_log 异步写入（OperLog 中间件 go func），轮询等待落库后再断言可读化。
	found := false
	for i := 0; i < 20 && !found; i++ {
		logs := doJSON(t, http.MethodGet, ts.URL+"/sys/logs/oper?path=/sys/dict/types&sort=created_at&order=desc", mgrTok, nil)
		if logs.status != http.StatusOK {
			t.Fatalf("列操作日志: got %d", logs.status)
		}
		rows, _ := dataMap(t, logs)["list"].([]any)
		for _, r := range rows {
			row, _ := r.(map[string]any)
			if row["method"] == "POST" && row["operator_name"] == "dept_mgr" {
				found = true
				break
			}
		}
		if !found {
			time.Sleep(100 * time.Millisecond)
		}
	}
	if !found {
		t.Fatalf("operator 可读化未生效：未在操作日志中找到 operator_name=dept_mgr 的 POST 行")
	}
	t.Log("operator 可读化 OK: dept_mgr 写动作的 oper_log operator_name=\"dept_mgr\"（内部 ID 已解析为用户名）")

	// ③ 负例：时间范围非法 → 400
	bad := doJSON(t, http.MethodGet, ts.URL+"/sys/logs/oper?start_time=not-a-date", mgrTok, nil)
	if bad.status != http.StatusBadRequest {
		t.Errorf("非法 start_time 应 400, got %d", bad.status)
	}
	// start>end → 400
	rev := doJSON(t, http.MethodGet, ts.URL+"/sys/logs/oper?start_time=2030-01-01&end_time=2020-01-01", mgrTok, nil)
	if rev.status != http.StatusBadRequest {
		t.Errorf("start>end 应 400, got %d", rev.status)
	}
	// ④ 排序注入 → 被忽略，不 500（仍 200）
	inj := doJSON(t, http.MethodGet, ts.URL+"/sys/logs/oper?sort=id%3BDROP+TABLE+x%3B--&order=asc", mgrTok, nil)
	if inj.status != http.StatusOK {
		t.Errorf("注入 sort 应被忽略返 200, got %d", inj.status)
	}
	t.Log("负例 OK: 非法时间范围 400 / start>end 400 / 排序注入被忽略 200（不进 ORDER BY、不 500）")

	t.Log("ALL PASSED: T-005b-4 enforce 正向 + operator 可读化 + 查询参数负例端到端")
}

// loginTok 登录并返回 access_token。
func loginTok(t *testing.T, baseURL, username, password string) string {
	t.Helper()
	res := doJSON(t, http.MethodPost, baseURL+"/auth/login", "", map[string]string{
		"username": username, "password": password,
	})
	if res.status != http.StatusOK {
		t.Fatalf("login %s: %d %+v", username, res.status, res.body)
	}
	tok, _ := dataMap(t, res)["access_token"].(string)
	if tok == "" {
		t.Fatalf("login %s: empty token", username)
	}
	return tok
}
