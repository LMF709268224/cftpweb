<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { RouterLink, useRoute, useRouter } from "vue-router"
import { toast } from "vue-sonner"
import {
  AlertCircle,
  ArrowLeft,
  Award,
  BookOpen,
  CheckCircle,
  Clock,
  ExternalLink,
  Loader2,
  Lock,
  Play,
  Sparkles,
} from "lucide-vue-next"
import {
  CANDIDATE_APPLICATION_STATUS_ENUM_NAMES,
  CANDIDATE_APPLICATION_STATUS_LABELS,
  courseUnitNextStepActionFromStatus,
  stageStatusHintLabel,
  statusLabel,
  timelineStatusBadgeClassForStatus,
  timelineStatusLabelWithDiagnostics,
  statusEnumNameForStatus,
  STAGE_STATUS_ENUM_NAMES,
} from "@/lib/status-labels"
import AppShell from "@/components/AppShell.vue"
import LoadingState from "@/components/LoadingState.vue"
import PageFeedback from "@/components/PageFeedback.vue"
import PaymentSessionDialog from "@/components/PaymentSessionDialog.vue"
import StageExemptionDialog from "@/components/StageExemptionDialog.vue"
import { ApiClientError, apiClient } from "@/lib/apiClient"
import {
  findInProgressStageOrder,
  isInProgressStagePurchaseConflict,
  reportsInProgressStagePurchase,
} from "@/lib/stageOrderRecovery"
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
  final_audit_quals?: Qualification[]
  award_certs?: Qualification[]
  has_certificate?: boolean
}

type StageConfig = {
  stage_id?: string
  name?: string
  sort_order?: number
  runtime_status?: string | number
  is_paid?: boolean
  units?: UnitConfig[]
}

type UnitConfig = {
  unit_id?: string
  name?: string
  glms_course_id?: string
  runtime_status?: string | number
  allow_retake?: boolean
  allow_exemption?: boolean
  exemption_quals?: Array<string | {
    qual_id?: string
    cred_def_ulid?: string
    name?: string
    name_hint?: string
    eligible?: boolean
    credential_status?: string
  }>
}

type Qualification = {
  qual_ulid?: string
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
  latest_application?: CredentialApplicationSummary | null
}

type CredentialApplicationSummary = {
  app_ulid?: string
  app_id?: string
  cred_def_ulid?: string
  cred_def_id?: string
  status?: string | number
  audit_remark?: string
  created_at?: string
}

type PipelineNextStep = {
  action?: string
  stage_id?: string
  stage_cc_ulid?: string
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
const { t, lang } = useTranslation()
const detail = ref<PipelineDetail | null>(null)
const courseSummaries = ref<Record<string, CourseSummary>>({})
const credentialDefinitions = ref<Record<string, CredentialDefinition>>({})
const firstCourseThumbnail = ref("")
const loading = ref(false)
const loadError = ref(false)
const courseSummariesLoading = ref(false)
const credentialDefinitionsLoading = ref(false)
const certificateLoading = ref(false)
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

const stagePaymentSession = ref<{
  paymentKey?: string
  orderId?: string
  stageId: string
  bizType: string
  bizRefUlid: string
  source: string
  returnPath: string
  extraReturnParams?: Record<string, string>
} | null>(null)
const stagePaymentDialogOpen = ref(false)
const stagePaymentLoading = ref(false)
const stagePaymentStageId = ref("")
const stageExemptionDialogOpen = ref(false)
const stageExemptionSubmitting = ref(false)
const stageExemptionOrderId = ref("")
const stageExemptionStage = ref<StageConfig | null>(null)
const stagePaymentSyncAttempts = 20
const stagePaymentSyncIntervalMs = 1000

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
const nextStep = computed<PipelineNextStep>(() => detail.value?.next_step || {})
const pipelineStatus = computed(() => detail.value?.pipeline_status)
const currentStageName = computed(() => detail.value?.current_stage_name || "")
const currentStageStatus = computed(() => detail.value?.current_stage_status)
const currentUnitStatus = computed(() => detail.value?.current_unit_status)
const nextUnitStatus = computed(() => nextStep.value?.status || currentUnitStatus.value)
const nextStepAction = computed(() =>
  nextStep.value?.action || courseUnitNextStepActionFromStatus(nextUnitStatus.value, Boolean(nextStep.value?.allow_retake)),
)
const finalQualifications = computed(() => {
  const quals = pipeline.value?.final_audit_quals || []
  return Array.isArray(quals)
    ? quals
        .map((qual) => ({
          qualId: firstString(qual?.qual_ulid),
          name: firstString(qual?.name_hint, qual?.nameHint, qual?.name),
        }))
        .map((qual) => {
          const definition = credentialDefinitions.value[qual.qualId]
          return {
            ...qual,
            name: firstString(definition?.name, qual.name),
            description: firstString(definition?.description),
            constraints: fileConstraintsOfDefinition(definition),
            application: definition?.latest_application || null,
          }
        })
        .filter((qual) => qual.qualId)
    : []
})
const finalQualificationIds = computed(() => {
  const quals = pipeline.value?.final_audit_quals || []
  return Array.isArray(quals)
    ? quals
        .map((qual) => firstString(qual?.qual_ulid))
        .filter((id): id is string => Boolean(id))
    : []
})
const finalQualificationIdsKey = computed(() => finalQualificationIds.value.join(","))
const pipelineHasCertificate = computed(() => {
  if (pipeline.value?.has_certificate) return true
  return Boolean(pipeline.value?.award_certs?.length) || nextStepAction.value === "view_certificate"
})
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
type FinalQualificationActionState = "loading" | "submit" | "pending_upload" | "pending" | "approved" | "resubmit"

const finalQualificationActionState = computed<FinalQualificationActionState>(() => {
  if (credentialDefinitionsLoading.value) return "loading"

  const applications = finalQualifications.value.map((qual) => qual.application)
  if (applications.some((application) => !application || isApplicationRejectedStatus(application?.status))) return "submit"
  if (applications.some((application) => isApplicationResubmitStatus(application?.status))) return "resubmit"
  if (applications.some((application) => isApplicationPendingUploadStatus(application?.status))) return "pending_upload"
  if (applications.some((application) => isApplicationPendingStatus(application?.status))) return "pending"
  if (applications.length > 0 && applications.every((application) => isApplicationApprovedStatus(application?.status))) {
    return "approved"
  }
  return "submit"
})

const finalQualificationPanelDescription = computed(() => {
  switch (finalQualificationActionState.value) {
    case "pending":
      return t.value.learning.finalQualificationPendingDesc
    case "pending_upload":
      return t.value.learning.finalQualificationDesc
    case "approved":
      return t.value.learning.finalQualificationApprovedDesc
    case "resubmit":
      return t.value.learning.finalQualificationResubmitDesc
    default:
      return t.value.learning.finalQualificationDesc
  }
})

const finalQualificationActionLabel = computed(() => {
  switch (finalQualificationActionState.value) {
    case "loading":
      return t.value.common.loading
    case "pending":
      return t.value.credentialsPage.applicationPendingHint
    case "pending_upload":
      return t.value.credentialsPage.uploadMaterials
    case "approved":
      return t.value.credentialsPage.applicationApprovedHint
    case "resubmit":
      return t.value.credentialsPage.appStatusResubmit
    default:
      return t.value.learning.finalQualificationSubmitButton
  }
})

const finalQualificationActionDisabled = computed(() =>
  finalQualificationLoading.value ||
  ["loading", "pending", "approved"].includes(finalQualificationActionState.value),
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
  if (certificateAvailable.value) return t.value.learning.certificateCongratulationsDesc
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
    lang.value,
  ) || "-",
)
const visibleCourseIds = computed(() =>
  Array.from(
    new Set(
      stages.value
        .flatMap((stage) => visibleStageUnits(stage))
        .map((unit) => unit.glms_course_id)
        .filter((id): id is string => Boolean(id)),
    ),
  ),
)
const visibleCourseIdsKey = computed(() => visibleCourseIds.value.join(","))
const firstCourseId = computed(() => visibleCourseIds.value[0] || "")
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

