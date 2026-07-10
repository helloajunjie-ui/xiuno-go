<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '../stores/user'

const router = useRouter()
const userStore = useUserStore()

const account = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function handleLogin() {
  if (!account.value || !password.value) {
    error.value = '请输入账号和密码'
    return
  }
  loading.value = true
  error.value = ''
  try {
    await userStore.login(account.value, password.value)
    router.push('/threads')
  } catch (e: any) {
    error.value = e.message || '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex justify-center pt-16">
    <div class="w-full max-w-sm">
      <h1 class="text-2xl font-bold text-center mb-8">登录 Xiuno</h1>
      <form @submit.prevent="handleLogin" class="bg-white rounded-xl shadow-sm border border-gray-200 p-6 space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">账号</label>
          <input v-model="account" type="text" placeholder="用户名或邮箱"
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
          {{ loading ? '登录中...' : '登录' }}
        </button>
        <p class="text-center text-sm text-gray-500">
          还没有账号？
          <router-link to="/register" class="text-indigo-600 hover:underline">注册</router-link>
        </p>
        <p class="text-center text-sm text-gray-500">
          <router-link to="/resetpw" class="text-gray-400 hover:text-indigo-600 transition-colors">忘记密码？</router-link>
        </p>
      </form>
    </div>
  </div>
</template>
