<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { RouterLink } from "vue-router"
import { toast } from "vue-sonner"
import { BookOpen, Eye } from "lucide-vue-next"
import { CANDIDATE_PIPELINE_STATUS_LABELS, statusLabel, timelineStatusBadgeClassForStatus } from "@/lib/status-labels"
import AppShell from "@/components/AppShell.vue"
import PageFeedback from "@/components/PageFeedback.vue"
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
const loadError = ref(false)

const copy = computed(() => t.value.myCertificationsPage)

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
  const showPageError = myCourses.value.length === 0
  loading.value = true
  if (showPageError) loadError.value = false
  try {
    const res = await apiClient("/api/pipeline")
    if (!Array.isArray(res?.list)) throw new Error("PIPELINE_LIST_INVALID_RESPONSE")
    myCourses.value = res.list.map(mapCandidatePipeline)
  } catch (error) {
    console.error(error)
    if (showPageError) loadError.value = true
  } finally {
    loading.value = false
  }
}

function handlePaymentReturn() {
  const url = new URL(window.location.href)
  const paymentStatus = url.searchParams.get("payment_status")
  if (!paymentStatus) return

  const paymentAction = url.searchParams.get("payment_action")
  const isUnlock = paymentAction === "unlock"
  const copy = t.value.paymentReturnHandler || {}

  if (paymentStatus === "success") {
    toast.success(isUnlock ? copy.unlockSuccess : copy.purchaseSuccess)
  } else if (paymentStatus === "cancelled") {
    toast.warning(copy.cancelled)
  } else if (paymentStatus === "failed") {
    toast.error(copy.failed)
  }

  localStorage.removeItem("pending_mall_payment")
  url.searchParams.delete("payment_status")
  url.searchParams.delete("payment_action")
  url.searchParams.delete("order_id")
  url.searchParams.delete("pipeline_id")
  url.searchParams.delete("bundle_id")
  window.history.replaceState({}, "", `${url.pathname}${url.search}${url.hash}`)
}

onMounted(() => {
  handlePaymentReturn()
  void refreshMyCourses()
})

watch(lang, () => {
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
        <div class="my-certifications-intro mb-6">
          <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ copy.title }}</h1>
          <p class="mt-2 text-muted-foreground">{{ copy.subtitle }}</p>
        </div>

        <PageFeedback v-if="loading && myCourses.length === 0" kind="loading" :loading-label="t.common.loading" />

        <PageFeedback
          v-else-if="loadError"
          kind="error"
          :title="copy.loadFailed"
          :description="copy.loadFailedDesc"
          :action-label="copy.retry"
          @action="refreshMyCourses"
        />

        <div v-else-if="myCourses.length > 0" class="my-certifications-grid grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
          <div
            v-for="course in myCourses"
            :key="`${course.configId}-${course.instanceId}`"
            class="my-certification-card group flex min-h-[320px] flex-col rounded-[18px] border-2 border-[#dfe4ea] bg-white p-5 shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all duration-300 hover:-translate-y-0.5 hover:border-primary hover:shadow-[0_18px_42px_rgba(16,30,67,0.16)]"
          >
            <div class="flex-1">
              <h3 class="line-clamp-2 text-xl font-bold leading-tight tracking-tight text-foreground transition-colors group-hover:text-primary">
                {{ course.title || t.common.unknownCourse }}
              </h3>

              <div class="my-certification-status mt-6 space-y-5 text-base text-muted-foreground">
                <div class="flex items-center justify-between gap-4">
                  <span>{{ copy.status }}:</span>
                  <span :class="['rounded-lg px-3 py-1.5 text-sm font-semibold', timelineStatusBadgeClassForStatus('PIPELINE', course.statusValue)]">
                    {{ statusLabel(t, CANDIDATE_PIPELINE_STATUS_LABELS, course.statusValue) }}
                  </span>
                </div>
              </div>

              <div v-if="course.progressAvailable" class="my-certification-progress mt-7">
                <div class="mb-2 flex items-center justify-between text-sm">
                  <span class="text-muted-foreground">{{ t.courses.courseProgress }}</span>
                  <span class="font-semibold text-foreground">{{ course.progress }}%</span>
                </div>
                <div class="h-2 overflow-hidden rounded-full bg-muted">
                  <div class="h-full rounded-full bg-primary transition-all duration-500" :style="{ width: `${course.progress}%` }" />
                </div>
              </div>

              <div v-if="course.currentStage || course.startedAt || course.completedAt" class="my-certification-meta mt-5 flex flex-wrap gap-x-4 gap-y-2 text-sm text-muted-foreground">
                <span v-if="course.currentStage">{{ t.courses.stage }}: {{ course.currentStage }}</span>
                <span v-if="course.startedAt">{{ copy.startedAt }}: {{ course.startedAt }}</span>
                <span v-if="course.completedAt">{{ copy.completedAt }}: {{ course.completedAt }}</span>
              </div>
            </div>

            <div class="my-certification-action mt-6">
              <RouterLink
                :to="certificationDetailHref(course)"
                data-testid="owned-certification-details"
                :data-pipeline-config-id="course.configId"
                :data-pipeline-instance-id="course.instanceId"
                class="flex h-10 w-full items-center justify-center gap-2 rounded-xl bg-primary px-3 text-sm font-bold text-white shadow-sm shadow-primary/20 transition-colors hover:bg-primary/90"
                :title="copy.viewDetailsHint"
              >
                <Eye class="h-5 w-5" />
                {{ copy.details }}
              </RouterLink>
            </div>
          </div>
        </div>

        <PageFeedback v-else kind="empty" :title="copy.emptyTitle" :description="copy.emptyDesc">
          <template #icon><BookOpen class="h-8 w-8" /></template>
          <template #action>
            <RouterLink to="/certifications" class="btn btn-primary mt-5 rounded-lg shadow-sm shadow-primary/20">{{ copy.browseMarketplace }}</RouterLink>
          </template>
        </PageFeedback>
      </main>
    </div>
  </AppShell>
</template>

<style scoped>
@media (max-width: 767px) {
  .my-certifications-intro {
    margin-bottom: 16px;
  }

  .my-certifications-grid {
    gap: 12px;
  }

  .my-certification-card {
    min-height: 0;
    padding: 16px;
  }

  .my-certification-status,
  .my-certification-progress,
  .my-certification-action {
    margin-top: 16px;
  }

  .my-certification-meta {
    flex-direction: column;
    gap: 4px;
    margin-top: 12px;
  }
}
</style>
