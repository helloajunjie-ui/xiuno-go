<!-- xiuno-go v2.1.0-beta 尼克修改版 -->
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
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

interface Tag {
  tagid: number
  name: string
  threads: number
}

const route = useRoute()
const router = useRouter()

const tag = ref<Tag | null>(null)
const threads = ref<ThreadItem[]>([])
const loading = ref(true)

onMounted(async () => {
  try {
    const tagid = route.params.tagid as string
    const [tagData, threadData]: any = await Promise.all([
      request.get(`/tag/${tagid}`),
      request.get(`/tag/${tagid}/thread`, { params: { page: 1 } }),
    ])
    tag.value = tagData as Tag
    threads.value = threadData.threads || []
  } catch (e) {
    console.error('获取标签数据失败', e)
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
    <div class="mb-6">
      <div v-if="loading" class="h-6 bg-gray-200 rounded w-48 animate-pulse mb-2"></div>
      <template v-else-if="tag">
        <h1 class="text-xl font-bold text-gray-900">#{{ tag.name }}</h1>
        <p class="text-sm text-gray-500 mt-1">{{ tag.threads }} 个主题</p>
      </template>
      <div v-else class="text-gray-400 text-sm">标签不存在</div>
    </div>

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
        该标签下暂无帖子
      </div>
    </div>
  </div>
</template>
