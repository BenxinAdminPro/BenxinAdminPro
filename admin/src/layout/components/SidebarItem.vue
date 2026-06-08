<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   侧边栏菜单项（递归）— M 目录→子菜单，C 菜单→叶子，F 按钮不渲染
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-08 16:00:00
  +----------------------------------------------------------------------
-->
<script setup lang="ts">
import { computed } from 'vue'
import { resolveIcon } from './menuIcon'
import type { MenuNode } from '@/api/authInfo'

defineOptions({ name: 'SidebarItem' })
const props = defineProps<{ node: MenuNode }>()

// 仅 M/C 进侧边栏（F 按钮不渲染）
const visibleChildren = computed(
  () => (props.node.children || []).filter((c) => c.menu_type === 'M' || c.menu_type === 'C'),
)
const isDir = computed(() => props.node.menu_type === 'M' && visibleChildren.value.length > 0)
</script>

<template>
  <!-- 目录 M（有可见子项）→ 子菜单 -->
  <el-sub-menu v-if="isDir" :index="node.path || node.id">
    <template #title>
      <el-icon><component :is="resolveIcon(node.icon)" /></el-icon>
      <span>{{ node.name }}</span>
    </template>
    <SidebarItem v-for="child in visibleChildren" :key="child.id" :node="child" />
  </el-sub-menu>

  <!-- 菜单 C → 叶子 -->
  <el-menu-item v-else-if="node.menu_type === 'C'" :index="node.path">
    <el-icon><component :is="resolveIcon(node.icon)" /></el-icon>
    <template #title>{{ node.name }}</template>
  </el-menu-item>
</template>
