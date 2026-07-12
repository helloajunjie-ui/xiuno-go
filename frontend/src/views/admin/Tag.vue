<!-- xiuno-go v2.1.0-beta 尼克修改版 -->
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import request from '../../utils/request'

interface Tag {
  tagid: number
  name: string
  threads: number
  create_date: number
}

const tags = ref<Tag[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 50
const loading = ref(true)
const editing = ref<{ tagid: number; name: string } | null>(null)
const showCreate = ref(false)
const newName = ref('')

const loadTags = async () => {
  loading.value = true
  try {
    const res: any = await request.get('/admin/tag', { params: { page: page.value } })
    tags.value = res.tags as Tag[]
    total.value = res.total
  } finally {
    loading.value = false
  }
}

const startEdit = (t: Tag) => {
  editing.value = { tagid: t.tagid, name: t.name }
}

const cancelEdit = () => {
  editing.value = null
}

const saveEdit = async () => {
  if (!editing.value || !editing.value.name.trim()) return
  try {
    await request.put(`/admin/tag/${editing.value.tagid}`, { name: editing.value.name.trim() })
    editing.value = null
    loadTags()
  } catch {
    alert('保存失败')
  }
}

const deleteTag = async (tagid: number, name: string) => {
  if (!confirm(`确定要删除标签「${name}」吗？将同时解除该标签与所有帖子的关联。`)) return
  try {
    await request.delete(`/admin/tag/${tagid}`)
    loadTags()
  } catch {
    alert('删除失败')
  }
}

const createTag = async () => {
  if (!newName.value.trim()) {
    alert('请输入标签名称')
    return
  }
  try {
    await request.post('/admin/tag', { name: newName.value.trim() })
    showCreate.value = false
    newName.value = ''
    loadTags()
  } catch {
    alert('创建失败')
  }
}

const totalPages = () => Math.ceil(total.value / pageSize)

const goPage = (p: number) => {
  if (p < 1 || p > totalPages()) return
  page.value = p
  loadTags()
}

const formatDate = (ts: number) => {
  if (!ts) return '-'
  return new Date(ts * 1000).toLocaleString('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit'
  })
}

onMounted(() => { loadTags() })
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h2 class="text-2xl font-bold">标签管理</h2>
      <button @click="showCreate = true"
        class="bg-indigo-600 text-white px-4 py-2 rounded-lg hover:bg-indigo-700 transition text-sm">
        + 新建标签
      </button>
    </div>

    <!-- 新建标签弹窗 -->
    <div v-if="showCreate" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50"
      @click.self="showCreate = false">
      <div class="bg-white rounded-xl shadow-xl p-6 w-full max-w-md mx-4">
        <h3 class="text-lg font-bold mb-4">新建标签</h3>
        <div>
          <label class="block text-sm text-gray-600 mb-1">标签名称</label>
          <input v-model="newName" type="text" class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm"
            placeholder="输入标签名" @keyup.enter="createTag" />
        </div>
        <div class="flex justify-end gap-2 mt-6">
          <button @click="showCreate = false" class="px-4 py-2 text-sm text-gray-600 hover:bg-gray-100 rounded-lg">取消</button>
          <button @click="createTag" class="px-4 py-2 text-sm bg-indigo-600 text-white rounded-lg hover:bg-indigo-700">创建</button>
        </div>
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
            <th class="pb-3 font-medium">ID</th>
            <th class="pb-3 font-medium">标签名</th>
            <th class="pb-3 font-medium">关联帖子数</th>
            <th class="pb-3 font-medium">创建时间</th>
            <th class="pb-3 font-medium">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="t in tags" :key="t.tagid" class="border-b border-gray-100 hover:bg-gray-50">
            <td class="py-3 text-sm font-mono text-gray-500">{{ t.tagid }}</td>
            <td class="py-3">
              <!-- 编辑模式 -->
              <div v-if="editing?.tagid === t.tagid" class="flex items-center gap-2">
                <input v-model="editing.name" type="text"
                  class="border border-gray-300 rounded px-2 py-1 text-sm w-40" @keyup.enter="saveEdit" />
                <button @click="saveEdit" class="text-xs text-indigo-600 hover:text-indigo-800">保存</button>
                <button @click="cancelEdit" class="text-xs text-gray-500 hover:text-gray-700">取消</button>
              </div>
              <span v-else class="font-medium text-gray-800">{{ t.name }}</span>
            </td>
            <td class="py-3 text-sm text-gray-500">{{ t.threads }}</td>
            <td class="py-3 text-sm text-gray-500">{{ formatDate(t.create_date) }}</td>
            <td class="py-3">
              <div v-if="editing?.tagid !== t.tagid" class="flex gap-2">
                <button @click="startEdit(t)" class="text-xs text-indigo-600 hover:text-indigo-800">编辑</button>
                <button @click="deleteTag(t.tagid, t.name)" class="text-xs text-red-500 hover:text-red-700">删除</button>
              </div>
            </td>
          </tr>
          <tr v-if="tags.length === 0">
            <td colspan="5" class="text-center text-gray-400 py-10">暂无标签数据</td>
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
