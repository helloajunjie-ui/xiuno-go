<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import request from '../utils/request'

interface Forum {
  fid: number
  name: string
}

const route = useRoute()
const router = useRouter()

const forums = ref<Forum[]>([])
const fid = ref(1)
const subject = ref('')
const message = ref('')
const submitting = ref(false)
const attachFiles = ref<File[]>([])
const uploading = ref(false)

onMounted(async () => {
  try {
    const data: any = await request.get('/forum')
    forums.value = data || []
    // 如果 URL 中有 fid 参数，自动选中
    const qfid = route.query.fid
    if (qfid) {
      const n = Number(qfid)
      if (forums.value.some(f => f.fid === n)) {
        fid.value = n
      }
    }
  } catch {
    // 默认版块兜底
    forums.value = [{ fid: 1, name: '默认版块' }]
  }
})

// 附件上传：上传成功后以 Markdown 图片语法插入 textarea 光标处
async function uploadAttachments() {
  if (!attachFiles.value.length) return
  uploading.value = true
  try {
    for (const file of attachFiles.value) {
      const formData = new FormData()
      formData.append('file', file)
      const res: any = await request.post('/attach', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      const url = res.url || `/upload/${res.path}`
      const markdownImg = `![${file.name}](${url})`
      message.value += (message.value ? '\n' : '') + markdownImg
    }
    attachFiles.value = []
  } catch (e: any) {
    alert(e.message || '上传失败')
  } finally {
    uploading.value = false
  }
}

function handleFileInput(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files) {
    attachFiles.value = Array.from(input.files)
    uploadAttachments()
  }
  input.value = ''
}

async function handleSubmit() {
  if (!subject.value.trim() || !message.value.trim()) return
  submitting.value = true
  try {
    const res: any = await request.post('/thread', {
      fid: fid.value,
      subject: subject.value,
      message: message.value,
    })
    router.push(`/thread/${res.tid}`)
  } catch (e: any) {
    alert(e.message || '发帖失败')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="max-w-3xl mx-auto">
    <h1 class="text-xl font-bold mb-6">发表新帖</h1>

    <div class="bg-white rounded-xl shadow-sm border border-gray-200 p-6 space-y-4">
      <!-- 版块选择 -->
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">版块</label>
        <select v-model.number="fid"
          class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 text-sm">
          <option v-for="f in forums" :key="f.fid" :value="f.fid">{{ f.name }}</option>
        </select>
      </div>

      <!-- 标题 -->
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">标题</label>
        <input v-model="subject" type="text" placeholder="输入标题..."
          class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 text-sm" />
      </div>

      <!-- 内容 -->
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">内容（支持 Markdown）</label>
        <textarea v-model="message" rows="10" placeholder="写下你的内容..."
          class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 resize-none text-sm"></textarea>
      </div>

      <!-- 附件上传 -->
      <div class="flex items-center gap-2">
        <label class="cursor-pointer text-sm text-indigo-600 hover:text-indigo-800 transition-colors">
          <input type="file" multiple accept="image/*,.pdf" class="hidden" @change="handleFileInput" />
          📎 上传附件
        </label>
        <span v-if="uploading" class="text-sm text-gray-400">上传中...</span>
      </div>

      <!-- 提交按钮 -->
      <div class="flex justify-end pt-2">
        <button @click="handleSubmit" :disabled="submitting || !subject.trim() || !message.trim()"
          class="bg-indigo-600 text-white px-6 py-2 rounded-lg text-sm hover:bg-indigo-700 disabled:opacity-50 transition-colors">
          {{ submitting ? '发布中...' : '发布帖子' }}
        </button>
      </div>
    </div>
  </div>
</template>
