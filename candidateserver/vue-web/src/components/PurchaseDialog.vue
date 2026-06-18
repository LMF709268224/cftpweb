<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { AlertCircle, Building2, CheckCircle2, CreditCard, Lock, Loader2, ShoppingCart } from "lucide-vue-next"
import { timelineStatusLabelWithDiagnostics, timelineStatusBadgeClassForStatus } from "@/lib/status-labels"
import PaymentSessionPanel from "@/components/PaymentSessionPanel.vue"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"

type PaymentMethod = "stripe" | "bank"
type MallAction = "purchase" | "unlock"

type EligibilityBlocker = {
  blocker_type?: string
  description?: string
  details?: unknown[]
}

type EligibilityPreview = {
  eligible?: boolean
  can_purchase?: boolean
  can_unlock?: boolean
  blockers?: EligibilityBlocker[]
}

type PaymentPreview = {
  subtotal?: number
  discount_total?: number
  tax_total?: number
  total?: number
  currency?: string
  amount_label?: string
  amount?: string | number
  pay_amount_label?: string
  pay_amount?: string | number
}

type ActiveOrder = {
  action: MallAction
  orderId: string
  status?: string
  payOrderId?: string
  message?: string
}

type ExemptionQual = {
  qual_id: string
  name?: string
  description?: string
  category?: string
  eligible?: boolean
  credential_status?: string
  message?: string
}

type ExemptionUnit = {
  unit_id: string
  unit_name?: string
  allow_exemption?: boolean
  exemption_quals?: ExemptionQual[]
  qualified?: boolean
  message?: string
}

type ExemptionStage = {
  index: number
  stage_id: string
  stage_name?: string
  sort_order?: number
  units?: ExemptionUnit[]
}

type ExemptionOptions = {
  stages?: ExemptionStage[]
}

const props = defineProps<{
  open: boolean
  courseName: string
  description?: string
  pipelineId: string
}>()

const emit = defineEmits<{ "update:open": [value: boolean] }>()
const { t } = useTranslation()
const paymentMethod = ref<PaymentMethod>("stripe")
const eligibilityLoading = ref(false)
const actionLoading = ref(false)
const paymentLoading = ref(false)
const previewLoading = ref(false)
const exemptionLoading = ref(false)
const credentialApplicationLoadingKey = ref("")
const eligibility = ref<EligibilityPreview | null>(null)
const exemptionOptions = ref<ExemptionOptions | null>(null)
const activeOrder = ref<ActiveOrder | null>(null)
const paymentPreview = ref<PaymentPreview | null>(null)
const previewError = ref("")
const exemptionError = ref("")
const selectedExemptionUnitIds = ref<Record<string, boolean>>({})
const activePaymentSession = ref<{
  bizType: string
  bizRefUlid: string
  orderId: string
  source: string
  returnPath: string
  extraReturnParams?: Record<string, string>
} | null>(null)
const credentialApplicationOrder = ref<{
  applicationOrderUlid: string
  orderStatus?: string
  payOrderUlid?: string
  paymentKey?: string
  message?: string
  qualIds: string[]
} | null>(null)

const copy = computed(() => t.value.purchaseDialog || {})
const blockers = computed(() => eligibility.value?.blockers || [])
const canPurchase = computed(() => Boolean(eligibility.value?.can_purchase))
const canUnlock = computed(() => Boolean(eligibility.value?.can_unlock))
const cannotContinue = computed(() => Boolean(eligibility.value && !canPurchase.value && !canUnlock.value))
const hasInProgressOrder = computed(() => blockers.value.some((blocker) => blocker.blocker_type === "IN_PROGRESS_PURCHASE"))
const exemptionStages = computed(() => exemptionOptions.value?.stages?.filter((stage) => (stage.units?.length || 0) > 0) || [])
const hasExemptionOptions = computed(() => exemptionStages.value.length > 0)
const selectedExemptionCount = computed(() => Object.values(selectedExemptionUnitIds.value).filter(Boolean).length)

watch(() => props.open, (open) => {
  if (open && props.pipelineId) {
    void loadEligibility()
  } else {
    activePaymentSession.value = null
  }
})

function close() {
  activePaymentSession.value = null
  credentialApplicationOrder.value = null
  emit("update:open", false)
}

function normalizedStatus(status: unknown) {
  return String(status || "").trim().toUpperCase()
}

function isCompletedStatus(status: unknown) {
  return normalizedStatus(status).includes("COMPLETED")
}

