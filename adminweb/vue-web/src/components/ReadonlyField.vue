<script setup lang="ts">
import { computed } from "vue"

const props = withDefaults(defineProps<{
  label: string
  value?: unknown
  text?: unknown
  emptyText?: string
  mono?: boolean
  minHeight?: string
  maxHeight?: string
}>(), {
  emptyText: "-",
  minHeight: "44px",
})

function stringifyValue(value: unknown) {
  if (value === null || value === undefined || value === "") return props.emptyText
  if (typeof value === "object") {
    try {
      return JSON.stringify(value, null, 2)
    } catch {
      return String(value)
    }
  }
  return String(value)
}

const displayText = computed(() => props.text !== undefined ? stringifyValue(props.text) : stringifyValue(props.value))
</script>

<template>
  <div class="grid gap-1.5 text-sm">
    <span class="text-xs font-medium text-slate-500">{{ label }}</span>
    <div
      class="overflow-auto whitespace-pre-wrap break-words rounded-[10px] border border-slate-200 bg-slate-50 px-3.5 py-2.5 leading-5 text-slate-700"
      :class="mono ? 'tnum font-mono text-xs' : 'text-sm'"
      :style="{ minHeight, maxHeight }"
    >
      {{ displayText }}
    </div>
  </div>
</template>
