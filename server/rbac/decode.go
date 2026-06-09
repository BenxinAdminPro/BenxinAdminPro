// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   入参对外 ID 解码 — body/query 内对外 ID 的单一解码出入口（hashid→uint64）
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-09 10:40:13
// +----------------------------------------------------------------------

package rbac

import (
	"fmt"
	"strconv"
)

// 本文件是 handler 边界对外 ID 入参的唯一解码入口（T-003e）。
// 设计：service/domain 层签名保持内部 uint64 不变（最小爆破面）；
// 所有引用对外 ID 的入参在 handler/DTO 边界经此解码为 uint64，再交给 service。
// 解码失败（非法/伪造/越界 hashid）一律由调用方映射 ErrInvalidID(400)，绝不回落、不暴露内部 id 空间。

// decodeID 单一解码出入口：对外 hashid 字符串 → 内部 uint64。
// 注入 hasher 时走 hashid 解码；hasher 为 nil（未启用 hashid、出参亦裸）时退化为裸十进制解析，保持进出对称。
func decodeID(h *Hasher, hash string) (uint64, error) {
	if h != nil {
		return h.Decode(hash)
	}
	id, err := strconv.ParseUint(hash, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("rbac: invalid id")
	}
	return id, nil
}

// decodeOptionalID 可选对外 ID：空串 → nil；非法 → error。
// 适用：用户 dept_id（空=无部门/清空）。
func decodeOptionalID(h *Hasher, hash string) (*uint64, error) {
	if hash == "" {
		return nil, nil
	}
	id, err := decodeID(h, hash)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// decodeZeroableID “根=0”语义：空串 → 0；非法 → error。
// 适用：部门/菜单创建 parent_id（空=挂根）。
func decodeZeroableID(h *Hasher, hash string) (uint64, error) {
	if hash == "" {
		return 0, nil
	}
	return decodeID(h, hash)
}

// decodeMovableID 可移动父节点三态：nil → nil(不移动)；指向空串 → 0(移到根)；其余解码。
// 适用：部门/菜单更新 parent_id（指针保留“缺省=不变”与“显式移到根”的区别）。
func decodeMovableID(h *Hasher, hash *string) (*uint64, error) {
	if hash == nil {
		return nil, nil
	}
	if *hash == "" {
		zero := uint64(0)
		return &zero, nil
	}
	id, err := decodeID(h, *hash)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// decodeIDSlice 对外 ID 列表：nil → nil(保留“不变”语义)；空非 nil → 空非 nil(保留“清空”语义)；任一非法 → error。
// 适用：用户分配角色 role_ids、角色分配菜单 menu_ids、用户岗位 post_ids。
func decodeIDSlice(h *Hasher, hashes []string) ([]uint64, error) {
	if hashes == nil {
		return nil, nil
	}
	ids := make([]uint64, 0, len(hashes))
	for _, s := range hashes {
		id, err := decodeID(h, s)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
