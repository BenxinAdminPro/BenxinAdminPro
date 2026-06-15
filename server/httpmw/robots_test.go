// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   X-Robots-Tag 中间件单测 — 默认安全四态（开/关/自定义/缺省内容）
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-15 14:30:00
// +----------------------------------------------------------------------

package httpmw

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

// runWith 用给定配置跑一次请求，返回响应的 X-Robots-Tag 头值。
func runWith(cfg RobotsConfig) string {
	r := gin.New()
	r.Use(XRobotsTag(cfg))
	r.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(w, req)
	return w.Header().Get("X-Robots-Tag")
}

// ① 配置开（默认内容）→ 响应含 noindex, nofollow。
func TestXRobotsTag_EnabledDefaultContent(t *testing.T) {
	got := runWith(RobotsConfig{Enabled: true, Content: DefaultXRobotsTag})
	if got != "noindex, nofollow" {
		t.Fatalf("got %q, want %q", got, "noindex, nofollow")
	}
}

// ② 配置关 → 响应不含该头。
func TestXRobotsTag_Disabled(t *testing.T) {
	got := runWith(RobotsConfig{Enabled: false, Content: DefaultXRobotsTag})
	if got != "" {
		t.Fatalf("disabled 应不注入头，got %q", got)
	}
}

// ③ 自定义内容 → 注入自定义值。
func TestXRobotsTag_CustomContent(t *testing.T) {
	got := runWith(RobotsConfig{Enabled: true, Content: "noindex"})
	if got != "noindex" {
		t.Fatalf("got %q, want %q", got, "noindex")
	}
}

// ④ 开启但内容缺省（空串）→ 回退默认安全内容（坐实「开了不会注入空头」）。
func TestXRobotsTag_EnabledEmptyContentFallsBackToDefault(t *testing.T) {
	got := runWith(RobotsConfig{Enabled: true, Content: ""})
	if got != DefaultXRobotsTag {
		t.Fatalf("空内容应回退 DefaultXRobotsTag，got %q want %q", got, DefaultXRobotsTag)
	}
}

// DefaultXRobotsTag 常量值锁定（默认安全：禁收录 + 禁跟随）。
func TestDefaultXRobotsTagValue(t *testing.T) {
	if DefaultXRobotsTag != "noindex, nofollow" {
		t.Fatalf("DefaultXRobotsTag = %q, want %q", DefaultXRobotsTag, "noindex, nofollow")
	}
}