function isFailedStatus(status: unknown) {
  const value = normalizedStatus(status)
  return value.includes("FAILED") || value.includes("CANCEL") || value.includes("REJECT")
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
  return normalizedStatus(status).includes("RESOLVED")
}

function isApplicationPendingStatus(status: unknown) {
  const value = normalizedStatus(status)
  return value === "PENDING" || value.includes("APPLICATION_STATUS_PENDING")
}

function isApplicationApprovedStatus(status: unknown) {
  const value = normalizedStatus(status)
  return value === "APPROVED" || value.includes("APPLICATION_STATUS_APPROVED")
}

function isApplicationResubmitStatus(status: unknown) {
  const value = normalizedStatus(status)
  return value.includes("REUPLOAD") || value.includes("RESUBMIT") || value.includes("NEEDS_RESUBMIT")
}

function formatMoney(amount?: number, currency = "usd") {
  if (typeof amount !== "number") return "-"
  return new Intl.NumberFormat(undefined, { style: "currency", currency: currency || "usd" }).format(amount / 100)
}

function detailText(detail: unknown) {
  if (typeof detail === "string") return detail
  if (detail && typeof detail === "object") {
    const record = detail as Record<string, unknown>
    return String(record.name || record.title || record.label || record.description || "")
  }
  return String(detail || "")
}

function blockerTitle(blocker: EligibilityBlocker) {
  if (blocker.blocker_type === "MISSING_UNLOCK_QUALIFICATION") return copy.value.missingQualification
  if (blocker.blocker_type === "ALREADY_PURCHASED") return copy.value.alreadyPurchased
  if (blocker.blocker_type === "IN_PROGRESS_PURCHASE") return copy.value.inProgressPurchase
  if (blocker.blocker_type === "PIPELINE_NOT_FOUND") return copy.value.pipelineNotFound
  return blocker.description || blocker.blocker_type || copy.value.unknownBlocker || t.value.common.unknown
}

function qualLabel(qual: ExemptionQual) {
  return qual.name || qual.qual_id || copy.value.unknownQualification || t.value.common.unknown
}

function applicationLoadingKey(unit: ExemptionUnit, qual: ExemptionQual) {
  return `${unit.unit_id || "unit"}:${qual.qual_id || "qual"}`
}

function resetExemptionSelection() {
  exemptionOptions.value = null
  exemptionError.value = ""
  selectedExemptionUnitIds.value = {}
}

function pruneSelectedExemptions(options: ExemptionOptions | null) {
  const allowed = new Set<string>()
  for (const stage of options?.stages || []) {
    for (const unit of stage.units || []) {
      if (unit.qualified && unit.unit_id) {
        allowed.add(unit.unit_id)
      }
    }
  }
  const next: Record<string, boolean> = {}
  for (const [unitId, selected] of Object.entries(selectedExemptionUnitIds.value)) {
    if (selected && allowed.has(unitId)) {
      next[unitId] = true
    }
  }
  selectedExemptionUnitIds.value = next
}

function onExemptionToggle(unit: ExemptionUnit, event: Event) {
  const input = event.target as HTMLInputElement | null
  if (!unit.qualified || !unit.unit_id) return
  selectedExemptionUnitIds.value = {
    ...selectedExemptionUnitIds.value,
    [unit.unit_id]: Boolean(input?.checked),
  }
}

function buildSelectedExemptionsJson() {
  const stages = exemptionStages.value
    .map((stage) => {
      const exemptedUnitIds = (stage.units || [])
        .filter((unit) => unit.qualified && unit.unit_id && selectedExemptionUnitIds.value[unit.unit_id])
        .map((unit) => unit.unit_id)
      return {
        index: stage.index,
        stage_cc_ulid: stage.stage_id,
        exempted_unit_cc_ulids: exemptedUnitIds,
      }
    })
    .filter((stage) => stage.exempted_unit_cc_ulids.length > 0)

  return JSON.stringify({ stages })
}

function orderIdFromDetail(order: any) {
  return order?.pipeline_order_ulid || order?.summary?.pipeline_order_ulid || ""
}

function orderStatusFromDetail(order: any) {
  return order?.order_status || order?.summary?.order_status || ""
}

function qualificationUploadUrl(qualId: string, orderId = "", appId = "") {
  const params = new URLSearchParams({ qual_ids: qualId })
  if (orderId) params.set("application_order_ulid", orderId)
  if (appId) params.set("application_id", appId)
  return `/credentials?${params.toString()}`
}

