//go:build integration

// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   集成测试连接参数收口 — 统一读 BENXIN_TEST_* env，缺省回退本地 dev 默认
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-15 21:07:05
// +----------------------------------------------------------------------
//
// 说明：
// - 仅供集成测试（-tags=integration）复用，单点维护 MySQL/Redis 连接默认值与 env 名。
// - 与 app 运行时配置 env（DEMO_* viper）严格分层：此处只管「测试连接到哪台 MySQL/Redis」，
//   不读取也不混用应用配置；BENXIN_TEST_* 是底座中性命名（非应用名）。
// - env 缺省时的默认值与各处原硬编码逐字一致（root/root @ localhost），保证不设 env
//   的本地 go test -tags=integration 原样跑通、零回归。

package testsupport

import "os"

const (
	// EnvMySQLDSN 集成测试 MySQL DSN 的环境变量名。
	EnvMySQLDSN = "BENXIN_TEST_MYSQL_DSN"
	// EnvRedisAddr 集成测试 Redis/Valkey 地址的环境变量名。
	EnvRedisAddr = "BENXIN_TEST_REDIS_ADDR"

	// DefaultMySQLDSN 本地 dev 默认连接串（对齐 deploy/docker-compose.dev.yml：
	// root/root @ localhost:3306 / benxinadminpro）。
	DefaultMySQLDSN = "root:root@tcp(localhost:3306)/benxinadminpro?charset=utf8mb4&parseTime=true&loc=Local"
	// DefaultRedisAddr 本地 dev 默认 Redis/Valkey 地址。
	DefaultRedisAddr = "localhost:6379"
)

// MySQLDSN 返回集成测试用 MySQL DSN：优先环境变量 BENXIN_TEST_MYSQL_DSN，缺省回退本地默认。
func MySQLDSN() string {
	if v := os.Getenv(EnvMySQLDSN); v != "" {
		return v
	}
	return DefaultMySQLDSN
}

// RedisAddr 返回集成测试用 Redis/Valkey 地址：优先环境变量 BENXIN_TEST_REDIS_ADDR，缺省回退本地默认。
func RedisAddr() string {
	if v := os.Getenv(EnvRedisAddr); v != "" {
		return v
	}
	return DefaultRedisAddr
}
