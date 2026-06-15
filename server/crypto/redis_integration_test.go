// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   ReplayStore Redis 集成测试 — 需 docker compose 起 Valkey
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 15:33:00
// | @updated   2026-06-15 21:07:05  T-010a：Redis Addr 改读 BENXIN_TEST_REDIS_ADDR（testsupport 收口，默认不变）
// +----------------------------------------------------------------------
//
// 运行方式：go test -tags=integration ./crypto/... -v -count=1
// 前置：docker compose -f deploy/docker-compose.dev.yml up -d

//go:build integration

package crypto

import (
	"context"
	"testing"
	"time"

	"github.com/benxin_dev/benxinadminpro-server/internal/testsupport"
	"github.com/redis/go-redis/v9"
)

func redisClient(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr: testsupport.RedisAddr(),
		DB:   1, // 用 DB 1 避免污染默认库
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("redis ping failed (is docker compose running?): %v", err)
	}
	t.Cleanup(func() {
		// 清理测试 key
		client.FlushDB(context.Background())
		client.Close()
	})
	return client
}

func TestRedisReplayStore_SetNXSemantics(t *testing.T) {
	client := redisClient(t)
	store := NewRedisReplayStore(client)
	ctx := context.Background()
	key := "testprefix:sec:nonce:integration-nonce-001"

	// 首次：不存在，标记成功
	exists, err := store.CheckAndSet(ctx, key, 10*time.Second)
	if err != nil {
		t.Fatalf("CheckAndSet first: %v", err)
	}
	if exists {
		t.Fatal("first call should return exists=false")
	}

	// 二次：已存在，重放
	exists, err = store.CheckAndSet(ctx, key, 10*time.Second)
	if err != nil {
		t.Fatalf("CheckAndSet second: %v", err)
	}
	if !exists {
		t.Fatal("second call should return exists=true (replay)")
	}
}

func TestRedisReplayStore_TTLExpiry(t *testing.T) {
	client := redisClient(t)
	store := NewRedisReplayStore(client)
	ctx := context.Background()
	key := "testprefix:sec:nonce:integration-ttl-001"

	// 设置 1 秒 TTL
	_, err := store.CheckAndSet(ctx, key, 1*time.Second)
	if err != nil {
		t.Fatalf("CheckAndSet: %v", err)
	}

	// 等待过期
	time.Sleep(1500 * time.Millisecond)

	// 过期后应可重新设置
	exists, err := store.CheckAndSet(ctx, key, 10*time.Second)
	if err != nil {
		t.Fatalf("CheckAndSet after expiry: %v", err)
	}
	if exists {
		t.Fatal("key should have expired, exists should be false")
	}
}

func TestRedisReplayStore_KeyPrefix(t *testing.T) {
	client := redisClient(t)
	store := NewRedisReplayStore(client)
	ctx := context.Background()

	// 验证 key 格式正确写入 Redis
	key := "myapp:sec:nonce:prefix-test-001"
	_, _ = store.CheckAndSet(ctx, key, 30*time.Second)

	// 直接用 redis client 验证 key 存在
	val, err := client.Get(ctx, key).Result()
	if err != nil {
		t.Fatalf("direct GET: %v", err)
	}
	if val != "1" {
		t.Errorf("expected value '1', got %q", val)
	}

	// 验证 TTL 已设置
	ttl, err := client.TTL(ctx, key).Result()
	if err != nil {
		t.Fatalf("TTL: %v", err)
	}
	if ttl <= 0 {
		t.Errorf("expected positive TTL, got %v", ttl)
	}
}
