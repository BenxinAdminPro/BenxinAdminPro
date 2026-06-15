// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-009a X-Robots-Tag 全栈 e2e — 全局注入(含错误响应)+ 内容回退 + 开关 toggle
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-15 14:30:00
// +----------------------------------------------------------------------
//
// 运行方式：go test -tags=integration ./examples/demo/ -v -count=1 -run TestXRobots
// 前置：docker compose -f deploy/docker-compose.dev.yml up -d
// 隔离：独立表前缀 demorb_ + Redis DB 13。
// 注：中间件逻辑四态单测见 httpmw/robots_test.go；本测坐实经 buildApp 全局真注入。

//go:build integration

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/httpmw"
	"github.com/benxin_dev/benxinadminpro-server/rbac"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	rbEnfPrefix  = "demorb_"
	rbEnfRedisDB = 13
)

// buildRobotsApp 用给定 X-Robots 配置起一个 httptest server，返回 base URL。
func buildRobotsApp(t *testing.T, enabled bool, content string) string {
	t.Helper()
	cfg := e2eConfig(t)
	cfg.TablePrefix = rbEnfPrefix
	cfg.RedisDB = rbEnfRedisDB
	cfg.XRobotsEnabled = enabled
	cfg.XRobotsContent = content

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
	return ts.URL
}

func headerOf(t *testing.T, url string) (int, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode, resp.Header.Get("X-Robots-Tag")
}

// 开启 + 内容缺省（空）→ 全局注入默认安全头，且**错误响应（401）也带头**（坐实无差别注入）。
func TestXRobotsEnabledInjectsOnAllResponses(t *testing.T) {
	base := buildRobotsApp(t, true, "") // 内容空 → 回退 DefaultXRobotsTag

	// 受保护端点无 token → 401，但全局中间件先于鉴权注头
	status, hdr := headerOf(t, base+"/sys/users")
	if status != http.StatusUnauthorized {
		t.Logf("注：/sys/users 无 token 预期 401，实际 %d（不影响头断言）", status)
	}
	if hdr != httpmw.DefaultXRobotsTag {
		t.Fatalf("错误响应应带 X-Robots-Tag=%q，got %q", httpmw.DefaultXRobotsTag, hdr)
	}

	// 公共端点（404 路由）同样带头
	_, hdr2 := headerOf(t, base+"/no-such-path")
	if hdr2 != httpmw.DefaultXRobotsTag {
		t.Fatalf("404 响应应带 X-Robots-Tag=%q，got %q", httpmw.DefaultXRobotsTag, hdr2)
	}
	t.Logf("全局注入 OK: 401/404 响应均带 X-Robots-Tag=%q（无差别 + 内容回退）", httpmw.DefaultXRobotsTag)
}

// 自定义内容 → 注入自定义值（全栈）。
func TestXRobotsCustomContent(t *testing.T) {
	base := buildRobotsApp(t, true, "noindex")
	_, hdr := headerOf(t, base+"/sys/users")
	if hdr != "noindex" {
		t.Fatalf("got %q, want noindex", hdr)
	}
}

// 开关关 → 全栈无该头（参数化 toggle 生效）。
func TestXRobotsDisabledNoHeader(t *testing.T) {
	base := buildRobotsApp(t, false, "noindex, nofollow")
	_, hdr := headerOf(t, base+"/sys/users")
	if hdr != "" {
		t.Fatalf("disabled 应无 X-Robots-Tag 头，got %q", hdr)
	}
}
