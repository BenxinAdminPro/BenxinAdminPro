<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   参数管理 — sys_config CRUD（#row-actions 插槽消费页；加密参数安全降级：脱敏展示+禁界面编辑）
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-10 09:38:01
  | @updated   2026-06-14 11:58:00  T-005b-4：config_key 模糊 + created_at 排序后端已就绪，列筛选/排序真生效
  +----------------------------------------------------------------------
  说明：列表/新增走 x-table 内置（permPrefix sys:config）；行操作经 #row-actions 插槽自定义。
       加密参数（is_encrypted=1）：值由后端恒脱敏为 ******（maskEncrypted，T-005），前端无任何
       明文展示/打印/回填路径。其「编辑」按钮禁用——后端 /sys/configs 更新为全量覆写且无再加密
       链路（CreateConfigInput 无 is_encrypted、Update 明文直写），界面编辑无论留空还是重填都会
       破坏密文，故安全降级为禁编辑（缺口已按 T-007c 模式上报，后端补语义后放开）。
       明文参数编辑走本页自有弹窗（config_key 唯一键禁改；表单仅回填非敏感明文值）。
       T-005b-4：后端 list 已支持 config_key 模糊 + created_at 排序，列筛选/排序真生效（加密写链路仍归 T-005b-3）。
-->
<script setup lang="ts">
import { ref, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Edit, Delete } from '@element-plus/icons-vue'
import XTable from '@/components/x-table/XTable.vue'
import type { XTableConfig, XRow } from '@/components/x-table/types'
import { listConfigs, createConfig, updateConfig, removeConfig, type ConfigRow } from '@/api/config'

const tableRef = ref<InstanceType<typeof XTable>>()

const encText = (v: unknown) => (Number(v) === 1 ? '是' : '否')
const isEncrypted = (row: XRow) => Number((row as ConfigRow).is_encrypted) === 1

const config: XTableConfig = {
  permPrefix: 'sys:config',
  api: {
    list: listConfigs,
    // 新增仅明文参数：后端 CreateConfigInput 无 is_encrypted，加密参数无法经管理 API 创建（已上报）
    create: createConfig,
  },
  columns: [
    // T-005b-4：config_key 模糊过滤后端已就绪，filterable 真生效
    { prop: 'config_key', label: '参数键', minWidth: 150, filterable: true },
    { prop: 'name', label: '参数名', minWidth: 110 },
    { prop: 'config_value', label: '参数值', minWidth: 140 },
    { prop: 'is_encrypted', label: '加密', width: 70, formatter: (_r, v) => encText(v) },
    { prop: 'remark', label: '备注', minWidth: 120 },
  ],
  fields: [
    { prop: 'config_key', label: '参数键', required: true, placeholder: '如 site.title' },
    { prop: 'name', label: '参数名' },
    { prop: 'config_value', label: '参数值', type: 'textarea' },
    { prop: 'remark', label: '备注', type: 'textarea' },
  ],
  actionsWidth: 150,
}

// ---- 编辑（自有弹窗，仅明文参数可达；加密行按钮已禁用）----
const editVisible = ref(false)
const editForm = reactive({ id: '', config_key: '', name: '', config_value: '', remark: '' })

function openEdit(row: XRow): void {
  const r = row as ConfigRow
  editForm.id = r.id
  editForm.config_key = r.config_key
  editForm.name = r.name
  // 仅明文参数走到这里（加密行编辑禁用），回填的是非敏感明文值，无明文泄漏面
  editForm.config_value = r.config_value
  editForm.remark = r.remark
  editVisible.value = true
}

async function submitEdit(): Promise<void> {
  try {
    await updateConfig(editForm.id, {
      config_key: editForm.config_key,
      config_value: editForm.config_value,
      name: editForm.name,
      remark: editForm.remark,
    })
  } catch {
    return // 请求层已 toast（如 409 参数键已存在），保留弹窗供修正
  }
  ElMessage.success('保存成功')
  editVisible.value = false
  tableRef.value?.refresh()
}

// ---- 删除（hashid 透传 + 二次确认）----
async function onDelete(row: XRow): Promise<void> {
  const r = row as ConfigRow
  try {
    await ElMessageBox.confirm(`确认删除参数「${r.config_key}」？`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
  } catch {
    return // 用户取消
  }
  try {
    await removeConfig(r.id)
  } catch {
    return // 请求层已 toast
  }
  ElMessage.success('删除成功')
  tableRef.value?.refresh()
}
</script>

<template>
  <el-card shadow="never">
    <XTable ref="tableRef" :config="config">
      <!-- 工具栏插槽：加密参数脱敏说明 -->
      <template #toolbar>
        <el-tag type="info">加密参数值恒显示为 ******（服务端脱敏）</el-tag>
      </template>

      <!-- 行操作插槽：编辑（加密行安全降级禁用）+ 删除，均挂真实权限码 -->
      <template #row-actions="{ row }">
        <span v-permission="'sys:config:update'" class="cfg-op">
          <el-tooltip
            v-if="isEncrypted(row)"
            content="加密参数暂不支持界面编辑（后端缺更新加密链路，已上报）"
            placement="top"
          >
            <span>
              <el-button link type="primary" :icon="Edit" disabled>编辑</el-button>
            </span>
          </el-tooltip>
          <el-button v-else link type="primary" :icon="Edit" @click="openEdit(row)">编辑</el-button>
        </span>
        <el-button v-permission="'sys:config:delete'" link type="danger" :icon="Delete" @click="onDelete(row)">
          删除
        </el-button>
      </template>
    </XTable>

    <!-- 编辑弹窗（仅明文参数；config_key 唯一键禁改） -->
    <el-dialog v-model="editVisible" title="编辑参数" width="480px" destroy-on-close>
      <el-form :model="editForm" label-width="92px">
        <el-form-item label="参数键">
          <el-input v-model="editForm.config_key" disabled />
        </el-form-item>
        <el-form-item label="参数名">
          <el-input v-model="editForm.name" />
        </el-form-item>
        <el-form-item label="参数值">
          <el-input v-model="editForm.config_value" type="textarea" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="editForm.remark" type="textarea" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" @click="submitEdit">确定</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<style scoped lang="scss">
.cfg-op {
  margin-right: 8px;
}
</style>
