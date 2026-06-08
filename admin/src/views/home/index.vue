<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   工作台占位页 — 登录成功落地页（业务卡片留 T-007b/后续）
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-08 14:00:00
  +----------------------------------------------------------------------
-->
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/store/user'

const { t } = useI18n()
const user = useUserStore()
</script>

<template>
  <div class="home">
    <el-card shadow="never">
      <h2 class="home__welcome">{{ t('layout.dashboard') }}</h2>
      <p class="home__hello">
        👋 {{ user.username || 'User' }}
      </p>
      <el-alert
        type="success"
        :closable="false"
        title="BenxinAdminPro Admin · T-007a 地基已就绪"
        description="请求层 / 登录 / 路由守卫 / 布局 / 主题 / i18n 已搭好。动态路由 + 按钮权限 + CRUD 列表见 T-007b。"
      />
    </el-card>
  </div>
</template>

<style scoped lang="scss">
.home {
  &__welcome {
    margin: 0 0 8px;
    font-size: 20px;
  }
  &__hello {
    margin: 0 0 20px;
    color: var(--el-text-color-secondary);
  }
}
</style>
