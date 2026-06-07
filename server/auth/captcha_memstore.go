// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   CaptchaStore 内存假实现 — 单测用，不依赖 Redis
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 17:10:00
// +----------------------------------------------------------------------

package auth

import (
	"context"
	"sync"
	"time"
)

// MemCaptchaStore 是 CaptchaStore 的内存实现，供单测使用。
type MemCaptchaStore struct {
	mu    sync.Mutex
	store map[string]captchaEntry
}

type captchaEntry struct {
	answer string
	expAt  time.Time
}

// NewMemCaptchaStore 创建内存 CaptchaStore。
func NewMemCaptchaStore() *MemCaptchaStore {
	return &MemCaptchaStore{store: make(map[string]captchaEntry)}
}

func (m *MemCaptchaStore) Set(_ context.Context, key, answer string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[key] = captchaEntry{answer: answer, expAt: time.Now().Add(ttl)}
	return nil
}

func (m *MemCaptchaStore) GetAndDelete(_ context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.store[key]
	if !ok {
		return "", nil
	}
	delete(m.store, key) // 一次性消费
	if time.Now().After(entry.expAt) {
		return "", nil
	}
	return entry.answer, nil
}
