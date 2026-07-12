<!-- xiuno-go v2.1.0-beta 尼克修改版 -->
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import request from '../utils/request'

interface Tag {
  tagid: number
  name: string
  threads: number
  create_date: number
}

const tags = ref<Tag[]>([])
const loading = ref(true)
const router = useRouter()

onMounted(async () => {
  try {
    const data: any = await request.get('/tag', { params: { page: 1 } })
    tags.value = data.tags || []
  } catch (e) {
    console.error('获取标签列表失败', e)
  } finally {
    loading.value = false
  }
})

function goTag(tagid: number) {
  router.push(`/tag/${tagid}`)
}
</script>

<template>
  <div>
    <div class="mb-6">
      <h1 class="text-xl font-bold text-gray-900">热门标签</h1>
      <p class="text-sm text-gray-500 mt-1">点击标签查看相关帖子</p>
    </div>

    <!-- 骨架屏 -->
    <div v-if="loading" class="flex flex-wrap gap-3">
      <div v-for="i in 12" :key="i"
        class="h-8 bg-gray-200 rounded-full animate-pulse" :style="{ width: (60 + Math.random() * 80) + 'px' }">
      </div>
    </div>

    <!-- 标签云 -->
    <div v-else class="flex flex-wrap gap-3">
      <button v-for="tag in tags" :key="tag.tagid"
        @click="goTag(tag.tagid)"
        class="inline-flex items-center gap-1.5 px-4 py-1.5 rounded-full text-sm font-medium
               transition-all duration-200 hover:shadow-md hover:-translate-y-0.5
               cursor-pointer border"
        :class="[
          tag.threads > 10
            ? 'bg-indigo-100 text-indigo-700 border-indigo-200 hover:bg-indigo-200'
            : tag.threads > 3
              ? 'bg-blue-50 text-blue-600 border-blue-200 hover:bg-blue-100'
              : 'bg-gray-50 text-gray-600 border-gray-200 hover:bg-gray-100'
        ]">
        <span>{{ tag.name }}</span>
        <span class="text-xs opacity-60">×{{ tag.threads }}</span>
      </button>

      <div v-if="tags.length === 0" class="w-full text-center py-16 text-gray-400">
        暂无标签，去发帖时添加标签吧！
      </div>
    </div>
  </div>
</template>
