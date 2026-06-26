<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { RouterLink } from "vue-router"
import { BookOpen, Clock, Eye } from "lucide-vue-next"
import { CANDIDATE_PIPELINE_STATUS_LABELS, statusLabel, timelineStatusBadgeClassForStatus } from "@/lib/status-labels"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { formatBackendDate } from "@/lib/utils"
import { useTranslation } from "@/lib/language"

type CandidatePipelineCard = {
  configId: string
  instanceId: string
  title: string
  currentStage: string
  progress?: number
  progressAvailable: boolean
  statusValue: string | number
  startedAt: string
  completedAt: string
}

const { t, lang } = useTranslation()
const myCourses = ref<CandidatePipelineCard[]>([])
const loading = ref(false)

const copy = computed(() => lang.value === "zh"
  ? {
      title: "我的认证",
      subtitle: "查看你已购买或正在进行的认证进度。",
      status: "状态",
      details: "查看详情",
      viewDetailsHint: "点击查看认证详情",
      emptyTitle: "还没有购买认证",
      emptyDesc: "前往商城浏览并选择适合你的认证或会员商品。",
      browseMarketplace: "浏览商城",
      configId: "认证配置 ID",
      instanceId: "认证实例 ID",
    }
  : {
      title: "My Certifications",
      subtitle: "View certifications you have purchased or are currently completing.",
      status: "Status",
      details: "View Details",
      viewDetailsHint: "Click to view certification details",
      emptyTitle: "No certifications purchased yet",
      emptyDesc: "Browse the marketplace and choose the certification or membership product that fits your goals.",
      browseMarketplace: "Browse Marketplace",
      configId: "Certification Config ID",
      instanceId: "Certification Instance ID",
    })

function certificationDisplayName(value?: string) {
  return String(value || "").replace(/\bPipeline\b/g, "Certification").replace(/管线/g, "认证")
}

function mapCandidatePipeline(pipeline: any): CandidatePipelineCard {
  return {
    configId: String(pipeline?.pipeline_cc_ulid || "").trim(),
    instanceId: String(pipeline?.pipeline_ulid || "").trim(),
    title: certificationDisplayName(pipeline?.pipeline_name) || pipeline?.pipeline_cc_ulid || pipeline?.pipeline_ulid || t.value.common.unknownCourse,
    currentStage: String(pipeline?.current_stage_name || pipeline?.current_stage_ulid || "").trim(),
    progress: pipeline?.progress_available ? Math.round(Number(pipeline.progress)) : undefined,
    progressAvailable: Boolean(pipeline?.progress_available),
    statusValue: pipeline?.status,
    startedAt: formatBackendDate(pipeline?.started_at),
    completedAt: formatBackendDate(pipeline?.completed_at),
  }
}

function certificationDetailHref(course: CandidatePipelineCard) {
  const target = course.configId || course.instanceId
  return target ? `/certifications/${encodeURIComponent(target)}` : "/certifications"
}

async function refreshMyCourses() {
  loading.value = true
  try {
    const res = await apiClient("/api/pipeline")
    const list = Array.isArray(res?.list) ? res.list : []
    myCourses.value = list.map(mapCandidatePipeline)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void refreshMyCourses()
})
</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <BookOpen class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ copy.title }}</span>
      </header>

      <main class="px-5 py-8 md:px-8 lg:px-10">
        <div class="mb-6">
          <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ copy.title }}</h1>
          <p class="mt-2 text-muted-foreground">{{ copy.subtitle }}</p>
        </div>

        <div v-if="loading && myCourses.length === 0" class="flex items-center justify-center gap-2 rounded-[16px] bg-white py-14 text-muted-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
          <Clock class="h-5 w-5 animate-spin" /> <span>{{ t.common.loading }}</span>
        </div>

        <div v-else-if="myCourses.length > 0" class="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
          <div
            v-for="course in myCourses"
            :key="`${course.configId}-${course.instanceId}`"
            class="group flex min-h-[320px] flex-col rounded-[18px] border-2 border-[#dfe4ea] bg-white p-5 shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all duration-300 hover:-translate-y-0.5 hover:border-primary hover:shadow-[0_18px_42px_rgba(16,30,67,0.16)]"
          >
            <div class="flex-1">
              <h3 class="line-clamp-2 text-xl font-bold leading-tight tracking-tight text-[#111827] transition-colors group-hover:text-primary">
                {{ course.title || t.common.unknownCourse }}
              </h3>

              <div class="mt-4 space-y-2 text-xs text-slate-500">
                <p v-if="course.configId" class="font-mono">
                  {{ copy.configId }}: {{ course.configId }}
                </p>
                <p v-if="course.instanceId" class="font-mono">
                  {{ copy.instanceId }}: {{ course.instanceId }}
                </p>
              </div>

              <div class="mt-6 space-y-5 text-base text-[#4b5563]">
                <div class="flex items-center justify-between gap-4">
                  <span>{{ copy.status }}:</span>
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
                :to="certificationDetailHref(course)"
                class="flex h-10 w-full items-center justify-center gap-2 rounded-xl bg-primary px-3 text-sm font-bold text-white shadow-sm shadow-primary/20 transition-colors hover:bg-primary/90"
                :title="copy.viewDetailsHint"
              >
                <Eye class="h-5 w-5" />
                {{ copy.details }}
              </RouterLink>
            </div>
          </div>
        </div>

        <div v-else class="flex flex-col items-center justify-center rounded-[16px] bg-white py-16 text-center shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
          <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10">
            <BookOpen class="h-8 w-8 text-primary" />
          </div>
          <h3 class="mb-2 text-lg font-semibold text-foreground">{{ copy.emptyTitle }}</h3>
          <p class="mb-4 text-muted-foreground">{{ copy.emptyDesc }}</p>
          <RouterLink to="/certifications" class="btn btn-primary rounded-lg shadow-sm shadow-primary/20">{{ copy.browseMarketplace }}</RouterLink>
        </div>
      </main>
    </div>
  </AppShell>
</template>
