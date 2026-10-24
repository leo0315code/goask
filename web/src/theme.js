import { ref, watchEffect } from 'vue'

// 主题：light / dark；首次访问跟随系统，切换后持久化到 localStorage
const saved = localStorage.getItem('goask_theme')
const theme = ref(
  saved === 'dark' || saved === 'light'
    ? saved
    : window.matchMedia('(prefers-color-scheme: dark)').matches
      ? 'dark'
      : 'light'
)

watchEffect(() => {
  document.documentElement.classList.toggle('dark', theme.value === 'dark')
  localStorage.setItem('goask_theme', theme.value)
})

export function useTheme() {
  const toggle = () => {
    theme.value = theme.value === 'dark' ? 'light' : 'dark'
  }
  return { theme, toggle }
}
