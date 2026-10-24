<script setup>
// Markdown 编辑器：工具栏（选区包裹/行前缀/块插入）+ 编辑/预览双模式 + 快捷键
import { computed, nextTick, ref } from 'vue'
import MarkdownContent from './MarkdownContent.vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  minHeight: { type: Number, default: 200 },
  placeholder: { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue'])

const mode = ref('edit') // edit | preview
const el = ref(null)

const styleHeight = computed(() => ({ minHeight: props.minHeight + 'px' }))

function set(v) {
  emit('update:modelValue', v)
}

function onInput(e) {
  set(e.target.value)
}

// 包裹选区（无选区时插入占位文字）
function wrapSelection(before, after, placeholder) {
  const ta = el.value
  if (!ta) return
  const start = ta.selectionStart
  const end = ta.selectionEnd
  const sel = ta.value.slice(start, end) || placeholder
  set(ta.value.slice(0, start) + before + sel + after + ta.value.slice(end))
  nextTick(() => {
    ta.focus()
    ta.setSelectionRange(start + before.length, start + before.length + sel.length)
  })
}

// 给选中行加前缀（再按一次可取消）
function prefixLines(prefix) {
  const ta = el.value
  if (!ta) return
  const start = ta.selectionStart
  const before = ta.value.slice(0, start)
  const sel = ta.value.slice(start, ta.selectionEnd)
  const after = ta.value.slice(ta.selectionEnd)
  const lineStart = before.lastIndexOf('\n') + 1
  const lines = (before.slice(lineStart) + sel).split('\n')
  const has = lines.every((l) => l.startsWith(prefix))
  const replaced = lines.map((l) => (has ? l.slice(prefix.length) : prefix + l)).join('\n')
  set(before.slice(0, lineStart) + replaced + after)
  nextTick(() => {
    ta.focus()
    ta.setSelectionRange(lineStart, lineStart + replaced.length)
  })
}

// 在当前位置插入块级内容
function insertBlock(text) {
  const ta = el.value
  if (!ta) return
  const start = ta.selectionStart
  const before = ta.value.slice(0, start)
  const after = ta.value.slice(ta.selectionEnd)
  const pad = !before ? '' : before.endsWith('\n\n') ? '' : before.endsWith('\n') ? '\n' : '\n\n'
  set(before + pad + text + '\n\n' + after)
  nextTick(() => {
    ta.focus()
    ta.setSelectionRange(start + pad.length + text.length, start + pad.length + text.length)
  })
}

const tools = [
  { key: 'bold', label: 'B', title: '加粗（⌘/Ctrl+B）', cls: 'b' },
  { key: 'italic', label: 'I', title: '斜体（⌘/Ctrl+I）', cls: 'i' },
  { key: 'h2', label: 'H2', title: '标题' },
  { key: 'ul', label: '• 列表', title: '无序列表' },
  { key: 'ol', label: '1. 列表', title: '有序列表' },
  { key: 'quote', label: '❝ 引用', title: '引用' },
  { key: 'code', label: '</>', title: '行内代码' },
  { key: 'codeblock', label: '``` 代码块', title: '代码块' },
  { key: 'link', label: '🔗 链接', title: '链接' },
  { key: 'hr', label: '— 分隔线', title: '分隔线' },
]

function apply(key) {
  switch (key) {
    case 'bold': wrapSelection('**', '**', '加粗文字'); break
    case 'italic': wrapSelection('*', '*', '斜体文字'); break
    case 'h2': prefixLines('## '); break
    case 'ul': prefixLines('- '); break
    case 'ol': prefixLines('1. '); break
    case 'quote': prefixLines('> '); break
    case 'code': wrapSelection('`', '`', 'code'); break
    case 'codeblock': insertBlock('```go\n// 代码\n```'); break
    case 'link': wrapSelection('[', '](https://)', '链接文字'); break
    case 'hr': insertBlock('---'); break
  }
}

function onKeydown(e) {
  if (!(e.metaKey || e.ctrlKey)) return
  if (e.key === 'b') { e.preventDefault(); apply('bold') }
  if (e.key === 'i') { e.preventDefault(); apply('italic') }
}
</script>

<template>
  <div class="md-editor">
    <div class="md-toolbar">
      <button
        v-for="t in tools" :key="t.key"
        type="button" class="md-btn" :class="t.cls"
        :title="t.title" @click="apply(t.key)"
      >{{ t.label }}</button>
      <div class="md-mode">
        <button type="button" :class="{ active: mode === 'edit' }" @click="mode = 'edit'">编辑</button>
        <button type="button" :class="{ active: mode === 'preview' }" @click="mode = 'preview'">预览</button>
      </div>
    </div>

    <textarea
      v-show="mode === 'edit'"
      ref="el" class="md-textarea" :style="styleHeight"
      :value="modelValue" :placeholder="placeholder"
      @input="onInput" @keydown="onKeydown"
    ></textarea>
    <div v-show="mode === 'preview'" class="md-preview" :style="styleHeight">
      <MarkdownContent v-if="modelValue" :content="modelValue" />
      <div v-else class="hint" style="padding: 20px 0; text-align: center">暂无内容</div>
    </div>
  </div>
</template>
