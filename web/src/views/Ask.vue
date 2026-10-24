<script setup>
// 提问页
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import { toast } from '../toast'
import { parseTags } from '../utils'

const router = useRouter()

const title = ref('')
const content = ref('')
const tags = ref('')
const submitting = ref(false)

async function submit() {
  if (!title.value.trim() || !content.value.trim()) {
    toast('标题和正文不能为空')
    return
  }
  submitting.value = true
  try {
    const q = await api('POST', '/questions', {
      title: title.value,
      content: content.value,
      tags: parseTags(tags.value),
    })
    toast('问题发布成功', 'success')
    router.push('/q/' + q.id)
  } catch (e) {
    toast(e.message)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="form card">
    <h1>提出你的问题</h1>
    <label>标题（10-150 字符，把问题说清楚）</label>
    <input v-model="title" maxlength="150" placeholder="例如：Go 语言如何连接 MySQL 并配置连接池？" />
    <label>正文</label>
    <textarea v-model="content" placeholder="描述你遇到的问题、已尝试的方案、相关环境信息…"></textarea>
    <label>标签（可选，最多 5 个，逗号分隔）</label>
    <input v-model="tags" placeholder="go, mysql" />
    <div class="actions">
      <button class="btn primary" :disabled="submitting" @click="submit">发布问题</button>
      <router-link to="/" class="btn">取消</router-link>
    </div>
  </div>
</template>
