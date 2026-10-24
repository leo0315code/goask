<script setup>
// 账号设置：修改密码
import { ref } from 'vue'
import { api } from '../api'
import { toast } from '../toast'

const oldPassword = ref('')
const newPassword = ref('')
const confirm = ref('')
const submitting = ref(false)
const error = ref('')

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
    toast('密码修改成功', 'success')
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
    <p class="hint">提示：修改密码后当前登录状态保持有效，下次登录请使用新密码。</p>
  </div>
</template>
