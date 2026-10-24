import { createApp } from 'vue'
import App from './App.vue'
import { router } from './router'
import { loadMe } from './store'
import './style.css'

// 先恢复登录态再挂载，避免导航守卫误判
loadMe().finally(() => {
  createApp(App).use(router).mount('#app')
})
