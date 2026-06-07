// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   C 端加密中间件 — 请求解密 + 响应加密 Gin HandlerFunc
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:05:00
// +----------------------------------------------------------------------

package crypto

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/gin-gonic/gin"
)

// Config 是加密中间件的配置，由 main/bootstrap 从应用配置映射后注入。
// crypto 包不 import config 包。
type Config struct {
	HeaderPrefix        string // 安全头前缀，默认 "X-Ca-"
	AESKey              []byte // 已解码，必须恰好 32 字节
	HMACKey             []byte // 已解码，建议 32 字节
	ReplayWindowSeconds int64  // 时间窗，默认 300
	NonceTTLSeconds     int64  // nonce Redis TTL，默认 600
	RedisKeyPrefix      string // Redis key 前缀，如 "myapp"
}

// Validate 校验配置合法性，启动时 fail-fast。
func (c Config) Validate() error {
	if len(c.AESKey) != 32 {
		return fmt.Errorf("crypto: AES key must be exactly 32 bytes, got %d", len(c.AESKey))
	}
	if len(c.HMACKey) == 0 {
		return fmt.Errorf("crypto: HMAC key must not be empty")
	}
	return nil
}

func (c Config) headerName(suffix string) string {
	prefix := c.HeaderPrefix
	if prefix == "" {
		prefix = "X-Ca-"
	}
	return prefix + suffix
}

func (c Config) replayWindow() int64 {
	if c.ReplayWindowSeconds <= 0 {
		return 300
	}
	return c.ReplayWindowSeconds
}

func (c Config) nonceTTL() time.Duration {
	ttl := c.NonceTTLSeconds
	if ttl <= 0 {
		ttl = 600
	}
	return time.Duration(ttl) * time.Second
}

func (c Config) nonceKey(nonce string) string {
	prefix := c.RedisKeyPrefix
	if prefix == "" {
		prefix = "app"
	}
	return prefix + ":sec:nonce:" + nonce
}

// NowFunc 允许测试注入时间，默认 time.Now。
var NowFunc = time.Now

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

// Middleware 是加密中间件实例，持有配置、存储和错误码。
type Middleware struct {
	cfg   Config
	store ReplayStore
	errs  *errcode.Registry
}

// NewMiddleware 创建加密中间件实例。
func NewMiddleware(cfg Config, store ReplayStore, errs *errcode.Registry) (*Middleware, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if store == nil {
		return nil, fmt.Errorf("crypto: ReplayStore must not be nil")
	}
	if errs == nil {
		return nil, fmt.Errorf("crypto: errcode.Registry must not be nil")
	}
	return &Middleware{cfg: cfg, store: store, errs: errs}, nil
}

// Handler 返回可挂载到任意路由组的 Gin 中间件。
// 入站：验头 → 验时间 → 验签名 → 验 nonce → 解密 → 写回 body。
// 出站：加密 handler 输出 → 补安全头。
func (m *Middleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// ---- 入站：请求解密 ----
		if !m.decryptRequest(c) {
			return // 已 abort
		}

		// ---- 包裹响应写入器 ----
		w := &encryptResponseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
			status:         200,
		}
		c.Writer = w

		c.Next()

		// ---- 出站：响应加密 ----
		if c.IsAborted() && w.body.Len() == 0 {
			return
		}
		if w.body.Len() > 0 {
			m.encryptResponse(c, w)
		}
	}
}

// ---------------------------------------------------------------------------
// 请求解密流程（严格按校验顺序：缺头 → 时间 → 签名 → nonce → 解密）
// ---------------------------------------------------------------------------

