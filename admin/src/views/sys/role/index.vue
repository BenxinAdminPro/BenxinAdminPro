<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   角色管理 — x-table 配置化 CRUD 样例（基本信息 + 数据范围）
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-08 16:00:00
  +----------------------------------------------------------------------
-->
<script setup lang="ts">
import XTable from '@/components/x-table/XTable.vue'
import type { XTableConfig } from '@/components/x-table/types'
import { listRoles, createRole, updateRole, removeRole } from '@/api/role'

const scopeMap: Record<number, string> = { 1: '全部', 2: '本人', 3: '本部门' }
const scopeText = (v: unknown) => scopeMap[Number(v)] || '本人'
const statusText = (v: unknown) => (Number(v) === 0 ? '正常' : '停用')

const config: XTableConfig = {
  permPrefix: 'sys:role',
  api: {
    list: listRoles,
    create: createRole,
    update: updateRole,
    remove: removeRole,
  },
  columns: [
    { prop: 'code', label: '角色编码', minWidth: 140 },
    { prop: 'name', label: '角色名称', minWidth: 120 },
    { prop: 'data_scope', label: '数据范围', width: 100, formatter: (_r, v) => scopeText(v) },
    { prop: 'sort', label: '排序', width: 80 },
    { prop: 'status', label: '状态', width: 90, formatter: (_r, v) => statusText(v) },
  ],
  fields: [
    { prop: 'code', label: '角色编码', required: true, editable: false },
    { prop: 'name', label: '角色名称', required: true },
    {
      prop: 'data_scope',
      label: '数据范围',
      type: 'select',
      default: 2,
      options: [
        { label: '全部', value: 1 },
        { label: '本人', value: 2 },
        { label: '本部门', value: 3 },
      ],
    },
    { prop: 'sort', label: '排序', type: 'number', default: 0 },
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
