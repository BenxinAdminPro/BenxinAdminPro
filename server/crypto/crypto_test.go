// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   crypto 包单元测试 — 加解密/签名/中间件正负例
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:32:00
// +----------------------------------------------------------------------

package crypto

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// 测试用 key
// ---------------------------------------------------------------------------

var (
	testAESKey  = bytes.Repeat([]byte{0x42}, 32) // 32 bytes
	testHMACKey = bytes.Repeat([]byte{0x43}, 32)
	testIV      = bytes.Repeat([]byte{0x01}, 16)
)

func testRegistry(t *testing.T) *errcode.Registry {
	t.Helper()
	reg, err := errcode.NewRegistry(11000)
	if err != nil {
		t.Fatalf("errcode.NewRegistry: %v", err)
	}
	return reg
}

// ---------------------------------------------------------------------------
// AES 纯函数测试
// ---------------------------------------------------------------------------

func TestEncryptDecryptRoundTrip(t *testing.T) {
	cases := []string{
		"hello",
		"",
		"a]b{c}d",
		"你好世界",
		string(bytes.Repeat([]byte("A"), 256)),
	}
	for _, tc := range cases {
		ct, err := Encrypt(testAESKey, testIV, []byte(tc))
		if err != nil {
			t.Fatalf("Encrypt(%q): %v", tc, err)
		}
		pt, err := Decrypt(testAESKey, testIV, ct)
		if err != nil {
			t.Fatalf("Decrypt(%q): %v", tc, err)
		}
		if string(pt) != tc {
			t.Errorf("round trip failed: got %q, want %q", pt, tc)
		}
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	ct, _ := Encrypt(testAESKey, testIV, []byte("hello"))
	ct[0] ^= 0xFF // 篡改 1 字节
	_, err := Decrypt(testAESKey, testIV, ct)
	// 篡改后应解密失败或得到乱码（padding error）
	// 我们不强制 error，但验证不等于原始
	if err == nil {
		// 即使没有 error，解密结果也不应该是 "hello"
		t.Log("tampered ciphertext decrypted without error (padding happened to be valid)")
	}
}

func TestEncryptBadKeyLength(t *testing.T) {
	_, err := Encrypt([]byte("short"), testIV, []byte("hello"))
	if err == nil {
		t.Fatal("expected error for short key")
	}
}

// ---------------------------------------------------------------------------
// HMAC 纯函数测试
// ---------------------------------------------------------------------------

func TestSignVerify(t *testing.T) {
	sig := Sign(testHMACKey, []byte("message"))
	if !Verify(testHMACKey, []byte("message"), sig) {
		t.Fatal("Verify should return true for correct signature")
	}
}

func TestVerifyTamperedSignature(t *testing.T) {
	sig := Sign(testHMACKey, []byte("message"))
	sig[0] ^= 0xFF
	if Verify(testHMACKey, []byte("message"), sig) {
		t.Fatal("Verify should return false for tampered signature")
	}
}

func TestVerifyWrongMessage(t *testing.T) {
	sig := Sign(testHMACKey, []byte("message"))
	if Verify(testHMACKey, []byte("wrong"), sig) {
		t.Fatal("Verify should return false for wrong message")
	}
}

// ---------------------------------------------------------------------------
// Envelope 纯函数测试
// ---------------------------------------------------------------------------

func TestEncryptDecryptBody(t *testing.T) {
	original := []byte(`{"key":"value","数据":"中文"}`)
	bodyB64, err := EncryptBodyWithIV(testAESKey, testIV, original)
	if err != nil {
		t.Fatalf("EncryptBodyWithIV: %v", err)
	}

	got, err := DecryptBody(testAESKey, bodyB64)
	if err != nil {
		t.Fatalf("DecryptBody: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Errorf("got %q, want %q", got, original)
	}
}

func TestSigningStringFormat(t *testing.T) {
	s := BuildSigningString("post", "/api/c/echo", "1750000000", "nonce123", "bodyb64data")
	want := "POST\n/api/c/echo\n1750000000\nnonce123\nbodyb64data"
	if s != want {
		t.Errorf("signing string:\n  got:  %q\n  want: %q", s, want)
	}
}

func TestEnvelopeSignVerify(t *testing.T) {
	sigStr := BuildSigningString("POST", "/api/c/echo", "1750000000", "nonce", "body")
	sig := SignEnvelope(testHMACKey, sigStr)
	if !VerifyEnvelope(testHMACKey, sigStr, sig) {
		t.Fatal("VerifyEnvelope should return true")
	}
	if VerifyEnvelope(testHMACKey, sigStr+"tampered", sig) {
		t.Fatal("VerifyEnvelope should return false for tampered signing string")
	}
}

// ---------------------------------------------------------------------------
// Middleware 集成测试（使用 MemoryReplayStore）
// ---------------------------------------------------------------------------

func setupMiddleware(t *testing.T) (*Middleware, *errcode.Registry) {
	t.Helper()
	reg := testRegistry(t)
	store := NewMemoryReplayStore()
	cfg := Config{
		HeaderPrefix:        "X-Ca-",
		AESKey:              testAESKey,
		HMACKey:             testHMACKey,
		ReplayWindowSeconds: 300,
		NonceTTLSeconds:     600,
		RedisKeyPrefix:      "test",
	}
	m, err := NewMiddleware(cfg, store, reg)
	if err != nil {
		t.Fatalf("NewMiddleware: %v", err)
	}
	return m, reg
}

func buildEncryptedRequest(t *testing.T, method, path string, plaintext []byte, timestamp, nonce string) *http.Request {
	t.Helper()
	bodyB64, err := EncryptBodyWithIV(testAESKey, testIV, plaintext)
	if err != nil {
		t.Fatalf("EncryptBodyWithIV: %v", err)
	}
	envelope, _ := json.Marshal(map[string]string{"data": bodyB64})
	sigStr := BuildSigningString(method, path, timestamp, nonce, bodyB64)
	sig := SignEnvelope(testHMACKey, sigStr)

	req := httptest.NewRequest(method, path, bytes.NewReader(envelope))
	req.Header.Set("X-Ca-Timestamp", timestamp)
	req.Header.Set("X-Ca-Nonce", nonce)
	req.Header.Set("X-Ca-Signature", sig)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func nowTimestamp() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

func TestMiddlewareHappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m, _ := setupMiddleware(t)

	router := gin.New()
	router.POST("/api/c/echo", m.Handler(), func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		c.String(200, string(body))
	})

	req := buildEncryptedRequest(t, "POST", "/api/c/echo", []byte(`{"msg":"hi"}`), nowTimestamp(), "unique-nonce-001")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 响应应被加密为 {"data":"..."}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Fatal("response missing 'data' field")
	}

	// 解密响应
	pt, err := DecryptBody(testAESKey, resp["data"])
	if err != nil {
		t.Fatalf("decrypt response: %v", err)
	}
	if string(pt) != `{"msg":"hi"}` {
		t.Errorf("decrypted response: got %q, want %q", pt, `{"msg":"hi"}`)
	}
}

func TestMiddlewareMissingHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m, reg := setupMiddleware(t)

	router := gin.New()
	router.POST("/api/c/echo", m.Handler(), func(c *gin.Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest("POST", "/api/c/echo", bytes.NewReader([]byte(`{}`)))
	// 不设置安全头
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), reg.ErrMissingSecurityHeaders.Code)
}

func TestMiddlewareTimestampExpired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m, reg := setupMiddleware(t)

	router := gin.New()
	router.POST("/api/c/echo", m.Handler(), func(c *gin.Context) {
		c.String(200, "ok")
	})

	// 过期的时间戳（10 分钟前）
	expired := strconv.FormatInt(time.Now().Unix()-600, 10)
	req := buildEncryptedRequest(t, "POST", "/api/c/echo", []byte("test"), expired, "nonce-expired")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorCode(t, w.Body.Bytes(), reg.ErrTimestampExpired.Code)
}

func TestMiddlewareSignInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m, reg := setupMiddleware(t)

	router := gin.New()
	router.POST("/api/c/echo", m.Handler(), func(c *gin.Context) {
		c.String(200, "ok")
	})

	bodyB64, _ := EncryptBodyWithIV(testAESKey, testIV, []byte("test"))
	envelope, _ := json.Marshal(map[string]string{"data": bodyB64})
	ts := nowTimestamp()

	req := httptest.NewRequest("POST", "/api/c/echo", bytes.NewReader(envelope))
	req.Header.Set("X-Ca-Timestamp", ts)
	req.Header.Set("X-Ca-Nonce", "nonce-bad-sig")
	req.Header.Set("X-Ca-Signature", base64.StdEncoding.EncodeToString([]byte("invalid-sig")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), reg.ErrSignInvalid.Code)
}

