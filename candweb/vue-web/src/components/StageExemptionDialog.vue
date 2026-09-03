<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useRouter } from "vue-router"
import { toast } from "vue-sonner"
import { Check, CircleAlert, GraduationCap, Loader2, X } from "lucide-vue-next"
import PaymentSessionDialog from "@/components/PaymentSessionDialog.vue"
import { ApiClientError, apiClient } from "@/lib/apiClient"
import { useBodyScrollLock } from "@/lib/bodyScrollLock"
import { useTranslation } from "@/lib/language"
import { learningUnitDisplayName } from "@/lib/learningUnitNames"
import { CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, statusEnumNameForStatus } from "@/lib/status-labels"

type ExemptionQualification = string | {
  qual_id?: string
  cred_def_ulid?: string
  name?: string
  name_hint?: string
  eligible?: boolean
  credential_status?: string
}

type StageExemptionUnit = {
  unit_id?: string
  name?: string
  unit_name?: string
  allow_exemption?: boolean
  exemption_quals?: ExemptionQualification[]
}

type StageExemptionStage = {
  stage_id?: string
  stage_cc_ulid?: string
  name?: string
  stage_name?: string
  units?: StageExemptionUnit[]
}

type StageUnitPricing = {
  accessAmount?: number
  exemptionAmount: number
  currency: string
}

const props = defineProps<{
  open: boolean
  stage: StageExemptionStage | null
  pipelineId: string
  submitting?: boolean
}>()

const emit = defineEmits<{
  "update:open": [value: boolean]
  submit: [selection: { exemptedUnitIds: string[]; waivedUnitIds: string[] }]
}>()

const { t, lang } = useTranslation()
const router = useRouter()
type ExemptionDecision = "exempt" | "waive"
type QualificationApplicationTarget = {
  unitId: string
  unitName: string
  qualificationId: string
  qualificationName: string
}

type QualificationApplicationState = "none" | "pending_upload" | "resubmit" | "pending" | "approved" | "rejected"

const exemptionDecisions = ref<Record<string, ExemptionDecision>>({})
const selectedQualificationIds = ref<Record<string, string>>({})
const eligibleQualificationIds = ref<Set<string>>(new Set())
const qualificationApplications = ref<Record<string, any | null>>({})
const eligibilityLoading = ref(false)
const eligibilityLoadFailed = ref(false)
const pricingByUnit = ref<Record<string, StageUnitPricing>>({})
const pricingLoading = ref(false)
const pricingLoaded = ref(false)
const applicationConfirmTarget = ref<QualificationApplicationTarget | null>(null)
const applicationLoadingUnitId = ref("")
const resolvedBundleId = ref("")
const credentialPaymentOpen = ref(false)
const credentialPaymentSession = ref<{
  orderId: string
  qualificationId: string
} | null>(null)

useBodyScrollLock(() => props.open)

function firstString(...values: unknown[]) {
  for (const value of values) {
    const normalized = String(value || "").trim()
    if (normalized) return normalized
  }
  return ""
}

function qualificationId(qualification: ExemptionQualification) {
  if (typeof qualification === "string") return qualification.trim()
  return firstString(qualification.qual_id, qualification.cred_def_ulid)
}

function qualificationIsEligible(qualification: ExemptionQualification) {
  if (typeof qualification === "string") {
    return eligibleQualificationIds.value.has(qualification.trim())
  }
  const status = String(qualification.credential_status || "").trim().toUpperCase()
  return Boolean(
    qualification.eligible
    || status === "CREDENTIAL_STATUS_ACTIVE"
    || eligibleQualificationIds.value.has(qualificationId(qualification)),
  )
}

function qualificationsForUnit(unit: StageExemptionUnit) {
  return (unit.exemption_quals || []).filter((qualification) => qualificationId(qualification))
}

function selectedQualificationIdForUnit(unit: StageExemptionUnit) {
  const unitId = firstString(unit.unit_id)
  const qualifications = qualificationsForUnit(unit)
  const selectedId = selectedQualificationIds.value[unitId]
  if (selectedId && qualifications.some((qualification) => qualificationId(qualification) === selectedId)) {
    return selectedId
  }
  const eligible = qualifications.find(qualificationIsEligible)
  return qualificationId(eligible || qualifications[0] || "")
}

