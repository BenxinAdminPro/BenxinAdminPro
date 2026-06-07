// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   UserProvider 接口 — 用户数据抽象 + demo 内存假实现
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 17:04:00
// +----------------------------------------------------------------------

package auth

import (
	"context"
	"errors"
	"sync"
)

// ErrUserNotFound 是 UserProvider 找不到用户时返回的 sentinel error。
var ErrUserNotFound = errors.New("auth: user not found")

// AuthUser 是认证所需的最小用户结构。
// 业务中立：不含昵称/头像/手机号等业务字段。
type AuthUser struct {
	ID           string // 不透明主体标识，将作为 JWT sub
	Username     string
	PasswordHash string // Argon2id PHC 串
	Status       int    // 0=正常，非0=不可登录
}

// UserProvider 定义底座获取用户认证信息的接口。
// 真实现（DB）由 T-003 注入；本片仅提供 MemUserProvider 供自测与 demo。
type UserProvider interface {
	FindByUsername(ctx context.Context, username string) (*AuthUser, error)
}

// StatusChecker 检查用户状态是否可登录。
// 默认实现：Status != 0 返回 ErrAccountDisabled。
// 消费方可注入自定义状态语义，保持业务中立。
type StatusChecker func(user AuthUser) error

// ---------------------------------------------------------------------------
// MemUserProvider — demo 内存假实现
// ---------------------------------------------------------------------------

// MemUserProvider 是 UserProvider 的内存实现，仅供单测与 demo。
type MemUserProvider struct {
	mu    sync.RWMutex
	users map[string]*AuthUser // key = username
}

// NewMemUserProvider 创建内存 UserProvider。
func NewMemUserProvider() *MemUserProvider {
	return &MemUserProvider{users: make(map[string]*AuthUser)}
}

// AddUser 添加一个测试用户。
func (p *MemUserProvider) AddUser(user AuthUser) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.users[user.Username] = &user
}

// FindByUsername 按用户名查找。
func (p *MemUserProvider) FindByUsername(_ context.Context, username string) (*AuthUser, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	u, ok := p.users[username]
	if !ok {
		return nil, ErrUserNotFound
	}
	copied := *u
	return &copied, nil
}
