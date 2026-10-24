<script setup>
// 投票组件：赞同 / 反对，可切换、可撤销（点击已激活方向即撤销）
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
  <div class="vote">
    <button :class="{ active: my === 1 }" title="赞同" @click="vote(1)">▲</button>
    <span class="count">{{ up - down }}</span>
    <button :class="{ active: my === -1 }" title="反对" @click="vote(-1)">▼</button>
  </div>
</template>
