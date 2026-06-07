// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   BlacklistStore 内存假实现 — 单测用，不依赖 Redis
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:30:00
// +----------------------------------------------------------------------

package auth

import (
	"context"
	"sync"
	"time"
)

// MemoryBlacklistStore 是 BlacklistStore 的内存实现，供单测使用。
type MemoryBlacklistStore struct {
	mu    sync.Mutex
	store map[string]time.Time
}

// NewMemoryBlacklistStore 创建内存 BlacklistStore。
func NewMemoryBlacklistStore() *MemoryBlacklistStore {
	return &MemoryBlacklistStore{store: make(map[string]time.Time)}
}

// Add 将 jti 加入黑名单。
func (m *MemoryBlacklistStore) Add(_ context.Context, jti string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[jti] = time.Now().Add(ttl)
	return nil
}

// IsBlacklisted 检查 jti 是否在黑名单中。
func (m *MemoryBlacklistStore) IsBlacklisted(_ context.Context, jti string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	exp, exists := m.store[jti]
	if !exists {
		return false, nil
	}
	if time.Now().After(exp) {
		delete(m.store, jti)
		return false, nil
	}
	return true, nil
}
