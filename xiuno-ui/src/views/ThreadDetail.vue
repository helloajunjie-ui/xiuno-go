<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import request from '../utils/request'
import { useUserStore } from '../stores/user'
import ModerateModal from '../components/ModerateModal.vue'

interface ThreadDetail {
  tid: number
  fid: number
  uid: number
  subject: string
  message: string
  message_fmt: string
  doctype: number
  username: string
  avatar: number
  create_date: number
  views: number
  posts: number
}

interface ReplyItem {
  pid: number
  uid: number
  username: string
  avatar: number
  message: string
  message_fmt: string
  doctype: number
  create_date: number
  quotepid: number
}

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const detail = ref<ThreadDetail | null>(null)
const replies = ref<ReplyItem[]>([])
const loading = ref(true)
const replyText = ref('')
const submitting = ref(false)
const attachFiles = ref<File[]>([])
const uploading = ref(false)

// 编辑模式
const editingPid = ref<number | null>(null) // null = 主帖, 0+ = 回帖 pid
const editSubject = ref('')
const editMessage = ref('')

// 引用回复
const quotePid = ref(0)

// 版务
const showModModal = ref(false)
const modAction = ref('')

const currentUser = computed(() => userStore.user)
const canModerate = computed(() => {
  return userStore.isLoggedIn && userStore.user?.gid !== undefined && userStore.user.gid <= 3
})

function openModModal(action: string) {
  modAction.value = action
  showModModal.value = true
}

function onModDone() {
  showModModal.value = false
  fetchDetail()
}

// 智能渲染分流：根据 doctype 选择渲染策略
function renderPostContent(post: { doctype?: number; message_fmt?: string; message?: string }): string {
  const doctype = post.doctype ?? 2
  if (doctype === 0 || doctype === 1) {
    return DOMPurify.sanitize(post.message_fmt || '')
  }
  const raw = marked.parse(post.message || '', { async: false }) as string
  return DOMPurify.sanitize(raw)
}

// 引用回复：从 replies 中找被引用的楼层，取前 50 字
function getQuotePreview(quotepid: number): string {
  if (!quotepid || !replies.value.length) return ''
  const quoted = replies.value.find(r => r.pid === quotepid)
  if (!quoted) return ''
  const text = quoted.message || ''
  return text.length > 50 ? text.slice(0, 50) + '...' : text
}

function timeAgo(ts: number): string {
  const diff = Date.now() / 1000 - ts
  if (diff < 60) return '刚刚'
  if (diff < 3600) return `${Math.floor(diff / 60)} 分钟前`
  if (diff < 86400) return `${Math.floor(diff / 3600)} 小时前`
  return `${Math.floor(diff / 86400)} 天前`
}

async function fetchDetail() {
  loading.value = true
  try {
    const tid = route.params.tid as string
    const [d, r]: any = await Promise.all([
      request.get(`/thread/${tid}`),
      request.get(`/thread/${tid}/post`, { params: { page: 1 } }),
    ])
    detail.value = d as ThreadDetail
    replies.value = r.list || []
  } catch (e) {
    console.error('获取帖子详情失败', e)
  } finally {
    loading.value = false
  }
}

// 附件上传
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
      replyText.value += (replyText.value ? '\n' : '') + markdownImg
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

async function handleReply() {
  if (!replyText.value.trim()) return
  submitting.value = true
  try {
    const tid = route.params.tid as string
    const payload: any = { message: replyText.value }
    if (quotePid.value > 0) {
      payload.quotepid = quotePid.value
    }
    await request.post(`/thread/${tid}/post`, payload)
    replyText.value = ''
    quotePid.value = 0
    await fetchDetail()
  } catch (e: any) {
    alert(e.message || '回复失败')
  } finally {
    submitting.value = false
  }
}

// --- 编辑功能 ---
function startEdit(pid: number | null) {
  editingPid.value = pid
  if (pid === null && detail.value) {
    // 编辑主帖
    editSubject.value = detail.value.subject
    editMessage.value = detail.value.message
  } else if (pid !== null) {
    // 编辑回帖
    const reply = replies.value.find(r => r.pid === pid)
    if (reply) {
      editSubject.value = ''
      editMessage.value = reply.message
    }
  }
}

function cancelEdit() {
  editingPid.value = null
  editSubject.value = ''
  editMessage.value = ''
}

