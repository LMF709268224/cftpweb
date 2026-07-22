<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { RouterLink, useRoute, useRouter } from "vue-router"
import { toast } from "vue-sonner"
import {
  ArrowLeft,
  ArrowRight,
  Award,
  BookOpen,
  Clock,
  CreditCard,
  ExternalLink,
  Loader2,
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
import LoadingState from "@/components/LoadingState.vue"
import PaymentSessionDialog from "@/components/PaymentSessionDialog.vue"
import PurchaseDialog from "@/components/PurchaseDialog.vue"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"
import { usePolling } from "@/lib/polling"
import { formatBackendDateOnly } from "@/lib/utils"

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
  description?: string
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
  qualId?: string
  id?: string
  name_hint?: string
  nameHint?: string
  name?: string
}

type CredentialFileConstraint = {
  name?: string
  type?: string | number
  is_required?: boolean
  isRequired?: boolean
}

type CredentialDefinition = {
  cred_def_ulid?: string
  cred_def_id?: string
  name?: string
  description?: string
  file_constraints?: CredentialFileConstraint[]
  fileConstraints?: CredentialFileConstraint[]
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
const router = useRouter()
const { t } = useTranslation()
const detail = ref<PipelineDetail | null>(null)
const courseSummaries = ref<Record<string, CourseSummary>>({})
const credentialDefinitions = ref<Record<string, CredentialDefinition>>({})
const firstCourseThumbnail = ref("")
const loading = ref(false)
const courseSummariesLoading = ref(false)
const credentialDefinitionsLoading = ref(false)
const purchaseOpen = ref(false)
const certificateLoading = ref(false)
const scheduleLoading = ref(false)
const finalQualificationLoading = ref(false)
const resolvedBundleId = ref("")
const finalQualificationPaymentOpen = ref(false)
const finalQualificationPaymentSession = ref<{
  paymentKey?: string
  orderId?: string
  bizType: string
  bizRefUlid: string
  source: string
  returnPath: string
  extraReturnParams?: Record<string, string>
} | null>(null)

function formatCourseDuration(minutes?: number) {
  const normalized = Number(minutes || 0)
  if (!Number.isFinite(normalized) || normalized <= 0) return ""
  return `${Math.floor(normalized)} ${t.value.common.minuteUnit}`
}

const pipelineId = computed(() => String(route.params.pipelineId || route.query.id || ""))
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
const finalQualifications = computed(() => {
  const quals = pipeline.value?.final_quals || []
  return Array.isArray(quals)
    ? quals
        .map((qual) => ({
          qualId: firstString(qual?.qual_id, qual?.qualId, qual?.id),
          name: firstString(qual?.name_hint, qual?.nameHint, qual?.name),
        }))
        .map((qual) => {
          const definition = credentialDefinitions.value[qual.qualId]
          return {
            ...qual,
            name: firstString(definition?.name, qual.name),
            description: firstString(definition?.description),
            constraints: fileConstraintsOfDefinition(definition),
          }
        })
        .filter((qual) => qual.qualId)
    : []
})
const finalQualificationIds = computed(() => {
  const quals = pipeline.value?.final_quals || []
  return Array.isArray(quals)
    ? quals
        .map((qual) => firstString(qual?.qual_id, qual?.qualId, qual?.id))
        .filter((id): id is string => Boolean(id))
    : []
})
const finalQualificationIdsKey = computed(() => finalQualificationIds.value.join(","))
const pipelineHasCertificate = computed(() => finalQualificationIds.value.length > 0)
const pipelineWaitsFinalEligibility = computed(() => {
  const raw = String(pipelineStatus.value ?? "").trim()
  return raw === "2" || raw.toUpperCase().includes("WAIT_FINAL_ELIG")
})
const pipelineCancelled = computed(() => {
  const raw = String(pipelineStatus.value ?? "").trim().toUpperCase().replace(/^PIPELINE_STATUS_/, "")
  return raw === "5" || raw === "CANCELLED"
})
const finalQualificationRequired = computed(() =>
  purchased.value &&
  finalQualificationIds.value.length > 0 &&
  !pipelineCancelled.value &&
  (pipelineWaitsFinalEligibility.value || nextStepAction.value === "final_qualification"),
)
const pipelineIssuingCertificate = computed(() => {
  const raw = String(pipelineStatus.value ?? "").trim().toUpperCase()
  return raw === "4" || raw.includes("ISSUING_CERT") || nextStepAction.value === "issuing_certificate"
})
const pipelineCompleted = computed(() =>
  pipelineIsTerminal(pipelineStatus.value) || nextStepAction.value === "completed",
)
const certificateAvailable = computed(() =>
  purchased.value &&
  Boolean(instancePipelineId.value) &&
  (nextStepAction.value === "view_certificate" || pipelineCompleted.value) &&
  !pipelineIssuingCertificate.value &&
  !pipelineCancelled.value &&
  pipelineHasCertificate.value,
)
const certificateDescription = computed(() => {
  if (certificateAvailable.value) {
    const name = pipeline.value?.name || t.value.common.unknownCourse
    return t.value.learning.certificateCongratulationsDesc.replace(/\{\{name\}\}/g, name)
  }
  if (pipelineCancelled.value) return t.value.learning.statusCancelled
  if (pipelineIssuingCertificate.value) return t.value.learning.certificateIssuingDesc
  if (finalQualificationRequired.value) return t.value.learning.finalQualificationDesc
  return t.value.learning.certificateUnavailableDesc
})
const certificateIssuedDate = computed(() =>
  formatBackendDateOnly(
    firstString(
      detail.value?.instance?.completed_at,
      detail.value?.instance?.completedAt,
      detail.value?.instance?.issued_at,
      detail.value?.instance?.issuedAt,
      detail.value?.instance?.updated_at,
      detail.value?.instance?.updatedAt,
      detail.value?.instance?.created_at,
      detail.value?.instance?.createdAt,
    ),
  ) || "-",
)
const firstCourseId = computed(() =>
  stages.value.flatMap((stage) => visibleStageUnits(stage)).find((unit) => unit.glms_course_id)?.glms_course_id || "",
)
const stageListLoading = computed(() => courseSummariesLoading.value)

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
  const normalized = String(status ?? "").trim().toUpperCase()
  return normalized === "3" || normalized.includes("COMPLETED")
}

