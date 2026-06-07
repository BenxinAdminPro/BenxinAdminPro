// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   Argon2id 密码哈希与校验 — PHC 标准编码串
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 17:02:00
// +----------------------------------------------------------------------

package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2idParams 是 Argon2id 哈希参数，由配置注入。
type Argon2idParams struct {
	MemoryKiB   uint32 // 内存用量（KiB），OWASP 基线 19456 (19 MiB)
	Iterations  uint32 // 迭代次数，OWASP 基线 2
	Parallelism uint8  // 并行度，OWASP 基线 1
	SaltLen     uint32 // 盐长度（字节），默认 16
	KeyLen      uint32 // 哈希输出长度（字节），默认 32
}

// Validate 校验参数合法性。
func (p Argon2idParams) Validate() error {
	if p.MemoryKiB == 0 {
		return fmt.Errorf("auth: argon2id memory_kib must be > 0")
	}
	if p.Iterations == 0 {
		return fmt.Errorf("auth: argon2id iterations must be > 0")
	}
	if p.Parallelism == 0 {
		return fmt.Errorf("auth: argon2id parallelism must be > 0")
	}
	if p.SaltLen == 0 {
		return fmt.Errorf("auth: argon2id salt_len must be > 0")
	}
	if p.KeyLen == 0 {
		return fmt.Errorf("auth: argon2id key_len must be > 0")
	}
	return nil
}

// DefaultArgon2idParams 返回 OWASP 推荐的基线参数。
func DefaultArgon2idParams() Argon2idParams {
	return Argon2idParams{
		MemoryKiB:   19456,
		Iterations:  2,
		Parallelism: 1,
		SaltLen:     16,
		KeyLen:      32,
	}
}

// PasswordHasher 定义密码哈希与校验接口。
type PasswordHasher interface {
	Hash(plain string) (encoded string, err error)
	Verify(plain, encoded string) (bool, error)
}

// argon2idHasher 实现 PasswordHasher。
type argon2idHasher struct {
	params Argon2idParams
}

// NewPasswordHasher 创建 Argon2id 密码哈希器。
func NewPasswordHasher(params Argon2idParams) (PasswordHasher, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return &argon2idHasher{params: params}, nil
}

// Hash 生成 Argon2id PHC 标准编码串。
// 格式：$argon2id$v=19$m=...,t=...,p=...$<salt_b64>$<hash_b64>
func (h *argon2idHasher) Hash(plain string) (string, error) {
	salt := make([]byte, h.params.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth: generate salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(plain),
		salt,
		h.params.Iterations,
		h.params.MemoryKiB,
		h.params.Parallelism,
		h.params.KeyLen,
	)

	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, h.params.MemoryKiB, h.params.Iterations, h.params.Parallelism,
		saltB64, hashB64), nil
}

// Verify 校验明文密码与 PHC 编码串。使用恒定时间比较。
// 参数从编码串中解析（便于日后调参不破旧哈希）。
func (h *argon2idHasher) Verify(plain, encoded string) (bool, error) {
	params, salt, hash, err := parsePHC(encoded)
	if err != nil {
		return false, err
	}

	computed := argon2.IDKey(
		[]byte(plain),
		salt,
		params.Iterations,
		params.MemoryKiB,
		params.Parallelism,
		uint32(len(hash)),
	)

	return subtle.ConstantTimeCompare(hash, computed) == 1, nil
}

// DummyVerify 执行一次 Argon2id 校验（用于防时序枚举）。
// 当用户不存在时调用，拉平响应时间。
func DummyVerify(params Argon2idParams) {
	salt := make([]byte, params.SaltLen)
	argon2.IDKey([]byte("dummy-password"), salt, params.Iterations, params.MemoryKiB, params.Parallelism, params.KeyLen)
}

// parsePHC 解析 PHC 标准编码串。
func parsePHC(encoded string) (Argon2idParams, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return Argon2idParams{}, nil, nil, fmt.Errorf("auth: invalid PHC string format")
	}
	// parts[0] = "" (leading $)
	// parts[1] = "argon2id"
	// parts[2] = "v=19"
	// parts[3] = "m=...,t=...,p=..."
	// parts[4] = salt_b64
	// parts[5] = hash_b64

	if parts[1] != "argon2id" {
		return Argon2idParams{}, nil, nil, fmt.Errorf("auth: unsupported algorithm %q", parts[1])
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return Argon2idParams{}, nil, nil, fmt.Errorf("auth: parse version: %w", err)
	}

	var p Argon2idParams
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.MemoryKiB, &p.Iterations, &p.Parallelism); err != nil {
		return Argon2idParams{}, nil, nil, fmt.Errorf("auth: parse params: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return Argon2idParams{}, nil, nil, fmt.Errorf("auth: decode salt: %w", err)
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return Argon2idParams{}, nil, nil, fmt.Errorf("auth: decode hash: %w", err)
	}

	p.SaltLen = uint32(len(salt))
	p.KeyLen = uint32(len(hash))
	return p, salt, hash, nil
}
