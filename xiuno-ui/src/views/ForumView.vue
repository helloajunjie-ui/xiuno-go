<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import request from '../utils/request'
import { useUserStore } from '../stores/user'
import ModerateModal from '../components/ModerateModal.vue'

interface Forum {
  fid: number
  name: string
  brief: string
  announcement: string
  threads: number
  todayposts: number
  todaythreads: number
  icon: number
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
const forum = ref<Forum | null>(null)
const threads = ref<ThreadItem[]>([])
const loading = ref(true)
const page = ref(1)
const hasMore = ref(true)

// 版务模式
const modMode = ref(false)
const selectedTids = ref<Set<number>>(new Set())
const modAction = ref('')
const showModModal = ref(false)

const canModerate = computed(() => {
  return userStore.isLoggedIn && userStore.user?.gid !== undefined && userStore.user.gid <= 3
})

function toggleSelect(tid: number) {
  const s = new Set(selectedTids.value)
  if (s.has(tid)) {
    s.delete(tid)
  } else {
    s.add(tid)
  }
  selectedTids.value = s
}

function selectAll() {
  selectedTids.value = new Set(threads.value.map(t => t.tid))
}

function clearSelection() {
  selectedTids.value = new Set()
}

function openModModal(action: string) {
  if (selectedTids.value.size === 0) return
  modAction.value = action
  showModModal.value = true
}

function onModDone() {
  clearSelection()
  modMode.value = false
  loadForum()
}

async function loadForum() {
  const fid = route.params.fid as string
  try {
    const [forumData, threadData]: any = await Promise.all([
      request.get(`/forum/${fid}`),
      request.get('/thread', { params: { fid, page: page.value } }),
    ])
    forum.value = forumData
    threads.value = threadData.threads || []
    hasMore.value = (threadData.threads || []).length >= 20
  } catch (e) {
    console.error('加载版块失败', e)
  } finally {
    loading.value = false
  }
}

onMounted(loadForum)

function goDetail(tid: number) {
  if (modMode.value) return
  router.push(`/thread/${tid}`)
}

function goCreate() {
  router.push(`/create?fid=${route.params.fid}`)
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
    <!-- 面包屑 -->
    <nav class="text-sm text-gray-500 mb-4">
      <router-link to="/" class="hover:text-indigo-600">首页</router-link>
      <span class="mx-2">/</span>
      <span class="text-gray-800" v-if="forum">{{ forum.name }}</span>
    </nav>

    <!-- 版块信息卡片 -->
    <div v-if="forum" class="bg-white rounded-xl shadow-sm border border-gray-200 p-6 mb-4">
      <div class="flex items-start gap-4">
        <div class="w-16 h-16 rounded-lg bg-indigo-100 flex items-center justify-center text-2xl font-bold text-indigo-600 shrink-0">
          {{ forum.name.charAt(0) }}
        </div>
        <div class="flex-1 min-w-0">
          <h1 class="text-xl font-bold text-gray-900 mb-1">{{ forum.name }}</h1>
          <p v-if="forum.brief" class="text-sm text-gray-500 mb-3">{{ forum.brief }}</p>
          <p v-if="forum.announcement" class="text-xs text-amber-600 bg-amber-50 rounded px-2 py-1 inline-block">
            📢 {{ forum.announcement }}
          </p>
        </div>
        <div class="shrink-0 flex gap-2">
          <!-- 版务模式切换 -->
          <button v-if="canModerate" @click="modMode = !modMode"
            class="px-3 py-2 text-sm border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
            :class="modMode ? 'bg-indigo-50 border-indigo-300 text-indigo-700' : 'text-gray-700'">
            {{ modMode ? '退出版务' : '版务' }}
          </button>
          <button @click="goCreate"
            class="bg-indigo-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-indigo-700 transition-colors">
            + 发帖
          </button>
        </div>
      </div>
      <!-- 统计 -->
      <div class="flex gap-6 mt-4 pt-4 border-t border-gray-100 text-sm text-gray-500">
        <span>主题 <b class="text-gray-800">{{ forum.threads }}</b></span>
        <span>今日帖 <b class="text-gray-800">{{ forum.todayposts }}</b></span>
        <span>今日主题 <b class="text-gray-800">{{ forum.todaythreads }}</b></span>
      </div>
    </div>

    <!-- 版务工具栏 -->
    <div v-if="modMode" class="bg-amber-50 border border-amber-200 rounded-xl p-3 mb-4 flex items-center gap-3 flex-wrap">
      <span class="text-sm text-amber-800 font-medium">版务模式</span>
      <span class="text-sm text-amber-700">已选 {{ selectedTids.size }} 篇</span>
      <button @click="selectAll" class="text-xs px-2 py-1 bg-white border border-amber-300 rounded hover:bg-amber-100 transition-colors">全选</button>
      <button @click="clearSelection" class="text-xs px-2 py-1 bg-white border border-amber-300 rounded hover:bg-amber-100 transition-colors">取消选择</button>
      <div class="flex-1"></div>
      <button @click="openModModal('top')" :disabled="selectedTids.size === 0"
        class="text-xs px-3 py-1.5 bg-white border border-gray-300 rounded hover:bg-gray-100 disabled:opacity-40 transition-colors">置顶</button>
      <button @click="openModModal('close')" :disabled="selectedTids.size === 0"
        class="text-xs px-3 py-1.5 bg-white border border-gray-300 rounded hover:bg-gray-100 disabled:opacity-40 transition-colors">关闭</button>
      <button @click="openModModal('move')" :disabled="selectedTids.size === 0"
        class="text-xs px-3 py-1.5 bg-white border border-gray-300 rounded hover:bg-gray-100 disabled:opacity-40 transition-colors">移动</button>
      <button @click="openModModal('delete')" :disabled="selectedTids.size === 0"
        class="text-xs px-3 py-1.5 bg-white border border-red-300 text-red-700 rounded hover:bg-red-50 disabled:opacity-40 transition-colors">删除</button>
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
        class="bg-white rounded-xl shadow-sm border border-gray-200 p-5 hover:shadow-md hover:border-indigo-200 transition-all cursor-pointer flex items-start gap-3">
        <!-- 版务勾选框 -->
        <div v-if="modMode" @click.stop class="mt-0.5 shrink-0">
          <input type="checkbox" :checked="selectedTids.has(thread.tid)" @change="toggleSelect(thread.tid)"
            class="accent-indigo-600 w-4 h-4" />
        </div>
        <div class="flex-1 min-w-0">
          <h2 class="text-base font-semibold text-gray-900 mb-2 line-clamp-1">{{ thread.subject }}</h2>
          <div class="flex items-center gap-4 text-xs text-gray-500">
            <span>{{ thread.username }}</span>
            <span>{{ timeAgo(thread.last_date) }}</span>
            <span class="ml-auto">{{ thread.views }} 浏览 · {{ thread.posts }} 回复</span>
          </div>
        </div>
      </div>

      <div v-if="threads.length === 0" class="text-center py-16 text-gray-400">
        该版块暂无帖子
      </div>
    </div>

    <!-- 版务操作弹窗 -->
    <ModerateModal
      v-if="showModModal"
      :action="modAction"
      :tidarr="Array.from(selectedTids)"
      :onDone="onModDone"
      @close="showModModal = false" />
  </div>
</template>
