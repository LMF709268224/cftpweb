<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useRoute, useRouter } from "vue-router"
import { AlertTriangle, ArrowLeft, BookOpen, ExternalLink, FileText, Loader2, Play } from "lucide-vue-next"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"
import {
  normalizeSupplementaryMaterials,
  parseSupplementaryMaterialItems,
  isPdfResourceUrl,
  type SupplementaryMaterial,
  type SupplementaryMaterialItem,
} from "@/lib/supplementaryMaterials"

type CompleteCourse = {
  supplementary_material?: SupplementaryMaterial | SupplementaryMaterial[]
  supplementaryMaterial?: SupplementaryMaterial | SupplementaryMaterial[]
  course?: {
    title?: string
  }
}

type CourseResponse = {
  complete_course?: CompleteCourse
  supplementary_material?: SupplementaryMaterial | SupplementaryMaterial[]
  supplementaryMaterial?: SupplementaryMaterial | SupplementaryMaterial[]
}

const route = useRoute()
const router = useRouter()
const { t } = useTranslation()
const payload = ref<CourseResponse | null>(null)
const loading = ref(false)

const courseId = computed(() => String(route.params.courseId || route.query.courseId || ""))
const pipelineId = computed(() => String(route.params.pipelineId || route.query.pipelineId || ""))
const fallbackTitle = computed(() => String(route.query.title || t.value.common.unknownCourse))
const courseTitle = computed(() => payload.value?.complete_course?.course?.title || fallbackTitle.value)
const supplementaryMaterials = computed(() => {
  const completeCourse = payload.value?.complete_course
  const raw =
    completeCourse?.supplementary_material ??
    completeCourse?.supplementaryMaterial ??
    payload.value?.supplementary_material ??
    payload.value?.supplementaryMaterial
  return normalizeSupplementaryMaterials(raw)
})
const items = computed(() => parseSupplementaryMaterialItems(supplementaryMaterials.value, t.value.learning.supplementaryChapterHeader))

function supplementaryChapterLabel(item: SupplementaryMaterialItem, index: number) {
  return items.value[index - 1]?.chapter === item.chapter ? "" : item.chapter
}

function supplementaryTypeLabel(type: string) {
  const normalized = type.trim().toLowerCase()
  if (normalized === "article") return t.value.learning.supplementaryTypeArticle
  if (normalized === "video") return t.value.learning.supplementaryTypeVideo
  if (normalized === "pdf") return t.value.learning.supplementaryTypePdf
  if (normalized === "link") return t.value.learning.supplementaryTypeLink
  return type || t.value.learning.materialTypeUnknown
}

function supplementaryTypeClass(type: string) {
  const normalized = type.trim().toLowerCase()
  if (normalized === "video") return "border-violet-200 bg-violet-100 text-violet-700"
  if (normalized === "article" || normalized === "pdf") return "border-blue-200 bg-blue-100 text-blue-700"
  return "border-slate-200 bg-slate-100 text-slate-700"
}

function supplementaryTypeIcon(type: string) {
  return type.trim().toLowerCase() === "video" ? Play : FileText
}

function backLink() {
  return courseId.value && pipelineId.value
    ? `/certifications/${encodeURIComponent(pipelineId.value)}/learn/${encodeURIComponent(courseId.value)}`
    : "/certifications"
}

function goBack() {
  if (window.history.length > 1) router.back()
  else router.push(backLink())
}

function openResource(item: SupplementaryMaterialItem) {
  if (!item.url) return
  if (isPdfResourceUrl(item.url)) {
    openExternalPdfPreview(item.url, item.title || t.value.learning.supplementaryDefaultTitle)
    return
  }

  window.open(item.url, "_blank", "noopener,noreferrer")
}

function openPreviewTab(url: string) {
  const link = document.createElement("a")
  link.href = url
  link.target = "_blank"
  link.rel = "noopener noreferrer"
  document.body.appendChild(link)
  link.click()
  link.remove()
}

function openExternalPdfPreview(src: string, title: string) {
  const resourceKey = crypto.randomUUID()
  sessionStorage.setItem(`external-pdf-preview-src:${resourceKey}`, src)
  sessionStorage.setItem(`external-pdf-preview-title:${resourceKey}`, title)
  openPreviewTab(`/pdf-preview/resources/${encodeURIComponent(resourceKey)}`)
}

async function loadCourse() {
  if (!courseId.value) return
  loading.value = true
  try {
    payload.value = await apiClient(`/api/pipeline/courses/${courseId.value}/complete`)
  } finally {
    loading.value = false
  }
}

