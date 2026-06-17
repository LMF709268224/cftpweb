<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { AlertTriangle, ArrowLeft, Loader2, Play, RotateCw } from "lucide-vue-next"
import { apiClient } from "@/lib/apiClient"

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const videoSrc = ref("")
const errorMessage = ref("")
const frameLoaded = ref(false)

const fileId = computed(() => String(route.params.fileId || ""))
const storedTitle = computed(() =>
  fileId.value ? sessionStorage.getItem(`resource-pack-file-preview-title:${fileId.value}`) || "" : "",
)
const title = computed(() => String(route.query.title || storedTitle.value || "Video Preview"))

async function loadVideo() {
  videoSrc.value = ""
  errorMessage.value = ""
  frameLoaded.value = false

  if (!fileId.value) {
    errorMessage.value = "No video resource found for preview."
    return
  }

  loading.value = true
  try {
    const resp = await apiClient(`/api/resource-pack-files/${encodeURIComponent(fileId.value)}/view-url`)
    const viewUrl = String(resp?.view_url || resp?.url || "").trim()
    if (!viewUrl) {
      errorMessage.value = "No video URL found for preview."
      return
    }
    videoSrc.value = viewUrl
  } catch {
    errorMessage.value = "Video preview failed. Please check your network and try again."
  } finally {
    loading.value = false
  }
}

function goBack() {
  if (window.history.length > 1) router.back()
  else router.push("/resource-packs")
}

watch(fileId, loadVideo, { immediate: true })
</script>

<template>
  <div class="flex h-screen flex-col bg-[#eef8f7]">
    <header class="flex h-16 shrink-0 items-center justify-between border-b border-slate-200 bg-white px-4 shadow-sm">
      <button class="inline-flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-100" @click="goBack">
        <ArrowLeft class="h-4 w-4" />
        Back to Resources
      </button>
      <div class="min-w-0 flex-1 px-4 text-center">
        <div class="inline-flex max-w-full items-center gap-2 rounded-full bg-emerald-50 px-4 py-2 text-sm font-semibold text-slate-900">
          <Play class="h-4 w-4 shrink-0 text-emerald-500" />
          <span class="truncate">{{ title }}</span>
        </div>
      </div>
      <div class="w-[112px]" />
    </header>

    <main class="min-h-0 flex-1 p-3">
      <div v-if="videoSrc" class="relative h-full overflow-hidden rounded-2xl border border-slate-200 bg-black shadow-[0_12px_30px_rgba(15,74,82,0.08)]">
        <iframe
          :src="videoSrc"
          class="h-full w-full border-0"
          allow="accelerometer; gyroscope; autoplay; encrypted-media; picture-in-picture"
          allowfullscreen
          :title="title"
          @load="frameLoaded = true"
        />
        <div v-if="!frameLoaded" class="absolute inset-0 z-10 flex items-center justify-center bg-black text-sm text-white/80">
          <div class="flex flex-col items-center gap-3 rounded-2xl bg-white/10 px-6 py-5 text-center backdrop-blur">
            <Loader2 class="h-5 w-5 animate-spin text-emerald-300" />
            <div class="font-semibold">Loading video...</div>
          </div>
        </div>
      </div>
      <div v-else class="flex h-full items-center justify-center rounded-2xl border border-dashed border-slate-300 bg-white text-sm text-slate-500">
        <div v-if="loading" class="flex items-center gap-2">
          <Loader2 class="h-5 w-5 animate-spin text-emerald-500" />
          Loading video preview...
        </div>
        <div v-else class="flex max-w-md flex-col items-center gap-3 text-center">
          <div class="rounded-full bg-rose-50 p-4 text-rose-500">
            <AlertTriangle class="h-8 w-8" />
          </div>
          <div class="text-base font-semibold text-slate-900">Video preview failed</div>
          <p>{{ errorMessage || "No video resource found for preview." }}</p>
          <button class="inline-flex items-center gap-2 rounded-lg bg-emerald-500 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-emerald-600" @click="loadVideo">
            <RotateCw class="h-4 w-4" />
            Reload
          </button>
        </div>
      </div>
    </main>
  </div>
</template>
