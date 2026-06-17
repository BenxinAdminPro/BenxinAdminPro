/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   展示格式化工具 — dateText（ISO→YYYY-MM-DD HH:mm:ss）+ statusText（0=正常/非0=停用）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-17 20:53:25
 * +----------------------------------------------------------------------
 *
 * 纯函数、无副作用；收敛此前分散在各 sys 页的逐字重复内联实现（T-015）。
 *  - dateText：后端 ISO 时间串（如 2026-06-08T16:16:33+08:00）→ 2026-06-08 16:16:33；
 *    非字符串（null/undefined/数字等）→ 空串（不抛、不显 Invalid Date）。
 *  - statusText：底座 sys 实体统一 status 语义 0=正常、非 0=停用（与各页 el-radio 选项一致）；
 *    登录日志的 success（1=成功/0=失败）语义不同，不归此函数。
 */

/** 后端 ISO 时间串 → YYYY-MM-DD HH:mm:ss；非字符串入参 → 空串。 */
export const dateText = (v: unknown) => (typeof v === 'string' ? v.slice(0, 19).replace('T', ' ') : '')

/** status 数值 → 展示文案：0=正常，非 0=停用。 */
export const statusText = (v: unknown) => (Number(v) === 0 ? '正常' : '停用')