func (m *Middleware) decryptRequest(c *gin.Context) bool {
	hdrTimestamp := m.cfg.headerName("Timestamp")
	hdrNonce := m.cfg.headerName("Nonce")
	hdrSignature := m.cfg.headerName("Signature")

	// 1. 缺任一安全头 → ERR_MISSING_SECURITY_HEADERS
	timestamp := c.GetHeader(hdrTimestamp)
	nonce := c.GetHeader(hdrNonce)
	signature := c.GetHeader(hdrSignature)
	if timestamp == "" || nonce == "" || signature == "" {
		m.abort(c, m.errs.ErrMissingSecurityHeaders)
		return false
	}

	// 读取并解析请求体 {"data":"..."}
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		m.abort(c, m.errs.ErrDecryptFailed)
		return false
	}
	_ = c.Request.Body.Close()

	var envelope struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(rawBody, &envelope); err != nil || envelope.Data == "" {
		m.abort(c, m.errs.ErrDecryptFailed)
		return false
	}
	bodyB64 := envelope.Data

	// 2. 时间窗校验 → ERR_TIMESTAMP_EXPIRED
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		m.abort(c, m.errs.ErrTimestampExpired)
		return false
	}
	now := NowFunc().Unix()
	diff := now - ts
	if diff < 0 {
		diff = -diff
	}
	if diff > m.cfg.replayWindow() {
		m.abort(c, m.errs.ErrTimestampExpired)
		return false
	}

	// 3. 签名验证（hmac.Equal 恒定时间比较）→ ERR_SIGN_INVALID
	signingString := BuildSigningString(c.Request.Method, c.Request.URL.Path, timestamp, nonce, bodyB64)
	if !VerifyEnvelope(m.cfg.HMACKey, signingString, signature) {
		m.abort(c, m.errs.ErrSignInvalid)
		return false
	}

	// 4. Nonce 防重放（先验签名后落 nonce）→ ERR_NONCE_REPLAY
	exists, err := m.store.CheckAndSet(c.Request.Context(), m.cfg.nonceKey(nonce), m.cfg.nonceTTL())
	if err != nil || exists {
		m.abort(c, m.errs.ErrNonceReplay)
		return false
	}

	// 5. 解密 → ERR_DECRYPT_FAILED
	plaintext, err := DecryptBody(m.cfg.AESKey, bodyB64)
	if err != nil {
		m.abort(c, m.errs.ErrDecryptFailed)
		return false
	}

	// 写回解密后的明文 body 供 handler 读取
	c.Request.Body = io.NopCloser(bytes.NewReader(plaintext))
	c.Request.ContentLength = int64(len(plaintext))
	return true
}

// ---------------------------------------------------------------------------
// 响应加密流程
// ---------------------------------------------------------------------------

func (m *Middleware) encryptResponse(c *gin.Context, w *encryptResponseWriter) {
	plaintext := w.body.Bytes()

	bodyB64, err := EncryptBody(m.cfg.AESKey, plaintext)
	if err != nil {
		// 加密失败回退：直接写明文（不应发生）
		w.ResponseWriter.WriteHeader(w.status)
		_, _ = w.ResponseWriter.Write(plaintext)
		return
	}

	now := NowFunc().Unix()
	timestamp := strconv.FormatInt(now, 10)

	nonceBytes := make([]byte, 16)
	_, _ = rand.Read(nonceBytes)
	nonce := base64.RawURLEncoding.EncodeToString(nonceBytes)

	signingString := BuildSigningString(c.Request.Method, c.Request.URL.Path, timestamp, nonce, bodyB64)
	sig := SignEnvelope(m.cfg.HMACKey, signingString)

	// 写安全响应头
	header := w.ResponseWriter.Header()
	header.Set(m.cfg.headerName("Timestamp"), timestamp)
	header.Set(m.cfg.headerName("Nonce"), nonce)
	header.Set(m.cfg.headerName("Signature"), sig)
	header.Set("Content-Type", "application/json; charset=utf-8")

	resp, _ := json.Marshal(map[string]string{"data": bodyB64})
	header.Set("Content-Length", strconv.Itoa(len(resp)))
	w.ResponseWriter.WriteHeader(w.status)
	_, _ = w.ResponseWriter.Write(resp)
}

// ---------------------------------------------------------------------------
// 响应写入器包裹（拦截 handler 输出，缓存到 buffer）
// ---------------------------------------------------------------------------

type encryptResponseWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (w *encryptResponseWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func (w *encryptResponseWriter) WriteHeader(code int) {
	w.status = code
}

func (w *encryptResponseWriter) WriteString(s string) (int, error) {
	return w.body.WriteString(s)
}

// ---------------------------------------------------------------------------
// 辅助
// ---------------------------------------------------------------------------

func (m *Middleware) abort(c *gin.Context, e errcode.Error) {
	c.AbortWithStatusJSON(e.HTTP, gin.H{
		"code":    e.Code,
		"message": e.Message,
	})
}
