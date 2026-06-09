// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   Hashid 类型别名 — 实现已搬迁至中立包 idcodec，rbac 仅持别名引用
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 21:04:00
// | @updated   2026-06-09 16:09:17  T-004d：实现搬迁至 idcodec，本文件降为类型别名（rbac 业务代码零改动）
// +----------------------------------------------------------------------

package rbac

import "github.com/benxin_dev/benxinadminpro-server/idcodec"

// Hasher 是 idcodec.Hasher 的类型别名（编译期同一类型）。
// T-004d 将 hasher 实现搬迁至中立包 idcodec 以保 system↔rbac 解耦；
// 保留本别名使 rbac 既有业务代码（response/decode/handlers 中的 *Hasher）零改动。
// 构造与配置请直接用 idcodec.NewHasher / idcodec.HashidConfig。
type Hasher = idcodec.Hasher