async function latestCredentialApplication(qualId: string) {
  try {
    const res = await apiClient(`/api/credentials/applications?cred_def_id=${encodeURIComponent(qualId)}`)
    return (res?.applications || [])[0] || null
  } catch (error) {
    console.error(error)
    return null
  }
}

async function hasCredentialUploadPermission(qualId: string) {
  try {
    const res = await apiClient(`/api/credentials/upload-permission?cred_def_id=${encodeURIComponent(qualId)}`)
    return Boolean(res?.granted)
  } catch (error) {
    console.error(error)
    return false
  }
}

async function loadEligibility() {
  eligibilityLoading.value = true
  activeOrder.value = null
  paymentPreview.value = null
  previewError.value = ""
  activePaymentSession.value = null
  try {
    const res: EligibilityPreview = await apiClient(`/api/mall/pipelines/${props.pipelineId}/eligibility`)
    eligibility.value = res
    if (res.can_purchase && !res.blockers?.some((blocker) => blocker.blocker_type === "IN_PROGRESS_PURCHASE")) {
      await loadExemptionOptions()
    } else {
      resetExemptionSelection()
    }
    if (res.blockers?.some((blocker) => blocker.blocker_type === "IN_PROGRESS_PURCHASE")) {
      await loadActiveOrder()
    }
  } finally {
    eligibilityLoading.value = false
  }
}

async function loadExemptionOptions() {
  exemptionLoading.value = true
  exemptionError.value = ""
  try {
    const res: ExemptionOptions = await apiClient(`/api/mall/pipelines/${props.pipelineId}/exemptions`)
    exemptionOptions.value = res
    pruneSelectedExemptions(res)
  } catch (error) {
    console.error(error)
    exemptionOptions.value = null
    exemptionError.value = copy.value.exemptionsLoadFailed || t.value.common.error
    selectedExemptionUnitIds.value = {}
  } finally {
    exemptionLoading.value = false
  }
}

async function previewPayment(action: MallAction, orderId: string) {
  previewError.value = ""
  previewLoading.value = true
  const bizType = action === "unlock" ? "PIPELINE_UNLOCK" : "PIPELINE_PAYMENT"
  try {
    paymentPreview.value = await apiClient("/api/mall/payments/preview", {
      method: "POST",
      body: JSON.stringify({ biz_type: bizType, biz_ref_ulid: orderId, coupon_codes: [] }),
    })
  } catch {
    paymentPreview.value = null
    previewError.value = copy.value.pricePreviewFailed || t.value.common.error
  } finally {
    previewLoading.value = false
  }
}

async function loadActiveOrder() {
  previewError.value = ""
  paymentPreview.value = null
  try {
    const order = await apiClient(`/api/mall/pipelines/${props.pipelineId}/active-order`)
    const orderId = orderIdFromDetail(order)
    if (!orderId) return
    activeOrder.value = {
      action: "purchase",
      orderId,
      status: orderStatusFromDetail(order),
      payOrderId: order.pipeline_pay_order_ulid,
      message: copy.value.inProgressPurchaseDesc,
    }
    await previewPayment("purchase", orderId)
  } catch (error) {
    console.error(error)
  }
}

async function createPurchaseOrder() {
  actionLoading.value = true
  try {
    const latest: EligibilityPreview = await apiClient(`/api/mall/pipelines/${props.pipelineId}/eligibility`)
    eligibility.value = latest
    if (!latest.can_purchase) return

    const order = await apiClient(`/api/mall/pipelines/${props.pipelineId}/purchase`, {
      method: "POST",
      body: JSON.stringify({
        payment_mode: "FULL_PIPELINE",
        candidate_selected_exemptions_json: buildSelectedExemptionsJson(),
      }),
    })
    const orderId = order.pipeline_order_ulid
    const orderStatus = order.order_status
    activeOrder.value = {
      action: "purchase",
      orderId,
      status: orderStatus,
      payOrderId: order.pipeline_pay_order_ulid,
      message: order.message,
    }
    if (isCompletedStatus(orderStatus)) {
      toast.success(copy.value.purchaseCompleted)
      close()
      window.setTimeout(() => window.location.reload(), 800)
      return
    }
    if (isFailedStatus(orderStatus)) {
      toast.error(copy.value.purchaseFailed)
      return
    }
    if (orderId) await previewPayment("purchase", orderId)
  } catch (error) {
    console.error(error)
  } finally {
    actionLoading.value = false
  }
}

