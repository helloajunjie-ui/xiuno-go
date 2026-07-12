<!-- xiuno-go v2.1.0-beta 尼克修改版 -->
<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '../stores/user'
import request from '../utils/request'

const router = useRouter()
const userStore = useUserStore()

const oldPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const submitting = ref(false)
const errorMsg = ref('')
const successMsg = ref('')

const isLoggedIn = computed(() => userStore.isLoggedIn)

// 未登录直接跳转
if (!isLoggedIn.value) {
  router.push('/login')
}

async function submit() {
  errorMsg.value = ''
  successMsg.value = ''

  if (!oldPassword.value) {
    errorMsg.value = '请输入旧密码'
    return
  }
  if (!newPassword.value) {
    errorMsg.value = '请输入新密码'
    return
  }
  if (newPassword.value.length < 6) {
    errorMsg.value = '新密码不能少于 6 位'
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    errorMsg.value = '两次输入的新密码不一致'
    return
  }

  submitting.value = true
  try {
    await request.put('/user/password', {
      old_password: oldPassword.value,
      new_password: newPassword.value,
    })
    successMsg.value = '密码修改成功，请重新登录'
    // 3 秒后跳转到登录页
    setTimeout(() => {
      userStore.logout()
      router.push('/login')
    }, 3000)
  } catch (e: any) {
    errorMsg.value = e.message || '密码修改失败'
  } finally {
    submitting.value = false
  }
}

function goBack() {
  router.push('/my')
}
</script>

<template>
  <div class="max-w-md mx-auto">
    <div class="flex items-center gap-3 mb-6">
      <button @click="goBack"
        class="text-gray-400 hover:text-gray-600 transition-colors">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
      </button>
      <h1 class="text-xl font-bold text-gray-900">修改密码</h1>
    </div>

    <div class="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
      <!-- 成功提示 -->
      <div v-if="successMsg"
        class="mb-4 p-3 bg-green-50 border border-green-200 rounded-lg text-sm text-green-700">
        {{ successMsg }}
      </div>

      <!-- 错误提示 -->
      <div v-if="errorMsg"
        class="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-600">
        {{ errorMsg }}
      </div>

      <form @submit.prevent="submit" class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">旧密码</label>
          <input v-model="oldPassword" type="password" required
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 outline-none transition-colors"
            placeholder="输入当前密码" />
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">新密码</label>
          <input v-model="newPassword" type="password" required minlength="6"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 outline-none transition-colors"
            placeholder="至少 6 位字符" />
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">确认新密码</label>
          <input v-model="confirmPassword" type="password" required
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 outline-none transition-colors"
            placeholder="再次输入新密码" />
        </div>

        <button type="submit" :disabled="submitting"
          class="w-full py-2.5 bg-indigo-600 hover:bg-indigo-700 disabled:bg-indigo-400 text-white font-medium rounded-lg transition-colors">
          {{ submitting ? '提交中...' : '更新密码' }}
        </button>
      </form>
    </div>
  </div>
</template>
