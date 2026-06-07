// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   BlacklistStore Redis 实现 — JWT jti 黑名单
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 15:31:00
// +----------------------------------------------------------------------

package auth

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisBlacklistStore 使用 Redis 实现 JWT jti 黑名单。
// key 格式由调用方（TokenService）组装：{redis_key_prefix}:sec:jwt:bl:{jti}。
type RedisBlacklistStore struct {
	client redis.Cmdable
}

// NewRedisBlacklistStore 创建 Redis BlacklistStore。
func NewRedisBlacklistStore(client redis.Cmdable) *RedisBlacklistStore {
	return &RedisBlacklistStore{client: client}
}

// Add 将 jti 加入黑名单，TTL 到期后自动移除。
func (s *RedisBlacklistStore) Add(ctx context.Context, key string, ttl time.Duration) error {
	return s.client.Set(ctx, key, "1", ttl).Err()
}

// IsBlacklisted 检查 jti 是否在黑名单中。
func (s *RedisBlacklistStore) IsBlacklisted(ctx context.Context, key string) (bool, error) {
	_, err := s.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
