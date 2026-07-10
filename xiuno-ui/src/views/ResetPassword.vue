<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import request from '../utils/request'

const router = useRouter()

// Step 1: 发送验证码
const email = ref('')
const step = ref<'email' | 'reset'>('email')
const loading = ref(false)
const error = ref('')
const successMsg = ref('')
const expireSec = ref(0)
const countdown = ref(0)
let countdownTimer: ReturnType<typeof setInterval> | null = null

// Step 2: 重置密码
const code = ref('')
const password = ref('')
const confirmPassword = ref('')

function startCountdown(sec: number) {
  countdown.value = sec
  if (countdownTimer) clearInterval(countdownTimer)
  countdownTimer = setInterval(() => {
    countdown.value--
    if (countdown.value <= 0) {
      if (countdownTimer) clearInterval(countdownTimer)
      countdownTimer = null
    }
  }, 1000)
}

async function handleSendCode() {
  if (!email.value) {
    error.value = '请输入邮箱'
    return
  }
  loading.value = true
  error.value = ''
  successMsg.value = ''
  try {
    const resp: any = await request.post('/user/send-code', {
      email: email.value,
      scene: 'resetpw',
    })
    expireSec.value = resp.expire_sec || 600
    successMsg.value = `验证码已发送到 ${email.value}，请查收邮件`
    startCountdown(expireSec.value)
    step.value = 'reset'
  } catch (e: any) {
    error.value = e.message || '发送验证码失败'
  } finally {
    loading.value = false
  }
}

async function handleReset() {
  if (!code.value || !password.value || !confirmPassword.value) {
    error.value = '请填写所有字段'
    return
  }
  if (password.value !== confirmPassword.value) {
    error.value = '两次密码输入不一致'
    return
  }
  if (password.value.length < 6) {
    error.value = '密码长度不能少于6位'
    return
  }
  loading.value = true
  error.value = ''
  try {
    await request.post('/user/reset-password', {
      email: email.value,
      code: code.value,
      password: password.value,
    })
    // 重置成功，跳转到登录页
    router.push('/login?reset=1')
  } catch (e: any) {
    error.value = e.message || '密码重置失败'
  } finally {
    loading.value = false
  }
}

function backToEmail() {
  step.value = 'email'
  error.value = ''
  successMsg.value = ''
}
</script>

<template>
  <div class="flex justify-center pt-16">
    <div class="w-full max-w-sm">
      <h1 class="text-2xl font-bold text-center mb-8">重置密码</h1>

      <!-- Step 1: 输入邮箱 -->
      <div v-if="step === 'email'">
        <form @submit.prevent="handleSendCode" class="bg-white rounded-xl shadow-sm border border-gray-200 p-6 space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">注册邮箱</label>
            <input v-model="email" type="email" placeholder="请输入注册时使用的邮箱"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent" />
          </div>
          <p v-if="error" class="text-red-500 text-sm">{{ error }}</p>
          <button type="submit" :disabled="loading"
            class="w-full bg-indigo-600 text-white py-2 rounded-lg hover:bg-indigo-700 disabled:opacity-50 transition-colors">
            {{ loading ? '发送中...' : '发送验证码' }}
          </button>
          <p class="text-center text-sm text-gray-500">
            想起密码了？
            <router-link to="/login" class="text-indigo-600 hover:underline">返回登录</router-link>
          </p>
        </form>
      </div>

      <!-- Step 2: 输入验证码 + 新密码 -->
      <div v-else>
        <div class="bg-green-50 border border-green-200 rounded-lg p-3 mb-4 text-sm text-green-700">
          {{ successMsg }}
        </div>
        <form @submit.prevent="handleReset" class="bg-white rounded-xl shadow-sm border border-gray-200 p-6 space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">邮箱</label>
            <input :value="email" type="email" disabled
              class="w-full px-3 py-2 border border-gray-200 rounded-lg bg-gray-50 text-gray-500 cursor-not-allowed" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">验证码</label>
            <div class="flex gap-2">
              <input v-model="code" type="text" placeholder="6 位验证码" maxlength="6"
                class="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent" />
              <button type="button" @click="handleSendCode" :disabled="countdown > 0"
                class="px-3 py-2 text-sm border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors whitespace-nowrap">
                {{ countdown > 0 ? `${countdown}s` : '重新发送' }}
              </button>
            </div>
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">新密码</label>
            <input v-model="password" type="password" placeholder="至少 6 位"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">确认新密码</label>
            <input v-model="confirmPassword" type="password" placeholder="再次输入新密码"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent" />
          </div>
          <p v-if="error" class="text-red-500 text-sm">{{ error }}</p>
          <button type="submit" :disabled="loading"
            class="w-full bg-indigo-600 text-white py-2 rounded-lg hover:bg-indigo-700 disabled:opacity-50 transition-colors">
            {{ loading ? '重置中...' : '重置密码' }}
          </button>
          <p class="text-center text-sm text-gray-500">
            <button type="button" @click="backToEmail" class="text-indigo-600 hover:underline bg-transparent border-none cursor-pointer">
              更换邮箱
            </button>
          </p>
        </form>
      </div>
    </div>
  </div>
</template>
