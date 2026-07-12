<!-- xiuno-go v2.1.0-beta 尼克修改版 -->
<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Editor from '@toast-ui/editor'
import '@toast-ui/editor/dist/toastui-editor.css'
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
const submitting = ref(false)
const attachFiles = ref<File[]>([])
const uploading = ref(false)

let editor: Editor | null = null
const editorEl = ref<HTMLDivElement | null>(null)

onMounted(async () => {
  // 加载版块列表
  try {
    const data: any = await request.get('/forum')
    forums.value = data || []
    const qfid = route.query.fid
    if (qfid) {
      const n = Number(qfid)
      if (forums.value.some(f => f.fid === n)) {
        fid.value = n
      }
    }
  } catch {
    forums.value = [{ fid: 1, name: '默认版块' }]
  }

  // 初始化 Toast UI Editor
  if (editorEl.value) {
    editor = new Editor({
      el: editorEl.value,
      height: '400px',
      initialEditType: 'wysiwyg',
      previewStyle: 'vertical',
      language: 'zh-CN',
      hideModeSwitch: false,
      hooks: {
        addImageBlobHook: async (blob: Blob | File, callback: (url: string, altText?: string) => void) => {
          try {
            const formData = new FormData()
            formData.append('file', blob)
            const res: any = await request.post('/attach', formData, {
              headers: { 'Content-Type': 'multipart/form-data' },
            })
            const url = res.url || `/upload/${res.path}`
            callback(url, (blob as File).name || 'image')
          } catch (e: any) {
            alert('图片上传失败: ' + (e.message || '未知错误'))
          }
        },
      },
    })
  }
})

onBeforeUnmount(() => {
  if (editor) {
    editor.destroy()
    editor = null
  }
})

// 附件上传（非图片文件）
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
      const markdownLink = `[${file.name}](${url})`
      if (editor) {
        editor.insertText(markdownLink)
      }
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

const tagInput = ref('')
const tagList = ref<string[]>([])

function addTag() {
  const t = tagInput.value.trim()
  if (!t) return
  if (tagList.value.includes(t)) {
    tagInput.value = ''
    return
  }
  if (tagList.value.length >= 10) return
  tagList.value.push(t)
  tagInput.value = ''
}

function removeTag(idx: number) {
  tagList.value.splice(idx, 1)
}

function onTagKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    e.preventDefault()
    addTag()
  }
}

async function handleSubmit() {
  if (!subject.value.trim()) return
  const message = editor ? editor.getMarkdown() : ''
  if (!message.trim()) return
  submitting.value = true
  try {
    const res: any = await request.post('/thread', {
      fid: fid.value,
      subject: subject.value,
      message: message,
      doctype: 2,
      tags: tagList.value.join(','),
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
  <div class="max-w-4xl mx-auto">
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

      <!-- 标签 -->
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">标签 <span class="text-xs text-gray-400 font-normal">（输入后按回车添加，最多 10 个）</span></label>
        <div class="flex flex-wrap gap-1.5 mb-2" v-if="tagList.length">
          <span v-for="(t, i) in tagList" :key="i"
            class="inline-flex items-center gap-1 px-2.5 py-0.5 bg-indigo-50 text-indigo-600 text-xs rounded-full">
            {{ t }}
            <button @click="removeTag(i)" class="text-indigo-400 hover:text-indigo-600 leading-none">&times;</button>
          </span>
        </div>
        <input v-model="tagInput" @keydown="onTagKeydown" type="text" placeholder="输入标签后按回车添加"
          class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 text-sm" />
      </div>

      <!-- Toast UI Editor -->
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">内容</label>
        <div ref="editorEl" class="border border-gray-300 rounded-lg"></div>
      </div>

      <!-- 附件上传（非图片文件） -->
      <div class="flex items-center gap-2">
        <label class="cursor-pointer text-sm text-indigo-600 hover:text-indigo-800 transition-colors">
          <input type="file" multiple accept="image/*,.pdf" class="hidden" @change="handleFileInput" />
          📎 上传附件
        </label>
        <span v-if="uploading" class="text-sm text-gray-400">上传中...</span>
        <span class="text-xs text-gray-400 ml-2">图片可直接拖拽或粘贴到编辑器中</span>
      </div>

      <!-- 提交按钮 -->
      <div class="flex justify-end pt-2">
        <button @click="handleSubmit" :disabled="submitting || !subject.trim()"
          class="bg-indigo-600 text-white px-6 py-2 rounded-lg text-sm hover:bg-indigo-700 disabled:opacity-50 transition-colors">
          {{ submitting ? '发布中...' : '发布帖子' }}
        </button>
      </div>
    </div>
  </div>
</template>
