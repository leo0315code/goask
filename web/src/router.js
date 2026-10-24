import { createRouter, createWebHashHistory } from 'vue-router'
import Home from './views/Home.vue'
import QuestionDetail from './views/QuestionDetail.vue'
import Ask from './views/Ask.vue'
import Login from './views/Login.vue'
import Register from './views/Register.vue'
import Profile from './views/Profile.vue'

export const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', component: Home },
    { path: '/q/:id', component: QuestionDetail },
    { path: '/ask', component: Ask, meta: { auth: true } },
    { path: '/login', component: Login },
    { path: '/register', component: Register },
    { path: '/u/:username', component: Profile },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
  scrollBehavior() {
    return { top: 0 }
  },
})

// 需要登录的页面未登录时跳转登录页，登录后回跳
router.beforeEach((to) => {
  if (to.meta.auth && !localStorage.getItem('goask_token')) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
})
