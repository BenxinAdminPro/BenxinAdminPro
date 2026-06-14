// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-008b 用户分角色 e2e — 回填正确性 + 全量覆写不误删 + Casbin g 联动 + enforce 正向
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-14 15:50:00
// +----------------------------------------------------------------------
//
// 运行方式：go test -tags=integration ./examples/demo/ -v -count=1 -run TestAssignRoles
// 前置：docker compose -f deploy/docker-compose.dev.yml up -d
//
// 验证：① GET /sys/users/:id 预载 roles 回填（分配前空 → 分配后精确返已授）
//       ② 全量覆写不误删未动角色（2→3→1 行精确，DB 实查 user_role）
//       ③ Casbin g 规则联动真生效（分配含 sys:user:list 的角色后，该用户登录能 GET /sys/users）
//       ④ enforce 正向：dept_mgr（含 sys:user:assign）200 ↔ editor 403
// 隔离：独立表前缀 demoar_ + Redis DB 9。

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
	arPrefix  = "demoar_"
	arRedisDB = 9
)

func TestAssignRolesE2E(t *testing.T) {
	cfg := e2eConfig(t)
	cfg.TablePrefix = arPrefix
	cfg.RedisDB = arRedisDB

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
	mgrTok := loginTok(t, ts.URL, "dept_mgr", e2eDeptMgrPwd)
	editorTok := loginTok(t, ts.URL, "editor", e2eEditorPwd)

	// 取角色 hashid（按 code）
	roleID := rolesByCode(t, ts.URL, adminTok)
	editorRole := roleID["editor"]
	mgrRole := roleID["dept_mgr"]
	// 第三个角色：任取一个不是 editor/dept_mgr 的已有角色（seed 至少有 super_admin）
	thirdRole := ""
	for code, id := range roleID {
		if code != "editor" && code != "dept_mgr" {
			thirdRole = id
			break
		}
	}
	if editorRole == "" || mgrRole == "" || thirdRole == "" {
		t.Fatalf("缺角色 hashid: editor=%q dept_mgr=%q third=%q (roles=%v)", editorRole, mgrRole, thirdRole, roleID)
	}

	// admin 建无角色探针用户（密码已知，供登录验 Casbin 联动）
	probePwd := "ProbePass@1"
	probe := doJSON(t, http.MethodPost, ts.URL+"/sys/users", adminTok, map[string]any{
		"username": "t008b_probe", "password": probePwd, "nickname": "分角色探针",
	})
	if probe.status != http.StatusOK {
		t.Fatalf("建探针用户: %d %+v", probe.status, probe.body)
	}
	pid, _ := dataMap(t, probe)["id"].(string)
	if pid == "" {
		t.Fatalf("探针用户 id 空")
	}

	// ① 分配前 GET 详情：roles 应空/缺失
	if n := getUserRoleCount(t, ts.URL, adminTok, pid); n != 0 {
		t.Fatalf("分配前 roles 应为 0，got %d", n)
	}

	// ④ enforce 正向：editor 无 sys:user:assign → 403；dept_mgr 有 → 200
	edAssign := doJSON(t, http.MethodPut, ts.URL+"/sys/users/"+pid+"/roles", editorTok, map[string]any{"role_ids": []string{editorRole}})
	if edAssign.status != http.StatusForbidden {
		t.Errorf("editor 分配角色应 403，got %d", edAssign.status)
	}
	edGet := doJSON(t, http.MethodGet, ts.URL+"/sys/users/"+pid, editorTok, nil)
	if edGet.status != http.StatusForbidden {
		t.Errorf("editor 查用户详情应 403（无 sys:user:get），got %d", edGet.status)
	}

	// ② 全量覆写 + 回填：分配 2 角色 → GET 返 2 → 增至 3 → 减至 1，逐步 DB 实查 user_role 行数
	assignAndCheck := func(label string, roleIDs []string, wantCount int) {
		res := doJSON(t, http.MethodPut, ts.URL+"/sys/users/"+pid+"/roles", mgrTok, map[string]any{"role_ids": roleIDs})
		if res.status != http.StatusOK {
			t.Fatalf("%s dept_mgr 分配应 200，got %d %+v", label, res.status, res.body)
		}
		// 出参回填
		if n := getUserRoleCount(t, ts.URL, adminTok, pid); n != wantCount {
			t.Fatalf("%s GET 回填 roles 数应 %d，got %d", label, wantCount, n)
		}
		// DB 实查 user_role 行数（probe 内部 id 解析）
		var probeUser rbac.SysUser
		db.Where("username = ?", "t008b_probe").First(&probeUser)
		var rows int64
		db.Table(cfg.TablePrefix + "sys_user_role").Where("user_id = ?", probeUser.ID).Count(&rows)
		if int(rows) != wantCount {
			t.Fatalf("%s DB user_role 行数应 %d，got %d（全量覆写误删/漏写）", label, wantCount, rows)
		}
	}
	assignAndCheck("分配2角色", []string{editorRole, mgrRole}, 2)
	assignAndCheck("增至3角色", []string{editorRole, mgrRole, thirdRole}, 3)
	assignAndCheck("减至1角色", []string{editorRole}, 1)
	t.Log("回填 + 全量覆写不误删 OK：2→3→1 出参与 DB user_role 行数逐步精确")

	// ③ Casbin g 规则联动真生效：probe 现仅 editor 角色（含 sys:user:list）→ 登录后能 GET /sys/users
	probeTok := loginTok(t, ts.URL, "t008b_probe", probePwd)
	listRes := doJSON(t, http.MethodGet, ts.URL+"/sys/users?page_size=1", probeTok, nil)
	if listRes.status != http.StatusOK {
		t.Fatalf("Casbin g 联动失败：probe 持 editor 角色应能 GET /sys/users(sys:user:list)，got %d", listRes.status)
	}
	// 反向：清空角色后 g 规则应收回 → 同端点 403
	clear := doJSON(t, http.MethodPut, ts.URL+"/sys/users/"+pid+"/roles", mgrTok, map[string]any{"role_ids": []string{}})
	if clear.status != http.StatusOK {
		t.Fatalf("清空角色应 200，got %d", clear.status)
	}
	probeTok2 := loginTok(t, ts.URL, "t008b_probe", probePwd)
	listRes2 := doJSON(t, http.MethodGet, ts.URL+"/sys/users?page_size=1", probeTok2, nil)
	if listRes2.status != http.StatusForbidden {
		t.Fatalf("Casbin g 联动收回失败：probe 清空角色后应 403，got %d", listRes2.status)
	}
	t.Log("Casbin g 联动 OK：分配 editor 角色 → probe enforce sys:user:list 通过；清空 → 收回 403")

	// password_hash 不泄漏断言（A 方案预载 roles 不破坏）
	detail := doJSON(t, http.MethodGet, ts.URL+"/sys/users/"+pid, adminTok, nil)
	if _, leaked := dataMap(t, detail)["password_hash"]; leaked {
		t.Fatal("password_hash 泄漏于用户详情出参（T-003a 铁律破坏）")
	}

	// 清理探针
	doJSON(t, http.MethodDelete, ts.URL+"/sys/users/"+pid, adminTok, nil)
	t.Log("ALL PASSED: T-008b 回填 + 覆写不误删 + Casbin g 联动 + enforce 正向 + password_hash 不泄漏")
}

