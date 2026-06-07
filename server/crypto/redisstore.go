// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   ReplayStore Redis 实现 — SET NX EX 原子防重放
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 15:30:00
// +----------------------------------------------------------------------

package crypto

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisReplayStore 使用 Redis SET NX EX 实现 nonce 防重放。
// key 格式由调用方（中间件）组装：{redis_key_prefix}:sec:nonce:{nonce}。
type RedisReplayStore struct {
	client redis.Cmdable
}

// NewRedisReplayStore 创建 Redis ReplayStore。
func NewRedisReplayStore(client redis.Cmdable) *RedisReplayStore {
	return &RedisReplayStore{client: client}
}

// CheckAndSet 原子地检查并标记 nonce。
// 使用 SET key "1" NX EX ttl 一次完成，无需分步。
// 返回 true 表示 nonce 已存在（重放攻击）。
func (s *RedisReplayStore) CheckAndSet(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	ok, err := s.client.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return false, err
	}
	// SetNX: true = key 不存在，已设置（首次）；false = key 已存在（重放）
	return !ok, nil
}
