<script setup>
// 热门问题卡片（按热度排序取前 5）
import { onMounted, ref } from 'vue'
import { api } from '../api'

const list = ref([])

onMounted(async () => {
  try {
    const data = await api('GET', '/questions?sort=hot&page=1&page_size=5')
    list.value = data.list || []
  } catch {
    list.value = []
  }
})
</script>

<template>
  <div class="card">
    <h3>热门问题</h3>
    <div v-for="q in list" :key="q.id" class="side-item">
      <router-link class="side-title" :to="'/q/' + q.id">{{ q.title }}</router-link>
      <div class="s-meta">{{ q.upvote_count }} 赞同 · {{ q.answer_count }} 回答 · {{ q.view_count }} 浏览</div>
    </div>
    <div v-if="list.length === 0" class="hint">暂无问题</div>
  </div>
</template>
