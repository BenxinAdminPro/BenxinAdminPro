<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   侧边栏 — Logo + 动态菜单（消费 store.menuTree，T-007b）
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-08 14:00:00
  | @updated   2026-06-08 16:00:00  T-007b：菜单改为按后端 menus 动态渲染（递归 SidebarItem）
  +----------------------------------------------------------------------
-->
<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { HomeFilled } from '@element-plus/icons-vue'
import { useUserStore } from '@/store/user'
import SidebarItem from './SidebarItem.vue'

defineProps<{ collapse: boolean }>()
const route = useRoute()
const { t } = useI18n()
const user = useUserStore()
const activeMenu = computed(() => route.path)
// 顶层只渲染 M/C（F 不进侧栏）
const topMenus = computed(() => user.menuTree.filter((m) => m.menu_type === 'M' || m.menu_type === 'C'))
</script>

<template>
  <div class="sidebar">
    <div class="sidebar__logo flex-center">
      <span v-if="!collapse" class="sidebar__logo-text">BenxinAdminPro</span>
      <span v-else class="sidebar__logo-mini">B</span>
    </div>
    <el-scrollbar>
      <el-menu :collapse="collapse" :default-active="activeMenu" router unique-opened>
        <!-- 固定首页 -->
        <el-menu-item index="/dashboard">
          <el-icon><HomeFilled /></el-icon>
          <template #title>{{ t('layout.dashboard') }}</template>
        </el-menu-item>
        <!-- 动态菜单（后端 menus） -->
        <SidebarItem v-for="node in topMenus" :key="node.id" :node="node" />
      </el-menu>
    </el-scrollbar>
  </div>
</template>

<style scoped lang="scss">
.sidebar {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color);
  border-right: 1px solid var(--el-border-color-light);

  &__logo {
    height: 56px;
    flex-shrink: 0;
    overflow: hidden;
    border-bottom: 1px solid var(--el-border-color-light);
  }
  &__logo-text {
    font-size: 16px;
    font-weight: 700;
    color: var(--el-color-primary);
    white-space: nowrap;
  }
  &__logo-mini {
    font-size: 20px;
    font-weight: 800;
    color: var(--el-color-primary);
  }

  :deep(.el-menu) {
    border-right: none;
  }
}
</style>
