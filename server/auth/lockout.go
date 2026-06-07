// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   登录失败计数与账号锁定 — LockoutStore 接口 + 锁定状态机
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 17:08:00
// +----------------------------------------------------------------------

package auth

import (
	"context"
	"fmt"
	"time"
)

// LockoutConfig 锁定配置。
type LockoutConfig struct {
	CaptchaThreshold    int   // 失败达此值后要求验证码，默认 3
	LockThreshold       int   // 失败达此值后锁定，默认 5
	FailWindowSeconds   int64 // 计数窗口 TTL，默认 900s
	LockDurationSeconds int64 // 锁定时长，默认 900s
}

// Validate 校验配置合法性。
func (c LockoutConfig) Validate() error {
	if c.CaptchaThreshold <= 0 {
		return fmt.Errorf("auth: captcha_threshold must be > 0")
	}
	if c.LockThreshold <= 0 {
		return fmt.Errorf("auth: lock_threshold must be > 0")
	}
	if c.CaptchaThreshold > c.LockThreshold {
		return fmt.Errorf("auth: captcha_threshold (%d) must be <= lock_threshold (%d)", c.CaptchaThreshold, c.LockThreshold)
	}
	if c.FailWindowSeconds <= 0 {
		return fmt.Errorf("auth: fail_window_seconds must be > 0")
	}
	if c.LockDurationSeconds <= 0 {
		return fmt.Errorf("auth: lock_duration_seconds must be > 0")
	}
	return nil
}

func (c LockoutConfig) failWindow() time.Duration {
	return time.Duration(c.FailWindowSeconds) * time.Second
}

func (c LockoutConfig) lockDuration() time.Duration {
	return time.Duration(c.LockDurationSeconds) * time.Second
}

// LockoutStore 抽象失败计数与锁定存储。
type LockoutStore interface {
	// IsLocked 检查用户是否被锁定。返回 true + 剩余秒数。
	IsLocked(ctx context.Context, key string) (locked bool, remainSeconds int64, err error)
	// Lock 锁定用户。
	Lock(ctx context.Context, key string, ttl time.Duration) error
	// GetFailCount 获取当前失败计数。不存在返回 0。
	GetFailCount(ctx context.Context, key string) (int, error)
	// IncrFail 递增失败计数，返回递增后的值。首次写入时设置 TTL。
	IncrFail(ctx context.Context, key string, ttl time.Duration) (count int, err error)
	// ResetFail 清零失败计数（登录成功时调用）。
	ResetFail(ctx context.Context, key string) error
}

// LockoutService 锁定服务。
type LockoutService struct {
	store          LockoutStore
	cfg            LockoutConfig
	redisKeyPrefix string
}

// NewLockoutService 创建锁定服务。
func NewLockoutService(store LockoutStore, cfg LockoutConfig, redisKeyPrefix string) (*LockoutService, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &LockoutService{store: store, cfg: cfg, redisKeyPrefix: redisKeyPrefix}, nil
}

func (s *LockoutService) lockKey(username string) string {
	prefix := s.redisKeyPrefix
	if prefix == "" {
		prefix = "app"
	}
	return prefix + ":auth:lock:" + username
}

func (s *LockoutService) failKey(username string) string {
	prefix := s.redisKeyPrefix
	if prefix == "" {
		prefix = "app"
	}
	return prefix + ":auth:fail:" + username
}

// CheckLocked 检查用户是否被锁定。
func (s *LockoutService) CheckLocked(ctx context.Context, username string) (bool, int64, error) {
	return s.store.IsLocked(ctx, s.lockKey(username))
}

// NeedsCaptcha 检查失败次数是否已达验证码阈值。
func (s *LockoutService) NeedsCaptcha(ctx context.Context, username string) (bool, error) {
	count, err := s.store.GetFailCount(ctx, s.failKey(username))
	if err != nil {
		return false, err
	}
	return count >= s.cfg.CaptchaThreshold, nil
}

// RecordFailure 记录一次登录失败，返回递增后的计数。
// 如果达到锁定阈值，自动锁定。
func (s *LockoutService) RecordFailure(ctx context.Context, username string) (count int, locked bool, err error) {
	count, err = s.store.IncrFail(ctx, s.failKey(username), s.cfg.failWindow())
	if err != nil {
		return 0, false, err
	}

	if count >= s.cfg.LockThreshold {
		if err := s.store.Lock(ctx, s.lockKey(username), s.cfg.lockDuration()); err != nil {
			return count, false, err
		}
		return count, true, nil
	}

	return count, false, nil
}

// ResetOnSuccess 登录成功后清零失败计数。
func (s *LockoutService) ResetOnSuccess(ctx context.Context, username string) error {
	return s.store.ResetFail(ctx, s.failKey(username))
}

// CaptchaThreshold 返回验证码阈值。
func (s *LockoutService) CaptchaThreshold() int {
	return s.cfg.CaptchaThreshold
}
