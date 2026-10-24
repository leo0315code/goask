<script setup>
// 个人主页：资料 + 该用户的提问列表
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api'
import QuestionItem from '../components/QuestionItem.vue'
import HotQuestions from '../components/HotQuestions.vue'
import HotTagsCard from '../components/HotTagsCard.vue'

const route = useRoute()
const username = route.params.username

const user = ref(null)
const questions = ref([])
const total = ref(0)
const loading = ref(true)
const notFound = ref(false)

onMounted(async () => {
  try {
    user.value = await api('GET', '/users/' + encodeURIComponent(username))
    const data = await api('GET', `/users/${encodeURIComponent(username)}/questions?page=1&page_size=20`)
    questions.value = data.list
    total.value = data.total
  } catch {
    notFound.value = true
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div v-if="loading" class="loading">加载中…</div>
  <div v-else-if="notFound" class="card empty">用户不存在</div>
  <div v-else class="layout">
    <div>
    <div class="card profile-head">
      <div class="avatar">{{ user.username[0].toUpperCase() }}</div>
      <div>
        <h1>{{ user.username }}
          <span v-if="user.role === 'admin'" class="tag">管理员</span>
        </h1>
        <p class="bio">{{ user.bio || '这个人很懒，什么都没写～' }}</p>
        <div class="profile-stats">
          <span>声望 <b>{{ user.reputation }}</b></span>
          <span>提问 <b>{{ user.question_count }}</b></span>
          <span>回答 <b>{{ user.answer_count }}</b></span>
          <span>加入于 <b>{{ new Date(user.created_at).toLocaleDateString('zh-CN') }}</b></span>
        </div>
      </div>
    </div>

    <div class="section-title">{{ total }} 个提问</div>
    <div v-if="questions.length === 0" class="card empty">还没有提问</div>
    <QuestionItem v-for="q in questions" :key="q.id" :q="q" />
    </div>

    <aside class="sidebar">
      <HotQuestions />
      <HotTagsCard />
    </aside>
  </div>
</template>
