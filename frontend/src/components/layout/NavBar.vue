<!-- xiuno-go v2.1.0-beta 尼克修改版 -->
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useUserStore } from '../../stores/user'
import { useRouter } from 'vue-router'
import request from '../../utils/request'

interface Forum {
  fid: number
  name: string
}

const userStore = useUserStore()
const router = useRouter()
const showDropdown = ref(false)
const showForumList = ref(false)
const forums = ref<Forum[]>([])

onMounted(async () => {
  try {
    const data: any = await request.get('/forum')
    forums.value = data || []
  } catch {
    forums.value = []
  }
})

function goLogin() {
  router.push('/login')
}

function goRegister() {
  router.push('/register')
}

function goHome() {
  router.push('/threads')
}

function goMy() {
  showDropdown.value = false
  router.push('/my')
}

function goAdmin() {
  showDropdown.value = false
  router.push('/admin')
}

async function logout() {
  showDropdown.value = false
  try {
    await fetch('/api/v1/user/logout')
  } catch {}
  userStore.logout()
  router.push('/')
}

function toggleDropdown() {
  showDropdown.value = !showDropdown.value
}

function closeDropdown() {
  showDropdown.value = false
}

function toggleForumList() {
  showForumList.value = !showForumList.value
}

function goForum(fid: number) {
  showForumList.value = false
  router.push(`/forum/${fid}`)
}
</script>

<template>
  <header class="bg-white shadow-sm border-b border-gray-200 sticky top-0 z-50">
    <div class="max-w-4xl mx-auto px-4 h-14 flex items-center justify-between">
      <div class="flex items-center gap-6">
        <a href="#" @click.prevent="goHome" class="text-lg font-bold text-indigo-600 hover:text-indigo-700">
          Xiuno
        </a>
        <nav class="hidden sm:flex gap-4 text-sm text-gray-600">
          <a href="#" @click.prevent="goHome" class="hover:text-indigo-600 transition-colors">社区</a>
          <!-- 版块导航 -->
          <div class="relative">
            <button @click="toggleForumList"
              class="hover:text-indigo-600 transition-colors cursor-pointer flex items-center gap-1">
              版块
              <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
              </svg>
            </button>
            <!-- 版块下拉列表 -->
            <div v-if="showForumList" @click="closeDropdown" class="fixed inset-0 z-10" />
            <div v-if="showForumList"
              class="absolute left-0 mt-2 w-44 bg-white rounded-xl shadow-lg border border-gray-200 py-1.5 z-20">
              <button v-for="f in forums" :key="f.fid" @click="goForum(f.fid)"
                class="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 transition-colors">
                {{ f.name }}
              </button>
              <div v-if="forums.length === 0" class="px-4 py-2 text-sm text-gray-400">暂无版块</div>
            </div>
          </div>
          <!-- 标签页链接 -->
          <a href="#" @click.prevent="router.push('/tags')" class="hover:text-indigo-600 transition-colors">标签</a>
        </nav>
      </div>
      <div class="flex items-center gap-3">
        <template v-if="userStore.user">
          <div class="relative">
            <button @click="toggleDropdown"
              class="flex items-center gap-1.5 text-sm text-gray-700 hover:text-indigo-600 transition-colors cursor-pointer">
              <span class="w-7 h-7 rounded-full bg-indigo-100 flex items-center justify-center text-xs font-medium text-indigo-700">
                {{ userStore.user.username.charAt(0).toUpperCase() }}
              </span>
              <span class="hidden sm:inline">{{ userStore.user.username }}</span>
              <svg class="w-3.5 h-3.5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
              </svg>
            </button>
            <!-- 下拉菜单 -->
            <div v-if="showDropdown" @click="closeDropdown"
              class="fixed inset-0 z-10" />
            <div v-if="showDropdown"
              class="absolute right-0 mt-2 w-44 bg-white rounded-xl shadow-lg border border-gray-200 py-1.5 z-20">
              <button @click="goMy" class="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 transition-colors flex items-center gap-2">
                <span>👤</span> 个人中心
              </button>
              <button v-if="userStore.user?.gid === 1" @click="goAdmin" class="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 transition-colors flex items-center gap-2">
                <span>⚙️</span> 后台管理
              </button>
              <hr class="my-1 border-gray-100" />
              <button @click="logout" class="w-full text-left px-4 py-2 text-sm text-red-600 hover:bg-red-50 transition-colors flex items-center gap-2">
                <span>🚪</span> 退出登录
              </button>
            </div>
          </div>
        </template>
        <template v-else>
          <button @click="goLogin" class="text-sm text-gray-600 hover:text-indigo-600 transition-colors">登录</button>
          <button @click="goRegister" class="text-sm bg-indigo-600 text-white px-4 py-1.5 rounded-lg hover:bg-indigo-700 transition-colors">注册</button>
        </template>
      </div>
    </div>
  </header>
</template>
