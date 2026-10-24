import { createRouter, createWebHashHistory } from 'vue-router'
import Home from './views/Home.vue'
import QuestionDetail from './views/QuestionDetail.vue'
import Ask from './views/Ask.vue'
import Login from './views/Login.vue'
import Register from './views/Register.vue'
import Profile from './views/Profile.vue'
import Notifications from './views/Notifications.vue'
import Settings from './views/Settings.vue'
import Admin from './views/Admin.vue'
import { store } from './store'

export const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', component: Home },
    { path: '/q/:id', component: QuestionDetail },
    { path: '/ask', component: Ask, meta: { auth: true } },
    { path: '/login', component: Login },
    { path: '/register', component: Register },
    { path: '/u/:username', component: Profile },
    { path: '/notifications', component: Notifications, meta: { auth: true } },
    { path: '/settings', component: Settings, meta: { auth: true } },
    { path: '/admin', component: Admin, meta: { auth: true, admin: true } },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
  scrollBehavior() {
    return { top: 0 }
  },
})

// 路由守卫：登录页未登录跳登录；管理页仅管理员可进
router.beforeEach((to) => {
  if (to.meta.auth && !localStorage.getItem('goask_token')) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (to.meta.admin && (!store.me || store.me.role !== 'admin')) {
    return { path: '/' }
  }
})