function selectedQualificationForUnit(unit: StageExemptionUnit) {
  const selectedId = selectedQualificationIdForUnit(unit)
  return qualificationsForUnit(unit).find((qualification) => qualificationId(qualification) === selectedId)
}

function selectedQualificationIsEligible(unit: StageExemptionUnit) {
  const qualification = selectedQualificationForUnit(unit)
  return Boolean(qualification && qualificationIsEligible(qualification))
}

const exemptionUnits = computed(() =>
  (props.stage?.units || []).filter((unit) =>
    Boolean(unit.allow_exemption || (unit.exemption_quals || []).some((qualification) => qualificationId(qualification))),
  ),
)

const selectedCount = computed(() =>
  exemptionUnits.value.filter((unit) => unit.unit_id && exemptionDecisions.value[unit.unit_id] === "exempt").length,
)
const undecidedCount = computed(() =>
  exemptionUnits.value.filter((unit) => unit.unit_id && !exemptionDecisions.value[unit.unit_id]).length,
)
const selectedCountText = computed(() =>
  t.value.learning.stageExemptionSelected.replace("{{count}}", String(selectedCount.value)),
)

function unitIsEligible(unit: StageExemptionUnit) {
  return (unit.exemption_quals || []).some(qualificationIsEligible)
}

function unitLabel(unit: StageExemptionUnit) {
  const name = firstString(unit.unit_name, unit.name, unit.unit_id, t.value.common.unknown)
  return learningUnitDisplayName(name, t.value.checkoutWizard)
}

function formatMoney(amount: number, currency: string) {
  const normalizedCurrency = firstString(currency, "USD").toUpperCase()
  try {
    return new Intl.NumberFormat(lang.value === "zh" ? "zh-CN" : "en-US", {
      style: "currency",
      currency: normalizedCurrency,
    }).format(amount / 100)
  } catch {
    return `${normalizedCurrency} ${(amount / 100).toFixed(2)}`
  }
}

function unitPriceText(unit: StageExemptionUnit, kind: "access" | "exemption") {
  if (pricingLoading.value) return t.value.learning.stageExemptionPriceLoading
  const pricing = pricingByUnit.value[firstString(unit.unit_id)]
  if (!pricingLoaded.value || !pricing) return t.value.learning.stageExemptionPriceUnavailable

  const amount = kind === "access" ? pricing.accessAmount : pricing.exemptionAmount
  if (typeof amount !== "number") return t.value.learning.stageExemptionPriceUnavailable
  const template = kind === "access"
    ? t.value.learning.stageExemptionAccessFee
    : t.value.learning.stageExemptionFee
  return template.replace("{{price}}", formatMoney(amount, pricing.currency))
}

function qualificationLabel(qualification: ExemptionQualification) {
  if (typeof qualification === "string") return qualification
  return firstString(
    qualification.name,
    qualification.name_hint,
    qualification.qual_id,
    qualification.cred_def_ulid,
    t.value.common.unknown,
  )
}

function close() {
  if (props.submitting || applicationLoadingUnitId.value) return
  applicationConfirmTarget.value = null
  emit("update:open", false)
}

function setUnitDecision(unit: StageExemptionUnit, decision: ExemptionDecision) {
  if (
    !unit.unit_id
    || (decision === "exempt" && !selectedQualificationIsEligible(unit))
    || (decision === "waive" && selectedQualificationIsEligible(unit))
  ) return
  exemptionDecisions.value = {
    ...exemptionDecisions.value,
    [unit.unit_id]: decision,
  }
}

function selectQualification(unit: StageExemptionUnit, event: Event) {
  const unitId = firstString(unit.unit_id)
  const selectedId = String((event.target as HTMLSelectElement).value || "").trim()
  if (!unitId || !selectedId) return
  selectedQualificationIds.value = {
    ...selectedQualificationIds.value,
    [unitId]: selectedId,
  }
  if (exemptionDecisions.value[unitId] === "exempt" && !selectedQualificationIsEligible(unit)) {
    const nextDecisions = { ...exemptionDecisions.value }
    delete nextDecisions[unitId]
    exemptionDecisions.value = nextDecisions
  }
}

