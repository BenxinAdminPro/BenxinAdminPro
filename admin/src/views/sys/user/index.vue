<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   用户管理 — x-table 配置化 CRUD 样例（按钮挂 v-permission，数据范围后端 enforce）
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-08 16:00:00
  | @updated   2026-06-12 14:24:57  T-007h：表单嵌入部门树/岗位选择器 + api.get 编辑回填（T-003e hashid 入参收口的消费验证）
  +----------------------------------------------------------------------
  回填语义（以后端源码为准）：
   - 详情 GET /sys/users/:id 出参 dept_id 为 hashid|null、posts 仅非空时出现（response.go User）；
     适配为表单值 dept_id: ''|hashid、post_ids: hashid[]，hashid 全程透传不解码。
   - 提交（updateUserReq）dept_id ''=清空部门；post_ids 数组=全量覆写岗位（[]=清空）。
     编辑必先经 api.get 回填再提交，避免无回填的全量覆写静默清空 dept/posts
     （改造前旧表单不带 dept_id 字段，每次编辑都会把部门清成 NULL——本片连带修复）。
-->
<script setup lang="ts">
import XTable from '@/components/x-table/XTable.vue'
import type { XTableConfig } from '@/components/x-table/types'
import DeptTreeSelect from '@/components/selectors/DeptTreeSelect.vue'
import PostSelect from '@/components/selectors/PostSelect.vue'
import { listUsers, createUser, updateUser, removeUser, getUser } from '@/api/user'

const statusText = (v: unknown) => (Number(v) === 0 ? '正常' : '停用')
const dateText = (v: unknown) => (typeof v === 'string' ? v.slice(0, 19).replace('T', ' ') : '')

const config: XTableConfig = {
  permPrefix: 'sys:user',
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
      prop: 'status',
      label: '状态',
      type: 'select',
      default: 0,
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
  </el-card>
</template>
