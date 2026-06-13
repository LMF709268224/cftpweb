<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { RouterLink, useRoute } from "vue-router"
import { toast } from "vue-sonner"
import {
  ArrowLeft,
  ArrowRight,
  Award,
  BookOpen,
  Clock,
  CreditCard,
  ExternalLink,
  Lock,
  Play,
  Sparkles,
} from "lucide-vue-next"
import {
  courseUnitNextStepActionFromStatus,
  stageStatusHintLabel,
  timelineStatusBadgeClassForStatus,
  timelineStatusLabelWithDiagnostics,
} from "@/lib/status-labels"
import AppShell from "@/components/AppShell.vue"
import PurchaseDialog from "@/components/PurchaseDialog.vue"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"

type PipelineDetail = {
  config?: PipelineConfig
  instance?: Record<string, any>
  next_step?: PipelineNextStep
  pipeline_status?: string | number
  current_stage_status?: string | number
  current_stage_name?: string
  current_unit_status?: string | number
}

type PipelineConfig = {
  pipeline_id?: string
  pipeline_guid?: string
  version?: number
  name?: string
  category_tips?: string
  unlock_stripe_price_id?: string
  package_stripe_price_id?: string
  stages?: StageConfig[]
  final_quals?: Qualification[]
}

type StageConfig = {
  stage_id?: string
  name?: string
  sort_order?: number
  runtime_status?: string | number
  units?: UnitConfig[]
}

type UnitConfig = {
  unit_id?: string
  name?: string
  glms_course_id?: string
  runtime_status?: string | number
  allow_retake?: boolean
}

type Qualification = {
  qual_id?: string
  name_hint?: string
}

type PipelineNextStep = {
  action?: string
  stage_name?: string
  course_unit_ulid?: string
  course_unit_cc_ulid?: string
  course_id?: string
  exam_id?: string
  allow_retake?: boolean
  status?: string | number
}

type CourseSummary = {
  course_id?: string
  title?: string
  category_tips?: string
  duration_min?: number
}

const route = useRoute()
const { t } = useTranslation()
const detail = ref<PipelineDetail | null>(null)
const courseSummaries = ref<Record<string, CourseSummary>>({})
const firstCourseThumbnail = ref("")
const loading = ref(false)
const purchaseOpen = ref(false)
const certificateLoading = ref(false)
const scheduleLoading = ref(false)

const pipelineId = computed(() => String(route.query.id || ""))
const pipeline = computed(() => detail.value?.config)
const stages = computed<StageConfig[]>(() => pipeline.value?.stages || [])
const totalUnits = computed(() => stages.value.reduce((total, stage) => total + (stage.units?.length || 0), 0))
const purchased = computed(() => Boolean(detail.value?.instance && Object.keys(detail.value.instance).length > 0))
const instancePipelineId = computed(() =>
  typeof detail.value?.instance?.pipeline_ulid === "string" ? detail.value.instance.pipeline_ulid : "",
)
const paymentConfigured = computed(() => Boolean(pipeline.value?.unlock_stripe_price_id || pipeline.value?.package_stripe_price_id))
const nextStep = computed<PipelineNextStep>(() => detail.value?.next_step || {})
const pipelineStatus = computed(() => detail.value?.pipeline_status)
const currentStageName = computed(() => detail.value?.current_stage_name || "")
const currentStageStatus = computed(() => detail.value?.current_stage_status)
const currentUnitStatus = computed(() => detail.value?.current_unit_status)
const nextUnitStatus = computed(() => nextStep.value?.status || currentUnitStatus.value)
const nextStepAction = computed(() =>
  nextStep.value?.action || courseUnitNextStepActionFromStatus(nextUnitStatus.value, Boolean(nextStep.value?.allow_retake)),
)
const isPipelineTerminal = computed(() => pipelineIsTerminal(pipelineStatus.value))
const firstCourseId = computed(() =>
  stages.value.flatMap((stage) => stage.units || []).find((unit) => unit.glms_course_id)?.glms_course_id || "",
)

const activeStageIndex = computed(() => {
  if (!purchased.value || stages.value.length === 0) return -1
  if (pipelineIsTerminal(pipelineStatus.value)) return stages.value.length
  const nextCourseId = nextStep.value?.course_id
  if (nextCourseId) {
    const byCourse = stages.value.findIndex((stage) =>
      (stage.units || []).some((unit) => unit.glms_course_id === nextCourseId),
    )
    if (byCourse >= 0) return byCourse
  }
  const byName = currentStageName.value
    ? stages.value.findIndex((stage) => stage.name && stage.name === currentStageName.value)
    : -1
  return byName >= 0 ? byName : 0
})

