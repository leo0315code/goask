<script setup>
// 通知中心：列表、点击跳转并标记已读、全部已读
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api, pageParams } from '../api'
import { store, refreshUnread } from '../store'
import { toast } from '../toast'
import { timeAgo } from '../utils'

const router = useRouter()
const list = ref([])
const total = ref(0)
const unread = ref(0)
const page = ref(1)
const pageSize = 20
const totalPages = ref(1)
const loading = ref(true)

function notifText(n) {
  switch (n.type) {
    case 'answer': return '回答了你的问题'
    case 'comment_question': return '评论了你的问题'
    case 'comment_answer': return '评论了你的回答'
    case 'comment_reply': return '回复了你的评论'
    case 'accept': return '采纳了你的回答'
    case 'vote_up_question': return '赞了你的问题'
    case 'vote_up_answer': return '赞了你的回答'
    default: return '与你互动'
  }
}

async function fetchList() {
  loading.value = true
  try {
    const data = await api('GET', '/notifications?' + pageParams(page.value, pageSize))
    list.value = data.list
    total.value = data.total
    unread.value = data.unread_count
    totalPages.value = Math.max(1, Math.ceil(data.total / pageSize))
  } finally {
    loading.value = false
  }
}

async function open(n) {
  if (!n.is_read) {
    try { await api('POST', `/notifications/${n.id}/read`) } catch { /* 忽略 */ }
  }
  refreshUnread()
  if (n.question_id) router.push('/q/' + n.question_id)
  else fetchList()
}

async function readAll() {
  try {
    await api('POST', '/notifications/read-all')
    toast('已全部标记为已读', 'success')
    await fetchList()
    refreshUnread()
  } catch (e) {
    toast(e.message)
  }
}

async function goPage(p) {
  page.value = p
  await fetchList()
}

onMounted(fetchList)
</script>

<template>
  <div class="page-head">
    <h1>通知 <span class="muted-h">（{{ unread }} 条未读）</span></h1>
    <button class="btn small" :disabled="unread === 0" @click="readAll">全部已读</button>
  </div>

  <div v-if="loading" class="loading">加载中…</div>
  <div v-else-if="list.length === 0" class="card empty">还没有通知</div>
  <template v-else>
    <div
      v-for="n in list" :key="n.id"
      class="card notif" :class="{ unread: !n.is_read }"
      @click="open(n)"
    >
      <span v-if="!n.is_read" class="dot"></span>
      <div class="notif-body">
        <div class="notif-text">
          <b>{{ n.actor_name || '有人' }}</b> {{ notifText(n) }}
          <span v-if="n.question_title" class="notif-title">《{{ n.question_title }}》</span>
        </div>
        <div class="s-meta">{{ timeAgo(n.created_at) }}</div>
      </div>
    </div>

    <div v-if="totalPages > 1" class="pager">
      <button class="btn small" :disabled="page <= 1" @click="goPage(page - 1)">上一页</button>
      <span>{{ page }} / {{ totalPages }}</span>
      <button class="btn small" :disabled="page >= totalPages" @click="goPage(page + 1)">下一页</button>
    </div>
  </template>
</template>
