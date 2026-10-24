<script setup>
// 管理后台：概览统计 / 用户管理 / 问题管理 / 评论管理
import { onMounted, ref } from 'vue'
import { api, pageParams } from '../api'
import { toast } from '../toast'
import { timeAgo } from '../utils'

const tab = ref('stats')

// ---- 概览 ----
const stats = ref(null)
async function fetchStats() {
  try {
    stats.value = await api('GET', '/admin/stats')
  } catch (e) {
    toast(e.message)
  }
}

const statCards = [
  { key: 'users', label: '用户总数' },
  { key: 'questions', label: '问题总数' },
  { key: 'answers', label: '回答总数' },
  { key: 'comments', label: '评论总数' },
  { key: 'votes', label: '投票总数' },
  { key: 'new_users_today', label: '今日新用户' },
  { key: 'new_questions_today', label: '今日新问题' },
  { key: 'new_answers_today', label: '今日新回答' },
]

// ---- 用户 ----
const users = ref([])
const userTotal = ref(0)
const userPage = ref(1)
const userPages = ref(1)
const userQ = ref('')
const userLoading = ref(false)
const acting = ref(0)

async function fetchUsers(p = 1) {
  userLoading.value = true
  userPage.value = p
  try {
    const params = new URLSearchParams({ q: userQ.value.trim(), ...Object.fromEntries(new URLSearchParams(pageParams(p, 20))) })
    const data = await api('GET', '/admin/users?' + params.toString())
    users.value = data.list
    userTotal.value = data.total
    userPages.value = Math.max(1, Math.ceil(data.total / 20))
  } finally {
    userLoading.value = false
  }
}

async function setStatus(u, status) {
  const verb = status === 'banned' ? '封禁' : '解封'
  if (!confirm(`确定${verb}用户「${u.username}」吗？`)) return
  acting.value = u.id
  try {
    await api('PUT', `/admin/users/${u.id}/status`, { status })
    u.status = status
    toast(`已${verb} ${u.username}`, 'success')
  } catch (e) {
    toast(e.message)
  } finally {
    acting.value = 0
  }
}

// ---- 问题 ----
const questions = ref([])
const qTotal = ref(0)
const qPage = ref(1)
const qPages = ref(1)
const qKeyword = ref('')
const qStatus = ref('all')
const qLoading = ref(false)

async function fetchQuestions(p = 1) {
  qLoading.value = true
  qPage.value = p
  try {
    const params = new URLSearchParams({ q: qKeyword.value.trim(), status: qStatus.value, ...Object.fromEntries(new URLSearchParams(pageParams(p, 20))) })
    const data = await api('GET', '/admin/questions?' + params.toString())
    questions.value = data.list
    qTotal.value = data.total
    qPages.value = Math.max(1, Math.ceil(data.total / 20))
  } finally {
    qLoading.value = false
  }
}

async function deleteQuestion(qu) {
  if (!confirm(`确定删除问题「${qu.title.slice(0, 30)}」吗？`)) return
  try {
    await api('DELETE', '/questions/' + qu.id)
    toast('问题已删除', 'success')
    await fetchQuestions(qPage.value)
    fetchStats()
  } catch (e) {
    toast(e.message)
  }
}

// ---- 评论 ----
const comments = ref([])
const cTotal = ref(0)
const cPage = ref(1)
const cPages = ref(1)
const cKeyword = ref('')
const cLoading = ref(false)

async function fetchComments(p = 1) {
  cLoading.value = true
  cPage.value = p
  try {
    const params = new URLSearchParams({ q: cKeyword.value.trim(), ...Object.fromEntries(new URLSearchParams(pageParams(p, 20))) })
    const data = await api('GET', '/admin/comments?' + params.toString())
    comments.value = data.list
    cTotal.value = data.total
    cPages.value = Math.max(1, Math.ceil(data.total / 20))
  } finally {
    cLoading.value = false
  }
}

async function deleteComment(cm) {
  if (!confirm('确定删除该评论吗？')) return
  try {
    await api('DELETE', '/comments/' + cm.id)
    toast('评论已删除', 'success')
    await fetchComments(cPage.value)
    fetchStats()
  } catch (e) {
    toast(e.message)
  }
}

function switchTab(t) {
  tab.value = t
  if (t === 'stats') fetchStats()
  if (t === 'users') fetchUsers(1)
  if (t === 'questions') fetchQuestions(1)
  if (t === 'comments') fetchComments(1)
}

onMounted(fetchStats)
</script>