function pipelineIsTerminal(status?: string | number | null) {
  const normalized = String(status ?? "").trim()
  return normalized === "3" || normalized === "4"
}

function stageStatusLabel(status?: string | number | null) {
  return timelineStatusLabelWithDiagnostics(t.value, "STAGE", status)
}

function unitStatusLabel(status?: string | number | null) {
  return timelineStatusLabelWithDiagnostics(t.value, "COURSE_UNIT", status)
}

function stageStateText(index: number) {
  if (!purchased.value) return t.value.courses.positionNotPurchased
  return stageStatusLabel(stages.value[index]?.runtime_status)
}

function stageStateClass(index: number) {
  if (!purchased.value) return "border-slate-200 bg-slate-50 text-slate-600"
  return timelineStatusBadgeClassForStatus("STAGE", stages.value[index]?.runtime_status)
}

function unitStateText(unit: UnitConfig) {
  if (!purchased.value) return t.value.courses.positionNotPurchased
  return unitStatusLabel(unit.runtime_status)
}

function unitStateClass(unit: UnitConfig) {
  if (!purchased.value) return "border-slate-200 bg-slate-50 text-slate-600"
  return timelineStatusBadgeClassForStatus("COURSE_UNIT", unit.runtime_status)
}

function learningHref(courseId?: string) {
  return courseId
    ? `/courses/learn?courseId=${encodeURIComponent(courseId)}&pipelineId=${encodeURIComponent(pipelineId.value)}`
    : "/courses"
}

function nextStepHref() {
  switch (nextStepAction.value) {
    case "continue_learning":
      return nextStep.value?.course_id ? learningHref(nextStep.value.course_id) : learningHref(firstCourseId.value)
    case "signup_exam":
      return `/exams/signup?unitId=${encodeURIComponent(nextStep.value?.course_unit_ulid || "")}&pipelineId=${encodeURIComponent(pipelineId.value)}`
    case "schedule_exam":
    case "view_exam_schedule":
    case "apply_retake":
    case "view_exam_result":
      return "/exams"
    case "view_certificate":
      return instancePipelineId.value ? `/courses/detail?id=${encodeURIComponent(pipelineId.value)}` : "/certificates"
    default:
      return "/courses"
  }
}

function nextStepLabel() {
  switch (nextStepAction.value) {
    case "continue_learning":
      return t.value.courses.openLearning
    case "signup_exam":
      return t.value.learning.goToExams
    case "schedule_exam":
      return t.value.learning.actionScheduleExam
    case "view_exam_schedule":
      return t.value.learning.actionViewExamSchedule
    case "apply_retake":
      return t.value.learning.actionApplyRetake
    case "view_exam_result":
      return t.value.learning.actionViewExamResult
    case "view_certificate":
      return t.value.courses.viewCertificate
    default:
      return t.value.courses.viewDetails
  }
}

function nextStepDescription() {
  switch (nextStepAction.value) {
    case "continue_learning":
      return t.value.learning.nextStepContinueLearningDesc
    case "signup_exam":
    case "schedule_exam":
    case "view_exam_schedule":
    case "apply_retake":
    case "view_exam_result":
      return t.value.learning.nextStepGoToExamsDesc
    case "view_certificate":
      return t.value.learning.nextStepViewCertificateDesc
    default:
      return t.value.learning.nextStepDesc
  }
}

async function loadDetail() {
  if (!pipelineId.value) {
    detail.value = null
    loading.value = false
    return
  }
  loading.value = true
  try {
    detail.value = await apiClient(`/api/mall/pipelines/${pipelineId.value}/runtime`)
  } finally {
    loading.value = false
  }
}

async function loadCourseSummaries() {
  if (!purchased.value) {
    courseSummaries.value = {}
    return
  }
  const courseIds = Array.from(
    new Set(
      stages.value
        .flatMap((stage) => stage.units || [])
        .map((unit) => unit.glms_course_id)
        .filter((id): id is string => Boolean(id)),
    ),
  )
  if (courseIds.length === 0) {
    courseSummaries.value = {}
    return
  }

  const items = await Promise.all(
    courseIds.map(async (courseId) => {
      try {
        const res = await apiClient(`/api/mall/courses/${courseId}`)
        return [courseId, res?.course || res] as const
      } catch {
        return [courseId, null] as const
      }
    }),
  )
  courseSummaries.value = Object.fromEntries(items.filter(([, course]) => Boolean(course))) as Record<string, CourseSummary>
}

