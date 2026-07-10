<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import request from '../utils/request'
import { useUserStore } from '../stores/user'

interface UserProfile {
  uid: number
  gid: number
  username: string
  email: string
  avatar: number
  threads: number
  posts: number
  credits: number
  create_date: number
  login_date: number
  logins: number
}

interface ThreadItem {
  tid: number
  fid: number
  subject: string
  username: string
  avatar: number
  create_date: number
  last_date: number
  views: number
  posts: number
}

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const user = ref<UserProfile | null>(null)
const threads = ref<ThreadItem[]>([])
const loading = ref(true)
const activeTab = ref<'threads' | 'posts'>('threads')
const deleting = ref(false)

const avatarUrl = computed(() => {
  if (!user.value) return ''
  return `/upload/avatar/${user.value.uid}.png?t=${user.value.avatar}`
})

const joinDate = computed(() => {
  if (!user.value) return ''
  const d = new Date(user.value.create_date * 1000)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
})

const lastLogin = computed(() => {
  if (!user.value) return ''
  const diff = Date.now() / 1000 - user.value.login_date
  if (diff < 60) return '刚刚'
  if (diff < 3600) return `${Math.floor(diff / 60)} 分钟前`
  if (diff < 86400) return `${Math.floor(diff / 3600)} 小时前`
  return `${Math.floor(diff / 86400)} 天前`
})

// 当前登录用户是否有权删除目标用户
const canDelete = computed(() => {
  if (!userStore.isLoggedIn || !user.value) return false
  const myGid = userStore.user?.gid ?? 0
  // 只有超管(gid=1)和超版(gid=2)可以删除用户
  if (myGid !== 1 && myGid !== 2) return false
  // 不能删除管理组成员(gid<6)
  if (user.value.gid < 6) return false
  // 不能删除自己
  if (userStore.user?.uid === user.value.uid) return false
  return true
})

async function handleDeleteUser() {
  if (!user.value) return
  if (!confirm(`确定要删除用户「${user.value.username}」吗？\n\n此操作将级联删除该用户的所有主题、回帖和附件，不可撤销！`)) return
  deleting.value = true
  try {
    await request.delete(`/user/${user.value.uid}/delete`)
    alert('用户已删除')
    router.push('/')
  } catch (e: any) {
    alert(e.message || '删除失败')
  } finally {
    deleting.value = false
  }
}

onMounted(async () => {
  const uid = route.params.uid as string
  try {
    const [userData, threadData]: any = await Promise.all([
      request.get(`/user/${uid}`),
      request.get(`/user/${uid}/thread`, { params: { page: 1 } }),
    ])
    user.value = userData
    threads.value = threadData.list || []
  } catch (e) {
    console.error('加载用户资料失败', e)
  } finally {
    loading.value = false
  }
})

function goThread(tid: number) {
  router.push(`/thread/${tid}`)
}

function timeAgo(ts: number): string {
  const diff = Date.now() / 1000 - ts
  if (diff < 60) return '刚刚'
  if (diff < 3600) return `${Math.floor(diff / 60)} 分钟前`
  if (diff < 86400) return `${Math.floor(diff / 3600)} 小时前`
  return `${Math.floor(diff / 86400)} 天前`
}
</script>

