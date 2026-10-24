<script setup>
// 相关问题卡片：按问题的第一个标签推荐，无标签时退化为热门问题
import { ref, watch } from 'vue'
import { api } from '../api'

const props = defineProps({
  question: { type: Object, required: true },
})

const list = ref([])
const mode = ref('related')

async function fetchRelated() {
  list.value = []
  const tag = props.question.tags && props.question.tags.length ? props.question.tags[0].name : ''
  mode.value = tag ? 'related' : 'hot'
  const path = tag
    ? `/questions?tag=${encodeURIComponent(tag)}&page=1&page_size=6`
    : '/questions?sort=hot&page=1&page_size=6'
  try {
    const data = await api('GET', path)
    list.value = (data.list || []).filter((q) => q.id !== props.question.id).slice(0, 5)
  } catch {
    list.value = []
  }
}

watch(() => props.question.id, fetchRelated, { immediate: true })
</script>

<template>
  <div class="card">
    <h3>{{ mode === 'related' ? '相关问题' : '热门问题' }}</h3>
    <div v-for="q in list" :key="q.id" class="side-item">
      <router-link class="side-title" :to="'/q/' + q.id">{{ q.title }}</router-link>
      <div class="s-meta">{{ q.upvote_count }} 赞同 · {{ q.answer_count }} 回答</div>
    </div>
    <div v-if="list.length === 0" class="hint">暂无相关问题</div>
  </div>
</template>
