<script setup>
// 个人主页：资料 + 提问 / 回答 / 收藏（收藏仅本人可见）
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api'
import { store } from '../store'
import QuestionItem from '../components/QuestionItem.vue'
import HotQuestions from '../components/HotQuestions.vue'
import HotTagsCard from '../components/HotTagsCard.vue'
import { excerpt, timeAgo } from '../utils'

const route = useRoute()
const username = route.params.username
const isMe = computed(() => store.me && store.me.username === username)

const tab = ref('questions')
const user = ref(null)
const loading = ref(true)
const notFound = ref(false)

const questions = ref([])
const qTotal = ref(0)
const answers = ref([])
const aTotal = ref(0)
const favorites = ref([])
const fTotal = ref(0)
const listLoading = ref(false)

async function fetchTab() {
  listLoading.value = true
  try {
    if (tab.value === 'questions') {
      const d = await api('GET', `/users/${encodeURIComponent(username)}/questions?page=1&page_size=20`)
      questions.value = d.list; qTotal.value = d.total
    } else if (tab.value === 'answers') {
      const d = await api('GET', `/users/${encodeURIComponent(username)}/answers?page=1&page_size=20`)
      answers.value = d.list; aTotal.value = d.total
    } else {
      const d = await api('GET', '/me/favorites?page=1&page_size=20')
      favorites.value = d.list; fTotal.value = d.total
    }
  } finally {
    listLoading.value = false
  }
}

function switchTab(t) {
  tab.value = t
  fetchTab()
}

onMounted(async () => {
  try {
    user.value = await api('GET', '/users/' + encodeURIComponent(username))
  } catch {
    notFound.value = true
    loading.value = false
    return
  }
  // 预取各 Tab 计数（page_size=1 只取 total）
  api('GET', `/users/${encodeURIComponent(username)}/answers?page=1&page_size=1`)
    .then((d) => { aTotal.value = d.total }).catch(() => {})
  if (isMe.value) {
    api('GET', '/me/favorites?page=1&page_size=1')
      .then((d) => { fTotal.value = d.total }).catch(() => {})
  }
  if (isMe.value) tab.value = 'questions'
  await fetchTab()
  loading.value = false
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

      <div class="page-head">
        <div class="tabs">
          <button class="tab" :class="{ active: tab === 'questions' }" @click="switchTab('questions')">提问 {{ qTotal }}</button>
          <button class="tab" :class="{ active: tab === 'answers' }" @click="switchTab('answers')">回答 {{ aTotal }}</button>
          <button v-if="isMe" class="tab" :class="{ active: tab === 'favorites' }" @click="switchTab('favorites')">收藏 {{ fTotal }}</button>
        </div>
      </div>

      <div v-if="listLoading" class="loading">加载中…</div>

      <template v-else-if="tab === 'questions'">
        <div v-if="questions.length === 0" class="card empty">还没有提问</div>
        <QuestionItem v-for="q in questions" :key="q.id" :q="q" />
      </template>

      <template v-else-if="tab === 'answers'">
        <div v-if="answers.length === 0" class="card empty">还没有回答</div>
        <div v-for="a in answers" :key="a.id" class="card answer-item">
          <router-link class="title" :to="'/q/' + a.question_id">{{ a.question_title }}</router-link>
          <p class="excerpt">{{ excerpt(a.content) }}</p>
          <div class="meta">
            <span v-if="a.is_accepted" class="accepted-flag">✓ 已采纳</span>
            <span class="meta-votes">{{ a.upvote_count }} 赞同</span>
            <span class="spacer"></span>
            <span>{{ timeAgo(a.created_at) }} 回答</span>
          </div>
        </div>
      </template>

      <template v-else>
        <div v-if="favorites.length === 0" class="card empty">还没有收藏</div>
        <div v-for="f in favorites" :key="f.question_id" class="card answer-item">
          <router-link class="title" :to="'/q/' + f.question_id">{{ f.title }}</router-link>
          <div class="meta">
            <span class="meta-votes">{{ f.upvote_count }} 赞同</span>
            <span>{{ f.answer_count }} 回答</span>
            <span>{{ f.view_count }} 浏览</span>
            <span class="spacer"></span>
            <span class="muted-h">收藏于 {{ timeAgo(f.favorited_at) }}</span>
          </div>
        </div>
      </template>
    </div>

    <aside class="sidebar">
      <HotQuestions />
      <HotTagsCard />
    </aside>
  </div>
</template>
