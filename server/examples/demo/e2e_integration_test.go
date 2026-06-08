// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   demo e2e 冒烟测试 — 真依赖跑通五大块全链路（T-006 核心交付）
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 12:00:00
// +----------------------------------------------------------------------
//
// 运行方式：
//   docker compose -f deploy/docker-compose.dev.yml up -d   # 起 MySQL + Redis
//   go test -tags=integration ./examples/demo/ -v -count=1 -run TestDemoE2ESmoke
//
// 依赖地址可经环境变量覆盖（端口冲突时用）：
//   DEMO_E2E_MYSQL_DSN   默认 root:root@tcp(127.0.0.1:3306)/benxinadminpro?charset=utf8mb4&parseTime=true&loc=Local
//   DEMO_E2E_REDIS_ADDR  默认 127.0.0.1:6379
//
// 覆盖链路（每步带断言）：
//   迁移建表 → 种子数据 → captcha → login 拿 token → 带 token 受保护接口 200
//   → 无 token 401 → 无权限 403 → 改配置触发热加载读到新值。

//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/benxin_dev/benxinadminpro-server/auth"
	"github.com/benxin_dev/benxinadminpro-server/rbac"
	"github.com/benxin_dev/benxinadminpro-server/system"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	e2ePrefix       = "demoe2e_"
	e2eRedisPrefix  = "demoe2e"
	e2eRedisDB      = 9
	e2eAdminPwd     = "admin-e2e-pwd"
	e2eEditorPwd    = "editor-e2e-pwd"
	e2eDeptMgrPwd   = "deptmgr-e2e-pwd"
	e2eBizPwd       = "biz-e2e-pwd"
	defaultE2EDSN   = "root:root@tcp(127.0.0.1:3306)/benxinadminpro?charset=utf8mb4&parseTime=true&loc=Local"
	defaultE2ERedis = "127.0.0.1:6379"
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// e2eConfig 构造自包含的 demoConfig：固定测试密钥、隔离表/redis 前缀、全部种子密码就位。
func e2eConfig(t *testing.T) demoConfig {
	t.Helper()
	key32 := base64.StdEncoding.EncodeToString(make([]byte, 32)) // 32 字节零密钥，仅测试用
	return demoConfig{
		TablePrefix:      e2ePrefix,
		RedisPrefix:      e2eRedisPrefix,
		Port:             0, // 用 httptest，不真正监听
		SegmentBase:      11000,
		AESKeyB64:        key32,
		HMACKeyB64:       key32,
		GCMKeyB64:        key32,
		JWTIssuer:        "benxinadminpro-e2e",
		JWTAccessSecret:  "e2e-access-secret-xxxxxxxxxxxxxxxx",
		JWTRefreshSecret: "e2e-refresh-secret-yyyyyyyyyyyyyyy",
		JWTAccessTTL:     7200,
		JWTRefreshTTL:    604800,
		Argon2Params: auth.Argon2idParams{
			MemoryKiB: 19456, Iterations: 2, Parallelism: 1, SaltLen: 16, KeyLen: 32,
		},
		SuperAdminRoles: []string{"super_admin"},
		HashidSalt:      "e2e-hashid-salt",
		FileRootDir:     t.TempDir(),
		FileURLPrefix:   "/sys/files/",
		FileMaxSize:     10 << 20,
		FileAllowedExts: []string{"jpg", "png", "txt"},
		SeedPasswords: map[string]string{
			"admin": e2eAdminPwd, "editor": e2eEditorPwd,
			"dept_mgr": e2eDeptMgrPwd, "biz_user": e2eBizPwd,
		},
		MySQLDSN:      envOr("DEMO_E2E_MYSQL_DSN", defaultE2EDSN),
		RedisAddr:     envOr("DEMO_E2E_REDIS_ADDR", defaultE2ERedis),
		RedisDB:       e2eRedisDB,
		RedisPassword: "",
	}
}

// e2eDB 连接 MySQL，先清掉本前缀的残留表保证可重复跑，结束再清一次。
func e2eDB(t *testing.T, cfg demoConfig) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), rbac.NewDBConfig(cfg.TablePrefix))
	if err != nil {
		t.Fatalf("connect mysql (%s): %v", cfg.MySQLDSN, err)
	}
	dropPrefixedTables(t, db, cfg.TablePrefix)
	t.Cleanup(func() { dropPrefixedTables(t, db, cfg.TablePrefix) })
	return db
}

