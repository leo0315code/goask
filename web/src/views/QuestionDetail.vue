<script setup>
// 问题详情：投票、评论、回答、采纳、编辑 / 删除
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api'
import { store } from '../store'
import { toast } from '../toast'
import { parseTags, timeAgo } from '../utils'
import VoteBox from '../components/VoteBox.vue'
import Comments from '../components/Comments.vue'
import AuthorCard from '../components/AuthorCard.vue'
import RelatedQuestions from '../components/RelatedQuestions.vue'
import HotTagsCard from '../components/HotTagsCard.vue'

const route = useRoute()
const router = useRouter()
const id = Number(route.params.id)

const q = ref(null)
const answers = ref([])
const viewerVotes = ref({}) // answerId -> 我的投票值
const loading = ref(true)
const notFound = ref(false)

// 编辑态
const editingQuestion = ref(false)
const editQ = ref({ title: '', content: '', tags: '' })
const editingAnswerId = ref(0)
const editAContent = ref('')

const answerForm = ref('')
const answering = ref(false)

const isAuthor = computed(() => store.me && q.value && store.me.id === q.value.user_id)
const canManage = computed(() => isAuthor.value || (store.me && store.me.role === 'admin'))
const canAnswer = computed(
  () =>
    store.me && q.value && q.value.status === 'open' &&
    !q.value.accepted_answer_id && store.me.id !== q.value.user_id
)

async function fetchAll() {
  loading.value = true
  try {
    q.value = await api('GET', `/questions/${id}`)
  } catch (e) {
    notFound.value = true
    loading.value = false
    return
  }
  try {
    const ans = await api('GET', `/questions/${id}/answers?page=1&page_size=100&sort=votes`)
    answers.value = ans.list || []
    if (store.me) {
      const results = await Promise.allSettled(
        answers.value.map((a) => api('GET', `/votes?target_type=answer&target_id=${a.id}`))
      )
      answers.value.forEach((a, i) => {
        if (results[i].status === 'fulfilled') viewerVotes.value[a.id] = results[i].value.value
      })
    }
  } catch {
    answers.value = []
  }
  loading.value = false
}

async function submitAnswer() {
  const content = answerForm.value.trim()
  if (!content) return
  answering.value = true
  try {
    await api('POST', `/questions/${id}/answers`, { content })
    answerForm.value = ''
    toast('回答已发表', 'success')
    await fetchAll()
  } catch (e) {
    toast(e.message)
  } finally {
    answering.value = false
  }
}

async function accept(a) {
  try {
    await api('POST', `/questions/${id}/accept`, { answer_id: a.id })
    toast('已采纳该回答', 'success')
    await fetchAll()
  } catch (e) {
    toast(e.message)
  }
}

function startEditQuestion() {
  editQ.value = {
    title: q.value.title,
    content: q.value.content,
    tags: (q.value.tags || []).map((t) => t.name).join(','),
  }
  editingQuestion.value = true
}

async function saveQuestion() {
  try {
    await api('PUT', `/questions/${id}`, {
      title: editQ.value.title,
      content: editQ.value.content,
      tags: parseTags(editQ.value.tags),
    })
    editingQuestion.value = false
    toast('问题已更新', 'success')
    await fetchAll()
  } catch (e) {
    toast(e.message)
  }
}

async function removeQuestion() {
  if (!confirm('确定删除该问题吗？')) return
  try {
    await api('DELETE', `/questions/${id}`)
    toast('问题已删除', 'success')
    router.push('/')
  } catch (e) {
    toast(e.message)
  }
}

function startEditAnswer(a) {
  editingAnswerId.value = a.id
  editAContent.value = a.content
}

async function saveAnswer(a) {
  try {
    await api('PUT', `/answers/${a.id}`, { content: editAContent.value })
    editingAnswerId.value = 0
    toast('回答已更新', 'success')
    await fetchAll()
  } catch (e) {
    toast(e.message)
  }
}

async function removeAnswer(a) {
  if (!confirm('确定删除该回答吗？')) return
  try {
    await api('DELETE', `/answers/${a.id}`)
    toast('回答已删除', 'success')
    await fetchAll()
  } catch (e) {
    toast(e.message)
  }
}

function closeQuestion() {
  if (!confirm('确定关闭该问题吗？关闭后不能再回答。')) return
  api('POST', `/questions/${id}/close`)
    .then(() => { toast('问题已关闭', 'success'); return fetchAll() })
    .catch((e) => toast(e.message))
}

onMounted(fetchAll)
</script>

