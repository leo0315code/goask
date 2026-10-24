import { reactive } from 'vue'
import { api } from './api'

// 全局登录态；token 存 localStorage，me 为当前用户信息
export const store = reactive({
  me: null,
  ready: false,
  unread: 0, // 未读通知数
})

export async function loadMe() {
  if (!localStorage.getItem('goask_token')) {
    store.me = null
    store.ready = true
    return
  }
  try {
    store.me = await api('GET', '/users/me')
  } catch {
    store.me = null
  }
  store.ready = true
}

export function setToken(token) {
  if (token) localStorage.setItem('goask_token', token)
  else localStorage.removeItem('goask_token')
}

export function logout() {
  localStorage.removeItem('goask_token')
  store.me = null
  store.unread = 0
}

// 刷新未读通知数（顶栏铃铛用）
export async function refreshUnread() {
  if (!localStorage.getItem('goask_token')) {
    store.unread = 0
    return
  }
  try {
    const data = await api('GET', '/notifications?page=1&page_size=1')
    store.unread = data.unread_count || 0
  } catch {
    store.unread = 0
  }
}
