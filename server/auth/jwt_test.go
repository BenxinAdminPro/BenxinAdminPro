// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   JWT TokenService 单元测试 — 签发/解析/校验/刷新/注销
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:38:00
// +----------------------------------------------------------------------

package auth

import (
	"context"
	"testing"
	"time"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
)

func testService(t *testing.T) (TokenService, *errcode.Registry) {
	t.Helper()
	reg, err := errcode.NewRegistry(11000)
	if err != nil {
		t.Fatalf("errcode.NewRegistry: %v", err)
	}
	cfg := Config{
		Issuer:            "test-issuer",
		AccessSecret:      "test-access-secret-32bytes!!!!!!", // 足够长
		RefreshSecret:     "test-refresh-secret-32bytes!!!!!", // 和 access 不同
		AccessTTLSeconds:  3600,
		RefreshTTLSeconds: 86400,
		RefreshRotate:     true,
		RedisKeyPrefix:    "test",
	}
	store := NewMemoryBlacklistStore()
	svc, err := NewTokenService(cfg, store, reg)
	if err != nil {
		t.Fatalf("NewTokenService: %v", err)
	}
	return svc, reg
}

func TestIssuePairAndParse(t *testing.T) {
	svc, _ := testService(t)
	ctx := context.Background()

	pair, err := svc.IssuePair(ctx, "user-123", map[string]any{"role": "admin"})
	if err != nil {
		t.Fatalf("IssuePair: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("tokens should not be empty")
	}
	if pair.AccessToken == pair.RefreshToken {
		t.Fatal("access and refresh tokens should differ")
	}

	// Parse access
	claims, err := svc.Parse(pair.AccessToken)
	if err != nil {
		t.Fatalf("Parse access: %v", err)
	}
	if claims.Subject != "user-123" {
		t.Errorf("subject: got %q, want %q", claims.Subject, "user-123")
	}
	if claims.TokenType != TokenTypeAccess {
		t.Errorf("token type: got %q, want %q", claims.TokenType, TokenTypeAccess)
	}
	if claims.Issuer != "test-issuer" {
		t.Errorf("issuer: got %q, want %q", claims.Issuer, "test-issuer")
	}
	if claims.JTI == "" {
		t.Error("jti should not be empty")
	}

	// Parse refresh
	rClaims, err := svc.Parse(pair.RefreshToken)
	if err != nil {
		t.Fatalf("Parse refresh: %v", err)
	}
	if rClaims.TokenType != TokenTypeRefresh {
		t.Errorf("token type: got %q, want %q", rClaims.TokenType, TokenTypeRefresh)
	}
}

func TestVerifyAccess(t *testing.T) {
	svc, _ := testService(t)
	ctx := context.Background()

	pair, _ := svc.IssuePair(ctx, "user-456", nil)

	claims, err := svc.Verify(ctx, pair.AccessToken, TokenTypeAccess)
	if err != nil {
		t.Fatalf("Verify access: %v", err)
	}
	if claims.Subject != "user-456" {
		t.Errorf("subject: got %q, want %q", claims.Subject, "user-456")
	}
}

func TestVerifyWrongTokenType(t *testing.T) {
	svc, reg := testService(t)
	ctx := context.Background()

	pair, _ := svc.IssuePair(ctx, "user-789", nil)

	// 用 access token 做 refresh 类型校验 → 应失败
	_, err := svc.Verify(ctx, pair.AccessToken, TokenTypeRefresh)
	if err == nil {
		t.Fatal("expected error for wrong token type")
	}
	assertIsError(t, err, reg.ErrTokenInvalid)
}

func TestVerifyExpiredToken(t *testing.T) {
	svc, reg := testService(t)
	ctx := context.Background()

	// 注入过去的时间
	origNow := NowFunc
	NowFunc = func() time.Time { return time.Now().Add(-2 * time.Hour) }
	pair, _ := svc.IssuePair(ctx, "user-exp", nil)
	NowFunc = origNow

	// access 应已过期（签发时 iat = 2小时前，TTL=1小时 → exp = 1小时前）
	_, err := svc.Verify(ctx, pair.AccessToken, TokenTypeAccess)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
	assertIsError(t, err, reg.ErrTokenExpired)
}

func TestRevokeAndVerify(t *testing.T) {
	svc, reg := testService(t)
	ctx := context.Background()

	pair, _ := svc.IssuePair(ctx, "user-revoke", nil)
	claims, _ := svc.Parse(pair.AccessToken)

	// Revoke access token
	if err := svc.Revoke(ctx, claims.JTI, claims.ExpiresAt); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	// Verify 应返回 ERR_TOKEN_REVOKED
	_, err := svc.Verify(ctx, pair.AccessToken, TokenTypeAccess)
	if err == nil {
		t.Fatal("expected error for revoked token")
	}
	assertIsError(t, err, reg.ErrTokenRevoked)
}

func TestRefreshRotation(t *testing.T) {
	svc, reg := testService(t)
	ctx := context.Background()

	pair1, _ := svc.IssuePair(ctx, "user-refresh", nil)

	// Refresh → 获得新令牌对
	pair2, err := svc.Refresh(ctx, pair1.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if pair2.AccessToken == pair1.AccessToken {
		t.Error("new access token should differ from old")
	}
	if pair2.RefreshToken == pair1.RefreshToken {
		t.Error("new refresh token should differ from old (rotation)")
	}

	// 旧 refresh token 应已被拉黑
	_, err = svc.Verify(ctx, pair1.RefreshToken, TokenTypeRefresh)
	if err == nil {
		t.Fatal("old refresh token should be revoked after rotation")
	}
	assertIsError(t, err, reg.ErrTokenRevoked)

	// 新 refresh token 应可用
	_, err = svc.Verify(ctx, pair2.RefreshToken, TokenTypeRefresh)
	if err != nil {
		t.Fatalf("new refresh token should be valid: %v", err)
	}
}

func TestConfigValidation(t *testing.T) {
	reg, _ := errcode.NewRegistry(11000)
	store := NewMemoryBlacklistStore()

	// 空 issuer
	_, err := NewTokenService(Config{AccessSecret: "s", RefreshSecret: "s"}, store, reg)
	if err == nil {
		t.Fatal("expected error for empty issuer")
	}

	// 空 access_secret
	_, err = NewTokenService(Config{Issuer: "i", RefreshSecret: "s"}, store, reg)
	if err == nil {
		t.Fatal("expected error for empty access_secret")
	}

	// 空 refresh_secret
	_, err = NewTokenService(Config{Issuer: "i", AccessSecret: "s"}, store, reg)
	if err == nil {
		t.Fatal("expected error for empty refresh_secret")
	}
}

func TestErrcodeRegistryValidation(t *testing.T) {
	// segmentBase <= 0
	_, err := errcode.NewRegistry(0)
	if err == nil {
		t.Fatal("expected error for segmentBase = 0")
	}
	_, err = errcode.NewRegistry(-1)
	if err == nil {
		t.Fatal("expected error for segmentBase = -1")
	}
}

// ---------------------------------------------------------------------------
// 辅助
// ---------------------------------------------------------------------------

func assertIsError(t *testing.T, err error, expected errcode.Error) {
	t.Helper()
	ecErr, ok := err.(errcode.Error)
	if !ok {
		t.Errorf("expected errcode.Error, got %T: %v", err, err)
		return
	}
	if ecErr.Code != expected.Code {
		t.Errorf("error code: got %d, want %d", ecErr.Code, expected.Code)
	}
}

// Ensure MemoryBlacklistStore satisfies BlacklistStore
var _ BlacklistStore = (*MemoryBlacklistStore)(nil)
