<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { RouterLink } from "vue-router"
import { toast } from "vue-sonner"
import {
  BookOpen,
  Bookmark,
  CheckCircle2,
  ChevronRight,
  Clock,
  Download,
  Eye,
  FileIcon,
  FileText,
  Play,
  Search,
  SlidersHorizontal,
  Video,
} from "lucide-vue-next"
import { CANDIDATE_PIPELINE_STATUS_LABELS, statusLabel, timelineStatusBadgeClassForStatus } from "@/lib/status-labels"
import AppShell from "@/components/AppShell.vue"
import CourseCard from "@/components/CourseCard.vue"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"

const { t, lang } = useTranslation()
const activeTab = ref("all")
const searchQuery = ref("")
const resourceFilter = ref<"all" | "video" | "pdf" | "document">("all")
const resourceFilters = ["all", "video", "pdf", "document"] as const
const refreshKey = ref(0)
const allCourses = ref<any[]>([])
const myCourses = ref<any[]>([])
const learningResources = ref<any[]>([])
const loading = ref(false)

const tabs = computed(() => [
  { id: "all", label: t.value.courses.tabs.all },
  { id: "my", label: t.value.courses.tabs.my },
  { id: "resources", label: t.value.courses.tabs.materials },
])

const filteredCourses = computed(() => allCourses.value.filter((course) =>
  course.title.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
  course.description.toLowerCase().includes(searchQuery.value.toLowerCase())
))

const filteredResources = computed(() => learningResources.value.filter((resource) => {
  const matchesSearch = resource.title.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
    resource.course.toLowerCase().includes(searchQuery.value.toLowerCase())
  const matchesFilter = resourceFilter.value === "all" || resource.type === resourceFilter.value
  return matchesSearch && matchesFilter
}))

function certificationDisplayName(value?: string) {
  return String(value || "").replace(/\bPipeline\b/g, "Certification").replace(/管线/g, "认证")
}

function setResourceFilter(filter: (typeof resourceFilters)[number]) {
  resourceFilter.value = filter
}

function courseTypeLabel(type: string) {
  return type === "video" ? t.value.courses.video : type === "pdf" ? t.value.courses.pdf : type === "document" ? t.value.courses.document : type
}

function mapCandidatePipeline(pipeline: any) {
  return {
    id: pipeline.pipeline_cc_ulid || pipeline.pipeline_ulid,
    instanceId: pipeline.pipeline_ulid,
    configId: pipeline.pipeline_cc_ulid,
    title: certificationDisplayName(pipeline.pipeline_name) || pipeline.pipeline_cc_ulid || pipeline.pipeline_ulid || t.value.common.unknownCourse,
    currentStage: pipeline.current_stage_name || pipeline.current_stage_ulid,
    progress: pipeline.progress_available ? Math.round(Number(pipeline.progress)) : undefined,
    progressAvailable: Boolean(pipeline.progress_available),
    statusValue: pipeline.status,
    startedAt: pipeline.started_at,
    completedAt: pipeline.completed_at,
  }
}

async function loadPipelineThumbnailUrl(pipelineId: string) {
  if (!pipelineId) return ""
  try {
    const data = await apiClient(`/api/mall/pipelines/${encodeURIComponent(pipelineId)}/thumbnail-url`)
    return typeof data?.url === "string" ? data.url : ""
  } catch {
    return ""
  }
}

async function refreshMyCourses() {
  const res = await apiClient("/api/pipeline")
  const list = Array.isArray(res?.list) ? res.list : []
  myCourses.value = list.map(mapCandidatePipeline)
}

