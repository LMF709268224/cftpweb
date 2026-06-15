<script setup lang="ts">
import { computed } from "vue"
import { useRoute, useRouter } from "vue-router"
import { PDFViewer } from "@embedpdf/vue-pdf-viewer"
import { ArrowLeft, FileText } from "lucide-vue-next"

const route = useRoute()
const router = useRouter()

const title = computed(() => String(route.query.title || "PDF Preview"))
const source = computed(() => {
  const lessonId = String(route.query.lessonId || "")
  if (!lessonId) return ""
  return `/api/pipeline/lessons/${encodeURIComponent(lessonId)}/preview`
})

const viewerConfig = computed(() => ({
  src: source.value,
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

function goBack() {
  if (window.history.length > 1) router.back()
  else router.push("/courses")
}
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
      <div v-if="source" class="h-full overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-[0_12px_30px_rgba(15,74,82,0.08)]">
        <PDFViewer class="h-full w-full" :config="viewerConfig" />
      </div>
      <div v-else class="flex h-full items-center justify-center rounded-2xl border border-dashed border-slate-300 bg-white text-sm text-slate-500">
        未找到可预览的 PDF 课时
      </div>
    </main>
  </div>
</template>
