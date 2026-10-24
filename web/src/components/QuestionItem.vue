<script setup>
// 问题列表项
import { excerpt, timeAgo } from '../utils'

defineProps({
  q: { type: Object, required: true },
})

function netVotes(q) {
  return q.upvote_count - q.downvote_count
}
</script>

<template>
  <div class="card qitem">
    <div class="stats">
      <div class="stat"><b>{{ netVotes(q) }}</b>赞同</div>
      <div class="stat answers" :class="{ none: q.answer_count === 0 }"><b>{{ q.answer_count }}</b>回答</div>
      <div class="stat views"><b>{{ q.view_count }}</b>浏览</div>
    </div>
    <div class="main">
      <router-link class="title" :to="'/q/' + q.id">{{ q.title }}</router-link>
      <p class="excerpt">{{ excerpt(q.content) }}</p>
      <div class="meta">
        <router-link v-for="t in q.tags" :key="t.id" class="tag" :to="{ path: '/', query: { tag: t.name } }">
          {{ t.name }}
        </router-link>
        <span v-if="q.accepted_answer_id" class="accepted-flag">✓ 已采纳</span>
        <span class="spacer"></span>
        <router-link :to="'/u/' + q.author.username">{{ q.author.username }}</router-link>
        <span>{{ timeAgo(q.created_at) }} 提问</span>
      </div>
    </div>
  </div>
</template>
