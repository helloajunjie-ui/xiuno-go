import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '../stores/user'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: '/threads',
    },
    {
      path: '/login',
      name: 'Login',
      component: () => import('../views/Login.vue'),
    },
    {
      path: '/register',
      name: 'Register',
      component: () => import('../views/Register.vue'),
    },
    {
      path: '/resetpw',
      name: 'ResetPassword',
      component: () => import('../views/ResetPassword.vue'),
    },
    {
      path: '/threads',
      name: 'ThreadList',
      component: () => import('../views/ThreadList.vue'),
    },
    {
      path: '/forum/:fid',
      name: 'ForumView',
      component: () => import('../views/ForumView.vue'),
    },
    {
      path: '/create',
      name: 'CreateThread',
      component: () => import('../views/CreateThread.vue'),
    },
    {
      path: '/thread/:tid',
      name: 'ThreadDetail',
      component: () => import('../views/ThreadDetail.vue'),
    },
    {
      path: '/user/:uid',
      name: 'UserProfile',
      component: () => import('../views/UserProfile.vue'),
    },
    {
      path: '/my',
      name: 'MyCenter',
      component: () => import('../views/MyCenter.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/my/password',
      name: 'MyPassword',
      component: () => import('../views/MyPassword.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/my/avatar',
      name: 'MyAvatar',
      component: () => import('../views/MyAvatar.vue'),
      meta: { requiresAuth: true },
    },

    // ================= 后台管理 (Admin) =================
    {
      path: '/admin',
      component: () => import('../views/admin/AdminLayout.vue'),
      meta: { requiresAdmin: true },
      redirect: '/admin/config',
      children: [
        { path: 'config', component: () => import('../views/admin/Config.vue'), meta: { title: '全局配置' } },
        { path: 'forum', component: () => import('../views/admin/Forum.vue'), meta: { title: '版块管理' } },
        { path: 'plugin', component: () => import('../views/admin/Plugin.vue'), meta: { title: '插件中枢' } },
        { path: 'user', component: () => import('../views/admin/User.vue'), meta: { title: '用户管控' } },
        { path: 'group', component: () => import('../views/admin/Group.vue'), meta: { title: '用户组管理' } },
        { path: 'modlog', component: () => import('../views/admin/ModLog.vue'), meta: { title: '版务日志' } },
      ],
    },
  ],
})

// 前端路由守卫：拦截未登录用户 + 非超管用户
router.beforeEach((to, _from, next) => {
  const userStore = useUserStore()
  if (to.meta.requiresAuth && !userStore.isLoggedIn) {
    return next('/login')
  }
  if (to.meta.requiresAdmin) {
    if (!userStore.isLoggedIn || userStore.user?.gid !== 1) {
      return next('/')
    }
  }
  next()
})

export default router
