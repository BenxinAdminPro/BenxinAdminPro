<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   操作日志 — sys_oper_log 只读列表 + 详情弹窗 + 清理（x-table 只读模式首个真消费页）
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-10 12:02:21
  +----------------------------------------------------------------------
  说明：x-table readonly 模式（api 只给 list，无增改删）；行操作「详情」用列表行内数据弹窗
       展示长字段（后端无详情端点，req_summary/user_agent 等已随列表返回，不发额外请求）。
       req_summary 敏感字段由后端脱敏为 ***（T-004a Sanitize），前端原样展示、不还原、不打印。
       清理：后端 DELETE /sys/logs/oper 无入参、固定删 3 个月前（handler 硬编码），破坏性操作
       走工具栏按钮 + 二次确认明示范围，挂独立权限码 sys:operlog:clean。
       列筛选：operator 真生效（后端精确匹配）；path 降级透传（后端暂无该过滤参数，已上报）。
       排序：created_at 开关挂上、sort/order 透传，后端暂固定 id DESC（降级，已上报）。
-->
<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete } from '@element-plus/icons-vue'
import XTable from '@/components/x-table/XTable.vue'
import type { XTableConfig, XRow } from '@/components/x-table/types'
import { listOperLogs, cleanOperLogs, type OperLogRow } from '@/api/operlog'

const tableRef = ref<InstanceType<typeof XTable>>()

const dateText = (v: unknown) => (typeof v === 'string' ? v.slice(0, 19).replace('T', ' ') : '')
const resultText = (v: unknown) => (Number(v) === 0 ? '成功' : String(v))

const config: XTableConfig = {
  readonly: true,
  api: { list: listOperLogs },
  columns: [
    // operator 过滤后端真生效（operator= 精确匹配）。注意：后端采集存的是 JWT subject
    // （内部用户 ID 字符串，如 "1"）而非用户名——列显示与过滤值均为该 ID（缺口已上报，
    // 后端改存用户名后此处零改动生效）；path 为降级透传（后端暂无此参数，已上报）
    { prop: 'operator', label: '操作人', width: 110, filterable: true },
    { prop: 'method', label: '方法', width: 80 },
    { prop: 'path', label: '路径', minWidth: 170, filterable: true },
    { prop: 'perm_code', label: '权限码', minWidth: 150 },
    { prop: 'ip', label: 'IP', width: 130 },
    { prop: 'result_code', label: '结果', width: 80, formatter: (_r, v) => resultText(v) },
    { prop: 'latency_ms', label: '耗时(ms)', width: 95 },
    // sortable 降级：后端列表暂固定 id DESC、不消费 sort/order（已上报），补参后零改动生效
    { prop: 'created_at', label: '操作时间', minWidth: 165, sortable: true, formatter: (_r, v) => dateText(v) },
  ],
  // 详情用行内数据弹窗（后端无详情端点）；列表权限即门槛，不挂额外权限码
  actions: [{ label: '详情', handler: (row: XRow) => openDetail(row) }],
  actionsWidth: 90,
}

// ---- 详情弹窗（行内数据直显，无额外请求；脱敏字段原样展示）----
const detailVisible = ref(false)
const detail = ref<OperLogRow | null>(null)

function openDetail(row: XRow): void {
  detail.value = row as OperLogRow
  detailVisible.value = true
}

// ---- 清理（破坏性：后端固定删 3 个月前全部操作日志）----
async function onClean(): Promise<void> {
  try {
    await ElMessageBox.confirm(
      '将删除 3 个月之前的全部操作日志（后端固定保留近 3 个月，不可恢复），是否继续？',
      '清理操作日志',
      { confirmButtonText: '确定清理', cancelButtonText: '取消', type: 'warning' },
    )
  } catch {
    return // 用户取消
  }
  let deleted = 0
  try {
    const res = await cleanOperLogs()
    deleted = res.deleted ?? 0
  } catch {
    return // 请求层已 toast
  }
  ElMessage.success(`清理完成，删除 ${deleted} 条`)
  tableRef.value?.refresh()
}
</script>

<template>
  <el-card shadow="never">
    <XTable ref="tableRef" :config="config">
      <!-- 工具栏：清理按钮挂独立清理权限码（非 list 码） -->
      <template #toolbar>
        <el-button v-permission="'sys:operlog:clean'" type="danger" plain :icon="Delete" @click="onClean">
          清理 3 个月前日志
        </el-button>
        <el-tag type="info">日志由中间件自动采集（GET 请求不记录）</el-tag>
      </template>
    </XTable>

    <!-- 详情弹窗：长字段（请求摘要/UA）完整展示 -->
    <el-dialog v-model="detailVisible" title="操作日志详情" width="640px" destroy-on-close>
      <el-descriptions v-if="detail" :column="2" border>
        <el-descriptions-item label="操作人">{{ detail.operator }}</el-descriptions-item>
        <el-descriptions-item label="操作时间">{{ dateText(detail.created_at) }}</el-descriptions-item>
        <el-descriptions-item label="方法">{{ detail.method }}</el-descriptions-item>
        <el-descriptions-item label="路径">{{ detail.path }}</el-descriptions-item>
        <el-descriptions-item label="权限码">{{ detail.perm_code }}</el-descriptions-item>
        <el-descriptions-item label="IP">{{ detail.ip }}</el-descriptions-item>
        <el-descriptions-item label="结果">{{ resultText(detail.result_code) }}</el-descriptions-item>
        <el-descriptions-item label="耗时">{{ detail.latency_ms }} ms</el-descriptions-item>
        <el-descriptions-item label="User-Agent" :span="2">{{ detail.user_agent }}</el-descriptions-item>
        <el-descriptions-item label="请求摘要" :span="2">
          <pre class="operlog-detail__pre">{{ detail.req_summary || '（空）' }}</pre>
        </el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<style scoped lang="scss">
.operlog-detail__pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: inherit;
}
</style>
