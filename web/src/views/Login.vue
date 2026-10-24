<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api'
import { setToken } from '../store'
import { loadMe } from '../store'
import { toast } from '../toast'

const route = useRoute()
const router = useRouter()

const account = ref('')
const password = ref('')
const submitting = ref(false)
const error = ref('')

async function submit() {
  error.value = ''
  submitting.value = true
  try {
    const data = await api('POST', '/auth/login', {
      account: account.value,
      password: password.value,
    })
    setToken(data.token)
    await loadMe()
    toast('登录成功，欢迎回来～', 'success')
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
    <h1>登录</h1>
    <label>用户名或邮箱</label>
    <input v-model="account" @keyup.enter="submit" />
    <label>密码</label>
    <input v-model="password" type="password" @keyup.enter="submit" />
    <div v-if="error" class="err">{{ error }}</div>
    <div class="actions">
      <button class="btn primary" :disabled="submitting" @click="submit">登录</button>
    </div>
    <p class="hint">
      还没有账号？<router-link :to="{ path: '/register', query: route.query }">立即注册</router-link>
    </p>
  </div>
</template>
