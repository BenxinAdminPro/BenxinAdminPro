// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-008c 角色授权树 e2e — 回填全量(M/C/F) + 全量覆写往返不丢/无残留 + enforce 正向 + policy 生命周期
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-14 18:20:00
// +----------------------------------------------------------------------
//
// 运行方式：go test -tags=integration ./examples/demo/ -v -count=1 -run TestRoleAssignMenus
// 前置：docker compose -f deploy/docker-compose.dev.yml up -d
//
// 验证：① 回填全量：GET /sys/roles/:id 返该角色当前全量 menu_ids（含 M/C/F 三层、均 hashid）
//       ② 全量覆写往返不丢/无残留：assign A → Get=A → assign A'(换一节点) → Get=A' → assign A → Get=A，
//          每步 GET 出参与 DB role_menu 行数精确（无静默丢、无残留）
//       ③ enforce 正向：editor 无 sys:role:assign/sys:role:list → PUT menus / GET 详情 403 ↔ dept_mgr 200
//       ④ policy 生命周期 + 全链路 enforce：经端点给测试角色授含 F(sys:user:list) 的菜单集 →
//          casbin_rule p 规则随之 0→1、绑该角色的探针用户登录 GET /sys/users 200；
//          覆写为不含该 F 的集合 → p 规则 1→0、探针重登录同端点 403（SyncRolePerms 真驱动、可收回）
//       ⑤ enc.Role 加 menu_ids 后无敏感字段（无 password_hash）
// 隔离：独立表前缀 demoram_ + Redis DB 10。countP/dataMap/doJSON/loginTok 复用同包既有 helper。

//go:build integration

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/rbac"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	ramPrefix  = "demoram_"
	ramRedisDB = 10
)