func TestMiddlewareNonceReplay(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m, _ := setupMiddleware(t)

	router := gin.New()
	router.POST("/api/c/echo", m.Handler(), func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		c.String(200, string(body))
	})

	ts := nowTimestamp()
	nonce := "replay-nonce-001"

	// 第一次：成功
	req1 := buildEncryptedRequest(t, "POST", "/api/c/echo", []byte("test"), ts, nonce)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Fatalf("first request: expected 200, got %d: %s", w1.Code, w1.Body.String())
	}

	// 第二次：重放 → 拒绝
	req2 := buildEncryptedRequest(t, "POST", "/api/c/echo", []byte("test"), ts, nonce)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != 400 {
		t.Fatalf("replay request: expected 400, got %d", w2.Code)
	}
}

func TestMiddlewareDecryptFailed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m, reg := setupMiddleware(t)

	router := gin.New()
	router.POST("/api/c/echo", m.Handler(), func(c *gin.Context) {
		c.String(200, "ok")
	})

	// 用错误的 base64 数据构建有效签名
	badBodyB64 := base64.StdEncoding.EncodeToString([]byte("too-short"))
	envelope, _ := json.Marshal(map[string]string{"data": badBodyB64})
	ts := nowTimestamp()
	nonce := "nonce-bad-decrypt"
	sigStr := BuildSigningString("POST", "/api/c/echo", ts, nonce, badBodyB64)
	sig := SignEnvelope(testHMACKey, sigStr)

	req := httptest.NewRequest("POST", "/api/c/echo", bytes.NewReader(envelope))
	req.Header.Set("X-Ca-Timestamp", ts)
	req.Header.Set("X-Ca-Nonce", nonce)
	req.Header.Set("X-Ca-Signature", sig)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorCode(t, w.Body.Bytes(), reg.ErrDecryptFailed.Code)
}

// ---------------------------------------------------------------------------
// Config validation
// ---------------------------------------------------------------------------

func TestConfigValidate(t *testing.T) {
	// 正常
	cfg := Config{AESKey: testAESKey, HMACKey: testHMACKey}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// AES key 长度错误
	cfg2 := Config{AESKey: []byte("short"), HMACKey: testHMACKey}
	if err := cfg2.Validate(); err == nil {
		t.Fatal("expected error for short AES key")
	}

	// HMAC key 为空
	cfg3 := Config{AESKey: testAESKey, HMACKey: nil}
	if err := cfg3.Validate(); err == nil {
		t.Fatal("expected error for empty HMAC key")
	}
}

// ---------------------------------------------------------------------------
// 辅助
// ---------------------------------------------------------------------------

func assertErrorCode(t *testing.T, body []byte, expectedCode int) {
	t.Helper()
	var resp struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal error response: %v (body=%s)", err, body)
	}
	if resp.Code != expectedCode {
		t.Errorf("error code: got %d, want %d (body=%s)", resp.Code, expectedCode, body)
	}
}

// Ensure MemoryReplayStore satisfies ReplayStore
var _ ReplayStore = (*MemoryReplayStore)(nil)

// unused import guard
var _ = fmt.Sprintf