function isStageWaitCandidate(stage: StageConfig) {
  return statusEnumNameForStatus(STAGE_STATUS_ENUM_NAMES, stage.runtime_status) === "STAGE_STATUS_WAIT_CANDIDATE"
}

function stageStatusLabel(status?: string | number | null) {
  return timelineStatusLabelWithDiagnostics(t.value, "STAGE", status)
}

function unitStatusLabel(status?: string | number | null) {
  return timelineStatusLabelWithDiagnostics(t.value, "COURSE_UNIT", status)
}

function stageStateText(index: number) {
  const stage = stages.value[index]
  const status = stage?.runtime_status
  if (!purchased.value) return t.value.courses.positionNotPurchased
  if (!hasRuntimeStatus(status)) {
    return stage?.is_paid ? t.value.courses.positionNotStarted : t.value.courses.positionNotPurchased
  }
  return stageStatusLabel(status)
}

function stageStateClass(index: number) {
  const stage = stages.value[index]
  const status = stage?.runtime_status
  if (!purchased.value || !hasRuntimeStatus(status)) return "border-slate-200 bg-slate-50 text-slate-600"
  return timelineStatusBadgeClassForStatus("STAGE", status)
}

function unitStateText(unit: UnitConfig) {
  if (!purchased.value || !unit.runtime_status) return t.value.courses.positionNotPurchased
  return unitStatusLabel(unit.runtime_status)
}

function unitStateClass(unit: UnitConfig) {
  if (!purchased.value || !unit.runtime_status) return "border-slate-200 bg-slate-50 text-slate-600"
  return timelineStatusBadgeClassForStatus("COURSE_UNIT", unit.runtime_status)
}

function learningHref(courseId?: string) {
  return courseId
    ? `/certifications/${encodeURIComponent(pipelineId.value)}/learn/${encodeURIComponent(courseId)}`
    : "/certifications"
}

async function loadDetail(showLoading = true, suppressErrorToast = false, handlePageError = false) {
  if (!pipelineId.value) {
    detail.value = null
    if (handlePageError) loadError.value = false
    if (showLoading) loading.value = false
    return false
  }
  if (showLoading) loading.value = true
  if (handlePageError) loadError.value = false
  try {
    detail.value = await apiClient(`/api/mall/pipelines/${pipelineId.value}/runtime`, { suppressErrorToast })
    return true
  } catch (error) {
    if (!handlePageError) throw error
    console.error(error)
    loadError.value = true
    return false
  } finally {
    if (showLoading) loading.value = false
  }
}

