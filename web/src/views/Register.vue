<script setup>
// 注册页：算术验证码 + 可选邀请码
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api'
import { setToken, loadMe } from '../store'
import { toast } from '../toast'

const route = useRoute()
const router = useRouter()

const username = ref('')
const email = ref('')
const password = ref('')
const captchaCode = ref('')
const inviteCode = ref('')
const captcha = ref({ captcha_id: '', svg: '', invite_required: false, register_enabled: true })
const submitting = ref(false)
const error = ref('')

async function refreshCaptcha() {
  try {
    const data = await api('GET', '/auth/captcha')
    captcha.value = data
  } catch {
    captcha.value = { captcha_id: '', svg: '', invite_required: false, register_enabled: true }
  }
}

onMounted(refreshCaptcha)

async function submit() {
  error.value = ''
  submitting.value = true
  try {
    const data = await api('POST', '/auth/register', {
      username: username.value,
      email: email.value,
      password: password.value,
      captcha_id: captcha.value.captcha_id,
      captcha_code: captchaCode.value,
      invite_code: inviteCode.value,
    })
    setToken(data.token)
    await loadMe()
    toast('注册成功，已自动登录', 'success')
    router.push(route.query.redirect || '/')
  } catch (e) {
    error.value = e.message
    captchaCode.value = ''
    refreshCaptcha() // 一次一码，失败后换新题
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="form card">
    <template v-if="captcha.register_enabled">
      <h1>注册</h1>
      <label>用户名（3-32 位字母、数字或下划线）</label>
      <input v-model="username" maxlength="32" />
      <label>邮箱</label>
      <input v-model="email" type="email" />
      <label>密码（至少 8 位）</label>
      <input v-model="password" type="password" />
      <label>验证码</label>
      <div class="captcha-row">
        <div class="captcha-img" v-html="captcha.svg" title="点击换一题" @click="refreshCaptcha"></div>
        <input v-model="captchaCode" inputmode="numeric" placeholder="计算结果" @keyup.enter="submit" />
      </div>
      <template v-if="captcha.invite_required">
        <label>邀请码</label>
        <input v-model="inviteCode" placeholder="本站开启了邀请注册" />
      </template>
      <div v-if="error" class="err">{{ error }}</div>
      <div class="actions">
        <button class="btn primary" :disabled="submitting" @click="submit">注册</button>
      </div>
      <p class="hint">
        已有账号？<router-link :to="{ path: '/login', query: route.query }">直接登录</router-link>
      </p>
    </template>
    <template v-else>
      <h1>注册</h1>
      <p class="empty">当前站点已关闭注册，请联系管理员。</p>
      <p class="hint">已有账号？<router-link :to="{ path: '/login', query: route.query }">直接登录</router-link></p>
    </template>
  </div>
</template>
