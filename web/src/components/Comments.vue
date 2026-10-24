<script setup>
// 评论列表 + 发表评论（自动拉取、发表后刷新）
import { onMounted, ref } from 'vue'
import { api } from '../api'
import { store } from '../store'
import { toast } from '../toast'
import { timeAgo } from '../utils'

const props = defineProps({
  targetType: { type: String, required: true }, // question | answer
  targetId: { type: Number, required: true },
})

const list = ref([])
const total = ref(0)
const content = ref('')
const submitting = ref(false)

async function fetchList() {
  const data = await api('GET', `/${props.targetType}s/${props.targetId}/comments?page=1&page_size=100`)
  list.value = data.list
  total.value = data.total
}

async function submit() {
  const text = content.value.trim()
  if (!text) return
  submitting.value = true
  try {
    await api('POST', `/${props.targetType}s/${props.targetId}/comments`, { content: text })
    content.value = ''
    toast('评论已发表', 'success')
    await fetchList()
  } catch (e) {
    toast(e.message)
  } finally {
    submitting.value = false
  }
}

onMounted(fetchList)
</script>

<template>
  <div class="comments">
    <div v-for="c in list" :key="c.id" class="comment">
      <router-link class="c-author" :to="'/u/' + c.author.username">{{ c.author.username }}</router-link>
      <span>{{ c.content }}</span>
      <span class="c-time">{{ timeAgo(c.created_at) }}</span>
    </div>
    <div v-if="store.me" class="comment-form">
      <input v-model="content" maxlength="1000" placeholder="添加评论…" @keyup.enter="submit" />
      <button class="btn small" :disabled="submitting" @click="submit">评论</button>
    </div>
    <div v-else class="hint">登录后可评论</div>
  </div>
</template>
