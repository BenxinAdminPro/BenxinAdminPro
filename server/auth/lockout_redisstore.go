// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   LockoutStore Redis 实现 — 失败计数与锁定
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 17:13:00
// +----------------------------------------------------------------------

package auth

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisLockoutStore 使用 Redis 实现失败计数与锁定存储。
type RedisLockoutStore struct {
	client redis.Cmdable
}

// NewRedisLockoutStore 创建 Redis LockoutStore。
func NewRedisLockoutStore(client redis.Cmdable) *RedisLockoutStore {
	return &RedisLockoutStore{client: client}
}

func (s *RedisLockoutStore) IsLocked(ctx context.Context, key string) (bool, int64, error) {
	ttl, err := s.client.TTL(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}
	if ttl <= 0 {
		return false, 0, nil
	}
	return true, int64(ttl.Seconds()), nil
}

func (s *RedisLockoutStore) Lock(ctx context.Context, key string, ttl time.Duration) error {
	return s.client.Set(ctx, key, "1", ttl).Err()
}

func (s *RedisLockoutStore) GetFailCount(ctx context.Context, key string) (int, error) {
	val, err := s.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	count, err := strconv.Atoi(val)
	if err != nil {
		return 0, nil
	}
	return count, nil
}

// IncrFail 递增失败计数。只在首次（count==1）设 TTL，实现固定窗口语义：
// 窗口从第一次失败开始计时，不随后续失败续命。
func (s *RedisLockoutStore) IncrFail(ctx context.Context, key string, ttl time.Duration) (int, error) {
	count, err := s.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	// 仅首次 INCR（count==1）设置 TTL，后续 INCR 不重置
	if count == 1 {
		s.client.Expire(ctx, key, ttl)
	}
	return int(count), nil
}

func (s *RedisLockoutStore) ResetFail(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}
