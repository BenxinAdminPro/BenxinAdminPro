<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   参数管理 — sys_config CRUD（#row-actions 插槽消费页；加密参数可建可编，编辑不破坏密文）
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-10 09:38:01
  | @updated   2026-06-14 11:58:00  T-005b-4：config_key 模糊 + created_at 排序后端已就绪，列筛选/排序真生效
  | @updated   2026-06-15 10:30:00  T-005b-3：新建加 is_encrypted 选择 + 解禁加密行编辑（留空保持/重填替换，不破坏密文）
  +----------------------------------------------------------------------
  说明：列表/新增走 x-table 内置（permPrefix sys:config）；行操作经 #row-actions 插槽自定义。
       加密参数（is_encrypted=1）：值由后端恒脱敏为 ******（maskEncrypted，T-005），前端无任何
       明文展示/打印/回填路径——加密行编辑表单的值框「打开即空」，绝不回填 ****** 或明文。
       编辑语义（T-005b-3 后端指针三态）：值框「留空＝保持原密文不动」「填新值＝重新加密替换」；
       明文行编辑值框正常回填、始终提交（含清空），零回归。config_key 唯一键禁改（编辑态锁定）。
       新建：弹窗 is_encrypted 选「是」即落密文（后端 EncryptGCM），选「否」明文存。
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

const config: XTableConfig = {
  permPrefix: 'sys:config',
  api: {
    list: listConfigs,
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
    // 加密选择：选「是」后端 EncryptGCM 落密文；默认明文
    {
      prop: 'is_encrypted',
      label: '加密存储',
      type: 'select',
      default: 0,
      options: [
        { label: '否（明文）', value: 0 },
        { label: '是（加密）', value: 1 },
      ],
    },
    { prop: 'remark', label: '备注', type: 'textarea' },
  ],
  actionsWidth: 150,
}

// ---- 编辑（自有弹窗；加密行与明文行均可达，值处理按加密标志分流）----
const editVisible = ref(false)
const editForm = reactive({ id: '', config_key: '', name: '', config_value: '', remark: '', is_encrypted: 0 })
const editIsEncrypted = () => editForm.is_encrypted === 1

function openEdit(row: XRow): void {
  const r = row as ConfigRow
  editForm.id = r.id
  editForm.config_key = r.config_key
  editForm.name = r.name
  editForm.remark = r.remark
  editForm.is_encrypted = Number(r.is_encrypted)
  // 加密行：值恒为脱敏 ******，绝不回填（留空＝保持原密文）；明文行正常回填
  editForm.config_value = editForm.is_encrypted === 1 ? '' : r.config_value
  editVisible.value = true
}

async function submitEdit(): Promise<void> {
  // config_key 编辑态锁定，不提交；值字段按加密标志分流（落实后端指针三态语义）
  const payload: Record<string, unknown> = { name: editForm.name, remark: editForm.remark }
  if (editIsEncrypted()) {
    // 加密行：填了新值才提交（重新加密替换）；留空则省略 config_value → 后端保持原密文
    if (editForm.config_value !== '') payload.config_value = editForm.config_value
  } else {
    // 明文行：始终提交值（含清空），零回归
    payload.config_value = editForm.config_value
  }
  try {
    await updateConfig(editForm.id, payload)
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

      <!-- 行操作插槽：编辑（加密行同样可编，值处理见弹窗）+ 删除，均挂真实权限码 -->
      <template #row-actions="{ row }">
        <el-button v-permission="'sys:config:update'" link type="primary" :icon="Edit" class="cfg-op" @click="openEdit(row)">
          编辑
        </el-button>
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
          <el-input
            v-model="editForm.config_value"
            type="textarea"
            :placeholder="editIsEncrypted() ? '留空＝保持原密文不变，填写＝重新加密替换' : ''"
          />
          <div v-if="editIsEncrypted()" class="cfg-enc-tip">加密参数：值不回显，留空即保持原值</div>
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
.cfg-enc-tip {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
