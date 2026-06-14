// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   system 列表查询公共工具 — LIKE 转义/排序白名单/时间范围/操作人可读化
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-14 10:30:00
// +----------------------------------------------------------------------

package system

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// errBadTimeParam 时间范围参数解析失败（handler 据此回 400，不静默吞）。
var errBadTimeParam = errors.New("system: invalid time param")

// 操作人可读化兜底文案（集中常量，禁散落硬编码；非错误码故不入 response.Registry）。
const (
	// DeletedUserName 操作人/上传人对应用户已软删（LEFT JOIN 未命中）时的展示文案。
	DeletedUserName = "已注销"
	// AnonymousName 操作人/上传人为空（无鉴权主体/系统动作）时的展示文案。
	AnonymousName = "匿名"
)

// escapeLike 转义 LIKE 通配符，杜绝用户输入中的 % _ \ 触发意外全表匹配或注入语义。
// MySQL/SQLite 默认转义字符为反斜杠。
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// likePattern 构造前后通配的模糊匹配串（输入先转义）。
func likePattern(s string) string {
	return "%" + escapeLike(s) + "%"
}

// applySort 按白名单把对外排序字段映射到真实列名，拼出安全的 ORDER BY。
// sort 未命中白名单（含空/注入尝试）→ 回退 defaultOrder，绝不把用户输入拼进 SQL。
// whitelist: 对外字段名 → 真实列名（值由代码字面量提供，非用户输入）。
func applySort(query *gorm.DB, sort, order string, whitelist map[string]string, defaultOrder string) *gorm.DB {
	col, ok := whitelist[sort]
	if !ok {
		return query.Order(defaultOrder)
	}
	dir := "DESC"
	if strings.EqualFold(order, "asc") {
		dir = "ASC"
	}
	return query.Order(col + " " + dir)
}

// userTableName 解析当前实例的 sys_user 表名（含实例前缀），不 import rbac 保持解耦。
// 沿 NamingStrategy 绑实例的铁律：前缀随 *gorm.DB 走，禁包级全局。
func userTableName(db *gorm.DB) string {
	return db.NamingStrategy.TableName("SysUser")
}

// resolveUserNames 把内部用户 ID 字符串集合解析为 username 映射（仅未软删用户）。
// 用于操作日志 operator / 文件 uploader 出参可读化（存的是 claims.Subject = 内部自增 ID 字符串）。
// 安全：只 SELECT id/username，绝不带出 password_hash 等敏感字段（T-003a 铁律）。
// 容错：用户表不存在/查询失败 → 返回空映射（纯展示增强，不阻断列表）。
func resolveUserNames(ctx context.Context, db *gorm.DB, idStrs []string) map[string]string {
	out := make(map[string]string)
	ids := make([]uint64, 0, len(idStrs))
	seen := make(map[uint64]bool)
	for _, s := range idStrs {
		id, err := strconv.ParseUint(s, 10, 64)
		if err != nil || id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return out
	}
	type userRow struct {
		ID       uint64
		Username string
	}
	var rows []userRow
	// 原生 .Table()（非 model 软删 scope），显式排除已软删用户 → 未命中者回落「已注销」。
	db.WithContext(ctx).Table(userTableName(db)).
		Select("id, username").
		Where("id IN ?", ids).
		Where("deleted_at IS NULL").
		Scan(&rows)
	for _, r := range rows {
		out[strconv.FormatUint(r.ID, 10)] = r.Username
	}
	return out
}

// displayUserName 据原始存储值（内部 ID 字符串）与解析映射给出展示名。
// 空 → 匿名；命中映射 → username；未命中（已软删/不存在）→ 已注销。
func displayUserName(raw string, names map[string]string) string {
	if raw == "" {
		return AnonymousName
	}
	if name, ok := names[raw]; ok {
		return name
	}
	return DeletedUserName
}

// userIDsByName 返回 username 模糊命中的未软删用户内部 ID 字符串集合（供按用户名过滤日志/文件）。
// 用于 operator/uploader 出参为用户名后，过滤入参也按用户名语义：先映射用户名→ID 集，再 WHERE 列 IN 集。
// 返回空集表示「无任何用户名匹配」→ 调用方应据此返回空结果（而非不过滤）。
func userIDsByName(ctx context.Context, db *gorm.DB, keyword string) []string {
	var ids []uint64
	db.WithContext(ctx).Table(userTableName(db)).
		Select("id").
		Where("username LIKE ?", likePattern(keyword)).
		Where("deleted_at IS NULL").
		Scan(&ids)
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, strconv.FormatUint(id, 10))
	}
	return out
}

// parseTimeParam 解析时间查询参数，支持 RFC3339 / "2006-01-02 15:04:05" / 纯日期。
// 空串 → (nil, nil) 表示未限定；非法 → (nil, error)。endOfDay=true 时纯日期补到当日 23:59:59。
func parseTimeParam(s string, endOfDay bool) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	if t, err := time.ParseInLocation(time.RFC3339, s, time.Local); err == nil {
		return &t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local); err == nil {
		return &t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		if endOfDay {
			t = t.Add(24*time.Hour - time.Second)
		}
		return &t, nil
	}
	return nil, errBadTimeParam
}
