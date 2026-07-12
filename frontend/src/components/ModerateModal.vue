<!-- xiuno-go v2.1.0-beta 尼克修改版 -->
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import request from '../utils/request'

const props = defineProps<{
  /** 操作类型: top / close / delete / move */
  action: string
  /** 选中的帖子 ID 列表 */
  tidarr: number[]
  /** 回调：操作完成后触发 */
  onDone?: () => void
}>()

const emit = defineEmits<{
  close: []
}>()

const topValue = ref(0)
const closeValue = ref(1)
const newFid = ref(0)
const forumList = ref<{ fid: number; name: string }[]>([])
const submitting = ref(false)
const errorMsg = ref('')

const actionLabels: Record<string, string> = {
  top: '置顶',
  close: '关闭/打开',
  delete: '删除',
  move: '移动',
}

onMounted(async () => {
  if (props.action === 'move') {
    try {
      const res: any = await request.get('/forum')
      forumList.value = res || []
    } catch {
      // ignore
    }
  }
})

async function handleSubmit() {
  submitting.value = true
  errorMsg.value = ''

  try {
    if (props.action === 'delete') {
      // 批量删除：逐个调用 DELETE /thread/{tid}
      for (const tid of props.tidarr) {
        await request.delete(`/thread/${tid}`)
      }
    } else if (props.action === 'top') {
      // 批量置顶：逐个调用 POST /thread/{tid}/moderate
      for (const tid of props.tidarr) {
        await request.post(`/thread/${tid}/moderate`, { action: 'top', value: topValue.value })
      }
    } else if (props.action === 'close') {
      for (const tid of props.tidarr) {
        await request.post(`/thread/${tid}/moderate`, { action: 'close', value: closeValue.value })
      }
    } else if (props.action === 'move') {
      if (!newFid.value) {
        errorMsg.value = '请选择目标版块'
        submitting.value = false
        return
      }
      for (const tid of props.tidarr) {
        await request.post(`/thread/${tid}/move`, { new_fid: newFid.value })
      }
    }
    props.onDone?.()
    emit('close')
  } catch (e: any) {
    errorMsg.value = e.message || '操作失败'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40" @click.self="emit('close')">
    <div class="bg-white rounded-xl shadow-xl border border-gray-200 w-full max-w-md mx-4 overflow-hidden">
      <!-- 头部 -->
      <div class="flex items-center justify-between px-5 py-4 border-b border-gray-100">
        <h3 class="text-lg font-semibold text-gray-900">
          {{ actionLabels[action] || '版务操作' }}
        </h3>
        <button @click="emit('close')" class="text-gray-400 hover:text-gray-600 text-xl leading-none">&times;</button>
      </div>

      <!-- 主体 -->
      <div class="px-5 py-4 space-y-4">
        <div class="text-sm text-gray-600">
          已选择 <span class="font-bold text-red-500">{{ tidarr.length }}</span> 篇帖子
        </div>

        <!-- 置顶选项 -->
        <div v-if="action === 'top'" class="space-y-2">
          <label class="flex items-center gap-2 text-sm">
            <input type="radio" v-model="topValue" :value="0" class="accent-indigo-600" />
            不置顶
          </label>
          <label class="flex items-center gap-2 text-sm">
            <input type="radio" v-model="topValue" :value="1" class="accent-indigo-600" />
            本版置顶
          </label>
          <label class="flex items-center gap-2 text-sm">
            <input type="radio" v-model="topValue" :value="3" class="accent-indigo-600" />
            全局置顶
          </label>
        </div>

        <!-- 关闭/打开选项 -->
        <div v-if="action === 'close'" class="space-y-2">
          <label class="flex items-center gap-2 text-sm">
            <input type="radio" v-model="closeValue" :value="1" class="accent-indigo-600" />
            关闭（禁止回复）
          </label>
          <label class="flex items-center gap-2 text-sm">
            <input type="radio" v-model="closeValue" :value="0" class="accent-indigo-600" />
            打开（允许回复）
          </label>
        </div>

        <!-- 删除确认 -->
        <div v-if="action === 'delete'" class="text-sm text-red-600 bg-red-50 rounded-lg px-3 py-2">
          确定要删除选中的 {{ tidarr.length }} 篇帖子吗？此操作不可撤销。
        </div>

        <!-- 移动版块选择 -->
        <div v-if="action === 'move'" class="space-y-2">
          <label class="text-sm text-gray-700">目标版块</label>
          <select v-model="newFid"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500">
            <option :value="0" disabled>请选择版块</option>
            <option v-for="f in forumList" :key="f.fid" :value="f.fid">{{ f.name }}</option>
          </select>
        </div>

        <!-- 错误提示 -->
        <div v-if="errorMsg" class="text-sm text-red-600 bg-red-50 rounded-lg px-3 py-2">
          {{ errorMsg }}
        </div>
      </div>

      <!-- 底部按钮 -->
      <div class="flex justify-end gap-3 px-5 py-4 border-t border-gray-100 bg-gray-50">
        <button @click="emit('close')"
          class="px-4 py-2 text-sm text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors">
          取消
        </button>
        <button @click="handleSubmit" :disabled="submitting"
          class="px-4 py-2 text-sm text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 disabled:opacity-50 transition-colors">
          {{ submitting ? '处理中...' : '确认' }}
        </button>
      </div>
    </div>
  </div>
</template>
