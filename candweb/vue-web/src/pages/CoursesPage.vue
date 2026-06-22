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
  GraduationCap,
  Loader2,
  Play,
  Search,
  Video,
} from "lucide-vue-next"
import { CANDIDATE_PIPELINE_STATUS_LABELS, statusLabel, timelineStatusBadgeClassForStatus } from "@/lib/status-labels"
import AppShell from "@/components/AppShell.vue"
import CourseCard from "@/components/CourseCard.vue"
import { apiClient } from "@/lib/apiClient"
import { formatBackendDate } from "@/lib/utils"
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
const openingResourceId = ref("")

const emptyCopy = computed(() => lang.value === "zh"
  ? {
      noAvailableTitle: "暂无可购买认证",
      noAvailableDesc: "认证商城开放课程后，会显示在这里。",
      noSearchTitle: "没有匹配的认证",
      noSearchDesc: "换个关键词再试，或清空搜索查看全部认证。",
      clearSearch: "清空搜索",
      noResourceSearchTitle: "没有匹配的学习资料",
      noResourceSearchDesc: "换个关键词或资源类型再试。",
    }
  : {
      noAvailableTitle: "No certifications available",
      noAvailableDesc: "Certifications will appear here once they are available in the catalog.",
      noSearchTitle: "No matching certifications",
      noSearchDesc: "Try another keyword or clear the search to view all certifications.",
      clearSearch: "Clear search",
      noResourceSearchTitle: "No matching materials",
      noResourceSearchDesc: "Try another keyword or resource type.",
    })

const myCertificationCopy = computed(() => lang.value === "zh"
  ? {
      status: "状态",
      details: "查看详情",
      viewDetailsHint: "点击查看认证信息",
    }
  : {
      status: "Status",
      details: "View Details",
      viewDetailsHint: "Click to view certification details",
    })

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

function formatDisplayAmount(amount: number, currency = "USD") {
  const normalizedCurrency = String(currency || "USD").trim().toUpperCase()
  try {
    return new Intl.NumberFormat(undefined, { style: "currency", currency: normalizedCurrency }).format(amount / 100)
  } catch {
    return `${normalizedCurrency} ${(amount / 100).toLocaleString()}`
  }
}

function bundlePriceLabel(bundle: any) {
  const min = Number(bundle?.display_amount_min || 0)
  const max = Number(bundle?.display_amount_max || 0)
  const currency = String(bundle?.display_currency || "USD").trim()
  if (min <= 0 && max <= 0) return ""
  if (max > 0 && max !== min) return `${formatDisplayAmount(min, currency)} - ${formatDisplayAmount(max, currency)}`
  return formatDisplayAmount(min || max, currency)
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
    description: String(pipeline.description || "").trim(),
    currentStage: pipeline.current_stage_name || pipeline.current_stage_ulid,
    progress: pipeline.progress_available ? Math.round(Number(pipeline.progress)) : undefined,
    progressAvailable: Boolean(pipeline.progress_available),
    statusValue: pipeline.status,
    startedAt: formatBackendDate(pipeline.started_at),
    completedAt: formatBackendDate(pipeline.completed_at),
  }
}