function firstString(...values: unknown[]) {
  for (const value of values) {
    const normalized = String(value || "").trim()
    if (normalized) return normalized
  }
  return ""
}

function fileConstraintsOfDefinition(definition?: CredentialDefinition) {
  const constraints = definition?.file_constraints || definition?.fileConstraints || []
  return Array.isArray(constraints) ? constraints : []
}

function constraintName(constraint: CredentialFileConstraint) {
  return firstString(constraint?.name)
}

function constraintRequired(constraint: CredentialFileConstraint) {
  return Boolean(constraint?.is_required ?? constraint?.isRequired)
}

function credentialDefinitionId(definition?: CredentialDefinition) {
  return firstString(definition?.cred_def_ulid, definition?.cred_def_id)
}

function hasRuntimeStatus(status?: string | number | null) {
  const normalized = String(status ?? "").trim()
  return normalized !== "" && normalized !== "0"
}

function canShowUnit(unit: UnitConfig) {
  return purchased.value && hasRuntimeStatus(unit.runtime_status)
}

function visibleStageUnits(stage: StageConfig) {
  return (stage.units || []).filter(canShowUnit)
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
    ? `/certifications/${encodeURIComponent(pipelineId.value)}/learn/${encodeURIComponent(courseId)}`
    : "/certifications"
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
      return instancePipelineId.value ? `/certifications/${encodeURIComponent(pipelineId.value)}` : "/certificates"
    case "completed":
      return "/certifications"
    default:
      return "/certifications"
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
    case "completed":
      return t.value.learning.completedTag
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
    case "completed":
      return t.value.learning.nextStepDesc
    default:
      return t.value.learning.nextStepDesc
  }
}

async function loadDetail(showLoading = true, suppressErrorToast = false) {
  if (!pipelineId.value) {
    detail.value = null
    if (showLoading) loading.value = false
    return
  }
  if (showLoading) loading.value = true
  try {
    detail.value = await apiClient(`/api/mall/pipelines/${pipelineId.value}/runtime`, { suppressErrorToast })
  } finally {
    if (showLoading) loading.value = false
  }
}

