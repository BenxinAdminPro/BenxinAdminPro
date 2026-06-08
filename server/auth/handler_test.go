// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   认证 handler httptest — 4 路由正常/错误响应
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 17:26:00
// +----------------------------------------------------------------------

package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/gin-gonic/gin"
)

func setupRouter(t *testing.T) (*gin.Engine, *errcode.Registry) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	svc, reg, _ := setupAuthService(t)
	h := NewHandler(svc, reg)

	r := gin.New()
	h.RegisterRoutes(&r.RouterGroup)
	return r, reg
}

func postJSON(router *gin.Engine, path string, body any, headers ...string) *httptest.ResponseRecorder {
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

type apiResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func parseResp(t *testing.T, w *httptest.ResponseRecorder) apiResp {
	t.Helper()
	var r apiResp
	if err := json.Unmarshal(w.Body.Bytes(), &r); err != nil {
		t.Fatalf("parse response: %v (body=%s)", err, w.Body.String())
	}
	return r
}

func TestHandlerCaptcha(t *testing.T) {
	router, _ := setupRouter(t)
	w := postJSON(router, "/auth/captcha", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseResp(t, w)
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
}

func TestHandlerPrecheck(t *testing.T) {
	router, _ := setupRouter(t)
	// 全新用户：captcha_required 应为 false
	w := postJSON(router, "/auth/precheck", map[string]string{"username": "alice"})
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseResp(t, w)
	var data map[string]any
	json.Unmarshal(resp.Data, &data)
	if data["captcha_required"] != false {
		t.Errorf("fresh user captcha_required should be false, got %v", data["captcha_required"])
	}

	// 缺 username → 400
	w = postJSON(router, "/auth/precheck", map[string]string{})
	if w.Code != http.StatusBadRequest {
		t.Errorf("missing username should be 400, got %d", w.Code)
	}
}

func TestHandlerLoginSuccess(t *testing.T) {
	router, _ := setupRouter(t)
	w := postJSON(router, "/auth/login", map[string]string{
		"username": "alice", "password": "correct-password",
	})
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseResp(t, w)
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
	// 检查 data 中有 access_token
	var data map[string]any
	json.Unmarshal(resp.Data, &data)
	if _, ok := data["access_token"]; !ok {
		t.Error("response data should contain access_token")
	}
	if data["token_type"] != "Bearer" {
		t.Errorf("token_type should be Bearer, got %v", data["token_type"])
	}
}

func TestHandlerLoginBadCredentials(t *testing.T) {
	router, reg := setupRouter(t)
	w := postJSON(router, "/auth/login", map[string]string{
		"username": "alice", "password": "wrong",
	})
	if w.Code != 401 {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseResp(t, w)
	if resp.Code != reg.ErrBadCredentials.Code {
		t.Errorf("expected code %d, got %d", reg.ErrBadCredentials.Code, resp.Code)
	}
}

func TestHandlerLoginMissingFields(t *testing.T) {
	router, _ := setupRouter(t)
	w := postJSON(router, "/auth/login", map[string]string{})
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandlerRefresh(t *testing.T) {
	router, _ := setupRouter(t)

	// 先登录
	w1 := postJSON(router, "/auth/login", map[string]string{
		"username": "alice", "password": "correct-password",
	})
	resp1 := parseResp(t, w1)
	var data1 map[string]any
	json.Unmarshal(resp1.Data, &data1)
	refreshToken := data1["refresh_token"].(string)

	// 刷新
	w2 := postJSON(router, "/auth/refresh", map[string]string{
		"refresh_token": refreshToken,
	})
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestHandlerLogout(t *testing.T) {
	router, _ := setupRouter(t)

	// 先登录
	w1 := postJSON(router, "/auth/login", map[string]string{
		"username": "alice", "password": "correct-password",
	})
	resp1 := parseResp(t, w1)
	var data1 map[string]any
	json.Unmarshal(resp1.Data, &data1)
	accessToken := data1["access_token"].(string)

	// 登出
	w2 := postJSON(router, "/auth/logout", map[string]string{},
		"Authorization", "Bearer "+accessToken)
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestHandlerLogoutNoToken(t *testing.T) {
	router, _ := setupRouter(t)
	w := postJSON(router, "/auth/logout", map[string]string{})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHandlerUnifiedEnvelope(t *testing.T) {
	router, _ := setupRouter(t)
	// 验证响应始终是 { code, message, data }
	w := postJSON(router, "/auth/login", map[string]string{
		"username": "alice", "password": "wrong",
	})
	resp := parseResp(t, w)
	if resp.Message == "" {
		t.Error("response should always have message")
	}
}