async function loadFirstCourseThumbnail() {
  if (!firstCourseId.value) {
    firstCourseThumbnail.value = ""
    return
  }
  try {
    const headers = new Headers()
    const token = localStorage.getItem("access_token")
    if (token) headers.set("Authorization", `Bearer ${token}`)
    const response = await fetch(`/api/mall/courses/${encodeURIComponent(firstCourseId.value)}/thumbnail-url`, {
      credentials: "include",
      headers,
    })
    if (!response.ok) {
      firstCourseThumbnail.value = ""
      return
    }
    const data = await response.json()
    firstCourseThumbnail.value = typeof data?.data?.url === "string" ? data.data.url : ""
  } catch {
    firstCourseThumbnail.value = ""
  }
}

async function openCertificate() {
  if (!instancePipelineId.value) return
  certificateLoading.value = true
  try {
    const res = await apiClient(`/api/pipeline/${instancePipelineId.value}/certificate-url`)
    if (res?.view_url) window.open(res.view_url, "_blank", "noopener,noreferrer")
    else toast.error(t.value.common.error)
  } finally {
    certificateLoading.value = false
  }
}

async function handleScheduleExam() {
  if (!nextStep.value?.exam_id || !instancePipelineId.value) return
  scheduleLoading.value = true
  try {
    const termUrlBase = window.location.origin + "/api/public/webhooks/exams/callback"
    const res = await apiClient(`/api/exams/${encodeURIComponent(nextStep.value.exam_id)}/schedule-url?pipeline_ulid=${encodeURIComponent(instancePipelineId.value)}&course_ulid=${encodeURIComponent(nextStep.value.course_unit_ulid || "")}&url_type=1&term_url_base=${encodeURIComponent(termUrlBase)}`)
    if (res?.url) window.open(res.url, "_blank", "noopener,noreferrer")
    else toast.error(t.value.common.error)
  } finally {
    scheduleLoading.value = false
  }
}

onMounted(loadDetail)
watch(pipelineId, loadDetail)
watch([stages, purchased], () => void loadCourseSummaries(), { deep: true })
watch(firstCourseId, () => void loadFirstCourseThumbnail(), { immediate: true })
</script>

