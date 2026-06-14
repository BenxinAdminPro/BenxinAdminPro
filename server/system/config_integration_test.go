// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-005 配置中心集成测试 — 真 Redis 缓存+Pub/Sub + 真 MySQL 加密参数
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 03:45:00
// +----------------------------------------------------------------------
//
// 运行方式：go test -tags=integration ./system/... -v -count=1 -run TestConfig
// 前置：docker compose -f deploy/docker-compose.dev.yml up -d

//go:build integration

package system

import (
	"context"
	"testing"
	"time"

	"github.com/benxin_dev/benxinadminpro-server/crypto"
	"github.com/redis/go-redis/v9"
)

func integrationRedis(t *testing.T, db int) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: db})
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("redis ping: %v", err)
	}
	t.Cleanup(func() { client.FlushDB(context.Background()); client.Close() })
	return client
}

// ---------------------------------------------------------------------------
// Redis 缓存 + ConfigCenter 端到端
// ---------------------------------------------------------------------------

func TestConfigIntegration_RedisCacheHitMiss(t *testing.T) {
	rdb := integrationRedis(t, 5)
	cache := NewRedisConfigCache(rdb)
	ctx := context.Background()

	// Set
	cache.Set(ctx, "test:config:site.name", "TestSite", 30*time.Second)

	// Hit
	val, hit, _ := cache.Get(ctx, "test:config:site.name")
	if !hit || val != "TestSite" {
		t.Errorf("expected hit=true val=TestSite, got hit=%v val=%q", hit, val)
	}

	// Del
	cache.Del(ctx, "test:config:site.name")
	_, hit, _ = cache.Get(ctx, "test:config:site.name")
	if hit {
		t.Error("should miss after Del")
	}
}

func TestConfigIntegration_RedisCacheTTL(t *testing.T) {
	rdb := integrationRedis(t, 5)
	cache := NewRedisConfigCache(rdb)
	ctx := context.Background()

	cache.Set(ctx, "test:config:ttl", "val", 1*time.Second)
	time.Sleep(1500 * time.Millisecond)

	_, hit, _ := cache.Get(ctx, "test:config:ttl")
	if hit {
		t.Error("should miss after TTL expiry")
	}
}

// ---------------------------------------------------------------------------
// Redis Pub/Sub 跨实例热加载
// ---------------------------------------------------------------------------

func TestConfigIntegration_PubSubCrossInstance(t *testing.T) {
	rdb := integrationRedis(t, 6)
	channel := "test:config:changed"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 实例 B：订阅
	sub := NewRedisSubscriber(rdb, channel)
	received := make(chan ChangeEvent, 10)
	sub.Subscribe(ctx, func(e ChangeEvent) {
		received <- e
	})

	// 等订阅就绪
	time.Sleep(200 * time.Millisecond)

	// 实例 A：发布变更
	pub := NewRedisPublisher(rdb, channel)
	pub.Publish(ctx, ChangeEvent{Type: "config", Key: "db.password"})

	// 实例 B 应收到
	select {
	case e := <-received:
		if e.Type != "config" || e.Key != "db.password" {
			t.Errorf("unexpected event: %+v", e)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for Pub/Sub event")
	}
}

func TestConfigIntegration_PubSubInvalidateCache(t *testing.T) {
	rdb := integrationRedis(t, 7)
	cache := NewRedisConfigCache(rdb)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	channel := "test:config:changed2"

	// 先缓存一个值
	cache.Set(ctx, "test:config:site.name", "OldValue", 5*time.Minute)

	// 建 ConfigCenter
	center := NewConfigCenter(nil, cache, NewRedisPublisher(rdb, channel), CenterConfig{
		CacheTTL: 5 * time.Minute, RedisKeyPrefix: "test",
	})

	// 订阅并绑定到 center.OnChange
	sub := NewRedisSubscriber(rdb, channel)
	sub.Subscribe(ctx, func(e ChangeEvent) {
		center.OnChange(e)
	})
	time.Sleep(200 * time.Millisecond)

	// 发布失效通知
	center.InvalidateConfig(ctx, "site.name")

	// 等通知生效
	time.Sleep(500 * time.Millisecond)

	// 缓存应已失效
	_, hit, _ := cache.Get(ctx, "test:config:site.name")
	if hit {
		t.Error("cache should be invalidated after Pub/Sub notification")
	}
}

// ---------------------------------------------------------------------------
// 加密参数脱敏断言
// ---------------------------------------------------------------------------

func TestConfigIntegration_EncryptedMaskedInList(t *testing.T) {
	db := setupDB(t) // 使用 system_test.go 的 setupDB
	gcmKey := testGCMKey

	encrypted, _ := crypto.EncryptGCM(gcmKey, []byte("super-secret"))
	db.Create(&SysConfig{ConfigKey: "api.secret", ConfigValue: encrypted, IsEncrypted: 1, Name: "API密钥"})
	db.Create(&SysConfig{ConfigKey: "site.name", ConfigValue: "MySite", IsEncrypted: 0, Name: "站点名"})

	svc := NewConfigService(db, reg(t))
	list, _, _ := svc.List(context.Background(), ConfigListQuery{Page: 1, PageSize: 10})

	for _, cfg := range list {
		if cfg.IsEncrypted == 1 && cfg.ConfigValue != MaskedValue {
			t.Errorf("encrypted param %q should be masked, got %q", cfg.ConfigKey, cfg.ConfigValue)
		}
		if cfg.IsEncrypted == 1 && cfg.ConfigValue == "super-secret" {
			t.Fatal("plaintext of encrypted param leaked in list!")
		}
	}

	// GetByKey 也应脱敏
	got, _ := svc.GetByKey(context.Background(), "api.secret")
	if got.ConfigValue != MaskedValue {
		t.Errorf("GetByKey should mask encrypted param, got %q", got.ConfigValue)
	}
}
