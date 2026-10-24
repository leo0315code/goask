<script setup>
// 提问者信息卡片
import { onMounted, ref } from 'vue'
import { api } from '../api'

const props = defineProps({
  username: { type: String, required: true },
})

const user = ref(null)

onMounted(async () => {
  try {
    user.value = await api('GET', '/users/' + encodeURIComponent(props.username))
  } catch {
    user.value = null
  }
})
</script>

<template>
  <div v-if="user" class="card">
    <h3>提问者</h3>
    <div class="author-mini">
      <router-link class="avatar small" :to="'/u/' + user.username">{{ user.username[0].toUpperCase() }}</router-link>
      <div>
        <router-link class="author-name" :to="'/u/' + user.username">{{ user.username }}</router-link>
        <div class="s-meta">声望 {{ user.reputation }} · 提问 {{ user.question_count }} · 回答 {{ user.answer_count }}</div>
      </div>
    </div>
  </div>
</template>
