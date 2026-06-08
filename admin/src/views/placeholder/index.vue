<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   占位页 — 动态路由组件未实现时的友好降级（不崩）
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-08 16:00:00
  +----------------------------------------------------------------------
-->
<script setup lang="ts">
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
const route = useRoute()
const { t } = useI18n()
</script>

<template>
  <el-card shadow="never">
    <el-empty :description="`${(route.meta.title as string) || route.path} · ${t('error.building')}`" />
  </el-card>
</template>
