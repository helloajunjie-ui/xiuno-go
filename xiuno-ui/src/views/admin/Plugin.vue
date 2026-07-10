<script setup lang="ts">
import { ref, onMounted } from 'vue'
import request from '../../utils/request'

interface PluginInfo {
  name: string
  title: string
  version: string
  desc: string
  active: boolean
}

const plugins = ref<PluginInfo[]>([])
const loading = ref(true)

const loadPlugins = async () => {
  try {
    const res = await request.get('/admin/plugin')
    plugins.value = res as PluginInfo[]
  } finally {
    loading.value = false
  }
}

const togglePlugin = async (plugin: PluginInfo) => {
  plugin.active = !plugin.active // 乐观更新 UI

  const activePlugins = plugins.value.filter(p => p.active).map(p => p.name)

  try {
    await request.put('/admin/plugin', { active_plugins: activePlugins })
    alert('插件状态已热更新')
  } catch (err) {
    plugin.active = !plugin.active // 失败则回滚 UI
    alert('更新失败')
  }
}

onMounted(() => { loadPlugins() })
</script>

<template>
  <div>
    <h2 class="text-2xl font-bold mb-6">插件中枢</h2>

    <div v-if="loading" class="animate-pulse flex space-x-4">
      <div class="flex-1 space-y-4 py-1">
        <div class="h-4 bg-gray-200 rounded w-3/4"></div>
        <div class="h-4 bg-gray-200 rounded w-1/2"></div>
      </div>
    </div>

    <div v-else class="space-y-4">
      <div v-for="p in plugins" :key="p.name" class="flex items-center justify-between p-5 border border-gray-100 rounded-lg hover:shadow-md transition">
        <div>
          <div class="flex items-center space-x-3">
            <h3 class="text-lg font-bold text-gray-800">{{ p.title }}</h3>
            <span class="text-xs bg-slate-100 text-slate-500 px-2 py-1 rounded">v{{ p.version }}</span>
          </div>
          <p class="text-sm text-gray-500 mt-1">{{ p.desc }}</p>
          <p class="text-xs text-gray-400 mt-1 font-mono">ID: {{ p.name }}</p>
        </div>

        <!-- Tailwind 风格的 Toggle 按钮 -->
        <button
          @click="togglePlugin(p)"
          :class="p.active ? 'bg-blue-600' : 'bg-gray-200'"
          class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none"
        >
          <span
            :class="p.active ? 'translate-x-5' : 'translate-x-0'"
            class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
          ></span>
        </button>
      </div>

      <div v-if="plugins.length === 0" class="text-center text-gray-400 py-10">
        系统内尚未编译入任何插件
      </div>
    </div>
  </div>
</template>
