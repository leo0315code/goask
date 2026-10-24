<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api'
import { setToken, loadMe } from '../store'
import { toast } from '../toast'

const route = useRoute()
const router = useRouter()

const username = ref('')
const email = ref('')
const password = ref('')
const submitting = ref(false)
const error = ref('')

async function submit() {
  error.value = ''
  submitting.value = true
  try {
    const data = await api('POST', '/auth/register', {
      username: username.value,
      email: email.value,
      password: password.value,
    })
    setToken(data.token)
    await loadMe()
    toast('注册成功，已自动登录', 'success')
    router.push(route.query.redirect || '/')
  } catch (e) {
    error.value = e.message
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="form card">
    <h1>注册</h1>
    <label>用户名（3-32 位字母、数字或下划线）</label>
    <input v-model="username" maxlength="32" />
    <label>邮箱</label>
    <input v-model="email" type="email" />
    <label>密码（至少 8 位）</label>
    <input v-model="password" type="password" @keyup.enter="submit" />
    <div v-if="error" class="err">{{ error }}</div>
    <div class="actions">
      <button class="btn primary" :disabled="submitting" @click="submit">注册</button>
    </div>
    <p class="hint">
      已有账号？<router-link :to="{ path: '/login', query: route.query }">直接登录</router-link>
    </p>
  </div>
</template>
