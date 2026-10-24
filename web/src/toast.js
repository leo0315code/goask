import { reactive } from 'vue'

const seq = { n: 0 }

export const toasts = reactive([])

// toast 全局提示，3 秒自动消失
export function toast(message, type = 'error') {
  const id = ++seq.n
  toasts.push({ id, message, type })
  setTimeout(() => {
    const i = toasts.findIndex((t) => t.id === id)
    if (i >= 0) toasts.splice(i, 1)
  }, 3000)
}
