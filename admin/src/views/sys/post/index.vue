<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   岗位管理 — x-table 配置化 CRUD（扁平列表，code 唯一 409 友好码，软删不解除用户关联）
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-12 14:24:57
  +----------------------------------------------------------------------
  说明（以后端 rbac/handler_post.go + post_service.go 为准）：
   - 列表仅 page/page_size，无 search/filter/sort——本页不伪造前端假搜索，诚实降级；
   - code 全局唯一：重名 409 ErrPostCodeExists（offset+34，demo 段=11034；T-004e 1062 兜底），弹窗保留供修正；
   - 删除为软删且【后端不检查用户关联】：已分配用户的关联残留但不再生效（用户详情
     按活跃岗位查询，软删岗位自然消失）——二次确认文案按此真实行为措辞。
-->
<script setup lang="ts">
import XTable from '@/components/x-table/XTable.vue'
import type { XTableConfig } from '@/components/x-table/types'
import { listPosts, createPost, updatePost, removePost } from '@/api/post'
import { statusText, dateText } from '@/utils/format'

const config: XTableConfig = {
  permPrefix: 'sys:post',
  api: {
    list: listPosts,
    create: createPost,
    update: updatePost,
    remove: removePost,
  },
  columns: [
    { prop: 'code', label: '岗位编码', minWidth: 140 },
    { prop: 'name', label: '岗位名称', minWidth: 140 },
    { prop: 'sort', label: '排序', width: 80 },
    { prop: 'status', label: '状态', width: 90, formatter: (_r, v) => statusText(v) },
    { prop: 'created_at', label: '创建时间', minWidth: 160, formatter: (_r, v) => dateText(v) },
  ],
  fields: [
    { prop: 'code', label: '岗位编码', required: true, placeholder: '全局唯一，如 dev_engineer' },
    { prop: 'name', label: '岗位名称', required: true },
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
  ],
  // 按后端真实删除行为措辞：软删、不可恢复；用户关联不自动解除、但岗位即刻对用户失效
  delConfirm: '确定删除该岗位？删除后不可恢复，已分配该岗位的用户将不再持有此岗位。',
}
</script>

<template>
  <el-card shadow="never">
    <XTable :config="config" />
  </el-card>
</template>
