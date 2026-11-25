<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from "vue"
import { apiClient } from "@/lib/apiClient"
import { useCourseAuthoringCopy } from "@/lib/courseAuthoringCopy"

const props = defineProps<{ objectKey: string }>()
const copy = useCourseAuthoringCopy()
const url = ref("")
const loading = ref(false)
const failed = ref(false)
const expired = ref(false)
let controller: AbortController | undefined
let expiryTimer: ReturnType<typeof setTimeout> | undefined

function reset() {
  controller?.abort()
  if (expiryTimer !== undefined) clearTimeout(expiryTimer)
  expiryTimer = undefined
  url.value = ""
  loading.value = false
  failed.value = false
  expired.value = false
}

async function preview() {
  reset()
  if (!props.objectKey) return
  const objectKey = props.objectKey
  const request = new AbortController()
  controller = request
  loading.value = true
  try {
    const result = await apiClient<{ object_key?: string; view_url?: string; expires_at?: string }>("/api/lms/view-url", {
      method: "POST", signal: request.signal, body: JSON.stringify({ object_key: objectKey }),
    })
    if (request.signal.aborted) return
    if (result.object_key !== objectKey || typeof result.view_url !== "string" || typeof result.expires_at !== "string") throw new Error("Incomplete PDF preview response")
    const address = new URL(result.view_url)
    const remaining = Date.parse(result.expires_at) - Date.now()
    if (address.protocol !== "https:" || address.username || address.password || !Number.isFinite(remaining) || remaining <= 0 || remaining > 2_147_483_647) throw new Error("Invalid PDF preview response")
    url.value = result.view_url
    expiryTimer = setTimeout(() => { url.value = ""; expired.value = true }, remaining)
  } catch {
    if (!request.signal.aborted) failed.value = true
  } finally {
    if (!request.signal.aborted) loading.value = false
  }
}

watch(() => props.objectKey, reset)
onBeforeUnmount(reset)
</script>

<template>
  <section class="space-y-3 rounded-xl border border-slate-200 bg-slate-50 p-4" :aria-label="copy.pdfPreview">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h3 class="text-sm font-bold">{{ copy.pdfPreview }}</h3>
      <button type="button" :disabled="loading || !objectKey" class="rounded-lg border border-blue-200 bg-white px-3 py-2 text-sm font-bold text-blue-700 disabled:opacity-40" @click="preview">{{ loading ? copy.pdfLoading : url || expired || failed ? copy.pdfRefresh : copy.pdfOpen }}</button>
    </div>
    <p class="text-xs text-slate-500">{{ copy.pdfSaved }}</p>
    <p v-if="failed" role="alert" class="text-sm text-red-700">{{ copy.pdfFailed }}</p>
    <p v-if="expired" role="status" class="text-sm text-amber-800">{{ copy.pdfExpired }}</p>
    <template v-if="url">
      <iframe :src="url" :title="copy.pdfPreview" class="h-[60vh] min-h-80 w-full rounded-lg border bg-white" referrerpolicy="no-referrer" />
      <a :href="url" target="_blank" rel="noopener noreferrer" class="inline-block text-sm font-bold text-blue-700 underline">{{ copy.pdfNewTab }}</a>
    </template>
  </section>
</template>