async function saveEdit() {
  if (!editMessage.value.trim()) return
  submitting.value = true
  try {
    const tid = route.params.tid as string
    if (editingPid.value === null) {
      // 编辑主帖
      await request.put(`/thread/${tid}`, {
        subject: editSubject.value,
        message: editMessage.value,
      })
    } else {
      // 编辑回帖
      await request.put(`/post/${editingPid.value}`, {
        message: editMessage.value,
      })
    }
    cancelEdit()
    await fetchDetail()
  } catch (e: any) {
    alert(e.message || '编辑失败')
  } finally {
    submitting.value = false
  }
}

// --- 引用回复 ---
function quoteReply(pid: number, username: string) {
  quotePid.value = pid
  replyText.value = `> **${username}** 说：\n\n`
  // 滚动到回复框
  setTimeout(() => {
    document.getElementById('reply-box')?.scrollIntoView({ behavior: 'smooth' })
  }, 100)
}

function cancelQuote() {
  quotePid.value = 0
}

onMounted(fetchDetail)
</script>

<template>
  <div v-if="loading" class="space-y-4">
    <div class="bg-white rounded-xl shadow-sm border border-gray-200 p-6 animate-pulse">
      <div class="h-6 bg-gray-200 rounded w-2/3 mb-4"></div>
      <div class="h-4 bg-gray-100 rounded w-1/4 mb-6"></div>
      <div class="space-y-2">
        <div v-for="i in 4" :key="i" class="h-4 bg-gray-100 rounded w-full"></div>
      </div>
    </div>
  </div>

  <div v-else-if="detail" class="space-y-4">
    <!-- 主帖 -->
    <article class="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
      <div class="flex items-start justify-between mb-3">
        <h1 class="text-2xl font-bold text-gray-900">{{ detail.subject }}</h1>
        <div class="shrink-0 flex gap-1 ml-4">
          <!-- 编辑按钮（自己的帖子） -->
          <button v-if="currentUser && currentUser.uid === detail.uid && editingPid !== null"
            @click="cancelEdit" class="text-xs px-2 py-1 border border-gray-300 rounded hover:bg-gray-100 transition-colors">取消</button>
          <button v-if="currentUser && currentUser.uid === detail.uid && editingPid === null"
            @click="startEdit(null)" class="text-xs px-2 py-1 border border-gray-300 rounded hover:bg-gray-100 transition-colors">编辑</button>
          <!-- 版务按钮组 -->
          <template v-if="canModerate">
            <button @click="openModModal('top')" class="text-xs px-2 py-1 border border-gray-300 rounded hover:bg-gray-100 transition-colors">置顶</button>
            <button @click="openModModal('close')" class="text-xs px-2 py-1 border border-gray-300 rounded hover:bg-gray-100 transition-colors">关闭</button>
            <button @click="openModModal('move')" class="text-xs px-2 py-1 border border-gray-300 rounded hover:bg-gray-100 transition-colors">移动</button>
            <button @click="openModModal('delete')" class="text-xs px-2 py-1 border border-red-300 text-red-700 rounded hover:bg-red-50 transition-colors">删除</button>
          </template>
        </div>
      </div>
      <div class="flex items-center gap-3 text-sm text-gray-500 mb-6">
        <span class="font-medium text-gray-700">{{ detail.username }}</span>
        <span>{{ timeAgo(detail.create_date) }}</span>
        <span class="ml-auto">{{ detail.views }} 浏览</span>
      </div>

      <!-- 编辑模式：主帖 -->
      <div v-if="editingPid === null">
        <div class="prose prose-gray max-w-none" v-html="renderPostContent(detail)"></div>
      </div>
      <div v-else class="space-y-3">
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">标题</label>
          <input v-model="editSubject" type="text"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 text-sm" />
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">内容</label>
          <textarea v-model="editMessage" rows="8"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 resize-none text-sm"></textarea>
        </div>
        <div class="flex justify-end gap-2">
          <button @click="cancelEdit" class="px-4 py-1.5 text-sm border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors">取消</button>
          <button @click="saveEdit" :disabled="submitting || !editMessage.trim()"
            class="px-4 py-1.5 text-sm bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 disabled:opacity-50 transition-colors">
            {{ submitting ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </article>

    <!-- 回帖列表 -->
    <div class="space-y-3">
      <h2 class="text-lg font-semibold text-gray-800">{{ detail.posts }} 条回复</h2>
      <div v-for="(reply, idx) in replies" :key="reply.pid"
        class="bg-white rounded-xl shadow-sm border border-gray-200 p-5">
        <div class="flex items-center gap-2 text-sm text-gray-500 mb-3">
          <span class="font-medium text-gray-700">{{ reply.username }}</span>
          <span>{{ timeAgo(reply.create_date) }}</span>
          <span class="ml-auto text-gray-400">#{{ idx + 2 }} 楼</span>
        </div>
        <!-- 引用块 -->
        <div v-if="reply.quotepid > 0 && getQuotePreview(reply.quotepid)"
          class="mb-3 pl-3 border-l-4 border-gray-300 text-sm text-gray-500 italic">
          引用 #{{ replies.findIndex(r => r.pid === reply.quotepid) + 2 }} 楼：{{ getQuotePreview(reply.quotepid) }}
        </div>

        <!-- 编辑模式：回帖 -->
        <div v-if="editingPid === reply.pid" class="space-y-3">
          <textarea v-model="editMessage" rows="5"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 resize-none text-sm"></textarea>
          <div class="flex justify-end gap-2">
            <button @click="cancelEdit" class="px-3 py-1 text-xs border border-gray-300 rounded hover:bg-gray-50 transition-colors">取消</button>
            <button @click="saveEdit" :disabled="submitting || !editMessage.trim()"
              class="px-3 py-1 text-xs bg-indigo-600 text-white rounded hover:bg-indigo-700 disabled:opacity-50 transition-colors">
              {{ submitting ? '保存中...' : '保存' }}
            </button>
          </div>
        </div>
        <div v-else>
          <div class="prose prose-sm prose-gray max-w-none" v-html="renderPostContent(reply)"></div>
          <!-- 操作按钮 -->
          <div class="flex gap-3 mt-3 pt-3 border-t border-gray-100">
            <button @click="quoteReply(reply.pid, reply.username)"
              class="text-xs text-gray-400 hover:text-indigo-600 transition-colors">引用</button>
            <button v-if="currentUser && currentUser.uid === reply.uid"
              @click="startEdit(reply.pid)"
              class="text-xs text-gray-400 hover:text-indigo-600 transition-colors">编辑</button>
          </div>
        </div>
      </div>
      <div v-if="replies.length === 0" class="text-center py-8 text-gray-400">
        暂无回复，来说点什么吧
      </div>
    </div>

    <!-- 回复框 -->
    <div v-if="userStore.user" id="reply-box" class="bg-white rounded-xl shadow-sm border border-gray-200 p-5">
      <!-- 引用提示 -->
      <div v-if="quotePid > 0" class="mb-2 flex items-center gap-2 text-xs text-indigo-600 bg-indigo-50 rounded px-3 py-1.5">
        <span>正在引用 #{{ replies.findIndex(r => r.pid === quotePid) + 2 }} 楼</span>
        <button @click="cancelQuote" class="ml-auto text-indigo-400 hover:text-indigo-700">&times;</button>
      </div>
      <textarea v-model="replyText" rows="3" placeholder="写下你的回复...（支持 Markdown）"
        class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 resize-none text-sm"></textarea>
      <div class="flex items-center gap-2 mt-2">
        <label class="cursor-pointer text-sm text-indigo-600 hover:text-indigo-800 transition-colors">
          <input type="file" multiple accept="image/*,.pdf" class="hidden" @change="handleFileInput" />
          📎 上传附件
        </label>
        <span v-if="uploading" class="text-sm text-gray-400">上传中...</span>
      </div>
      <div class="flex justify-end mt-3">
        <button @click="handleReply" :disabled="submitting || !replyText.trim()"
          class="bg-indigo-600 text-white px-5 py-1.5 rounded-lg text-sm hover:bg-indigo-700 disabled:opacity-50 transition-colors">
          {{ submitting ? '发送中...' : '回复' }}
        </button>
      </div>
    </div>
    <div v-else class="text-center py-4 text-sm text-gray-500">
      <router-link to="/login" class="text-indigo-600 hover:underline">登录</router-link> 后参与回复
    </div>
  </div>

  <div v-else class="text-center py-16 text-gray-400">
    帖子不存在或已被删除
  </div>

  <!-- 版务操作弹窗 -->
  <ModerateModal
    v-if="showModModal && detail"
    :action="modAction"
    :tidarr="[detail.tid]"
    :onDone="onModDone"
    @close="showModModal = false" />
</template>
