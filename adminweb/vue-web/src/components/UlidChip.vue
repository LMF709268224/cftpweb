<script setup lang="ts">
import { Check, Copy } from "lucide-vue-next"
import { computed, onBeforeUnmount, ref } from "vue"
import { copyTextToClipboard } from "@/lib/clipboard"

const props = withDefaults(
  defineProps<{
    value: string
    truncate?: boolean
  }>(),
  {
    truncate: true,
  },
)

const copied = ref(false)
let resetTimer: number | undefined

const displayValue = computed(() => {
  const raw = (props.value || "").trim()
  if (!raw) return "-"
  if (!props.truncate || raw.length <= 14) return raw
  return `${raw.slice(0, 6)}…${raw.slice(-5)}`
})

async function handleCopy() {
  const raw = (props.value || "").trim()
  if (!raw) return
  try {
    await copyTextToClipboard(raw)
    copied.value = true
    if (resetTimer) window.clearTimeout(resetTimer)
    resetTimer = window.setTimeout(() => {
      copied.value = false
    }, 1200)
  } catch (err) {
    console.error(err)
  }
}

onBeforeUnmount(() => {
  if (resetTimer) window.clearTimeout(resetTimer)
})
</script>

<template>
  <span v-if="!(props.value || '').trim()" class="tnum text-xs text-slate-400">-</span>
  <button
    v-else
    type="button"
    class="tnum inline-flex max-w-full items-center gap-1 rounded-full bg-slate-100 px-2 py-0.5 font-mono text-xs text-slate-500 transition-colors hover:bg-slate-200/70 hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30"
    :title="props.value"
    @click.stop="handleCopy"
  >
    <span class="truncate">{{ displayValue }}</span>
    <Check v-if="copied" class="h-3 w-3 shrink-0 text-success-600" />
    <Copy v-else class="h-3 w-3 shrink-0 text-slate-400" />
  </button>
</template>
