/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   通用树工具单测 — buildTree/flattenTree/subtreeIds（收编 T-007g 一次性 node 自测脚本为正式 vitest 断言）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-13 17:30:57
 * +----------------------------------------------------------------------
 *
 * 收编自 T-007g 附录 B 的脏数据自测（当时 esbuild→node 一次性跑、未入仓）。
 * 覆盖：正常树/孤儿降根/自引用环/互引环/三元环入链/键名参数化/环状嵌套 visited 防护/空输入/纯函数不改写入参。
 * tree.ts 是 T-007g/T-007h 已验证的中立工具，本片仅"测"不"改"——若断言暴露 bug 应停手回 PM。
 *
 * dev 告警：tree.ts devWarn 仅在 import.meta.env.DEV 为真时 console.warn；vitest 默认 mode='test' → DEV=true，
 * 故异常输入用例会触发告警，下方对孤儿/环用例断言 console.warn 被调用。
 *
 * 类型说明：buildTree<T> 返回 T[]（T 不含 children 字段），故节点类型用带「可选 children」的 type 别名表达；
 * 用 type 而非 interface 是因 interface 缺隐式索引签名、不满足 buildTree 的 `T extends Record<string, unknown>` 约束。
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { buildTree, flattenTree, subtreeIds } from '@/utils/tree'

// 默认 id/parent_id/children 字段族
type Node = {
  id: string
  parent_id?: string | null
  name?: string
  children?: Node[]
}
// 自定义键名字段族（验证参数化）
type DeptNode = {
  deptId: number
  pid?: number
  label?: string
  kids?: DeptNode[]
}

// 断言 DEV 取值符合预期（devWarn 触发的前提），不符则后续告警断言会假绿
describe('环境前置', () => {
  it('vitest 下 import.meta.env.DEV 为真（devWarn 会触发）', () => {
    expect(import.meta.env.DEV).toBe(true)
  })
})

describe('buildTree', () => {
  let warnSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
  })
  afterEach(() => {
    warnSpy.mockRestore()
  })

  it('正常树：单根 + 嵌套层级正确', () => {
    const flat: Node[] = [
      { id: 'a', parent_id: null, name: '根' },
      { id: 'b', parent_id: 'a', name: '子1' },
      { id: 'c', parent_id: 'a', name: '子2' },
      { id: 'd', parent_id: 'b', name: '孙' },
    ]
    const tree = buildTree(flat)
    expect(tree).toHaveLength(1) // 1 根
    expect(tree[0].id).toBe('a')
    expect(tree[0].children!.map((n) => n.id)).toEqual(['b', 'c'])
    expect(tree[0].children![0].children!.map((n) => n.id)).toEqual(['d']) // 嵌套正确
    expect(warnSpy).not.toHaveBeenCalled() // 正常树无告警
  })

  it('正常树：同层保持输入顺序（不重排）', () => {
    const flat: Node[] = [
      { id: 'r', parent_id: null },
      { id: 'z', parent_id: 'r' },
      { id: 'm', parent_id: 'r' },
      { id: 'a', parent_id: 'r' },
    ]
    const tree = buildTree(flat)
    expect(tree[0].children!.map((n) => n.id)).toEqual(['z', 'm', 'a'])
  })

  it('纯函数：不改写输入数组/节点（不挂 children）', () => {
    const flat: Node[] = [
      { id: 'a', parent_id: null },
      { id: 'b', parent_id: 'a' },
    ]
    const snapshot = structuredClone(flat)
    buildTree(flat)
    expect(flat).toEqual(snapshot) // 输入逐字段未变
    expect('children' in flat[0]).toBe(false) // 原节点未被挂 children
  })

  it('孤儿：parent 指向不存在节点 → 降级为根 + dev 告警，节点不丢，其子归位', () => {
    const flat: Node[] = [
      { id: 'orphan', parent_id: 'GONE' }, // 父不存在
      { id: 'child', parent_id: 'orphan' },
    ]
    const tree = buildTree(flat)
    expect(tree).toHaveLength(1) // orphan 降根
    expect(tree[0].id).toBe('orphan')
    expect(tree[0].children![0].id).toBe('child') // 子归位
    expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining('孤儿'))
  })

  it('自引用环：a.parent=a → a 降根 + 告警，子归位，不死循环', () => {
    const flat: Node[] = [
      { id: 'a', parent_id: 'a' }, // 自引用
      { id: 'c', parent_id: 'a' },
    ]
    const tree = buildTree(flat)
    expect(tree).toHaveLength(1)
    expect(tree[0].id).toBe('a')
    expect(tree[0].children![0].id).toBe('c') // 子归位
    expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining('成环'))
  })

  it('互引环：a→b→a + 环外子挂 a → 环成员全降根，子不丢，总数不丢', () => {
    const flat: Node[] = [
      { id: 'a', parent_id: 'b' },
      { id: 'b', parent_id: 'a' },
      { id: 'x', parent_id: 'a' }, // 环外子挂环成员 a
    ]
    const tree = buildTree(flat)
    // a、b 均为环成员降根；x 挂在降根后的 a 下
    const rootIds = tree.map((n) => n.id).sort()
    expect(rootIds).toEqual(['a', 'b']) // 环两成员各自成根
    const a = tree.find((n) => n.id === 'a')!
    expect(a.children!.map((n) => n.id)).toEqual(['x'])
    // 总节点数守恒（2 根 + 1 子 = 3）
    expect(flattenTree(tree)).toHaveLength(3)
    expect(warnSpy).toHaveBeenCalled() // 环成员告警
  })

  it('三元环入链：a→b→c→a + x→a → 三员全降根，入链 x 归位降根后的 a 下', () => {
    const flat: Node[] = [
      { id: 'a', parent_id: 'b' },
      { id: 'b', parent_id: 'c' },
      { id: 'c', parent_id: 'a' },
      { id: 'x', parent_id: 'a' }, // 入链节点（非环成员）
    ]
    const tree = buildTree(flat)
    const rootIds = tree.map((n) => n.id).sort()
    expect(rootIds).toEqual(['a', 'b', 'c']) // 三环员全降根
    const a = tree.find((n) => n.id === 'a')!
    expect(a.children!.map((n) => n.id)).toEqual(['x']) // x 归位
    expect(flattenTree(tree)).toHaveLength(4) // 总数守恒
  })

  it('键名参数化：自定义 idKey/parentKey/childrenKey + 数值 0 作根', () => {
    const flat: DeptNode[] = [
      { deptId: 1, pid: 0, label: '总公司' }, // pid=0 视为根（默认 isRoot 含 0）
      { deptId: 2, pid: 1, label: '技术部' },
      { deptId: 3, pid: 1, label: '市场部' },
    ]
    const tree = buildTree(flat, { idKey: 'deptId', parentKey: 'pid', childrenKey: 'kids' })
    expect(tree).toHaveLength(1)
    expect(tree[0].deptId).toBe(1)
    expect(tree[0].kids!.map((n) => n.deptId)).toEqual([2, 3])
    expect('children' in tree[0]).toBe(false) // 用的是自定义 kids 字段，无 children
  })

  it('空输入：[] → 空树', () => {
    expect(buildTree([])).toEqual([])
  })
})