function handleExemptionAction(unit: StageExemptionUnit) {
  if (selectedQualificationIsEligible(unit)) {
    setUnitDecision(unit, "exempt")
    return
  }
  const qualification = selectedQualificationForUnit(unit)
  const unitId = firstString(unit.unit_id)
  if (!unitId || !qualification) return
  const selectedId = qualificationId(qualification)
  if (!selectedId) return
  const state = qualificationApplicationState(unit)
  if (state === "pending_upload" || state === "resubmit" || state === "pending") {
    openQualificationUpload(selectedId)
    return
  }
  if (state === "approved") {
    void loadEligibility()
    return
  }
  applicationConfirmTarget.value = {
    unitId,
    unitName: unitLabel(unit),
    qualificationId: selectedId,
    qualificationName: qualificationLabel(qualification),
  }
}

function closeApplicationConfirm() {
  if (applicationLoadingUnitId.value) return
  applicationConfirmTarget.value = null
}

function submit() {
  if (undecidedCount.value > 0) return
  const exemptedUnitIds = exemptionUnits.value
    .filter((unit) => unit.unit_id && exemptionDecisions.value[unit.unit_id] === "exempt")
    .map((unit) => String(unit.unit_id))
  const waivedUnitIds = exemptionUnits.value
    .filter((unit) => unit.unit_id && exemptionDecisions.value[unit.unit_id] === "waive")
    .map((unit) => String(unit.unit_id))
  emit("submit", { exemptedUnitIds, waivedUnitIds })
}

async function loadEligibility() {
  exemptionDecisions.value = {}
  selectedQualificationIds.value = {}
  eligibleQualificationIds.value = new Set()
  qualificationApplications.value = {}
  eligibilityLoadFailed.value = false

  const qualificationIds = Array.from(new Set(
    exemptionUnits.value
      .flatMap((unit) => unit.exemption_quals || [])
      .map(qualificationId)
      .filter(Boolean),
  ))
  if (qualificationIds.length === 0) return

  eligibilityLoading.value = true
  try {
    const [response, applications] = await Promise.all([
      apiClient(
        `/api/credentials/qualifications?qual_ulids=${encodeURIComponent(qualificationIds.join(","))}`,
        { suppressErrorToast: true },
      ),
      Promise.all(qualificationIds.map(async (id) => [id, await latestCredentialApplication(id)] as const)),
    ])
    qualificationApplications.value = Object.fromEntries(applications)
    eligibleQualificationIds.value = new Set(
      (Array.isArray(response?.qualifications) ? response.qualifications : [])
        .filter((qualification: any) =>
          Boolean(
            qualification?.eligible
            || String(qualification?.credential_status || "").trim().toUpperCase() === "CREDENTIAL_STATUS_ACTIVE",
          ),
        )
        .map((qualification: any) => firstString(qualification?.qual_id, qualification?.cred_def_ulid))
        .filter(Boolean),
    )
    exemptionDecisions.value = Object.fromEntries(
      exemptionUnits.value
        .filter((unit) => unit.unit_id && unitIsEligible(unit))
        .map((unit) => [String(unit.unit_id), "exempt" as const]),
    )
    selectedQualificationIds.value = Object.fromEntries(
      exemptionUnits.value
        .filter((unit) => unit.unit_id && qualificationsForUnit(unit).length > 0)
        .map((unit) => [String(unit.unit_id), selectedQualificationIdForUnit(unit)]),
    )
  } catch (error) {
    console.error("Failed to load stage exemption eligibility", error)
    eligibilityLoadFailed.value = true
  } finally {
    eligibilityLoading.value = false
  }
}

function normalizedStatus(status: unknown) {
  return String(status || "").trim().toUpperCase()
}

function applicationStatus(status: unknown) {
  const enumName = statusEnumNameForStatus(CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, status as string)
  return firstString(enumName, status).toUpperCase()
}

function applicationIsPendingUpload(status: unknown) {
  return applicationStatus(status) === "APPLICATION_STATUS_PENDING_UPLOAD"
}

function applicationIsResubmit(status: unknown) {
  const value = applicationStatus(status)
  return value === "APPLICATION_STATUS_RESUBMIT" || value === "APPLICATION_STATUS_REUPLOAD"
}

function applicationIsPending(status: unknown) {
  return applicationStatus(status) === "APPLICATION_STATUS_PENDING"
}

function applicationIsApproved(status: unknown) {
  return applicationStatus(status) === "APPLICATION_STATUS_APPROVED"
}

function applicationIsRejected(status: unknown) {
  return applicationStatus(status) === "APPLICATION_STATUS_REJECTED"
}

function selectedApplicationForUnit(unit: StageExemptionUnit) {
  return qualificationApplications.value[selectedQualificationIdForUnit(unit)] || null
}

