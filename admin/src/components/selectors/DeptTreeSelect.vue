<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   部门树选择器 — el-tree-select 直接消费后端嵌套树（hashid 透传，v-model 为 hashid/空串）
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-12 14:24:57
  +----------------------------------------------------------------------
  说明：后端 GET /sys/depts/tree 返回【已嵌套树】（服务端 buildTree，sort ASC, id ASC），
       本组件零再组装直接消费（同菜单页消费 /sys/menus/tree 的口径）；utils/tree.ts 的
       buildTree 在本场景无需介入——选部门给用户不存在"挂到自己子孙下"问题（subtreeIds
       亦不需要），将来部门管理页的父级选择器才是其 dept 侧消费点。
       v-model 语义对齐后端入参（decodeOptionalID）：'' = 无部门/清空，hashid = 指定部门。
       清空选择提交空串即清空用户部门，是后端真实语义，非前端约定。
       停用部门（status=1）如实展示并标注，不擅自隐藏（后端对 dept_id 无状态校验，前端不伪造约束）。
-->
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getDeptTree, type DeptNode } from '@/api/dept'

defineProps<{ disabled?: boolean }>()

// v-model：hashid 字符串；'' = 无部门（后端 decodeOptionalID 空串→nil）
const model = defineModel<string>({ default: '' })

const tree = ref<DeptNode[]>([])
const loading = ref(false)

async function fetchTree(): Promise<void> {
  loading.value = true
  try {
    tree.value = (await getDeptTree()) || []
  } catch {
    // 请求层已按错误码 toast（如无 sys:dept:tree 权限 403）；保留空选项，不伪造数据
  } finally {
    loading.value = false
  }
}

function nodeLabel(d: DeptNode): string {
  return d.status === 1 ? `${d.name}（停用）` : d.name
}

onMounted(fetchTree)
</script>

<template>
  <el-tree-select
    v-model="model"
    :data="tree"
    :loading="loading"
    :props="{ label: nodeLabel, children: 'children' }"
    node-key="id"
    value-key="id"
    check-strictly
    clearable
    :disabled="disabled"
    placeholder="不选 = 无部门"
    style="width: 100%"
  />
</template>
