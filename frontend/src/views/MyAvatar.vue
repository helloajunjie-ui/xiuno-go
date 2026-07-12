<!-- xiuno-go v2.1.0-beta 尼克修改版 -->
<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '../stores/user'

const router = useRouter()
const userStore = useUserStore()

const selectedFile = ref<File | null>(null)
const previewUrl = ref('')
const uploading = ref(false)
const errorMsg = ref('')
const successMsg = ref('')
const progress = ref(0)

const user = computed(() => userStore.user)
const isLoggedIn = computed(() => userStore.isLoggedIn)

const currentAvatarUrl = computed(() => {
  if (!user.value) return ''
  // 优先使用后端返回的完整 avatar_url（含 3 层目录切分），回退到扁平路径
  return user.value.avatar_url || `/upload/avatar/${user.value.uid}.png?t=${user.value.avatar || 0}`
})

if (!isLoggedIn.value) {
  router.push('/login')
}

function onFileSelected(e: Event) {
  errorMsg.value = ''
  successMsg.value = ''
  const input = e.target as HTMLInputElement
  if (!input.files || input.files.length === 0) return

  const file = input.files[0]

  // 校验文件类型
  const allowedTypes = ['image/gif', 'image/jpeg', 'image/png', 'image/jpg', 'image/bmp']
  if (!allowedTypes.includes(file.type)) {
    errorMsg.value = '仅支持 GIF、JPEG、PNG、BMP 格式'
    input.value = ''
    return
  }

  // 校验文件大小（2MB）
  if (file.size > 2 * 1024 * 1024) {
    errorMsg.value = '文件大小不能超过 2MB'
    input.value = ''
    return
  }

  selectedFile.value = file

  // 生成预览
  const reader = new FileReader()
  reader.onload = (ev) => {
    previewUrl.value = ev.target?.result as string
  }
  reader.readAsDataURL(file)
}

async function upload() {
  if (!selectedFile.value) {
    errorMsg.value = '请先选择一张图片'
    return
  }

  errorMsg.value = ''
  successMsg.value = ''
  uploading.value = true
  progress.value = 0

  try {
    const formData = new FormData()
    formData.append('file', selectedFile.value)

    // 模拟进度（XMLHttpRequest 方式）
    const data: any = await new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest()
      xhr.open('POST', '/api/v1/user/avatar')

      xhr.upload.onprogress = (ev) => {
        if (ev.lengthComputable) {
          progress.value = Math.round((ev.loaded / ev.total) * 100)
        }
      }

      xhr.onload = () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          resolve(JSON.parse(xhr.responseText))
        } else {
          try {
            reject(JSON.parse(xhr.responseText))
          } catch {
            reject(new Error('上传失败'))
          }
        }
      }

      xhr.onerror = () => reject(new Error('网络错误'))
      xhr.send(formData)
    })

    // 更新本地用户头像时间戳
    if (data?.data?.avatar) {
      userStore.updateAvatar(data.data.avatar)
    }

    successMsg.value = '头像上传成功！'
    progress.value = 100
  } catch (e: any) {
    errorMsg.value = e?.message || e?.data?.message || '上传失败'
  } finally {
    uploading.value = false
  }
}

function goBack() {
  router.push('/my')
}
</script>

<template>
  <div class="max-w-md mx-auto">
    <div class="flex items-center gap-3 mb-6">
      <button @click="goBack"
        class="text-gray-400 hover:text-gray-600 transition-colors">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
      </button>
      <h1 class="text-xl font-bold text-gray-900">上传头像</h1>
    </div>

    <div class="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
      <!-- 当前头像预览 -->
      <div class="flex justify-center mb-6">
        <img :src="previewUrl || currentAvatarUrl" alt="avatar"
          class="w-28 h-28 rounded-full object-cover border-4 border-gray-100"
          @error="($event.target as HTMLImageElement).src='/upload/avatar/0.png'" />
      </div>

      <!-- 成功提示 -->
      <div v-if="successMsg"
        class="mb-4 p-3 bg-green-50 border border-green-200 rounded-lg text-sm text-green-700">
        {{ successMsg }}
      </div>

      <!-- 错误提示 -->
      <div v-if="errorMsg"
        class="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-600">
        {{ errorMsg }}
      </div>

      <!-- 进度条 -->
      <div v-if="uploading && progress > 0 && progress < 100" class="mb-4">
        <div class="w-full bg-gray-200 rounded-full h-2">
          <div class="bg-indigo-600 h-2 rounded-full transition-all duration-300" :style="{ width: progress + '%' }"></div>
        </div>
        <div class="text-xs text-gray-400 text-center mt-1">{{ progress }}%</div>
      </div>

      <div class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">选择图片</label>
          <input type="file" accept="image/gif,image/jpeg,image/png,image/jpg,image/bmp"
            @change="onFileSelected"
            class="w-full text-sm text-gray-500 file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-sm file:font-medium file:bg-indigo-50 file:text-indigo-700 hover:file:bg-indigo-100 cursor-pointer" />
          <p class="text-xs text-gray-400 mt-1">支持 GIF、JPEG、PNG、BMP，最大 2MB</p>
        </div>

        <button @click="upload" :disabled="uploading || !selectedFile"
          class="w-full py-2.5 bg-indigo-600 hover:bg-indigo-700 disabled:bg-indigo-400 text-white font-medium rounded-lg transition-colors">
          {{ uploading ? '上传中...' : '上传头像' }}
        </button>
      </div>
    </div>
  </div>
</template>
