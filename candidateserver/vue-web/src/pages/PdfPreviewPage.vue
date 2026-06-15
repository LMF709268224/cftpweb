<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { PDFViewer } from "@embedpdf/vue-pdf-viewer"
import { AlertTriangle, ArrowLeft, FileText, Loader2 } from "lucide-vue-next"

const route = useRoute()
const router = useRouter()
const viewerSrc = ref("")
const loading = ref(false)
const errorMessage = ref("")
let objectUrl = ""

const title = computed(() => String(route.query.title || "PDF Preview"))
const source = computed(() => {
  const lessonId = String(route.query.lessonId || "")
  if (!lessonId) return ""
  return `/api/pipeline/lessons/${encodeURIComponent(lessonId)}/preview`
})

const viewerConfig = computed(() => ({
  src: viewerSrc.value,
  theme: {
    preference: "light",
    light: {
      accent: {
        primary: "#36c39f",
      },
    },
  },
  tabBar: "never",
  disabledCategories: ["annotation", "print", "export", "redaction"],
}))

async function loadPdf() {
  cleanupObjectUrl()
  errorMessage.value = ""

  if (!source.value) {
    errorMessage.value = "未找到可预览的 PDF 课时"
    return
  }

  loading.value = true
  try {
    const headers = new Headers()
    const token = localStorage.getItem("access_token")
    if (token) headers.set("Authorization", `Bearer ${token}`)

    const res = await fetch(source.value, {
      credentials: "include",
      headers,
    })

    if (!res.ok) {
      errorMessage.value = res.status === 401 ? "登录已过期或没有权限，请重新登录后再试。" : "PDF 加载失败，请稍后重试。"
      return
    }

    const blob = await res.blob()
    objectUrl = URL.createObjectURL(new Blob([blob], { type: "application/pdf" }))
    viewerSrc.value = objectUrl
  } catch {
    errorMessage.value = "PDF 加载失败，请检查网络后重试。"
  } finally {
    loading.value = false
  }
}

function cleanupObjectUrl() {
  if (objectUrl) URL.revokeObjectURL(objectUrl)
  objectUrl = ""
  viewerSrc.value = ""
}

function goBack() {
  if (window.history.length > 1) router.back()
  else router.push("/courses")
}

watch(source, loadPdf, { immediate: true })
onBeforeUnmount(cleanupObjectUrl)
</script>

<template>
  <div class="flex h-screen flex-col bg-[#eef8f7]">
    <header class="flex h-16 shrink-0 items-center justify-between border-b border-slate-200 bg-white px-4 shadow-sm">
      <button class="inline-flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-100" @click="goBack">
        <ArrowLeft class="h-4 w-4" />
        返回课程
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
        <PDFViewer class="h-full w-full" :config="viewerConfig" />
      </div>
      <div v-else class="flex h-full items-center justify-center rounded-2xl border border-dashed border-slate-300 bg-white text-sm text-slate-500">
        <div v-if="loading" class="flex items-center gap-2">
          <Loader2 class="h-5 w-5 animate-spin text-emerald-500" />
          正在加载 PDF 预览...
        </div>
        <div v-else class="flex max-w-md flex-col items-center gap-3 text-center">
          <div class="rounded-full bg-rose-50 p-4 text-rose-500">
            <AlertTriangle class="h-8 w-8" />
          </div>
          <div class="text-base font-semibold text-slate-900">PDF 加载失败</div>
          <p>{{ errorMessage || "未找到可预览的 PDF 课时" }}</p>
          <button class="rounded-lg bg-emerald-500 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-emerald-600" @click="loadPdf">
            重新加载
          </button>
        </div>
      </div>
    </main>
  </div>
</template>
