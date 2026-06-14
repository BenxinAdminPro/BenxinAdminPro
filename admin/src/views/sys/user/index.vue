<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   用户管理 — x-table 配置化 CRUD 样例（按钮挂 v-permission，数据范围后端 enforce）
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-08 16:00:00
  | @updated   2026-06-12 14:24:57  T-007h：表单嵌入部门树/岗位选择器 + api.get 编辑回填（T-003e hashid 入参收口的消费验证）
  | @updated   2026-06-14 14:30:00  T-008a：+重置密码行操作 + status 假能力修复（编辑表单去 status，改行操作切换接 PUT :id/status）
  | @updated   2026-06-14 16:45:00  T-008b：+分配角色行操作（多选弹窗回填+全量覆写）+ 列表「角色」列（后端批量回填）
  +----------------------------------------------------------------------
  回填语义（以后端源码为准）：
   - 详情 GET /sys/users/:id 出参 dept_id 为 hashid|null、posts 仅非空时出现（response.go User）；
     适配为表单值 dept_id: ''|hashid、post_ids: hashid[]，hashid 全程透传不解码。
   - 提交（updateUserReq）dept_id ''=清空部门；post_ids 数组=全量覆写岗位（[]=清空）。
     编辑必先经 api.get 回填再提交，避免无回填的全量覆写静默清空 dept/posts
     （改造前旧表单不带 dept_id 字段，每次编辑都会把部门清成 NULL——本片连带修复）。
  T-008a status 假能力修复：updateUserReq 后端无 Status 字段（handler_user.go:312），编辑弹窗里改
   status 提交被静默吞（T-007h §8-3 假能力）。本片把 status 字段标 createOnly（仅新增可设初始态、
   编辑弹窗不再显示 → 假能力根除），状态变更改走「行操作切换」调独立端点 PUT /sys/users/:id/status。
  T-008a 状态开关落点说明：§2 原意「列内 el-switch」需 x-table 单元格插槽（改 x-table 核心，§2 禁止）；
   且覆写 #row-actions 插槽会丢内置编辑/删除（插槽仅暴露 row）。故按 §2 显式回退「走行操作插槽」，
   用 config.actions 行操作按钮（动态二次确认）实现状态切换，与内置编辑/删除并列、不改 x-table 核心。
-->
<script setup lang="ts">
import { ref, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Key, Switch, UserFilled } from '@element-plus/icons-vue'
import XTable from '@/components/x-table/XTable.vue'
import type { XTableConfig, XRow } from '@/components/x-table/types'
import DeptTreeSelect from '@/components/selectors/DeptTreeSelect.vue'
import PostSelect from '@/components/selectors/PostSelect.vue'
import {
  listUsers, createUser, updateUser, removeUser, getUser,
  resetUserPassword, setUserStatus, assignUserRoles,
} from '@/api/user'
import { listAllRoles, type RoleRow } from '@/api/role'

const statusText = (v: unknown) => (Number(v) === 0 ? '正常' : '停用')
const dateText = (v: unknown) => (typeof v === 'string' ? v.slice(0, 19).replace('T', ' ') : '')
// 角色列：roles 数组 → 角色名顿号分隔；无角色显「—」。
const rolesText = (v: unknown) => {
  const rs = Array.isArray(v) ? (v as RoleRow[]) : []
  return rs.length ? rs.map((r) => r.name).join('、') : '—'
}

// ---- 分配角色弹窗（页级控件；全量覆写，回填以当前已授为基准）----
const roleVisible = ref(false)
const roleTarget = reactive<{ id: string; username: string }>({ id: '', username: '' })
const roleOptions = ref<RoleRow[]>([])
const roleSelected = ref<string[]>([]) // 角色 hashid 数组
const roleSubmitting = ref(false)

async function openAssignRoles(row: XRow): Promise<void> {
  const id = String(row.id)
  try {
    // 回填以「当前已授全量」为基准（全量覆写下回填是正确性前提）；并行取全量角色选项。
    const [detail, all] = await Promise.all([getUser(id), listAllRoles()])
    roleOptions.value = all
    roleSelected.value = (detail.roles ?? []).map((r) => r.id)
  } catch {
    // 回填/选项拉取失败 → 不开弹窗（避免残缺回填被全量覆写静默清空，同 T-007h 防误清）
    return
  }
  roleTarget.id = id
  roleTarget.username = String(row.username ?? '')
  roleVisible.value = true
}