onMounted(loadCourse)
</script>

<template>
  <div class="min-h-screen bg-[#eef8f7]">
    <header class="sticky top-0 z-10 border-b border-slate-200 bg-white/95 backdrop-blur">
      <div class="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-4">
        <button class="inline-flex items-center gap-2 text-sm font-medium text-slate-600 transition-colors hover:text-slate-900" @click="goBack">
          <ArrowLeft class="h-4 w-4" />
          {{ t.learning.backToCourse }}
        </button>
        <div class="min-w-0 flex-1 text-center">
          <div class="truncate text-sm text-slate-500">{{ courseTitle }}</div>
          <h1 class="truncate text-lg font-bold text-slate-950">{{ t.learning.supplementaryMaterialsTitle }}</h1>
        </div>
        <span class="badge border-slate-200 bg-slate-50 text-slate-700">{{ items.length }} {{ t.learning.materialsCountSuffix }}</span>
      </div>
    </header>

    <main class="mx-auto max-w-6xl px-4 py-6">
      <div v-if="loading" class="flex items-center justify-center gap-2 rounded-2xl bg-white py-16 text-slate-500">
        <Loader2 class="h-5 w-5 animate-spin text-emerald-500" />
        {{ t.learning.supplementaryMaterialsLoading }}
      </div>

      <div v-else-if="items.length === 0" class="flex flex-col items-center justify-center gap-3 rounded-2xl bg-white py-16 text-center text-slate-500">
        <AlertTriangle class="h-10 w-10 text-amber-500" />
        <div class="text-base font-semibold text-slate-900">{{ t.learning.supplementaryMaterialsEmpty }}</div>
        <p class="text-sm">{{ t.learning.supplementaryMaterialsEmptyDesc }}</p>
      </div>

      <div v-else class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-[0_12px_30px_rgba(15,74,82,0.08)]">
        <div class="border-b border-slate-100 bg-slate-50 px-5 py-4">
          <div class="flex items-center gap-2 text-base font-bold text-slate-950">
            <BookOpen class="h-5 w-5 text-emerald-500" />
            {{ t.learning.supplementaryMaterialsTitle }}
          </div>
          <p class="mt-1 text-sm text-slate-500">{{ t.learning.supplementaryMaterialsDesc }}</p>
        </div>

        <div class="hidden grid-cols-[minmax(180px,0.9fr)_120px_minmax(260px,2fr)_120px] border-b border-slate-100 px-5 py-3 text-sm font-medium text-slate-500 md:grid">
          <div>{{ t.learning.supplementaryChapterHeader }}</div>
          <div>{{ t.learning.supplementaryTypeHeader }}</div>
          <div>{{ t.learning.supplementaryTitleDescHeader }}</div>
          <div>{{ t.learning.supplementaryActionHeader }}</div>
        </div>

        <div class="divide-y divide-slate-100">
          <div
            v-for="(item, index) in items"
            :key="item.key"
            class="grid gap-3 px-5 py-4 text-sm md:grid-cols-[minmax(180px,0.9fr)_120px_minmax(260px,2fr)_120px]"
          >
            <div class="font-medium text-slate-700">
              <span class="md:hidden text-xs text-slate-400">{{ t.learning.supplementaryChapterPrefix }} </span>
              {{ supplementaryChapterLabel(item, index) }}
            </div>
            <div>
              <span class="badge gap-1 border text-xs" :class="supplementaryTypeClass(item.type)">
                <component :is="supplementaryTypeIcon(item.type)" class="h-3 w-3" />
                {{ supplementaryTypeLabel(item.type) }}
              </span>
            </div>
            <div>
              <div class="font-semibold text-slate-950">{{ item.title }}</div>
              <p v-if="item.description" class="mt-1 text-xs leading-relaxed text-slate-500">{{ item.description }}</p>
              <p v-if="item.url" class="mt-1 truncate text-xs text-emerald-600">{{ item.url }}</p>
            </div>
            <div>
              <button
                v-if="item.url"
                class="inline-flex items-center gap-1 rounded-lg bg-emerald-500 px-3 py-2 text-xs font-semibold text-white transition-colors hover:bg-emerald-600"
                @click="openResource(item)"
              >
                <ExternalLink class="h-3.5 w-3.5" />
                {{ t.learning.supplementaryPreview }}
              </button>
              <span v-else class="text-xs text-slate-400">{{ t.learning.supplementaryNoUrl }}</span>
            </div>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>
