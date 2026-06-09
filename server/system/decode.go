// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   system path :id 解码 — 对外 hashid → 内部 uint64 单一出入口（沿 T-003e 范式）
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-09 16:09:17
// +----------------------------------------------------------------------

package system

import (
	"strconv"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/benxin_dev/benxinadminpro-server/idcodec"
	"github.com/benxin_dev/benxinadminpro-server/response"
	"github.com/gin-gonic/gin"
)

// decodePathID 解码 path 参数 :id（对外 hashid → 内部 uint64）。
// 注入 hasher 时走 hashid 解码；hasher 为 nil（未启用 hashid、出参亦裸）时退化裸十进制，保进出对称。
// 非法/伪造/越界 → 写 400(ErrInvalidID) 并 abort，返回 err 由调用方提前返回；
// 不回落、不暴露内部 id、错误中立（沿 T-003e 防探测）。
func decodePathID(c *gin.Context, hasher *idcodec.Hasher, errs *errcode.Registry) (uint64, error) {
	raw := c.Param("id")
	if hasher != nil {
		id, err := hasher.Decode(raw)
		if err != nil {
			response.AbortErr(c, errs.ErrInvalidID.Code)
			return 0, err
		}
		return id, nil
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		response.BadReq(c)
		return 0, err
	}
	return id, nil
}
