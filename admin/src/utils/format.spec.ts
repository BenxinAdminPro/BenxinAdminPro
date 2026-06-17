/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   展示格式化工具单测 — dateText / statusText 边界（镜像 tree.spec 范式）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-17 20:53:25
 * +----------------------------------------------------------------------
 *
 * format.ts 是 T-015 收敛各页逐字重复实现得来的纯函数，本片锁边界：
 *  - dateText：典型 ISO 串裁切替换 / 非字符串（null/''）兜底空串（不显 Invalid Date）。
 *  - statusText：0=正常、非 0（1/其它）=停用。
 */
import { describe, expect, it } from 'vitest'

import { dateText, statusText } from './format'

describe('dateText', () => {
  it('典型 ISO 串 → YYYY-MM-DD HH:mm:ss（裁前 19 位、T→空格）', () => {
    expect(dateText('2026-06-08T16:16:33+08:00')).toBe('2026-06-08 16:16:33')
  })
  it('空串 → 空串（裁切后仍空）', () => {
    expect(dateText('')).toBe('')
  })
  it('null → 空串（非字符串兜底，不抛、不显 Invalid Date）', () => {
    expect(dateText(null)).toBe('')
  })
  it('undefined / 数字 → 空串（非字符串兜底）', () => {
    expect(dateText(undefined)).toBe('')
    expect(dateText(1718000000)).toBe('')
  })
})

describe('statusText', () => {
  it('0 → 正常', () => {
    expect(statusText(0)).toBe('正常')
  })
  it('1 → 停用', () => {
    expect(statusText(1)).toBe('停用')
  })
  it('非 0（其它数值）→ 停用', () => {
    expect(statusText(2)).toBe('停用')
    expect(statusText(-1)).toBe('停用')
  })
  it("字符串 '0' → 正常（Number 归一）", () => {
    expect(statusText('0')).toBe('正常')
  })
})