async function submitAssignRoles(): Promise<void> {
  roleSubmitting.value = true
  try {
    await assignUserRoles(roleTarget.id, roleSelected.value)
    ElMessage.success('角色已分配')
    roleVisible.value = false
  } catch {
    // 请求层已 toast；保留弹窗供修正，不冒未处理 rejection
  } finally {
    roleSubmitting.value = false
  }
}

// ---- 重置密码弹窗（页级控件，非 x-table 内置）----
const pwdVisible = ref(false)
const pwdTarget = reactive<{ id: string; username: string }>({ id: '', username: '' })
const pwdForm = reactive<{ password: string; confirm: string }>({ password: '', confirm: '' })
const pwdSubmitting = ref(false)

function openResetPwd(row: XRow): void {
  pwdTarget.id = String(row.id)
  pwdTarget.username = String(row.username ?? '')
  pwdForm.password = ''
  pwdForm.confirm = ''
  pwdVisible.value = true
}

async function submitResetPwd(): Promise<void> {
  // 前端预校验仅 UX：后端入参仅 binding:"required"（非空），无长度/强度校验，不伪造规则。
  if (!pwdForm.password) {
    ElMessage.warning('请输入新密码')
    return
  }
  if (pwdForm.password !== pwdForm.confirm) {
    ElMessage.warning('两次输入的密码不一致')
    return
  }
  pwdSubmitting.value = true
  try {
    await resetUserPassword(pwdTarget.id, pwdForm.password)
    ElMessage.success('密码已重置')
    pwdVisible.value = false
    // 不回显/不缓存明文：关闭即清空（destroy-on-close 亦兜底）
    pwdForm.password = ''
    pwdForm.confirm = ''
  } catch {
    // 请求层已按错误码 toast；此处保留弹窗供修正，不冒未处理 rejection
  } finally {
    pwdSubmitting.value = false
  }
}