async function createUnlockOrder() {
  actionLoading.value = true
  try {
    const latest: EligibilityPreview = await apiClient(`/api/mall/pipelines/${props.pipelineId}/eligibility`)
    eligibility.value = latest
    if (!latest.can_unlock) return

    const order = await apiClient(`/api/mall/pipelines/${props.pipelineId}/unlock`, { method: "POST" })
    const orderId = order.pipeline_unlock_order_ulid
    const paymentKey = order.payment_key
    const orderStatus = order.order_status
    activeOrder.value = {
      action: "unlock",
      orderId,
      status: orderStatus,
      payOrderId: order.pay_order_ulid,
      message: order.message,
    }
    if (isCompletedStatus(orderStatus)) {
      toast.success(copy.value.unlockCompleted)
      await loadEligibility()
      return
    }
    if (isFailedStatus(orderStatus)) {
      toast.error(copy.value.unlockFailed)
      return
    }
    if (orderId && (paymentKey || order.pay_order_ulid || normalizedStatus(orderStatus).includes("PAYMENT"))) {
      await previewPayment("unlock", orderId)
    } else {
      toast.info(copy.value.refreshEligibility)
    }
  } catch (error) {
    console.error(error)
  } finally {
    actionLoading.value = false
  }
}

async function createCredentialApplicationOrder(unit: ExemptionUnit, qual: ExemptionQual) {
  const qualId = String(qual.qual_id || "").trim()
  if (!props.pipelineId || !qualId) return
  const loadingKey = applicationLoadingKey(unit, qual)
  credentialApplicationLoadingKey.value = loadingKey
  try {
    const existingApplication = await latestCredentialApplication(qualId)
    if (existingApplication?.status) {
      if (isApplicationPendingStatus(existingApplication.status)) {
        toast.info(copy.value.qualificationUnderReview || "资格申请已提交，请等待审核结果。")
        return
      }
      if (isApplicationApprovedStatus(existingApplication.status)) {
        toast.success(copy.value.qualificationAlreadyApproved || "资格已审核通过，正在重新检查免考资格。")
        await loadExemptionOptions()
        return
      }
      if (isApplicationResubmitStatus(existingApplication.status)) {
        window.location.assign(qualificationUploadUrl(qualId, "", existingApplication.app_id || ""))
        return
      }
    }

    const order = await apiClient("/api/credentials/application-orders", {
      method: "POST",
      body: JSON.stringify({
        pipeline_cc_ulid: props.pipelineId,
        qual_ids: [qualId],
      }),
    })
    const orderId = String(order?.application_order_ulid || "").trim()
    const orderStatus = String(order?.order_status || "")
    credentialApplicationOrder.value = {
      applicationOrderUlid: orderId,
      orderStatus,
      payOrderUlid: order?.pay_order_ulid,
      paymentKey: order?.payment_key,
      message: order?.message,
      qualIds: [qualId],
    }

    if (isCredentialApplicationPaymentStatus(orderStatus) || order?.payment_key || order?.pay_order_ulid) {
      activePaymentSession.value = {
        bizType: "CREDENTIAL_APPLICATION",
        bizRefUlid: orderId,
        orderId,
        source: "credential_application",
        returnPath: "/credentials",
        extraReturnParams: { qual_ids: qualId, application_order_ulid: orderId },
      }
      return
    }
    if (isUploadReadyStatus(orderStatus)) {
      if (await hasCredentialUploadPermission(qualId)) {
        window.location.assign(qualificationUploadUrl(qualId, orderId))
        return
      }
      toast.info(copy.value.qualificationUploadNotReady || "资格申请订单已存在，但当前资格还没有开放资料上传，请先完成审核费支付或稍后重试。")
      return
    }
    if (isCredentialApplicationUnderReviewStatus(orderStatus)) {
      toast.info(copy.value.qualificationUnderReview || "资格申请已提交，请等待审核结果。")
      return
    }
    if (isCredentialApplicationResolvedStatus(orderStatus)) {
      toast.info(order?.message || copy.value.refreshEligibility)
      await loadExemptionOptions()
      return
    }
    toast.info(order?.message || copy.value.qualificationApplicationCreated)
  } catch (error) {
    console.error(error)
  } finally {
    credentialApplicationLoadingKey.value = ""
  }
}

