<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '../stores/user'
import request from '../utils/request'

const router = useRouter()
const userStore = useUserStore()

const user = computed(() => userStore.user)
const isLoggedIn = computed(() => userStore.isLoggedIn)

const threads = ref<any[]>([])
const loading = ref(true)

const avatarUrl = computed(() => {
  if (!user.value) return ''
  return `/upload/avatar/${user.value.uid}.png?t=${user.value.avatar || 0}`
})

onMounted(async () => {
  if (!isLoggedIn.value) {
    router.push('/login')
    return
  }
  try {
    // 使用 mythread 接口获取用户参与过的主题（含回帖的帖子）
    const data: any = await request.get('/my/thread', { params: { page: 1 } })
    threads.value = data.list || []
  } catch (e) {
    console.error('加载失败', e)
  } finally {
    loading.value = false
  }
})

function go(path: string) {
  router.push(path)
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
    <h1 class="text-xl font-bold mb-4">个人中心</h1>

    <!-- 用户信息卡片 -->
    <div class="bg-white rounded-xl shadow-sm border border-gray-200 p-6 mb-4">
      <div class="flex items-center gap-4">
        <img :src="avatarUrl" alt="avatar"
          class="w-16 h-16 rounded-full object-cover border-2 border-gray-100 shrink-0"
          @error="($event.target as HTMLImageElement).src='/upload/avatar/0.png'" />
        <div class="flex-1 min-w-0">
          <h2 class="text-lg font-bold text-gray-900">{{ user?.username }}</h2>
          <p class="text-sm text-gray-500">{{ user?.email }}</p>
        </div>
      </div>
    </div>

    <!-- 快捷操作 -->
    <div class="grid grid-cols-2 gap-3 mb-4">
      <button @click="go('/my/password')"
        class="bg-white rounded-xl shadow-sm border border-gray-200 p-4 hover:shadow-md hover:border-indigo-200 transition-all text-left">
        <div class="text-lg mb-1">🔑</div>
        <div class="text-sm font-medium text-gray-900">修改密码</div>
        <div class="text-xs text-gray-400">定期更换保障安全</div>
      </button>
      <button @click="go('/my/avatar')"
        class="bg-white rounded-xl shadow-sm border border-gray-200 p-4 hover:shadow-md hover:border-indigo-200 transition-all text-left">
        <div class="text-lg mb-1">🖼️</div>
        <div class="text-sm font-medium text-gray-900">上传头像</div>
        <div class="text-xs text-gray-400">换个新形象</div>
      </button>
    </div>

    <!-- 统计 -->
    <div class="bg-white rounded-xl shadow-sm border border-gray-200 p-4 mb-4">
      <div class="flex justify-around text-center">
        <div>
          <div class="text-lg font-bold text-gray-900">{{ user?.threads || 0 }}</div>
          <div class="text-xs text-gray-400">主题</div>
        </div>
        <div>
          <div class="text-lg font-bold text-gray-900">{{ user?.posts || 0 }}</div>
          <div class="text-xs text-gray-400">回复</div>
        </div>
        <div>
          <div class="text-lg font-bold text-gray-900">{{ user?.credits || 0 }}</div>
          <div class="text-xs text-gray-400">积分</div>
        </div>
      </div>
    </div>

    <!-- 我的帖子 -->
    <h2 class="text-base font-bold text-gray-900 mb-3">我的最新主题</h2>
    <div v-if="loading" class="space-y-3">
      <div v-for="i in 3" :key="i" class="bg-white rounded-xl shadow-sm border border-gray-200 p-5 animate-pulse">
        <div class="h-5 bg-gray-200 rounded w-3/4 mb-3"></div>
        <div class="h-4 bg-gray-100 rounded w-1/2"></div>
      </div>
    </div>
    <div v-else class="space-y-3">
      <div v-for="thread in threads" :key="thread.tid"
        @click="go(`/thread/${thread.tid}`)"
        class="bg-white rounded-xl shadow-sm border border-gray-200 p-4 hover:shadow-md hover:border-indigo-200 transition-all cursor-pointer">
        <h3 class="text-sm font-semibold text-gray-900 mb-1 line-clamp-1">{{ thread.subject }}</h3>
        <div class="text-xs text-gray-400">{{ timeAgo(thread.last_date) }}</div>
      </div>
      <div v-if="threads.length === 0" class="text-center py-8 text-gray-400 text-sm">
        还没有发过帖
      </div>
    </div>
  </div>
</template>
