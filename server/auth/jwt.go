// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   JWT HS256 实现 — TokenService 的完整实现
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:10:00
// +----------------------------------------------------------------------

package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Config 是 JWT 令牌服务的配置，由 main/bootstrap 注入。
// auth 包不 import config 包。
type Config struct {
	Issuer            string // iss 签发者
	AccessSecret      string // access token 签名密钥
	RefreshSecret     string // refresh token 签名密钥（可与 access 不同）
	AccessTTLSeconds  int64  // access token 有效期，默认 7200(2h)
	RefreshTTLSeconds int64  // refresh token 有效期，默认 604800(7d)
	RefreshRotate     bool   // 刷新时是否轮换 refresh token，默认 true
	RedisKeyPrefix    string // Redis key 前缀
}

// Validate 校验配置合法性。
func (c Config) Validate() error {
	if c.Issuer == "" {
		return fmt.Errorf("auth: issuer must not be empty")
	}
	if c.AccessSecret == "" {
		return fmt.Errorf("auth: access_secret must not be empty")
	}
	if c.RefreshSecret == "" {
		return fmt.Errorf("auth: refresh_secret must not be empty")
	}
	return nil
}

func (c Config) accessTTL() time.Duration {
	if c.AccessTTLSeconds <= 0 {
		return 2 * time.Hour
	}
	return time.Duration(c.AccessTTLSeconds) * time.Second
}

func (c Config) refreshTTL() time.Duration {
	if c.RefreshTTLSeconds <= 0 {
		return 7 * 24 * time.Hour
	}
	return time.Duration(c.RefreshTTLSeconds) * time.Second
}

func (c Config) blacklistKey(jti string) string {
	prefix := c.RedisKeyPrefix
	if prefix == "" {
		prefix = "app"
	}
	return prefix + ":sec:jwt:bl:" + jti
}

// NowFunc 允许测试注入时间，默认 time.Now。
var NowFunc = time.Now

// ---------------------------------------------------------------------------
// jwtService 实现 TokenService
// ---------------------------------------------------------------------------

type jwtService struct {
	cfg   Config
	store BlacklistStore
	errs  *errcode.Registry
}

// NewTokenService 创建 JWT 令牌服务实例。
func NewTokenService(cfg Config, store BlacklistStore, errs *errcode.Registry) (TokenService, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if store == nil {
		return nil, fmt.Errorf("auth: BlacklistStore must not be nil")
	}
	if errs == nil {
		return nil, fmt.Errorf("auth: errcode.Registry must not be nil")
	}
	return &jwtService{cfg: cfg, store: store, errs: errs}, nil
}

// IssuePair 签发一对 access + refresh 令牌。
func (s *jwtService) IssuePair(_ context.Context, subject string, extra map[string]any) (TokenPair, error) {
	now := NowFunc()

	accessJTI, err := uuid.NewV7()
	if err != nil {
		return TokenPair{}, fmt.Errorf("auth: generate access jti: %w", err)
	}
	refreshJTI, err := uuid.NewV7()
	if err != nil {
		return TokenPair{}, fmt.Errorf("auth: generate refresh jti: %w", err)
	}

	accessExp := now.Add(s.cfg.accessTTL())
	refreshExp := now.Add(s.cfg.refreshTTL())

	accessToken, err := s.signToken(subject, TokenTypeAccess, accessJTI.String(), extra, now, accessExp, s.cfg.AccessSecret)
	if err != nil {
		return TokenPair{}, fmt.Errorf("auth: sign access token: %w", err)
	}

	refreshToken, err := s.signToken(subject, TokenTypeRefresh, refreshJTI.String(), extra, now, refreshExp, s.cfg.RefreshSecret)
	if err != nil {
		return TokenPair{}, fmt.Errorf("auth: sign refresh token: %w", err)
	}

	return TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		AccessExp:    accessExp.Unix(),
		RefreshExp:   refreshExp.Unix(),
	}, nil
}

