<!-- xiuno-go v2.1.0-beta 尼克修改版 -->
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import request from '../../utils/request'

interface ModLogItem {
  logid: number
  uid: number
  tid: number
  pid: number
  subject: string
  comment: string
  create_date: number
  action: string
  username: string
}

const list = ref<ModLogItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(true)
const actionFilter = ref('')

const actions = [
  { value: '', label: '全部' },
  { value: 'top', label: '置顶' },
  { value: 'untop', label: '取消置顶' },
  { value: 'close', label: '关闭' },
  { value: 'open', label: '打开' },
  { value: 'delete', label: '删除' },
  { value: 'move', label: '移动' },
]

const loadList = async () => {
  loading.value = true
  try {
    const params: any = { page: page.value }
    if (actionFilter.value) params.action = actionFilter.value
    const res = await request.get<{ list: ModLogItem[], total: number }>('/admin/modlog', { params })
    list.value = res.list
    total.value = res.total
  } finally {
    loading.value = false
  }
}

const formatDate = (ts: number) => {
  const d = new Date(ts * 1000)
  return d.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

const actionLabel = (a: string) => {
  const found = actions.find(x => x.value === a)
  return found ? found.label : a
}

const totalPages = () => Math.ceil(total.value / pageSize)

const goPage = (p: number) => {
  if (p < 1 || p > totalPages()) return
  page.value = p
  loadList()
}

const onFilterChange = () => {
  page.value = 1
  loadList()
}

onMounted(() => { loadList() })
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h2 class="text-2xl font-bold">版务日志</h2>
      <div class="flex items-center gap-2">
        <label class="text-sm text-gray-500">操作类型：</label>
        <select v-model="actionFilter" @change="onFilterChange"
          class="border border-gray-300 rounded-lg px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
          <option v-for="a in actions" :key="a.value" :value="a.value">{{ a.label }}</option>
        </select>
      </div>
    </div>

    <div v-if="loading" class="animate-pulse space-y-4">
      <div class="h-12 bg-gray-200 rounded w-full"></div>
      <div class="h-12 bg-gray-200 rounded w-full"></div>
    </div>

    <div v-else class="overflow-x-auto">
      <table class="w-full text-left">
        <thead>
          <tr class="border-b border-gray-200 text-sm text-gray-500">
            <th class="pb-3 font-medium">时间</th>
            <th class="pb-3 font-medium">操作人</th>
            <th class="pb-3 font-medium">操作</th>
            <th class="pb-3 font-medium">主题</th>
            <th class="pb-3 font-medium">备注</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in list" :key="item.logid" class="border-b border-gray-100 hover:bg-gray-50">
            <td class="py-3 text-sm text-gray-500 whitespace-nowrap">{{ formatDate(item.create_date) }}</td>
            <td class="py-3 text-sm text-gray-700">{{ item.username || '未知' }}</td>
            <td class="py-3">
              <span class="inline-block px-2 py-0.5 rounded text-xs font-medium"
                :class="{
                  'bg-blue-50 text-blue-600': item.action === 'top' || item.action === 'untop',
                  'bg-yellow-50 text-yellow-600': item.action === 'close' || item.action === 'open',
                  'bg-red-50 text-red-600': item.action === 'delete',
                  'bg-purple-50 text-purple-600': item.action === 'move',
                  'bg-gray-50 text-gray-600': !['top','untop','close','open','delete','move'].includes(item.action)
                }">
                {{ actionLabel(item.action) }}
              </span>
            </td>
            <td class="py-3 text-sm text-gray-700 max-w-xs truncate">{{ item.subject || '-' }}</td>
            <td class="py-3 text-sm text-gray-500 max-w-xs truncate">{{ item.comment || '-' }}</td>
          </tr>
          <tr v-if="list.length === 0">
            <td colspan="5" class="text-center text-gray-400 py-10">暂无版务日志</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 分页 -->
    <div v-if="totalPages() > 1" class="flex items-center justify-center gap-2 mt-6">
      <button @click="goPage(page - 1)" :disabled="page <= 1"
        class="px-3 py-1 text-sm border rounded hover:bg-gray-50 disabled:opacity-30 disabled:cursor-not-allowed">上一页</button>
      <span class="text-sm text-gray-500">第 {{ page }} / {{ totalPages() }} 页（共 {{ total }} 条）</span>
      <button @click="goPage(page + 1)" :disabled="page >= totalPages()"
        class="px-3 py-1 text-sm border rounded hover:bg-gray-50 disabled:opacity-30 disabled:cursor-not-allowed">下一页</button>
    </div>
  </div>
</template>
