<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   x-table 增强能力 harness — 只读/工具栏插槽/行操作/列筛选排序 目视验证页
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-09 14:30:00
  +----------------------------------------------------------------------
  说明：本页仅用于 T-007c 单独目视验证 x-table 新能力，使用内存 mock 数据不依赖后端。
       mock api 真实按 query 参数过滤/排序/分页，并回显最近一次 query，证明筛选/排序条件确实并入请求。
       端到端真验证在 T-007d/e/f 首个消费方页落地时回证（见任务书测试铁律）。
-->
<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { View, Delete, Upload } from '@element-plus/icons-vue'
import XTable from '@/components/x-table/XTable.vue'
import type { XTableConfig, XRow } from '@/components/x-table/types'

// ---- 内存数据集（业务中立的演示字段）----
interface DemoRow extends XRow {
  id: string
  name: string
  category: string
  status: number
  sort: number
  created_at: string
}

const CATEGORIES = ['alpha', 'beta', 'gamma']
const DATASET: DemoRow[] = Array.from({ length: 26 }, (_, i) => ({
  id: `h${(i + 1).toString(36)}xK${i}`, // 模拟 hashid 字符串
  name: `item-${String(i + 1).padStart(2, '0')}`,
  category: CATEGORIES[i % 3],
  status: i % 4 === 0 ? 1 : 0,
  sort: (i * 7) % 50,
  created_at: `2026-06-${String((i % 28) + 1).padStart(2, '0')} 10:00:00`,
}))

// 最近一次发往「后端」的 query（用于目视确认筛选/排序条件已并入）
const lastQuery = ref<Record<string, unknown>>({})

function compare(a: unknown, b: unknown): number {
  if (typeof a === 'number' && typeof b === 'number') return a - b
  return String(a).localeCompare(String(b))
}

// mock api：真实按 query 过滤/排序/分页，模拟后端为过滤/排序权威
const mockApi: XTableConfig<DemoRow>['api'] = {
  list: (params) => {
    lastQuery.value = { ...params }
    let data = [...DATASET]
    if (params.name) data = data.filter((d) => d.name.includes(String(params.name)))
    if (params.category) data = data.filter((d) => d.category === params.category)
    if (params.status !== undefined && params.status !== '' && params.status !== null) {
      data = data.filter((d) => d.status === Number(params.status))
    }
    if (params.sort && params.order) {
      const f = String(params.sort)
      const dir = params.order === 'asc' ? 1 : -1
      data.sort((a, b) => compare(a[f], b[f]) * dir)
    }
    const total = data.length
    const pg = Number(params.page) || 1
    const ps = Number(params.page_size) || 10
    const list = data.slice((pg - 1) * ps, pg * ps)
    return Promise.resolve({ list, total, page: pg, page_size: ps })
  },
}

const statusText = (v: unknown) => (Number(v) === 0 ? '正常' : '停用')

// 只读 + 列筛选/排序 + 自定义行操作（无权码=始终显示；危险态二次确认）
const readonlyConfig: XTableConfig<DemoRow> = {
  readonly: true,
  api: mockApi,
  actionsWidth: 200,
  search: [{ prop: 'name', label: '名称' }],
  columns: [
    { prop: 'name', label: '名称', minWidth: 140, filterable: true },
    { prop: 'category', label: '分类', minWidth: 120, filterable: true },
    { prop: 'sort', label: '排序值', width: 110, sortable: true },
    { prop: 'status', label: '状态', width: 100, formatter: (_r, v) => statusText(v) },
    { prop: 'created_at', label: '创建时间', minWidth: 170, sortable: true },
  ],
  actions: [
    {
      label: '详情',
      icon: View,
      handler: (row) => {
        ElMessageBox.alert(JSON.stringify(row, null, 2), '行详情', { confirmButtonText: '关闭' })
      },
    },
    {
      label: '清理',
      icon: Delete,
      type: 'danger',
      perm: 'sys:user:delete', // 超管可见；普通无此码用户会被移除（v-permission）
      confirm: '确定执行清理（演示，仅前端提示）？',
      refresh: true,
      handler: (row) => {
        ElMessage.success(`已对 ${(row as DemoRow).name} 执行清理（演示）`)
      },
    },
  ],
}
</script>

<template>
  <div class="harness">
    <el-alert
      type="info"
      :closable="false"
      title="x-table 增强 harness（T-007c）"
      description="只读模式 + 工具栏插槽 + 自定义行操作（含 v-permission / 危险确认）+ 列筛选 + 服务端排序。mock 数据，最近 query 见下方。"
      style="margin-bottom: 16px"
    />

    <el-card shadow="never">
      <XTable :config="readonlyConfig">
        <!-- 工具栏插槽：自定义按钮区（如上传/批量），不传则无空位 -->
        <template #toolbar>
          <el-button type="success" :icon="Upload" @click="ElMessage.info('工具栏插槽按钮点击（演示）')">
            自定义工具栏按钮
          </el-button>
        </template>
      </XTable>
    </el-card>

    <el-card shadow="never" style="margin-top: 16px">
      <template #header>最近一次发往后端的 query（证明筛选/排序条件已并入）</template>
      <pre class="harness__query">{{ JSON.stringify(lastQuery, null, 2) }}</pre>
    </el-card>
  </div>
</template>

<style scoped lang="scss">
.harness {
  &__query {
    margin: 0;
    font-family: var(--el-font-family-mono, monospace);
    font-size: 13px;
    color: var(--el-text-color-regular);
  }
}
</style>