async function initializeDetailPage() {
  const loaded = await loadDetail(true, true, true)
  if (loaded) await handleStagePaymentReturn()
}

async function loadCourseSummaries() {
  if (!purchased.value) {
    courseSummariesLoading.value = false
    courseSummaries.value = {}
    return
  }
  const courseIds = visibleCourseIds.value
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

function normalizedCredentialApplicationStatus(status: unknown) {
  const enumName = statusEnumNameForStatus(CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, status as string)
  return firstString(enumName, status).toUpperCase()
}

function isApplicationPendingStatus(status: unknown) {
  return normalizedCredentialApplicationStatus(status) === "APPLICATION_STATUS_PENDING"
}

function isApplicationPendingUploadStatus(status: unknown) {
  return normalizedCredentialApplicationStatus(status) === "APPLICATION_STATUS_PENDING_UPLOAD"
}

function isApplicationApprovedStatus(status: unknown) {
  return normalizedCredentialApplicationStatus(status) === "APPLICATION_STATUS_APPROVED"
}

function isApplicationResubmitStatus(status: unknown) {
  const value = normalizedCredentialApplicationStatus(status)
  return value === "APPLICATION_STATUS_RESUBMIT" || value === "APPLICATION_STATUS_REUPLOAD"
}

function isApplicationRejectedStatus(status: unknown) {
  return normalizedCredentialApplicationStatus(status) === "APPLICATION_STATUS_REJECTED"
}

function finalQualificationApplicationStatusLabel(status: unknown) {
  return statusLabel(
    t.value,
    CANDIDATE_APPLICATION_STATUS_LABELS,
    status as string | number,
    "credentialsPage.appStatusUnknown",
  )
}

function finalQualificationApplicationStatusClass(status: unknown) {
  const value = normalizedCredentialApplicationStatus(status)
  if (value === "APPLICATION_STATUS_APPROVED") return "border-emerald-200 bg-emerald-50 text-emerald-700"
  if (value === "APPLICATION_STATUS_REJECTED") return "border-red-200 bg-red-50 text-red-700"
  if (value === "APPLICATION_STATUS_RESUBMIT" || value === "APPLICATION_STATUS_REUPLOAD") {
    return "border-amber-200 bg-amber-50 text-amber-700"
  }
  if (value === "APPLICATION_STATUS_PENDING_UPLOAD") return "border-sky-200 bg-sky-50 text-sky-700"
  if (value === "APPLICATION_STATUS_PENDING") return "border-blue-200 bg-blue-50 text-blue-700"
  return "border-slate-200 bg-slate-50 text-slate-600"
}

function finalQualificationApplicationStatusIcon(status: unknown) {
  const value = normalizedCredentialApplicationStatus(status)
  if (value === "APPLICATION_STATUS_APPROVED") return CheckCircle
  if (
    value === "APPLICATION_STATUS_REJECTED" ||
    value === "APPLICATION_STATUS_RESUBMIT" ||
    value === "APPLICATION_STATUS_REUPLOAD"
  ) {
    return AlertCircle
  }
  return Clock
}

function shouldShowFinalQualificationRequirements(qual: { application?: CredentialApplicationSummary | null }) {
  return isApplicationPendingUploadStatus(qual.application?.status) || isApplicationResubmitStatus(qual.application?.status)
}

function finalQualificationUploadPath(qualIds = finalQualificationIds.value) {
  const params = new URLSearchParams()
  if (qualIds.length > 0) params.set("qual_ulids", qualIds.join(","))
  return `/credentials${params.toString() ? `?${params.toString()}` : ""}`
}

function openFinalQualificationUpload(qualIds: string[]) {
  window.setTimeout(() => router.push(finalQualificationUploadPath(qualIds)), 300)
}

async function latestCredentialApplication(qualId: string) {
  const response = await apiClient(`/api/credentials/applications?cred_def_ulid=${encodeURIComponent(qualId)}`, {
    suppressErrorToast: true,
  })
  return (response?.applications || [])[0] || null
}

function isInProgressCredentialApplicationError(error: unknown) {
  if (!(error instanceof ApiClientError)) return false
  const message = firstString(error.rawMessage, error.errorCode, error.message).toLowerCase()
  return error.status === 409 && (
    message.includes("in-progress credential application")
    || message.includes("进行中")
    || message.includes("请先处理")
  )
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
    let missingQualIds = await missingFinalQualificationIds()
    if (missingQualIds.length === 0) {
      toast.success(t.value.learning.finalQualificationApproved)
      await loadDetail()
      return
    }

    const existingApplications = await Promise.all(
      missingQualIds.map(async (qualId) => ({
        qualId,
        application: await latestCredentialApplication(qualId),
      })),
    )
    const approvedQualIds = new Set(
      existingApplications
        .filter(({ application }) => isApplicationApprovedStatus(application?.status))
        .map(({ qualId }) => qualId),
    )
    missingQualIds = missingQualIds.filter((qualId) => !approvedQualIds.has(qualId))
    if (missingQualIds.length === 0) {
      toast.success(t.value.learning.finalQualificationApproved)
      await loadDetail()
      return
    }

    const createTarget = existingApplications.find(({ qualId, application }) =>
      missingQualIds.includes(qualId)
      && !isApplicationPendingStatus(application?.status)
      && !isApplicationPendingUploadStatus(application?.status)
      && !isApplicationResubmitStatus(application?.status)
      && !isApplicationApprovedStatus(application?.status),
    )
    if (!createTarget) {
      const pendingUploadApplication = existingApplications.find(({ application }) =>
        isApplicationPendingUploadStatus(application?.status),
      )
      if (pendingUploadApplication) {
        toast.info(t.value.learning.finalQualificationUploadReady)
        openFinalQualificationUpload([pendingUploadApplication.qualId])
        return
      }
      const resubmitApplication = existingApplications.find(({ application }) =>
        isApplicationResubmitStatus(application?.status),
      )
      if (resubmitApplication) {
        toast.info(t.value.learning.finalQualificationResubmit)
        openFinalQualificationUpload([resubmitApplication.qualId])
        return
      }
      toast.info(t.value.learning.finalQualificationUnderReview)
      return
    }
    const targetQualId = createTarget.qualId

    const bundleId = await resolveBundleIdForPipeline()
    if (!bundleId) {
      toast.error(t.value.learning.finalQualificationBundleMissing)
      return
    }
    let order
    try {
      order = await apiClient("/api/credentials/application-orders", {
        method: "POST",
        suppressErrorToast: true,
        body: JSON.stringify({
          pipeline_cc_ulid: pipelineId.value,
          bundle_ulid: bundleId,
          qual_ulids: [targetQualId],
        }),
      })
    } catch (error) {
      if (isInProgressCredentialApplicationError(error)) {
        toast.info(t.value.learning.finalQualificationUnderReview)
        openFinalQualificationUpload([targetQualId])
        return
      }
      throw error
    }
    const orderId = firstString(order?.application_order_ulid, order?.application_order_id)
    const orderStatus = firstString(order?.order_status, order?.status)
    if (isUploadReadyStatus(orderStatus)) {
      toast.info(t.value.learning.finalQualificationUploadReady)
      openFinalQualificationUpload([targetQualId])
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
        extraReturnParams: { qual_ulids: targetQualId },
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
  } catch (error: any) {
    console.error(error)
    toast.error(error?.message || t.value.common.error)
  } finally {
    finalQualificationLoading.value = false
  }
}

function normalizeStageOrderStatus(value: unknown) {
  return String(value || "").trim().toUpperCase()
}

function resetStageExemptionSelection() {
  stageExemptionDialogOpen.value = false
  stageExemptionOrderId.value = ""
  stageExemptionStage.value = null
}

function stageHasExemptionChoices(stage: StageConfig) {
  return (stage.units || []).some((unit) =>
    Boolean(unit.allow_exemption || (unit.exemption_quals || []).length > 0),
  )
}

function stageExemptionSelectionJSON(
  stageCcUlid: string,
  exemptedUnitIds: string[] = [],
  waivedUnitIds: string[] = [],
) {
  return JSON.stringify({
    stage_cc_ulid: stageCcUlid,
    exempted_unit_cc_ulids: exemptedUnitIds,
    waived_unit_cc_ulids: waivedUnitIds,
  })
}

async function createStageOrder(
  stage: StageConfig,
  exemptedUnitIds: string[] = [],
  waivedUnitIds: string[] = [],
) {
  const stageCcUlid = firstString(stage.stage_id)
  const stageUlid = firstString(nextStep.value?.stage_id)
  if (!stageCcUlid || !pipelineId.value || !instancePipelineId.value || !stageUlid) return null

  return apiClient(`/api/mall/pipelines/${encodeURIComponent(pipelineId.value)}/stages/${encodeURIComponent(stageCcUlid)}/purchase`, {
    method: "POST",
    suppressErrorToast: true,
    body: JSON.stringify({
      pipeline_ulid: instancePipelineId.value,
      stage_ulid: stageUlid,
      selected_exemptions_json: stageExemptionSelectionJSON(stageCcUlid, exemptedUnitIds, waivedUnitIds),
    }),
  })
}

async function openStagePayment(stageOrderId: string, stage: StageConfig) {
  const returnPath = `${window.location.pathname}${window.location.search}${window.location.hash}`
  const stageId = firstString(stage.stage_id)
  const paymentReturnUrl = (paymentStatus: "success" | "cancelled") => {
    const returnUrl = new URL(returnPath, window.location.origin)
    returnUrl.searchParams.set("payment_status", paymentStatus)
    returnUrl.searchParams.set("payment_action", "stage")
    returnUrl.searchParams.set("order_id", stageOrderId)
    if (stageId) returnUrl.searchParams.set("stage_id", stageId)
    return returnUrl.toString()
  }
  const initResp = await apiClient("/api/mall/payments/initiate", {
    method: "POST",
    body: JSON.stringify({
      biz_type: "STAGE_PAYMENT",
      biz_ref_ulid: stageOrderId,
      success_url: paymentReturnUrl("success"),
      cancel_url: paymentReturnUrl("cancelled"),
    }),
  })

  stagePaymentSession.value = {
    paymentKey: initResp?.payment_key,
    orderId: stageOrderId,
    stageId,
    bizType: "STAGE_PAYMENT",
    bizRefUlid: stageOrderId,
    source: "stage",
    returnPath,
    extraReturnParams: { stage_id: stageId },
  }
  stagePaymentDialogOpen.value = true
}

async function continueStageOrder(orderResponse: any, stage: StageConfig) {
  const stageOrderId = firstString(orderResponse?.stage_order_ulid)
  const orderStatus = normalizeStageOrderStatus(orderResponse?.order_status)
  if (!stageOrderId || !orderStatus) {
    throw new Error(t.value.learning.stageOrderUnexpectedStatus)
  }

  if (orderStatus === "COMPLETED") {
    resetStageExemptionSelection()
    toast.success(t.value.learning.stageUnlockCompleted)
    await loadDetail(false)
    return
  }
  // Existing pre-v9 orders may still be waiting for the old second-step declaration.
  if (orderStatus === "WAIT_EXEMPTION_SELECTION") {
    stageExemptionOrderId.value = stageOrderId
    stageExemptionStage.value = stage
    stageExemptionDialogOpen.value = true
    return
  }
  if (orderStatus === "WAIT_STAGE_PAYMENT") {
    resetStageExemptionSelection()
    await openStagePayment(stageOrderId, stage)
    return
  }
  throw new Error(t.value.learning.stageOrderUnexpectedStatus)
}

function showExistingStageOrderAction() {
  toast.error(t.value.learning.stageExistingOrderNeedsAction, {
    action: {
      label: t.value.learning.stageExistingOrderViewOrders,
      onClick: () => void router.push("/orders"),
    },
  })
}

async function redirectToExistingStageOrder() {
  try {
    const existingOrder = await findInProgressStageOrder()
    if (!existingOrder) return false
    toast.info(t.value.learning.stageExistingOrderRedirecting)
    await router.push("/orders")
    return true
  } catch (error) {
    console.error("Failed to check in-progress stage order", error)
    toast.error(t.value.learning.stageExistingOrderCheckFailed, {
      action: {
        label: t.value.learning.stageExistingOrderViewOrders,
        onClick: () => void router.push("/orders"),
      },
    })
    return true
  }
}

async function recoverInProgressStageOrder(error: unknown) {
  if (!isInProgressStagePurchaseConflict(error)) return false
  const serviceReportedExistingOrder = reportsInProgressStagePurchase(error)

  try {
    const existingOrder = await findInProgressStageOrder()
    if (!existingOrder) {
      if (!serviceReportedExistingOrder) return false
      showExistingStageOrderAction()
      return true
    }

    toast.info(t.value.learning.stageExistingOrderRedirecting)
    resetStageExemptionSelection()
    await router.push("/orders")
    return true
  } catch (recoveryError: any) {
    console.error("Failed to recover in-progress stage order", recoveryError)
  }

  if (!serviceReportedExistingOrder) return false
  showExistingStageOrderAction()
  return true
}

async function handleStagePaymentClick(stage: StageConfig) {
  if (stagePaymentLoading.value) return
  if (!stage.stage_id || !pipelineId.value || !instancePipelineId.value || !nextStep.value?.stage_id) return

  stagePaymentLoading.value = true
  stagePaymentStageId.value = stage.stage_id
  try {
    if (await redirectToExistingStageOrder()) return
    if (stageHasExemptionChoices(stage)) {
      stageExemptionOrderId.value = ""
      stageExemptionStage.value = stage
      stageExemptionDialogOpen.value = true
      return
    }
    const orderResponse = await createStageOrder(stage)
    if (orderResponse) await continueStageOrder(orderResponse, stage)
  } catch (err: any) {
    if (await recoverInProgressStageOrder(err)) return
    toast.error(err.message || t.value.common.error)
  } finally {
    stagePaymentLoading.value = false
    stagePaymentStageId.value = ""
  }
}

async function handleStageExemptionSubmit(selection: { exemptedUnitIds: string[]; waivedUnitIds: string[] }) {
  const stage = stageExemptionStage.value
  const stageCcUlid = firstString(stage?.stage_id)
  if (!stage || !stageCcUlid || stageExemptionSubmitting.value) return

  stageExemptionSubmitting.value = true
  try {
    const existingOrderId = stageExemptionOrderId.value
    const orderResponse = existingOrderId
      ? await apiClient(`/api/mall/stage-orders/${encodeURIComponent(existingOrderId)}/exemptions`, {
          method: "POST",
          body: JSON.stringify({
            stage_cc_ulid: stageCcUlid,
            exempted_unit_cc_ulids: selection.exemptedUnitIds,
            waived_unit_cc_ulids: selection.waivedUnitIds,
          }),
        })
      : await createStageOrder(
          stage,
          selection.exemptedUnitIds,
          selection.waivedUnitIds,
        )
    if (orderResponse) await continueStageOrder(orderResponse, stage)
  } catch (err: any) {
    if (await recoverInProgressStageOrder(err)) return
    toast.error(err.message || t.value.common.error)
  } finally {
    stageExemptionSubmitting.value = false
  }
}

function stageStillWaitsForCandidate(stageId: string) {
  if (!stageId) return stages.value.some(isStageWaitCandidate)
  const stage = stages.value.find((item) => firstString(item.stage_id) === stageId)
  return Boolean(stage && isStageWaitCandidate(stage))
}

function waitForStagePaymentSync() {
  return new Promise<void>((resolve) => window.setTimeout(resolve, stagePaymentSyncIntervalMs))
}

async function refreshAfterStagePayment(stageId: string) {
  for (let attempt = 0; attempt < stagePaymentSyncAttempts; attempt += 1) {
    try {
      await loadDetail(false, true)
      if (!stageStillWaitsForCandidate(stageId)) return true
    } catch (error) {
      console.error("Failed to synchronize stage payment", error)
    }
    if (attempt < stagePaymentSyncAttempts - 1) await waitForStagePaymentSync()
  }
  return false
}

async function handleStagePaymentComplete() {
  const stageId = stagePaymentSession.value?.stageId || ""
  stagePaymentDialogOpen.value = false
  stagePaymentSession.value = null
  stagePaymentLoading.value = true
  stagePaymentStageId.value = stageId
  try {
    const synchronized = await refreshAfterStagePayment(stageId)
    if (synchronized) toast.success(t.value.learning.stageUnlockCompleted)
    else toast.info(t.value.learning.stagePaymentSyncDelayed)
  } finally {
    stagePaymentLoading.value = false
    stagePaymentStageId.value = ""
  }
}

function stagePaymentReturnValue(name: string) {
  const value = route.query[name]
  return firstString(Array.isArray(value) ? value[0] : value)
}

async function clearStagePaymentReturnQuery() {
  const nextQuery = { ...route.query }
  delete nextQuery.payment_status
  delete nextQuery.payment_action
  delete nextQuery.order_id
  delete nextQuery.stage_id
  await router.replace({ path: route.path, query: nextQuery, hash: route.hash })
}

async function handleStagePaymentReturn() {
  const paymentAction = stagePaymentReturnValue("payment_action")
  const paymentStatus = stagePaymentReturnValue("payment_status").toLowerCase()
  if (paymentAction !== "stage" || !paymentStatus) return

  try {
    if (paymentStatus === "success") {
      const stageId = stagePaymentReturnValue("stage_id")
        || firstString(stages.value.find(isStageWaitCandidate)?.stage_id)
      stagePaymentLoading.value = true
      stagePaymentStageId.value = stageId
      const synchronized = await refreshAfterStagePayment(stageId)
      if (synchronized) toast.success(t.value.learning.stageUnlockCompleted)
      else toast.info(t.value.learning.stagePaymentSyncDelayed)
    } else if (paymentStatus === "cancelled") {
      toast.warning(t.value.paymentReturnHandler.cancelled)
    } else if (paymentStatus === "failed") {
      toast.error(t.value.paymentReturnHandler.failed)
    }
  } finally {
    stagePaymentLoading.value = false
    stagePaymentStageId.value = ""
    await clearStagePaymentReturnQuery()
  }
}

const detailPolling = usePolling(
  async () => {
    await loadDetail(false, true)
  },
  { shouldPoll: () => Boolean(pipelineId.value && pipelineIssuingCertificate.value) },
)

onMounted(async () => {
  await initializeDetailPage()
})
watch(pipelineId, () => {
  detailPolling.stop()
  detail.value = null
  void loadDetail(true, true, true)
})
watch(pipelineIssuingCertificate, (issuing) => {
  if (issuing && pipelineId.value) detailPolling.start()
  else detailPolling.stop()
})
watch(visibleCourseIdsKey, () => void loadCourseSummaries(), { immediate: true })
watch(finalQualificationIdsKey, () => void loadCredentialDefinitions(), { immediate: true })
watch(firstCourseId, () => void loadFirstCourseThumbnail(), { immediate: true })
watch(lang, async () => {
  try {
    await loadDetail(false, true)
    await loadCourseSummaries()
    await loadCredentialDefinitions()
  } catch (error) {
    console.warn("Failed to refresh localized certification detail", error)
  }
})
</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <BookOpen class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ pipeline?.name || t.common.unknownCourse }}</span>
      </header>

      <main class="course-detail-page px-5 py-8 md:px-8 lg:px-10">
        <RouterLink :to="purchased ? '/my-certifications' : '/certifications'" class="course-detail-back mb-6 inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground">
          <ArrowLeft class="h-4 w-4" />
          {{ t.courses.backToPipelines }}
        </RouterLink>

    <LoadingState v-if="loading" :label="t.common.loading" variant="page" :rows="4" />
    <PageFeedback
      v-else-if="loadError"
      kind="error"
      :title="t.learning.certificationDetailLoadFailed"
      :description="t.learning.certificationDetailLoadFailedDesc"
      :action-label="t.learning.retry"
      @action="initializeDetailPage"
    />
    <div v-else-if="!pipeline" class="course-detail-empty rounded-md bg-white p-8 text-center text-muted-foreground">
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
      <div :class="['course-detail-hero mb-4 rounded-md bg-white p-6', firstCourseThumbnail && 'grid gap-6 lg:grid-cols-[340px_1fr]']">
        <div v-if="firstCourseThumbnail" class="relative flex aspect-video items-center justify-center overflow-hidden rounded-md bg-muted">
          <img :src="firstCourseThumbnail" :alt="pipeline.name || t.common.unknownCourse" class="h-full w-full object-cover" />
          <div class="absolute inset-0 bg-gradient-to-t from-black/45 via-black/5 to-transparent" />
        </div>

        <div>
          <h1 class="mb-2 text-2xl font-bold text-foreground">{{ pipeline.name || t.common.unknownCourse }}</h1>
          <p v-if="pipeline.description" class="mb-4 max-w-3xl text-sm leading-6 text-muted-foreground">{{ pipeline.description }}</p>

          <div class="course-detail-meta mb-5 flex flex-wrap gap-6 text-sm text-muted-foreground">
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
              <span>{{ pipeline.award_certs?.length || 0 }} {{ t.courses.awardedCertificates }}</span>
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
        class="course-detail-panel mb-4 rounded-md border border-blue-200 bg-blue-50 p-5 shadow-[0_10px_24px_rgba(15,74,82,0.04)]"
      >
        <div class="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
          <div class="min-w-0">
            <div class="flex items-center gap-2 text-blue-950">
              <Award class="h-5 w-5 text-blue-700" />
              <h2 class="text-lg font-semibold">{{ t.learning.finalQualificationTitle }}</h2>
            </div>
            <p class="mt-2 max-w-3xl text-sm leading-6 text-blue-800">{{ finalQualificationPanelDescription }}</p>
            <div class="mt-4 grid gap-2 md:grid-cols-2">
              <div v-for="qual in finalQualifications" :key="qual.qualId" class="course-detail-subcard rounded-lg border border-blue-100 bg-white px-4 py-3">
                <div class="font-semibold text-blue-950">{{ qual.name || t.credentialsPage.availableQualifications }}</div>
                <p v-if="qual.description" class="mt-2 text-xs leading-5 text-slate-600">{{ qual.description }}</p>
                <div v-if="qual.application" class="mt-3 flex flex-wrap items-center gap-2">
                  <span
                    :class="[
                      'inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-semibold',
                      finalQualificationApplicationStatusClass(qual.application.status),
                    ]"
                  >
                    <component :is="finalQualificationApplicationStatusIcon(qual.application.status)" class="h-3.5 w-3.5" />
                    {{ finalQualificationApplicationStatusLabel(qual.application.status) }}
                  </span>
                  <span v-if="qual.application.created_at" class="text-xs text-slate-500">
                    {{ t.credentialsPage.submittedAt }} {{ formatBackendDateOnly(qual.application.created_at, lang) }}
                  </span>
                </div>
                <p
                  v-if="qual.application?.audit_remark"
                  class="mt-2 rounded-md bg-slate-50 px-3 py-2 text-xs leading-5 text-slate-600"
                >
                  {{ t.credentialsPage.auditRemark }}: {{ qual.application.audit_remark }}
                </p>
                <div
                  v-if="shouldShowFinalQualificationRequirements(qual) && qual.constraints.length > 0"
                  class="mt-3 space-y-2"
                >
                  <div class="text-xs font-semibold uppercase tracking-wide text-blue-900">{{ t.credentialsPage.uploadMaterials }}</div>
                  <div v-for="constraint in qual.constraints" :key="constraintName(constraint) || String(constraint.type)" class="flex items-center gap-1 rounded-md bg-blue-50 px-3 py-2">
                    <span v-if="constraintRequired(constraint)" class="text-sm font-bold text-destructive">*</span>
                    <span class="text-sm text-blue-950">{{ constraintName(constraint) || t.common.na }}</span>
                  </div>
                </div>
                <div v-else-if="credentialDefinitionsLoading" class="mt-3 flex items-center gap-2 text-xs text-blue-700">
                  <Loader2 class="h-3.5 w-3.5 animate-spin text-primary" />
                  {{ t.common.loading }}
                </div>
              </div>
            </div>
          </div>
          <button
            data-testid="final-qualification-action"
            class="btn btn-primary shrink-0 rounded-lg"
            :disabled="finalQualificationActionDisabled"
            @click="handleFinalQualificationApplication"
          >
            <Loader2
              v-if="finalQualificationLoading || finalQualificationActionState === 'loading'"
              class="h-4 w-4 animate-spin"
            />
            <Clock v-else-if="finalQualificationActionState === 'pending'" class="h-4 w-4" />
            <CheckCircle v-else-if="finalQualificationActionState === 'approved'" class="h-4 w-4" />
            <AlertCircle v-else-if="finalQualificationActionState === 'resubmit'" class="h-4 w-4" />
            <Award v-else class="h-4 w-4" />
            {{ finalQualificationActionLabel }}
          </button>
        </div>
      </section>

      <section
        v-if="purchased && pipelineHasCertificate"
        class="course-detail-panel mb-4 rounded-md border border-slate-200 bg-white p-5 shadow-[0_10px_24px_rgba(15,23,42,0.04)]"
      >
        <div v-if="certificateAvailable" class="space-y-5">
          <div class="flex items-center gap-2 text-foreground">
            <Award class="h-5 w-5 text-orange-500" />
            <h2 class="text-lg font-semibold">{{ t.learning.certificatePanelTitle }}</h2>
          </div>
          <p class="max-w-3xl text-sm leading-6 text-muted-foreground">{{ t.learning.certificatePanelDesc }}</p>

          <div class="course-detail-certificate-banner relative overflow-hidden rounded-lg bg-emerald-600 px-5 py-5 text-white">
            <div class="course-detail-certificate-banner-content relative z-10 flex items-center justify-between gap-4">
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

          <div class="course-detail-certificate-info rounded-lg border border-slate-200 bg-white p-5">
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

      <section class="course-detail-stages rounded-md bg-white p-6">
        <div class="course-detail-stages-heading mb-4 flex flex-wrap items-end justify-between gap-3">
          <div>
            <h2 class="text-lg font-semibold text-foreground">{{ t.courses.stageListTitle }}</h2>
            <p class="mt-1 text-sm text-muted-foreground">{{ t.courses.stageListDesc }}</p>
          </div>
          <span class="badge border-slate-200 bg-slate-50 text-slate-700">{{ stages.length }} {{ t.courses.stages }} / {{ totalUnits }} {{ t.courses.units }}</span>
        </div>

        <LoadingState v-if="stageListLoading" :label="t.common.loading" variant="section" :rows="3" />
        <div v-else-if="stages.length === 0" class="course-detail-empty rounded-md bg-slate-50 p-8 text-center text-muted-foreground">
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
              'course-detail-stage overflow-hidden rounded-md border bg-white',
              stageIndex === activeStageIndex ? 'border-primary/25' : 'border-slate-100',
            ]"
          >
          <div class="course-detail-stage-header flex flex-col gap-4 border-b border-slate-100 px-5 py-4 md:flex-row md:items-center md:justify-between">
            <div class="course-detail-stage-title flex min-w-0 items-center gap-3">
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
              data-testid="course-unit-link"
              :data-course-id="unit.glms_course_id || ''"
              class="course-detail-unit flex items-center justify-between gap-4 border-t border-slate-50 px-5 py-4 transition-colors first:border-t-0 hover:bg-slate-50"
            >
              <div class="course-detail-unit-main flex min-w-0 items-center gap-3">
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
                <div class="min-w-0">
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
              <div class="course-detail-unit-actions flex flex-wrap items-center justify-end gap-2">
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
          <div v-else-if="isStageWaitCandidate(stage)" class="flex justify-center border-t border-slate-100 p-6">
            <button
              class="btn btn-primary rounded-lg"
              :disabled="stagePaymentLoading"
              @click="handleStagePaymentClick(stage)"
            >
              <Loader2 v-if="stagePaymentLoading && stagePaymentStageId === stage.stage_id" class="mr-2 h-4 w-4 animate-spin" />
              <Lock v-else class="mr-2 h-4 w-4" />
              {{ t.learning.actionWaitCandidate }}
            </button>
          </div>
          </div>
        </div>
      </section>

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
        @complete="loadDetail(false)"
      />

      <StageExemptionDialog
        v-model:open="stageExemptionDialogOpen"
        :stage="stageExemptionStage"
        :pipeline-id="pipelineId"
        :submitting="stageExemptionSubmitting"
        @submit="handleStageExemptionSubmit"
      />

      <PaymentSessionDialog
        v-if="stagePaymentSession"
        v-model:open="stagePaymentDialogOpen"
        :title="t.learning.actionWaitCandidate"
        :subtitle="stagePaymentSession.orderId"
        :payment-key="stagePaymentSession.paymentKey"
        :biz-type="stagePaymentSession.bizType"
        :biz-ref-ulid="stagePaymentSession.bizRefUlid"
        :order-id="stagePaymentSession.orderId"
        :source="stagePaymentSession.source"
        :return-path="stagePaymentSession.returnPath"
        :extra-return-params="stagePaymentSession.extraReturnParams"
        :redirect-on-complete="false"
        @complete="handleStagePaymentComplete"
      />
    </template>
      </main>
    </div>
  </AppShell>
