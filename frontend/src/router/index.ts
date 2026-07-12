import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '../stores/user'

// 应用就绪标志：fetchProfile() 完成前阻止路由守卫误判
// App.vue onMounted 中 fetchProfile() 完成后设为 true
export let appReady = false
export function setAppReady() {
  appReady = true
}

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
      path: '/tags',
      name: 'TagCloud',
      component: () => import('../views/TagCloud.vue'),
    },
    {
      path: '/tag/:tagid',
      name: 'TagThreadList',
      component: () => import('../views/TagThreadList.vue'),
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
      redirect: '/admin',
      children: [
        { path: '', component: () => import('../views/admin/Dashboard.vue'), meta: { title: '控制台概览' } },
        { path: 'config', component: () => import('../views/admin/Config.vue'), meta: { title: '全局配置' } },
        { path: 'forum', component: () => import('../views/admin/Forum.vue'), meta: { title: '版块管理' } },
        { path: 'tag', component: () => import('../views/admin/Tag.vue'), meta: { title: '标签管理' } },
        { path: 'thread', component: () => import('../views/admin/Thread.vue'), meta: { title: '主题管理' } },
        { path: 'plugin', component: () => import('../views/admin/Plugin.vue'), meta: { title: '插件中枢' } },
        { path: 'theme', component: () => import('../views/admin/Theme.vue'), meta: { title: '外观实验室' } },
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

  // 等待 fetchProfile() 完成后再做鉴权判断
  if (!appReady) {
    // 返回 false 阻止本次导航，Vue Router 会自动重试
    return next(false)
  }

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