// ---- 状态切换（行操作，调独立端点 PUT :id/status）----
async function toggleStatus(row: XRow): Promise<void> {
  const target = Number(row.status) === 0 ? 1 : 0
  const word = target === 0 ? '启用' : '停用'
  try {
    await ElMessageBox.confirm(`确认${word}用户「${row.username}」？`, '状态变更', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
  } catch {
    return // 用户取消
  }
  try {
    await setUserStatus(String(row.id), target)
    ElMessage.success(`已${word}`)
  } catch {
    // 请求层已 toast；吞掉避免未处理 rejection（refresh 由 action.refresh 触发，失败态不写脏行）
  }
}

const config: XTableConfig = {
  permPrefix: 'sys:user',
  actionsWidth: 330, // 容纳 编辑/删除/分配角色/重置密码/状态切换
  actions: [
    {
      label: '分配角色',
      perm: 'sys:user:assign',
      type: 'primary',
      icon: UserFilled,
      handler: (row: XRow) => openAssignRoles(row),
    },
    {
      label: '重置密码',
      perm: 'sys:user:password',
      type: 'warning',
      icon: Key,
      handler: (row: XRow) => openResetPwd(row),
    },
    {
      label: '停用/启用',
      perm: 'sys:user:status',
      type: 'info',
      icon: Switch,
      handler: (row: XRow) => toggleStatus(row),
      refresh: true, // 切换后刷新列表，状态列文案随之更新
    },
  ],
  api: {
    list: listUsers,
    create: createUser,
    update: updateUser,
    remove: removeUser,
    // 编辑回填：详情出参 → 表单值适配（null→''、posts→post_ids），hashid 原样透传
    get: async (id: string) => {
      const u = await getUser(id)
      return {
        ...u,
        dept_id: u.dept_id ?? '',
        post_ids: (u.posts ?? []).map((p) => p.id),
      }
    },
  },
  search: [
    { prop: 'username', label: '用户名' },
    {
      prop: 'status',
      label: '状态',
      type: 'select',
      options: [
        { label: '正常', value: 0 },
        { label: '停用', value: 1 },
      ],
    },
  ],
  columns: [
    { prop: 'username', label: '用户名', minWidth: 120 },
    { prop: 'nickname', label: '昵称', minWidth: 120 },
    { prop: 'mobile', label: '手机号', minWidth: 120 },
    // T-008b：角色列（后端 List 批量回填 roles）。多个顿号分隔、无角色显「—」。
    // 文本展示（非 el-tag）：el-tag 需 x-table 单元格插槽=改核心，§2 禁（同 status 落点）。
    { prop: 'roles', label: '角色', minWidth: 160, formatter: (_r, v) => rolesText(v) },
    { prop: 'status', label: '状态', width: 90, formatter: (_r, v) => statusText(v) },
    { prop: 'created_at', label: '创建时间', minWidth: 160, formatter: (_r, v) => dateText(v) },
  ],
  fields: [
    { prop: 'username', label: '用户名', required: true, editable: false },
    { prop: 'password', label: '密码', type: 'password', required: true, createOnly: true },
    { prop: 'nickname', label: '昵称' },
    { prop: 'dept_id', label: '部门', type: 'slot', default: '' },
    { prop: 'post_ids', label: '岗位', type: 'slot', default: [] },
    { prop: 'email', label: '邮箱' },
    { prop: 'mobile', label: '手机号' },
    {
      // T-008a：status 标 createOnly —— 仅新增可设初始状态（CreateUserInput 支持 Status）；
      // 编辑弹窗不再显示 status（updateUserReq 无 Status 字段，编辑改它会被静默吞=假能力）。
      // 状态变更改走行操作「停用/启用」调独立端点 PUT /sys/users/:id/status。
      prop: 'status',
      label: '状态',
      type: 'select',
      default: 0,
      createOnly: true,
      options: [
        { label: '正常', value: 0 },
        { label: '停用', value: 1 },
      ],
    },
    { prop: 'remark', label: '备注', type: 'textarea' },
  ],
}
</script>

<template>
  <el-card shadow="never">
    <XTable :config="config">
      <template #field-dept_id="{ form, disabled }">
        <DeptTreeSelect v-model="form.dept_id as string" :disabled="disabled" />
      </template>
      <template #field-post_ids="{ form, disabled }">
        <PostSelect v-model="form.post_ids as string[]" :disabled="disabled" />
      </template>
    </XTable>

    <!-- 重置密码弹窗（页级控件；新密码不回显既有、不缓存、关闭即清空） -->
    <el-dialog
      v-model="pwdVisible"
      :title="`重置密码 - ${pwdTarget.username}`"
      width="420px"
      destroy-on-close
    >
      <el-form label-width="90px" @submit.prevent>
        <el-form-item label="新密码" required>
          <el-input
            v-model="pwdForm.password"
            type="password"
            show-password
            autocomplete="new-password"
            placeholder="请输入新密码"
          />
        </el-form-item>
        <el-form-item label="确认密码" required>
          <el-input
            v-model="pwdForm.confirm"
            type="password"
            show-password
            autocomplete="new-password"
            placeholder="请再次输入新密码"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pwdVisible = false">取消</el-button>
        <el-button type="primary" :loading="pwdSubmitting" @click="submitResetPwd">确定</el-button>
      </template>
    </el-dialog>

    <!-- 分配角色弹窗（多选；回填当前已授 → 改选 → 全量覆写 PUT :id/roles） -->
    <el-dialog
      v-model="roleVisible"
      :title="`分配角色 - ${roleTarget.username}`"
      width="460px"
      destroy-on-close
    >
      <el-select
        v-model="roleSelected"
        multiple
        clearable
        collapse-tags
        collapse-tags-tooltip
        placeholder="选择角色（留空=移除全部角色）"
        style="width: 100%"
      >
        <el-option
          v-for="r in roleOptions"
          :key="r.id"
          :label="`${r.name}（${r.code}）`"
          :value="r.id"
        />
      </el-select>
      <div class="role-assign-hint">提交后将以当前选择全量覆写该用户角色（含 Casbin 权限联动）。</div>
      <template #footer>
        <el-button @click="roleVisible = false">取消</el-button>
        <el-button type="primary" :loading="roleSubmitting" @click="submitAssignRoles">确定</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<style scoped>
.role-assign-hint {
  margin-top: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
