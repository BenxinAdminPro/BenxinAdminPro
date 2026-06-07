// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   Argon2id 密码哈希测试 — Hash/Verify 往返 + PHC 解析 + 改参数兼容
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 17:22:00
// +----------------------------------------------------------------------

package auth

import (
	"strings"
	"testing"
)

// 测试用轻量参数（加速单测）
var testArgon2Params = Argon2idParams{
	MemoryKiB:   1024,
	Iterations:  1,
	Parallelism: 1,
	SaltLen:     16,
	KeyLen:      32,
}

func TestHashAndVerify(t *testing.T) {
	hasher, err := NewPasswordHasher(testArgon2Params)
	if err != nil {
		t.Fatalf("NewPasswordHasher: %v", err)
	}

	encoded, err := hasher.Hash("mypassword123")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	// PHC 格式检查
	if !strings.HasPrefix(encoded, "$argon2id$") {
		t.Errorf("hash should start with $argon2id$, got %q", encoded)
	}

	// 正确密码
	ok, err := hasher.Verify("mypassword123", encoded)
	if err != nil {
		t.Fatalf("Verify correct: %v", err)
	}
	if !ok {
		t.Error("Verify should return true for correct password")
	}

	// 错误密码
	ok, err = hasher.Verify("wrongpassword", encoded)
	if err != nil {
		t.Fatalf("Verify wrong: %v", err)
	}
	if ok {
		t.Error("Verify should return false for wrong password")
	}
}

func TestHashDifferentSalts(t *testing.T) {
	hasher, _ := NewPasswordHasher(testArgon2Params)
	h1, _ := hasher.Hash("same")
	h2, _ := hasher.Hash("same")
	if h1 == h2 {
		t.Error("two hashes of the same password should differ (different salts)")
	}
	// 但两个都应该验证通过
	ok1, _ := hasher.Verify("same", h1)
	ok2, _ := hasher.Verify("same", h2)
	if !ok1 || !ok2 {
		t.Error("both hashes should verify correctly")
	}
}

func TestVerifyWithDifferentParams(t *testing.T) {
	// 用一套参数哈希
	params1 := Argon2idParams{MemoryKiB: 1024, Iterations: 1, Parallelism: 1, SaltLen: 16, KeyLen: 32}
	hasher1, _ := NewPasswordHasher(params1)
	encoded, _ := hasher1.Hash("testpwd")

	// 用不同参数的 hasher 校验（应从 PHC 串解析参数，仍能校验）
	params2 := Argon2idParams{MemoryKiB: 2048, Iterations: 2, Parallelism: 1, SaltLen: 16, KeyLen: 32}
	hasher2, _ := NewPasswordHasher(params2)
	ok, err := hasher2.Verify("testpwd", encoded)
	if err != nil {
		t.Fatalf("Verify with different params: %v", err)
	}
	if !ok {
		t.Error("should verify old hash even with new hasher params")
	}
}

func TestArgon2idParamsValidation(t *testing.T) {
	tests := []Argon2idParams{
		{MemoryKiB: 0, Iterations: 1, Parallelism: 1, SaltLen: 16, KeyLen: 32},
		{MemoryKiB: 1024, Iterations: 0, Parallelism: 1, SaltLen: 16, KeyLen: 32},
		{MemoryKiB: 1024, Iterations: 1, Parallelism: 0, SaltLen: 16, KeyLen: 32},
		{MemoryKiB: 1024, Iterations: 1, Parallelism: 1, SaltLen: 0, KeyLen: 32},
		{MemoryKiB: 1024, Iterations: 1, Parallelism: 1, SaltLen: 16, KeyLen: 0},
	}
	for _, p := range tests {
		if _, err := NewPasswordHasher(p); err == nil {
			t.Errorf("expected error for params %+v", p)
		}
	}
}

func TestDummyVerifyDoesNotPanic(t *testing.T) {
	// 只检查不 panic
	DummyVerify(testArgon2Params)
}
