<script setup>
// 热门标签卡片
import { onMounted, ref } from 'vue'
import { api } from '../api'

const tags = ref([])

onMounted(async () => {
  try {
    tags.value = (await api('GET', '/tags?limit=10')) || []
  } catch {
    tags.value = []
  }
})
</script>

<template>
  <div class="card">
    <h3>热门标签</h3>
    <router-link
      v-for="t in tags" :key="t.id"
      class="tag" :to="{ path: '/', query: { tag: t.name } }"
    >{{ t.name }} × {{ t.question_count }}</router-link>
    <div v-if="tags.length === 0" class="hint">暂无标签</div>
  </div>
</template>
