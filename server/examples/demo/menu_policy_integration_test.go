// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   菜单 CUD 触发 Casbin policy 重载 e2e 对照 — T-005b-1 安全修复真跑验证
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-13 15:30:00
// +----------------------------------------------------------------------
//
// 运行方式：
//   docker compose -f deploy/docker-compose.dev.yml up -d   # 起 MySQL + Redis
//   go test -tags=integration ./examples/demo/ -v -count=1 -run TestMenuPolicyReload
//
// 验证目标（修 T-003b 既有安全缺陷 / T-007g 观察到的 80→80→80 纹丝不动）：
//   建/改/删 menu_type=F 节点后 casbin_rule 真跟着变 + 删/改 F 后 dept_mgr enforce 立即从 200 变 403。
// 隔离：独立表前缀 demomp_ + Redis DB 8，跑前跑后清表/清库，可重复跑、与 E2E 冒烟互不干扰。

//go:build integration

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/rbac"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	mpPrefix  = "demomp_"
	mpRedisDB = 8
)

// mpConfig 复用 e2eConfig 基底，仅改隔离前缀/库，避免与 E2E 冒烟撞表。
func mpConfig(t *testing.T) demoConfig {
	cfg := e2eConfig(t)
	cfg.TablePrefix = mpPrefix
	cfg.RedisDB = mpRedisDB
	return cfg
}

func mpDB(t *testing.T, cfg demoConfig) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), rbac.NewDBConfig(cfg.TablePrefix))
	if err != nil {
		t.Fatalf("connect mysql (%s): %v", cfg.MySQLDSN, err)
	}
	dropPrefixedTables(t, db, cfg.TablePrefix)
	t.Cleanup(func() { dropPrefixedTables(t, db, cfg.TablePrefix) })
	return db
}

func mpRedis(t *testing.T, cfg demoConfig) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, DB: cfg.RedisDB, Password: cfg.RedisPassword})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("connect redis (%s db=%d): %v", cfg.RedisAddr, cfg.RedisDB, err)
	}
	rdb.FlushDB(ctx)
	t.Cleanup(func() { rdb.FlushDB(ctx); rdb.Close() })
	return rdb
}

// countP 统计某角色某 perm_code 的 p 规则条数（ptype='p' AND v0=role AND v1=perm AND v2='access'）。
func countP(t *testing.T, db *gorm.DB, table, role, perm string) int64 {
	t.Helper()
	var n int64
	if err := db.Table(table).
		Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", role, perm, "access").
		Count(&n).Error; err != nil {
		t.Fatalf("count p rule (%s,%s): %v", role, perm, err)
	}
	return n
}

// totalP 统计全部 p 规则条数。
func totalP(t *testing.T, db *gorm.DB, table string) int64 {
	t.Helper()
	var n int64
	if err := db.Table(table).Where("ptype = ?", "p").Count(&n).Error; err != nil {
		t.Fatalf("count total p: %v", err)
	}
	return n
}

// findMenuByPerm 在 GET /sys/menus/tree 返回的嵌套树里按 perm_code 找节点（返回该节点 map，含 hashid id）。
func findMenuByPerm(nodes []any, perm string) map[string]any {
	for _, raw := range nodes {
		n, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if pc, _ := n["perm_code"].(string); pc == perm {
			return n
		}
		if kids, ok := n["children"].([]any); ok {
			if found := findMenuByPerm(kids, perm); found != nil {
				return found
			}
		}
	}
	return nil
}

