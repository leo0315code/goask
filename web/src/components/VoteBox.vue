<script setup>
// 投票组件：「赞同 N」+「反对」药丸按钮（可切换、可撤销）
import { ref, watch } from 'vue'
import { api } from '../api'
import { toast } from '../toast'

const props = defineProps({
  targetType: { type: String, required: true },
  targetId: { type: Number, required: true },
  upvotes: { type: Number, default: 0 },
  downvotes: { type: Number, default: 0 },
  value: { type: Number, default: 0 }, // 当前用户投票值：1 / -1 / 0
})

const up = ref(props.upvotes)
const down = ref(props.downvotes)
const my = ref(props.value)

watch(
  () => [props.upvotes, props.downvotes, props.value],
  ([u, d, v]) => {
    up.value = u
    down.value = d
    my.value = v
  }
)

async function vote(v) {
  try {
    const state = await api('POST', '/votes', {
      target_type: props.targetType,
      target_id: props.targetId,
      value: v,
    })
    up.value = state.upvote_count
    down.value = state.downvote_count
    my.value = state.value
  } catch (e) {
    toast(e.message)
  }
}
</script>

<template>
  <div class="vote-row">
    <button class="vote-pill" :class="{ active: my === 1 }" title="赞同" @click="vote(1)">
      <svg viewBox="0 0 24 24"><path d="M1 21h4V9H1v12zm22-11c0-1.1-.9-2-2-2h-6.31l.95-4.57.03-.32c0-.41-.17-.79-.44-1.06L14.17 1 7.59 7.59C7.22 7.95 7 8.45 7 9v10c0 1.1.9 2 2 2h9c.83 0 1.54-.5 1.84-1.22l3.02-7.05c.09-.23.14-.47.14-.73v-2z"/></svg>
      <span>赞同</span><b class="n">{{ up }}</b>
    </button>
    <button class="vote-pill icon-only" :class="{ active: my === -1 }" title="反对" @click="vote(-1)">
      <svg viewBox="0 0 24 24"><path d="M15 3H6c-.83 0-1.54.5-1.84 1.22l-3.02 7.05c-.09.23-.14.47-.14.73v2c0 1.1.9 2 2 2h6.31l-.95 4.57-.03.32c0 .41.17.79.44 1.06L9.83 23l6.59-6.59c.36-.36.58-.86.58-1.41V5c0-1.1-.9-2-2-2z"/></svg>
      <b v-if="my === -1 || down > 0" class="n">{{ down }}</b>
    </button>
  </div>
</template>
