<script setup>
// 提问页：标签输入器（空格/回车/逗号成胶囊，× 删除，粘贴批量添加）+ 右侧提问建议
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import { toast } from '../toast'
import MarkdownEditor from '../components/MarkdownEditor.vue'

const router = useRouter()

const title = ref('')
const content = ref('')
const tagList = ref([])
const tagInput = ref('')
const tagInputEl = ref(null)
const submitting = ref(false)

const titleInvalid = computed(() => title.value.length > 0 && title.value.length < 10)

function commitTag() {
  const t = tagInput.value.trim().toLowerCase()
  tagInput.value = ''
  if (!t) return
  if (tagList.value.length >= 5) {
    toast('标签最多 5 个')
    return
  }
  if (t.length > 32) {
    toast('单个标签最多 32 字符')
    return
  }
  if (tagList.value.includes(t)) {
    toast(`标签「${t}」已存在`)
    return
  }
  tagList.value.push(t)
}

function onTagKeydown(e) {
  if (e.key === ' ' || e.key === 'Enter' || e.key === ',' || e.key === '，') {
    // 有内容则成胶囊；无内容时空格/回车不产生字符
    if (tagInput.value.trim() || !tagInput.value) e.preventDefault()
    commitTag()
  } else if (e.key === 'Backspace' && !tagInput.value) {
    tagList.value.pop()
  }
}

// 粘贴含分隔符的文本时批量添加
function onTagPaste(e) {
  const text = (e.clipboardData || window.clipboardData).getData('text') || ''
  if (!/[,，\s]/.test(text)) return
  e.preventDefault()
  for (const t of text.split(/[,，\s]+/)) {
    tagInput.value = t
    commitTag()
  }
}

function removeTag(i) {
  tagList.value.splice(i, 1)
}

function focusTags() {
  tagInputEl.value && tagInputEl.value.focus()
}

async function submit() {
  if (!title.value.trim() || !content.value.trim()) {
    toast('标题和正文不能为空')
    return
  }
  if (titleInvalid.value) {
    toast('标题至少 10 个字符')
    return
  }
  submitting.value = true
  try {
    const q = await api('POST', '/questions', {
      title: title.value,
      content: content.value,
      tags: tagList.value,
    })
    toast('问题发布成功', 'success')
    router.push('/q/' + q.id)
  } catch (e) {
    toast(e.message)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="ask-wrap">
    <div class="ask-main card">
      <h1>提出你的问题</h1>
      <p class="ask-sub">描述清楚的问题更容易得到高质量回答，正文支持 Markdown</p>

      <label class="f-label">
        标题
        <span class="f-count" :class="{ over: titleInvalid }">{{ title.length }}/150</span>
      </label>
      <input
        v-model="title" maxlength="150" class="f-input title-input"
        placeholder="例如：Go 语言如何连接 MySQL 并配置连接池？"
      />
      <div v-if="titleInvalid" class="f-err-hint">标题至少 10 个字符，建议一句话概括核心问题</div>

      <label class="f-label">
        正文
        <span class="f-markdown">支持 Markdown</span>
      </label>
      <MarkdownEditor v-model="content" :min-height="240"
        placeholder="描述你遇到的问题、已尝试的方案、相关环境信息…" />

      <label class="f-label">
        标签
        <span class="f-hint">最多 5 个，空格 / 回车 / 逗号添加</span>
      </label>
      <div class="tag-input-box" @click="focusTags">
        <span v-for="(t, i) in tagList" :key="t" class="tag-pill">
          {{ t }}
          <button class="pill-x" title="移除" @click.stop="removeTag(i)">×</button>
        </span>
        <input
          ref="tagInputEl" v-model="tagInput" class="tag-input-el"
          :placeholder="tagList.length ? '' : '输入后按空格添加，如 go'"
          maxlength="32"
          @keydown="onTagKeydown"
          @paste="onTagPaste"
        />
      </div>

      <div class="actions">
        <button class="btn primary" :disabled="submitting" @click="submit">发布问题</button>
        <router-link to="/" class="btn">取消</router-link>
      </div>
    </div>

    <aside class="ask-side card">
      <h3>提问建议</h3>
      <ul>
        <li>标题用一句话概括核心问题</li>
        <li>正文包含环境信息、期望结果与实际结果</li>
        <li>用代码块贴出关键代码，删掉敏感信息</li>
        <li>发布前先搜索一下，可能已有人问过</li>
      </ul>
    </aside>
  </div>
</template>