</template>

<style scoped>
@media (max-width: 767px) {
  .course-detail-back {
    margin-bottom: 16px;
  }

  .course-detail-empty {
    padding: 32px 12px;
  }

  .course-detail-hero {
    gap: 16px;
    padding: 16px;
    border-radius: 12px;
  }

  .course-detail-hero h1 {
    font-size: 22px;
    line-height: 30px;
  }

  .course-detail-meta {
    gap: 10px 16px;
    margin-bottom: 16px;
  }

  .course-detail-panel {
    padding: 16px;
    border-radius: 12px;
  }

  .course-detail-panel .btn {
    width: 100%;
    min-height: 42px;
    justify-content: center;
  }

  .course-detail-subcard {
    padding: 12px;
  }

  .course-detail-certificate-banner {
    padding: 16px;
  }

  .course-detail-certificate-banner-content {
    align-items: flex-start;
  }

  .course-detail-certificate-banner-content > svg {
    width: 36px;
    height: 36px;
  }

  .course-detail-certificate-banner-content div div {
    font-size: 18px;
    line-height: 26px;
  }

  .course-detail-certificate-info {
    padding: 16px;
  }

  .course-detail-certificate-info > div {
    gap: 12px;
    margin-top: 12px;
  }

  .course-detail-stages {
    padding: 16px;
    border-radius: 12px;
  }

  .course-detail-stages-heading {
    align-items: flex-start;
    margin-bottom: 12px;
  }

  .course-detail-stage {
    border-radius: 8px;
  }

  .course-detail-stage-header {
    gap: 12px;
    padding: 12px;
  }

  .course-detail-stage-title > div:last-child,
  .course-detail-unit-main > div:last-child {
    min-width: 0;
  }

  .course-detail-stage-title h3,
  .course-detail-unit-main .font-medium {
    overflow-wrap: anywhere;
  }

  .course-detail-unit {
    flex-direction: column;
    align-items: stretch;
    gap: 10px;
    padding: 12px;
  }

  .course-detail-unit-actions {
    justify-content: flex-start;
    padding-left: 44px;
  }
}
</style>