async function fetchData() {
  loading.value = true
  try {
    if (activeTab.value === "all") {
      const res = await apiClient("/api/mall/pipelines")
      if (res?.pipelines) {
        allCourses.value = await Promise.all(res.pipelines.map(async (p: any) => {
          const stages = p.stages || []
          const unitCount = stages.reduce((total: number, stage: any) => total + (stage.units?.length || 0), 0)
          const finalQualCount = p.final_quals?.length || 0
          const image = await loadPipelineThumbnailUrl(p.pipeline_id)
          const firstStageNames = stages.slice(0, 2).map((stage: any) => stage.name).filter(Boolean).join(" / ")
          return {
            id: p.pipeline_id,
            title: certificationDisplayName(p.name) || t.value.common.unknownCourse,
            description: firstStageNames || `${stages.length} ${t.value.courses.stages} · ${unitCount} ${t.value.courses.units}`,
            provider: p.category_tips || t.value.courses.certificationPath,
            isPurchased: false,
            image,
            students: typeof p.purchase_count === "number" ? p.purchase_count : undefined,
            versionLabel: `${t.value.courses.version} ${p.version || 0}`,
            stats: [
              { label: t.value.courses.stages, value: stages.length },
              { label: t.value.courses.units, value: unitCount },
              { label: t.value.courses.finalQualifications, value: finalQualCount },
            ],
          }
        }))
      }
    } else if (activeTab.value === "my") {
      await refreshMyCourses()
    } else if (activeTab.value === "resources") {
      const res = await apiClient("/api/pipeline/materials")
      if (res?.materials) {
        learningResources.value = res.materials.map((m: any) => {
          let type = "document"
          if (m.type === 1) type = "video"
          else if (m.type === 2) type = "pdf"
          return {
            id: m.id,
            title: m.title || t.value.common.unknown,
            type,
            duration: m.duration_seconds ? `${Math.floor(m.duration_seconds / 60)}:${m.duration_seconds % 60}` : "",
            course: m.course_title || m.course_id || t.value.common.unknownCourse,
            size: m.file_size ? `${Math.round(m.file_size / 1024 / 1024)} MB` : t.value.common.unknown,
            progress: m.progress_value || 0,
          }
        })
      }
    }
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

async function openResource(resource: any) {
  if (!resource?.id) return
  try {
    const res = await apiClient(`/api/pipeline/materials/${resource.id}/url`)
    if (res?.url) window.open(res.url, "_blank", "noopener,noreferrer")
    else toast.error(t.value.common.error)
  } catch {
    // apiClient handles localized errors.
  }
}

function handlePaymentReturn() {
  const url = new URL(window.location.href)
  const paymentStatus = url.searchParams.get("payment_status")
  if (!paymentStatus) return

  const paymentAction = url.searchParams.get("payment_action")
  const purchasedPipelineId = url.searchParams.get("pipeline_id")
  const isUnlock = paymentAction === "unlock"
  const copy = t.value.paymentReturnHandler || {}
  if (paymentStatus === "success") {
    toast.success(isUnlock ? copy.unlockSuccess : copy.purchaseSuccess)
    if (!isUnlock && purchasedPipelineId) {
      void apiClient(`/api/mall/pipelines/${encodeURIComponent(purchasedPipelineId)}/eligibility`)
        .then(() => {
          allCourses.value = allCourses.value.map((course) =>
            course.id === purchasedPipelineId ? { ...course, eligibilityRefreshKey: Date.now() } : course,
          )
        })
        .catch((error) => console.error(error))
    }
  }
  else if (paymentStatus === "cancelled") toast.warning(copy.cancelled)
  else if (paymentStatus === "failed") toast.error(copy.failed)

  localStorage.removeItem("pending_mall_payment")
  refreshKey.value += 1
  url.searchParams.delete("payment_status")
  url.searchParams.delete("payment_action")
  url.searchParams.delete("order_id")
  url.searchParams.delete("pipeline_id")
  window.history.replaceState({}, "", `${url.pathname}${url.search}${url.hash}`)
}

watch([activeTab, refreshKey, lang], () => {
  searchQuery.value = ""
  void fetchData()
})

onMounted(() => {
  handlePaymentReturn()
  void fetchData()
})
</script>

<template>
  <AppShell content-class="p-4">
    <div class="mb-4 overflow-hidden rounded-[16px] bg-white shadow-[0_12px_30px_rgba(15,74,82,0.06)]">
      <div class="bg-gradient-to-r from-[#ecfbf7] via-white to-[#f4fbff] p-4">
        <div class="mb-3 inline-flex items-center gap-2 rounded-full bg-primary/10 px-3 py-1 text-xs font-semibold text-primary">
          <BookOpen class="h-3.5 w-3.5" />
          {{ t.sidebar.courses }}
        </div>
        <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.courses.title }}</h1>
        <p class="mt-2 text-muted-foreground">{{ t.courses.subtitle }}</p>
      </div>
    </div>

    <div class="mb-4 flex flex-col gap-4 rounded-[16px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)] sm:flex-row sm:items-center sm:justify-between">
      <div class="relative flex-1 sm:max-w-md">
        <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <input v-model="searchQuery" class="input pl-10" :placeholder="t.courses.searchPlaceholder" />
      </div>
      <div v-if="activeTab === 'resources'" class="flex flex-wrap gap-2">
        <button v-for="filter in resourceFilters" :key="filter" :class="['btn rounded-lg', resourceFilter === filter ? 'btn-primary shadow-sm shadow-primary/20' : 'btn-outline']" @click="setResourceFilter(filter)">
          <Video v-if="filter === 'video'" class="h-3.5 w-3.5" />
          <FileText v-if="filter === 'pdf'" class="h-3.5 w-3.5" />
          <FileIcon v-if="filter === 'document'" class="h-3.5 w-3.5" />
          {{ filter === 'all' ? t.messagesPage.all : courseTypeLabel(filter) }}
        </button>
      </div>
      <button v-else class="btn btn-outline rounded-lg">
        <SlidersHorizontal class="h-4 w-4" /> {{ t.courses.filterBtn }}
      </button>
    </div>

    <div class="mb-4 rounded-md bg-white px-8 pt-6">
      <div class="flex flex-wrap gap-10 border-b border-[#edf0f2]">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        :class="['relative cursor-pointer px-1 pb-7 text-base font-medium transition-colors duration-200', activeTab === tab.id ? 'text-primary' : 'text-[#111827] hover:text-primary']"
        @click="activeTab = tab.id"
      >
        {{ tab.label }}
        <span v-if="activeTab === tab.id" class="absolute bottom-[-1px] left-0 h-0.5 w-full rounded-full bg-primary" />
      </button>
      </div>
    </div>

    <div v-if="activeTab === 'all'">
      <div v-if="loading && allCourses.length === 0" class="flex items-center justify-center gap-2 rounded-[16px] bg-white py-14 text-muted-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
        <Clock class="h-5 w-5 animate-spin" /> <span>{{ t.common.loading }}</span>
      </div>
      <div v-else-if="filteredCourses.length > 0" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <CourseCard v-for="course in filteredCourses" :key="`${course.id}-${course.eligibilityRefreshKey || 0}`" v-bind="course" />
      </div>
      <div v-else class="flex flex-col items-center justify-center rounded-[16px] bg-white py-16 text-center shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10">
          <Search class="h-8 w-8 text-primary" />
        </div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">暂无数据</h3>
      </div>
    </div>

    <div v-if="activeTab === 'my'" class="space-y-4">
      <div v-if="loading && myCourses.length === 0" class="flex items-center justify-center gap-2 rounded-[16px] bg-white py-14 text-muted-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
        <Clock class="h-5 w-5 animate-spin" /> <span>{{ t.common.loading }}</span>
      </div>
      <div v-for="course in myCourses" :key="course.id" class="group relative overflow-hidden rounded-[16px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all duration-300 hover:ring-primary/25 hover:shadow-md hover:shadow-primary/10">
        <div class="absolute left-0 top-0 h-full w-1 bg-primary" />
        <div class="flex gap-4">
          <div class="flex h-14 w-14 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
            <BookOpen class="h-7 w-7" />
          </div>
          <div class="flex flex-1 flex-col justify-between">
            <div>
              <div class="flex items-start justify-between gap-4">
                <h3 class="text-lg font-semibold text-card-foreground transition-colors group-hover:text-primary">{{ course.title || t.common.unknownCourse }}</h3>
                <span :class="['badge', timelineStatusBadgeClassForStatus('PIPELINE', course.statusValue)]">
                  {{ statusLabel(t, CANDIDATE_PIPELINE_STATUS_LABELS, course.statusValue) }}
                </span>
              </div>
              <div v-if="course.progressAvailable" class="mt-4">
                <div class="mb-2 flex items-center justify-between text-sm">
                  <span class="text-muted-foreground">{{ t.courses.courseProgress }}</span>
                  <span class="font-medium text-foreground">{{ course.progress }}%</span>
                </div>
                <div class="h-2 overflow-hidden rounded-full bg-muted">
                  <div class="h-full rounded-full bg-primary transition-all duration-500" :style="{ width: `${course.progress}%` }" />
                </div>
              </div>
            </div>
            <div class="mt-4 flex flex-wrap gap-4 text-sm text-muted-foreground">
              <span v-if="course.currentStage">{{ t.courses.stage }}: {{ course.currentStage }}</span>
              <span v-if="course.startedAt">{{ course.startedAt }}</span>
              <span v-if="course.completedAt">{{ course.completedAt }}</span>
            </div>
          </div>
          <div class="flex flex-col items-end justify-between">
            <RouterLink :to="`/courses/detail?id=${encodeURIComponent(course.id)}`" class="btn btn-primary rounded-lg shadow-sm shadow-primary/20">
              {{ t.courses.viewDetails }} <ChevronRight class="h-4 w-4" />
            </RouterLink>
          </div>
        </div>
      </div>
      <div v-if="!loading && myCourses.length === 0" class="flex flex-col items-center justify-center rounded-[16px] bg-white py-16 text-center shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10">
          <BookOpen class="h-8 w-8 text-primary" />
        </div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.courses.noCourses }}</h3>
        <p class="mb-4 text-muted-foreground">{{ t.courses.noCoursesDesc }}</p>
        <button class="btn btn-primary rounded-lg shadow-sm shadow-primary/20" @click="activeTab = 'all'">{{ t.courses.browseCoursesBtn }}</button>
      </div>
    </div>

    <div v-if="activeTab === 'resources'" class="space-y-4">
      <div
        v-for="resource in filteredResources"
        :key="resource.id"
        class="group relative cursor-pointer overflow-hidden rounded-[16px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all duration-300 hover:ring-primary/25 hover:shadow-md hover:shadow-primary/10"
        @click="openResource(resource)"
      >
        <div class="flex items-center gap-4">
          <div :class="['flex h-12 w-12 shrink-0 items-center justify-center rounded-lg', resource.type === 'video' ? 'bg-red-100 text-red-600' : resource.type === 'pdf' ? 'bg-orange-100 text-orange-600' : 'bg-blue-100 text-blue-600']">
            <Video v-if="resource.type === 'video'" class="h-6 w-6" />
            <FileText v-else-if="resource.type === 'pdf'" class="h-6 w-6" />
            <FileIcon v-else class="h-6 w-6" />
          </div>
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <h3 class="truncate font-medium text-card-foreground transition-colors group-hover:text-primary">{{ resource.title }}</h3>
              <CheckCircle2 v-if="resource.progress === 100" class="h-4 w-4 shrink-0 text-green-500" />
            </div>
            <div class="mt-1 flex items-center gap-3 text-sm text-muted-foreground">
              <span class="badge">{{ resource.course }}</span>
              <span>{{ courseTypeLabel(resource.type) }}</span>
              <span v-if="resource.type === 'video'" class="flex items-center gap-1"><Clock class="h-3 w-3" /> {{ resource.duration }}</span>
              <span>{{ resource.size }}</span>
            </div>
          </div>
          <div v-if="resource.progress > 0 && resource.progress < 100" class="flex items-center gap-2 text-sm">
            <div class="h-1.5 w-24 overflow-hidden rounded-full bg-muted">
              <div class="h-full rounded-full bg-primary" :style="{ width: `${resource.progress}%` }" />
            </div>
            <span class="w-10 text-right text-muted-foreground">{{ resource.progress }}%</span>
          </div>
          <button class="btn btn-primary rounded-lg shadow-sm shadow-primary/20" @click.stop="openResource(resource)">
            <Play v-if="resource.type === 'video'" class="h-3.5 w-3.5" />
            <Eye v-else class="h-3.5 w-3.5" />
            {{ resource.type === 'video' ? t.courses.watch : t.courses.read }}
          </button>
          <button class="btn btn-ghost px-2" @click.stop="openResource(resource)"><Download class="h-4 w-4" /></button>
          <button class="btn btn-ghost px-2" @click.stop><Bookmark class="h-4 w-4" /></button>
        </div>
      </div>
      <div v-if="filteredResources.length === 0" class="flex flex-col items-center justify-center rounded-[16px] bg-white py-16 text-center shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10">
          <FileText class="h-8 w-8 text-primary" />
        </div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.courses.noResources }}</h3>
        <p class="text-muted-foreground">{{ t.courses.noResourcesDesc }}</p>
      </div>
    </div>
  </AppShell>
</template>
