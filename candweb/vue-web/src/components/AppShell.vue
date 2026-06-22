<script setup lang="ts">
import { onBeforeUnmount, onMounted } from "vue"
import Sidebar from "./Sidebar.vue"
import { disposeSidebarCollapse, initializeSidebarCollapse, useSidebarCollapse } from "@/lib/sidebar"

defineProps<{
  contentClass?: string
}>()

const { isSidebarCollapsed, toggleSidebarCollapsed } = useSidebarCollapse()

function handleSidebarToggleEvent(event: Event) {
  const target = event.target as Element | null
  if (!target?.closest(".page-panel > header > svg:first-child")) return

  if (typeof window !== "undefined" && !window.matchMedia("(min-width: 1024px)").matches) {
    window.dispatchEvent(new Event("open-mobile-sidebar"))
    return
  }

  toggleSidebarCollapsed()
}

onMounted(() => {
  initializeSidebarCollapse()
  document.addEventListener("click", handleSidebarToggleEvent)
})

onBeforeUnmount(() => {
  document.removeEventListener("click", handleSidebarToggleEvent)
  disposeSidebarCollapse()
})
</script>

<template>
  <div class="bg-background">
    <Sidebar />
    <main class="page-main">
      <div :class="contentClass || 'px-4 py-4'">
        <slot />
      </div>
    </main>
  </div>
</template>
