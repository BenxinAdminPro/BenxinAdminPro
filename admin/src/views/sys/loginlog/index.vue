<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   登录日志 — sys_login_log 只读列表 + 清理（x-table 只读模式消费页）
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-10 12:02:21
  +----------------------------------------------------------------------
  说明：x-table readonly 模式（api 只给 list，无增改删、无行操作）；字段全部短文本直接进列，
       无详情弹窗（user_agent 超宽走 x-table 内置 show-overflow-tooltip）。
       清理：后端 DELETE /sys/logs/login 无入参、固定删 3 个月前（handler 硬编码），破坏性操作
       走工具栏按钮 + 二次确认明示范围，挂独立权限码 sys:loginlog:clean。
       列筛选：username 真生效（后端精确匹配）。排序：created_at 开关挂上、sort/order 透传，
       后端暂固定 id DESC（降级，已上报）。
-->
<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete } from '@element-plus/icons-vue'
import XTable from '@/components/x-table/XTable.vue'
import type { XTableConfig } from '@/components/x-table/types'
import { listLoginLogs, cleanLoginLogs } from '@/api/loginlog'

const tableRef = ref<InstanceType<typeof XTable>>()

const dateText = (v: unknown) => (typeof v === 'string' ? v.slice(0, 19).replace('T', ' ') : '')

const config: XTableConfig = {
  readonly: true,
  api: { list: listLoginLogs },
  columns: [
    // username 过滤后端真生效（username= 精确匹配）
    { prop: 'username', label: '用户名', width: 130, filterable: true },
    { prop: 'ip', label: 'IP', width: 140 },
    { prop: 'user_agent', label: 'User-Agent', minWidth: 200 },
    { prop: 'success', label: '结果', width: 80, formatter: (_r, v) => (Number(v) === 1 ? '成功' : '失败') },
    { prop: 'reason', label: '原因', minWidth: 140 },
    // sortable 降级：后端列表暂固定 id DESC、不消费 sort/order（已上报），补参后零改动生效
    { prop: 'created_at', label: '登录时间', minWidth: 165, sortable: true, formatter: (_r, v) => dateText(v) },
  ],
}

// ---- 清理（破坏性：后端固定删 3 个月前全部登录日志）----
async function onClean(): Promise<void> {
  try {
    await ElMessageBox.confirm(
      '将删除 3 个月之前的全部登录日志（后端固定保留近 3 个月，不可恢复），是否继续？',
      '清理登录日志',
      { confirmButtonText: '确定清理', cancelButtonText: '取消', type: 'warning' },
    )
  } catch {
    return // 用户取消
  }
  let deleted = 0
  try {
    const res = await cleanLoginLogs()
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
        <el-button v-permission="'sys:loginlog:clean'" type="danger" plain :icon="Delete" @click="onClean">
          清理 3 个月前日志
        </el-button>
      </template>
    </XTable>
  </el-card>
</template>