<template>
  <div>
    <!-- 骨架屏 -->
    <div v-if="loading" class="animate-pulse">
      <div class="bg-white rounded-xl shadow-sm border border-gray-200 p-6 mb-4">
        <div class="flex items-center gap-4">
          <div class="w-20 h-20 rounded-full bg-gray-200"></div>
          <div class="flex-1">
            <div class="h-6 bg-gray-200 rounded w-32 mb-2"></div>
            <div class="h-4 bg-gray-100 rounded w-48"></div>
          </div>
        </div>
      </div>
      <div class="bg-white rounded-xl shadow-sm border border-gray-200 p-5">
        <div class="h-5 bg-gray-200 rounded w-3/4 mb-3"></div>
        <div class="h-4 bg-gray-100 rounded w-1/2"></div>
      </div>
    </div>

    <!-- 用户资料 -->
    <div v-else-if="user">
      <!-- 面包屑 -->
      <nav class="text-sm text-gray-500 mb-4">
        <router-link to="/" class="hover:text-indigo-600">首页</router-link>
        <span class="mx-2">/</span>
        <span class="text-gray-800">{{ user.username }}</span>
      </nav>

      <!-- 用户信息卡片 -->
      <div class="bg-white rounded-xl shadow-sm border border-gray-200 p-6 mb-4">
        <div class="flex items-start gap-5">
          <img :src="avatarUrl" alt="avatar"
            class="w-20 h-20 rounded-full object-cover border-2 border-gray-100 shrink-0"
            @error="($event.target as HTMLImageElement).src='/upload/avatar/0.png'" />
          <div class="flex-1 min-w-0">
            <h1 class="text-xl font-bold text-gray-900">{{ user.username }}</h1>
            <div class="flex flex-wrap gap-x-6 gap-y-1 mt-2 text-sm text-gray-500">
              <span>积分 <b class="text-gray-800">{{ user.credits }}</b></span>
              <span>主题 <b class="text-gray-800">{{ user.threads }}</b></span>
              <span>回复 <b class="text-gray-800">{{ user.posts }}</b></span>
              <span>登录 <b class="text-gray-800">{{ user.logins }}</b> 次</span>
            </div>
            <div class="flex flex-wrap gap-x-6 gap-y-1 mt-1 text-xs text-gray-400">
              <span>加入于 {{ joinDate }}</span>
              <span>最后活跃 {{ lastLogin }}</span>
            </div>
          </div>
          <div v-if="canDelete" class="mt-3 pt-3 border-t border-gray-100">
            <button @click="handleDeleteUser" :disabled="deleting"
              class="px-4 py-1.5 text-sm text-red-600 border border-red-200 rounded-lg hover:bg-red-50 disabled:opacity-50 transition-colors">
              {{ deleting ? '删除中...' : '删除用户' }}
            </button>
          </div>
        </div>
      </div>

      <!-- Tab 切换 -->
      <div class="flex gap-1 mb-3">
        <button @click="activeTab = 'threads'"
          class="px-4 py-2 text-sm rounded-lg transition-colors"
          :class="activeTab === 'threads' ? 'bg-indigo-600 text-white' : 'bg-white text-gray-600 hover:bg-gray-50 border border-gray-200'">
          主题
        </button>
        <button @click="activeTab = 'posts'"
          class="px-4 py-2 text-sm rounded-lg transition-colors"
          :class="activeTab === 'posts' ? 'bg-indigo-600 text-white' : 'bg-white text-gray-600 hover:bg-gray-50 border border-gray-200'">
          回复
        </button>
      </div>

      <!-- 帖子列表 -->
      <div class="space-y-3">
        <div v-for="thread in threads" :key="thread.tid"
          @click="goThread(thread.tid)"
          class="bg-white rounded-xl shadow-sm border border-gray-200 p-5 hover:shadow-md hover:border-indigo-200 transition-all cursor-pointer">
          <h2 class="text-base font-semibold text-gray-900 mb-2 line-clamp-1">{{ thread.subject }}</h2>
          <div class="flex items-center gap-4 text-xs text-gray-500">
            <span>{{ thread.username }}</span>
            <span>{{ timeAgo(thread.last_date) }}</span>
            <span class="ml-auto">{{ thread.views }} 浏览 · {{ thread.posts }} 回复</span>
          </div>
        </div>

        <div v-if="threads.length === 0" class="text-center py-16 text-gray-400">
          暂无内容
        </div>
      </div>
    </div>

    <!-- 用户不存在 -->
    <div v-else class="text-center py-20 text-gray-400">
      <div class="text-5xl mb-4">😶</div>
      <p>该用户已遁入虚空</p>
    </div>
  </div>
</template>
