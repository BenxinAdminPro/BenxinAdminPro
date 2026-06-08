<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   侧边栏 — Logo + 菜单（T-007a 静态占位；T-007b 接 /sys/auth/menus 动态生成）
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-08 14:00:00
  +----------------------------------------------------------------------
-->
<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { HomeFilled } from '@element-plus/icons-vue'

defineProps<{ collapse: boolean }>()
const route = useRoute()
const { t } = useI18n()
const activeMenu = computed(() => route.path)
</script>

<template>
  <div class="sidebar">
    <div class="sidebar__logo flex-center">
      <span v-if="!collapse" class="sidebar__logo-text">BenxinAdminPro</span>
      <span v-else class="sidebar__logo-mini">B</span>
    </div>
    <el-scrollbar>
      <el-menu :collapse="collapse" :default-active="activeMenu" router unique-opened>
        <!-- 静态占位菜单；动态路由 + 按钮权限留 T-007b -->
        <el-menu-item index="/dashboard">
          <el-icon><HomeFilled /></el-icon>
          <template #title>{{ t('layout.dashboard') }}</template>
        </el-menu-item>
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
