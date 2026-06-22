<script setup lang="ts">
import { onBeforeUnmount, onMounted } from "vue"
import { PanelLeft } from "lucide-vue-next"
import Sidebar from "./Sidebar.vue"
import { disposeSidebarCollapse, initializeSidebarCollapse, useSidebarCollapse } from "@/lib/sidebar"

defineProps<{
  contentClass?: string
}>()

const { isSidebarCollapsed, toggleSidebarCollapsed } = useSidebarCollapse()

onMounted(() => {
  initializeSidebarCollapse()
})

onBeforeUnmount(() => {
  disposeSidebarCollapse()
})
</script>

<template>
  <div class="bg-background">
    <Sidebar />
    <button
      class="app-sidebar-toggle hidden h-8 w-8 items-center justify-center rounded-md text-slate-800 transition-colors duration-200 hover:bg-[#004f8f] hover:text-white lg:inline-flex"
      :aria-label="isSidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'"
      :title="isSidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'"
      type="button"
      @click="toggleSidebarCollapsed"
    >
      <PanelLeft class="h-4 w-4" :stroke-width="2" />
    </button>
    <main class="page-main">
      <div :class="contentClass || 'px-4 py-4'">
        <slot />
      </div>
    </main>
  </div>
</template>
