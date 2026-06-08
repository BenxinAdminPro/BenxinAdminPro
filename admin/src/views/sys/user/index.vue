<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   用户管理 — x-table 配置化 CRUD 样例（按钮挂 v-permission，数据范围后端 enforce）
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-08 16:00:00
  +----------------------------------------------------------------------
-->
<script setup lang="ts">
import XTable from '@/components/x-table/XTable.vue'
import type { XTableConfig } from '@/components/x-table/types'
import { listUsers, createUser, updateUser, removeUser } from '@/api/user'

const statusText = (v: unknown) => (Number(v) === 0 ? '正常' : '停用')
const dateText = (v: unknown) => (typeof v === 'string' ? v.slice(0, 19).replace('T', ' ') : '')

const config: XTableConfig = {
  permPrefix: 'sys:user',
  api: {
    list: listUsers,
    create: createUser,
    update: updateUser,
    remove: removeUser,
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
    <XTable :config="config" />
  </el-card>
</template>
