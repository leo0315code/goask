<script setup>
// 账号设置：修改密码
import { ref } from 'vue'
import { api } from '../api'
import { setToken, loadMe } from '../store'
import { toast } from '../toast'

const oldPassword = ref('')
const newPassword = ref('')
const confirm = ref('')
const submitting = ref(false)
const kicking = ref(false)
const error = ref('')

async function kickAll() {
  if (!confirm('确定让所有设备退出登录吗？当前设备会自动换发新 token 保持登录。')) return
  kicking.value = true
  try {
    const d = await api('POST', '/users/me/logout-all')
    setToken(d.token)
    await loadMe()
    toast('其他设备已退出登录', 'success')
  } catch (e) {
    toast(e.message)
  } finally {
    kicking.value = false
  }
}

async function submit() {
  error.value = ''
  if (newPassword.value !== confirm.value) {
    error.value = '两次输入的新密码不一致'
    return
  }
  submitting.value = true
  try {
    await api('PUT', '/users/me/password', {
      old_password: oldPassword.value,
      new_password: newPassword.value,
    })
    setToken(data.token)
    await loadMe()
    toast('密码修改成功，其他设备已退出登录', 'success')
    oldPassword.value = newPassword.value = confirm.value = ''
  } catch (e) {
    error.value = e.message
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="form card">
    <h1>账号设置</h1>
    <label>原密码</label>
    <input v-model="oldPassword" type="password" />
    <label>新密码（至少 8 位）</label>
    <input v-model="newPassword" type="password" />
    <label>确认新密码</label>
    <input v-model="confirm" type="password" @keyup.enter="submit" />
    <div v-if="error" class="err">{{ error }}</div>
    <div class="actions">
      <button class="btn primary" :disabled="submitting" @click="submit">修改密码</button>
    </div>
    <div class="actions">
      <button class="btn danger" :disabled="kicking" @click="kickAll">让所有设备退出登录</button>
    </div>
    <p class="hint">提示：修改密码 / 全端登出后其他设备立即失效，当前设备自动换发新 token。</p>
  </div>
</template>
