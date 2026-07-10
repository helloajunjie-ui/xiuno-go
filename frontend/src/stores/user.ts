import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import request from '../utils/request'

export interface User {
  uid: number
  username: string
  avatar: number
  gid: number
  email?: string
  create_date?: number
  threads?: number
  posts?: number
  credits?: number
  login_date?: number
  logins?: number
}

export const useUserStore = defineStore('user', () => {
  const user = ref<User | null>(null)

  const isLoggedIn = computed(() => user.value !== null)

  async function login(account: string, password: string) {
    const data = await request.post('/user/login', { account, password })
    user.value = data as unknown as User
    return user.value
  }

  async function register(username: string, email: string, password: string) {
    const data = await request.post('/user/register', { username, email, password })
    user.value = data as unknown as User
    return user.value
  }

  async function fetchProfile() {
    try {
      const data = await request.get('/my/profile')
      user.value = data as unknown as User
    } catch {
      user.value = null
    }
  }

  function updateAvatar(avatar: number) {
    if (user.value) {
      user.value.avatar = avatar
    }
  }

  function logout() {
    user.value = null
  }

  return { user, isLoggedIn, login, register, fetchProfile, updateAvatar, logout }
})