function qualificationApplicationState(unit: StageExemptionUnit): QualificationApplicationState {
  const status = selectedApplicationForUnit(unit)?.status
  if (applicationIsPendingUpload(status)) return "pending_upload"
  if (applicationIsResubmit(status)) return "resubmit"
  if (applicationIsPending(status)) return "pending"
  if (applicationIsApproved(status)) return "approved"
  if (applicationIsRejected(status)) return "rejected"
  return "none"
}

function qualificationStatusLabel(unit: StageExemptionUnit) {
  if (selectedQualificationIsEligible(unit)) return t.value.learning.stageExemptionEligible
  switch (qualificationApplicationState(unit)) {
    case "pending_upload": return t.value.learning.stageExemptionPendingUpload
    case "resubmit": return t.value.learning.stageExemptionNeedsResubmit
    case "pending": return t.value.learning.stageExemptionUnderReview
    case "approved": return t.value.learning.stageExemptionApproved
    case "rejected": return t.value.learning.stageExemptionRejected
    default: return t.value.learning.stageExemptionUnavailable
  }
}

function qualificationStatusHint(unit: StageExemptionUnit) {
  if (selectedQualificationIsEligible(unit)) return t.value.learning.stageExemptionAutomaticHint
  switch (qualificationApplicationState(unit)) {
    case "pending_upload": return t.value.learning.stageExemptionPendingUploadHint
    case "resubmit": return t.value.learning.stageExemptionNeedsResubmitHint
    case "pending": return t.value.learning.stageExemptionUnderReviewHint
    case "approved": return t.value.learning.stageExemptionApprovedHint
    case "rejected": return t.value.learning.stageExemptionRejectedHint
    default: return ""
  }
}

function qualificationActionLabel(unit: StageExemptionUnit) {
  if (selectedQualificationIsEligible(unit)) return t.value.learning.stageExemptionApply
  switch (qualificationApplicationState(unit)) {
    case "pending_upload":
    case "resubmit":
      return t.value.learning.stageExemptionSubmitMaterials
    case "pending": return t.value.learning.stageExemptionViewReview
    case "approved": return t.value.learning.stageExemptionRecheck
    case "rejected": return t.value.learning.stageExemptionReapply
    default: return t.value.checkoutWizard.applyThisExemption
  }
}

function unitPaymentBlockedByApplication(unit: StageExemptionUnit) {
  return ["pending_upload", "resubmit", "pending"].includes(qualificationApplicationState(unit))
}

function orderIsUploadReady(status: unknown) {
  return normalizedStatus(status).includes("UPLOAD_READY")
}

function orderNeedsReviewFeePayment(status: unknown) {
  return normalizedStatus(status).includes("WAIT_REVIEW_FEE_PAYMENT")
}

function orderIsUnderReview(status: unknown) {
  return normalizedStatus(status).includes("UNDER_REVIEW")
}

function orderIsResolved(status: unknown) {
  const value = normalizedStatus(status)
  return value.includes("RESOLVED") || value.includes("APPROVED") || value.includes("COMPLETED")
}

function qualificationUploadPath(qualificationId: string) {
  return `/credentials?qual_ulids=${encodeURIComponent(qualificationId)}`
}

function openQualificationUpload(qualificationId: string) {
  emit("update:open", false)
  void router.push(qualificationUploadPath(qualificationId))
}

async function latestCredentialApplication(qualificationId: string) {
  const response = await apiClient(
    `/api/credentials/applications?cred_def_ulid=${encodeURIComponent(qualificationId)}`,
    { suppressErrorToast: true },
  )
  return (Array.isArray(response?.applications) ? response.applications : [])[0] || null
}

async function resolveBundleIdForPipeline() {
  if (resolvedBundleId.value) return resolvedBundleId.value
  const pipelineId = firstString(props.pipelineId)
  if (!pipelineId) return ""
  const response = await apiClient("/api/mall/bundles?page_size=100")
  const bundle = (Array.isArray(response?.bundles) ? response.bundles : []).find(
    (item: any) => firstString(item?.pipeline_id, item?.pipeline_cc_ulid) === pipelineId,
  )
  resolvedBundleId.value = firstString(bundle?.bundle_id, bundle?.bundle_ulid)
  return resolvedBundleId.value
}

