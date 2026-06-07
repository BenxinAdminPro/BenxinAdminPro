// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   AuthService 编排层 — 登录/刷新/登出/验证码业务编排
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 17:15:00
// +----------------------------------------------------------------------

package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
)

// AuthConfig 是认证编排层的聚合配置。
// auth 包不 import config 包。
type AuthConfig struct {
	Argon2id       Argon2idParams
	Captcha        CaptchaConfig
	Lockout        LockoutConfig
	RedisKeyPrefix string
}

// Validate 校验配置合法性。
func (c AuthConfig) Validate() error {
	if err := c.Argon2id.Validate(); err != nil {
		return err
	}
	if err := c.Lockout.Validate(); err != nil {
		return err
	}
	if c.Captcha.TTLSeconds < 0 {
		return errorf("captcha ttl must be >= 0")
	}
	return nil
}

// LoginInput 登录请求参数。
type LoginInput struct {
	Username    string
	Password    string
	CaptchaID   string
	CaptchaCode string
	ClientIP    string // 仅用于日志，不参与锁定键
}

// AuthService 定义认证编排接口。
type AuthService interface {
	IssueCaptcha(ctx context.Context) (Captcha, error)
	Login(ctx context.Context, in LoginInput) (TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (TokenPair, error)
	Logout(ctx context.Context, accessToken, refreshToken string) error
}

// ---------------------------------------------------------------------------
// authService 实现
// ---------------------------------------------------------------------------

type authService struct {
	tokens        TokenService
	users         UserProvider
	hasher        PasswordHasher
	captcha       *CaptchaService
	lockout       *LockoutService
	errs          *errcode.Registry
	statusChecker StatusChecker
	argon2Params  Argon2idParams
	logger        *slog.Logger
}

// AuthServiceDeps 是 AuthService 的依赖注入参数。
type AuthServiceDeps struct {
	Tokens        TokenService
	Users         UserProvider
	Hasher        PasswordHasher
	CaptchaSvc    *CaptchaService
	LockoutSvc    *LockoutService
	Errs          *errcode.Registry
	StatusChecker StatusChecker   // 可选，默认 Status!=0 → disabled
	Argon2Params  Argon2idParams  // 用于 dummy 校验
	Logger        *slog.Logger    // 可选，默认 slog.Default()
}

// NewAuthService 创建认证编排服务。
func NewAuthService(deps AuthServiceDeps) (AuthService, error) {
	if deps.Tokens == nil {
		return nil, errorf("TokenService required")
	}
	if deps.Users == nil {
		return nil, errorf("UserProvider required")
	}
	if deps.Hasher == nil {
		return nil, errorf("PasswordHasher required")
	}
	if deps.Errs == nil {
		return nil, errorf("errcode.Registry required")
	}

	sc := deps.StatusChecker
	if sc == nil {
		sc = defaultStatusChecker(deps.Errs)
	}

	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &authService{
		tokens:        deps.Tokens,
		users:         deps.Users,
		hasher:        deps.Hasher,
		captcha:       deps.CaptchaSvc,
		lockout:       deps.LockoutSvc,
		errs:          deps.Errs,
		statusChecker: sc,
		argon2Params:  deps.Argon2Params,
		logger:        logger,
	}, nil
}

func defaultStatusChecker(errs *errcode.Registry) StatusChecker {
	return func(user AuthUser) error {
		if user.Status != 0 {
			return errs.ErrAccountDisabled
		}
		return nil
	}
}

// IssueCaptcha 生成验证码。
func (s *authService) IssueCaptcha(ctx context.Context) (Captcha, error) {
	if s.captcha == nil {
		return Captcha{}, errorf("captcha not enabled")
	}
	return s.captcha.Generate(ctx)
}

// Login 执行登录编排。
// 顺序：锁定检查 → 验证码（如需）→ 查用户 → 密码校验 → 状态检查 → 发令牌。
func (s *authService) Login(ctx context.Context, in LoginInput) (TokenPair, error) {
	// 1. 查锁定
	if s.lockout != nil {
		locked, remain, err := s.lockout.CheckLocked(ctx, in.Username)
		if err != nil {
			return TokenPair{}, err
		}
		if locked {
			s.logger.Warn("login_blocked_locked",
				slog.String("username", in.Username),
				slog.String("client_ip", in.ClientIP),
				slog.Int64("remain_seconds", remain))
			return TokenPair{}, s.errs.ErrAccountLocked
		}
	}

	// 2. 查验证码需求
	if s.lockout != nil && s.captcha != nil {
		needs, err := s.lockout.NeedsCaptcha(ctx, in.Username)
		if err != nil {
			return TokenPair{}, err
		}
		if needs {
			if in.CaptchaID == "" || in.CaptchaCode == "" {
				return TokenPair{}, s.errs.ErrCaptchaRequired
			}
			ok, err := s.captcha.Verify(ctx, in.CaptchaID, in.CaptchaCode)
			if err != nil {
				return TokenPair{}, err
			}
			if !ok {
				return TokenPair{}, s.errs.ErrCaptchaInvalid
			}
		}
	}

	// 3. 查用户
	user, err := s.users.FindByUsername(ctx, in.Username)
	if errors.Is(err, ErrUserNotFound) {
		// 防时序枚举：执行 dummy Argon2id
		DummyVerify(s.argon2Params)
		s.handleFailure(ctx, in)
		return TokenPair{}, s.errs.ErrBadCredentials
	}
	if err != nil {
		return TokenPair{}, err
	}

	// 4. 密码校验
	ok, err := s.hasher.Verify(in.Password, user.PasswordHash)
	if err != nil {
		return TokenPair{}, err
	}
	if !ok {
		s.handleFailure(ctx, in)
		return TokenPair{}, s.errs.ErrBadCredentials
	}

	// 5. 状态检查
	if err := s.statusChecker(*user); err != nil {
		s.logger.Warn("login_blocked_disabled",
			slog.String("username", in.Username),
			slog.String("client_ip", in.ClientIP),
			slog.Int("status", user.Status))
		return TokenPair{}, err
	}

	// 6. 成功：清计数 + 发令牌
	if s.lockout != nil {
		_ = s.lockout.ResetOnSuccess(ctx, in.Username)
	}

	pair, err := s.tokens.IssuePair(ctx, user.ID, nil)
	if err != nil {
		return TokenPair{}, err
	}

	s.logger.Info("login_success",
		slog.String("username", in.Username),
		slog.String("client_ip", in.ClientIP),
		slog.String("user_id", user.ID))

	return pair, nil
}

// Refresh 刷新令牌。
func (s *authService) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	return s.tokens.Refresh(ctx, refreshToken)
}

