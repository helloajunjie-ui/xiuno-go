<script setup lang="ts">
import NavBar from './components/layout/NavBar.vue'
</script>

<template>
  <div class="min-h-screen bg-gray-50">
    <NavBar />
    <main class="max-w-4xl mx-auto px-4 py-6">
      <router-view />
    </main>
  </div>
</template>