<template>
  <AppShell content-class="p-4">
    <RouterLink to="/courses" class="mb-6 inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground">
      <ArrowLeft class="h-4 w-4" />
      {{ t.courses.backToPipelines }}
    </RouterLink>

    <div v-if="loading" class="text-muted-foreground">{{ t.common.loading }}</div>
    <div v-else-if="!pipeline" class="rounded-[22px] bg-white p-8 text-center text-muted-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)]">{{ t.common.na }}</div>
    <template v-else>
      <div :class="['mb-8 grid gap-8', firstCourseThumbnail && 'lg:grid-cols-[380px_1fr]']">
        <div v-if="firstCourseThumbnail" class="relative flex aspect-video items-center justify-center overflow-hidden rounded-lg bg-muted">
          <img :src="firstCourseThumbnail" :alt="pipeline.name || t.common.unknownCourse" class="h-full w-full object-cover" />
          <div class="absolute inset-0 bg-gradient-to-t from-black/45 via-black/5 to-transparent" />
        </div>

        <div>
          <div class="mb-3 flex flex-wrap gap-2">
            <span class="badge border-0 bg-primary/10 text-primary">{{ t.courses.pipeline }}</span>
            <span v-if="pipeline.category_tips" class="badge">{{ pipeline.category_tips }}</span>
          </div>
          <h1 class="mb-2 text-2xl font-bold text-foreground">{{ pipeline.name || t.common.unknownCourse }}</h1>
          <p class="mb-6 text-muted-foreground">{{ pipeline.category_tips || t.courses.certificationPath }}</p>

          <div class="mb-6 flex flex-wrap gap-6 text-sm text-muted-foreground">
            <div class="flex items-center gap-1.5">
              <BookOpen class="h-4 w-4" />
              <span>{{ stages.length }} {{ t.courses.stages }}</span>
            </div>
            <div class="flex items-center gap-1.5">
              <Clock class="h-4 w-4" />
              <span>{{ totalUnits }} {{ t.courses.units }}</span>
            </div>
            <div class="flex items-center gap-1.5">
              <Award class="h-4 w-4" />
              <span>{{ pipeline.final_quals?.length || 0 }} {{ t.credentialsPage.availableQualifications }}</span>
            </div>
          </div>

          <div class="flex flex-wrap gap-2">
            <button class="btn btn-primary" :disabled="!paymentConfigured || purchased" @click="!purchased && (purchaseOpen = true)">
              <CreditCard class="h-4 w-4" />
              {{ purchased ? t.courses.purchased : t.courses.purchasePipeline }}
            </button>
            <button v-if="purchased && instancePipelineId" class="btn btn-outline" :disabled="certificateLoading" @click="openCertificate">
              <ExternalLink class="h-4 w-4" />
              {{ t.courses.viewCertificate }}
            </button>
          </div>

          <div v-if="purchased && nextStepAction" class="mt-6 rounded-[22px] bg-primary/5 p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
            <div class="mb-2 flex items-center gap-2 text-sm font-semibold text-primary">
              <Sparkles class="h-4 w-4" />
              {{ t.learning.nextStepTitle }}
            </div>
            <div class="text-sm text-muted-foreground">{{ nextStepDescription() }}</div>
            <div class="mt-3 flex flex-wrap items-center gap-3">
              <span v-if="!isPipelineTerminal && currentStageName" class="badge">{{ t.learning.currentStageNameLabel }}: {{ currentStageName }}</span>
              <span
                v-if="!isPipelineTerminal && currentStageStatus !== undefined && currentStageStatus !== ''"
                :class="['badge', timelineStatusBadgeClassForStatus('STAGE', currentStageStatus)]"
              >
                {{ t.learning.currentStageStatusLabel }}: {{ stageStatusLabel(currentStageStatus) }}
              </span>
              <span
                v-if="!isPipelineTerminal && currentUnitStatus !== undefined && currentUnitStatus !== ''"
                :class="['badge', timelineStatusBadgeClassForStatus('COURSE_UNIT', currentUnitStatus)]"
              >
                {{ t.learning.unitStatusLabel }}: {{ unitStatusLabel(currentUnitStatus) }}
              </span>
              <span v-if="nextStep.stage_name" class="badge">{{ nextStep.stage_name }}</span>
              <span v-if="nextStep.course_id && courseSummaries[nextStep.course_id]?.title" class="badge">
                {{ courseSummaries[nextStep.course_id]?.title }}
              </span>
              <button v-if="nextStepAction === 'schedule_exam'" class="btn btn-primary py-1.5 text-xs" :disabled="scheduleLoading" @click="handleScheduleExam">
                {{ nextStepLabel() }}
                <ArrowRight class="ml-1 h-4 w-4" />
              </button>
              <RouterLink v-else :to="nextStepHref()" class="btn btn-primary py-1.5 text-xs">
                {{ nextStepLabel() }}
                <ArrowRight class="ml-1 h-4 w-4" />
              </RouterLink>
            </div>
          </div>

          <div v-if="purchased" class="mt-4 rounded-[22px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
            <div class="flex items-center justify-between gap-3">
              <div>
                <div class="flex items-center gap-2 text-sm font-semibold text-foreground">
                  <Sparkles class="h-4 w-4 text-primary" />
                  {{ t.learning.pipelineTimelineTitle }}
                </div>
                <p class="text-xs text-muted-foreground">{{ stageStatusHintLabel(t, currentStageStatus) }}</p>
              </div>
              <RouterLink :to="`/courses/timeline?id=${encodeURIComponent(pipelineId)}`" class="btn btn-outline py-1.5 text-xs">
                {{ t.learning.viewTimeline }}
              </RouterLink>
            </div>
          </div>
        </div>
      </div>

      <section class="space-y-4">
        <div class="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h2 class="text-lg font-semibold text-foreground">{{ t.courses.stageListTitle }}</h2>
            <p class="mt-1 text-sm text-muted-foreground">{{ t.courses.stageListDesc }}</p>
          </div>
          <span class="badge">{{ stages.length }} {{ t.courses.stages }} / {{ totalUnits }} {{ t.courses.units }}</span>
        </div>

        <div v-if="stages.length === 0" class="rounded-[22px] bg-white p-8 text-center text-muted-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)]">{{ t.common.na }}</div>
        <div
          v-for="(stage, stageIndex) in stages"
          v-else
          :key="stage.stage_id || stageIndex"
          :class="[
            'overflow-hidden rounded-[22px] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)]',
            stageIndex === activeStageIndex ? 'border-primary/30 ring-1 ring-primary/15' : '',
          ]"
        >
          <div class="flex flex-col gap-4 px-5 py-4 md:flex-row md:items-center md:justify-between">
            <div class="flex items-center gap-3">
              <div
                :class="[
                  'flex h-10 w-10 items-center justify-center rounded-xl text-sm font-semibold',
                  stageIndex === activeStageIndex ? 'bg-primary text-primary-foreground' : 'bg-primary/10 text-primary',
                ]"
              >
                {{ stageIndex + 1 }}
              </div>
              <div>
                <h3 class="font-semibold">{{ stage.name || `${t.courses.stage} ${stageIndex + 1}` }}</h3>
                <p class="text-sm text-muted-foreground">{{ stage.units?.length || 0 }} {{ t.courses.units }}</p>
              </div>
            </div>
            <div class="flex flex-wrap gap-2">
              <span :class="['badge', stageStateClass(stageIndex)]">
                {{ t.learning.currentStageStatusLabel }}: {{ stageStateText(stageIndex) }}
              </span>
              <span class="badge">{{ t.learning.stageOrderLabel }} {{ stage.sort_order || stageIndex + 1 }}</span>
            </div>
          </div>

          <div class="space-y-2">
            <component
              :is="purchased && unit.glms_course_id && (stageIndex <= activeStageIndex || activeStageIndex >= stages.length) ? RouterLink : 'div'"
              v-for="(unit, unitIndex) in stage.units || []"
              :key="unit.unit_id || unit.glms_course_id || `${stageIndex}-${unitIndex}`"
              :to="learningHref(unit.glms_course_id)"
              :class="[
                'flex items-center justify-between gap-4 px-5 py-3',
                purchased && unit.glms_course_id && (stageIndex <= activeStageIndex || activeStageIndex >= stages.length)
                  ? 'transition-colors hover:bg-primary/10'
                  : 'opacity-75',
              ]"
            >
              <div class="flex items-center gap-3">
                <div
                  v-if="purchased && unit.glms_course_id && (stageIndex <= activeStageIndex || activeStageIndex >= stages.length)"
                  :class="[
                    'flex h-8 w-8 items-center justify-center rounded-full',
                    purchased && stageIndex === activeStageIndex && (!nextStep.course_id || unit.glms_course_id === nextStep.course_id)
                      ? 'bg-primary text-primary-foreground'
                      : 'bg-primary/10 text-primary',
                  ]"
                >
                  <Play class="h-3.5 w-3.5 fill-current" />
                </div>
                <div v-else class="flex h-8 w-8 items-center justify-center rounded-full bg-muted text-muted-foreground">
                  <Lock class="h-3.5 w-3.5" />
                </div>
                <div>
                  <div class="font-medium text-foreground">
                    {{ (unit.glms_course_id && courseSummaries[unit.glms_course_id]?.title) || unit.name || unit.glms_course_id || t.common.unknownCourse }}
                  </div>
                  <div v-if="unit.glms_course_id && (courseSummaries[unit.glms_course_id]?.category_tips || courseSummaries[unit.glms_course_id]?.duration_min)" class="text-xs text-muted-foreground">
                    {{
                      [
                        courseSummaries[unit.glms_course_id]?.category_tips,
                        courseSummaries[unit.glms_course_id]?.duration_min ? `${courseSummaries[unit.glms_course_id]?.duration_min} min` : "",
                      ]
                        .filter(Boolean)
                        .join(" · ")
                    }}
                  </div>
                </div>
              </div>
              <div class="flex flex-wrap items-center justify-end gap-2">
                <span :class="['badge', unitStateClass(unit)]">{{ t.learning.unitStatusLabel }}: {{ unitStateText(unit) }}</span>
                <span v-if="unit.glms_course_id && courseSummaries[unit.glms_course_id]?.category_tips" class="badge">
                  {{ courseSummaries[unit.glms_course_id]?.category_tips }}
                </span>
                <span v-if="unit.allow_retake" class="badge">{{ t.courses.reviewCourse }}</span>
                <span
                  v-if="purchased && stageIndex === activeStageIndex && (!nextStep.course_id || unit.glms_course_id === nextStep.course_id)"
                  class="badge border-primary bg-primary text-primary-foreground"
                >
                  {{ t.courses.currentLearningBadge }}
                </span>
                <span
                  v-if="purchased && unit.glms_course_id && (stageIndex <= activeStageIndex || activeStageIndex >= stages.length)"
                  class="badge border-primary bg-primary text-primary-foreground"
                >
                  {{ t.courses.openLearning }}
                </span>
              </div>
            </component>
          </div>
        </div>
      </section>

      <PurchaseDialog
        v-model:open="purchaseOpen"
        :course-name="pipeline.name || t.common.unknownCourse"
        :pipeline-id="pipeline.pipeline_id || pipelineId"
      />
    </template>
  </AppShell>
</template>
