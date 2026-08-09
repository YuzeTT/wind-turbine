import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: () => import('@/layouts/MainLayout.vue'),
      redirect: '/dashboard',
      children: [
        {
          path: 'dashboard',
          name: 'dashboard',
          component: () => import('@/views/DashboardView.vue'),
          meta: { title: '总览看板' },
        },
        {
          path: 'turbines',
          name: 'turbines',
          component: () => import('@/views/TurbineListView.vue'),
          meta: { title: '风机列表' },
        },
        {
          path: 'turbines/:id',
          name: 'turbine-detail',
          component: () => import('@/views/TurbineDetailView.vue'),
          meta: { title: '风机详情' },
        },
        {
          path: 'alarms',
          name: 'alarms',
          component: () => import('@/views/AlarmView.vue'),
          meta: { title: '报警管理' },
        },
        {
          path: 'operation-logs',
          name: 'operation-logs',
          component: () => import('@/views/OperationLogView.vue'),
          meta: { title: '操作日志' },
        },
        {
          path: 'system-logs',
          name: 'system-logs',
          component: () => import('@/views/SystemLogView.vue'),
          meta: { title: '系统日志' },
        },
        {
          path: 'users',
          name: 'users',
          component: () => import('@/views/UserManagementView.vue'),
          meta: { title: '用户管理', adminOnly: true },
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/',
    },
  ],
})

// 全局前置守卫
router.beforeEach(async (to) => {
  const auth = useAuthStore()

  // 公开路由直接放行
  if (to.meta.public) {
    // 已登录用户不再进登录页
    if (to.name === 'login' && auth.isLoggedIn) {
      return { name: 'dashboard' }
    }
    return true
  }

  // 未登录 → 跳登录
  if (!auth.isLoggedIn) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  // 已有 token 但还没拉取用户信息
  if (!auth.user) {
    await auth.fetchProfile()
    if (!auth.isLoggedIn) {
      return { name: 'login' }
    }
  }

  // 管理员专属路由
  if (to.meta.adminOnly && !auth.isAdmin) {
    return { name: 'dashboard' }
  }

  return true
})

export default router
