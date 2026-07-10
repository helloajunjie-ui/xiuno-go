<script setup lang="ts">
import { ref, onMounted } from 'vue'
import request from '../../utils/request'

interface SiteConfig {
  site_name: string
  site_brief: string
  [key: string]: unknown
}

const config = ref<SiteConfig>({
  site_name: '',
  site_brief: '',
})
const loading = ref(true)
const saving = ref(false)

const loadConfig = async () => {
  try {
    const res = await request.get('/config')
    config.value = res as SiteConfig
  } finally {
    loading.value = false
  }
}

const saveConfig = async () => {
  saving.value = true
  try {
    await request.put('/admin/config', config.value)
    alert('配置已保存')
  } catch {
    alert('保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(() => { loadConfig() })
</script>

<template>
  <div>
    <h2 class="text-2xl font-bold mb-6">全局配置</h2>

    <div v-if="loading" class="animate-pulse space-y-4">
      <div class="h-10 bg-gray-200 rounded w-full"></div>
      <div class="h-10 bg-gray-200 rounded w-full"></div>
    </div>

    <form v-else @submit.prevent="saveConfig" class="space-y-6">
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">站点名称</label>
        <input
          v-model="config.site_name"
          type="text"
          class="w-full border border-gray-300 rounded-lg px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          placeholder="Xiuno BBS"
        />
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">站点简介</label>
        <textarea
          v-model="config.site_brief"
          rows="3"
          class="w-full border border-gray-300 rounded-lg px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          placeholder="一个轻量级社区"
        ></textarea>
      </div>

      <div class="flex justify-end">
        <button
          type="submit"
          :disabled="saving"
          class="bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700 disabled:opacity-50 transition"
        >
          {{ saving ? '保存中...' : '保存配置' }}
        </button>
      </div>
    </form>
  </div>
</template>
