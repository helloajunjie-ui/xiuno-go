<!-- xiuno-go v2.1.0-beta 尼克修改版 -->
<script setup lang="ts">
import { useRouter } from 'vue-router'

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

const props = defineProps<{
  threads: ThreadItem[]
}>()

const router = useRouter()

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
  <div class="waterfall-grid">
    <div v-for="thread in props.threads" :key="thread.tid"
      @click="goDetail(thread.tid)"
      class="waterfall-item bg-white rounded-xl shadow-sm border border-gray-200 p-4 hover:shadow-md hover:border-indigo-200 transition-all cursor-pointer break-inside-avoid mb-4"
      :style="{ borderRadius: 'var(--radius-base)' }">
      <h2 class="text-sm font-semibold text-gray-900 mb-2 line-clamp-2">{{ thread.subject }}</h2>
      <div class="flex items-center gap-2 text-xs text-gray-500 mt-2">
        <span class="font-medium" :style="{ color: 'var(--c-primary)' }">{{ thread.username }}</span>
        <span class="ml-auto">{{ thread.views }} 浏览</span>
      </div>
      <div class="text-xs text-gray-400 mt-1">
        {{ timeAgo(thread.last_date) }} · {{ thread.posts }} 回复
      </div>
    </div>

    <div v-if="props.threads.length === 0" class="text-center py-16 text-gray-400 col-span-full">
      暂无帖子，快去发第一帖吧！
    </div>
  </div>
</template>

<style scoped>
.waterfall-grid {
  column-count: 2;
  column-gap: 1rem;
}

@media (max-width: 640px) {
  .waterfall-grid {
    column-count: 1;
  }
}
</style>
