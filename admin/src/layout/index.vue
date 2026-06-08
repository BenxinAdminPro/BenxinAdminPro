<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   布局壳 — 响应式：桌面/平板固定侧栏，手机抽屉；顶栏 + 内容区
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-08 14:00:00
  +----------------------------------------------------------------------
-->
<script setup lang="ts">
import { computed } from 'vue'
import { useAppStore } from '@/store/app'
import { useDevice } from '@/composables/useDevice'
import Sidebar from './components/Sidebar.vue'
import Navbar from './components/Navbar.vue'
import AppMain from './components/AppMain.vue'

useDevice()
const app = useAppStore()

const isMobile = computed(() => app.device === 'mobile')
const asideWidth = computed(() => (app.sidebarCollapsed ? '64px' : '210px'))

function onDrawer(open: boolean): void {
  app.setSidebar(!open)
}
</script>

<template>
  <div class="layout">
    <!-- 桌面/平板：固定侧栏 -->
    <aside v-if="!isMobile" class="layout__aside" :style="{ width: asideWidth }">
      <Sidebar :collapse="app.sidebarCollapsed" />
    </aside>

    <!-- 手机：抽屉侧栏 -->
    <el-drawer
      v-else
      :model-value="!app.sidebarCollapsed"
      :with-header="false"
      :size="210"
      direction="ltr"
      class="layout__drawer"
      @update:model-value="onDrawer"
    >
      <Sidebar :collapse="false" />
    </el-drawer>

    <div class="layout__body">
      <header class="layout__header">
        <Navbar />
      </header>
      <AppMain />
    </div>
  </div>
</template>

<style scoped lang="scss">
.layout {
  display: flex;
  height: 100vh;
  overflow: hidden;

  &__aside {
    flex-shrink: 0;
    height: 100%;
    transition: width 0.25s ease;
    overflow: hidden;
  }
  &__body {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  &__header {
    flex-shrink: 0;
  }
  &__drawer {
    :deep(.el-drawer__body) {
      padding: 0;
    }
  }
}
</style>
