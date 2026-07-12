<!-- xiuno-go v2.1.0-beta 尼克修改版 -->
<script setup lang="ts">
import { onMounted } from 'vue'
import { useUserStore } from './stores/user'
import { useThemeStore } from './stores/theme'
import { setAppReady } from './router'
import NavBar from './components/layout/NavBar.vue'

const userStore = useUserStore()
const themeStore = useThemeStore()

onMounted(async () => {
  // 并行初始化：用户信息 + 主题配置
  await Promise.all([
    userStore.fetchProfile(),
    themeStore.fetchTheme(),
  ])
  setAppReady()
})
</script>

<template>
  <div class="min-h-screen bg-gray-50 flex flex-col">
    <NavBar />
    <main class="max-w-6xl mx-auto px-4 py-6 flex-1 w-full">
      <router-view />
    </main>
    <footer class="text-center text-xs text-gray-400 py-4 border-t border-gray-200 bg-white">
      xiuno-go v2.1.0-beta 尼克修改版
    </footer>
  </div>
</template>
