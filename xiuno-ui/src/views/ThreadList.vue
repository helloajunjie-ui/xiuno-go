<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import request from '../utils/request'

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

const threads = ref<ThreadItem[]>([])
const loading = ref(true)
const router = useRouter()

onMounted(async () => {
  try {
    // fid 为空 → 全站模式（首页），显示所有版块的最新帖子
    const data: any = await request.get('/thread', { params: { page: 1 } })
    threads.value = data.threads || []
  } catch (e) {
    console.error('获取帖子列表失败', e)
  } finally {
    loading.value = false
  }
})

function goDetail(tid: number) {
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
    <h1 class="text-xl font-bold mb-6">社区最新</h1>

    <!-- 骨架屏 -->
    <div v-if="loading" class="space-y-3">
      <div v-for="i in 5" :key="i" class="bg-white rounded-xl shadow-sm border border-gray-200 p-5 animate-pulse">
        <div class="h-5 bg-gray-200 rounded w-3/4 mb-3"></div>
        <div class="h-4 bg-gray-100 rounded w-1/2"></div>
      </div>
    </div>

    <!-- 帖子列表 -->
    <div v-else class="space-y-3">
      <div v-for="thread in threads" :key="thread.tid"
        @click="goDetail(thread.tid)"
        class="bg-white rounded-xl shadow-sm border border-gray-200 p-5 hover:shadow-md hover:border-indigo-200 transition-all cursor-pointer">
        <h2 class="text-base font-semibold text-gray-900 mb-2 line-clamp-1">{{ thread.subject }}</h2>
        <div class="flex items-center gap-4 text-xs text-gray-500">
          <span>{{ thread.username }}</span>
          <span>{{ timeAgo(thread.last_date) }}</span>
          <span class="ml-auto">{{ thread.views }} 浏览 · {{ thread.posts }} 回复</span>
        </div>
      </div>

      <div v-if="threads.length === 0" class="text-center py-16 text-gray-400">
        暂无帖子，快去发第一帖吧！
      </div>
    </div>
  </div>
</template>