<template>
  <div v-if="loading" class="loading">加载中…</div>
  <div v-else-if="notFound" class="card empty">问题不存在或已被删除 <router-link to="/">返回首页</router-link></div>
  <div v-else class="layout">
    <div>
    <div class="card q-head">
      <h1>{{ q.title }}</h1>
      <div class="meta">
        <router-link v-for="t in q.tags" :key="t.id" class="tag" :to="{ path: '/', query: { tag: t.name } }">{{ t.name }}</router-link>
        <span :class="{ 'accepted-flag': q.status === 'closed' }">
          {{ q.status === 'open' ? '开放中' : q.status === 'closed' ? '已关闭' : '' }}
        </span>
        <span class="spacer"></span>
        <router-link :to="'/u/' + q.author.username">{{ q.author.username }}</router-link>
        <span>{{ timeAgo(q.created_at) }} 提问 · {{ q.view_count }} 次浏览</span>
      </div>

      <template v-if="!editingQuestion">
        <div class="content">{{ q.content }}</div>
      </template>
      <template v-else>
        <div class="edit-form">
          <input v-model="editQ.title" maxlength="150" />
          <textarea v-model="editQ.content"></textarea>
          <input v-model="editQ.tags" placeholder="标签，逗号分隔" />
          <div class="actions">
            <button class="btn primary small" @click="saveQuestion">保存</button>
            <button class="btn small" @click="editingQuestion = false">取消</button>
          </div>
        </div>
      </template>

      <div class="foot-row">
        <VoteBox
          target-type="question" :target-id="q.id"
          :upvotes="q.upvote_count" :downvotes="q.downvote_count" :value="q.viewer_vote"
        />
        <span class="spacer"></span>
        <template v-if="isAuthor && q.status === 'open' && !editingQuestion">
          <button class="btn small" @click="startEditQuestion">编辑</button>
          <button class="btn small" @click="closeQuestion">关闭</button>
        </template>
        <button v-if="canManage" class="btn small danger" @click="removeQuestion">删除</button>
      </div>

      <Comments target-type="question" :target-id="q.id" />
    </div>

    <div class="section-title">{{ answers.length }} 个回答</div>

    <div v-if="answers.length === 0" class="card empty">还没有回答，来抢沙发 →</div>

    <div v-for="a in answers" :key="a.id" class="card answer" :class="{ accepted: a.is_accepted }">
      <div v-if="a.is_accepted" class="accept-badge">✓ 已采纳的回答</div>
      <template v-if="editingAnswerId !== a.id">
        <div class="content">{{ a.content }}</div>
      </template>
      <template v-else>
        <div class="edit-form">
          <textarea v-model="editAContent"></textarea>
          <div class="actions">
            <button class="btn primary small" @click="saveAnswer(a)">保存</button>
            <button class="btn small" @click="editingAnswerId = 0">取消</button>
          </div>
        </div>
      </template>

      <div class="foot-row">
        <VoteBox
          target-type="answer" :target-id="a.id"
          :upvotes="a.upvote_count" :downvotes="a.downvote_count" :value="viewerVotes[a.id] || 0"
        />
        <span class="answer-author">
          <router-link :to="'/u/' + a.author.username">{{ a.author.username }}</router-link>
          <span>{{ timeAgo(a.created_at) }} 回答</span>
        </span>
        <span class="spacer"></span>
        <button
          v-if="isAuthor && !a.is_accepted"
          class="btn small accept-btn" @click="accept(a)"
        >采纳</button>
        <button
          v-if="store.me && store.me.id === a.user_id && editingAnswerId !== a.id"
          class="btn small" @click="startEditAnswer(a)"
        >编辑</button>
        <button
          v-if="store.me && (store.me.id === a.user_id || store.me.role === 'admin')"
          class="btn small danger" @click="removeAnswer(a)"
        >删除</button>
      </div>

      <Comments target-type="answer" :target-id="a.id" />
    </div>

    <div v-if="canAnswer" class="card">
      <div class="section-title inline">撰写回答</div>
      <textarea v-model="answerForm" placeholder="分享你的答案…"></textarea>
      <div class="actions">
        <button class="btn primary" :disabled="answering" @click="submitAnswer">发表回答</button>
      </div>
    </div>
    <div v-else-if="q.accepted_answer_id" class="card empty resolved-notice">
      ✓ 该问题已解决（已采纳答案），不再接受新的回答
    </div>
    <div v-else-if="!store.me" class="card empty">
      <router-link to="/login">登录</router-link> 后可以回答该问题
    </div>
    <div v-else-if="q.status === 'closed'" class="card empty">该问题已关闭，不能再回答</div>
    </div>

    <aside class="sidebar">
      <AuthorCard :username="q.author.username" />
      <RelatedQuestions :question="q" />
      <HotTagsCard />
    </aside>
  </div>
</template>
