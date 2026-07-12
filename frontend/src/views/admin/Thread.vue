<!-- xiuno-go v2.1.0-beta 尼克修改版 -->
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import request from '../../utils/request'

interface ThreadItem {
  tid: number
  fid: number
  subject: string
  username: string
  uid: number
  create_date: number
  last_date: number
  views: number
  posts: number
  closed: number
}

const threads = ref<ThreadItem[]>([])
const loading = ref(true)
const keyword = ref('')
const fidFilter = ref('')
const page = ref(1)
const pageSize = 20
const total = ref(0)
const queueid = ref(0)
const scanning = ref(false)
const operating = ref(false)
const actionMsg = ref('')

// 扫描主题
const scanThreads = async () => {
  scanning.value = true
  actionMsg.value = '正在扫描...'
  try {
    const res: any = await request.post('/admin/thread/scan', {
      keyword: keyword.value.trim(),
      fid: fidFilter.value ? parseInt(fidFilter.value) : 0,
      page: page.value,
    })
    queueid.value = res.queueid
    actionMsg.value = `扫描完成，匹配到 ${res.tids?.length || 0} 条`
    // 加载扫描结果
    await loadFound()
  } catch {
    actionMsg.value = '扫描失败'
  } finally {
    scanning.value = false
  }
}

// 查看扫描结果
const loadFound = async () => {
  if (!queueid.value) return
  loading.value = true
  try {
    const res: any = await request.get('/admin/thread/found', {
      params: { queueid: queueid.value, page: page.value }
    })
    threads.value = res.list as ThreadItem[]
    total.value = res.total
  } finally {
    loading.value = false
  }
}

// 批量操作
const batchOperation = async (action: string) => {
  const actionLabel = action === 'delete' ? '删除' : action === 'close' ? '关闭' : '打开'
  if (!confirm(`确定要批量${actionLabel}当前扫描结果中的主题吗？`)) return
  if (!queueid.value) {
    alert('请先扫描主题')
    return
  }
  operating.value = true
  actionMsg.value = `正在批量${actionLabel}...`
  try {
    const res: any = await request.post('/admin/thread/operation', {
      queueid: queueid.value,
      action,
      limit: 100,
    })
    actionMsg.value = `已${actionLabel} ${res.count} 条`
    await loadFound()
  } catch {
    actionMsg.value = `${actionLabel}失败`
  } finally {
    operating.value = false
  }
}

// 硬删除单个主题
const deleteThread = async (tid: number, subject: string) => {
  if (!confirm(`确定要永久删除主题「${subject}」吗？此操作不可恢复！`)) return
  try {
    await request.delete(`/admin/thread/${tid}`)
    threads.value = threads.value.filter(t => t.tid !== tid)
    actionMsg.value = `已删除 #${tid}`
  } catch {
    alert('删除失败')
  }
}

const formatDate = (ts: number) => {
  if (!ts) return '-'
  return new Date(ts * 1000).toLocaleString('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit'
  })
}

const totalPages = () => Math.ceil(total.value / pageSize)

const goPage = (p: number) => {
  if (p < 1 || p > totalPages()) return
  page.value = p
  scanThreads()
}

onMounted(() => { /* 初始不自动扫描 */ })
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h2 class="text-2xl font-bold">主题管理</h2>
    </div>

    <!-- 搜索条件 -->
    <div class="bg-gray-50 rounded-lg p-4 mb-6 space-y-3">
      <div class="flex items-center gap-4">
        <div class="flex-1">
          <label class="block text-xs text-gray-500 mb-1">关键词（标题）</label>
          <input v-model="keyword" type="text" placeholder="搜索标题关键词..."
            class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            @keyup.enter="scanThreads" />
        </div>
        <div class="w-32">
          <label class="block text-xs text-gray-500 mb-1">版块 FID</label>
          <input v-model.number="fidFilter" type="number" placeholder="留空全部"
            class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
        </div>
        <div class="pt-5">
          <button @click="scanThreads" :disabled="scanning"
            class="bg-blue-600 text-white px-5 py-2 rounded-lg hover:bg-blue-700 disabled:opacity-50 transition text-sm">
            {{ scanning ? '扫描中...' : '扫描主题' }}
          </button>
        </div>
      </div>

      <!-- 操作按钮 -->
      <div v-if="queueid" class="flex items-center gap-3 pt-2 border-t border-gray-200">
        <span class="text-xs text-gray-500">批量操作：</span>
        <button @click="batchOperation('delete')" :disabled="operating"
          class="bg-red-500 text-white px-4 py-1.5 rounded-lg hover:bg-red-600 disabled:opacity-50 text-xs">批量删除</button>
        <button @click="batchOperation('close')" :disabled="operating"
          class="bg-yellow-500 text-white px-4 py-1.5 rounded-lg hover:bg-yellow-600 disabled:opacity-50 text-xs">批量关闭</button>
        <button @click="batchOperation('open')" :disabled="operating"
          class="bg-green-500 text-white px-4 py-1.5 rounded-lg hover:bg-green-600 disabled:opacity-50 text-xs">批量打开</button>
        <span v-if="actionMsg" class="text-xs text-gray-500 ml-2">{{ actionMsg }}</span>
      </div>
    </div>

    <div v-if="loading" class="animate-pulse space-y-4">
      <div class="h-12 bg-gray-200 rounded w-full"></div>
      <div class="h-12 bg-gray-200 rounded w-full"></div>
    </div>

    <div v-else-if="threads.length === 0 && !loading" class="text-center text-gray-400 py-10">
      请先在上方输入搜索条件并点击「扫描主题」
    </div>

    <div v-else class="overflow-x-auto">
      <table class="w-full text-left">
        <thead>
          <tr class="border-b border-gray-200 text-sm text-gray-500">
            <th class="pb-3 font-medium">TID</th>
            <th class="pb-3 font-medium">标题</th>
            <th class="pb-3 font-medium">作者</th>
            <th class="pb-3 font-medium">FID</th>
            <th class="pb-3 font-medium">回复</th>
            <th class="pb-3 font-medium">浏览</th>
            <th class="pb-3 font-medium">状态</th>
            <th class="pb-3 font-medium">创建时间</th>
            <th class="pb-3 font-medium">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="t in threads" :key="t.tid" class="border-b border-gray-100 hover:bg-gray-50">
            <td class="py-3 text-sm font-mono text-gray-500">{{ t.tid }}</td>
            <td class="py-3 text-sm font-medium text-gray-800 max-w-xs truncate">{{ t.subject }}</td>
            <td class="py-3 text-sm text-gray-500">{{ t.username || '未知' }}</td>
            <td class="py-3 text-sm text-gray-500">{{ t.fid }}</td>
            <td class="py-3 text-sm text-gray-500">{{ t.posts }}</td>
            <td class="py-3 text-sm text-gray-500">{{ t.views }}</td>
            <td class="py-3">
              <span v-if="t.closed" class="text-xs bg-yellow-100 text-yellow-700 px-2 py-0.5 rounded-full">已关闭</span>
              <span v-else class="text-xs bg-green-100 text-green-700 px-2 py-0.5 rounded-full">开放</span>
            </td>
            <td class="py-3 text-sm text-gray-500 whitespace-nowrap">{{ formatDate(t.create_date) }}</td>
            <td class="py-3">
              <button @click="deleteThread(t.tid, t.subject)"
                class="text-xs text-red-500 hover:text-red-700">删除</button>
            </td>
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
