<script setup lang="ts">
import { onBeforeUnmount, onMounted } from "vue"
import { PanelLeft } from "lucide-vue-next"
import Sidebar from "./Sidebar.vue"
import { disposeSidebarCollapse, initializeSidebarCollapse, useSidebarCollapse } from "@/lib/sidebar"

defineProps<{
  contentClass?: string
}>()

const { isSidebarCollapsed, toggleSidebarCollapsed } = useSidebarCollapse()

function handleSidebarToggle() {
  if (typeof window !== "undefined" && !window.matchMedia("(min-width: 1024px)").matches) {
    window.dispatchEvent(new Event("open-mobile-sidebar"))
    return
  }

  toggleSidebarCollapsed()
}

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
      class="app-sidebar-toggle inline-flex h-8 w-8 items-center justify-center rounded-md text-slate-800 transition-colors duration-200 hover:bg-[#004f8f] hover:text-white"
      :aria-label="isSidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'"
      :title="isSidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'"
      type="button"
      @click="handleSidebarToggle"
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
