<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   内容区 — router-view + 过渡 + keep-alive 容器
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-08 14:00:00
  +----------------------------------------------------------------------
-->
<script setup lang="ts"></script>

<template>
  <section class="app-main">
    <router-view v-slot="{ Component }">
      <transition name="fade-slide" mode="out-in">
        <component :is="Component" />
      </transition>
    </router-view>
  </section>
</template>

<style scoped lang="scss">
.app-main {
  height: calc(100vh - 56px);
  overflow-y: auto;
  padding: 16px;
  box-sizing: border-box;
  background: var(--el-bg-color-page);
}

.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: all 0.2s ease;
}
.fade-slide-enter-from {
  opacity: 0;
  transform: translateX(12px);
}
.fade-slide-leave-to {
  opacity: 0;
  transform: translateX(-12px);
}
</style>
