<!-- xiuno-go v2.1.0-beta 尼克修改版 -->
<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '../stores/user'
import { useThemeStore } from '../stores/theme'
import request from '../utils/request'
import ClassicList from '../components/layout/ClassicList.vue'
import WaterfallList from '../components/layout/WaterfallList.vue'

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

interface Forum {
  fid: number
  name: string
  threads: number
  todayposts: number
}

interface Tag {
  tagid: number
  name: string
  threads: number
}

const threads = ref<ThreadItem[]>([])
const forums = ref<Forum[]>([])
const tags = ref<Tag[]>([])
const loading = ref(true)
const router = useRouter()
const userStore = useUserStore()
const themeStore = useThemeStore()

// 根据主题配置动态选择布局组件
const layoutComp = computed(() => {
  if (themeStore.config?.list_layout === 'waterfall') {
    return WaterfallList
  }
  return ClassicList
})

onMounted(async () => {
  try {
    const [threadData, forumData, tagData]: any = await Promise.all([
      request.get('/thread', { params: { page: 1 } }),
      request.get('/forum'),
      request.get('/tag', { params: { page: 1 } }),
    ])
    threads.value = threadData.threads || []
    forums.value = forumData || []
    tags.value = tagData.tags || []
  } catch (e) {
    console.error('获取数据失败', e)
  } finally {
    loading.value = false
  }
})

function goForum(fid: number) {
  router.push(`/forum/${fid}`)
}

function goTag(tagid: number) {
  router.push(`/tag/${tagid}`)
}

function tagStyle(tag: Tag): Record<string, string> {
  const maxThreads = Math.max(...tags.value.map(t => t.threads), 1)
  const ratio = tag.threads / maxThreads
  // 字体大小: 0.75rem(12px) ~ 1.125rem(18px)
  const size = 0.75 + ratio * 0.375
  // 颜色深浅: 最热 indigo-700 (#4338ca) → 最冷 gray-400 (#9ca3af)
  const r = Math.round(156 + (67 - 156) * ratio)   // 9c → 43
  const g = Math.round(163 + (56 - 163) * ratio)   // a3 → 38
  const b = Math.round(175 + (202 - 175) * ratio)  // af → ca
  return {
    fontSize: `${size}rem`,
    color: `rgb(${r}, ${g}, ${b})`,
    fontWeight: ratio > 0.5 ? '600' : '400',
  }
}

</script>

<template>
  <div class="flex gap-5">
    <!-- ========== 左侧：版块列表 ========== -->
    <aside class="w-52 shrink-0 hidden lg:block">
      <div class="bg-white rounded-xl shadow-sm border border-gray-200 p-4 sticky top-20">
        <h3 class="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-3">版块</h3>
        <div class="space-y-0.5">
          <div v-for="f in forums" :key="f.fid"
            @click="goForum(f.fid)"
            class="flex items-center justify-between px-3 py-2 rounded-lg text-sm cursor-pointer
                   hover:bg-indigo-50 hover:text-indigo-600 transition-colors group">
            <span class="text-gray-700 group-hover:text-indigo-600">{{ f.name }}</span>
            <span class="text-xs text-gray-400 group-hover:text-indigo-400">{{ f.threads }}</span>
          </div>
          <div v-if="forums.length === 0" class="text-xs text-gray-400 text-center py-4">
            暂无版块
          </div>
        </div>
      </div>
    </aside>

    <!-- ========== 中间：帖子列表 ========== -->
    <div class="flex-1 min-w-0">
      <div class="flex items-center justify-between mb-5">
        <h1 class="text-lg font-bold text-gray-900">社区最新</h1>
        <button @click="router.push('/create')"
          class="bg-indigo-600 text-white px-4 py-1.5 rounded-lg text-sm font-medium hover:bg-indigo-700 transition-colors shadow-sm">
          + 发帖
        </button>
      </div>

      <!-- 骨架屏 -->
      <div v-if="loading" class="space-y-3">
        <div v-for="i in 5" :key="i" class="bg-white rounded-xl shadow-sm border border-gray-200 p-5 animate-pulse">
          <div class="h-5 bg-gray-200 rounded w-3/4 mb-3"></div>
          <div class="h-4 bg-gray-100 rounded w-1/2"></div>
        </div>
      </div>

      <!-- 帖子列表（动态组件：根据主题配置切换 classic / waterfall） -->
      <div v-else>
        <component :is="layoutComp" :threads="threads" />
      </div>
    </div>

    <!-- ========== 右侧：用户信息 + 标签云 ========== -->
    <aside class="w-64 shrink-0 hidden lg:block">
      <div class="space-y-4 sticky top-20">
        <!-- 用户信息卡片 -->
        <div class="bg-white rounded-xl shadow-sm border border-gray-200 p-5">
          <template v-if="userStore.user">
            <div class="flex items-center gap-3 mb-3">
              <span class="w-10 h-10 rounded-full bg-indigo-100 flex items-center justify-center text-sm font-bold text-indigo-700">
                {{ userStore.user.username.charAt(0).toUpperCase() }}
              </span>
              <div>
                <p class="text-sm font-semibold text-gray-900">{{ userStore.user.username }}</p>
                <p class="text-xs text-gray-400">欢迎回来</p>
              </div>
            </div>
            <div class="flex gap-3 text-xs text-gray-500 border-t border-gray-100 pt-3">
              <router-link to="/my" class="hover:text-indigo-600 transition-colors">个人中心</router-link>
              <router-link v-if="userStore.user?.gid === 1" to="/admin" class="hover:text-indigo-600 transition-colors">后台管理</router-link>
            </div>
          </template>
          <template v-else>
            <p class="text-sm text-gray-600 mb-3">登录后体验更多功能</p>
            <div class="flex gap-2">
              <router-link to="/login"
                class="flex-1 text-center text-sm bg-indigo-600 text-white px-3 py-1.5 rounded-lg hover:bg-indigo-700 transition-colors">
                登录
              </router-link>
              <router-link to="/register"
                class="flex-1 text-center text-sm border border-gray-300 text-gray-700 px-3 py-1.5 rounded-lg hover:bg-gray-50 transition-colors">
                注册
              </router-link>
            </div>
          </template>
        </div>

        <!-- 标签云卡片 -->
        <div class="bg-white rounded-xl shadow-sm border border-gray-200 p-4">
          <div class="flex items-center justify-between mb-3">
            <h3 class="text-xs font-semibold text-gray-400 uppercase tracking-wider">热门标签</h3>
            <router-link to="/tags" class="text-xs text-indigo-500 hover:text-indigo-700 transition-colors">更多</router-link>
          </div>
          <div class="flex flex-wrap items-center gap-x-2 gap-y-2.5 leading-none">
            <span v-for="tag in tags.slice(0, 12)" :key="tag.tagid"
              @click="goTag(tag.tagid)"
              class="inline-block cursor-pointer transition-all hover:opacity-80 hover:scale-105"
              :style="tagStyle(tag)">
              {{ tag.name }}
            </span>
            <div v-if="tags.length === 0" class="text-xs text-gray-400 py-2 w-full text-center">
              暂无标签
            </div>
          </div>
        </div>
      </div>
    </aside>
  </div>
</template>