<template>
  <div class="page-head">
    <h1>管理后台</h1>
    <div class="tabs">
      <button class="tab" :class="{ active: tab === 'stats' }" @click="switchTab('stats')">概览</button>
      <button class="tab" :class="{ active: tab === 'users' }" @click="switchTab('users')">用户</button>
      <button class="tab" :class="{ active: tab === 'questions' }" @click="switchTab('questions')">问题</button>
      <button class="tab" :class="{ active: tab === 'comments' }" @click="switchTab('comments')">评论</button>
    </div>
  </div>

  <!-- 概览 -->
  <div v-if="tab === 'stats'" class="stat-grid">
    <div v-for="c in statCards" :key="c.key" class="card stat-card">
      <b>{{ stats ? stats[c.key] : '—' }}</b>
      <span>{{ c.label }}</span>
    </div>
  </div>

  <!-- 用户 -->
  <template v-if="tab === 'users'">
    <form class="admin-search" @submit.prevent="fetchUsers(1)">
      <input v-model="userQ" type="search" placeholder="搜索用户名 / 邮箱…" />
      <button class="btn small">搜索</button>
    </form>
    <div v-if="userLoading" class="loading">加载中…</div>
    <div v-else class="card admin-table-wrap">
      <table class="admin-table">
        <thead><tr><th>ID</th><th>用户名</th><th>邮箱</th><th>声望</th><th>提问</th><th>回答</th><th>状态</th><th>注册时间</th><th></th></tr></thead>
        <tbody>
          <tr v-for="u in users" :key="u.id">
            <td>{{ u.id }}</td>
            <td><router-link :to="'/u/' + u.username">{{ u.username }}</router-link></td>
            <td class="muted-td">{{ u.email }}</td>
            <td>{{ u.reputation }}</td>
            <td>{{ u.question_count }}</td>
            <td>{{ u.answer_count }}</td>
            <td>
              <span class="tag" :class="{ banned: u.status === 'banned' }">
                {{ u.status === 'banned' ? '已封禁' : u.role === 'admin' ? '管理员' : '正常' }}
              </span>
            </td>
            <td class="muted-td">{{ timeAgo(u.created_at) }}</td>
            <td>
              <button
                v-if="u.role !== 'admin'"
                class="btn small" :class="u.status === 'banned' ? '' : 'danger'"
                :disabled="acting === u.id"
                @click="setStatus(u, u.status === 'banned' ? 'active' : 'banned')"
              >{{ u.status === 'banned' ? '解封' : '封禁' }}</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-if="userPages > 1" class="pager">
      <button class="btn small" :disabled="userPage <= 1" @click="fetchUsers(userPage - 1)">上一页</button>
      <span>{{ userPage }} / {{ userPages }}</span>
      <button class="btn small" :disabled="userPage >= userPages" @click="fetchUsers(userPage + 1)">下一页</button>
    </div>
  </template>

  <!-- 问题 -->
  <template v-if="tab === 'questions'">
    <form class="admin-search" @submit.prevent="fetchQuestions(1)">
      <input v-model="qKeyword" type="search" placeholder="搜索标题…" />
      <select v-model="qStatus" class="admin-select" @change="fetchQuestions(1)">
        <option value="all">全部状态</option>
        <option value="open">开放中</option>
        <option value="closed">已关闭</option>
        <option value="deleted">已删除</option>
      </select>
      <button class="btn small">搜索</button>
    </form>
    <div v-if="qLoading" class="loading">加载中…</div>
    <div v-else class="card admin-table-wrap">
      <table class="admin-table">
        <thead><tr><th>ID</th><th>标题</th><th>作者</th><th>状态</th><th>回答</th><th>赞同</th><th>浏览</th><th>时间</th><th></th></tr></thead>
        <tbody>
          <tr v-for="qu in questions" :key="qu.id">
            <td>{{ qu.id }}</td>
            <td class="title-td"><router-link :to="'/q/' + qu.id">{{ qu.title }}</router-link></td>
            <td>{{ qu.author_name }}</td>
            <td>
              <span class="tag" :class="{ banned: qu.status === 'deleted' }">
                {{ qu.status === 'open' ? '开放中' : qu.status === 'closed' ? '已关闭' : '已删除' }}
              </span>
            </td>
            <td>{{ qu.answer_count }}</td>
            <td>{{ qu.upvote_count }}</td>
            <td>{{ qu.view_count }}</td>
            <td class="muted-td">{{ timeAgo(qu.created_at) }}</td>
            <td>
              <button v-if="qu.status !== 'deleted'" class="btn small danger" @click="deleteQuestion(qu)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-if="qPages > 1" class="pager">
      <button class="btn small" :disabled="qPage <= 1" @click="fetchQuestions(qPage - 1)">上一页</button>
      <span>{{ qPage }} / {{ qPages }}</span>
      <button class="btn small" :disabled="qPage >= qPages" @click="fetchQuestions(qPage + 1)">下一页</button>
    </div>
  </template>

  <!-- 评论 -->
  <template v-if="tab === 'comments'">
    <form class="admin-search" @submit.prevent="fetchComments(1)">
      <input v-model="cKeyword" type="search" placeholder="搜索评论内容…" />
      <button class="btn small">搜索</button>
    </form>
    <div v-if="cLoading" class="loading">加载中…</div>
    <div v-else class="card admin-table-wrap">
      <table class="admin-table">
        <thead><tr><th>ID</th><th>作者</th><th>内容</th><th>类型</th><th>状态</th><th>时间</th><th></th></tr></thead>
        <tbody>
          <tr v-for="cm in comments" :key="cm.id">
            <td>{{ cm.id }}</td>
            <td>{{ cm.author_name }}</td>
            <td class="title-td">{{ cm.content.slice(0, 50) }}{{ cm.content.length > 50 ? '…' : '' }}</td>
            <td>{{ cm.target_type === 'question' ? '问题' : '回答' }}</td>
            <td>
              <span class="tag" :class="{ banned: cm.status === 'deleted' }">{{ cm.status === 'published' ? '正常' : '已删除' }}</span>
            </td>
            <td class="muted-td">{{ timeAgo(cm.created_at) }}</td>
            <td>
              <button v-if="cm.status === 'published'" class="btn small danger" @click="deleteComment(cm)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-if="cPages > 1" class="pager">
      <button class="btn small" :disabled="cPage <= 1" @click="fetchComments(cPage - 1)">上一页</button>
      <span>{{ cPage }} / {{ cPages }}</span>
      <button class="btn small" :disabled="cPage >= cPages" @click="fetchComments(cPage + 1)">下一页</button>
    </div>
  </template>
</template>
