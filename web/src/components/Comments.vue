<script setup>
// 评论（楼中楼）：根评论分页展示、嵌套回复、回复定位输入框
import { nextTick, onMounted, ref, watch } from 'vue'
import { api } from '../api'
import { store } from '../store'
import { toast } from '../toast'
import { timeAgo } from '../utils'
import MarkdownContent from './MarkdownContent.vue'

const props = defineProps({
  targetType: { type: String, required: true }, // question | answer
  targetId: { type: Number, required: true },
})

const list = ref([]) // 根评论（含 replies）
const total = ref(0)
const page = ref(1)
const pages = ref(1)
const content = ref('')
const replyContent = ref('')
const submitting = ref(false)
const inputEl = ref(null)
const showRootForm = ref(false)
const replyAt = ref(null) // { rootID, replyToID, name } 回复定位；null = 根评论

const canManage = (c) => store.me && (store.me.id === c.user_id || store.me.role === 'admin')

async function fetchList(p = 1) {
  const data = await api('GET', `/${props.targetType}s/${props.targetId}/comments?page=${p}&page_size=20`)
  list.value = data.list
  total.value = data.total
  page.value = data.page
  pages.value = Math.max(1, Math.ceil(data.total / 20))
}

// 「添加评论」：根级评论框
async function toggleRootForm() {
  showRootForm.value = !showRootForm.value
  replyAt.value = null
  if (showRootForm.value) {
    await nextTick()
    inputEl.value && inputEl.value.focus()
  }
}

// 「回复」：在该根评论的楼层内展开输入框
async function openReply(root, reply) {
  replyAt.value = {
    rootID: root.id,
    replyToID: reply ? reply.user_id : root.user_id,
    name: reply ? reply.author_name : root.author_name,
  }
  showRootForm.value = false
  await nextTick()
  const ta = document.querySelector(`.reply-form[data-root="${root.id}"] textarea`)
  ta && ta.focus()
}

function cancelReply() {
  replyAt.value = null
  replyContent.value = ''
}

async function doSubmit(parentID, text) {
  submitting.value = true
  try {
    await api('POST', `/${props.targetType}s/${props.targetId}/comments`, {
      content: text,
      parent_id: parentID || undefined,
    })
    toast('评论已发表', 'success')
    await fetchList(replyAt.value ? page.value : 1)
    refreshUnreadSafe()
  } catch (e) {
    toast(e.message)
    throw e
  } finally {
    submitting.value = false
  }
}

function refreshUnreadSafe() { /* 顶栏未读由全局逻辑处理 */ }

async function remove(c) {
  if (!confirm('确定删除该评论吗？')) return
  try {
    await api('DELETE', `/comments/${c.id}`)
    toast('评论已删除', 'success')
    await fetchList(page.value)
  } catch (e) {
    toast(e.message)
  }
}

async function submitRoot() {
  const text = content.value.trim()
  if (!text) return
  try {
    await doSubmit(0, text)
    content.value = ''
  } catch { /* 错误已提示 */ }
}

async function submitReply(root) {
  const text = replyContent.value.trim()
  if (!text) return
  try {
    await doSubmit(replyAt.value.rootID, text)
    replyContent.value = ''
    replyAt.value = null
  } catch { /* 错误已提示 */ }
}

onMounted(() => fetchList(1))
</script>

<template>
  <div class="comments">
    <div class="comments-head">
      <span>{{ total }} 条评论</span>
      <button v-if="store.me" class="add-cmt" @click="toggleRootForm">💬 添加评论</button>
    </div>

    <div v-if="list.length === 0" class="hint">还没有评论，来抢沙发</div>

    <div v-for="c in list" :key="c.id" class="comment">
      <router-link class="avatar xs" :to="'/u/' + c.author_name">{{ c.author_name[0].toUpperCase() }}</router-link>
      <div class="comment-body">
        <div class="comment-head">
          <router-link class="c-author" :to="'/u/' + c.author_name">{{ c.author_name }}</router-link>
          <span v-if="c.author_role === 'admin'" class="tag">管理员</span>
          <span class="c-time">{{ timeAgo(c.created_at) }}</span>
          <button v-if="canManage(c)" class="c-del" @click="remove && remove(c)">删除</button>
        </div>
        <MarkdownContent class="comment-md" :content="c.content" />
        <div class="c-actions">
          <button class="c-reply" @click="openReply(c, null)">回复</button>
        </div>

        <!-- 楼中楼回复 -->
        <div v-for="r in c.replies" :key="r.id" class="comment reply">
          <router-link class="avatar xxs" :to="'/u/' + r.author_name">{{ r.author_name[0].toUpperCase() }}</router-link>
          <div class="comment-body">
            <div class="comment-head">
              <router-link class="c-author" :to="'/u/' + r.author_name">{{ r.author_name }}</router-link>
              <span v-if="r.reply_to_name && r.reply_to_name !== r.author_name" class="reply-tag">回复 @{{ r.reply_to_name }}</span>
              <span v-if="r.author_role === 'admin'" class="tag">管理员</span>
              <span class="c-time">{{ timeAgo(r.created_at) }}</span>
              <button v-if="canManage(r)" class="c-del" @click="remove(r)">删除</button>
            </div>
            <MarkdownContent class="comment-md" :content="r.content" />
            <div class="c-actions">
              <button class="c-reply" @click="openReply(c, r)">回复</button>
            </div>
          </div>
        </div>

        <!-- 楼层内回复输入框 -->
        <div v-if="replyAt && replyAt.rootID === c.id" class="comment-form reply-form" :data-root="c.id">
          <div class="avatar xs">{{ store.me.username[0].toUpperCase() }}</div>
          <div class="cf-main">
            <textarea
              ref="replyInputEl" v-model="replyContent" rows="2" maxlength="1000"
              :placeholder="'回复 @' + replyAt.name + '…（支持 Markdown，⌘/Ctrl + Enter 发送）'"
              @keydown.ctrl.enter.prevent="submitReply(c)"
              @keydown.meta.enter.prevent="submitReply(c)"
            ></textarea>
            <div class="cf-actions">
              <span class="hint">{{ replyContent.length }}/1000</span>
              <button class="btn small" @click="cancelReply">取消</button>
              <button class="btn primary small" :disabled="submitting || !replyContent.trim()" @click="submitReply(c)">回复</button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 根级评论输入框 -->
    <div v-if="store.me && showRootForm" class="comment-form">
      <div class="avatar xs">{{ store.me.username[0].toUpperCase() }}</div>
      <div class="cf-main">
        <textarea
          ref="inputEl" v-model="content" rows="2" maxlength="1000"
          placeholder="添加评论…（支持 Markdown，⌘/Ctrl + Enter 发送）"
          @keydown.ctrl.enter.prevent="submitRoot"
          @keydown.meta.enter.prevent="submitRoot"
        ></textarea>
        <div class="cf-actions">
          <span class="hint">{{ content.length }}/1000</span>
          <button class="btn primary small" :disabled="submitting || !content.trim()" @click="submitRoot">评论</button>
        </div>
      </div>
    </div>
    <div v-else-if="!store.me" class="hint comment-form-hint">
      <router-link to="/login">登录</router-link> 后可评论
    </div>

    <div v-if="pages > 1" class="pager">
      <button class="btn small" :disabled="page <= 1" @click="fetchList(page - 1)">上一页</button>
      <span>{{ page }} / {{ pages }}</span>
      <button class="btn small" :disabled="page >= pages" @click="fetchList(page + 1)">下一页</button>
    </div>
  </div>
</template>
