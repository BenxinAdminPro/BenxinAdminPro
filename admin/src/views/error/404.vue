<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   404 页 — 未匹配路由兜底
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-08 14:00:00
  +----------------------------------------------------------------------
-->
<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const { t } = useI18n()
</script>

<template>
  <div class="not-found flex-center">
    <div class="not-found__inner">
      <div class="not-found__code">404</div>
      <p class="not-found__title">{{ t('error.notFound') }}</p>
      <p class="not-found__desc">{{ t('error.notFoundDesc') }}</p>
      <el-button type="primary" @click="router.replace('/')">{{ t('common.backHome') }}</el-button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.not-found {
  height: 100vh;
  background: var(--el-bg-color-page);
  &__inner {
    text-align: center;
  }
  &__code {
    font-size: 96px;
    font-weight: 800;
    color: var(--el-color-primary);
    line-height: 1;
  }
  &__title {
    font-size: 20px;
    margin: 12px 0 4px;
  }
  &__desc {
    color: var(--el-text-color-secondary);
    margin: 0 0 24px;
  }
}
</style>
