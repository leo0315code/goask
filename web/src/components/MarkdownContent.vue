<script setup>
// Markdown 渲染组件：marked 解析 + DOMPurify 消毒 + highlight.js 代码高亮
import { computed, nextTick, ref, watch } from 'vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import hljs from 'highlight.js/lib/common'
import 'highlight.js/styles/github-dark.css'

const props = defineProps({
  content: { type: String, default: '' },
})

const el = ref(null)
const html = computed(() =>
  DOMPurify.sanitize(marked.parse(props.content || '', { breaks: true, async: false }))
)

function highlight() {
  if (!el.value) return
  el.value.querySelectorAll('pre code').forEach((block) => {
    try {
      hljs.highlightElement(block)
    } catch {
      /* 忽略高亮失败 */
    }
  })
}

watch(() => props.content, async () => {
  await nextTick()
  highlight()
}, { immediate: true })
</script>

<template>
  <div ref="el" class="markdown" v-html="html"></div>
</template>
