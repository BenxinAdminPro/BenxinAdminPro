// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   ReplayStore 内存假实现 — 单测用，不依赖 Redis
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:30:00
// +----------------------------------------------------------------------

package crypto

import (
	"context"
	"sync"
	"time"
)

// MemoryReplayStore 是 ReplayStore 的内存实现，供单测使用。
type MemoryReplayStore struct {
	mu    sync.Mutex
	store map[string]time.Time
}

// NewMemoryReplayStore 创建内存 ReplayStore。
func NewMemoryReplayStore() *MemoryReplayStore {
	return &MemoryReplayStore{store: make(map[string]time.Time)}
}

// CheckAndSet 检查并标记 nonce。
func (m *MemoryReplayStore) CheckAndSet(_ context.Context, nonce string, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 清除过期条目
	now := time.Now()
	for k, exp := range m.store {
		if now.After(exp) {
			delete(m.store, k)
		}
	}

	if _, exists := m.store[nonce]; exists {
		return true, nil
	}
	m.store[nonce] = now.Add(ttl)
	return false, nil
}