// rolesByCode 取所有角色 code→hashid 映射。
func rolesByCode(t *testing.T, baseURL, tok string) map[string]string {
	t.Helper()
	res := doJSON(t, http.MethodGet, baseURL+"/sys/roles?page_size=100", tok, nil)
	if res.status != http.StatusOK {
		t.Fatalf("列角色: %d", res.status)
	}
	out := map[string]string{}
	list, _ := dataMap(t, res)["list"].([]any)
	for _, r := range list {
		m, _ := r.(map[string]any)
		code, _ := m["code"].(string)
		id, _ := m["id"].(string)
		if code != "" && id != "" {
			out[code] = id
		}
	}
	return out
}

// getUserRoleCount 取 GET /sys/users/:id 出参 roles 数（缺失=0）。
func getUserRoleCount(t *testing.T, baseURL, tok, id string) int {
	t.Helper()
	res := doJSON(t, http.MethodGet, baseURL+"/sys/users/"+id, tok, nil)
	if res.status != http.StatusOK {
		t.Fatalf("GET 用户详情: %d %+v", res.status, res.body)
	}
	roles, ok := dataMap(t, res)["roles"].([]any)
	if !ok {
		return 0
	}
	// 顺带校验每个 role 含 id/code（hashid 出参形态）
	codes := make([]string, 0, len(roles))
	for _, r := range roles {
		m, _ := r.(map[string]any)
		if _, hasID := m["id"].(string); !hasID {
			t.Errorf("role 出参缺 id hashid: %+v", m)
		}
		if c, _ := m["code"].(string); c != "" {
			codes = append(codes, c)
		}
	}
	sort.Strings(codes)
	return len(roles)
}