async function loadBundleThumbnailUrl(bundleId: string) {
  if (!bundleId) return ""
  try {
    const data = await apiClient(`/api/mall/bundles/${encodeURIComponent(bundleId)}/thumbnail-url`)
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
      const res = await apiClient("/api/mall/bundles")
      if (res?.bundles) {
        allCourses.value = await Promise.all(res.bundles.map(async (b: any) => {
          const stages = b.stages || []
          const unitCount = stages.reduce((total: number, stage: any) => total + (stage.units?.length || 0), 0)
          const finalQualCount = b.final_quals?.length || 0
          const image = await loadBundleThumbnailUrl(b.bundle_id)
          const firstStageNames = stages.slice(0, 2).map((stage: any) => stage.name).filter(Boolean).join(" / ")
          return {
            id: b.bundle_id,
            pipelineId: b.pipeline_id,
            title: certificationDisplayName(b.name) || t.value.common.unknownCourse,
            description: String(b.description || "").trim() || firstStageNames || `${stages.length} ${t.value.courses.stages} · ${unitCount} ${t.value.courses.units}`,
            provider: b.category_tips || t.value.courses.certificationPath,
            isPurchased: false,
            image,
            priceLabel: bundlePriceLabel(b),
            students: typeof b.purchase_count === "number" ? b.purchase_count : undefined,
            versionLabel: `${t.value.courses.version} ${b.version || 0}`,
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
  if (openingResourceId.value) return
  openingResourceId.value = resource.id
  try {
    const res = await apiClient(`/api/pipeline/materials/${resource.id}/url`)
    if (res?.url) window.open(res.url, "_blank", "noopener,noreferrer")
    else toast.error(t.value.common.error)
  } catch {
    // apiClient handles localized errors.
  } finally {
    openingResourceId.value = ""
  }
}

function handlePaymentReturn() {
  const url = new URL(window.location.href)
  const paymentStatus = url.searchParams.get("payment_status")
  if (!paymentStatus) return

  const paymentAction = url.searchParams.get("payment_action")
  const purchasedPipelineId = url.searchParams.get("pipeline_id")
  const purchasedBundleId = url.searchParams.get("bundle_id")
  const targetId = purchasedBundleId || purchasedPipelineId
  const isUnlock = paymentAction === "unlock"
  const copy = t.value.paymentReturnHandler || {}
  if (paymentStatus === "success") {
    toast.success(isUnlock ? copy.unlockSuccess : copy.purchaseSuccess)
    if (!isUnlock && targetId) {
      void apiClient(`/api/mall/pipelines/${encodeURIComponent(purchasedPipelineId || "")}/eligibility`)
        .then(() => {
          allCourses.value = allCourses.value.map((course) =>
            course.id === targetId || course.pipelineId === purchasedPipelineId ? { ...course, eligibilityRefreshKey: Date.now() } : course,
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
  url.searchParams.delete("bundle_id")
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
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <GraduationCap class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ t.courses.title }}</span>
      </header>

      <main class="px-5 py-8 md:px-8 lg:px-10">
        <div class="mb-6">
          <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.courses.title }}</h1>
          <p class="mt-2 text-muted-foreground">{{ t.courses.subtitle }}</p>
        </div>

    <div v-if="activeTab === 'resources'" class="mb-4 flex flex-col gap-4 rounded-[16px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)] sm:flex-row sm:items-center sm:justify-between">
      <div class="relative flex-1 sm:max-w-md">
        <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <input v-model="searchQuery" class="input pl-10" :placeholder="t.courses.searchPlaceholder" />
      </div>
      <div class="flex flex-wrap gap-2">
        <button v-for="filter in resourceFilters" :key="filter" :class="['btn rounded-lg', resourceFilter === filter ? 'btn-primary shadow-sm shadow-primary/20' : 'btn-outline']" @click="setResourceFilter(filter)">
          <Video v-if="filter === 'video'" class="h-3.5 w-3.5" />
          <FileText v-if="filter === 'pdf'" class="h-3.5 w-3.5" />
          <FileIcon v-if="filter === 'document'" class="h-3.5 w-3.5" />
          {{ filter === 'all' ? t.messagesPage.all : courseTypeLabel(filter) }}
        </button>
      </div>
    </div>

    <div class="mb-4 rounded-md bg-white px-4 pt-4 md:px-8 md:pt-6">
      <div class="grid grid-cols-3 border-b border-[#edf0f2] md:flex md:flex-wrap md:gap-10">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        :class="['relative cursor-pointer px-1 pb-4 text-center text-sm font-medium transition-colors duration-200 md:pb-7 md:text-left md:text-base', activeTab === tab.id ? 'text-primary' : 'text-[#111827] hover:text-primary']"
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
      <div v-else-if="filteredCourses.length > 0" class="grid gap-4 sm:grid-cols-2 2xl:grid-cols-3">
        <CourseCard v-for="course in filteredCourses" :key="`${course.id}-${course.eligibilityRefreshKey || 0}`" v-bind="course" />
      </div>
      <div v-else class="flex flex-col items-center justify-center rounded-[16px] bg-white py-16 text-center shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10">
          <Search class="h-8 w-8 text-primary" />
        </div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ searchQuery.trim() ? emptyCopy.noSearchTitle : emptyCopy.noAvailableTitle }}</h3>
        <p class="mx-auto max-w-md text-sm leading-6 text-muted-foreground">{{ searchQuery.trim() ? emptyCopy.noSearchDesc : emptyCopy.noAvailableDesc }}</p>
        <button v-if="searchQuery.trim()" class="btn btn-primary mt-5 rounded-lg shadow-sm shadow-primary/20" @click="searchQuery = ''">
          {{ emptyCopy.clearSearch }}
        </button>
      </div>
    </div>

    <div v-if="activeTab === 'my'">
      <div v-if="loading && myCourses.length === 0" class="flex items-center justify-center gap-2 rounded-[16px] bg-white py-14 text-muted-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
        <Clock class="h-5 w-5 animate-spin" /> <span>{{ t.common.loading }}</span>
      </div>
      <div v-else-if="myCourses.length > 0" class="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
        <div
          v-for="course in myCourses"
          :key="course.id"
          class="group flex min-h-[300px] flex-col rounded-[18px] border-2 border-[#dfe4ea] bg-white p-5 shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all duration-300 hover:-translate-y-0.5 hover:border-primary hover:shadow-[0_18px_42px_rgba(16,30,67,0.16)]"
        >
          <div class="flex-1">
            <h3 class="line-clamp-2 text-xl font-bold leading-tight tracking-tight text-[#111827] transition-colors group-hover:text-primary">
              {{ course.title || t.common.unknownCourse }}
            </h3>

            <div class="mt-8 space-y-5 text-base text-[#4b5563]">
              <div class="flex items-center justify-between gap-4">
                <span>{{ myCertificationCopy.status }}:</span>
                <span :class="['rounded-lg px-3 py-1.5 text-sm font-semibold', timelineStatusBadgeClassForStatus('PIPELINE', course.statusValue)]">
                  {{ statusLabel(t, CANDIDATE_PIPELINE_STATUS_LABELS, course.statusValue) }}
                </span>
              </div>
            </div>

            <div v-if="course.progressAvailable" class="mt-7">
              <div class="mb-2 flex items-center justify-between text-sm">
                <span class="text-muted-foreground">{{ t.courses.courseProgress }}</span>
                <span class="font-semibold text-foreground">{{ course.progress }}%</span>
              </div>
              <div class="h-2 overflow-hidden rounded-full bg-muted">
                <div class="h-full rounded-full bg-primary transition-all duration-500" :style="{ width: `${course.progress}%` }" />
              </div>
            </div>

            <div v-if="course.currentStage || course.startedAt || course.completedAt" class="mt-5 flex flex-wrap gap-x-4 gap-y-2 text-sm text-muted-foreground">
              <span v-if="course.currentStage">{{ t.courses.stage }}: {{ course.currentStage }}</span>
              <span v-if="course.startedAt">{{ course.startedAt }}</span>
              <span v-if="course.completedAt">{{ course.completedAt }}</span>
            </div>
          </div>

          <div class="mt-6">
            <RouterLink
              :to="`/certifications/${encodeURIComponent(course.id)}`"
              class="flex h-10 w-full items-center justify-center gap-2 rounded-xl bg-primary px-3 text-sm font-bold text-white shadow-sm shadow-primary/20 transition-colors hover:bg-primary/90"
              :title="myCertificationCopy.viewDetailsHint"
            >
              <Eye class="h-5 w-5" />
              {{ myCertificationCopy.details }}
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
      <div v-if="loading && learningResources.length === 0" class="flex items-center justify-center gap-2 rounded-[16px] bg-white py-14 text-muted-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
        <Clock class="h-5 w-5 animate-spin" /> <span>{{ t.common.loading }}</span>
      </div>
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
          <button class="btn btn-primary rounded-lg shadow-sm shadow-primary/20" :disabled="Boolean(openingResourceId)" @click.stop="openResource(resource)">
            <Loader2 v-if="openingResourceId === resource.id" class="h-3.5 w-3.5 animate-spin" />
            <Play v-else-if="resource.type === 'video'" class="h-3.5 w-3.5" />
            <Eye v-else class="h-3.5 w-3.5" />
            {{ resource.type === 'video' ? t.courses.watch : t.courses.read }}
          </button>
          <button class="btn btn-ghost px-2" :disabled="Boolean(openingResourceId)" @click.stop="openResource(resource)">
            <Loader2 v-if="openingResourceId === resource.id" class="h-4 w-4 animate-spin" />
            <Download v-else class="h-4 w-4" />
          </button>
          <button class="btn btn-ghost px-2" @click.stop><Bookmark class="h-4 w-4" /></button>
        </div>
      </div>
      <div v-if="!loading && filteredResources.length === 0" class="flex flex-col items-center justify-center rounded-[16px] bg-white py-16 text-center shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10">
          <FileText class="h-8 w-8 text-primary" />
        </div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ searchQuery.trim() || resourceFilter !== 'all' ? emptyCopy.noResourceSearchTitle : t.courses.noResources }}</h3>
        <p class="mx-auto max-w-md text-sm leading-6 text-muted-foreground">{{ searchQuery.trim() || resourceFilter !== 'all' ? emptyCopy.noResourceSearchDesc : t.courses.noResourcesDesc }}</p>
        <button v-if="searchQuery.trim() || resourceFilter !== 'all'" class="btn btn-primary mt-5 rounded-lg shadow-sm shadow-primary/20" @click="searchQuery = ''; resourceFilter = 'all'">
          {{ emptyCopy.clearSearch }}
        </button>
      </div>
        </div>
      </main>
    </div>
  </AppShell>
</template>
