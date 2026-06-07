// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   Hashid 对外 ID 编解码 — 盐配置注入，统一收口
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 21:04:00
// +----------------------------------------------------------------------

package rbac

import (
	"fmt"

	"github.com/speps/go-hashids/v2"
)

// HashidConfig Hashid 配置。
type HashidConfig struct {
	Salt      string // 盐，配置注入，禁硬编码
	MinLength int    // 最小输出长度，默认 8
}

// Hasher 提供 Hashid 编解码。实例级，盐随实例走。
type Hasher struct {
	h *hashids.HashID
}

// NewHasher 创建 Hasher。
func NewHasher(cfg HashidConfig) (*Hasher, error) {
	if cfg.Salt == "" {
		return nil, fmt.Errorf("rbac: hashid salt must not be empty")
	}
	minLen := cfg.MinLength
	if minLen <= 0 {
		minLen = 8
	}
	hd := hashids.NewData()
	hd.Salt = cfg.Salt
	hd.MinLength = minLen
	h, err := hashids.NewWithData(hd)
	if err != nil {
		return nil, fmt.Errorf("rbac: create hashid: %w", err)
	}
	return &Hasher{h: h}, nil
}

// Encode 将内部 uint64 ID 编码为对外 hashid 字符串。
func (h *Hasher) Encode(id uint64) string {
	s, _ := h.h.EncodeInt64([]int64{int64(id)})
	return s
}

// Decode 将对外 hashid 字符串解码为内部 uint64 ID。
func (h *Hasher) Decode(hash string) (uint64, error) {
	ids, err := h.h.DecodeInt64WithError(hash)
	if err != nil || len(ids) == 0 {
		return 0, fmt.Errorf("rbac: invalid hashid")
	}
	return uint64(ids[0]), nil
}

// EncodeOptional 编码可选 ID（指针类型）。nil → 空字符串。
func (h *Hasher) EncodeOptional(id *uint64) string {
	if id == nil {
		return ""
	}
	return h.Encode(*id)
}