// Parse 解析令牌（仅验签名和格式，不校验黑名单）。
// 尝试用 access_secret 和 refresh_secret 分别解析。
func (s *jwtService) Parse(token string) (*Claims, error) {
	// 先尝试 access secret
	claims, err := s.parseWithSecret(token, s.cfg.AccessSecret)
	if err == nil {
		return claims, nil
	}
	// 再尝试 refresh secret
	claims, err = s.parseWithSecret(token, s.cfg.RefreshSecret)
	if err == nil {
		return claims, nil
	}
	return nil, s.errs.ErrTokenInvalid
}

// Verify 完整校验：签名 + 过期 + nbf + iss + tt + 黑名单。
func (s *jwtService) Verify(ctx context.Context, token string, expectType string) (*Claims, error) {
	secret := s.cfg.AccessSecret
	if expectType == TokenTypeRefresh {
		secret = s.cfg.RefreshSecret
	}

	claims, err := s.parseWithSecret(token, secret)
	if err != nil {
		return nil, s.errs.ErrTokenInvalid
	}

	now := NowFunc().Unix()

	// 校验过期
	if now > claims.ExpiresAt {
		return nil, s.errs.ErrTokenExpired
	}

	// 校验 nbf
	if now < claims.NotBefore {
		return nil, s.errs.ErrTokenInvalid
	}

	// 校验 issuer
	if claims.Issuer != s.cfg.Issuer {
		return nil, s.errs.ErrTokenInvalid
	}

	// 校验 token type
	if claims.TokenType != expectType {
		return nil, s.errs.ErrTokenInvalid
	}

	// 校验黑名单
	blacklisted, err := s.store.IsBlacklisted(ctx, s.cfg.blacklistKey(claims.JTI))
	if err != nil {
		return nil, fmt.Errorf("auth: check blacklist: %w", err)
	}
	if blacklisted {
		return nil, s.errs.ErrTokenRevoked
	}

	return claims, nil
}

// Refresh 校验 refresh 令牌 → 签发新令牌对。
// 默认轮换 refresh 并拉黑旧 refresh jti。
func (s *jwtService) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	claims, err := s.Verify(ctx, refreshToken, TokenTypeRefresh)
	if err != nil {
		return TokenPair{}, err
	}

	// 拉黑旧 refresh jti
	if err := s.Revoke(ctx, claims.JTI, claims.ExpiresAt); err != nil {
		return TokenPair{}, fmt.Errorf("auth: revoke old refresh: %w", err)
	}

	// 签发新令牌对
	return s.IssuePair(ctx, claims.Subject, claims.Extra)
}

// Revoke 将 jti 加入黑名单。
func (s *jwtService) Revoke(ctx context.Context, jti string, exp int64) error {
	now := NowFunc().Unix()
	ttl := exp - now
	if ttl <= 0 {
		ttl = 1 // 已过期的令牌最少保留 1 秒
	}
	return s.store.Add(ctx, s.cfg.blacklistKey(jti), time.Duration(ttl)*time.Second)
}

// ---------------------------------------------------------------------------
// 内部方法
// ---------------------------------------------------------------------------

// jwtClaims 是 golang-jwt 的 claims 结构。
type jwtClaims struct {
	jwt.RegisteredClaims
	TokenType string         `json:"tt"`
	Extra     map[string]any `json:"extra,omitempty"`
}

func (s *jwtService) signToken(subject, tokenType, jti string, extra map[string]any, now, exp time.Time, secret string) (string, error) {
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.cfg.Issuer,
			Subject:   subject,
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
		TokenType: tokenType,
		Extra:     extra,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (s *jwtService) parseWithSecret(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("auth: unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	}, jwt.WithoutClaimsValidation()) // 我们自己做校验，不依赖库的自动校验
	if err != nil {
		return nil, err
	}

	jc, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("auth: invalid token claims")
	}

	return &Claims{
		Issuer:    jc.Issuer,
		Subject:   jc.Subject,
		TokenType: jc.TokenType,
		JTI:       jc.ID,
		IssuedAt:  jc.IssuedAt.Unix(),
		NotBefore: jc.NotBefore.Unix(),
		ExpiresAt: jc.ExpiresAt.Unix(),
		Extra:     jc.Extra,
	}, nil
}
