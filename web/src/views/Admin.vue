<script setup>
// 管理后台：用户列表、搜索、封禁/解封
import { onMounted, ref } from 'vue'
import { api, pageParams } from '../api'
import { toast } from '../toast'
import { timeAgo } from '../utils'

const q = ref('')
const list = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const totalPages = ref(1)
const loading = ref(true)
const acting = ref(0)

async function fetchList(p = 1) {
  loading.value = true
  page.value = p
  try {
    const params = new URLSearchParams({ q: q.value.trim(), ...Object.fromEntries(new URLSearchParams(pageParams(p, pageSize))) })
    const data = await api('GET', '/admin/users?' + params.toString())
    list.value = data.list
    total.value = data.total
    totalPages.value = Math.max(1, Math.ceil(data.total / pageSize))
  } finally {
    loading.value = false
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

onMounted(() => fetchList(1))
</script>

<template>
  <div class="page-head">
    <h1>用户管理 <span class="muted-h">（{{ total }} 个用户）</span></h1>
    <form class="admin-search" @submit.prevent="fetchList(1)">
      <input v-model="q" type="search" placeholder="搜索用户名 / 邮箱…" />
      <button class="btn small">搜索</button>
    </form>
  </div>

  <div v-if="loading" class="loading">加载中…</div>
  <div v-else-if="list.length === 0" class="card empty">没有匹配的用户</div>
  <div v-else class="card admin-table-wrap">
    <table class="admin-table">
      <thead>
        <tr><th>ID</th><th>用户名</th><th>邮箱</th><th>声望</th><th>提问</th><th>回答</th><th>状态</th><th>注册时间</th><th></th></tr>
      </thead>
      <tbody>
        <tr v-for="u in list" :key="u.id">
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

  <div v-if="totalPages > 1" class="pager">
    <button class="btn small" :disabled="page <= 1" @click="fetchList(page - 1)">上一页</button>
    <span>{{ page }} / {{ totalPages }}</span>
    <button class="btn small" :disabled="page >= totalPages" @click="fetchList(page + 1)">下一页</button>
  </div>
</template>
