// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   配置中心 — 缓存层 + GCM 加密读写 + 热加载发布
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 03:10:00
// +----------------------------------------------------------------------

package system

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/benxin_dev/benxinadminpro-server/crypto"
	"gorm.io/gorm"
)

// ConfigCache 配置缓存接口。
type ConfigCache interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key, val string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

// ChangeEvent 变更事件（Pub/Sub 消息体）。
type ChangeEvent struct {
	Type string `json:"type"` // "config" | "dict"
	Key  string `json:"key"`  // config_key 或 dict_type
}

// ConfigPublisher 变更发布接口。
type ConfigPublisher interface {
	Publish(ctx context.Context, event ChangeEvent) error
}

// ConfigSubscriber 变更订阅接口。
type ConfigSubscriber interface {
	Subscribe(ctx context.Context, handler func(ChangeEvent)) error
}

// CenterConfig 配置中心配置。
type CenterConfig struct {
	CacheTTL       time.Duration // 缓存 TTL，默认 5 分钟
	RedisKeyPrefix string        // Redis key 前缀
	GCMKey         []byte        // GCM 主密钥（32 字节），nil 时不加密
}

// ConfigCenter 配置中心。
type ConfigCenter struct {
	db        *gorm.DB
	cache     ConfigCache
	publisher ConfigPublisher
	cfg       CenterConfig
}

// NewConfigCenter 创建配置中心。
func NewConfigCenter(db *gorm.DB, cache ConfigCache, pub ConfigPublisher, cfg CenterConfig) *ConfigCenter {
	if cfg.CacheTTL <= 0 {
		cfg.CacheTTL = 5 * time.Minute
	}
	return &ConfigCenter{db: db, cache: cache, publisher: pub, cfg: cfg}
}

func (c *ConfigCenter) cacheKey(k string) string {
	prefix := c.cfg.RedisKeyPrefix
	if prefix == "" {
		prefix = "app"
	}
	return prefix + ":config:" + k
}

func (c *ConfigCenter) dictCacheKey(dictType string) string {
	prefix := c.cfg.RedisKeyPrefix
	if prefix == "" {
		prefix = "app"
	}
	return prefix + ":dict:" + dictType
}

// GetConfig 读参数：缓存→回源 DB→回填。自动解密 is_encrypted。
func (c *ConfigCenter) GetConfig(ctx context.Context, key string) (string, error) {
	// 缓存
	if c.cache != nil {
		val, hit, _ := c.cache.Get(ctx, c.cacheKey(key))
		if hit {
			return val, nil
		}
	}

	// 回源 DB
	var cfg SysConfig
	err := c.db.WithContext(ctx).Where("config_key = ?", key).First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", fmt.Errorf("config key %q not found", key)
	}
	if err != nil {
		return "", err
	}

	val := cfg.ConfigValue
	// 自动解密
	if cfg.IsEncrypted == 1 && len(c.cfg.GCMKey) > 0 {
		plain, err := crypto.DecryptGCM(c.cfg.GCMKey, val)
		if err != nil {
			return "", fmt.Errorf("config: decrypt %q: %w", key, err)
		}
		val = string(plain)
	}

	// 回填缓存
	if c.cache != nil {
		c.cache.Set(ctx, c.cacheKey(key), val, c.cfg.CacheTTL)
	}

	return val, nil
}

// GetDict 读字典项列表（带缓存）。
func (c *ConfigCenter) GetDict(ctx context.Context, dictType string) ([]SysDictData, error) {
	// 缓存
	if c.cache != nil {
		val, hit, _ := c.cache.Get(ctx, c.dictCacheKey(dictType))
		if hit {
			var items []SysDictData
			if json.Unmarshal([]byte(val), &items) == nil {
				return items, nil
			}
		}
	}

	// 回源 DB
	var items []SysDictData
	c.db.WithContext(ctx).Where("dict_type = ?", dictType).Order("sort ASC, id ASC").Find(&items)

	// 回填缓存
	if c.cache != nil {
		data, _ := json.Marshal(items)
		c.cache.Set(ctx, c.dictCacheKey(dictType), string(data), c.cfg.CacheTTL)
	}

	return items, nil
}

// InvalidateConfig 失效参数缓存 + 发布变更。
func (c *ConfigCenter) InvalidateConfig(ctx context.Context, key string) {
	if c.cache != nil {
		c.cache.Del(ctx, c.cacheKey(key))
	}
	if c.publisher != nil {
		c.publisher.Publish(ctx, ChangeEvent{Type: "config", Key: key})
	}
}

// InvalidateDict 失效字典缓存 + 发布变更。
func (c *ConfigCenter) InvalidateDict(ctx context.Context, dictType string) {
	if c.cache != nil {
		c.cache.Del(ctx, c.dictCacheKey(dictType))
	}
	if c.publisher != nil {
		c.publisher.Publish(ctx, ChangeEvent{Type: "dict", Key: dictType})
	}
}

// OnChange 订阅回调：收到变更通知时失效对应缓存。
func (c *ConfigCenter) OnChange(event ChangeEvent) {
	ctx := context.Background()
	switch event.Type {
	case "config":
		if c.cache != nil {
			c.cache.Del(ctx, c.cacheKey(event.Key))
		}
	case "dict":
		if c.cache != nil {
			c.cache.Del(ctx, c.dictCacheKey(event.Key))
		}
	}
}

// EncryptAndStore 加密存储参数值。
func (c *ConfigCenter) EncryptValue(plaintext string) (string, error) {
	if len(c.cfg.GCMKey) == 0 {
		return "", fmt.Errorf("config: GCM key not configured")
	}
	return crypto.EncryptGCM(c.cfg.GCMKey, []byte(plaintext))
}
