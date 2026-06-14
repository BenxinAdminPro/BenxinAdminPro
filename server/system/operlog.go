// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   操作日志中间件 — 自动采集 + 异步写入 + 脱敏 + 排除列表
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 00:18:00
// | @updated   2026-06-14 10:45:00  T-005b-4：oper/login 列表补用户名模糊/路径/时间范围/排序 + operator 可读化
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

// OperLogQuery 操作日志列表查询参数。
type OperLogQuery struct {
	OperatorName string     // 按操作人用户名模糊（先映射 username→内部 ID 集再过滤 operator 列）
	Path         string     // 请求路径模糊
	StartTime    *time.Time // 时间范围起（含）
	EndTime      *time.Time // 时间范围止（含）
	Sort         string
	Order        string
	Page         int
	PageSize     int
}

func (q *OperLogQuery) normalize() {
	if q.Page <= 0 { q.Page = 1 }
	if q.PageSize <= 0 || q.PageSize > 100 { q.PageSize = 20 }
}

// operLogSortable 排序白名单。默认 id DESC（最新在前）。
var operLogSortable = map[string]string{
	"created_at":  "created_at",
	"latency_ms":  "latency_ms",
	"result_code": "result_code",
	"id":          "id",
}

func (s *LogService) ListOperLogs(ctx context.Context, q OperLogQuery) ([]SysOperLog, int64, error) {
	q.normalize()
	query := s.db.WithContext(ctx).Model(&SysOperLog{})
	if q.OperatorName != "" {
		// operator 存内部 ID 字符串；按用户名过滤须先映射用户名→ID 集。
		ids := userIDsByName(ctx, s.db, q.OperatorName)
		if len(ids) == 0 {
			query = query.Where("1 = 0") // 无任何用户名匹配 → 空结果（而非不过滤）
		} else {
			query = query.Where("operator IN ?", ids)
		}
	}
	if q.Path != "" {
		query = query.Where("path LIKE ?", likePattern(q.Path))
	}
	if q.StartTime != nil {
		query = query.Where("created_at >= ?", *q.StartTime)
	}
	if q.EndTime != nil {
		query = query.Where("created_at <= ?", *q.EndTime)
	}
	var total int64
	query.Count(&total)
	var list []SysOperLog
	query = applySort(query, q.Sort, q.Order, operLogSortable, "id DESC")
	query.Offset((q.Page-1)*q.PageSize).Limit(q.PageSize).Find(&list)
	fillOperatorNames(ctx, s.db, list)
	return list, total, nil
}

// fillOperatorNames 批量解析本页 operator 内部 ID → 用户名展示（一次查询）。
func fillOperatorNames(ctx context.Context, db *gorm.DB, list []SysOperLog) {
	if len(list) == 0 {
		return
	}
	ids := make([]string, 0, len(list))
	for i := range list {
		if list[i].Operator != "" {
			ids = append(ids, list[i].Operator)
		}
	}
	names := resolveUserNames(ctx, db, ids)
	for i := range list {
		list[i].OperatorName = displayUserName(list[i].Operator, names)
	}
}

func (s *LogService) CleanOperLogs(ctx context.Context, before time.Time) (int64, error) {
	result := s.db.WithContext(ctx).Where("created_at < ?", before).Delete(&SysOperLog{})
	return result.RowsAffected, result.Error
}

// LoginLogQuery 登录日志列表查询参数。
type LoginLogQuery struct {
	Username  string     // 用户名模糊（登录日志直接存 username，无需可读化）
	IP        string     // IP 模糊
	Success   *int8      // 登录结果精确（0 失败 / 1 成功）
	StartTime *time.Time // 时间范围起（含）
	EndTime   *time.Time // 时间范围止（含）
	Sort      string
	Order     string
	Page      int
	PageSize  int
}

func (q *LoginLogQuery) normalize() {
	if q.Page <= 0 { q.Page = 1 }
	if q.PageSize <= 0 || q.PageSize > 100 { q.PageSize = 20 }
}

var loginLogSortable = map[string]string{"created_at": "created_at", "id": "id"}

func (s *LogService) ListLoginLogs(ctx context.Context, q LoginLogQuery) ([]SysLoginLog, int64, error) {
	q.normalize()
	query := s.db.WithContext(ctx).Model(&SysLoginLog{})
	if q.Username != "" {
		query = query.Where("username LIKE ?", likePattern(q.Username))
	}
	if q.IP != "" {
		query = query.Where("ip LIKE ?", likePattern(q.IP))
	}
	if q.Success != nil {
		query = query.Where("success = ?", *q.Success)
	}
	if q.StartTime != nil {
		query = query.Where("created_at >= ?", *q.StartTime)
	}
	if q.EndTime != nil {
		query = query.Where("created_at <= ?", *q.EndTime)
	}
	var total int64
	query.Count(&total)
	var list []SysLoginLog
	query = applySort(query, q.Sort, q.Order, loginLogSortable, "id DESC")
	query.Offset((q.Page-1)*q.PageSize).Limit(q.PageSize).Find(&list)
	return list, total, nil
}

func (s *LogService) CleanLoginLogs(ctx context.Context, before time.Time) (int64, error) {
	result := s.db.WithContext(ctx).Where("created_at < ?", before).Delete(&SysLoginLog{})
	return result.RowsAffected, result.Error
}
