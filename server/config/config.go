// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   应用配置加载 — viper YAML→结构体 + fail-fast 校验
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:15:00
// +----------------------------------------------------------------------

package config

import (
	"encoding/base64"
	"fmt"

	"github.com/spf13/viper"
)

// AppConfig 是应用顶层配置结构体。
// 仅负责 YAML 加载和结构映射，由 main/bootstrap 将字段映射到各包的 Config。
// crypto/auth/rbac 包不 import 本包。
type AppConfig struct {
	App     AppSection     `mapstructure:"app"`
	Errcode ErrcodeSection `mapstructure:"errcode"`
	Crypto  CryptoSection  `mapstructure:"crypto"`
	JWT     JWTSection     `mapstructure:"jwt"`
	Redis   RedisSection   `mapstructure:"redis"`
	MySQL   MySQLSection   `mapstructure:"mysql"`
}

// AppSection 应用级配置。
type AppSection struct {
	TablePrefix    string `mapstructure:"table_prefix"`
	RedisKeyPrefix string `mapstructure:"redis_key_prefix"`
}

// ErrcodeSection 错误码配置。
type ErrcodeSection struct {
	SecuritySegmentBase int `mapstructure:"security_segment_base"`
}

// CryptoSection C 端加密配置。
type CryptoSection struct {
	HeaderPrefix        string `mapstructure:"header_prefix"`
	AESKeyB64           string `mapstructure:"aes_key_b64"`
	HMACKeyB64          string `mapstructure:"hmac_key_b64"`
	ReplayWindowSeconds int64  `mapstructure:"replay_window_seconds"`
	NonceTTLSeconds     int64  `mapstructure:"nonce_ttl_seconds"`
}

// JWTSection JWT 令牌配置。
type JWTSection struct {
	Issuer            string `mapstructure:"issuer"`
	AccessSecret      string `mapstructure:"access_secret"`
	RefreshSecret     string `mapstructure:"refresh_secret"`
	AccessTTLSeconds  int64  `mapstructure:"access_ttl_seconds"`
	RefreshTTLSeconds int64  `mapstructure:"refresh_ttl_seconds"`
	RefreshRotate     bool   `mapstructure:"refresh_rotate"`
}

// RedisSection Redis 配置。
type RedisSection struct {
	Addr     string `mapstructure:"addr"`
	DB       int    `mapstructure:"db"`
	Password string `mapstructure:"password"`
}

// MySQLSection MySQL 配置。
type MySQLSection struct {
	DSN string `mapstructure:"dsn"`
}

// Load 从指定路径加载 YAML 配置文件。
// 支持环境变量覆盖（前缀 BENXIN_）。
func Load(configPath string) (*AppConfig, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetEnvPrefix("BENXIN")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("config: read config file %q: %w", configPath, err)
	}

	var cfg AppConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal config: %w", err)
	}

	return &cfg, nil
}

// Validate 对配置做 fail-fast 校验，任何不合法立即返回错误。
func (c *AppConfig) Validate() error {
	// errcode segment_base
	if c.Errcode.SecuritySegmentBase <= 0 {
		return fmt.Errorf("config: errcode.security_segment_base must be > 0, got %d", c.Errcode.SecuritySegmentBase)
	}

	// crypto.aes_key_b64 解码后必须恰好 32 字节
	if c.Crypto.AESKeyB64 == "" {
		return fmt.Errorf("config: crypto.aes_key_b64 must not be empty")
	}
	aesKey, err := base64.StdEncoding.DecodeString(c.Crypto.AESKeyB64)
	if err != nil {
		return fmt.Errorf("config: crypto.aes_key_b64 invalid base64: %w", err)
	}
	if len(aesKey) != 32 {
		return fmt.Errorf("config: crypto.aes_key_b64 must decode to exactly 32 bytes, got %d", len(aesKey))
	}

	// crypto.hmac_key_b64
	if c.Crypto.HMACKeyB64 == "" {
		return fmt.Errorf("config: crypto.hmac_key_b64 must not be empty")
	}
	if _, err := base64.StdEncoding.DecodeString(c.Crypto.HMACKeyB64); err != nil {
		return fmt.Errorf("config: crypto.hmac_key_b64 invalid base64: %w", err)
	}

	// jwt secrets
	if c.JWT.Issuer == "" {
		return fmt.Errorf("config: jwt.issuer must not be empty")
	}
	if c.JWT.AccessSecret == "" {
		return fmt.Errorf("config: jwt.access_secret must not be empty")
	}
	if c.JWT.RefreshSecret == "" {
		return fmt.Errorf("config: jwt.refresh_secret must not be empty")
	}

	// mysql dsn
	if c.MySQL.DSN == "" {
		return fmt.Errorf("config: mysql.dsn must not be empty")
	}

	// redis addr
	if c.Redis.Addr == "" {
		return fmt.Errorf("config: redis.addr must not be empty")
	}

	return nil
}

// DecodeAESKey 解码 base64 AES 密钥（调用前应先 Validate）。
func (c *AppConfig) DecodeAESKey() ([]byte, error) {
	return base64.StdEncoding.DecodeString(c.Crypto.AESKeyB64)
}

// DecodeHMACKey 解码 base64 HMAC 密钥。
func (c *AppConfig) DecodeHMACKey() ([]byte, error) {
	return base64.StdEncoding.DecodeString(c.Crypto.HMACKeyB64)
}