func dropPrefixedTables(t *testing.T, db *gorm.DB, prefix string) {
	t.Helper()
	var names []string
	if err := db.Raw(
		"SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name LIKE ?",
		prefix+"%",
	).Scan(&names).Error; err != nil {
		t.Fatalf("list tables: %v", err)
	}
	if len(names) == 0 {
		return
	}
	db.Exec("SET FOREIGN_KEY_CHECKS=0")
	for _, n := range names {
		if err := db.Exec("DROP TABLE IF EXISTS `" + n + "`").Error; err != nil {
			t.Fatalf("drop table %s: %v", n, err)
		}
	}
	db.Exec("SET FOREIGN_KEY_CHECKS=1")
}

func e2eRedis(t *testing.T, cfg demoConfig) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, DB: cfg.RedisDB, Password: cfg.RedisPassword})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("connect redis (%s db=%d): %v", cfg.RedisAddr, cfg.RedisDB, err)
	}
	rdb.FlushDB(context.Background())
	t.Cleanup(func() { rdb.FlushDB(context.Background()); rdb.Close() })
	return rdb
}

// ---------------------------------------------------------------------------
// HTTP 小工具
// ---------------------------------------------------------------------------

type httpResult struct {
	status int
	body   map[string]any
}

func doJSON(t *testing.T, method, url, token string, payload any) httpResult {
	t.Helper()
	var bodyReader io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		t.Fatalf("build request %s %s: %v", method, url, err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request %s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out) // 错误响应也是 JSON；解析失败留空 map
	return httpResult{status: resp.StatusCode, body: out}
}

func dataMap(t *testing.T, r httpResult) map[string]any {
	t.Helper()
	d, ok := r.body["data"].(map[string]any)
	if !ok {
		t.Fatalf("response has no data object: %+v", r.body)
	}
	return d
}

// ---------------------------------------------------------------------------
// 主测试：全链路冒烟
// ---------------------------------------------------------------------------

