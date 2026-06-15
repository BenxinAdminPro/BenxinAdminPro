// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-002 认证 Redis 集成测试 — CaptchaStore + LockoutStore 真 Valkey
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 18:00:00
// | @updated   2026-06-15 21:07:05  T-010a：Redis Addr 改读 BENXIN_TEST_REDIS_ADDR（testsupport 收口，默认不变）
// +----------------------------------------------------------------------
//
// 运行方式：go test -tags=integration ./auth/... -v -count=1
// 前置：docker compose -f deploy/docker-compose.dev.yml up -d

//go:build integration

package auth

import (
	"context"
	"testing"
	"time"

	"github.com/benxin_dev/benxinadminpro-server/internal/testsupport"
	"github.com/redis/go-redis/v9"
)

func integrationRedisClient(t *testing.T, db int) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr: testsupport.RedisAddr(),
		DB:   db,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("redis ping failed (is docker compose running?): %v", err)
	}
	t.Cleanup(func() {
		client.FlushDB(context.Background())
		client.Close()
	})
	return client
}

// ---------------------------------------------------------------------------
// CaptchaRedisStore 集成测试
// ---------------------------------------------------------------------------

func TestRedisCaptchaStore_GetDelOneTimeConsume(t *testing.T) {
	client := integrationRedisClient(t, 3)
	store := NewRedisCaptchaStore(client)
	ctx := context.Background()
	key := "inttest:auth:captcha:test-001"

	// 存储答案
	if err := store.Set(ctx, key, "ABC123", 30*time.Second); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// 第一次 GetAndDelete → 拿到答案
	answer, err := store.GetAndDelete(ctx, key)
	if err != nil {
		t.Fatalf("GetAndDelete first: %v", err)
	}
	if answer != "ABC123" {
		t.Errorf("expected ABC123, got %q", answer)
	}

	// 第二次 GetAndDelete → 已删除，返回空
	answer, err = store.GetAndDelete(ctx, key)
	if err != nil {
		t.Fatalf("GetAndDelete second: %v", err)
	}
	if answer != "" {
		t.Errorf("expected empty (consumed), got %q", answer)
	}
}

func TestRedisCaptchaStore_TTLExpiry(t *testing.T) {
	client := integrationRedisClient(t, 3)
	store := NewRedisCaptchaStore(client)
	ctx := context.Background()
	key := "inttest:auth:captcha:ttl-001"

	// 1 秒 TTL
	store.Set(ctx, key, "XYZ", 1*time.Second)

	// 等待过期
	time.Sleep(1500 * time.Millisecond)

	answer, _ := store.GetAndDelete(ctx, key)
	if answer != "" {
		t.Errorf("expected empty after expiry, got %q", answer)
	}
}

func TestRedisCaptchaStore_GetDelAtomic(t *testing.T) {
	client := integrationRedisClient(t, 3)
	store := NewRedisCaptchaStore(client)
	ctx := context.Background()
	key := "inttest:auth:captcha:atomic-001"

	store.Set(ctx, key, "ATOMICTEST", 30*time.Second)

	// GetAndDelete 后 key 应不存在
	store.GetAndDelete(ctx, key)

	// 直接用 Redis GET 验证 key 已删除
	_, err := client.Get(ctx, key).Result()
	if err != redis.Nil {
		t.Errorf("key should not exist after GetDel, err=%v", err)
	}
}

// ---------------------------------------------------------------------------
// LockoutRedisStore 集成测试
// ---------------------------------------------------------------------------

func TestRedisLockoutStore_IncrFixedWindow(t *testing.T) {
	client := integrationRedisClient(t, 4)
	store := NewRedisLockoutStore(client)
	ctx := context.Background()
	key := "inttest:auth:fail:fixed-window"

	// 第一次 INCR：count=1，应设置 TTL
	count, err := store.IncrFail(ctx, key, 30*time.Second)
	if err != nil {
		t.Fatalf("IncrFail first: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count=1, got %d", count)
	}

	// 记录首次 TTL
	ttl1, _ := client.TTL(ctx, key).Result()
	if ttl1 <= 0 {
		t.Fatalf("TTL should be set after first INCR, got %v", ttl1)
	}

	// 等待一小段时间让 TTL 消耗
	time.Sleep(2 * time.Second)

	// 第二次 INCR：count=2，TTL 不应重置（固定窗口）
	count, _ = store.IncrFail(ctx, key, 30*time.Second)
	if count != 2 {
		t.Errorf("expected count=2, got %d", count)
	}

	ttl2, _ := client.TTL(ctx, key).Result()
	// TTL2 应该小于 TTL1（消耗了 2 秒），而非被重置为 30s
	if ttl2 >= ttl1 {
		t.Errorf("TTL should NOT reset on subsequent INCR (fixed window): ttl1=%v, ttl2=%v", ttl1, ttl2)
	}
}

func TestRedisLockoutStore_LockAndCheck(t *testing.T) {
	client := integrationRedisClient(t, 4)
	store := NewRedisLockoutStore(client)
	ctx := context.Background()
	key := "inttest:auth:lock:lock-test"

	// 未锁定
	locked, _, _ := store.IsLocked(ctx, key)
	if locked {
		t.Error("should not be locked initially")
	}

	// 锁定
	store.Lock(ctx, key, 10*time.Second)

	locked, remain, _ := store.IsLocked(ctx, key)
	if !locked {
		t.Error("should be locked")
	}
	if remain <= 0 || remain > 10 {
		t.Errorf("remain should be in (0, 10], got %d", remain)
	}
}

func TestRedisLockoutStore_LockTTLExpiry(t *testing.T) {
	client := integrationRedisClient(t, 4)
	store := NewRedisLockoutStore(client)
	ctx := context.Background()
	key := "inttest:auth:lock:ttl-test"

	// 1 秒锁定
	store.Lock(ctx, key, 1*time.Second)

	time.Sleep(1500 * time.Millisecond)

	locked, _, _ := store.IsLocked(ctx, key)
	if locked {
		t.Error("should not be locked after TTL expiry")
	}
}

func TestRedisLockoutStore_GetFailCountAndReset(t *testing.T) {
	client := integrationRedisClient(t, 4)
	store := NewRedisLockoutStore(client)
	ctx := context.Background()
	key := "inttest:auth:fail:get-reset"

	// 初始 0
	count, _ := store.GetFailCount(ctx, key)
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	// INCR 3 次
	store.IncrFail(ctx, key, 30*time.Second)
	store.IncrFail(ctx, key, 30*time.Second)
	store.IncrFail(ctx, key, 30*time.Second)

	count, _ = store.GetFailCount(ctx, key)
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}

	// Reset
	store.ResetFail(ctx, key)
	count, _ = store.GetFailCount(ctx, key)
	if count != 0 {
		t.Errorf("expected 0 after reset, got %d", count)
	}
}
