import { reactive } from 'vue'
import { api } from './api'

// 全局登录态；token 存 localStorage，me 为当前用户信息
export const store = reactive({
  me: null,
  ready: false,
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
}
