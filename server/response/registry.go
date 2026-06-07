// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   统一错误码注册表 — 各模块集中登记 + 冲突检测 + fail-fast
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 00:00:00
// +----------------------------------------------------------------------

package response

import (
	"fmt"
	"sync"
)

// ErrSpec 描述一个错误码的完整规格。
type ErrSpec struct {
	Code    int    // 最终码（segmentBase + offset）
	HTTP    int    // HTTP 状态码
	I18nKey string // i18n 消息键
}

// Registry 集中管理所有模块的错误码。
// 各模块在装配时调 Register 注册自己的码段。
// response 不硬编码任何模块的具体码（业务中立）。
type Registry struct {
	mu    sync.RWMutex
	codes map[int]ErrSpec
}

// NewRegistry 创建空的错误码注册表。
func NewRegistry() *Registry {
	return &Registry{codes: make(map[int]ErrSpec)}
}

// Register 注册一个错误码。重复 code 触发 error（启动 fail-fast）。
func (r *Registry) Register(spec ErrSpec) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.codes[spec.Code]; ok {
		return fmt.Errorf("response: duplicate error code %d (existing: %s, new: %s)",
			spec.Code, existing.I18nKey, spec.I18nKey)
	}
	r.codes[spec.Code] = spec
	return nil
}

// MustRegister 注册一个错误码，冲突时 panic（适用于 init 阶段）。
func (r *Registry) MustRegister(spec ErrSpec) {
	if err := r.Register(spec); err != nil {
		panic(err)
	}
}

// Lookup 查找错误码。
func (r *Registry) Lookup(code int) (ErrSpec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	spec, ok := r.codes[code]
	return spec, ok
}

// RegisterFromErrcode 从既有 errcode.Registry 批量迁移错误码。
// 保留既有 code 数值与段语义，仅改为集中登记。
func (r *Registry) RegisterFromErrcode(errs interface{ AllSpecs() []ErrSpec }) error {
	for _, spec := range errs.AllSpecs() {
		if err := r.Register(spec); err != nil {
			return err
		}
	}
	return nil
}
