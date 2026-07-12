<!-- xiuno-go v2.1.0-beta 尼克修改版 -->
<template>
  <div class="min-h-screen bg-gray-50 flex flex-col">
    <div class="flex flex-1">
      <!-- 左侧侧边栏 -->
      <aside class="w-64 bg-slate-900 text-white flex flex-col">
        <div class="p-6 border-b border-slate-800">
          <h1 class="text-xl font-bold tracking-wider">Xiuno Admin</h1>
          <p class="text-xs text-slate-400 mt-1">统治者控制台</p>
        </div>
        <nav class="flex-1 py-4">
          <router-link to="/admin" class="block px-6 py-3 hover:bg-slate-800 transition" active-class="bg-blue-600 text-white font-medium hover:bg-blue-600">📊 控制台概览</router-link>
          <router-link to="/admin/config" class="block px-6 py-3 hover:bg-slate-800 transition" active-class="bg-blue-600 text-white font-medium hover:bg-blue-600">⚙️ 全局配置</router-link>
          <router-link to="/admin/theme" class="block px-6 py-3 hover:bg-slate-800 transition" active-class="bg-blue-600 text-white font-medium hover:bg-blue-600">🎨 外观实验室</router-link>
          <router-link to="/admin/forum" class="block px-6 py-3 hover:bg-slate-800 transition" active-class="bg-blue-600 text-white font-medium hover:bg-blue-600">📁 版块管理</router-link>
          <router-link to="/admin/tag" class="block px-6 py-3 hover:bg-slate-800 transition" active-class="bg-blue-600 text-white font-medium hover:bg-blue-600">🏷️ 标签管理</router-link>
          <router-link to="/admin/thread" class="block px-6 py-3 hover:bg-slate-800 transition" active-class="bg-blue-600 text-white font-medium hover:bg-blue-600">📝 主题管理</router-link>
          <router-link to="/admin/group" class="block px-6 py-3 hover:bg-slate-800 transition" active-class="bg-blue-600 text-white font-medium hover:bg-blue-600">👤 用户组管理</router-link>
          <router-link to="/admin/plugin" class="block px-6 py-3 hover:bg-slate-800 transition" active-class="bg-blue-600 text-white font-medium hover:bg-blue-600">🔌 插件中枢</router-link>
          <router-link to="/admin/user" class="block px-6 py-3 hover:bg-slate-800 transition" active-class="bg-blue-600 text-white font-medium hover:bg-blue-600">👥 用户管控</router-link>
          <router-link to="/admin/modlog" class="block px-6 py-3 hover:bg-slate-800 transition" active-class="bg-blue-600 text-white font-medium hover:bg-blue-600">📋 版务日志</router-link>
        </nav>
        <div class="p-4 border-t border-slate-800">
          <router-link to="/" class="text-sm text-slate-400 hover:text-white">← 返回前台社区</router-link>
        </div>
      </aside>

      <!-- 右侧主内容区 -->
      <main class="flex-1 p-8 overflow-y-auto flex flex-col">
        <div class="max-w-4xl mx-auto bg-white rounded-xl shadow-sm border border-gray-100 p-8 min-h-[70vh] flex-1 w-full">
          <router-view></router-view>
        </div>
        <footer class="text-center text-xs text-gray-400 py-4 mt-4">
          xiuno-go v2.1.0-beta 尼克修改版
        </footer>
      </main>
    </div>
  </div>
</template>