describe('flattenTree', () => {
  let warnSpy: ReturnType<typeof vi.spyOn>
  beforeEach(() => {
    warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
  })
  afterEach(() => {
    warnSpy.mockRestore()
  })

  it('嵌套树 → 扁平（深度优先，剥离 children）', () => {
    const tree: Node[] = [
      {
        id: 'a',
        children: [
          { id: 'b', children: [{ id: 'd', children: [] }] },
          { id: 'c', children: [] },
        ],
      },
    ]
    const flat = flattenTree(tree)
    expect(flat.map((n) => n.id)).toEqual(['a', 'b', 'd', 'c']) // DFS 顺序
    expect('children' in flat[0]).toBe(false) // children 已剥离
  })

  it('环状嵌套输入：visited 防护，不死循环 + 去重 + 告警', () => {
    const a: Record<string, unknown> = { id: 'a' }
    const b: Record<string, unknown> = { id: 'b' }
    a.children = [b]
    b.children = [a] // 对象级互引环
    const flat = flattenTree([a])
    expect(flat.map((n) => n.id)).toEqual(['a', 'b']) // 各只出现一次，不死循环
    expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining('环状'))
  })

  it('空输入：[] → []', () => {
    expect(flattenTree([])).toEqual([])
  })
})

describe('subtreeIds', () => {
  it('收集自身 + 全部后代 id（父级选择器排除自己及子孙）', () => {
    const tree: Node[] = [
      {
        id: 'a',
        children: [
          { id: 'b', children: [{ id: 'd', children: [] }] },
          { id: 'c', children: [] },
        ],
      },
    ]
    const ids = subtreeIds(tree, 'b')
    expect([...ids].sort()).toEqual(['b', 'd']) // b 及其后代 d
    expect(ids.has('a')).toBe(false) // 不含祖先
    expect(ids.has('c')).toBe(false) // 不含旁系
  })

  it('根节点：返回整棵子树所有 id', () => {
    const tree: Node[] = [{ id: 'a', children: [{ id: 'b', children: [] }] }]
    expect([...subtreeIds(tree, 'a')].sort()).toEqual(['a', 'b'])
  })

  it('环状嵌套输入：visited 防护不死循环', () => {
    const a: Record<string, unknown> = { id: 'a' }
    const b: Record<string, unknown> = { id: 'b' }
    a.children = [b]
    b.children = [a]
    const ids = subtreeIds([a], 'a')
    expect([...ids].sort()).toEqual(['a', 'b']) // 完成且去重
  })

  it('目标 id 不存在：返回空集', () => {
    const tree: Node[] = [{ id: 'a', children: [] }]
    expect(subtreeIds(tree, 'nope').size).toBe(0)
  })
})
