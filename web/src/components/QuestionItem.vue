<script setup>
// 问题列表项
import { excerpt, timeAgo } from '../utils'

const props = defineProps({
  q: { type: Object, required: true },
  keyword: { type: String, default: '' },
});

// 搜索结果关键词高亮：先转义文本与关键词再替换，避免 XSS
function hl(text) {
  const safe = String(text ?? '').replace(/[&<>"']/g, (ch) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[ch]));
  const kw = props.keyword.trim();
  if (!kw) return safe;
  const re = new RegExp(kw.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gi');
  return safe.replace(re, (m) => `<mark>${m}</mark>`);
}
</script>

<template>
  <div class="card qitem">
    <div class="stats">
      <div class="stat answers" :class="{ none: q.answer_count === 0 }"><b>{{ q.answer_count }}</b>回答</div>
      <div class="stat views"><b>{{ q.view_count }}</b>浏览</div>
    </div>
    <div class="main">
      <router-link class="title" :to="'/q/' + q.id">
        <span v-if="keyword" v-html="hl(q.title)"></span>
        <template v-else>{{ q.title }}</template>
      </router-link>
      <p class="excerpt">
        <span v-if="keyword" v-html="hl(excerpt(q.content))"></span>
        <template v-else>{{ excerpt(q.content) }}</template>
      </p>
      <div class="meta">
        <span class="meta-votes">{{ q.upvote_count }} 赞同</span>
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
