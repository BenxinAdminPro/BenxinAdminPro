<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   顶栏 — 折叠按钮 + 暗色/语言切换 + 用户下拉(登出)
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-08 14:00:00
  +----------------------------------------------------------------------
-->
<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessageBox } from 'element-plus'
import { Fold, Expand, Moon, Sunny, ArrowDown } from '@element-plus/icons-vue'
import { useAppStore } from '@/store/app'
import { useUserStore } from '@/store/user'
import { isDark, toggleDark } from '@/theme'
import { setLocale, getStoredLocale, type LocaleKey } from '@/i18n'

const app = useAppStore()
const user = useUserStore()
const router = useRouter()
const { t } = useI18n()

const currentLocale = computed(() => getStoredLocale())

function onLocale(locale: LocaleKey): void {
  setLocale(locale)
}

async function onLogout(): Promise<void> {
  await ElMessageBox.confirm(t('common.logoutConfirm'), t('common.logout'), {
    confirmButtonText: t('common.confirm'),
    cancelButtonText: t('common.cancel'),
    type: 'warning',
  })
  await user.logoutAction()
  router.replace('/login')
}
</script>

<template>
  <div class="navbar flex-between">
    <div class="navbar__left flex items-center">
      <el-icon class="navbar__toggle" @click="app.toggleSidebar()">
        <Fold v-if="!app.sidebarCollapsed" />
        <Expand v-else />
      </el-icon>
    </div>

    <div class="navbar__right flex items-center">
      <!-- 暗色切换 -->
      <el-tooltip :content="t('layout.theme')" placement="bottom">
        <el-icon class="navbar__action" @click="toggleDark()">
          <Moon v-if="!isDark" />
          <Sunny v-else />
        </el-icon>
      </el-tooltip>

      <!-- 语言切换 -->
      <el-dropdown class="navbar__action" trigger="click" @command="onLocale">
        <span class="navbar__lang">{{ currentLocale === 'en' ? 'EN' : '中' }}</span>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="zh-CN" :disabled="currentLocale === 'zh-CN'">简体中文</el-dropdown-item>
            <el-dropdown-item command="en" :disabled="currentLocale === 'en'">English</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>

      <!-- 用户下拉 -->
      <el-dropdown trigger="click">
        <span class="navbar__user flex items-center">
          <el-avatar :size="28">{{ (user.username[0] || 'U').toUpperCase() }}</el-avatar>
          <span class="navbar__username">{{ user.username || 'User' }}</span>
          <el-icon><ArrowDown /></el-icon>
        </span>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item divided @click="onLogout">{{ t('common.logout') }}</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </div>
</template>

<style scoped lang="scss">
.navbar {
  height: 56px;
  padding: 0 16px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-light);

  &__toggle,
  &__action {
    font-size: 18px;
    cursor: pointer;
    padding: 8px;
    border-radius: 6px;
    color: var(--el-text-color-regular);
    &:hover {
      background: var(--el-fill-color-light);
    }
  }
  &__right {
    gap: 4px;
  }
  &__lang {
    font-size: 14px;
    font-weight: 600;
    user-select: none;
  }
  &__user {
    cursor: pointer;
    gap: 8px;
    padding: 4px 8px;
    outline: none;
  }
  &__username {
    font-size: 14px;
    color: var(--el-text-color-regular);
  }
}
</style>
