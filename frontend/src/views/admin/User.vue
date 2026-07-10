<script setup lang="ts">
import { ref, onMounted } from 'vue'
import request from '../../utils/request'

interface AdminUser {
  uid: number
  username: string
  gid: number
  email?: string
  create_date?: number
  banned?: number
}

const users = ref<AdminUser[]>([])
const loading = ref(true)
const keyword = ref('')

const loadUsers = async () => {
  try {
    const params: Record<string, string> = {}
    if (keyword.value.trim()) {
      params.keyword = keyword.value.trim()
    }
    const res: any = await request.get('/admin/user', { params })
    users.value = (res.users || res) as AdminUser[]
  } finally {
    loading.value = false
  }
}

const banUser = async (uid: number, banned: number) => {
  const action = banned ? '解封' : '封禁'
  if (!confirm(`确定要${action}该用户？`)) return
  try {
    // 封禁 → gid=7(封禁组)，解封 → gid=101(普通用户)
    const gid = banned ? 101 : 7
    await request.put(`/admin/user/${uid}/group`, { gid })
    const user = users.value.find(u => u.uid === uid)
    if (user) user.banned = banned ? 0 : 1
    alert(`用户已${action}`)
  } catch {
    alert('操作失败')
  }
}

onMounted(() => { loadUsers() })
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h2 class="text-2xl font-bold">用户管控</h2>
      <div class="flex space-x-2">
        <input
          v-model="keyword"
          type="text"
          placeholder="搜索用户名..."
          class="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          @keyup.enter="loadUsers"
        />
        <button @click="loadUsers" class="bg-gray-100 text-gray-700 px-4 py-2 rounded-lg hover:bg-gray-200 transition text-sm">搜索</button>
      </div>
    </div>

    <div v-if="loading" class="animate-pulse space-y-4">
      <div class="h-12 bg-gray-200 rounded w-full"></div>
      <div class="h-12 bg-gray-200 rounded w-full"></div>
    </div>

    <div v-else class="overflow-x-auto">
      <table class="w-full text-left">
        <thead>
          <tr class="border-b border-gray-200 text-sm text-gray-500">
            <th class="pb-3 font-medium">UID</th>
            <th class="pb-3 font-medium">用户名</th>
            <th class="pb-3 font-medium">GID</th>
            <th class="pb-3 font-medium">Email</th>
            <th class="pb-3 font-medium">状态</th>
            <th class="pb-3 font-medium">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in users" :key="u.uid" class="border-b border-gray-100 hover:bg-gray-50">
            <td class="py-3 text-sm font-mono text-gray-500">{{ u.uid }}</td>
            <td class="py-3 font-medium text-gray-800">{{ u.username }}</td>
            <td class="py-3 text-sm text-gray-500">{{ u.gid }}</td>
            <td class="py-3 text-sm text-gray-500">{{ u.email || '-' }}</td>
            <td class="py-3">
              <span
                :class="u.banned ? 'bg-red-100 text-red-700' : 'bg-green-100 text-green-700'"
                class="text-xs px-2 py-1 rounded-full"
              >
                {{ u.banned ? '已封禁' : '正常' }}
              </span>
            </td>
            <td class="py-3">
              <button
                @click="banUser(u.uid, u.banned ?? 0)"
                :class="u.banned ? 'text-green-600 hover:text-green-800' : 'text-red-500 hover:text-red-700'"
                class="text-sm"
              >
                {{ u.banned ? '解封' : '封禁' }}
              </button>
            </td>
          </tr>
          <tr v-if="users.length === 0">
            <td colspan="6" class="text-center text-gray-400 py-10">暂无用户数据</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
