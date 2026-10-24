<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { store, logout, loadMe } from './store'
import { toasts, toast } from './toast'

const router = useRouter()
const keyword = ref('')

function doSearch() {
  const kw = keyword.value.trim()
  router.push({ path: '/', query: kw ? { q: kw } : {} })
}

function onUnauthorized() {
  store.me = null
  toast('登录已过期，请重新登录')
}

function doLogout() {
  logout()
  toast('已退出登录', 'success')
  router.push('/')
}

onMounted(() => {
  window.addEventListener('goask:unauthorized', onUnauthorized)
  // 顶部登录态兜底刷新
  if (!store.ready) loadMe()
})
onUnmounted(() => window.removeEventListener('goask:unauthorized', onUnauthorized))
</script>

<template>
  <header class="topbar">
    <div class="container nav">
      <router-link class="logo" to="/">Go<span>Ask</span></router-link>
      <form class="search" @submit.prevent="doSearch">
        <input v-model="keyword" type="search" placeholder="搜索问题（标题 / 正文）…" />
      </form>
      <router-link to="/ask" class="btn primary">提问</router-link>
      <nav class="user-box">
        <template v-if="store.me">
          <router-link :to="'/u/' + store.me.username">{{ store.me.username }}</router-link>
          <span class="rep" :title="'声望 ' + store.me.reputation">{{ store.me.reputation }}</span>
          <button class="btn small" @click="doLogout">退出</button>
        </template>
        <template v-else>
          <router-link to="/login">登录</router-link>
          <router-link to="/register">注册</router-link>
        </template>
      </nav>
    </div>
  </header>

  <main class="container">
    <router-view :key="$route.fullPath" />
  </main>

  <footer class="footer">
    <div class="container">
      <span>GoAsk · Go + MySQL 问答社区</span>
      <span class="sep">·</span>
      <a href="/api/v1/questions" target="_blank" rel="noopener">开放 API</a>
    </div>
  </footer>

  <div v-for="t in toasts" :key="t.id" class="toast" :class="[t.type, t.id]">{{ t.message }}</div>
</template>
