<script setup lang="ts">
import { Check, Copy } from "lucide-vue-next"
import { computed, ref } from "vue"
import { toast } from "vue-sonner"
import { copyTextToClipboard } from "@/lib/clipboard"

const props = withDefaults(defineProps<{
  title: string
  value?: unknown
  text?: string
  copyLabel: string
  copiedLabel: string
  copiedMessage?: string
  copyErrorMessage?: string
  maxHeight?: string
}>(), {
  maxHeight: "420px",
})

const copied = ref(false)
let copiedTimer: number | undefined

const previewText = computed(() => props.text ?? JSON.stringify(props.value ?? {}, null, 2))

async function copyJson() {
  try {
    await copyTextToClipboard(previewText.value)
    copied.value = true
    if (props.copiedMessage) toast.success(props.copiedMessage)
    if (copiedTimer) window.clearTimeout(copiedTimer)
    copiedTimer = window.setTimeout(() => {
      copied.value = false
    }, 1600)
  } catch (err) {
    console.error(err)
    if (props.copyErrorMessage) toast.error(props.copyErrorMessage)
  }
}
</script>

<template>
  <details class="rounded-2xl border border-slate-200 bg-white p-4">
    <summary class="cursor-pointer text-sm font-black text-slate-700">{{ title }}</summary>
    <div class="mt-4 overflow-hidden rounded-2xl bg-slate-950">
      <div class="flex items-center justify-between gap-3 border-b border-white/10 px-4 py-3">
        <span class="text-xs font-black uppercase text-slate-400">{{ title }}</span>
        <button class="inline-flex h-8 items-center gap-2 rounded-lg border border-white/10 px-3 text-xs font-bold text-slate-100 transition hover:bg-white/10" type="button" @click="copyJson">
          <Check v-if="copied" class="h-3.5 w-3.5" />
          <Copy v-else class="h-3.5 w-3.5" />
          {{ copied ? copiedLabel : copyLabel }}
        </button>
      </div>
      <pre class="overflow-auto p-5 text-xs leading-6 text-slate-100" :style="{ maxHeight }">{{ previewText }}</pre>
    </div>
  </details>
</template>
