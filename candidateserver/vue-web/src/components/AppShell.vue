<script setup lang="ts">
import Sidebar from "./Sidebar.vue"

defineProps<{
  contentClass?: string
}>()
</script>

<template>
  <div class="min-h-screen bg-background">
    <Sidebar />
    <main class="page-main">
      <div :class="contentClass || 'px-4 py-4'">
        <slot />
      </div>
    </main>
  </div>
</template>
