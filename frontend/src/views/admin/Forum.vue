<!-- xiuno-go v2.1.0-beta 尼克修改版 -->
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import request from '../../utils/request'

interface Forum {
  fid: number
  name: string
  rank: number
  threads: number
  todayposts: number
  brief?: string
  accesson?: number
}

interface ForumAccess {
  fid: number
  gid: number
  allowread: number
  allowthread: number
  allowpost: number
  allowattach: number
  allowdown: number
}

interface Group {
  gid: number
  name: string
}

const forums = ref<Forum[]>([])
const loading = ref(true)

// 版块编辑弹窗
const showModal = ref(false)
const isEdit = ref(false)
const currentFid = ref(0)
const form = ref({ name: '', brief: '', rank: 0 })

// 权限管理弹窗
const showAccessModal = ref(false)
const accessFid = ref(0)
const accessForumName = ref('')
const accessOn = ref(0)
const allGroups = ref<Group[]>([])
const accessList = ref<ForumAccess[]>([])
const accessLoading = ref(false)

const loadForums = async () => {
  try {
    const res = await request.get<Forum[]>('/admin/forum')
    forums.value = res
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  isEdit.value = false
  currentFid.value = 0
  form.value = { name: '', brief: '', rank: 0 }
  showModal.value = true
}

const openEdit = (f: Forum) => {
  isEdit.value = true
  currentFid.value = f.fid
  form.value = { name: f.name, brief: f.brief || '', rank: f.rank }
  showModal.value = true
}

const submitForm = async () => {
  if (!form.value.name.trim()) return alert('版块名不能为空')
  try {
    if (isEdit.value) {
      await request.put(`/admin/forum/${currentFid.value}`, form.value)
    } else {
      await request.post('/admin/forum', form.value)
    }
    showModal.value = false
    await loadForums()
  } catch (e: any) {
    alert(e.message || '操作失败')
  }
}

const deleteForum = async (fid: number) => {
  if (!confirm('确定要删除此版块？')) return
  try {
    await request.delete(`/admin/forum/${fid}`)
    forums.value = forums.value.filter(f => f.fid !== fid)
  } catch {
    alert('删除失败')
  }
}

// ==================== 权限管理 ====================

const openAccess = async (f: Forum) => {
  accessFid.value = f.fid
  accessForumName.value = f.name
  accessOn.value = f.accesson || 0
  accessLoading.value = true
  showAccessModal.value = true
  try {
    const res = await request.get<{ access_list: ForumAccess[], groups: Group[] }>(`/admin/forum/${f.fid}/access`)
    accessList.value = res.access_list
    allGroups.value = res.groups
  } catch {
    alert('加载权限数据失败')
  } finally {
    accessLoading.value = false
  }
}

const getAccess = (gid: number): ForumAccess | undefined => {
  return accessList.value.find(a => a.gid === gid)
}

const ensureAccess = (gid: number): ForumAccess => {
  let a = getAccess(gid)
  if (!a) {
    a = { fid: accessFid.value, gid, allowread: 0, allowthread: 0, allowpost: 0, allowattach: 0, allowdown: 0 }
    accessList.value.push(a)
  }
  return a
}

const toggleAccess = (gid: number, field: keyof ForumAccess) => {
  const a = ensureAccess(gid)
  a[field] = a[field] ? 0 : 1
}

const saveAccess = async () => {
  try {
    await request.put(`/admin/forum/${accessFid.value}/access`, {
      accesson: accessOn.value,
      access_list: accessList.value,
    })
    // 更新本地版块列表中的 accesson
    const f = forums.value.find(f => f.fid === accessFid.value)
    if (f) f.accesson = accessOn.value
    showAccessModal.value = false
  } catch (e: any) {
    alert(e.message || '保存失败')
  }
}

onMounted(() => { loadForums() })
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h2 class="text-2xl font-bold">版块管理</h2>
      <button @click="openCreate" class="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition text-sm">+ 新建版块</button>
    </div>

    <div v-if="loading" class="animate-pulse space-y-4">
      <div class="h-12 bg-gray-200 rounded w-full"></div>
      <div class="h-12 bg-gray-200 rounded w-full"></div>
    </div>

    <div v-else class="overflow-x-auto">
      <table class="w-full text-left">
        <thead>
          <tr class="border-b border-gray-200 text-sm text-gray-500">
            <th class="pb-3 font-medium">FID</th>
            <th class="pb-3 font-medium">版块名称</th>
            <th class="pb-3 font-medium">排序</th>
            <th class="pb-3 font-medium">主题数</th>
            <th class="pb-3 font-medium">今日帖</th>
            <th class="pb-3 font-medium">权限</th>
            <th class="pb-3 font-medium">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="f in forums" :key="f.fid" class="border-b border-gray-100 hover:bg-gray-50">
            <td class="py-3 text-sm font-mono text-gray-500">{{ f.fid }}</td>
            <td class="py-3 font-medium text-gray-800">{{ f.name }}</td>
            <td class="py-3 text-sm text-gray-500">{{ f.rank }}</td>
            <td class="py-3 text-sm text-gray-500">{{ f.threads }}</td>
            <td class="py-3 text-sm text-gray-500">{{ f.todayposts }}</td>
            <td class="py-3">
              <span :class="f.accesson ? 'text-green-600 bg-green-50 px-2 py-0.5 rounded text-xs font-medium' : 'text-gray-400 text-xs'">
                {{ f.accesson ? '独立权限' : '全局默认' }}
              </span>
            </td>
            <td class="py-3">
              <button @click="openAccess(f)" class="text-indigo-600 hover:text-indigo-800 text-sm mr-3">权限</button>
              <button @click="openEdit(f)" class="text-blue-600 hover:text-blue-800 text-sm mr-3">编辑</button>
              <button @click="deleteForum(f.fid)" class="text-red-500 hover:text-red-700 text-sm">删除</button>
            </td>
          </tr>
          <tr v-if="forums.length === 0">
            <td colspan="7" class="text-center text-gray-400 py-10">暂无版块</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 版块编辑弹窗 -->
    <div v-if="showModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-40 backdrop-blur-sm transition-opacity">
      <div class="bg-white rounded-xl shadow-2xl w-full max-w-md p-6 transform transition-all">
        <h3 class="text-xl font-bold mb-4">{{ isEdit ? '编辑版块' : '新建版块' }}</h3>

        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">版块名称</label>
            <input v-model="form.name" type="text" class="w-full border border-gray-300 rounded-lg px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent" placeholder="例如：日常灌水" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">版块简介</label>
            <textarea v-model="form.brief" rows="3" class="w-full border border-gray-300 rounded-lg px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent" placeholder="一句话介绍这个版块"></textarea>
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">排序权重 (Rank)</label>
            <input v-model.number="form.rank" type="number" class="w-full border border-gray-300 rounded-lg px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent" placeholder="数字越小越靠前" />
          </div>
        </div>

        <div class="mt-6 flex justify-end space-x-3">
          <button @click="showModal = false" class="px-4 py-2 text-gray-600 hover:bg-gray-100 rounded-lg transition">取消</button>
          <button @click="submitForm" class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition">保存</button>
        </div>
      </div>
    </div>

    <!-- 权限管理弹窗 -->
    <div v-if="showAccessModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-40 backdrop-blur-sm transition-opacity">
      <div class="bg-white rounded-xl shadow-2xl w-full max-w-3xl p-6 transform transition-all max-h-[80vh] overflow-y-auto">
        <h3 class="text-xl font-bold mb-1">版块权限配置</h3>
        <p class="text-sm text-gray-500 mb-4">{{ accessForumName }} (FID: {{ accessFid }})</p>

        <div v-if="accessLoading" class="text-center py-8 text-gray-400">加载中...</div>

        <div v-else>
          <!-- 权限开关 -->
          <div class="flex items-center gap-3 mb-6 p-4 bg-gray-50 rounded-lg">
            <label class="text-sm font-medium text-gray-700">独立权限开关：</label>
            <button @click="accessOn = accessOn ? 0 : 1"
              :class="accessOn ? 'bg-green-500' : 'bg-gray-300'"
              class="relative w-12 h-6 rounded-full transition-colors">
              <span :class="accessOn ? 'translate-x-6' : 'translate-x-1'"
                class="absolute top-1 left-0 w-4 h-4 bg-white rounded-full transition-transform shadow"></span>
            </button>
            <span class="text-xs text-gray-500">{{ accessOn ? '开启 - 各用户组独立配置权限' : '关闭 - 使用全局用户组默认权限' }}</span>
          </div>

          <!-- 权限配置表格 -->
          <div v-if="accessOn" class="overflow-x-auto">
            <table class="w-full text-left text-sm">
              <thead>
                <tr class="border-b border-gray-200 text-gray-500">
                  <th class="pb-2 font-medium">用户组</th>
                  <th class="pb-2 font-medium text-center">浏览</th>
                  <th class="pb-2 font-medium text-center">发帖</th>
                  <th class="pb-2 font-medium text-center">回复</th>
                  <th class="pb-2 font-medium text-center">附件</th>
                  <th class="pb-2 font-medium text-center">下载</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="g in allGroups" :key="g.gid" class="border-b border-gray-100 hover:bg-gray-50">
                  <td class="py-2.5 font-medium text-gray-700">{{ g.name }} (GID={{ g.gid }})</td>
                  <td class="py-2.5 text-center">
                    <input type="checkbox" :checked="getAccess(g.gid)?.allowread === 1"
                      @change="toggleAccess(g.gid, 'allowread')"
                      class="w-4 h-4 text-blue-600 rounded" />
                  </td>
                  <td class="py-2.5 text-center">
                    <input type="checkbox" :checked="getAccess(g.gid)?.allowthread === 1"
                      @change="toggleAccess(g.gid, 'allowthread')"
                      class="w-4 h-4 text-blue-600 rounded" />
                  </td>
                  <td class="py-2.5 text-center">
                    <input type="checkbox" :checked="getAccess(g.gid)?.allowpost === 1"
                      @change="toggleAccess(g.gid, 'allowpost')"
                      class="w-4 h-4 text-blue-600 rounded" />
                  </td>
                  <td class="py-2.5 text-center">
                    <input type="checkbox" :checked="getAccess(g.gid)?.allowattach === 1"
                      @change="toggleAccess(g.gid, 'allowattach')"
                      class="w-4 h-4 text-blue-600 rounded" />
                  </td>
                  <td class="py-2.5 text-center">
                    <input type="checkbox" :checked="getAccess(g.gid)?.allowdown === 1"
                      @change="toggleAccess(g.gid, 'allowdown')"
                      class="w-4 h-4 text-blue-600 rounded" />
                  </td>
                </tr>
                <tr v-if="allGroups.length === 0">
                  <td colspan="6" class="text-center text-gray-400 py-6">暂无用户组数据</td>
                </tr>
              </tbody>
            </table>
          </div>

          <div v-else class="text-center py-6 text-gray-400 text-sm">
            独立权限已关闭，所有用户组使用全局默认权限。
          </div>
        </div>

        <div class="mt-6 flex justify-end space-x-3 border-t border-gray-100 pt-4">
          <button @click="showAccessModal = false" class="px-4 py-2 text-gray-600 hover:bg-gray-100 rounded-lg transition">取消</button>
          <button @click="saveAccess" class="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition">保存权限配置</button>
        </div>
      </div>
    </div>
  </div>
</template>