func TestMenuPolicyReloadE2E(t *testing.T) {
	cfg := mpConfig(t)
	db := mpDB(t, cfg)
	rdb := mpRedis(t, cfg)

	app, err := buildApp(cfg, db, rdb)
	if err != nil {
		t.Fatalf("buildApp: %v", err)
	}
	t.Cleanup(app.subCancel)
	ts := httptest.NewServer(app.handler)
	t.Cleanup(ts.Close)

	casbinTable := cfg.TablePrefix + "casbin_rule"
	const role = "dept_mgr"
	const targetPerm = "sys:post:list" // dept_mgr 全量授权持有；对应 GET /sys/posts
	const postsURL = "/sys/posts"

	// ---- 登录：admin(super_admin) 做菜单 CUD；dept_mgr(非超管/走真 Casbin) 验 enforce ----
	login := func(user, pwd string) string {
		r := doJSON(t, http.MethodPost, ts.URL+"/auth/login", "", map[string]string{"username": user, "password": pwd})
		if r.status != http.StatusOK {
			t.Fatalf("login %s: expected 200, got %d body=%+v", user, r.status, r.body)
		}
		tok, _ := dataMap(t, r)["access_token"].(string)
		if tok == "" {
			t.Fatalf("login %s: empty token", user)
		}
		return tok
	}
	adminTok := login("admin", e2eAdminPwd)
	mgrTok := login("dept_mgr", e2eDeptMgrPwd)

	deptMgrGetPosts := func() int {
		return doJSON(t, http.MethodGet, ts.URL+postsURL, mgrTok, nil).status
	}

	// ===================================================================
	// 基线：dept_mgr 经真 Casbin 命中 sys:post:list → 200；p 规则在册
	// ===================================================================
	if got := deptMgrGetPosts(); got != http.StatusOK {
		t.Fatalf("baseline: dept_mgr GET %s expected 200, got %d", postsURL, got)
	}
	if n := countP(t, db, casbinTable, role, targetPerm); n != 1 {
		t.Fatalf("baseline: p(%s,%s) expected 1, got %d", role, targetPerm, n)
	}
	baseTotal := totalP(t, db, casbinTable)
	t.Logf("baseline OK: dept_mgr GET %s=200, p(%s,%s)=1, total p=%d", postsURL, role, targetPerm, baseTotal)

	// 取菜单树，定位 sys:post:list 的 F 节点（拿 hashid + 现有字段，供全量覆写更新）
	tree := doJSON(t, http.MethodGet, ts.URL+"/sys/menus/tree", adminTok, nil)
	if tree.status != http.StatusOK {
		t.Fatalf("get menu tree: %d %+v", tree.status, tree.body)
	}
	nodes, _ := tree.body["data"].([]any)
	node := findMenuByPerm(nodes, targetPerm)
	if node == nil {
		t.Fatalf("menu node with perm_code=%s not found in tree", targetPerm)
	}
	hid, _ := node["id"].(string)
	if hid == "" {
		t.Fatalf("menu node missing hashid id: %+v", node)
	}
	// 全量覆写 body 基底（Update 是全量覆写，须带回现有字段，仅改 perm_code）
	baseBody := map[string]any{
		"menu_type": node["menu_type"], "name": node["name"],
		"path": node["path"], "component": node["component"], "icon": node["icon"],
		"sort": node["sort"], "visible": node["visible"], "status": node["status"],
	}
	putMenu := func(permCode string) httpResult {
		body := map[string]any{}
		for k, v := range baseBody {
			body[k] = v
		}
		body["perm_code"] = permCode
		return doJSON(t, http.MethodPut, ts.URL+"/sys/menus/"+hid, adminTok, body)
	}

	// ===================================================================
	// 改 F：perm_code sys:post:list → sys:post:list_renamed
	//   期望：旧码 p 规则消失、新码进入；dept_mgr GET /sys/posts 立即 403（旧码失联）
	// ===================================================================
	const renamed = "sys:post:list_renamed"
	if r := putMenu(renamed); r.status != http.StatusOK {
		t.Fatalf("update F perm_code: expected 200, got %d body=%+v", r.status, r.body)
	}
	if n := countP(t, db, casbinTable, role, targetPerm); n != 0 {
		t.Fatalf("after rename: old p(%s,%s) expected 0, got %d (旧码 p 规则未消失=缺陷未修)", role, targetPerm, n)
	}
	if n := countP(t, db, casbinTable, role, renamed); n != 1 {
		t.Fatalf("after rename: new p(%s,%s) expected 1, got %d (新码未进 policy)", role, renamed, n)
	}
	if got := deptMgrGetPosts(); got != http.StatusForbidden {
		t.Fatalf("after rename: dept_mgr GET %s expected 403 (旧码已收回), got %d "+
			"(若仍 200=菜单 CUD 未触发 policy 重载=T-003b 缺陷)", postsURL, got)
	}
	t.Logf("改 F OK: old p→0, new p→1, dept_mgr GET %s 200→403（perm_code 改名权限即时失联）", postsURL)

	// ===================================================================
	// 改回：sys:post:list_renamed → sys:post:list
	//   期望：dept_mgr GET /sys/posts 恢复 200（双向 + 可恢复）
	// ===================================================================
	if r := putMenu(targetPerm); r.status != http.StatusOK {
		t.Fatalf("restore perm_code: expected 200, got %d body=%+v", r.status, r.body)
	}
	if n := countP(t, db, casbinTable, role, targetPerm); n != 1 {
		t.Fatalf("after restore: p(%s,%s) expected 1, got %d", role, targetPerm, n)
	}
	if got := deptMgrGetPosts(); got != http.StatusOK {
		t.Fatalf("after restore: dept_mgr GET %s expected 200, got %d", postsURL, got)
	}
	t.Logf("改回 OK: p→1, dept_mgr GET %s 恢复 200", postsURL)

	// ===================================================================
	// 建 F：新建一个未授任何角色的 F 节点
	//   期望：total p 不变（新菜单尚无 role_menu 关联 → 全量重载不增 p；语义诚实=create no-op-until-assign）
	// ===================================================================
	const probePerm = "sys:probe:create"
	create := doJSON(t, http.MethodPost, ts.URL+"/sys/menus", adminTok, map[string]any{
		"parent_id": "", "menu_type": "F", "name": "探针按钮", "perm_code": probePerm,
	})
	if create.status != http.StatusOK {
		t.Fatalf("create F: expected 200, got %d body=%+v", create.status, create.body)
	}
	if n := countP(t, db, casbinTable, role, probePerm); n != 0 {
		t.Fatalf("after create: p(%s,%s) expected 0 (未授角色), got %d", role, probePerm, n)
	}
	if nowTotal := totalP(t, db, casbinTable); nowTotal != baseTotal {
		t.Fatalf("after create: total p expected unchanged %d, got %d", baseTotal, nowTotal)
	}
	t.Logf("建 F OK: 新建未授角色 F 节点，total p 不变=%d（create 对 policy 为 no-op-until-assign，语义诚实）", baseTotal)

	// ===================================================================
	// 删 F：删除 sys:post:list 节点（dept_mgr 持有）
	//   期望：该 perm_code 的 p 规则消失；dept_mgr GET /sys/posts 立即 403（权限真收回 = 本片头号安全证据）
	// ===================================================================
	del := doJSON(t, http.MethodDelete, ts.URL+"/sys/menus/"+hid, adminTok, nil)
	if del.status != http.StatusOK {
		t.Fatalf("delete F: expected 200, got %d body=%+v", del.status, del.body)
	}
	if n := countP(t, db, casbinTable, role, targetPerm); n != 0 {
		t.Fatalf("after delete: p(%s,%s) expected 0, got %d (删 F 后 p 规则滞留=权限收不回=缺陷)", role, targetPerm, n)
	}
	if got := deptMgrGetPosts(); got != http.StatusForbidden {
		t.Fatalf("after delete: dept_mgr GET %s expected 403 (权限真收回), got %d "+
			"(若仍 200=删了门没关=T-003b 安全缺陷未修)", postsURL, got)
	}
	t.Logf("删 F OK: p(%s,%s)→0, dept_mgr GET %s 403（删 F 权限真收回，对照 T-007g 80→80→80 缺陷态）", role, targetPerm, postsURL)

	t.Log("ALL PASSED: 建/改/删 F 节点 casbin_rule 真跟着变 + 改/删 F 后 dept_mgr enforce 即时 200→403（T-003b 安全缺陷已修）")
}
