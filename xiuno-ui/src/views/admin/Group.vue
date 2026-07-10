<script setup lang="ts">
import { ref, onMounted } from 'vue'
import request from '../../utils/request'

interface Group {
  gid: number
  name: string
  creditsfrom: number
  creditsto: number
  allowread: number
  allowthread: number
  allowpost: number
  allowattach: number
  allowdown: number
  allowtop: number
  allowupdate: number
  allowdelete: number
  allowmove: number
  allowbanuser: number
  allowdeleteuser: number
  allowviewip: number
}

const groups = ref<Group[]>([])
const loading = ref(true)
const editing = ref<Group | null>(null)
const showCreate = ref(false)
const newGroup = ref<Partial<Group>>({ name: '', creditsfrom: 0, creditsto: 0 })

const loadGroups = async () => {
  try {
    const res = await request.get('/admin/group')
    groups.value = res as Group[]
  } finally {
    loading.value = false
  }
}

const startEdit = (g: Group) => {
  editing.value = { ...g }
}

const cancelEdit = () => {
  editing.value = null
}

const saveEdit = async () => {
  if (!editing.value) return
  try {
    await request.put(`/admin/group/${editing.value.gid}`, editing.value)
    alert('保存成功')
    editing.value = null
    loadGroups()
  } catch {
    alert('保存失败')
  }
}

const deleteGroup = async (gid: number, name: string) => {
  if (!confirm(`确定要删除用户组「${name}」吗？`)) return
  try {
    await request.delete(`/admin/group/${gid}`)
    alert('已删除')
    loadGroups()
  } catch {
    alert('删除失败')
  }
}

const createGroup = async () => {
  if (!newGroup.value.name) {
    alert('请输入用户组名称')
    return
  }
  try {
    await request.post('/admin/group', newGroup.value)
    alert('创建成功')
    showCreate.value = false
    newGroup.value = { name: '', creditsfrom: 0, creditsto: 0 }
    loadGroups()
  } catch {
    alert('创建失败')
  }
}

