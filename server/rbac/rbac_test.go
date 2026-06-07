// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   Casbin RBAC 单元测试 — enforcer + Authz 中间件 + perm code
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:40:00
// | @updated   2026-06-07 21:35:00
// +----------------------------------------------------------------------

package rbac

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/casbin/casbin/v2"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
	"github.com/gin-gonic/gin"
)

func modelConfPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "spec", "rbac", "model.conf")
}

func newTestEnforcer(t *testing.T) *casbin.Enforcer {
	t.Helper()
	tmpDir := t.TempDir()
	policyFile := filepath.Join(tmpDir, "policy.csv")
	os.WriteFile(policyFile, []byte(""), 0o644)
	adapter := fileadapter.NewAdapter(policyFile)
	e, err := casbin.NewEnforcer(modelConfPath(), adapter)
	if err != nil {
		t.Fatalf("NewEnforcer: %v", err)
	}
	return e
}

// T-003b: model.conf 改为精确匹配 perm code
func TestEnforcerPermCodeExactMatch(t *testing.T) {
	e := newTestEnforcer(t)

	_, _ = e.AddPolicy("admin", "sys:user:list", "access")

	allowed, _ := e.Enforce("admin", "sys:user:list", "access")
	if !allowed {
		t.Error("should allow exact perm code match")
	}

	// 不同 perm code → 拒绝
	allowed, _ = e.Enforce("admin", "sys:user:create", "access")
	if allowed {
		t.Error("should deny different perm code")
	}

	// 部分匹配 → 拒绝（精确匹配不做模式匹配）
	allowed, _ = e.Enforce("admin", "sys:user", "access")
	if allowed {
		t.Error("should deny partial perm code")
	}
}

func TestEnforcerWildcardAct(t *testing.T) {
	e := newTestEnforcer(t)
	_, _ = e.AddPolicy("admin", "sys:user:list", "*")

	allowed, _ := e.Enforce("admin", "sys:user:list", "access")
	if !allowed {
		t.Error("act=* should match any act")
	}
	allowed, _ = e.Enforce("admin", "sys:user:list", "other")
	if !allowed {
		t.Error("act=* should match 'other'")
	}
}

func TestEnforcerRoleInheritance(t *testing.T) {
	e := newTestEnforcer(t)
	_, _ = e.AddPolicy("admin", "sys:user:list", "access")
	_, _ = e.AddGroupingPolicy("alice", "admin")

	allowed, _ := e.Enforce("alice", "sys:user:list", "access")
	if !allowed {
		t.Error("alice should inherit admin's permissions")
	}

	allowed, _ = e.Enforce("bob", "sys:user:list", "access")
	if allowed {
		t.Error("bob should be denied")
	}
}

func TestEnforcerDefaultDeny(t *testing.T) {
	e := newTestEnforcer(t)
	// 无策略 → 默认拒绝
	allowed, _ := e.Enforce("anyone", "sys:user:list", "access")
	if allowed {
		t.Error("should deny by default (no policy)")
	}
}

// ---------------------------------------------------------------------------
// Authz 中间件测试（T-003b 改造版）
// ---------------------------------------------------------------------------

func TestAuthzMiddlewarePermCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg, _ := errcode.NewRegistry(11000)
	e := newTestEnforcer(t)

	_, _ = e.AddPolicy("admin", "sys:user:list", "access")
	_, _ = e.AddGroupingPolicy("alice", "admin")

	subjectFn := func(c *gin.Context) string { return c.GetHeader("X-Subject") }
	ae := NewAuthzEnforcer(e, AuthzConfig{}, subjectFn, reg)

	router := gin.New()
	router.GET("/sys/users", ae.RequirePerm("sys:user:list"), func(c *gin.Context) { c.String(200, "ok") })

	// alice（admin 角色）→ 允许
	req := httptest.NewRequest("GET", "/sys/users", nil)
	req.Header.Set("X-Subject", "alice")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// bob（无角色）→ 拒绝
	req = httptest.NewRequest("GET", "/sys/users", nil)
	req.Header.Set("X-Subject", "bob")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestAuthzSuperAdminBypass(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg, _ := errcode.NewRegistry(11000)
	e := newTestEnforcer(t)

	_, _ = e.AddGroupingPolicy("alice", "super_admin")

	subjectFn := func(c *gin.Context) string { return c.GetHeader("X-Subject") }
	ae := NewAuthzEnforcer(e, AuthzConfig{SuperAdminRoles: []string{"super_admin"}}, subjectFn, reg)

	router := gin.New()
	router.GET("/sys/users", ae.RequirePerm("sys:user:list"), func(c *gin.Context) { c.String(200, "ok") })

	// alice（super_admin）→ 放行（无需 p 规则）
	req := httptest.NewRequest("GET", "/sys/users", nil)
	req.Header.Set("X-Subject", "alice")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("super admin should be allowed, got %d", w.Code)
	}

	// bob → 拒绝
	req = httptest.NewRequest("GET", "/sys/users", nil)
	req.Header.Set("X-Subject", "bob")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Errorf("non-admin should be denied, got %d", w.Code)
	}
}

func TestAuthzNoPermCodePassThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	// 不挂任何 RequirePerm 的路由 → 直接可访问
	router.GET("/public", func(c *gin.Context) { c.String(200, "public") })

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/public", nil))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for route without RequirePerm, got %d", w.Code)
	}
}

func TestAuthzDoesNotLeakPolicyDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg, _ := errcode.NewRegistry(11000)
	e := newTestEnforcer(t)
	subjectFn := func(c *gin.Context) string { return "nobody" }
	ae := NewAuthzEnforcer(e, AuthzConfig{}, subjectFn, reg)

	router := gin.New()
	router.GET("/secret", ae.RequirePerm("sys:secret:view"), func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/secret", nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	if len(w.Body.String()) > 200 {
		t.Error("response too long — may leak policy details")
	}
}
