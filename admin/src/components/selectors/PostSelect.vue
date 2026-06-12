<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   岗位多选器 — el-select 多选消费 /sys/posts 全量（hashid 透传，v-model 为 hashid[]）
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-12 14:24:57
  +----------------------------------------------------------------------
  说明：用户↔岗位为多对多（sys_user_post），后端入参 post_ids 为 hashid 数组
       （decodeIDSlice：缺省=不变 / []=清空），本组件 v-model 恒为数组——表单提交即
       "全量覆写为当前所选"，配合编辑回填（详情 posts → post_ids）语义自洽。
       岗位列表分页上限 100/页，经 listAllPosts 循环取齐不静默截断。
       停用岗位（status=1）如实展示并标注（后端对 post_ids 无状态校验，前端不伪造约束）。
-->
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listAllPosts, type PostRow } from '@/api/post'

defineProps<{ disabled?: boolean }>()

// v-model：hashid 数组；[] = 清空岗位（后端 decodeIDSlice 空数组→清空）
const model = defineModel<string[]>({ default: () => [] })

const posts = ref<PostRow[]>([])
const loading = ref(false)

async function fetchPosts(): Promise<void> {
  loading.value = true
  try {
    posts.value = await listAllPosts()
  } catch {
    // 请求层已按错误码 toast（如无 sys:post:list 权限 403）；保留空选项，不伪造数据
  } finally {
    loading.value = false
  }
}

function postLabel(p: PostRow): string {
  return p.status === 1 ? `${p.name}（停用）` : p.name
}

onMounted(fetchPosts)
</script>

<template>
  <el-select
    v-model="model"
    multiple
    clearable
    :loading="loading"
    :disabled="disabled"
    placeholder="不选 = 无岗位"
    style="width: 100%"
  >
    <el-option v-for="p in posts" :key="p.id" :label="postLabel(p)" :value="p.id" />
  </el-select>
</template>
