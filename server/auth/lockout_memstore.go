// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   LockoutStore 内存假实现 — 单测用，不依赖 Redis
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 17:11:00
// +----------------------------------------------------------------------

package auth

import (
	"context"
	"sync"
	"time"
)

// MemLockoutStore 是 LockoutStore 的内存实现，供单测使用。
type MemLockoutStore struct {
	mu    sync.Mutex
	locks map[string]time.Time // key → 锁定过期时间
	fails map[string]failEntry // key → 失败计数 + 过期时间
}

type failEntry struct {
	count int
	expAt time.Time
}

// NewMemLockoutStore 创建内存 LockoutStore。
func NewMemLockoutStore() *MemLockoutStore {
	return &MemLockoutStore{
		locks: make(map[string]time.Time),
		fails: make(map[string]failEntry),
	}
}

func (m *MemLockoutStore) IsLocked(_ context.Context, key string) (bool, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	exp, ok := m.locks[key]
	if !ok {
		return false, 0, nil
	}
	remain := time.Until(exp).Seconds()
	if remain <= 0 {
		delete(m.locks, key)
		return false, 0, nil
	}
	return true, int64(remain), nil
}

func (m *MemLockoutStore) Lock(_ context.Context, key string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.locks[key] = time.Now().Add(ttl)
	return nil
}

func (m *MemLockoutStore) GetFailCount(_ context.Context, key string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.fails[key]
	if !ok {
		return 0, nil
	}
	if time.Now().After(entry.expAt) {
		delete(m.fails, key)
		return 0, nil
	}
	return entry.count, nil
}

func (m *MemLockoutStore) IncrFail(_ context.Context, key string, ttl time.Duration) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.fails[key]
	if !ok || time.Now().After(entry.expAt) {
		entry = failEntry{count: 0, expAt: time.Now().Add(ttl)}
	}
	entry.count++
	m.fails[key] = entry
	return entry.count, nil
}

func (m *MemLockoutStore) ResetFail(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.fails, key)
	return nil
}