async function loadCourseSummaries() {
  if (!purchased.value) {
    courseSummariesLoading.value = false
    courseSummaries.value = {}
    return
  }
  const courseIds = Array.from(
    new Set(
      stages.value
        .flatMap((stage) => visibleStageUnits(stage))
        .map((unit) => unit.glms_course_id)
        .filter((id): id is string => Boolean(id)),
    ),
  )
  if (courseIds.length === 0) {
    courseSummariesLoading.value = false
    courseSummaries.value = {}
    return
  }

  courseSummariesLoading.value = true
  try {
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
  } finally {
    courseSummariesLoading.value = false
  }
}

async function loadCredentialDefinitions() {
  const ids = finalQualificationIds.value
  if (ids.length === 0) {
    credentialDefinitions.value = {}
    credentialDefinitionsLoading.value = false
    return
  }
  credentialDefinitionsLoading.value = true
  try {
    const res = await apiClient(`/api/credentials/definitions?qual_ulids=${encodeURIComponent(ids.join(","))}`, {
      suppressErrorToast: true,
    })
    const definitions = Array.isArray(res?.definitions) ? res.definitions : []
    const entries: Array<[string, CredentialDefinition]> = definitions
      .map((definition: CredentialDefinition) => [credentialDefinitionId(definition), definition])
      .filter(([id]: [string, CredentialDefinition]) => Boolean(id))
    credentialDefinitions.value = Object.fromEntries(entries)
  } catch (err) {
    console.warn("Failed to load final qualification definitions", err)
    credentialDefinitions.value = {}
  } finally {
    credentialDefinitionsLoading.value = false
  }
}

async function loadFirstCourseThumbnail() {
  if (!firstCourseId.value) {
    firstCourseThumbnail.value = ""
    return
  }
  try {
    const data = await apiClient(`/api/mall/courses/${encodeURIComponent(firstCourseId.value)}/thumbnail-url`, {
      suppressErrorToast: true,
    })
    firstCourseThumbnail.value = typeof data?.url === "string" ? data.url : ""
  } catch {
    firstCourseThumbnail.value = ""
  }
}