async function loadPricing() {
  pricingByUnit.value = {}
  pricingLoading.value = true
  pricingLoaded.value = false
  try {
    const bundleId = await resolveBundleIdForPipeline()
    if (!bundleId) return

    const params = new URLSearchParams({ payment_mode: "BY_STAGE" })
    const response = await apiClient(
      `/api/mall/bundles/${encodeURIComponent(bundleId)}/pricing-detail?${params.toString()}`,
      { suppressErrorToast: true },
    )
    const rawDetail = response?.pricing_detail_json
    const detail = typeof rawDetail === "string" ? JSON.parse(rawDetail) : rawDetail
    const prices: Record<string, StageUnitPricing> = {}
    for (const unit of Array.isArray(detail?.units) ? detail.units : []) {
      const unitId = firstString(unit?.unit_id)
      if (!unitId) continue
      prices[unitId] = {
        accessAmount: typeof unit?.access?.amount === "number" ? unit.access.amount : undefined,
        exemptionAmount: typeof unit?.exemption?.amount === "number" ? unit.exemption.amount : 0,
        currency: firstString(unit?.exemption?.currency, unit?.access?.currency, "USD"),
      }
    }
    pricingByUnit.value = prices
    pricingLoaded.value = true
  } catch (error) {
    console.error("Failed to load stage exemption pricing", error)
  } finally {
    pricingLoading.value = false
  }
}

function isInProgressApplicationError(error: unknown) {
  if (!(error instanceof ApiClientError)) return false
  const message = firstString(error.rawMessage, error.errorCode, error.message).toLowerCase()
  return error.status === 409 && (
    message.includes("in-progress credential application")
    || message.includes("进行中")
    || message.includes("请先处理")
  )
}

async function confirmQualificationApplication() {
  const target = applicationConfirmTarget.value
  const pipelineId = firstString(props.pipelineId)
  if (!target || !pipelineId || applicationLoadingUnitId.value) return

  applicationLoadingUnitId.value = target.unitId
  try {
    const existingApplication = await latestCredentialApplication(target.qualificationId)
    if (applicationIsPendingUpload(existingApplication?.status) || applicationIsResubmit(existingApplication?.status)) {
      toast.info(t.value.checkoutWizard.qualificationUploadReady)
      applicationConfirmTarget.value = null
      openQualificationUpload(target.qualificationId)
      return
    }
    if (applicationIsPending(existingApplication?.status)) {
      toast.info(t.value.checkoutWizard.qualificationUnderReview)
      applicationConfirmTarget.value = null
      return
    }
    if (applicationIsApproved(existingApplication?.status)) {
      toast.success(t.value.checkoutWizard.qualificationAlreadyApproved)
      applicationConfirmTarget.value = null
      await loadEligibility()
      return
    }

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
          pipeline_cc_ulid: pipelineId,
          bundle_ulid: bundleId,
          qual_ulids: [target.qualificationId],
        }),
      })
    } catch (error) {
      if (isInProgressApplicationError(error)) {
        toast.info(t.value.checkoutWizard.qualificationUnderReview)
        applicationConfirmTarget.value = null
        return
      }
      throw error
    }

    const orderId = firstString(order?.application_order_ulid, order?.application_order_id)
    const orderStatus = firstString(order?.order_status, order?.status)
    applicationConfirmTarget.value = null
    if (orderIsUploadReady(orderStatus)) {
      toast.info(t.value.checkoutWizard.qualificationUploadReady)
      openQualificationUpload(target.qualificationId)
      return
    }
    if (orderNeedsReviewFeePayment(orderStatus) || order?.payment_key) {
      if (!orderId) throw new Error(t.value.checkoutWizard.qualificationApplicationFailed)
      credentialPaymentSession.value = {
        orderId,
        qualificationId: target.qualificationId,
      }
      emit("update:open", false)
      credentialPaymentOpen.value = true
      return
    }
    if (orderIsUnderReview(orderStatus)) {
      toast.info(t.value.checkoutWizard.qualificationUnderReview)
      return
    }
    if (orderIsResolved(orderStatus)) {
      toast.success(t.value.checkoutWizard.qualificationAlreadyApproved)
      await loadEligibility()
      return
    }
    toast.info(t.value.checkoutWizard.qualificationApplicationCreated)
  } catch (error: any) {
    console.error(error)
    toast.error(error?.message || t.value.checkoutWizard.qualificationApplicationFailed)
  } finally {
    applicationLoadingUnitId.value = ""
  }
}

