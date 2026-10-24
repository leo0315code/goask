<script setup>
// 首页：左栏问题列表（最新 / 热门 / 未回答 + 标签筛选 + 搜索 + 分页），右栏热门问题与标签
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, pageParams } from '../api'
import QuestionItem from '../components/QuestionItem.vue'
import HotQuestions from '../components/HotQuestions.vue'
import HotTagsCard from '../components/HotTagsCard.vue'

const route = useRoute()
const router = useRouter()

const sorts = [
  { key: 'latest', label: '最新' },
  { key: 'hot', label: '热门' },
  { key: 'unanswered', label: '未回答' },
]

const times = [
  { key: 'all', label: '全部时间' },
  { key: 'day', label: '一天内' },
  { key: 'week', label: '一周内' },
  { key: 'month', label: '一月内' },
  { key: 'year', label: '一年内' },
]

const accepts = [
  { key: 'all', label: '全部' },
  { key: 'yes', label: '已采纳' },
  { key: 'no', label: '未采纳' },
]

const list = ref([])
const total = ref(0)
const loading = ref(true)

const sort = computed(() => route.query.sort || 'latest')
const keyword = computed(() => route.query.q || '')
const tag = computed(() => route.query.tag || '')
const time = computed(() => route.query.time || 'all')
const accepted = computed(() => route.query.accepted || 'all')
const page = computed(() => Number(route.query.page) || 1)
const pageSize = 15
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

async function fetchList() {
  loading.value = true
  try {
    const params = new URLSearchParams({ sort: sort.value, ...Object.fromEntries(new URLSearchParams(pageParams(page.value, pageSize))) })
    if (tag.value) params.set('tag', tag.value)
    if (time.value !== 'all') params.set('time', time.value)
    if (accepted.value !== 'all') params.set('accepted', accepted.value)
    const path = keyword.value ? '/search?q=' + encodeURIComponent(keyword.value) : '/questions'
    const data = await api('GET', path + '?' + params.toString())
    list.value = data.list
    total.value = data.total
  } catch (e) {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function go(change) {
  const query = {}
  if (sort.value !== 'latest') query.sort = sort.value
  if (keyword.value) query.q = keyword.value
  if (tag.value) query.tag = tag.value
  const targetPage = change.page !== undefined ? change.page : page.value
  if (targetPage > 1) query.page = String(targetPage)
  if (change.sort !== undefined) {
    if (change.sort === 'latest') delete query.sort
    else query.sort = change.sort
    delete query.page
  }
  if (change.tag !== undefined) {
    if (change.tag) query.tag = change.tag
    else delete query.tag
    delete query.page
  }
  const currentTime = change.time !== undefined ? change.time : time.value
  if (currentTime !== 'all') query.time = currentTime
  if (change.time !== undefined) delete query.page
  const currentAccepted = change.accepted !== undefined ? change.accepted : accepted.value
  if (currentAccepted !== 'all') query.accepted = currentAccepted
  if (change.accepted !== undefined) delete query.page
  router.push({ path: '/', query })
}

watch(() => route.query, fetchList, { immediate: true })
</script>

<template>
  <div class="layout">
    <div>
      <div class="page-head">
        <h1 v-if="keyword">搜索“{{ keyword }}”<span class="muted-h">（{{ total }} 条结果）</span></h1>
        <h1 v-else-if="tag">标签「{{ tag }}」<span class="muted-h">（{{ total }} 个问题）</span></h1>
        <h1 v-else>全部问题</h1>
        <div v-if="!keyword && !tag" class="tabs">
          <button
            v-for="s in sorts" :key="s.key"
            class="tab" :class="{ active: sort === s.key }"
            @click="go({ sort: s.key })"
          >{{ s.label }}</button>
        </div>
        <router-link v-else class="btn small" to="/">查看全部</router-link>
      </div>

      <div v-if="!keyword" class="filter-row">
        <span class="filter-label">时间</span>
        <button
          v-for="t in times" :key="t.key"
          class="chip" :class="{ active: time === t.key }"
          @click="go({ time: t.key })"
        >{{ t.label }}</button>
      </div>

      <div v-if="!keyword" class="filter-row">
        <span class="filter-label">采纳</span>
        <button
          v-for="a in accepts" :key="a.key"
          class="chip" :class="{ active: accepted === a.key }"
          @click="go({ accepted: a.key })"
        >{{ a.label }}</button>
      </div>

      <div v-if="loading" class="loading">加载中…</div>
      <div v-else-if="list.length === 0" class="card empty">
        暂无问题，<router-link to="/ask">去提第一个问题 →</router-link>
      </div>
      <QuestionItem v-for="q in list" v-else :key="q.id" :q="q" />

      <div v-if="totalPages > 1" class="pager">
        <button class="btn small" :disabled="page <= 1" @click="go({ page: page - 1 })">上一页</button>
        <span>{{ page }} / {{ totalPages }}</span>
        <button class="btn small" :disabled="page >= totalPages" @click="go({ page: page + 1 })">下一页</button>
      </div>
    </div>

    <aside class="sidebar">
      <HotQuestions />
      <HotTagsCard />
    </aside>
  </div>
</template>
