<script setup lang="ts">
import { computed, onBeforeUnmount, onErrorCaptured, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { PDFViewer } from "@embedpdf/vue-pdf-viewer"
import { AlertTriangle, ArrowLeft, Calendar, FileText, Loader2, RotateCw } from "lucide-vue-next"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"

const SLOW_PREVIEW_NOTICE_MS =  60 * 1000

const route = useRoute()
const router = useRouter()
const { t } = useTranslation()
const viewerSrc = ref("")
const loading = ref(false)
const viewerReady = ref(false)
const viewerFailed = ref(false)
const slowPreview = ref(false)
type PdfPreviewErrorKey = "pdfNoResource" | "pdfFailed"
const errorMessageKey = ref<PdfPreviewErrorKey | "">("")
const previewTitle = ref("")
const previewExpiresAt = ref("")
const pdfViewerRegistry = ref<any>(null)
let slowPreviewTimer: number | undefined

const routeFileId = computed(() => String(route.params.fileId || ""))
const routeLessonId = computed(() => String(route.params.lessonId || route.query.lessonId || ""))
const routeResourceKey = computed(() => String(route.params.resourceKey || ""))
const storedResourceTitle = computed(() =>
  routeFileId.value ? sessionStorage.getItem(`resource-pack-file-preview-title:${routeFileId.value}`) || "" : "",
)
const storedLessonTitle = computed(() =>
  routeLessonId.value ? sessionStorage.getItem(`lesson-pdf-preview-title:${routeLessonId.value}`) || "" : "",
)
const storedExternalTitle = computed(() =>
  routeResourceKey.value ? sessionStorage.getItem(`external-pdf-preview-title:${routeResourceKey.value}`) || "" : "",
)
const title = computed(() => String(route.query.title || storedResourceTitle.value || storedLessonTitle.value || storedExternalTitle.value || t.value.preview.pdfTitle))
const isResourcePackPreview = computed(() => Boolean(routeFileId.value))
const resourceDetailTitle = computed(() => previewTitle.value || title.value)
const resourceDetailDate = computed(() => formatPreviewDate(previewExpiresAt.value))
const errorMessage = computed(() => (errorMessageKey.value ? t.value.preview[errorMessageKey.value] : ""))
const viewerLocale = computed(() => (t.value.app.htmlLang === "zh-CN" ? "zh-CN" : "en"))
const source = computed(() => {
  const fileId = routeFileId.value
  if (fileId) return `/api/resource-pack-files/${encodeURIComponent(fileId)}/preview-url`

  const lessonId = routeLessonId.value
  if (lessonId) return `/api/pipeline/lessons/${encodeURIComponent(lessonId)}/preview-url`

  const resourceKey = routeResourceKey.value
  if (resourceKey) {
    const src = sessionStorage.getItem(`external-pdf-preview-src:${resourceKey}`) || ""
    return src ? `/api/pipeline/resource-preview-url?src=${encodeURIComponent(src)}` : ""
  }

  const src = String(route.query.src || "")
  if (src) return `/api/pipeline/resource-preview-url?src=${encodeURIComponent(src)}`

  return ""
})

const viewerConfig = computed(() => ({
  src: viewerSrc.value,
  i18n: {
    defaultLocale: viewerLocale.value,
    fallbackLocale: "en",
  },
  theme: {
    preference: "light",
    light: {
      accent: {
        primary: "#101e43",
      },
    },
  },
  tabBar: "never",
  disabledCategories: ["annotation", "insert", "form", "redaction", "panel-comment"],
}))

function clearSlowPreviewTimer() {
  if (slowPreviewTimer) {
    window.clearTimeout(slowPreviewTimer)
    slowPreviewTimer = undefined
  }
}

function startSlowPreviewTimer() {
  clearSlowPreviewTimer()
  slowPreviewTimer = window.setTimeout(() => {
    slowPreview.value = true
  }, SLOW_PREVIEW_NOTICE_MS)
}

function syncViewerLocale() {
  const i18n = pdfViewerRegistry.value?.getPlugin?.("i18n")?.provides?.()
  i18n?.setLocale?.(viewerLocale.value)
}

function padDatePart(value: number) {
  return String(value).padStart(2, "0")
}

function formatPreviewDate(value: string) {
  if (!value) return ""

  const numericValue = Number(value)
  const date = Number.isFinite(numericValue) && value.trim() !== ""
    ? new Date(numericValue < 10000000000 ? numericValue * 1000 : numericValue)
    : new Date(value)

  if (Number.isNaN(date.getTime())) return value

  const year = date.getFullYear()
  const month = padDatePart(date.getMonth() + 1)
  const day = padDatePart(date.getDate())
  const hours = padDatePart(date.getHours())
  const minutes = padDatePart(date.getMinutes())
  const seconds = padDatePart(date.getSeconds())
  return `${year}/${month}/${day} ${hours}:${minutes}:${seconds}`
}

async function loadPdf() {
  clearSlowPreviewTimer()
  viewerSrc.value = ""
  viewerReady.value = false
  viewerFailed.value = false
  slowPreview.value = false
  errorMessageKey.value = ""
  previewTitle.value = ""
  previewExpiresAt.value = ""

  if (!source.value) {
    errorMessageKey.value = "pdfNoResource"
    return
  }

  loading.value = true
  try {
    const res = await apiClient(source.value, { timeoutMs: 60000 })
    if (!res?.url) {
      errorMessageKey.value = "pdfNoResource"
      return
    }
    previewTitle.value = String(res.title || "")
    previewExpiresAt.value = String(res.expires_at || "")
    viewerSrc.value = res.url
    startSlowPreviewTimer()
  } catch (err) {
    errorMessageKey.value = "pdfFailed"
  } finally {
    loading.value = false
  }
}

function handleViewerReady(registry?: any) {
  viewerReady.value = true
  if (registry) {
    pdfViewerRegistry.value = registry
    syncViewerLocale()

    if (typeof registry.getPlugin === 'function') {
      const scrollPlugin = registry.getPlugin('scroll')
      if (scrollPlugin && scrollPlugin.capability) {
        const cap = scrollPlugin.capability
        const storageKey = `pdf-bookmark:${routeFileId.value || routeLessonId.value || routeResourceKey.value}`
        const savedPage = localStorage.getItem(storageKey)
        
        if (savedPage) {
          cap.onLayoutReady((e: any) => {
            if (e.isInitial) {
              setTimeout(() => {
                cap.scrollToPage({ pageNumber: parseInt(savedPage, 10) })
              }, 100)
            }
          })
        }
        
        cap.onPageChange((e: any) => {
          localStorage.setItem(storageKey, e.pageNumber.toString())
        })
      }
    }
  }
}

function goBack() {
  if (window.opener) {
    window.close()
  } else if (window.history.length > 1) {
    router.back()
  } else {
    router.push("/my-certifications")
  }
}

function goBackFromResourcePack() {
  if (window.opener) {
    window.close()
  } else if (window.history.length > 1) {
    router.back()
  } else {
    router.push("/resource-packs")
  }
}

watch(source, loadPdf, { immediate: true })
watch(viewerLocale, syncViewerLocale)
onBeforeUnmount(clearSlowPreviewTimer)

onErrorCaptured((err) => {
  console.error("PDF viewer failed:", err)
  viewerFailed.value = true
  slowPreview.value = true
  clearSlowPreviewTimer()
  return false
})
</script>

<template>
  <div v-if="isResourcePackPreview" class="flex h-screen flex-col overflow-hidden bg-slate-50">
    <header class="shrink-0 px-4 pb-8 pt-4 sm:px-8 sm:pb-16 sm:pt-8">
      <button class="inline-flex h-10 items-center gap-3 rounded-lg border border-slate-200 bg-white px-4 text-sm font-semibold text-slate-800 shadow-sm transition-colors hover:bg-slate-100" @click="goBackFromResourcePack">
        <ArrowLeft class="h-4 w-4" />
        {{ t.preview.backToInsights }}
      </button>
    </header>

    <main class="grid min-h-0 flex-1 gap-5 px-4 pb-4 sm:px-8 lg:grid-cols-[minmax(0,1fr)_468px] lg:gap-8">
      <section class="min-h-0 overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
        <div v-if="viewerSrc" :key="viewerSrc" class="relative h-full overflow-hidden">
          <PDFViewer
            v-if="!viewerFailed"
            class="h-full w-full"
            :config="viewerConfig"
            @ready="handleViewerReady"
          />
          <iframe
            v-else
            :src="viewerSrc"
            class="h-full w-full border-0"
            :title="t.preview.pdfFallbackTitle"
          />
          <div v-if="!viewerReady && !viewerFailed" class="absolute inset-0 z-10 flex items-center justify-center bg-white/90 text-sm text-slate-600 backdrop-blur-[1px]">
            <div class="flex max-w-sm flex-col items-center gap-4 rounded-2xl border border-slate-200 bg-white px-6 py-5 text-center shadow-[0_16px_40px_rgba(15,74,82,0.12)]">
              <Loader2 class="h-5 w-5 animate-spin text-blue-500" />
              <div class="space-y-1">
                <div class="font-semibold text-slate-900">{{ t.preview.pdfViewerLoading }}</div>
                <p v-if="slowPreview" class="text-xs leading-5 text-slate-500">
                  {{ t.preview.pdfSlowNotice }}
                </p>
              </div>
              <div class="flex flex-wrap justify-center gap-2">
                <button class="inline-flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-2 text-xs font-semibold text-slate-700 transition-colors hover:bg-slate-50" @click="loadPdf">
                  <RotateCw class="h-3.5 w-3.5" />
                  {{ t.preview.reload }}
                </button>
              </div>
            </div>
          </div>
        </div>
        <div v-else class="flex h-full items-center justify-center text-sm text-slate-500">
          <div v-if="loading" class="flex items-center gap-2">
            <Loader2 class="h-5 w-5 animate-spin text-blue-500" />
            {{ t.preview.pdfPreviewLoading }}
          </div>
          <div v-else class="flex max-w-md flex-col items-center gap-3 px-6 text-center">
            <div class="rounded-full bg-rose-50 p-4 text-rose-500">
              <AlertTriangle class="h-8 w-8" />
            </div>
            <div class="text-base font-semibold text-slate-900">{{ t.preview.pdfFailedTitle }}</div>
            <p>{{ errorMessage || t.preview.pdfNoResource }}</p>
            <button class="rounded-lg bg-blue-500 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-blue-600" @click="loadPdf">
              {{ t.preview.reload }}
            </button>
          </div>
        </div>
      </section>

      <aside class="h-fit self-start rounded-xl border border-slate-200 bg-white p-8 shadow-sm">
        <h1 class="text-xl font-bold leading-tight text-slate-950">
          {{ resourceDetailTitle }}
        </h1>
        <div v-if="resourceDetailDate" class="mt-6 flex items-center gap-3 text-sm text-slate-500">
          <Calendar class="h-4 w-4 text-slate-600" />
          <span>{{ resourceDetailDate }}</span>
        </div>
      </aside>
    </main>
  </div>

  <div v-else class="flex h-screen flex-col bg-[#eef8f7]">
    <header class="flex h-16 shrink-0 items-center justify-between border-b border-slate-200 bg-white px-4 shadow-sm">
      <!-- The back button is removed because the page opens in a new window -->
      <div class="w-[88px]"></div>
      <div class="min-w-0 flex-1 px-4 text-center">
        <div class="inline-flex max-w-full items-center gap-2 rounded-full bg-emerald-50 px-4 py-2 text-sm font-semibold text-slate-900">
          <FileText class="h-4 w-4 shrink-0 text-emerald-500" />
          <span class="truncate">{{ title }}</span>
        </div>
      </div>
      <div class="w-[88px]" />
    </header>

    <main class="min-h-0 flex-1 p-3">
      <div v-if="viewerSrc" :key="viewerSrc" class="relative h-full overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-[0_12px_30px_rgba(15,74,82,0.08)]">
        <PDFViewer
          v-if="!viewerFailed"
          class="h-full w-full"
          :config="viewerConfig"
          @ready="handleViewerReady"
        />
        <iframe
          v-else
          :src="viewerSrc"
          class="h-full w-full border-0"
          :title="t.preview.pdfFallbackTitle"
        />
        <div v-if="!viewerReady && !viewerFailed" class="absolute inset-0 z-10 flex items-center justify-center bg-white/90 text-sm text-slate-600 backdrop-blur-[1px]">
          <div class="flex max-w-sm flex-col items-center gap-4 rounded-2xl border border-slate-200 bg-white px-6 py-5 text-center shadow-[0_16px_40px_rgba(15,74,82,0.12)]">
            <Loader2 class="h-5 w-5 animate-spin text-emerald-500" />
            <div class="space-y-1">
              <div class="font-semibold text-slate-900">{{ t.preview.pdfViewerLoading }}</div>
              <p v-if="slowPreview" class="text-xs leading-5 text-slate-500">
                {{ t.preview.pdfSlowNotice }}
              </p>
            </div>
            <div class="flex flex-wrap justify-center gap-2">
              <button class="inline-flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-2 text-xs font-semibold text-slate-700 transition-colors hover:bg-slate-50" @click="loadPdf">
                <RotateCw class="h-3.5 w-3.5" />
                {{ t.preview.reload }}
              </button>
            </div>
          </div>
        </div>
      </div>
      <div v-else class="flex h-full items-center justify-center rounded-2xl border border-dashed border-slate-300 bg-white text-sm text-slate-500">
        <div v-if="loading" class="flex items-center gap-2">
          <Loader2 class="h-5 w-5 animate-spin text-emerald-500" />
          {{ t.preview.pdfPreviewLoading }}
        </div>
        <div v-else class="flex max-w-md flex-col items-center gap-3 text-center">
          <div class="rounded-full bg-rose-50 p-4 text-rose-500">
            <AlertTriangle class="h-8 w-8" />
          </div>
          <div class="text-base font-semibold text-slate-900">{{ t.preview.pdfFailedTitle }}</div>
          <p>{{ errorMessage || t.preview.pdfNoResource }}</p>
          <button class="rounded-lg bg-emerald-500 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-emerald-600" @click="loadPdf">
            {{ t.preview.reload }}
          </button>
        </div>
      </div>
    </main>
  </div>
</template>
