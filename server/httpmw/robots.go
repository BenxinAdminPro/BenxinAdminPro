// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   HTTP 中间件 — X-Robots-Tag 注入（禁搜索引擎收录，参数化权威控制点）
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-15 14:30:00
// +----------------------------------------------------------------------

// Package httpmw 收纳与具体业务/领域无关的通用 HTTP 中间件（响应头注入等基础能力）。
package httpmw

import "github.com/gin-gonic/gin"

// DefaultXRobotsTag X-Robots-Tag 默认内容（默认安全：禁收录 + 禁跟随）。
// 默认值集中此处单一定义，中间件与装配方均引用，不在逻辑分支硬编码字面。
const DefaultXRobotsTag = "noindex, nofollow"

// RobotsConfig X-Robots-Tag 中间件配置（部署级静态配置，非热改）。
type RobotsConfig struct {
	// Enabled 是否注入响应头。底座中立：私有化部署若需被（企业内）搜索引擎索引可关。
	// 注意默认安全——装配方须保证「配置缺省时 Enabled=true」（见 demo loadConfig 的 SetDefault）。
	Enabled bool
	// Content 头内容；为空回退 DefaultXRobotsTag（避免「开了却注入空头」的静默失效）。
	Content string
}

// XRobotsTag 返回全局中间件，对所有响应无差别注入 X-Robots-Tag 头。
// Enabled=false → 不注入；Content 为空 → 用 DefaultXRobotsTag。
// 纯响应头注入：不读请求体、不依赖鉴权上下文、不查 DB，对 JSON/HTML 响应均无副作用。
func XRobotsTag(cfg RobotsConfig) gin.HandlerFunc {
	content := cfg.Content
	if content == "" {
		content = DefaultXRobotsTag
	}
	enabled := cfg.Enabled
	return func(c *gin.Context) {
		if enabled {
			c.Header("X-Robots-Tag", content)
		}
		c.Next()
	}
}