watch(
  () => [props.open, props.stage?.stage_id, props.stage?.stage_cc_ulid, props.pipelineId],
  ([open]) => {
    if (open) {
      void loadEligibility()
      void loadPricing()
    }
  },
  { immediate: true },
)

watch(
  () => props.pipelineId,
  () => {
    resolvedBundleId.value = ""
  },
)
</script>

<template>
  <div v-if="open" class="app-safe-area-overlay fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4 backdrop-blur-sm">
    <div class="app-dialog-viewport flex max-h-[92vh] w-full max-w-2xl flex-col overflow-hidden rounded-2xl bg-white shadow-2xl shadow-slate-950/20">
      <div class="flex items-start justify-between gap-4 border-b border-slate-100 px-6 py-5">
        <div class="flex min-w-0 items-start gap-3">
          <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-blue-50 text-primary">
            <GraduationCap class="h-5 w-5" />
          </div>
          <div class="min-w-0">
            <h2 class="text-xl font-bold text-slate-950">{{ t.learning.stageExemptionTitle }}</h2>
            <p class="mt-1 text-sm text-slate-500">
              {{ stage?.stage_name || stage?.name || t.learning.stageExemptionStageFallback }}
            </p>
          </div>
        </div>
        <button
          type="button"
          class="rounded-xl p-2 text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-700"
          :aria-label="t.paymentSession.close"
          :disabled="submitting || Boolean(applicationLoadingUnitId)"
          @click="close"
        >
          <X class="h-5 w-5" />
        </button>
      </div>

      <div class="overflow-y-auto px-6 py-5">
        <div class="rounded-xl border border-blue-200 bg-blue-50 p-4 text-sm leading-6 text-blue-950">
          {{ t.learning.stageExemptionDesc }}
        </div>

        <div v-if="eligibilityLoading" class="mt-4 flex items-center justify-center gap-2 rounded-xl border border-slate-200 p-8 text-sm text-slate-500">
          <Loader2 class="h-4 w-4 animate-spin" />
          {{ t.learning.stageExemptionLoading }}
        </div>

        <div v-else-if="eligibilityLoadFailed" class="mt-4 rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm leading-6 text-amber-900">
          {{ t.learning.stageExemptionLoadFailed }}
        </div>

        <div v-else-if="exemptionUnits.length === 0" class="mt-4 rounded-xl border border-dashed border-slate-200 p-8 text-center text-sm text-slate-500">
          {{ t.learning.stageExemptionEmpty }}
        </div>

        <div v-else class="mt-4 space-y-3">
          <div
            v-for="unit in exemptionUnits"
            :key="unit.unit_id || unitLabel(unit)"
            data-testid="stage-exemption-unit"
            :data-unit-id="unit.unit_id"
            :class="[
              'rounded-xl border border-slate-200 p-4 transition-colors',
              unit.unit_id && exemptionDecisions[unit.unit_id] === 'exempt'
                ? 'border-emerald-300 bg-emerald-50/70 ring-1 ring-emerald-100'
                : unit.unit_id && exemptionDecisions[unit.unit_id] === 'waive'
                  ? 'border-blue-300 bg-blue-50/60 ring-1 ring-blue-100'
                  : '',
            ]"
          >
            <div class="flex min-w-0 items-start gap-3">
              <span
                :class="[
                  'mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-md border',
                  unit.unit_id && exemptionDecisions[unit.unit_id] === 'exempt'
                    ? 'border-emerald-500 bg-emerald-500 text-white'
                    : 'border-slate-300 bg-white text-transparent',
                ]"
              >
                <Check class="h-3.5 w-3.5 stroke-[3]" />
              </span>
              <span class="min-w-0 flex-1">
                <span class="flex flex-wrap items-center gap-2">
                  <span class="font-semibold text-slate-950">{{ unitLabel(unit) }}</span>
                  <span
                    :class="[
                      'badge text-xs',
                      selectedQualificationIsEligible(unit) || qualificationApplicationState(unit) === 'approved'
                        ? 'border-emerald-200 bg-emerald-50 text-emerald-700'
                        : qualificationApplicationState(unit) === 'pending'
                          ? 'border-blue-200 bg-blue-50 text-blue-700'
                          : qualificationApplicationState(unit) === 'rejected'
                            ? 'border-rose-200 bg-rose-50 text-rose-700'
                            : 'border-amber-200 bg-amber-50 text-amber-700',
                    ]"
                    data-testid="stage-exemption-status"
                    :data-status="qualificationApplicationState(unit)"
                  >
                    {{ qualificationStatusLabel(unit) }}
                  </span>
                </span>
                <span v-if="qualificationsForUnit(unit).length === 1" class="mt-2 flex flex-wrap gap-1.5">
                  <span
                    v-for="qualification in qualificationsForUnit(unit)"
                    :key="qualificationId(qualification)"
                    class="rounded-full border border-slate-200 bg-white px-2 py-1 text-xs text-slate-500"
                  >
                    {{ qualificationLabel(qualification) }}
                  </span>
                </span>
                <label v-else-if="qualificationsForUnit(unit).length > 1" class="mt-3 block text-xs font-semibold text-slate-600">
                  {{ t.checkoutWizard.exemptionQualificationLabel }}
                  <select
                    :value="selectedQualificationIdForUnit(unit)"
                    :data-unit-id="unit.unit_id"
                    data-testid="stage-exemption-qualification-select"
                    class="mt-1 min-h-10 w-full rounded-lg border border-slate-300 bg-white px-3 text-sm font-normal text-slate-800"
                    :disabled="Boolean(applicationLoadingUnitId)"
                    @change="selectQualification(unit, $event)"
                  >
                    <option
                      v-for="qualification in qualificationsForUnit(unit)"
                      :key="qualificationId(qualification)"
                      :value="qualificationId(qualification)"
                    >
                      {{ qualificationLabel(qualification) }}
                    </option>
                  </select>
                </label>
                <span
                  v-if="qualificationStatusHint(unit)"
                  class="mt-3 block text-xs leading-5 text-slate-600"
                >
                  {{ qualificationStatusHint(unit) }}
                </span>
              </span>
            </div>
            <div class="mt-4 grid gap-2 sm:grid-cols-2" role="group" :aria-label="t.learning.stageExemptionDecisionLabel">
              <div class="flex min-w-0 flex-col gap-1.5">
                <span data-testid="stage-exemption-fee" class="min-h-5 text-center text-xs font-semibold text-emerald-700">
                  {{ unitPriceText(unit, 'exemption') }}
                </span>
                <button
                  type="button"
                  class="btn min-h-10 rounded-lg border px-3 text-sm"
                  :class="exemptionDecisions[String(unit.unit_id)] === 'exempt' ? 'border-emerald-600 bg-emerald-600 text-white' : 'border-slate-300 bg-white text-slate-700'"
                  :data-unit-id="unit.unit_id"
                  data-testid="stage-exemption-apply"
                  :disabled="!selectedQualificationIdForUnit(unit) || Boolean(applicationLoadingUnitId)"
                  @click="handleExemptionAction(unit)"
                >
                  <Loader2 v-if="applicationLoadingUnitId === String(unit.unit_id)" class="mr-2 h-4 w-4 animate-spin" />
                  {{ qualificationActionLabel(unit) }}
                </button>
              </div>
              <div class="flex min-w-0 flex-col gap-1.5">
                <span data-testid="stage-access-fee" class="min-h-5 text-center text-xs font-semibold text-slate-600">
                  {{ unitPriceText(unit, 'access') }}
                </span>
                <button
                  type="button"
                  class="btn min-h-10 rounded-lg border px-3 text-sm"
                  :class="exemptionDecisions[String(unit.unit_id)] === 'waive' ? 'border-blue-600 bg-blue-600 text-white' : 'border-slate-300 bg-white text-slate-700'"
                  :data-unit-id="unit.unit_id"
                  data-testid="stage-exemption-waive"
                  :disabled="selectedQualificationIsEligible(unit) || Boolean(applicationLoadingUnitId) || unitPaymentBlockedByApplication(unit)"
                  :title="selectedQualificationIsEligible(unit) || unitPaymentBlockedByApplication(unit) ? qualificationStatusHint(unit) : undefined"
                  @click="setUnitDecision(unit, 'waive')"
                >
                  {{ selectedQualificationIsEligible(unit)
                    ? t.learning.stageExemptionAutomatic
                    : t.learning.stageExemptionWaive }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="flex flex-wrap items-center justify-between gap-3 border-t border-slate-100 px-6 py-4">
        <span class="text-sm text-slate-500">
          {{ selectedCountText }}
        </span>
        <div class="flex flex-wrap justify-end gap-2">
          <button type="button" class="btn btn-outline rounded-lg" :disabled="submitting || Boolean(applicationLoadingUnitId)" @click="close">
            {{ t.common.cancel }}
          </button>
          <button
            type="button"
            class="btn btn-primary rounded-lg"
            :disabled="eligibilityLoading || submitting || Boolean(applicationLoadingUnitId) || undecidedCount > 0"
            @click="submit"
          >
            <Loader2 v-if="submitting" class="mr-2 h-4 w-4 animate-spin" />
            {{ submitting
              ? t.learning.stageExemptionSubmitting
              : t.learning.stageExemptionConfirm
            }}
          </button>
        </div>
      </div>
    </div>
  </div>

  <div
    v-if="applicationConfirmTarget"
    class="app-safe-area-overlay fixed inset-0 z-[60] flex items-center justify-center bg-slate-950/60 p-4 backdrop-blur-sm"
  >
    <section
      data-testid="stage-exemption-application-confirm"
      class="app-dialog-viewport flex max-h-[92vh] w-full max-w-xl flex-col overflow-hidden rounded-2xl bg-white shadow-2xl"
      role="dialog"
      aria-modal="true"
      aria-labelledby="stage-exemption-application-confirm-title"
    >
      <header class="flex items-start justify-between gap-4 border-b border-slate-100 px-6 py-5">
        <div class="flex min-w-0 items-start gap-3">
          <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-amber-50 text-amber-700">
            <CircleAlert class="h-5 w-5" />
          </div>
          <div>
            <h2 id="stage-exemption-application-confirm-title" class="text-lg font-bold text-slate-950">
              {{ t.checkoutWizard.qualificationOrderConfirmTitle }}
            </h2>
            <p class="mt-2 text-sm leading-6 text-slate-600">
              {{ t.checkoutWizard.qualificationOrderConfirmDescription }}
            </p>
          </div>
        </div>
        <button
          type="button"
          class="rounded-xl p-2 text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-700"
          :aria-label="t.paymentSession.close"
          :disabled="Boolean(applicationLoadingUnitId)"
          @click="closeApplicationConfirm"
        >
          <X class="h-5 w-5" />
        </button>
      </header>

      <div class="space-y-4 overflow-y-auto px-6 py-5">
        <div class="rounded-xl border border-slate-200 px-4 py-3">
          <div class="font-semibold text-slate-950">{{ applicationConfirmTarget.unitName }}</div>
          <div class="mt-1 text-sm text-slate-600">{{ applicationConfirmTarget.qualificationName }}</div>
        </div>
        <div class="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm leading-6 text-amber-900">
          <p class="font-semibold">{{ t.checkoutWizard.qualificationOrderConfirmWarningTitle }}</p>
          <p class="mt-1">{{ t.checkoutWizard.qualificationOrderConfirmWarning }}</p>
          <p class="mt-1">{{ t.checkoutWizard.qualificationOrderConfirmNextStep }}</p>
        </div>
      </div>

      <footer class="flex flex-wrap justify-end gap-2 border-t border-slate-100 px-6 py-4">
        <button
          type="button"
          class="btn btn-outline rounded-lg"
          :disabled="Boolean(applicationLoadingUnitId)"
          @click="closeApplicationConfirm"
        >
          {{ t.common.cancel }}
        </button>
        <button
          data-testid="stage-exemption-confirm-application"
          type="button"
          class="btn btn-primary rounded-lg"
          :disabled="Boolean(applicationLoadingUnitId)"
          @click="confirmQualificationApplication"
        >
          <Loader2 v-if="applicationLoadingUnitId" class="mr-2 h-4 w-4 animate-spin" />
          {{ t.checkoutWizard.qualificationOrderConfirmAction }}
        </button>
      </footer>
    </section>
  </div>

  <PaymentSessionDialog
    v-if="credentialPaymentSession"
    v-model:open="credentialPaymentOpen"
    :title="t.checkoutWizard.qualificationPaymentTitle"
    :subtitle="credentialPaymentSession.orderId"
    biz-type="CREDENTIAL_APPLICATION"
    :biz-ref-ulid="credentialPaymentSession.orderId"
    :order-id="credentialPaymentSession.orderId"
    source="credential_application"
    return-path="/credentials"
    :extra-return-params="{ qual_ulids: credentialPaymentSession.qualificationId }"
  />
</template>
