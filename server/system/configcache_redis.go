// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   ConfigCache + Publisher Redis 真实现 — 缓存+Pub/Sub 热加载
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 03:42:00
// +----------------------------------------------------------------------

package system

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConfigCache 使用 Redis 实现配置缓存。
type RedisConfigCache struct {
	client redis.Cmdable
}

// NewRedisConfigCache 创建 Redis 配置缓存。
func NewRedisConfigCache(client redis.Cmdable) *RedisConfigCache {
	return &RedisConfigCache{client: client}
}

func (c *RedisConfigCache) Get(ctx context.Context, key string) (string, bool, error) {
	val, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return val, true, nil
}

func (c *RedisConfigCache) Set(ctx context.Context, key, val string, ttl time.Duration) error {
	return c.client.Set(ctx, key, val, ttl).Err()
}

func (c *RedisConfigCache) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}

// RedisPublisher 使用 Redis Pub/Sub 发布变更。
type RedisPublisher struct {
	client  redis.Cmdable
	channel string
}

// NewRedisPublisher 创建 Redis 发布器。
// channel 如 "{prefix}:config:changed"。
func NewRedisPublisher(client redis.Cmdable, channel string) *RedisPublisher {
	return &RedisPublisher{client: client, channel: channel}
}

func (p *RedisPublisher) Publish(ctx context.Context, event ChangeEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.client.Publish(ctx, p.channel, string(data)).Err()
}

// RedisSubscriber 使用 Redis Pub/Sub 订阅变更。
type RedisSubscriber struct {
	client  *redis.Client
	channel string
}

// NewRedisSubscriber 创建 Redis 订阅器。
func NewRedisSubscriber(client *redis.Client, channel string) *RedisSubscriber {
	return &RedisSubscriber{client: client, channel: channel}
}

// Subscribe 启动订阅协程。context 取消时优雅退出。
func (s *RedisSubscriber) Subscribe(ctx context.Context, handler func(ChangeEvent)) error {
	pubsub := s.client.Subscribe(ctx, s.channel)
	go func() {
		defer pubsub.Close()
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				var event ChangeEvent
				if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
					slog.Error("config_sub_unmarshal", slog.String("error", err.Error()))
					continue
				}
				handler(event)
			}
		}
	}()
	return nil
}

// 接口合规
var (
	_ ConfigCache     = (*RedisConfigCache)(nil)
	_ ConfigPublisher = (*RedisPublisher)(nil)
)
