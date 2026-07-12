<!-- xiuno-go v2.1.0-beta 尼克修改版 -->
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useThemeStore, type SiteTheme } from '../../stores/theme'
import request from '../../utils/request'

const themeStore = useThemeStore()

// 预设主题
const presets = [
  { name: '默认靛蓝', theme: { primary_color: '#4f46e5', bg_color: '#f9fafb', card_radius: '0.75rem', list_layout: 'classic', theme_mode: 'light', custom_css: '' } },
  { name: '翡翠绿', theme: { primary_color: '#059669', bg_color: '#f0fdf4', card_radius: '0.5rem', list_layout: 'classic', theme_mode: 'light', custom_css: '' } },
  { name: '玫瑰红', theme: { primary_color: '#e11d48', bg_color: '#fff1f2', card_radius: '1rem', list_layout: 'classic', theme_mode: 'light', custom_css: '' } },
  { name: '暗夜紫', theme: { primary_color: '#7c3aed', bg_color: '#1e1e2e', card_radius: '0.75rem', list_layout: 'waterfall', theme_mode: 'dark', custom_css: '' } },
  { name: '深海蓝', theme: { primary_color: '#2563eb', bg_color: '#0f172a', card_radius: '0.5rem', list_layout: 'waterfall', theme_mode: 'dark', custom_css: '' } },
  { name: '暖阳橙', theme: { primary_color: '#ea580c', bg_color: '#fff7ed', card_radius: '0.75rem', list_layout: 'classic', theme_mode: 'light', custom_css: '' } },
]

const form = ref<SiteTheme>({
  primary_color: '#4f46e5',
  bg_color: '#f9fafb',
  card_radius: '0.75rem',
  list_layout: 'classic',
  theme_mode: 'light',
  custom_css: '',
})

const saving = ref(false)
const message = ref('')

onMounted(() => {
  if (themeStore.config) {
    form.value = { ...themeStore.config }
  }
})

function applyPreset(preset: typeof presets[0]) {
  form.value = { ...preset.theme }
  themeStore.previewTheme(form.value)
}

function preview() {
  themeStore.previewTheme(form.value)
}

function resetPreview() {
  if (themeStore.config) {
    form.value = { ...themeStore.config }
    themeStore.resetPreview()
  }
}

async function save() {
  saving.value = true
  message.value = ''
  try {
    await request.put('/admin/theme', form.value)
    // 更新 store
    themeStore.config = { ...form.value }
    themeStore.applyTheme()
    message.value = '主题已保存并生效'
  } catch (e: any) {
    message.value = '保存失败: ' + (e.message || '未知错误')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div>
    <h2 class="text-xl font-bold text-gray-900 mb-6">🎨 外观实验室</h2>

    <!-- 预设主题 -->
    <div class="mb-8">
      <h3 class="text-sm font-semibold text-gray-700 mb-3">预设主题</h3>
      <div class="flex flex-wrap gap-3">
        <button v-for="p in presets" :key="p.name"
          @click="applyPreset(p)"
          class="px-4 py-2 rounded-lg text-sm font-medium border transition-all hover:shadow-md"
          :class="form.primary_color === p.theme.primary_color && form.bg_color === p.theme.bg_color
            ? 'ring-2 ring-offset-2 border-transparent'
            : 'border-gray-200 hover:border-gray-300'"
          :style="{
            backgroundColor: p.theme.bg_color,
            color: p.theme.theme_mode === 'dark' ? '#fff' : '#374151',
            borderColor: p.theme.primary_color,
          }">
          {{ p.name }}
        </button>
      </div>
    </div>

    <!-- 自定义配置 -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">主色 (Primary Color)</label>
        <div class="flex items-center gap-2">
          <input type="color" v-model="form.primary_color"
            class="w-10 h-10 rounded cursor-pointer border border-gray-300" />
          <input type="text" v-model="form.primary_color"
            class="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm font-mono" />
        </div>
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">背景色 (Background)</label>
        <div class="flex items-center gap-2">
          <input type="color" v-model="form.bg_color"
            class="w-10 h-10 rounded cursor-pointer border border-gray-300" />
          <input type="text" v-model="form.bg_color"
            class="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm font-mono" />
        </div>
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">卡片圆角 (Card Radius)</label>
        <select v-model="form.card_radius"
          class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm">
          <option value="0.25rem">小 (0.25rem)</option>
          <option value="0.5rem">中 (0.5rem)</option>
          <option value="0.75rem">大 (0.75rem)</option>
          <option value="1rem">圆 (1rem)</option>
          <option value="1.5rem">超大 (1.5rem)</option>
        </select>
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">列表布局 (List Layout)</label>
        <select v-model="form.list_layout"
          class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm">
          <option value="classic">经典列表 (Classic)</option>
          <option value="waterfall">瀑布流 (Waterfall)</option>
        </select>
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">主题模式 (Theme Mode)</label>
        <select v-model="form.theme_mode"
          class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm">
          <option value="light">亮色 (Light)</option>
          <option value="dark">暗色 (Dark)</option>
        </select>
      </div>
    </div>

    <!-- 自定义 CSS -->
    <div class="mb-8">
      <label class="block text-sm font-medium text-gray-700 mb-1">自定义 CSS（高级）</label>
      <textarea v-model="form.custom_css" rows="6"
        class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm font-mono"
        placeholder="/* 在此输入自定义 CSS，例如： */&#10;.thread-card { box-shadow: 0 4px 6px rgba(0,0,0,0.1); }&#10;.nav-bar { backdrop-filter: blur(10px); }"></textarea>
      <p class="text-xs text-gray-400 mt-1" v-text="'将直接注入到页面 <head> 中，覆盖默认样式。'"></p>
    </div>

    <!-- 操作按钮 -->
    <div class="flex items-center gap-3">
      <button @click="preview"
        class="px-5 py-2 border border-gray-300 text-gray-700 rounded-lg text-sm font-medium hover:bg-gray-50 transition-colors">
        预览
      </button>
      <button @click="resetPreview"
        class="px-5 py-2 border border-gray-300 text-gray-700 rounded-lg text-sm font-medium hover:bg-gray-50 transition-colors">
        重置预览
      </button>
      <button @click="save" :disabled="saving"
        class="px-5 py-2 rounded-lg text-sm font-medium text-white transition-colors shadow-sm"
        :class="saving ? 'bg-gray-400 cursor-not-allowed' : 'bg-blue-600 hover:bg-blue-700'"
        :style="{ backgroundColor: saving ? undefined : form.primary_color }">
        {{ saving ? '保存中...' : '保存主题' }}
      </button>
    </div>

    <!-- 消息 -->
    <div v-if="message" class="mt-4 text-sm"
      :class="message.includes('失败') ? 'text-red-600' : 'text-green-600'">
      {{ message }}
    </div>
  </div>
</template>