// Logout 登出：拉黑 access（及可选 refresh）的 jti。
func (s *authService) Logout(ctx context.Context, accessToken, refreshToken string) error {
	// 拉黑 access token
	claims, err := s.tokens.Parse(accessToken)
	if err == nil && claims != nil {
		_ = s.tokens.Revoke(ctx, claims.JTI, claims.ExpiresAt)
	}

	// 可选：拉黑 refresh token
	if refreshToken != "" {
		rClaims, err := s.tokens.Parse(refreshToken)
		if err == nil && rClaims != nil {
			_ = s.tokens.Revoke(ctx, rClaims.JTI, rClaims.ExpiresAt)
		}
	}

	return nil
}

// handleFailure 记录失败并累计计数。
func (s *authService) handleFailure(ctx context.Context, in LoginInput) {
	if s.lockout == nil {
		return
	}
	count, locked, err := s.lockout.RecordFailure(ctx, in.Username)
	if err != nil {
		s.logger.Error("lockout_record_failure", slog.String("error", err.Error()))
		return
	}
	s.logger.Warn("login_failed",
		slog.String("username", in.Username),
		slog.String("client_ip", in.ClientIP),
		slog.Int("fail_count", count),
		slog.Bool("locked", locked))
}

func errorf(format string, args ...any) error {
	return errors.New("auth: " + fmt.Sprintf(format, args...))
}
