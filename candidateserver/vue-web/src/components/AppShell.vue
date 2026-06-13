<script setup lang="ts">
import Sidebar from "./Sidebar.vue"
</script>

<template>
  <div class="min-h-screen bg-background">
    <Sidebar />
    <main class="page-main">
      <div class="px-8 py-8">
        <slot />
      </div>
    </main>
  </div>
</template>
