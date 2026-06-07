// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   BlacklistStore Redis 集成测试 — 需 docker compose 起 Valkey
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 15:34:00
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

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/redis/go-redis/v9"
)

func redisClient(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   2, // 用 DB 2 避免污染
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

func TestRedisBlacklistStore_AddAndCheck(t *testing.T) {
	client := redisClient(t)
	store := NewRedisBlacklistStore(client)
	ctx := context.Background()
	key := "testprefix:sec:jwt:bl:jti-integration-001"

	// 初始不在黑名单
	blacklisted, err := store.IsBlacklisted(ctx, key)
	if err != nil {
		t.Fatalf("IsBlacklisted: %v", err)
	}
	if blacklisted {
		t.Fatal("should not be blacklisted initially")
	}

	// 加入黑名单
	if err := store.Add(ctx, key, 30*time.Second); err != nil {
		t.Fatalf("Add: %v", err)
	}

	// 现在应在黑名单
	blacklisted, err = store.IsBlacklisted(ctx, key)
	if err != nil {
		t.Fatalf("IsBlacklisted after add: %v", err)
	}
	if !blacklisted {
		t.Fatal("should be blacklisted after Add")
	}
}

func TestRedisBlacklistStore_TTLExpiry(t *testing.T) {
	client := redisClient(t)
	store := NewRedisBlacklistStore(client)
	ctx := context.Background()
	key := "testprefix:sec:jwt:bl:jti-ttl-001"

	// 1 秒 TTL
	if err := store.Add(ctx, key, 1*time.Second); err != nil {
		t.Fatalf("Add: %v", err)
	}

	// 等待过期
	time.Sleep(1500 * time.Millisecond)

	blacklisted, err := store.IsBlacklisted(ctx, key)
	if err != nil {
		t.Fatalf("IsBlacklisted after expiry: %v", err)
	}
	if blacklisted {
		t.Fatal("should not be blacklisted after TTL expiry")
	}
}

func TestRedisBlacklistStore_KeyFormat(t *testing.T) {
	client := redisClient(t)
	store := NewRedisBlacklistStore(client)
	ctx := context.Background()

	key := "myapp:sec:jwt:bl:format-test-001"
	_ = store.Add(ctx, key, 30*time.Second)

	// 直接验证 Redis key
	val, err := client.Get(ctx, key).Result()
	if err != nil {
		t.Fatalf("direct GET: %v", err)
	}
	if val != "1" {
		t.Errorf("expected '1', got %q", val)
	}

	ttl, err := client.TTL(ctx, key).Result()
	if err != nil {
		t.Fatalf("TTL: %v", err)
	}
	if ttl <= 0 {
		t.Errorf("expected positive TTL, got %v", ttl)
	}
}

func TestRedisBlacklistStore_FullJWTFlow(t *testing.T) {
	client := redisClient(t)
	store := NewRedisBlacklistStore(client)
	ctx := context.Background()

	reg, err := errcode.NewRegistry(11000)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}

	cfg := Config{
		Issuer:            "integration-test",
		AccessSecret:      "integration-access-secret-32b!!!!",
		RefreshSecret:     "integration-refresh-secret-32b!!!",
		AccessTTLSeconds:  3600,
		RefreshTTLSeconds: 86400,
		RefreshRotate:     true,
		RedisKeyPrefix:    "inttest",
	}

	svc, err := NewTokenService(cfg, store, reg)
	if err != nil {
		t.Fatalf("NewTokenService: %v", err)
	}

	// 签发
	pair, err := svc.IssuePair(ctx, "user-inttest", nil)
	if err != nil {
		t.Fatalf("IssuePair: %v", err)
	}

	// 验证 access
	claims, err := svc.Verify(ctx, pair.AccessToken, TokenTypeAccess)
	if err != nil {
		t.Fatalf("Verify access: %v", err)
	}

	// Revoke access
	if err := svc.Revoke(ctx, claims.JTI, claims.ExpiresAt); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	// 验证已吊销
	_, err = svc.Verify(ctx, pair.AccessToken, TokenTypeAccess)
	if err == nil {
		t.Fatal("expected error for revoked token")
	}

	// Refresh → 旧 refresh 应被拉黑
	pair2, err := svc.Refresh(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	_, err = svc.Verify(ctx, pair.RefreshToken, TokenTypeRefresh)
	if err == nil {
		t.Fatal("old refresh should be revoked after rotation")
	}

	// 新令牌有效
	_, err = svc.Verify(ctx, pair2.AccessToken, TokenTypeAccess)
	if err != nil {
		t.Fatalf("new access should be valid: %v", err)
	}
}
