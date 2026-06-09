import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/', component: () => import('@/views/HomeView.vue') },
  { path: '/bracket', component: () => import('@/views/BracketView.vue') },
  { path: '/wheel', component: () => import('@/views/WheelView.vue') },
  { path: '/settings', component: () => import('@/views/SettingsView.vue') },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router
