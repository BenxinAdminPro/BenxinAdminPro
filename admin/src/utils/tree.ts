/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   通用树工具 — buildTree 扁平→嵌套（防环/防孤儿）+ flattenTree + subtreeIds（菜单/部门树共用）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-12 10:51:19
 * +----------------------------------------------------------------------
 *
 * 纯函数、无副作用（不改输入数组/节点）；id/parent_id/children 字段名可参数化，
 * 不绑定 menu 专属字段，供菜单页与 T-007h dept 选择器等复用。
 *
 * 脏数据兜底（铁律：不死循环、不白屏）：
 *  - 孤儿（parent_id 指向不存在的节点）→ 降级为根 + dev 告警；
 *  - 成环（自引用 a→a 或互引 a→b→a 等）→ 环上节点全部降级为根（斩断环边）+ dev 告警，
 *    环外挂在环成员之下的节点不受影响（环成员成根后其子树正常归位）。
 */

export interface TreeOptions {
  /** 节点唯一键字段名，默认 'id' */
  idKey?: string
  /** 父引用字段名，默认 'parent_id' */
  parentKey?: string
  /** 子节点数组字段名，默认 'children' */
  childrenKey?: string
  /** 根判定：父引用为这些值视为根。默认 null/undefined/''/0/'0' */
  isRoot?: (parentVal: unknown) => boolean
}

interface ResolvedOptions {
  idKey: string
  parentKey: string
  childrenKey: string
  isRoot: (parentVal: unknown) => boolean
}

function resolveOptions(opts?: TreeOptions): ResolvedOptions {
  return {
    idKey: opts?.idKey ?? 'id',
    parentKey: opts?.parentKey ?? 'parent_id',
    childrenKey: opts?.childrenKey ?? 'children',
    isRoot:
      opts?.isRoot ??
      ((v: unknown) => v === null || v === undefined || v === '' || v === 0 || v === '0'),
  }
}

function devWarn(message: string): void {
  if (import.meta.env.DEV) {
    console.warn(`[tree] ${message}`)
  }
}

/**
 * buildTree 扁平数组 → 嵌套树。
 * 同层顺序保持输入顺序（调用方先排好序即可，如后端 sort ASC）；
 * 浅拷贝节点（children 为新数组），不改输入。
 */
export function buildTree<T extends Record<string, unknown>>(
  flat: T[],
  opts?: TreeOptions,
): T[] {
  const { idKey, parentKey, childrenKey, isRoot } = resolveOptions(opts)

  const nodeMap = new Map<unknown, T>()
  for (const item of flat) {
    nodeMap.set(item[idKey], { ...item, [childrenKey]: [] } as T)
  }

  // 先判环：沿 parent 链上溯，命中"当前路径中的节点"即成环 → 环成员降级为根。
  // memo 保证每个节点只判一次，整体 O(n)。
  const cyclic = new Set<unknown>()
  const safe = new Set<unknown>()
  for (const item of flat) {
    const startId = item[idKey]
    if (cyclic.has(startId) || safe.has(startId)) continue
    const path: unknown[] = []
    const onPath = new Set<unknown>()
    let cur: unknown = startId
    for (;;) {
      if (onPath.has(cur)) {
        // 命中当前路径 → cur 起到路径末尾为环成员，环之前的入链节点是安全的
        const cycleStart = path.indexOf(cur)
        for (let i = 0; i < path.length; i++) {
          if (i >= cycleStart) cyclic.add(path[i])
          else safe.add(path[i])
        }
        break
      }
      if (cyclic.has(cur)) {
        // 链入了已知环：环成员将降级为根，链上节点都能归位
        for (const id of path) safe.add(id)
        break
      }
      const node = nodeMap.get(cur)
      const parentVal = node ? node[parentKey] : undefined
      if (safe.has(cur) || !node || isRoot(parentVal) || !nodeMap.has(parentVal)) {
        // 已判安全 / 到根 / 父不存在（孤儿，buildTree 主循环降级为根）→ 整条路径安全
        for (const id of path) safe.add(id)
        if (node) safe.add(cur)
        break
      }
      path.push(cur)
      onPath.add(cur)
      cur = parentVal
    }
  }

  const roots: T[] = []
  for (const item of flat) {
    const id = item[idKey]
    const node = nodeMap.get(id) as T
    const parentVal = item[parentKey]
    if (isRoot(parentVal)) {
      roots.push(node)
      continue
    }
    if (cyclic.has(id)) {
      devWarn(`节点 ${String(id)} 的父引用成环，已降级为根`)
      roots.push(node)
      continue
    }
    const parent = nodeMap.get(parentVal)
    if (!parent) {
      devWarn(`节点 ${String(id)} 的父节点 ${String(parentVal)} 不存在（孤儿），已降级为根`)
      roots.push(node)
      continue
    }
    ;(parent[childrenKey] as T[]).push(node)
  }
  return roots
}

/**
 * flattenTree 嵌套树 → 扁平数组（深度优先，剥离 children 字段）。
 * 带 visited 防护：输入若有环状引用不死循环（dev 告警 + 跳过已访问节点）。
 */
export function flattenTree<T extends Record<string, unknown>>(
  tree: T[],
  opts?: TreeOptions,
): T[] {
  const { idKey, childrenKey } = resolveOptions(opts)
  const out: T[] = []
  const visited = new Set<unknown>()
  const walk = (nodes: T[] | undefined): void => {
    for (const n of nodes || []) {
      const id = n[idKey]
      if (visited.has(id)) {
        devWarn(`flattenTree 发现重复/环状节点 ${String(id)}，已跳过`)
        continue
      }
      visited.add(id)
      const { [childrenKey]: children, ...rest } = n
      out.push(rest as unknown as T)
      walk(children as T[] | undefined)
    }
  }
  walk(tree)
  return out
}

/**
 * subtreeIds 收集某节点自身 + 全部后代的 id 集合（如父级选择器排除"自己及子孙"防成环）。
 * 带 visited 防护，环状输入不死循环。
 */
export function subtreeIds<T extends Record<string, unknown>>(
  tree: T[],
  rootId: unknown,
  opts?: TreeOptions,
): Set<unknown> {
  const { idKey, childrenKey } = resolveOptions(opts)
  const ids = new Set<unknown>()
  const visited = new Set<unknown>()
  const walk = (nodes: T[] | undefined, inSubtree: boolean): void => {
    for (const n of nodes || []) {
      const id = n[idKey]
      if (visited.has(id)) continue
      visited.add(id)
      const hit = inSubtree || id === rootId
      if (hit) ids.add(id)
      walk(n[childrenKey] as T[] | undefined, hit)
    }
  }
  walk(tree, false)
  return ids
}
