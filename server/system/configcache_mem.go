// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   ConfigCache + Publisher 内存假实现 — 单测用
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 03:12:00
// +----------------------------------------------------------------------

package system

import (
	"context"
	"sync"
	"time"
)

// MemConfigCache 内存缓存假实现。
type MemConfigCache struct {
	mu    sync.Mutex
	store map[string]cacheEntry
}

type cacheEntry struct {
	val   string
	expAt time.Time
}

func NewMemConfigCache() *MemConfigCache {
	return &MemConfigCache{store: make(map[string]cacheEntry)}
}

func (m *MemConfigCache) Get(_ context.Context, key string) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.store[key]
	if !ok || time.Now().After(e.expAt) {
		delete(m.store, key)
		return "", false, nil
	}
	return e.val, true, nil
}

func (m *MemConfigCache) Set(_ context.Context, key, val string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[key] = cacheEntry{val: val, expAt: time.Now().Add(ttl)}
	return nil
}

func (m *MemConfigCache) Del(_ context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.store, k)
	}
	return nil
}

// MemPublisher 内存发布假实现。
type MemPublisher struct {
	mu       sync.Mutex
	handlers []func(ChangeEvent)
}

func NewMemPublisher() *MemPublisher {
	return &MemPublisher{}
}

func (p *MemPublisher) Publish(_ context.Context, event ChangeEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, h := range p.handlers {
		h(event)
	}
	return nil
}

func (p *MemPublisher) Subscribe(_ context.Context, handler func(ChangeEvent)) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers = append(p.handlers, handler)
	return nil
}
