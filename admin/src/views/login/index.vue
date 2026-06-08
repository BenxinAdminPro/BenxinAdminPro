<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   登录页 — 图形验证码 + 用户名/密码登录 + 错误码友好提示
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-08 14:00:00
  +----------------------------------------------------------------------
-->
<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { User, Lock, Picture as PictureIcon, Moon, Sunny } from '@element-plus/icons-vue'
import { fetchCaptcha, type Captcha } from '@/api/auth'
import { useUserStore } from '@/store/user'
import { isDark, toggleDark } from '@/theme'
import type { AxiosError } from 'axios'
import type { ApiEnvelope } from '@/request/types'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const user = useUserStore()

const formRef = ref<FormInstance>()
const loading = ref(false)
const captcha = ref<Captcha | null>(null)

const form = reactive({
  username: '',
  password: '',
  captcha_code: '',
})

const rules = computed<FormRules>(() => ({
  username: [{ required: true, message: t('login.rule.username'), trigger: 'blur' }],
  password: [{ required: true, message: t('login.rule.password'), trigger: 'blur' }],
  captcha_code: [{ required: true, message: t('login.rule.captcha'), trigger: 'blur' }],
}))

const captchaSrc = computed(() => {
  const img = captcha.value?.image_base64 || ''
  if (!img) return ''
  return img.startsWith('data:') ? img : `data:image/png;base64,${img}`
})

async function loadCaptcha(): Promise<void> {
  try {
    captcha.value = await fetchCaptcha()
    form.captcha_code = ''
  } catch {
    captcha.value = null
  }
}

async function onSubmit(): Promise<void> {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  loading.value = true
  try {
    await user.loginAction({
      username: form.username,
      password: form.password,
      captcha_id: captcha.value?.captcha_id,
      captcha_code: form.captcha_code,
    })
    ElMessage.success(t('login.success'))
    const redirect = (route.query.redirect as string) || '/'
    router.replace(redirect)
  } catch (e) {
    // 后端 message 已是友好文案（凭证错/需验证码/验证码错/锁定/禁用），直接呈现
    const err = e as AxiosError<ApiEnvelope>
    ElMessage.error(err.response?.data?.message || '登录失败，请重试')
    await loadCaptcha() // 失败刷新验证码（防重放）
  } finally {
    loading.value = false
  }
}

onMounted(loadCaptcha)
</script>

<template>
  <div class="login">
    <el-icon class="login__theme" @click="toggleDark()">
      <Moon v-if="!isDark" />
      <Sunny v-else />
    </el-icon>

    <div class="login__card">
      <div class="login__brand">
        <h1 class="login__title">{{ t('login.title') }}</h1>
        <p class="login__subtitle">{{ t('login.subtitle') }}</p>
      </div>

      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        size="large"
        @keyup.enter="onSubmit"
      >
        <el-form-item prop="username">
          <el-input
            v-model="form.username"
            :placeholder="t('login.usernamePlaceholder')"
            :prefix-icon="User"
            clearable
          />
        </el-form-item>

        <el-form-item prop="password">
          <el-input
            v-model="form.password"
            type="password"
            :placeholder="t('login.passwordPlaceholder')"
            :prefix-icon="Lock"
            show-password
          />
        </el-form-item>

        <el-form-item prop="captcha_code">
          <div class="login__captcha">
            <el-input
              v-model="form.captcha_code"
              :placeholder="t('login.captchaPlaceholder')"
              :prefix-icon="PictureIcon"
            />
            <div class="login__captcha-img" :title="t('login.refreshCaptcha')" @click="loadCaptcha">
              <img v-if="captchaSrc" :src="captchaSrc" alt="captcha" />
              <span v-else class="login__captcha-empty">{{ t('login.refreshCaptcha') }}</span>
            </div>
          </div>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" class="login__submit" :loading="loading" @click="onSubmit">
            {{ loading ? t('login.submitting') : t('login.submit') }}
          </el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<style scoped lang="scss">
.login {
  position: relative;
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--el-bg-color-page);

  &__theme {
    position: absolute;
    top: 20px;
    right: 24px;
    font-size: 20px;
    cursor: pointer;
    color: var(--el-text-color-regular);
  }

  &__card {
    width: 380px;
    max-width: calc(100vw - 32px);
    padding: 36px 32px;
    background: var(--el-bg-color);
    border-radius: 12px;
    box-shadow: var(--el-box-shadow-light);
    border: 1px solid var(--el-border-color-lighter);
  }

  &__brand {
    text-align: center;
    margin-bottom: 28px;
  }
  &__title {
    margin: 0;
    font-size: 24px;
    font-weight: 700;
    color: var(--el-color-primary);
  }
  &__subtitle {
    margin: 6px 0 0;
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }

  &__captcha {
    display: flex;
    gap: 12px;
    width: 100%;
  }
  &__captcha-img {
    flex-shrink: 0;
    width: 120px;
    height: 40px;
    border: 1px solid var(--el-border-color);
    border-radius: 4px;
    overflow: hidden;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--el-fill-color-light);
    img {
      width: 100%;
      height: 100%;
      object-fit: contain;
    }
  }
  &__captcha-empty {
    font-size: 11px;
    color: var(--el-text-color-placeholder);
    text-align: center;
    padding: 0 4px;
  }

  &__submit {
    width: 100%;
  }
}
</style>
