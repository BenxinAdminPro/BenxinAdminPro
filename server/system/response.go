// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   system 出参 ID 编码器 — 将实体 id 统一编码为 hashid（沿 T-003b 模式）
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-09 16:09:17
// +----------------------------------------------------------------------

package system

import (
	"encoding/json"
	"reflect"

	"github.com/benxin_dev/benxinadminpro-server/idcodec"
	"github.com/gin-gonic/gin"
)

// ResponseEncoder 将 system 实体的内部自增 id 编码为对外 hashid 字符串输出。
// 设计：system 全部实体仅含单一对外 ID 字段 `ID uint64`、无跨实体 ID 引用，
// 故以「json 序列化 → 反射读 typed ID → 覆盖 id 字段为 hashid」通用处理 6 个实体，
// 避免逐实体手写 gin.H 的样板与字段漂移；从 typed uint64 取 id，无 float64 精度风险。
// 保留 model 的 json 标签语义（如 DeletedAt `json:"-"` 不输出、ConfigValue 已由 service 脱敏）。
// hasher 为 nil（未启用 hashid）时退化为原样输出裸 id，与 rbac 的 nil 退化对称。
type ResponseEncoder struct {
	h *idcodec.Hasher
}

// NewResponseEncoder 创建编码器。h 可为 nil（退化为不编码）。
func NewResponseEncoder(h *idcodec.Hasher) *ResponseEncoder {
	return &ResponseEncoder{h: h}
}

// Item 编码单个实体（指针或值），返回 id 为 hashid 的 gin.H。
func (e *ResponseEncoder) Item(v any) gin.H {
	b, _ := json.Marshal(v)
	var m gin.H
	if err := json.Unmarshal(b, &m); err != nil || m == nil {
		return gin.H{}
	}
	if e.h == nil {
		return m // 退化：保留裸 id
	}
	rv := reflect.Indirect(reflect.ValueOf(v))
	if rv.Kind() == reflect.Struct {
		if f := rv.FieldByName("ID"); f.IsValid() && f.Kind() == reflect.Uint64 {
			if _, ok := m["id"]; ok {
				m["id"] = e.h.Encode(f.Uint())
			}
		}
	}
	return m
}

// Page 编码分页列表（slice 任意实体类型）。
func (e *ResponseEncoder) Page(slice any, total int64, page, pageSize int) gin.H {
	return gin.H{"list": e.Items(slice), "total": total, "page": page, "page_size": pageSize}
}

// Items 编码实体切片为 []gin.H。
func (e *ResponseEncoder) Items(slice any) []gin.H {
	rv := reflect.ValueOf(slice)
	if rv.Kind() != reflect.Slice {
		return nil
	}
	out := make([]gin.H, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		out = append(out, e.Item(rv.Index(i).Interface()))
	}
	return out
}