onMounted(() => { loadGroups() })
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h2 class="text-2xl font-bold">用户组管理</h2>
      <button @click="showCreate = true"
        class="bg-indigo-600 text-white px-4 py-2 rounded-lg hover:bg-indigo-700 transition text-sm">
        + 新建用户组
      </button>
    </div>

    <!-- 新建用户组弹窗 -->
    <div v-if="showCreate" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50"
      @click.self="showCreate = false">
      <div class="bg-white rounded-xl shadow-xl p-6 w-full max-w-md mx-4">
        <h3 class="text-lg font-bold mb-4">新建用户组</h3>
        <div class="space-y-3">
          <div>
            <label class="block text-sm text-gray-600 mb-1">名称</label>
            <input v-model="newGroup.name" type="text" class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm" />
          </div>
          <div class="flex gap-3">
            <div class="flex-1">
              <label class="block text-sm text-gray-600 mb-1">积分下限</label>
              <input v-model.number="newGroup.creditsfrom" type="number" class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm" />
            </div>
            <div class="flex-1">
              <label class="block text-sm text-gray-600 mb-1">积分上限</label>
              <input v-model.number="newGroup.creditsto" type="number" class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm" />
            </div>
          </div>
        </div>
        <div class="flex justify-end gap-2 mt-6">
          <button @click="showCreate = false" class="px-4 py-2 text-sm text-gray-600 hover:bg-gray-100 rounded-lg">取消</button>
          <button @click="createGroup" class="px-4 py-2 text-sm bg-indigo-600 text-white rounded-lg hover:bg-indigo-700">创建</button>
        </div>
      </div>
    </div>

    <div v-if="loading" class="animate-pulse space-y-4">
      <div class="h-12 bg-gray-200 rounded w-full"></div>
      <div class="h-12 bg-gray-200 rounded w-full"></div>
    </div>

    <div v-else class="space-y-4">
      <div v-for="g in groups" :key="g.gid"
        class="bg-white rounded-xl shadow-sm border border-gray-200 p-5">
        <!-- 编辑模式 -->
        <div v-if="editing?.gid === g.gid" class="space-y-3">
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-xs text-gray-500 mb-1">名称</label>
              <input v-model="editing.name" type="text" class="w-full border border-gray-300 rounded px-2 py-1 text-sm" />
            </div>
            <div>
              <label class="block text-xs text-gray-500 mb-1">积分下限</label>
              <input v-model.number="editing.creditsfrom" type="number" class="w-full border border-gray-300 rounded px-2 py-1 text-sm" />
            </div>
            <div>
              <label class="block text-xs text-gray-500 mb-1">积分上限</label>
              <input v-model.number="editing.creditsto" type="number" class="w-full border border-gray-300 rounded px-2 py-1 text-sm" />
            </div>
          </div>
          <div class="grid grid-cols-4 gap-2">
            <label class="flex items-center gap-1 text-xs">
              <input type="checkbox" v-model.number="editing.allowread" :true-value="1" :false-value="0" />
              浏览
            </label>
            <label class="flex items-center gap-1 text-xs">
              <input type="checkbox" v-model.number="editing.allowthread" :true-value="1" :false-value="0" />
              发帖
            </label>
            <label class="flex items-center gap-1 text-xs">
              <input type="checkbox" v-model.number="editing.allowpost" :true-value="1" :false-value="0" />
              回复
            </label>
            <label class="flex items-center gap-1 text-xs">
              <input type="checkbox" v-model.number="editing.allowattach" :true-value="1" :false-value="0" />
              附件
            </label>
            <label class="flex items-center gap-1 text-xs">
              <input type="checkbox" v-model.number="editing.allowdown" :true-value="1" :false-value="0" />
              下载
            </label>
            <label class="flex items-center gap-1 text-xs">
              <input type="checkbox" v-model.number="editing.allowtop" :true-value="1" :false-value="0" />
              置顶
            </label>
            <label class="flex items-center gap-1 text-xs">
              <input type="checkbox" v-model.number="editing.allowupdate" :true-value="1" :false-value="0" />
              编辑
            </label>
            <label class="flex items-center gap-1 text-xs">
              <input type="checkbox" v-model.number="editing.allowdelete" :true-value="1" :false-value="0" />
              删除
            </label>
            <label class="flex items-center gap-1 text-xs">
              <input type="checkbox" v-model.number="editing.allowmove" :true-value="1" :false-value="0" />
              移动
            </label>
            <label class="flex items-center gap-1 text-xs">
              <input type="checkbox" v-model.number="editing.allowbanuser" :true-value="1" :false-value="0" />
              封禁
            </label>
            <label class="flex items-center gap-1 text-xs">
              <input type="checkbox" v-model.number="editing.allowdeleteuser" :true-value="1" :false-value="0" />
              删用户
            </label>
            <label class="flex items-center gap-1 text-xs">
              <input type="checkbox" v-model.number="editing.allowviewip" :true-value="1" :false-value="0" />
              查看IP
            </label>
          </div>
          <div class="flex gap-2 pt-2">
            <button @click="saveEdit" class="px-3 py-1 text-xs bg-indigo-600 text-white rounded hover:bg-indigo-700">保存</button>
            <button @click="cancelEdit" class="px-3 py-1 text-xs bg-gray-100 text-gray-600 rounded hover:bg-gray-200">取消</button>
          </div>
        </div>

        <!-- 展示模式 -->
        <div v-else>
          <div class="flex items-center justify-between mb-2">
            <div>
              <span class="font-semibold text-gray-900">{{ g.name }}</span>
              <span class="ml-2 text-xs text-gray-400">GID {{ g.gid }}</span>
              <span v-if="g.gid < 100" class="ml-2 text-xs text-amber-500">系统组</span>
            </div>
            <div class="flex gap-2">
              <button @click="startEdit(g)" class="text-xs text-indigo-600 hover:text-indigo-800">编辑</button>
              <button v-if="g.gid >= 100" @click="deleteGroup(g.gid, g.name)"
                class="text-xs text-red-500 hover:text-red-700">删除</button>
            </div>
          </div>
          <div class="flex flex-wrap gap-2 text-xs text-gray-500">
            <span v-if="g.allowread" class="bg-blue-50 text-blue-600 px-2 py-0.5 rounded">浏览</span>
            <span v-if="g.allowthread" class="bg-blue-50 text-blue-600 px-2 py-0.5 rounded">发帖</span>
            <span v-if="g.allowpost" class="bg-blue-50 text-blue-600 px-2 py-0.5 rounded">回复</span>
            <span v-if="g.allowattach" class="bg-blue-50 text-blue-600 px-2 py-0.5 rounded">附件</span>
            <span v-if="g.allowdown" class="bg-blue-50 text-blue-600 px-2 py-0.5 rounded">下载</span>
            <span v-if="g.allowtop" class="bg-blue-50 text-blue-600 px-2 py-0.5 rounded">置顶</span>
            <span v-if="g.allowupdate" class="bg-blue-50 text-blue-600 px-2 py-0.5 rounded">编辑</span>
            <span v-if="g.allowdelete" class="bg-blue-50 text-blue-600 px-2 py-0.5 rounded">删除</span>
            <span v-if="g.allowmove" class="bg-blue-50 text-blue-600 px-2 py-0.5 rounded">移动</span>
            <span v-if="g.allowbanuser" class="bg-blue-50 text-blue-600 px-2 py-0.5 rounded">封禁</span>
            <span v-if="g.allowdeleteuser" class="bg-blue-50 text-blue-600 px-2 py-0.5 rounded">删用户</span>
            <span v-if="g.allowviewip" class="bg-blue-50 text-blue-600 px-2 py-0.5 rounded">查看IP</span>
            <span v-if="g.creditsfrom || g.creditsto" class="bg-gray-50 text-gray-500 px-2 py-0.5 rounded">
              积分区间: {{ g.creditsfrom }} ~ {{ g.creditsto }}
            </span>
          </div>
        </div>
      </div>

      <div v-if="groups.length === 0" class="text-center text-gray-400 py-10">暂无用户组数据</div>
    </div>
  </div>
</template>