func TestDemoE2ESmoke(t *testing.T) {
	cfg := e2eConfig(t)
	db := e2eDB(t, cfg)
	rdb := e2eRedis(t, cfg)

	// ---- 装配（含迁移建表 + 种子数据）----
	app, err := buildApp(cfg, db, rdb)
	if err != nil {
		t.Fatalf("buildApp: %v", err)
	}
	t.Cleanup(app.subCancel)

	ts := httptest.NewServer(app.handler)
	t.Cleanup(ts.Close)

	// ---- 步骤 1：迁移建表落地 ----
	var tableCount int64
	db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name LIKE ?",
		e2ePrefix+"%").Scan(&tableCount)
	if tableCount == 0 {
		t.Fatal("step1 migration: no tables created with prefix " + e2ePrefix)
	}
	t.Logf("step1 migration OK: %d tables created", tableCount)

	// ---- 步骤 2：种子数据落库 ----
	var admin rbac.SysUser
	if err := db.Where("username = ?", "admin").First(&admin).Error; err != nil {
		t.Fatalf("step2 seed: admin user not found: %v", err)
	}
	if admin.PasswordHash == "" || admin.PasswordHash == e2eAdminPwd {
		t.Fatalf("step2 seed: admin password not hashed properly: %q", admin.PasswordHash)
	}
	t.Logf("step2 seed OK: admin user id=%d, password hashed", admin.ID)

	// ---- 步骤 3：captcha 端点 ----
	cap := doJSON(t, http.MethodPost, ts.URL+"/auth/captcha", "", nil)
	if cap.status != http.StatusOK {
		t.Fatalf("step3 captcha: expected 200, got %d body=%+v", cap.status, cap.body)
	}
	if id, _ := dataMap(t, cap)["captcha_id"].(string); id == "" {
		t.Fatalf("step3 captcha: empty captcha_id in %+v", cap.body)
	}
	t.Logf("step3 captcha OK")

	// ---- 步骤 4：login 拿 token（干净账号首登无需验证码）----
	login := doJSON(t, http.MethodPost, ts.URL+"/auth/login", "", map[string]string{
		"username": "admin", "password": e2eAdminPwd,
	})
	if login.status != http.StatusOK {
		t.Fatalf("step4 login: expected 200, got %d body=%+v", login.status, login.body)
	}
	adminToken, _ := dataMap(t, login)["access_token"].(string)
	if adminToken == "" {
		t.Fatalf("step4 login: empty access_token in %+v", login.body)
	}
	t.Logf("step4 login OK: got admin access_token")

	// ---- 步骤 5：带 token 调受保护接口 → 200 ----
	listed := doJSON(t, http.MethodGet, ts.URL+"/sys/users", adminToken, nil)
	if listed.status != http.StatusOK {
		t.Fatalf("step5 protected-with-token: expected 200, got %d body=%+v", listed.status, listed.body)
	}
	t.Logf("step5 protected(200) OK")

	// ---- 步骤 6：无 token → 401 ----
	noToken := doJSON(t, http.MethodGet, ts.URL+"/sys/users", "", nil)
	if noToken.status != http.StatusUnauthorized {
		t.Fatalf("step6 no-token: expected 401, got %d body=%+v", noToken.status, noToken.body)
	}
	t.Logf("step6 no-token(401) OK")

	// ---- 步骤 7：无权限 → 403 ----
	// editor 仅有 sys:user:list，调 POST /sys/users（需 sys:user:create）应被拒。
	editorLogin := doJSON(t, http.MethodPost, ts.URL+"/auth/login", "", map[string]string{
		"username": "editor", "password": e2eEditorPwd,
	})
	if editorLogin.status != http.StatusOK {
		t.Fatalf("step7 editor login: expected 200, got %d body=%+v", editorLogin.status, editorLogin.body)
	}
	editorToken, _ := dataMap(t, editorLogin)["access_token"].(string)
	if editorToken == "" {
		t.Fatalf("step7 editor login: empty access_token")
	}
	// 先证明 editor 有 list 权限：GET 应 200（排除"全拒"的假阳性）
	editorList := doJSON(t, http.MethodGet, ts.URL+"/sys/users", editorToken, nil)
	if editorList.status != http.StatusOK {
		t.Fatalf("step7 editor GET (has perm): expected 200, got %d body=%+v", editorList.status, editorList.body)
	}
	// 无 create 权限：POST 应 403
	editorCreate := doJSON(t, http.MethodPost, ts.URL+"/sys/users", editorToken, map[string]string{
		"username": "should_not_create", "password": "whatever123",
	})
	if editorCreate.status != http.StatusForbidden {
		t.Fatalf("step7 no-perm: editor POST /sys/users expected 403, got %d body=%+v "+
			"(说明权限 enforce 未生效——RequirePerm 用了非 enforce 的包级版本)", editorCreate.status, editorCreate.body)
	}
	t.Logf("step7 no-perm(403) OK")

	// ---- 步骤 8：改配置触发热加载读到新值 ----
	ctx := context.Background()
	const ckey = "demo.greeting"
	if err := db.Create(&system.SysConfig{
		ConfigKey: ckey, ConfigValue: "hello-v1", IsEncrypted: 0, Name: "问候语",
	}).Error; err != nil {
		t.Fatalf("step8 seed config: %v", err)
	}

	// 首读 → 回源 DB 并入缓存
	v1, err := app.configCenter.GetConfig(ctx, ckey)
	if err != nil || v1 != "hello-v1" {
		t.Fatalf("step8 first read: want hello-v1, got %q err=%v", v1, err)
	}

	// 直接改 DB（模拟另一实例写入），此时缓存仍是旧值
	if err := db.Model(&system.SysConfig{}).Where("config_key = ?", ckey).
		Update("config_value", "hello-v2").Error; err != nil {
		t.Fatalf("step8 update db: %v", err)
	}
	if cached, _ := app.configCenter.GetConfig(ctx, ckey); cached != "hello-v1" {
		t.Fatalf("step8 cache check: expected stale cache hello-v1, got %q (缓存层未生效)", cached)
	}

	// 触发热加载：发布失效 → 订阅协程 OnChange 删缓存
	app.configCenter.InvalidateConfig(ctx, ckey)

	// 轮询等 Pub/Sub 生效（最多 ~3s）
	var v2 string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		v2, _ = app.configCenter.GetConfig(ctx, ckey)
		if v2 == "hello-v2" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if v2 != "hello-v2" {
		t.Fatalf("step8 hot-reload: expected hello-v2 after invalidate, got %q (热加载未读到新值)", v2)
	}
	t.Logf("step8 hot-reload OK: hello-v1 → hello-v2")

	t.Log("ALL STEPS PASSED: 迁移→种子→captcha→login→200→401→403→热加载 全链路打通")
}