func TestRoleAssignMenusE2E(t *testing.T) {
	cfg := e2eConfig(t)
	cfg.TablePrefix = ramPrefix
	cfg.RedisDB = ramRedisDB

	ctx := context.Background()
	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), rbac.NewDBConfig(cfg.TablePrefix))
	if err != nil {
		t.Fatalf("connect mysql: %v", err)
	}
	dropPrefixedTables(t, db, cfg.TablePrefix)
	t.Cleanup(func() { dropPrefixedTables(t, db, cfg.TablePrefix) })

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, DB: cfg.RedisDB, Password: cfg.RedisPassword})
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
	mgrTok := loginTok(t, ts.URL, "dept_mgr", e2eDeptMgrPwd) // 非超管，走真 Casbin
	editorTok := loginTok(t, ts.URL, "editor", e2eEditorPwd)

	// ---- 从菜单树取覆盖 M/C/F 三层的节点（admin 载树）----
	tree := doJSON(t, http.MethodGet, ts.URL+"/sys/menus/tree", adminTok, nil)
	if tree.status != http.StatusOK {
		t.Fatalf("get menu tree: %d %+v", tree.status, tree.body)
	}
	nodes, _ := tree.body["data"].([]any)
	var flat []map[string]any
	collectMenus(nodes, &flat)

	firstByType := func(mt string) map[string]any {
		for _, n := range flat {
			if t2, _ := n["menu_type"].(string); t2 == mt {
				return n
			}
		}
		return nil
	}
	byPerm := func(perm string) map[string]any {
		for _, n := range flat {
			if pc, _ := n["perm_code"].(string); pc == perm {
				return n
			}
		}
		return nil
	}
	mNode := firstByType("M")
	cNode := firstByType("C")
	fUserList := byPerm("sys:user:list") // F，供全链路 enforce
	fPostList := byPerm("sys:post:list") // F，覆写换入
	if mNode == nil || cNode == nil || fUserList == nil || fPostList == nil {
		t.Fatalf("菜单树缺必要节点：M=%v C=%v F(user:list)=%v F(post:list)=%v", mNode != nil, cNode != nil, fUserList != nil, fPostList != nil)
	}
	mID := mNode["id"].(string)
	cID := cNode["id"].(string)
	fUL := fUserList["id"].(string)
	fPL := fPostList["id"].(string)

	// 集合 A = {M, C, F(user:list)}（含三层 + 含 sys:user:list）；A' = 换 F(user:list)→F(post:list)
	setA := []string{mID, cID, fUL}
	setAp := []string{mID, cID, fPL}

	// ---- admin 建测试角色 + 探针用户，并把测试角色挂到探针用户（验全链路 enforce）----
	const roleCode = "t008c_probe_role"
	roleRes := doJSON(t, http.MethodPost, ts.URL+"/sys/roles", adminTok, map[string]any{
		"code": roleCode, "name": "授权树探针角色", "data_scope": 1,
	})
	if roleRes.status != http.StatusOK {
		t.Fatalf("建测试角色: %d %+v", roleRes.status, roleRes.body)
	}
	roleHID, _ := dataMap(t, roleRes)["id"].(string)
	if roleHID == "" {
		t.Fatalf("测试角色 id 空")
	}
	var roleRow rbac.SysRole
	db.Where("code = ?", roleCode).First(&roleRow) // 内部 id，供 DB 行数核对

	probePwd := "ProbePass@1"
	probe := doJSON(t, http.MethodPost, ts.URL+"/sys/users", adminTok, map[string]any{
		"username": "t008c_probe", "password": probePwd, "nickname": "授权探针",
	})
	if probe.status != http.StatusOK {
		t.Fatalf("建探针用户: %d %+v", probe.status, probe.body)
	}
	pid, _ := dataMap(t, probe)["id"].(string)
	if assign := doJSON(t, http.MethodPut, ts.URL+"/sys/users/"+pid+"/roles", adminTok, map[string]any{"role_ids": []string{roleHID}}); assign.status != http.StatusOK {
		t.Fatalf("探针挂测试角色: %d %+v", assign.status, assign.body)
	}

	casbinTable := cfg.TablePrefix + "casbin_rule"
	roleMenuTable := cfg.TablePrefix + "sys_role_menu"
	dbRoleMenuRows := func() int64 {
		var n int64
		db.Table(roleMenuTable).Where("role_id = ?", roleRow.ID).Count(&n)
		return n
	}

	// ===================================================================
	// ③ enforce 正向：editor 无 sys:role:assign / sys:role:list → 403 ↔ dept_mgr 200（贯穿后续 assign）
	// ===================================================================
	if r := doJSON(t, http.MethodPut, ts.URL+"/sys/roles/"+roleHID+"/menus", editorTok, map[string]any{"menu_ids": setA}); r.status != http.StatusForbidden {
		t.Errorf("③ editor 授菜单应 403（无 sys:role:assign），got %d", r.status)
	}
	if r := doJSON(t, http.MethodGet, ts.URL+"/sys/roles/"+roleHID, editorTok, nil); r.status != http.StatusForbidden {
		t.Errorf("③ editor 取角色详情应 403（无 sys:role:list），got %d", r.status)
	}

	// 回填前：menu_ids 应空
	if got := getRoleMenuIDs(t, ts.URL, adminTok, roleHID); len(got) != 0 {
		t.Fatalf("① 回填前 menu_ids 应空，got %v", got)
	}

	// assign 全部用 mgrTok（dept_mgr 非超管）→ 同时坐实 ③ dept_mgr 在 T-008c 端点 200
	assign := func(label string, ids []string) {
		r := doJSON(t, http.MethodPut, ts.URL+"/sys/roles/"+roleHID+"/menus", mgrTok, map[string]any{"menu_ids": ids})
		if r.status != http.StatusOK {
			t.Fatalf("%s dept_mgr 授菜单应 200，got %d %+v", label, r.status, r.body)
		}
	}
	sortedEqual := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		ca, cb := append([]string(nil), a...), append([]string(nil), b...)
		sort.Strings(ca)
		sort.Strings(cb)
		for i := range ca {
			if ca[i] != cb[i] {
				return false
			}
		}
		return true
	}

	// ===================================================================
	// ① 回填全量(M/C/F) + ② 覆写往返不丢/无残留：assign A → Get=A → A' → Get=A' → A → Get=A
	// ===================================================================
	step := func(label string, ids []string) {
		assign(label, ids)
		got := getRoleMenuIDs(t, ts.URL, adminTok, roleHID)
		if !sortedEqual(got, ids) {
			t.Fatalf("%s GET 回填 menu_ids 应=%v（全量 M/C/F），got %v", label, ids, got)
		}
		if rows := dbRoleMenuRows(); int(rows) != len(ids) {
			t.Fatalf("%s DB role_menu 行数应 %d，got %d（覆写误删/漏写/残留）", label, len(ids), rows)
		}
	}
	step("assign A", setA)
	step("assign A'(换一节点)", setAp)
	step("assign A(回到原集)", setA)
	t.Log("①② 回填全量(M/C/F)+全量覆写往返不丢/无残留 OK：A→A'→A 出参与 DB role_menu 行数逐步精确")

	// ===================================================================
	// ④ policy 生命周期 + 全链路 enforce：当前集 A 含 F(sys:user:list)
	//    → casbin p(roleCode, sys:user:list)=1，探针登录 GET /sys/users 200
	// ===================================================================
	if n := countP(t, db, casbinTable, roleCode, "sys:user:list"); n != 1 {
		t.Fatalf("④ 集 A 含 F(sys:user:list)，p(%s,sys:user:list) 应=1，got %d（SyncRolePerms 未生效）", roleCode, n)
	}
	probeTok := loginTok(t, ts.URL, "t008c_probe", probePwd)
	if r := doJSON(t, http.MethodGet, ts.URL+"/sys/users?page_size=1", probeTok, nil); r.status != http.StatusOK {
		t.Fatalf("④ 探针持含 sys:user:list 的角色应能 GET /sys/users，got %d", r.status)
	}
	// 覆写为 A'(不含 sys:user:list、含 sys:post:list) → p 收回 + 探针 403
	assign("覆写为 A'(去 user:list)", setAp)
	if n := countP(t, db, casbinTable, roleCode, "sys:user:list"); n != 0 {
		t.Fatalf("④ 覆写去 F(sys:user:list) 后 p 应=0，got %d（权限未收回）", n)
	}
	if n := countP(t, db, casbinTable, roleCode, "sys:post:list"); n != 1 {
		t.Fatalf("④ 覆写换入 F(sys:post:list) 后 p 应=1，got %d", n)
	}
	probeTok2 := loginTok(t, ts.URL, "t008c_probe", probePwd)
	if r := doJSON(t, http.MethodGet, ts.URL+"/sys/users?page_size=1", probeTok2, nil); r.status != http.StatusForbidden {
		t.Fatalf("④ 覆写去 sys:user:list 后探针应 403，got %d（policy 未收回=全量覆写未驱动 Casbin）", r.status)
	}
	t.Log("④ policy 生命周期 + 全链路 enforce OK：授 F→p=1→探针 200；覆写去 F→p=0→探针 403（SyncRolePerms 真驱动、可收回）")

	// ===================================================================
	// ⑤ enc.Role 加 menu_ids 后无敏感字段；menu_ids 均为 hashid 字符串
	// ===================================================================
	detail := doJSON(t, http.MethodGet, ts.URL+"/sys/roles/"+roleHID, adminTok, nil)
	dm := dataMap(t, detail)
	if _, leaked := dm["password_hash"]; leaked {
		t.Fatal("⑤ password_hash 泄漏于角色详情出参（T-003a 铁律破坏）")
	}
	mids, ok := dm["menu_ids"].([]any)
	if !ok || len(mids) == 0 {
		t.Fatalf("⑤ 角色详情应含非空 menu_ids，got %+v", dm["menu_ids"])
	}
	for _, v := range mids {
		if _, isStr := v.(string); !isStr {
			t.Fatalf("⑤ menu_ids 元素应为 hashid 字符串，got %T(%v)", v, v)
		}
	}

	// 清理
	doJSON(t, http.MethodDelete, ts.URL+"/sys/users/"+pid, adminTok, nil)
	t.Log("ALL PASSED: T-008c 回填全量(M/C/F) + 覆写往返不丢/无残留 + enforce 正向 + policy 生命周期全链路 + 无敏感字段")
}

// collectMenus 递归扁平化菜单树节点（含 children）到 out。
func collectMenus(nodes []any, out *[]map[string]any) {
	for _, raw := range nodes {
		n, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		*out = append(*out, n)
		if kids, ok := n["children"].([]any); ok {
			collectMenus(kids, out)
		}
	}
}

// getRoleMenuIDs 取 GET /sys/roles/:id 出参 menu_ids（缺失=空），并校验均为 hashid 字符串。
func getRoleMenuIDs(t *testing.T, baseURL, tok, id string) []string {
	t.Helper()
	res := doJSON(t, http.MethodGet, baseURL+"/sys/roles/"+id, tok, nil)
	if res.status != http.StatusOK {
		t.Fatalf("GET 角色详情: %d %+v", res.status, res.body)
	}
	raw, ok := dataMap(t, res)["menu_ids"].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		s, isStr := v.(string)
		if !isStr {
			t.Fatalf("menu_ids 元素非 hashid 字符串: %T(%v)", v, v)
		}
		out = append(out, s)
	}
	return out
}
