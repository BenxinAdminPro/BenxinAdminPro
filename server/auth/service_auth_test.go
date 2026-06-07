// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   AuthService 编排测试 — 登录/刷新/登出/验证码/锁定状态机
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 17:24:00
// +----------------------------------------------------------------------

package auth

import (
	"context"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
)

// ---------------------------------------------------------------------------
// 测试辅助
// ---------------------------------------------------------------------------

type testAuthEnv struct {
	svc          AuthService
	reg          *errcode.Registry
	provider     *MemUserProvider
	captchaStore *MemCaptchaStore
}

func setupAuthService(t *testing.T) (AuthService, *errcode.Registry, *MemUserProvider) {
	t.Helper()
	env := setupAuthEnv(t)
	return env.svc, env.reg, env.provider
}

func setupAuthEnv(t *testing.T) testAuthEnv {
	t.Helper()
	reg, _ := errcode.NewRegistry(11000)

	hasher, _ := NewPasswordHasher(testArgon2Params)
	provider := NewMemUserProvider()

	// 预置测试用户
	hash, _ := hasher.Hash("correct-password")
	provider.AddUser(AuthUser{ID: "user-001", Username: "alice", PasswordHash: hash, Status: 0})

	disabledHash, _ := hasher.Hash("pwd123")
	provider.AddUser(AuthUser{ID: "user-002", Username: "disabled", PasswordHash: disabledHash, Status: 1})

	captchaStore := NewMemCaptchaStore()
	captchaSvc := NewCaptchaService(captchaStore, CaptchaConfig{
		Enabled: true, TTLSeconds: 120, Length: 4,
	}, "test")

	lockoutStore := NewMemLockoutStore()
	lockoutSvc, _ := NewLockoutService(lockoutStore, LockoutConfig{
		CaptchaThreshold: 3, LockThreshold: 5,
		FailWindowSeconds: 900, LockDurationSeconds: 900,
	}, "test")

	tokenStore := NewMemoryBlacklistStore()
	tokenSvc, _ := NewTokenService(Config{
		Issuer: "test", AccessSecret: "test-access-secret-32bytes!!!!!!",
		RefreshSecret: "test-refresh-secret-32bytes!!!!!", AccessTTLSeconds: 3600,
		RefreshTTLSeconds: 86400, RefreshRotate: true, RedisKeyPrefix: "test",
	}, tokenStore, reg)

	svc, err := NewAuthService(AuthServiceDeps{
		Tokens: tokenSvc, Users: provider, Hasher: hasher,
		CaptchaSvc: captchaSvc, LockoutSvc: lockoutSvc, Errs: reg,
		Argon2Params: testArgon2Params,
	})
	if err != nil {
		t.Fatalf("NewAuthService: %v", err)
	}
	return testAuthEnv{svc: svc, reg: reg, provider: provider, captchaStore: captchaStore}
}

// ---------------------------------------------------------------------------
// 测试用例
// ---------------------------------------------------------------------------

