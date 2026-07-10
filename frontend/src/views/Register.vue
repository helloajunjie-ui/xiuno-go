<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '../stores/user'

const router = useRouter()
const userStore = useUserStore()

const username = ref('')
const email = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function handleRegister() {
  if (!username.value || !email.value || !password.value) {
    error.value = '请填写所有字段'
    return
  }
  loading.value = true
  error.value = ''
  try {
    await userStore.register(username.value, email.value, password.value)
    router.push('/threads')
  } catch (e: any) {
    error.value = e.message || '注册失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex justify-center pt-16">
    <div class="w-full max-w-sm">
      <h1 class="text-2xl font-bold text-center mb-8">注册 Xiuno</h1>
      <form @submit.prevent="handleRegister" class="bg-white rounded-xl shadow-sm border border-gray-200 p-6 space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">用户名</label>
          <input v-model="username" type="text" placeholder="用户名"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent" />
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">邮箱</label>
          <input v-model="email" type="email" placeholder="邮箱"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent" />
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">密码</label>
          <input v-model="password" type="password" placeholder="密码"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent" />
        </div>
        <p v-if="error" class="text-red-500 text-sm">{{ error }}</p>
        <button type="submit" :disabled="loading"
          class="w-full bg-indigo-600 text-white py-2 rounded-lg hover:bg-indigo-700 disabled:opacity-50 transition-colors">
          {{ loading ? '注册中...' : '注册' }}
        </button>
        <p class="text-center text-sm text-gray-500">
          已有账号？
          <router-link to="/login" class="text-indigo-600 hover:underline">登录</router-link>
        </p>
      </form>
    </div>
  </div>
</template>
