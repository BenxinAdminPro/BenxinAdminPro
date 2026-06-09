// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   system handler 鉴权接线测试 — 验 RegisterRoutes 每路由挂上 PermGuard
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 13:00:00
// +----------------------------------------------------------------------
//
// T-003d：system 包 handler 不直接依赖 rbac，经 PermGuard 接口注入鉴权中间件。
// 本测试用记录式假 guard 验证：① 每条受保护路由都向 guard 申领了 perm code；
// ② guard 返回的中间件确实挂在 handler 之前（无权 → 403，handler 体未触达）。

package system

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// recordingGuard 记录所有被申领的 perm code，并返回一个一律 403 的中间件。
// 用于验证"路由确实把鉴权挂在 handler 之前"——若中间件没挂上，请求会触达 nil 服务 panic 而非干净 403。
type recordingGuard struct {
	codes []string
}

func (g *recordingGuard) RequirePerm(code string) gin.HandlerFunc {
	g.codes = append(g.codes, code)
	return func(c *gin.Context) { c.AbortWithStatus(http.StatusForbidden) }
}

func TestSystemHandlerRegisterRoutesWiresGuard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	g := &recordingGuard{}
	h := NewHandler(nil, nil, nil, reg(t), nil) // 服务/hasher nil：403 路径不触达 handler 体
	router := gin.New()
	h.RegisterRoutes(&router.RouterGroup, g)

	// 16 条系统管理路由，每条都应向 guard 申领过 perm code
	if len(g.codes) != 16 {
		t.Fatalf("expected 16 routes wired to guard, got %d: %v", len(g.codes), g.codes)
	}
	for _, want := range []string{"sys:dict:list", "sys:config:create", "sys:operlog:clean", "sys:loginlog:list"} {
		if !contains(g.codes, want) {
			t.Errorf("perm code %q not wired", want)
		}
	}

	// 任取一条写路由：guard 中间件在 handler 前生效 → 403（而非触达 nil 服务 panic）
	req := httptest.NewRequest("POST", "/sys/configs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected guard to block before handler (403), got %d", w.Code)
	}
}

func TestFileHandlerRegisterRoutesWiresGuard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	g := &recordingGuard{}
	h := NewFileHandler(nil, reg(t), nil)
	router := gin.New()
	h.RegisterRoutes(&router.RouterGroup, g)

	if len(g.codes) != 4 {
		t.Fatalf("expected 4 file routes wired to guard, got %d: %v", len(g.codes), g.codes)
	}
	for _, want := range []string{"sys:file:upload", "sys:file:download", "sys:file:list", "sys:file:delete"} {
		if !contains(g.codes, want) {
			t.Errorf("perm code %q not wired", want)
		}
	}

	req := httptest.NewRequest("DELETE", "/sys/files/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected guard to block before handler (403), got %d", w.Code)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