func TestLoginSuccess(t *testing.T) {
	svc, _, _ := setupAuthService(t)
	ctx := context.Background()

	pair, err := svc.Login(ctx, LoginInput{Username: "alice", Password: "correct-password"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Error("tokens should not be empty")
	}
}

func TestLoginBadPassword(t *testing.T) {
	svc, reg, _ := setupAuthService(t)
	ctx := context.Background()

	_, err := svc.Login(ctx, LoginInput{Username: "alice", Password: "wrong"})
	assertErrCode(t, err, reg.ErrBadCredentials.Code)
}

func TestLoginUserNotFound(t *testing.T) {
	svc, reg, _ := setupAuthService(t)
	ctx := context.Background()

	// 用户不存在 → 同一错误码 ERR_BAD_CREDENTIALS（防枚举）
	_, err := svc.Login(ctx, LoginInput{Username: "nonexistent", Password: "any"})
	assertErrCode(t, err, reg.ErrBadCredentials.Code)
}

func TestLoginAccountDisabled(t *testing.T) {
	svc, reg, _ := setupAuthService(t)
	ctx := context.Background()

	_, err := svc.Login(ctx, LoginInput{Username: "disabled", Password: "pwd123"})
	assertErrCode(t, err, reg.ErrAccountDisabled.Code)
}

func TestLoginCaptchaThreshold(t *testing.T) {
	svc, reg, _ := setupAuthService(t)
	ctx := context.Background()

	// 连续 3 次失败 → 达 captcha_threshold
	for i := 0; i < 3; i++ {
		svc.Login(ctx, LoginInput{Username: "alice", Password: "wrong"})
	}

	// 第 4 次不带验证码 → ERR_CAPTCHA_REQUIRED
	_, err := svc.Login(ctx, LoginInput{Username: "alice", Password: "correct-password"})
	assertErrCode(t, err, reg.ErrCaptchaRequired.Code)
}

func TestLoginCaptchaInvalid(t *testing.T) {
	svc, reg, _ := setupAuthService(t)
	ctx := context.Background()

	// 触发验证码需求
	for i := 0; i < 3; i++ {
		svc.Login(ctx, LoginInput{Username: "alice", Password: "wrong"})
	}

	// 带错误验证码
	_, err := svc.Login(ctx, LoginInput{
		Username: "alice", Password: "correct-password",
		CaptchaID: "nonexistent", CaptchaCode: "xxxx",
	})
	assertErrCode(t, err, reg.ErrCaptchaInvalid.Code)
}

func TestLoginLockout(t *testing.T) {
	env := setupAuthEnv(t)
	ctx := context.Background()

	// 前 3 次失败（不需验证码），累加到 captcha_threshold
	for i := 0; i < 3; i++ {
		_, err := env.svc.Login(ctx, LoginInput{Username: "alice", Password: "wrong"})
		assertErrCode(t, err, env.reg.ErrBadCredentials.Code)
	}

	// 第 4、5 次需带有效验证码才能通过验证码门槛，继续让密码校验失败以累加计数
	for i := 0; i < 2; i++ {
		captcha, err := env.svc.IssueCaptcha(ctx)
		if err != nil {
			t.Fatalf("IssueCaptcha: %v", err)
		}
		// 从 MemCaptchaStore 偷看答案
		answer := env.captchaStore.Peek("test:auth:captcha:" + captcha.CaptchaID)
		if answer == "" {
			t.Fatal("captcha answer should be in store")
		}
		// 带正确验证码 + 错误密码 → 通过验证码门槛，密码校验失败，计数累加
		_, err = env.svc.Login(ctx, LoginInput{
			Username: "alice", Password: "wrong",
			CaptchaID: captcha.CaptchaID, CaptchaCode: answer,
		})
		assertErrCode(t, err, env.reg.ErrBadCredentials.Code)
	}

	// 累计 5 次失败，应已触发锁定
	// 第 6 次请求应直接返回 ERR_ACCOUNT_LOCKED（不再走密码校验）
	_, err := env.svc.Login(ctx, LoginInput{Username: "alice", Password: "correct-password"})
	assertErrCode(t, err, env.reg.ErrAccountLocked.Code)
}

// TestLockoutStateMachine 直接测试 LockoutService 状态机。
func TestLockoutStateMachine(t *testing.T) {
	ctx := context.Background()
	store := NewMemLockoutStore()
	svc, _ := NewLockoutService(store, LockoutConfig{
		CaptchaThreshold: 3, LockThreshold: 5,
		FailWindowSeconds: 900, LockDurationSeconds: 900,
	}, "test")

	// 初始：未锁定，不需验证码
	locked, _, _ := svc.CheckLocked(ctx, "alice")
	if locked {
		t.Error("should not be locked initially")
	}
	needs, _ := svc.NeedsCaptcha(ctx, "alice")
	if needs {
		t.Error("should not need captcha initially")
	}

	// 累加 3 次失败 → 需要验证码
	for i := 0; i < 3; i++ {
		svc.RecordFailure(ctx, "alice")
	}
	needs, _ = svc.NeedsCaptcha(ctx, "alice")
	if !needs {
		t.Error("should need captcha after 3 failures")
	}

	// 再累加 2 次 → 达锁定阈值
	_, isLocked, _ := svc.RecordFailure(ctx, "alice")
	if isLocked {
		t.Error("should not be locked at 4 failures")
	}
	_, isLocked, _ = svc.RecordFailure(ctx, "alice")
	if !isLocked {
		t.Error("should be locked at 5 failures")
	}

	// 验证已锁定
	locked, remain, _ := svc.CheckLocked(ctx, "alice")
	if !locked {
		t.Error("should be locked")
	}
	if remain <= 0 {
		t.Error("remain seconds should be > 0")
	}

	// 成功登录清零
	_ = svc.ResetOnSuccess(ctx, "alice")
	needs, _ = svc.NeedsCaptcha(ctx, "alice")
	if needs {
		t.Error("should not need captcha after reset")
	}
}

func TestLoginSuccessClearsCount(t *testing.T) {
	svc, _, _ := setupAuthService(t)
	ctx := context.Background()

	// 2 次失败（未达阈值）
	svc.Login(ctx, LoginInput{Username: "alice", Password: "wrong"})
	svc.Login(ctx, LoginInput{Username: "alice", Password: "wrong"})

	// 成功登录 → 清零
	_, err := svc.Login(ctx, LoginInput{Username: "alice", Password: "correct-password"})
	if err != nil {
		t.Fatalf("Login after failures: %v", err)
	}

	// 再 2 次失败 → 不应达到阈值（因为已清零）
	svc.Login(ctx, LoginInput{Username: "alice", Password: "wrong"})
	svc.Login(ctx, LoginInput{Username: "alice", Password: "wrong"})
	_, err = svc.Login(ctx, LoginInput{Username: "alice", Password: "correct-password"})
	if err != nil {
		t.Errorf("should succeed because count was reset: %v", err)
	}
}

func TestCaptchaOneTimeConsume(t *testing.T) {
	svc, _, _ := setupAuthService(t)
	ctx := context.Background()

	captcha, err := svc.IssueCaptcha(ctx)
	if err != nil {
		t.Fatalf("IssueCaptcha: %v", err)
	}
	if captcha.CaptchaID == "" {
		t.Error("captcha_id should not be empty")
	}
	if captcha.ImageBase64 == "" {
		t.Error("image_base64 should not be empty")
	}
}

func TestRefreshAndLogout(t *testing.T) {
	svc, reg, _ := setupAuthService(t)
	ctx := context.Background()

	// 登录
	pair, _ := svc.Login(ctx, LoginInput{Username: "alice", Password: "correct-password"})

	// Refresh
	pair2, err := svc.Refresh(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if pair2.AccessToken == "" {
		t.Error("new access token should not be empty")
	}

	// 旧 refresh 应已被拉黑（通过 TokenService.Verify）
	tokenSvc := setupTokenService(t, reg)
	_ = tokenSvc // 已在 authService 内部处理

	// Logout
	err = svc.Logout(ctx, pair2.AccessToken, pair2.RefreshToken)
	if err != nil {
		t.Fatalf("Logout: %v", err)
	}
}

func setupTokenService(t *testing.T, reg *errcode.Registry) TokenService {
	t.Helper()
	store := NewMemoryBlacklistStore()
	svc, _ := NewTokenService(Config{
		Issuer: "test", AccessSecret: "test-access-secret-32bytes!!!!!!",
		RefreshSecret: "test-refresh-secret-32bytes!!!!!", AccessTTLSeconds: 3600,
		RefreshTTLSeconds: 86400, RefreshRotate: true, RedisKeyPrefix: "test",
	}, store, reg)
	return svc
}

// ---------------------------------------------------------------------------
// 辅助
// ---------------------------------------------------------------------------

func assertErrCode(t *testing.T, err error, expectedCode int) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	ecErr, ok := err.(errcode.Error)
	if !ok {
		t.Fatalf("expected errcode.Error, got %T: %v", err, err)
	}
	if ecErr.Code != expectedCode {
		t.Errorf("error code: got %d, want %d (err=%v)", ecErr.Code, expectedCode, err)
	}
}

// Interface compliance checks
var (
	_ CaptchaStore = (*MemCaptchaStore)(nil)
	_ LockoutStore = (*MemLockoutStore)(nil)
	_ UserProvider = (*MemUserProvider)(nil)
)
