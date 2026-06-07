// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   CaptchaStore Redis 实现 — 一次性消费验证码存储
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 17:12:00
// +----------------------------------------------------------------------

package auth

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCaptchaStore 使用 Redis 实现验证码存储。
type RedisCaptchaStore struct {
	client redis.Cmdable
}

// NewRedisCaptchaStore 创建 Redis CaptchaStore。
func NewRedisCaptchaStore(client redis.Cmdable) *RedisCaptchaStore {
	return &RedisCaptchaStore{client: client}
}

func (s *RedisCaptchaStore) Set(ctx context.Context, key, answer string, ttl time.Duration) error {
	return s.client.Set(ctx, key, answer, ttl).Err()
}

// GetAndDelete 原子获取并删除（一次性消费）。
func (s *RedisCaptchaStore) GetAndDelete(ctx context.Context, key string) (string, error) {
	val, err := s.client.GetDel(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return val, nil
}
