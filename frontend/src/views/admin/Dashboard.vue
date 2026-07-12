<!-- xiuno-go v2.1.0-beta 尼克修改版 -->
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import request from '../../utils/request'

interface Stats {
  threads: number
  posts: number
  users: number
  forums: number
  tags: number
  today_threads: number
  today_posts: number
  today_users: number
}

const stats = ref<Stats>({
  threads: 0, posts: 0, users: 0, forums: 0, tags: 0,
  today_threads: 0, today_posts: 0, today_users: 0,
})
const loading = ref(true)

const loadStats = async () => {
  loading.value = true
  try {
    // 并发获取各统计数据
    const [threadRes, forumRes, tagRes, userRes] = await Promise.all([
      request.get<any>('/thread', { params: { page: 1, pageSize: 1 } }),
      request.get<any>('/forum'),
      request.get<any>('/tag', { params: { page: 1, pageSize: 1 } }),
      request.get<any>('/admin/user', { params: { page: 1 } }),
    ])

    stats.value = {
      threads: threadRes.total || 0,
      posts: 0, // 暂无公开的 post count API
      users: userRes.users?.length || 0,
      forums: Array.isArray(forumRes) ? forumRes.length : 0,
      tags: tagRes.total || 0,
      today_threads: 0,
      today_posts: 0,
      today_users: 0,
    }
  } catch {
    // 静默失败，保留默认值
  } finally {
    loading.value = false
  }
}

onMounted(() => { loadStats() })
</script>

<template>
  <div>
    <h2 class="text-2xl font-bold mb-6">控制台概览</h2>

    <div v-if="loading" class="animate-pulse">
      <div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
        <div v-for="i in 5" :key="i" class="h-28 bg-gray-200 rounded-xl"></div>
      </div>
    </div>

    <div v-else>
      <!-- 统计卡片 -->
      <div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
        <div class="bg-gradient-to-br from-blue-500 to-blue-600 rounded-xl p-5 text-white shadow-md">
          <div class="text-3xl font-bold">{{ stats.threads }}</div>
          <div class="text-sm text-blue-100 mt-1">主题总数</div>
        </div>
        <div class="bg-gradient-to-br from-emerald-500 to-emerald-600 rounded-xl p-5 text-white shadow-md">
          <div class="text-3xl font-bold">{{ stats.posts }}</div>
          <div class="text-sm text-emerald-100 mt-1">回复总数</div>
        </div>
        <div class="bg-gradient-to-br from-purple-500 to-purple-600 rounded-xl p-5 text-white shadow-md">
          <div class="text-3xl font-bold">{{ stats.users }}</div>
          <div class="text-sm text-purple-100 mt-1">注册用户</div>
        </div>
        <div class="bg-gradient-to-br from-amber-500 to-amber-600 rounded-xl p-5 text-white shadow-md">
          <div class="text-3xl font-bold">{{ stats.forums }}</div>
          <div class="text-sm text-amber-100 mt-1">版块数量</div>
        </div>
        <div class="bg-gradient-to-br from-rose-500 to-rose-600 rounded-xl p-5 text-white shadow-md">
          <div class="text-3xl font-bold">{{ stats.tags }}</div>
          <div class="text-sm text-rose-100 mt-1">标签数量</div>
        </div>
      </div>

      <!-- 快捷入口 -->
      <div class="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
        <h3 class="text-lg font-bold text-gray-800 mb-4">快捷操作</h3>
        <div class="grid grid-cols-2 md:grid-cols-3 gap-3">
          <router-link to="/admin/forum"
            class="flex items-center gap-3 p-3 bg-gray-50 rounded-lg hover:bg-blue-50 hover:text-blue-700 transition text-sm text-gray-700">
            📁 管理版块
          </router-link>
          <router-link to="/admin/tag"
            class="flex items-center gap-3 p-3 bg-gray-50 rounded-lg hover:bg-blue-50 hover:text-blue-700 transition text-sm text-gray-700">
            🏷️ 管理标签
          </router-link>
          <router-link to="/admin/thread"
            class="flex items-center gap-3 p-3 bg-gray-50 rounded-lg hover:bg-blue-50 hover:text-blue-700 transition text-sm text-gray-700">
            📝 管理主题
          </router-link>
          <router-link to="/admin/user"
            class="flex items-center gap-3 p-3 bg-gray-50 rounded-lg hover:bg-blue-50 hover:text-blue-700 transition text-sm text-gray-700">
            👥 管理用户
          </router-link>
          <router-link to="/admin/group"
            class="flex items-center gap-3 p-3 bg-gray-50 rounded-lg hover:bg-blue-50 hover:text-blue-700 transition text-sm text-gray-700">
            👤 用户组设置
          </router-link>
          <router-link to="/admin/config"
            class="flex items-center gap-3 p-3 bg-gray-50 rounded-lg hover:bg-blue-50 hover:text-blue-700 transition text-sm text-gray-700">
            ⚙️ 站点配置
          </router-link>
        </div>
      </div>
    </div>
  </div>
</template>
