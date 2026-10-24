<script setup>
// 提问页：左侧表单（字数统计、标签实时预览、Markdown），右侧提问建议
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import { toast } from '../toast'
import { parseTags } from '../utils'

const router = useRouter()

const title = ref('')
const content = ref('')
const tags = ref('')
const submitting = ref(false)

const previewTags = computed(() => parseTags(tags.value).slice(0, 5))
const titleInvalid = computed(() => title.value.length > 0 && title.value.length < 10)

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
      tags: parseTags(tags.value),
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
      <textarea
        v-model="content" class="f-input content-input"
        placeholder="描述你遇到的问题、已尝试的方案、相关环境信息…&#10;&#10;提示：用 ``` 可以插入代码块"
      ></textarea>

      <label class="f-label">
        标签
        <span class="f-hint">可选，最多 5 个，逗号分隔</span>
      </label>
      <input v-model="tags" class="f-input" placeholder="go, mysql" />
      <div v-if="previewTags.length" class="tag-preview">
        <span v-for="t in previewTags" :key="t" class="tag">{{ t }}</span>
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
