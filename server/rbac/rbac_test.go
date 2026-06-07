// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   Casbin RBAC 单元测试 — enforcer 加载 + 策略 + Authz 中间件
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:40:00
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

// newTestEnforcer 创建带空策略文件的 enforcer（不依赖 DB）。
func newTestEnforcer(t *testing.T) *casbin.Enforcer {
	t.Helper()
	tmpDir := t.TempDir()
	policyFile := filepath.Join(tmpDir, "policy.csv")
	// file adapter 需要文件存在
	if err := os.WriteFile(policyFile, []byte(""), 0o644); err != nil {
		t.Fatalf("create policy file: %v", err)
	}
	adapter := fileadapter.NewAdapter(policyFile)
	e, err := casbin.NewEnforcer(modelConfPath(), adapter)
	if err != nil {
		t.Fatalf("NewEnforcer: %v", err)
	}
	return e
}

// TestEnforcerWithFileAdapter 测试 model.conf 可被加载并正确执行策略。
func TestEnforcerWithFileAdapter(t *testing.T) {
	e := newTestEnforcer(t)

	// 无策略时应拒绝
	allowed, err := e.Enforce("alice", "/api/data", "GET")
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	if allowed {
		t.Error("should deny without any policy")
	}

	// 添加策略
	_, err = e.AddPolicy("alice", "/api/data", "GET")
	if err != nil {
		t.Fatalf("AddPolicy: %v", err)
	}

	// 现在应允许
	allowed, err = e.Enforce("alice", "/api/data", "GET")
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	if !allowed {
		t.Error("should allow alice GET /api/data")
	}

	// 其他方法应拒绝
	allowed, _ = e.Enforce("alice", "/api/data", "POST")
	if allowed {
		t.Error("should deny alice POST /api/data")
	}

	// 其他用户应拒绝
	allowed, _ = e.Enforce("bob", "/api/data", "GET")
	if allowed {
		t.Error("should deny bob GET /api/data")
	}
}

func TestEnforcerWildcardAct(t *testing.T) {
	e := newTestEnforcer(t)

	// act = "*" 应匹配所有方法
	_, _ = e.AddPolicy("admin", "/api/admin/*", "*")

	for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
		allowed, _ := e.Enforce("admin", "/api/admin/users", method)
		if !allowed {
			t.Errorf("admin should be allowed %s /api/admin/users", method)
		}
	}
}

func TestEnforcerRoleInheritance(t *testing.T) {
	e := newTestEnforcer(t)

	_, _ = e.AddPolicy("admin", "/api/data", "*")
	_, _ = e.AddGroupingPolicy("alice", "admin")

	// alice 通过角色继承应被允许
	allowed, _ := e.Enforce("alice", "/api/data", "GET")
	if !allowed {
		t.Error("alice should inherit admin's permissions")
	}

	// bob 无角色应被拒绝
	allowed, _ = e.Enforce("bob", "/api/data", "GET")
	if allowed {
		t.Error("bob should be denied without role")
	}
}

func TestKeyMatch2(t *testing.T) {
	e := newTestEnforcer(t)

	_, _ = e.AddPolicy("alice", "/api/users/:id", "GET")

	allowed, _ := e.Enforce("alice", "/api/users/123", "GET")
	if !allowed {
		t.Error("keyMatch2 should match /api/users/123")
	}

	allowed, _ = e.Enforce("alice", "/api/users/456", "GET")
	if !allowed {
		t.Error("keyMatch2 should match /api/users/456")
	}
}

// ---------------------------------------------------------------------------
// Authz 中间件测试
// ---------------------------------------------------------------------------

func TestAuthzMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg, _ := errcode.NewRegistry(11000)
	e := newTestEnforcer(t)

	_, _ = e.AddPolicy("alice", "/api/data", "GET")

	subjectFn := func(c *gin.Context) string {
		return c.GetHeader("X-Subject")
	}

	router := gin.New()
	router.Use(Authz(e, subjectFn, reg))
	router.GET("/api/data", func(c *gin.Context) { c.String(200, "ok") })
	router.POST("/api/data", func(c *gin.Context) { c.String(200, "ok") })

	// 允许
	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-Subject", "alice")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// 拒绝（未授权方法）
	req = httptest.NewRequest("POST", "/api/data", nil)
	req.Header.Set("X-Subject", "alice")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Errorf("expected 403, got %d: %s", w.Code, w.Body.String())
	}

	// 拒绝（未授权用户）
	req = httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-Subject", "bob")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestAuthzDoesNotLeakPolicyDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg, _ := errcode.NewRegistry(11000)
	e := newTestEnforcer(t)

	router := gin.New()
	router.Use(Authz(e, func(c *gin.Context) string { return "nobody" }, reg))
	router.GET("/secret", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/secret", nil))

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}

	body := w.Body.String()
	if len(body) > 200 {
		t.Error("response body too long — may leak policy details")
	}
}
