// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   操作日志中间件 — 自动采集 + 异步写入 + 脱敏 + 排除列表
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 00:18:00
// +----------------------------------------------------------------------

package system

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// OperLogConfig 操作日志中间件配置。
type OperLogConfig struct {
	Enabled        bool
	ExcludePaths   []string // 排除路径前缀/精确匹配
	ExcludeMethods []string // 排除 HTTP 方法（如 GET）
	BodyMaxBytes   int      // 请求摘要最大字节，默认 1024
}

// OperLogSink 操作日志写入接口。
type OperLogSink interface {
	Write(ctx context.Context, entry SysOperLog) error
}

// sensitiveFields 脱敏黑名单。
var sensitiveFields = []string{
	"password", "password_hash", "token", "access_token", "refresh_token",
	"captcha_code", "secret", "aes_key", "hmac_key",
}

// OperLog 返回操作日志采集中间件。
// 异步写入：主流程不阻塞，写失败仅告警。
func OperLog(cfg OperLogConfig, sink OperLogSink, subjectFn func(*gin.Context) string) gin.HandlerFunc {
	excludeMethodSet := make(map[string]bool)
	for _, m := range cfg.ExcludeMethods {
		excludeMethodSet[strings.ToUpper(m)] = true
	}
	maxBytes := cfg.BodyMaxBytes
	if maxBytes <= 0 {
		maxBytes = 1024
	}

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		// 检查排除
		if excludeMethodSet[c.Request.Method] {
			c.Next()
			return
		}
		for _, p := range cfg.ExcludePaths {
			if strings.HasPrefix(c.Request.URL.Path, p) || c.Request.URL.Path == p {
				c.Next()
				return
			}
		}

		// 读取请求体摘要（有限长度）
		var reqSummary string
		if c.Request.Body != nil && c.Request.ContentLength > 0 {
			bodyBytes := make([]byte, 0, maxBytes)
			buf := make([]byte, maxBytes)
			n, _ := c.Request.Body.Read(buf)
			bodyBytes = append(bodyBytes, buf[:n]...)
			// 恢复 body 供 handler 读取
			remaining, _ := io.ReadAll(c.Request.Body)
			fullBody := append(bodyBytes, remaining...)
			c.Request.Body = io.NopCloser(bytes.NewReader(fullBody))
			reqSummary = Sanitize(string(bodyBytes))
		}

		start := time.Now()
		c.Next()
		latency := time.Since(start).Milliseconds()

		operator := ""
		if subjectFn != nil {
			operator = subjectFn(c)
		}

		permCode, _ := c.Get("required_perm_code")
		pc, _ := permCode.(string)

		resultCode := 0
		if c.Writer.Status() >= 400 {
			resultCode = c.Writer.Status()
		}

		entry := SysOperLog{
			Operator:   operator,
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			PermCode:   pc,
			IP:         c.ClientIP(),
			UserAgent:  truncate(c.Request.UserAgent(), 255),
			ReqSummary: reqSummary,
			ResultCode: resultCode,
			LatencyMs:  int(latency),
		}

		// 异步写入，写失败不影响主流程
		go func() {
			if err := sink.Write(context.Background(), entry); err != nil {
				slog.Error("operlog_write_failed", slog.String("error", err.Error()))
			}
		}()
	}
}

// Sanitize 移除请求体中的敏感字段值。
func Sanitize(body string) string {
	result := body
	for _, field := range sensitiveFields {
		patterns := []string{
			`"` + field + `":"`,
			`"` + field + `": "`,
		}
		for _, p := range patterns {
			if idx := strings.Index(result, p); idx >= 0 {
				start := idx + len(p)
				end := strings.Index(result[start:], `"`)
				if end >= 0 {
					result = result[:start] + "***" + result[start+end:]
				}
			}
		}
	}
	return result
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}

// GormOperLogSink 是 OperLogSink 的 GORM 实现。
type GormOperLogSink struct {
	DB *gorm.DB
}

func (s *GormOperLogSink) Write(_ context.Context, entry SysOperLog) error {
	return s.DB.Create(&entry).Error
}

// LogService 日志查询服务。
type LogService struct {
	db *gorm.DB
}

func NewLogService(db *gorm.DB) *LogService {
	return &LogService{db: db}
}

func (s *LogService) ListOperLogs(ctx context.Context, operator string, startTime, endTime *time.Time, page, pageSize int) ([]SysOperLog, int64, error) {
	if page <= 0 { page = 1 }
	if pageSize <= 0 || pageSize > 100 { pageSize = 20 }
	query := s.db.WithContext(ctx).Model(&SysOperLog{})
	if operator != "" {
		query = query.Where("operator = ?", operator)
	}
	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at <= ?", *endTime)
	}
	var total int64
	query.Count(&total)
	var list []SysOperLog
	query.Offset((page-1)*pageSize).Limit(pageSize).Order("id DESC").Find(&list)
	return list, total, nil
}

func (s *LogService) CleanOperLogs(ctx context.Context, before time.Time) (int64, error) {
	result := s.db.WithContext(ctx).Where("created_at < ?", before).Delete(&SysOperLog{})
	return result.RowsAffected, result.Error
}

func (s *LogService) ListLoginLogs(ctx context.Context, username string, page, pageSize int) ([]SysLoginLog, int64, error) {
	if page <= 0 { page = 1 }
	if pageSize <= 0 || pageSize > 100 { pageSize = 20 }
	query := s.db.WithContext(ctx).Model(&SysLoginLog{})
	if username != "" {
		query = query.Where("username = ?", username)
	}
	var total int64
	query.Count(&total)
	var list []SysLoginLog
	query.Offset((page-1)*pageSize).Limit(pageSize).Order("id DESC").Find(&list)
	return list, total, nil
}

func (s *LogService) CleanLoginLogs(ctx context.Context, before time.Time) (int64, error) {
	result := s.db.WithContext(ctx).Where("created_at < ?", before).Delete(&SysLoginLog{})
	return result.RowsAffected, result.Error
}