function rememberPendingMallPayment() {
  if (!activeOrder.value?.orderId) return
  localStorage.setItem("pending_mall_payment", JSON.stringify({
    action: activeOrder.value.action,
    orderId: activeOrder.value.orderId,
    pipelineId: props.pipelineId,
  }))
}

async function initiatePayment() {
  if (!activeOrder.value?.orderId) return
  const bizType = activeOrder.value.action === "unlock" ? "PIPELINE_UNLOCK" : "PIPELINE_PAYMENT"
  if (paymentMethod.value !== "stripe") {
    toast.error(copy.value.unsupportedPaymentKey)
    return
  }
  paymentLoading.value = true
  try {
    rememberPendingMallPayment()
    activePaymentSession.value = {
      bizType,
      bizRefUlid: activeOrder.value.orderId,
      orderId: activeOrder.value.orderId,
      source: activeOrder.value.action,
      returnPath: "/certifications",
      extraReturnParams: { pipeline_id: props.pipelineId },
    }
  } catch (error) {
    console.error(error)
  } finally {
    paymentLoading.value = false
  }
}
</script>

<template>
  <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="close">
    <div class="flex max-h-[86vh] w-full max-w-[620px] flex-col overflow-hidden rounded-xl bg-card shadow-2xl">
      <div class="shrink-0 border-b border-border px-6 pb-4 pt-6">
        <h2 class="text-xl font-semibold">{{ copy.title }}: {{ courseName }}</h2>
        <p v-if="description" class="mt-2 text-sm leading-6 text-muted-foreground">{{ description }}</p>
      </div>

      <div class="min-h-0 flex-1 space-y-5 overflow-y-auto px-6 py-5">
        <div class="flex items-center justify-between border-b border-border/50 py-2">
          <span class="text-sm text-muted-foreground">{{ t.common.purchaseDialogCourse }}</span>
          <span class="text-sm font-medium text-foreground">{{ courseName }}</span>
        </div>

        <div v-if="eligibilityLoading && !eligibility" class="rounded-lg border border-border bg-muted/30 p-4">
          <div class="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 class="h-4 w-4 animate-spin" />
            {{ copy.checking }}
          </div>
        </div>
        <div v-else-if="canPurchase" class="rounded-lg border border-emerald-200 bg-emerald-50 p-4">
          <div class="flex items-center gap-2 font-semibold text-emerald-900"><CheckCircle2 class="h-4 w-4" />{{ copy.canPurchaseTitle }}</div>
          <p class="mt-2 text-sm text-emerald-800">{{ copy.canPurchaseDesc }}</p>
        </div>
        <div v-else-if="canUnlock" class="rounded-lg border border-blue-200 bg-blue-50 p-4">
          <div class="flex items-center gap-2 font-semibold text-blue-900"><Lock class="h-4 w-4" />{{ copy.canUnlockTitle }}</div>
          <p class="mt-2 text-sm text-blue-800">{{ copy.canUnlockDesc }}</p>
        </div>
        <div v-else-if="cannotContinue && hasInProgressOrder" class="rounded-lg border border-blue-200 bg-blue-50 p-4">
          <div class="flex items-center gap-2 font-semibold text-blue-900"><CreditCard class="h-4 w-4" />{{ copy.inProgressPurchase }}</div>
          <p class="mt-2 text-sm text-blue-800">{{ copy.inProgressPurchaseDesc }}</p>
        </div>
        <div v-else-if="cannotContinue" class="rounded-lg border border-amber-200 bg-amber-50 p-4">
          <div class="flex items-center gap-2 font-semibold text-amber-900"><AlertCircle class="h-4 w-4" />{{ copy.blockedTitle }}</div>
          <p class="mt-2 text-sm text-amber-800">{{ copy.blockedDesc }}</p>
        </div>

        <div v-if="blockers.length > 0" class="rounded-lg border border-amber-200 bg-amber-50/70 p-4">
          <div class="mb-3 text-sm font-semibold text-amber-950">{{ copy.blockersTitle }}</div>
          <ul class="space-y-2">
            <li v-for="(blocker, index) in blockers" :key="`${blocker.blocker_type || 'blocker'}-${index}`" class="rounded-lg border border-amber-200 bg-white/80 p-3">
              <div class="font-medium text-amber-950">{{ blockerTitle(blocker) }}</div>
              <div v-if="Array.isArray(blocker.details) && blocker.details.map(detailText).filter(Boolean).length > 0" class="mt-2">
                <div class="mb-1 text-xs font-medium text-muted-foreground">{{ copy.requiredItems }}</div>
                <ul class="space-y-1">
                  <li v-for="(detail, detailIndex) in blocker.details.map(detailText).filter(Boolean)" :key="`${detail}-${detailIndex}`" class="flex items-center gap-2 rounded-md bg-amber-100/70 px-2 py-1.5 text-sm font-medium text-amber-950">
                    <AlertCircle class="h-3.5 w-3.5 shrink-0 text-amber-600" />
                    <span>{{ detail }}</span>
                  </li>
                </ul>
              </div>
            </li>
          </ul>
        </div>

        <div v-if="canPurchase && !activeOrder" class="rounded-lg border border-border bg-muted/20 p-4">
          <div class="mb-3 flex items-start justify-between gap-3">
            <div>
              <div class="text-sm font-semibold text-foreground">{{ copy.exemptionsTitle }}</div>
              <p class="mt-1 text-xs leading-5 text-muted-foreground">{{ copy.exemptionsDesc }}</p>
            </div>
            <span v-if="selectedExemptionCount > 0" class="badge border-emerald-200 bg-emerald-50 text-xs text-emerald-700">
              {{ selectedExemptionCount }} {{ copy.selectedExemptions }}
            </span>
          </div>

          <div v-if="exemptionLoading" class="flex items-center gap-2 rounded-lg bg-background/70 p-3 text-sm text-muted-foreground">
            <Loader2 class="h-4 w-4 animate-spin text-primary" />
            {{ copy.exemptionsChecking }}
          </div>
          <div v-else-if="exemptionError" class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
            <div class="flex items-center gap-2 font-semibold"><AlertCircle class="h-4 w-4" />{{ copy.exemptionsLoadFailed }}</div>
            <p class="mt-2">{{ copy.exemptionsFallback }}</p>
          </div>
          <div v-else-if="!hasExemptionOptions" class="rounded-lg bg-background/70 p-3 text-sm text-muted-foreground">
            {{ copy.noExemptions }}
          </div>
          <div v-else class="space-y-3">
            <div v-for="stage in exemptionStages" :key="stage.stage_id || stage.index" class="rounded-lg border border-border bg-background p-3">
              <div class="mb-3 flex items-center justify-between gap-3">
                <div class="text-sm font-semibold text-foreground">{{ stage.stage_name || `${copy.stageLabel} ${stage.index + 1}` }}</div>
                <span class="badge text-xs">{{ stage.units?.length || 0 }} {{ copy.exemptionUnits }}</span>
              </div>
              <div class="space-y-2">
                <label
                  v-for="unit in stage.units"
                  :key="unit.unit_id"
                  :class="[
                    'flex gap-3 rounded-lg border p-3 transition-colors',
                    unit.qualified ? 'cursor-pointer border-emerald-200 bg-emerald-50/40 hover:border-emerald-300' : 'border-border bg-muted/30 opacity-75',
                  ]"
                >
                  <input
                    type="checkbox"
                    class="mt-1 h-4 w-4 rounded border-border text-primary"
                    :checked="Boolean(selectedExemptionUnitIds[unit.unit_id])"
                    :disabled="!unit.qualified"
                    @change="onExemptionToggle(unit, $event)"
                  />
                  <div class="min-w-0 flex-1">
                    <div class="flex flex-wrap items-center gap-2">
                      <span class="text-sm font-semibold text-foreground">{{ unit.unit_name || unit.unit_id }}</span>
                      <span v-if="unit.qualified" class="badge border-emerald-200 bg-emerald-50 text-xs text-emerald-700">{{ copy.exemptionEligible }}</span>
                      <span v-else class="badge border-amber-200 bg-amber-50 text-xs text-amber-700">{{ copy.exemptionMissing }}</span>
                    </div>
                    <div class="mt-2 flex flex-wrap gap-1.5">
                      <span
                        v-for="qual in unit.exemption_quals || []"
                        :key="qual.qual_id"
                        :class="[
                          'rounded-full border px-2 py-1 text-xs',
                          qual.eligible ? 'border-emerald-200 bg-emerald-50 text-emerald-700' : 'border-border bg-muted text-muted-foreground',
                        ]"
                      >
                        {{ qualLabel(qual) }}
                      </span>
                    </div>
                    <div v-if="!unit.qualified" class="mt-2 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                      <span>{{ copy.exemptionMissingHint }}</span>
                      <button
                        v-for="qual in unit.exemption_quals || []"
                        :key="`apply-${unit.unit_id}-${qual.qual_id}`"
                        type="button"
                        class="inline-flex items-center gap-1 rounded-md px-1.5 py-1 font-semibold text-primary hover:bg-primary/10 disabled:cursor-not-allowed disabled:opacity-60"
                        :disabled="credentialApplicationLoadingKey === applicationLoadingKey(unit, qual)"
                        @click.prevent="createCredentialApplicationOrder(unit, qual)"
                      >
                        <Loader2 v-if="credentialApplicationLoadingKey === applicationLoadingKey(unit, qual)" class="h-3 w-3 animate-spin" />
                        {{ copy.goApplyQualification }}
                      </button>
                    </div>
                  </div>
                </label>
              </div>
            </div>
          </div>
        </div>

        <div v-if="activeOrder" class="rounded-lg border border-border bg-muted/30 p-4">
          <div class="mb-2 flex items-center justify-between gap-3">
            <div class="text-sm font-semibold text-foreground">{{ activeOrder.message === copy.inProgressPurchaseDesc ? copy.activeOrder : copy.orderCreated }}</div>
            <span v-if="activeOrder.status" class="badge text-xs" :class="timelineStatusBadgeClassForStatus('MALL_ORDER', activeOrder.status)">
              {{ timelineStatusLabelWithDiagnostics(t, 'MALL_ORDER', activeOrder.status) }}
            </span>
          </div>
          <div class="break-all text-xs text-muted-foreground">{{ activeOrder.orderId }}</div>
          <p v-if="activeOrder.message" class="mt-2 text-sm text-muted-foreground">{{ activeOrder.message }}</p>
        </div>

        <div v-if="previewLoading && activeOrder && !paymentPreview" class="rounded-lg border border-border bg-muted/30 p-4">
          <div class="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 class="h-4 w-4 animate-spin text-primary" />
            <span>{{ copy.pricePreviewTitle || t.common.loading }}</span>
          </div>
        </div>

        <div v-if="paymentPreview" class="rounded-lg border border-border bg-muted/30 p-4">
          <div class="mb-3 text-sm font-semibold text-foreground">{{ copy.pricePreviewTitle }}</div>
          <div class="space-y-2 text-sm">
            <div class="flex justify-between">
              <span class="text-muted-foreground">{{ copy.subtotal }}</span>
              <span class="font-medium">{{ paymentPreview.amount_label || formatMoney(paymentPreview.subtotal, paymentPreview.currency) }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-muted-foreground">{{ copy.discount }}</span>
              <span class="font-medium">-{{ formatMoney(paymentPreview.discount_total || 0, paymentPreview.currency) }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-muted-foreground">{{ copy.tax }}</span>
              <span class="font-medium">{{ formatMoney(paymentPreview.tax_total || 0, paymentPreview.currency) }}</span>
            </div>
            <div class="mt-2 flex justify-between border-t border-border pt-2">
              <span class="font-semibold text-foreground">{{ copy.total }}</span>
              <span class="text-lg font-bold text-foreground">{{ paymentPreview.pay_amount_label || formatMoney(paymentPreview.total, paymentPreview.currency) }}</span>
            </div>
          </div>
        </div>

        <div v-if="activeOrder && previewError" class="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
          <div class="flex items-center gap-2 font-semibold"><AlertCircle class="h-4 w-4" />{{ copy.pricePreviewTitle }}</div>
          <p class="mt-2">{{ previewError }}</p>
        </div>

        <div v-if="activePaymentSession" class="space-y-3">
          <div class="rounded-lg border border-blue-200 bg-blue-50 p-4 text-sm text-blue-900">
            <div class="flex items-center gap-2 font-semibold"><CreditCard class="h-4 w-4" />{{ credentialApplicationOrder ? copy.qualificationPaymentTitle : copy.embeddedCheckoutTitle }}</div>
            <p class="mt-2">{{ credentialApplicationOrder ? copy.qualificationPaymentDesc : copy.embeddedCheckoutDesc }}</p>
          </div>
          <div class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-800">
            <strong>⚠️ 测试环境提示：</strong> 当前为测试环境，请使用通用测试信用卡号 <code>4242 4242 4242 4242</code>，任意有效日期和CVV进行体验。
          </div>
          <PaymentSessionPanel
            :biz-type="activePaymentSession.bizType"
            :biz-ref-ulid="activePaymentSession.bizRefUlid"
            :order-id="activePaymentSession.orderId"
            :source="activePaymentSession.source"
            :return-path="activePaymentSession.returnPath"
            :extra-return-params="activePaymentSession.extraReturnParams"
            min-height-class="min-h-[420px]"
          />
        </div>

        <div v-if="activeOrder && paymentPreview && !activePaymentSession" class="space-y-3">
          <label class="text-sm font-medium text-foreground">{{ t.common.purchaseDialogPaymentMethod }}</label>
          <div class="space-y-2">
            <button
              type="button"
              :class="[
                'flex w-full items-center gap-3 rounded-lg border p-3 transition-all',
                paymentMethod === 'stripe' ? 'border-primary bg-primary/5' : 'border-border hover:border-primary/50',
              ]"
              @click="paymentMethod = 'stripe'"
            >
              <div :class="['flex h-5 w-5 items-center justify-center rounded-full border-2 transition-colors', paymentMethod === 'stripe' ? 'border-primary' : 'border-muted-foreground/30']">
                <div v-if="paymentMethod === 'stripe'" class="h-2.5 w-2.5 rounded-full bg-primary" />
              </div>
              <CreditCard class="h-4 w-4 text-primary" />
              <span class="text-sm font-medium text-foreground">{{ copy.stripe }}</span>
              <span class="badge ml-auto border-0 bg-amber-500/10 text-xs text-amber-700">{{ t.common.purchaseDialogStripeBadge }}</span>
            </button>
            <button
              type="button"
              :class="[
                'flex w-full items-center gap-3 rounded-lg border p-3 transition-all',
                paymentMethod === 'bank' ? 'border-primary bg-primary/5' : 'border-border hover:border-primary/50',
              ]"
              @click="paymentMethod = 'bank'"
            >
              <div :class="['flex h-5 w-5 items-center justify-center rounded-full border-2 transition-colors', paymentMethod === 'bank' ? 'border-primary' : 'border-muted-foreground/30']">
                <div v-if="paymentMethod === 'bank'" class="h-2.5 w-2.5 rounded-full bg-primary" />
              </div>
              <Building2 class="h-4 w-4 text-muted-foreground" />
              <span class="text-sm font-medium text-foreground">{{ copy.bank }}</span>
            </button>
          </div>
          <div v-if="paymentMethod === 'stripe'" class="mt-4 rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-800">
            <strong>⚠️ 测试环境提示：</strong> 当前为测试环境，请使用通用测试信用卡号 <code>4242 4242 4242 4242</code>，任意有效日期和CVV进行体验。
          </div>
        </div>
      </div>

      <div class="shrink-0 flex items-center justify-end gap-3 border-t border-border bg-muted/30 px-6 py-4">
        <button class="btn btn-outline" @click="close">{{ t.common.cancel }}</button>
        <button v-if="cannotContinue" class="btn btn-outline" :disabled="eligibilityLoading" @click="loadEligibility">
          <Loader2 v-if="eligibilityLoading" class="h-4 w-4 animate-spin" />
          {{ copy.refreshEligibility }}
        </button>
        <button v-if="canUnlock && !activeOrder" class="btn btn-primary" :disabled="actionLoading" @click="createUnlockOrder">
          <Loader2 v-if="actionLoading" class="h-4 w-4 animate-spin" />
          <Lock v-else class="h-4 w-4" />
          {{ copy.createUnlockOrder }}
        </button>
        <button v-if="canPurchase && !activeOrder" class="btn btn-primary" :disabled="actionLoading" @click="createPurchaseOrder">
          <Loader2 v-if="actionLoading" class="h-4 w-4 animate-spin" />
          <ShoppingCart v-else class="h-4 w-4" />
          {{ copy.createPurchaseOrder }}
        </button>
        <button v-if="activeOrder && previewError" class="btn btn-outline" :disabled="actionLoading" @click="previewPayment(activeOrder.action, activeOrder.orderId)">
          {{ copy.retryPreview }}
        </button>
        <button v-if="activeOrder && paymentPreview && !activePaymentSession" class="btn btn-primary" :disabled="paymentLoading" @click="initiatePayment">
          <Loader2 v-if="paymentLoading" class="h-4 w-4 animate-spin" />
          <CreditCard v-else class="h-4 w-4" />
          {{ copy.payNow }}
        </button>
      </div>
    </div>
  </div>
</template>
