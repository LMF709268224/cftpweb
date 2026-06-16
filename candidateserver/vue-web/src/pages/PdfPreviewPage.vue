<script setup lang="ts">
import { computed, onErrorCaptured, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { PDFViewer } from "@embedpdf/vue-pdf-viewer"
import { AlertTriangle, ArrowLeft, FileText, Loader2 } from "lucide-vue-next"
import { apiClient } from "@/lib/apiClient"

const route = useRoute()
const router = useRouter()
const viewerSrc = ref("")
const loading = ref(false)
const errorMessage = ref("")
const viewerError = ref(false)

const title = computed(() => String(route.query.title || "PDF Preview"))
const source = computed(() => {
  const lessonId = String(route.query.lessonId || "")
  if (lessonId) return `/api/pipeline/lessons/${encodeURIComponent(lessonId)}/preview-url`

  const src = String(route.query.src || "")
  if (src) return `/api/pipeline/resource-preview-url?src=${encodeURIComponent(src)}`

  return ""
})

const viewerConfig = computed(() => ({
  src: viewerSrc.value,
  theme: {
    preference: "light",
    light: {
      accent: {
        primary: "#101e43",
      },
    },
  },
  tabBar: "never",
  disabledCategories: ["annotation", "print", "export", "redaction"],
}))

async function loadPdf() {
  viewerSrc.value = ""
  errorMessage.value = ""
  viewerError.value = false

  if (!source.value) {
    errorMessage.value = "No PDF resource found for preview."
    return
  }

  loading.value = true
  try {
    const res = await apiClient(source.value, { timeoutMs: 30000 })
    if (!res?.url) {
      errorMessage.value = "No PDF resource found for preview."
      return
    }
    viewerSrc.value = res.url
  } catch (err) {
    errorMessage.value = "PDF preview failed. Please check your network and try again."
  } finally {
    loading.value = false
  }
}

function goBack() {
  if (window.history.length > 1) router.back()
  else router.push("/courses")
}

onErrorCaptured((err) => {
  console.error("PDF viewer failed:", err)
  viewerError.value = true
  errorMessage.value = "The advanced PDF viewer failed to load. A basic browser preview is shown instead."
  return false
})

watch(source, loadPdf, { immediate: true })
</script>

<template>
  <div class="flex h-screen flex-col bg-[#eef8f7]">
    <header class="flex h-16 shrink-0 items-center justify-between border-b border-slate-200 bg-white px-4 shadow-sm">
      <button class="inline-flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-100" @click="goBack">
        <ArrowLeft class="h-4 w-4" />
        Back to Course
      </button>
      <div class="min-w-0 flex-1 px-4 text-center">
        <div class="inline-flex max-w-full items-center gap-2 rounded-full bg-emerald-50 px-4 py-2 text-sm font-semibold text-slate-900">
          <FileText class="h-4 w-4 shrink-0 text-emerald-500" />
          <span class="truncate">{{ title }}</span>
        </div>
      </div>
      <div class="w-[88px]" />
    </header>

    <main class="min-h-0 flex-1 p-3">
      <div v-if="viewerSrc" class="h-full overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-[0_12px_30px_rgba(15,74,82,0.08)]">
        <PDFViewer v-if="!viewerError" class="h-full w-full" :config="viewerConfig" />
        <iframe
          v-else
          :src="viewerSrc"
          class="h-full w-full border-0"
          title="PDF preview fallback"
        />
      </div>
      <div v-else class="flex h-full items-center justify-center rounded-2xl border border-dashed border-slate-300 bg-white text-sm text-slate-500">
        <div v-if="loading" class="flex items-center gap-2">
          <Loader2 class="h-5 w-5 animate-spin text-emerald-500" />
          Loading PDF preview...
        </div>
        <div v-else class="flex max-w-md flex-col items-center gap-3 text-center">
          <div class="rounded-full bg-rose-50 p-4 text-rose-500">
            <AlertTriangle class="h-8 w-8" />
          </div>
          <div class="text-base font-semibold text-slate-900">PDF preview failed</div>
          <p>{{ errorMessage || "No PDF resource found for preview." }}</p>
          <button class="rounded-lg bg-emerald-500 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-emerald-600" @click="loadPdf">
            Reload
          </button>
        </div>
      </div>
    </main>
  </div>
</template>