async function openCertificate() {
  if (!instancePipelineId.value || !certificateAvailable.value) {
    toast.error(t.value.learning.certificateIssuingDesc)
    return
  }
  certificateLoading.value = true
  try {
    const res = await apiClient(`/api/pipeline/${instancePipelineId.value}/certificate-url`)
    if (res?.view_url) window.open(res.view_url, "_blank", "noopener,noreferrer")
    else toast.error(t.value.common.error)
  } catch (err) {
    console.error(err)
    toast.error(t.value.learning.certificateUnavailableDesc)
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

function normalizedStatus(status: unknown) {
  return String(status || "").trim().toUpperCase()
}

function isUploadReadyStatus(status: unknown) {
  return normalizedStatus(status).includes("UPLOAD_READY")
}

function isCredentialApplicationPaymentStatus(status: unknown) {
  return normalizedStatus(status).includes("WAIT_REVIEW_FEE_PAYMENT")
}

function isCredentialApplicationUnderReviewStatus(status: unknown) {
  return normalizedStatus(status).includes("UNDER_REVIEW")
}

function isCredentialApplicationResolvedStatus(status: unknown) {
  const value = normalizedStatus(status)
  return value.includes("RESOLVED") || value.includes("APPROVED") || value.includes("COMPLETED")
}

function finalQualificationUploadPath(qualIds = finalQualificationIds.value) {
  const params = new URLSearchParams()
  if (qualIds.length > 0) params.set("qual_ulids", qualIds.join(","))
  return `/credentials${params.toString() ? `?${params.toString()}` : ""}`
}

async function resolveBundleIdForPipeline() {
  if (resolvedBundleId.value) return resolvedBundleId.value
  if (!pipelineId.value) return ""
  const res = await apiClient("/api/mall/bundles?page_size=100")
  const found = (res?.bundles || []).find((bundle: any) => firstString(bundle?.pipeline_id, bundle?.pipeline_cc_ulid) === pipelineId.value)
  const bundleId = firstString(found?.bundle_id, found?.bundle_ulid)
  resolvedBundleId.value = bundleId
  return bundleId
}

async function missingFinalQualificationIds() {
  const ids = finalQualificationIds.value
  if (ids.length === 0) return []
  const res = await apiClient(`/api/credentials/qualifications?qual_ulids=${encodeURIComponent(ids.join(","))}`)
  const qualifications = Array.isArray(res?.qualifications) ? res.qualifications : []
  if (qualifications.length === 0) return ids
  const eligible = new Set(
    qualifications
      .filter((item: any) => Boolean(item?.eligible))
      .map((item: any) => firstString(item?.qual_id, item?.cred_def_ulid)),
  )
  return ids.filter((id) => !eligible.has(id))
}

async function handleFinalQualificationApplication() {
  if (finalQualificationLoading.value) return
  if (!pipelineId.value || finalQualificationIds.value.length === 0) {
    toast.error(t.value.common.error)
    return
  }
  finalQualificationLoading.value = true
  try {
    const missingQualIds = await missingFinalQualificationIds()
    if (missingQualIds.length === 0) {
      toast.success(t.value.learning.finalQualificationApproved)
      await loadDetail()
      return
    }
    const bundleId = await resolveBundleIdForPipeline()
    if (!bundleId) {
      toast.error(t.value.learning.finalQualificationBundleMissing)
      return
    }
    const order = await apiClient("/api/credentials/application-orders", {
      method: "POST",
      body: JSON.stringify({
        pipeline_cc_ulid: pipelineId.value,
        bundle_ulid: bundleId,
        qual_ulids: missingQualIds,
      }),
    })
    const orderId = firstString(order?.application_order_ulid, order?.application_order_id)
    const orderStatus = firstString(order?.order_status, order?.status)
    if (isUploadReadyStatus(orderStatus)) {
      toast.info(t.value.learning.finalQualificationUploadReady)
      window.setTimeout(() => router.push(finalQualificationUploadPath(missingQualIds)), 300)
      return
    }
    if (isCredentialApplicationPaymentStatus(orderStatus) || order?.payment_key) {
      finalQualificationPaymentSession.value = {
        paymentKey: order?.payment_key,
        orderId,
        bizType: "CREDENTIAL_APPLICATION",
        bizRefUlid: orderId,
        source: "credential_application",
        returnPath: "/credentials",
        extraReturnParams: { qual_ulids: missingQualIds.join(",") },
      }
      finalQualificationPaymentOpen.value = true
      return
    }
    if (isCredentialApplicationUnderReviewStatus(orderStatus)) {
      toast.info(t.value.learning.finalQualificationUnderReview)
      return
    }
    if (isCredentialApplicationResolvedStatus(orderStatus)) {
      toast.success(t.value.learning.finalQualificationApproved)
      await loadDetail()
      return
    }
    toast.info(t.value.learning.finalQualificationOrderCreated)
  } catch (error) {
    console.error(error)
  } finally {
    finalQualificationLoading.value = false
  }
}

const detailPolling = usePolling(
  () => loadDetail(false, true),
  { shouldPoll: () => Boolean(pipelineId.value && purchased.value && !isPipelineTerminal.value) },
)

onMounted(() => {
  void loadDetail()
  detailPolling.start()
})
watch(pipelineId, () => void loadDetail())
watch([stages, purchased], () => void loadCourseSummaries(), { deep: true })
watch(finalQualificationIdsKey, () => void loadCredentialDefinitions(), { immediate: true })
watch(firstCourseId, () => void loadFirstCourseThumbnail(), { immediate: true })
</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <BookOpen class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ pipeline?.name || t.common.unknownCourse }}</span>
      </header>

      <main class="px-5 py-8 md:px-8 lg:px-10">
        <RouterLink :to="purchased ? '/my-certifications' : '/certifications'" class="mb-6 inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground">
          <ArrowLeft class="h-4 w-4" />
          {{ t.courses.backToPipelines }}
        </RouterLink>

    <LoadingState v-if="loading" :label="t.common.loading" variant="page" :rows="4" />
    <div v-else-if="!pipeline" class="rounded-md bg-white p-8 text-center text-muted-foreground">
      <div class="mx-auto max-w-md space-y-4">
        <div>
          <h2 class="text-lg font-semibold text-foreground">{{ t.learning.courseUnavailableTitle }}</h2>
          <p class="mt-2 text-sm">{{ t.learning.courseUnavailableDesc }}</p>
        </div>
        <RouterLink :to="purchased ? '/my-certifications' : '/certifications'" class="btn btn-primary mx-auto w-fit rounded-lg">
          {{ t.courses.backToPipelines }}
        </RouterLink>
      </div>
    </div>
    <template v-else>
      <div :class="['mb-4 rounded-md bg-white p-6', firstCourseThumbnail && 'grid gap-6 lg:grid-cols-[340px_1fr]']">
        <div v-if="firstCourseThumbnail" class="relative flex aspect-video items-center justify-center overflow-hidden rounded-md bg-muted">
          <img :src="firstCourseThumbnail" :alt="pipeline.name || t.common.unknownCourse" class="h-full w-full object-cover" />
          <div class="absolute inset-0 bg-gradient-to-t from-black/45 via-black/5 to-transparent" />
        </div>

        <div>
          <h1 class="mb-2 text-2xl font-bold text-foreground">{{ pipeline.name || t.common.unknownCourse }}</h1>
          <p v-if="pipeline.description" class="mb-4 max-w-3xl text-sm leading-6 text-muted-foreground">{{ pipeline.description }}</p>

          <div class="mb-5 flex flex-wrap gap-6 text-sm text-muted-foreground">
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

          <div v-if="false && purchased" class="mt-4 rounded-md bg-slate-50 p-4">
            <div class="flex items-center justify-between gap-3">
              <div>
                <div class="flex items-center gap-2 text-sm font-semibold text-foreground">
                  <Sparkles class="h-4 w-4 text-primary" />
                  {{ t.learning.pipelineTimelineTitle }}
                </div>
                <p class="text-xs text-muted-foreground">{{ stageStatusHintLabel(t, currentStageStatus) }}</p>
              </div>
              <RouterLink :to="`/certifications/${encodeURIComponent(pipelineId)}/timeline`" class="btn btn-outline rounded-lg py-1.5 text-xs">
                {{ t.learning.viewTimeline }}
              </RouterLink>
            </div>
          </div>
        </div>
      </div>

      <section
        v-if="finalQualificationRequired"
        class="mb-4 rounded-md border border-blue-200 bg-blue-50 p-5 shadow-[0_10px_24px_rgba(15,74,82,0.04)]"
      >
        <div class="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
          <div class="min-w-0">
            <div class="flex items-center gap-2 text-blue-950">
              <Award class="h-5 w-5 text-blue-700" />
              <h2 class="text-lg font-semibold">{{ t.learning.finalQualificationTitle }}</h2>
            </div>
            <p class="mt-2 max-w-3xl text-sm leading-6 text-blue-800">{{ t.learning.finalQualificationDesc }}</p>
            <div class="mt-4 grid gap-2 md:grid-cols-2">
              <div v-for="qual in finalQualifications" :key="qual.qualId" class="rounded-lg border border-blue-100 bg-white px-4 py-3">
                <div class="font-semibold text-blue-950">{{ qual.name || t.credentialsPage.availableQualifications }}</div>
                <p v-if="qual.description" class="mt-2 text-xs leading-5 text-slate-600">{{ qual.description }}</p>
                <div v-if="qual.constraints.length > 0" class="mt-3 space-y-2">
                  <div class="text-xs font-semibold uppercase tracking-wide text-blue-900">{{ t.credentialsPage.uploadMaterials }}</div>
                  <div v-for="constraint in qual.constraints" :key="constraintName(constraint) || String(constraint.type)" class="flex items-center gap-1 rounded-md bg-blue-50 px-3 py-2">
                    <span v-if="constraintRequired(constraint)" class="text-sm font-bold text-destructive">*</span>
                    <span class="text-sm text-blue-950">{{ constraintName(constraint) || t.common.na }}</span>
                  </div>
                </div>
                <div v-else-if="credentialDefinitionsLoading" class="mt-3 flex items-center gap-2 text-xs text-blue-700">
                  <Loader2 class="h-3.5 w-3.5 animate-spin" />
                  {{ t.common.loading }}
                </div>
              </div>
            </div>
          </div>
          <button class="btn btn-primary shrink-0 rounded-lg" :disabled="finalQualificationLoading" @click="handleFinalQualificationApplication">
            <Loader2 v-if="finalQualificationLoading" class="h-4 w-4 animate-spin" />
            <Award v-else class="h-4 w-4" />
            {{ t.learning.finalQualificationSubmitButton }}
          </button>
        </div>
      </section>

      <section
        v-if="purchased && pipelineHasCertificate"
        class="mb-4 rounded-md border border-slate-200 bg-white p-5 shadow-[0_10px_24px_rgba(15,23,42,0.04)]"
      >
        <div v-if="certificateAvailable" class="space-y-5">
          <div class="flex items-center gap-2 text-foreground">
            <Award class="h-5 w-5 text-orange-500" />
            <h2 class="text-lg font-semibold">{{ t.learning.certificatePanelTitle }}</h2>
          </div>
          <p class="max-w-3xl text-sm leading-6 text-muted-foreground">{{ t.learning.certificatePanelDesc }}</p>

          <div class="relative overflow-hidden rounded-lg bg-emerald-600 px-5 py-5 text-white">
            <div class="relative z-10 flex items-center justify-between gap-4">
              <div class="min-w-0">
                <div class="flex items-center gap-2 text-xl font-semibold">
                  <Sparkles class="h-5 w-5" />
                  {{ t.learning.certificateCongratulationsTitle }}
                </div>
                <p class="mt-2 text-sm font-medium text-emerald-50">{{ certificateDescription }}</p>
              </div>
              <Award class="h-14 w-14 shrink-0 text-white" />
            </div>
          </div>

          <div class="rounded-lg border border-slate-200 bg-white p-5">
            <h3 class="text-base font-semibold text-foreground">{{ t.learning.certificateDetailsTitle }}</h3>
            <div class="mt-5 grid gap-5 sm:grid-cols-2">
              <div>
                <p class="text-sm text-muted-foreground">{{ t.certificatesPage.title }}</p>
                <p class="mt-2 text-sm font-medium text-foreground">{{ pipeline.name || t.common.unknownCourse }}</p>
              </div>
              <div>
                <p class="text-sm text-muted-foreground">{{ t.certificatesPage.issueDate }}</p>
                <p class="mt-2 text-sm font-medium text-foreground">{{ certificateIssuedDate }}</p>
              </div>
            </div>
          </div>

          <button
            class="btn btn-primary rounded-lg"
            :disabled="certificateLoading"
            @click="openCertificate"
          >
            <Loader2 v-if="certificateLoading" class="h-4 w-4 animate-spin" />
            <ExternalLink v-else class="h-4 w-4" />
            {{ t.learning.certificateViewCenterButton }}
          </button>
        </div>
        <div v-else class="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div class="min-w-0">
            <div class="flex items-center gap-2 text-foreground">
              <Award class="h-5 w-5 text-orange-500" />
              <h2 class="text-lg font-semibold">{{ t.learning.certificatePanelTitle }}</h2>
            </div>
            <p class="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">{{ certificateDescription }}</p>
            <div class="mt-4 grid gap-3 sm:grid-cols-2">
              <div class="rounded-lg border border-slate-100 bg-slate-50 px-4 py-3">
                <p class="text-xs font-semibold uppercase tracking-wide text-muted-foreground">{{ t.certificatesPage.title }}</p>
                <p class="mt-1 text-sm font-medium text-foreground">{{ pipeline.name || t.common.unknownCourse }}</p>
              </div>
              <div class="rounded-lg border border-slate-100 bg-slate-50 px-4 py-3">
                <p class="text-xs font-semibold uppercase tracking-wide text-muted-foreground">{{ t.learning.currentStageStatusLabel }}</p>
                <p class="mt-1 text-sm font-medium text-foreground">
                  {{ certificateAvailable ? t.learning.certificationCertificateAvailableTag : pipelineIssuingCertificate ? t.learning.certificationCertificateIssuingTag : finalQualificationRequired ? t.learning.certificationFinalQualRequiredTag : t.learning.certificationCertificateAfterExamTag }}
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="rounded-md bg-white p-6">
        <div class="mb-4 flex flex-wrap items-end justify-between gap-3">
          <div>
            <h2 class="text-lg font-semibold text-foreground">{{ t.courses.stageListTitle }}</h2>
            <p class="mt-1 text-sm text-muted-foreground">{{ t.courses.stageListDesc }}</p>
          </div>
          <span class="badge border-slate-200 bg-slate-50 text-slate-700">{{ stages.length }} {{ t.courses.stages }} / {{ totalUnits }} {{ t.courses.units }}</span>
        </div>

        <LoadingState v-if="stageListLoading" :label="t.common.loading" variant="section" :rows="3" />
        <div v-else-if="stages.length === 0" class="rounded-md bg-slate-50 p-8 text-center text-muted-foreground">
          <div class="mx-auto max-w-md space-y-4">
            <div>
              <h3 class="text-base font-semibold text-foreground">{{ t.courses.noStagesTitle }}</h3>
              <p class="mt-2 text-sm">{{ t.courses.noStagesDesc }}</p>
            </div>
            <RouterLink :to="purchased ? '/my-certifications' : '/certifications'" class="btn btn-primary mx-auto w-fit rounded-lg">
              {{ t.courses.backToPipelines }}
            </RouterLink>
          </div>
        </div>
        <div v-else class="space-y-3">
          <div
            v-for="(stage, stageIndex) in stages"
            :key="stage.stage_id || stageIndex"
            :class="[
              'overflow-hidden rounded-md border bg-white',
              stageIndex === activeStageIndex ? 'border-primary/25' : 'border-slate-100',
            ]"
          >
          <div class="flex flex-col gap-4 border-b border-slate-100 px-5 py-4 md:flex-row md:items-center md:justify-between">
            <div class="flex items-center gap-3">
              <div
                :class="[
                  'flex h-10 w-10 items-center justify-center rounded-lg text-sm font-semibold',
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
              <span class="badge border-slate-200 bg-slate-50 text-slate-700">{{ t.learning.stageOrderLabel }} {{ stage.sort_order || stageIndex + 1 }}</span>
            </div>
          </div>

          <div v-if="visibleStageUnits(stage).length > 0">
            <component
              :is="unit.glms_course_id ? RouterLink : 'div'"
              v-for="(unit, unitIndex) in visibleStageUnits(stage)"
              :key="unit.unit_id || unit.glms_course_id || `${stageIndex}-${unitIndex}`"
              :to="learningHref(unit.glms_course_id)"
              class="flex items-center justify-between gap-4 border-t border-slate-50 px-5 py-4 transition-colors first:border-t-0 hover:bg-slate-50"
            >
              <div class="flex items-center gap-3">
                <div
                  :class="[
                    'flex h-8 w-8 items-center justify-center rounded-full',
                    purchased && stageIndex === activeStageIndex && (!nextStep.course_id || unit.glms_course_id === nextStep.course_id)
                      ? 'bg-primary text-primary-foreground'
                      : 'bg-primary/10 text-primary',
                  ]"
                >
                  <Play class="h-3.5 w-3.5 fill-current" />
                </div>
                <div>
                  <div class="font-medium text-foreground">
                    {{ (unit.glms_course_id && courseSummaries[unit.glms_course_id]?.title) || unit.name || unit.glms_course_id || t.common.unknownCourse }}
                  </div>
                  <div v-if="unit.glms_course_id && (courseSummaries[unit.glms_course_id]?.category_tips || courseSummaries[unit.glms_course_id]?.duration_min)" class="text-xs text-muted-foreground">
                    {{
                      [
                        courseSummaries[unit.glms_course_id]?.category_tips,
                        formatCourseDuration(courseSummaries[unit.glms_course_id]?.duration_min),
                      ]
                        .filter(Boolean)
                        .join(" · ")
                    }}
                  </div>
                </div>
              </div>
              <div class="flex flex-wrap items-center justify-end gap-2">
                <span :class="['badge', unitStateClass(unit)]">{{ t.learning.unitStatusLabel }}: {{ unitStateText(unit) }}</span>
                <span
                  v-if="unit.glms_course_id"
                  class="badge border-primary bg-primary text-primary-foreground"
                >
                  {{ t.courses.openLearning }}
                </span>
              </div>
            </component>
          </div>
          </div>
        </div>
      </section>

      <PurchaseDialog
        v-model:open="purchaseOpen"
        :course-name="pipeline.name || t.common.unknownCourse"
        :description="pipeline.description || ''"
        :pipeline-id="pipeline.pipeline_id || pipelineId"
      />
      <PaymentSessionDialog
        v-if="finalQualificationPaymentSession"
        v-model:open="finalQualificationPaymentOpen"
        :title="t.learning.finalQualificationPaymentTitle"
        :subtitle="finalQualificationPaymentSession.orderId"
        :payment-key="finalQualificationPaymentSession.paymentKey"
        :biz-type="finalQualificationPaymentSession.bizType"
        :biz-ref-ulid="finalQualificationPaymentSession.bizRefUlid"
        :order-id="finalQualificationPaymentSession.orderId"
        :source="finalQualificationPaymentSession.source"
        :return-path="finalQualificationPaymentSession.returnPath"
        :extra-return-params="finalQualificationPaymentSession.extraReturnParams"
      />
    </template>
      </main>
    </div>
  </AppShell>
</template>
