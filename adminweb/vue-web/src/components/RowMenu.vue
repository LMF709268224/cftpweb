<script setup lang="ts">
import { MoreHorizontal } from "lucide-vue-next"
import type { Component } from "vue"
import { computed, onBeforeUnmount, onMounted, ref } from "vue"
import { useAdminLanguage } from "@/lib/language"

export type RowMenuItem = {
  key: string
  label: string
  icon?: Component
  danger?: boolean
  disabled?: boolean
}

const props = defineProps<{
  items: RowMenuItem[]
}>()

const emit = defineEmits<{
  (e: "select", key: string): void
}>()

const { t } = useAdminLanguage()

const rootEl = ref<HTMLElement | null>(null)
const open = ref(false)

const safeItems = computed(() => props.items.filter((item) => item.label !== null && item.label !== undefined))

function firstDangerIndex() {
  return safeItems.value.findIndex((item) => item.danger)
}

function showDividerBefore(index: number) {
  const dangerIndex = firstDangerIndex()
  return dangerIndex > 0 && index === dangerIndex
}

function toggle() {
  open.value = !open.value
}

function onSelect(item: RowMenuItem) {
  if (item.disabled) return
  open.value = false
  emit("select", item.key)
}

function onPointerDown(event: PointerEvent) {
  if (!open.value) return
  if (rootEl.value && event.target instanceof Node && !rootEl.value.contains(event.target)) {
    open.value = false
  }
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === "Escape") {
    open.value = false
  }
}

onMounted(() => {
  document.addEventListener("pointerdown", onPointerDown)
})

onBeforeUnmount(() => {
  document.removeEventListener("pointerdown", onPointerDown)
})
</script>

<template>
  <div ref="rootEl" class="relative inline-flex" @keydown="onKeydown">
    <button
      type="button"
      class="inline-flex h-8 w-8 items-center justify-center rounded-lg text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30"
      :class="open ? 'bg-slate-100 text-slate-600' : ''"
      :aria-label="t.common.moreActions"
      :title="t.common.moreActions"
      aria-haspopup="menu"
      :aria-expanded="open"
      @click.stop="toggle"
    >
      <MoreHorizontal class="h-4 w-4" />
    </button>

    <Transition name="menu">
      <div
        v-if="open"
        class="absolute right-0 top-full z-30 mt-1.5 min-w-[10rem] overflow-hidden rounded-xl border border-edge bg-white py-1 shadow-lg shadow-slate-900/5"
        role="menu"
      >
        <template v-for="(item, index) in safeItems" :key="item.key">
          <div v-if="showDividerBefore(index)" class="my-1 border-t border-hairline" role="separator" />
          <button
            type="button"
            role="menuitem"
            class="flex w-full items-center gap-2 whitespace-nowrap px-3 py-2 text-left text-sm transition-colors disabled:pointer-events-none disabled:opacity-50"
            :class="item.danger ? 'text-danger-600 hover:bg-danger-50' : 'text-slate-600 hover:bg-slate-50 hover:text-slate-900'"
            :disabled="item.disabled"
            @click.stop="onSelect(item)"
          >
            <component
              :is="item.icon"
              v-if="item.icon"
              class="h-4 w-4 shrink-0"
              :class="item.danger ? 'text-danger-600' : 'text-slate-400'"
            />
            {{ item.label }}
          </button>
        </template>
      </div>
    </Transition>
  </div>
</template>
