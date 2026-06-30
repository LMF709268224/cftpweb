<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { AlertCircle, Building2, Check, CreditCard, Lock, Loader2, ShoppingCart } from "lucide-vue-next"
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
  breakdown?: CouponDiscount[]
  invalid?: InvalidCoupon[]
  amount_label?: string
  amount?: string | number
  pay_amount_label?: string
  pay_amount?: string | number
}

type CouponDiscount = {
  code?: string
  name?: string
  type?: string
  percent_off?: number
  amount_off?: number
  discount?: number
  description?: string
}

type InvalidCoupon = {
  code?: string
  reason?: string
}

type ActiveOrder = {
  action: MallAction
  orderId: string
  status?: string
  payOrderId?: string
  canCancel?: boolean
  message?: string
}

type ActiveOrderPayload = {
  action?: MallAction
  order_id?: string
  orderId?: string
  status?: string
  pay_order_id?: string
  payOrderId?: string
  can_cancel?: boolean
  canCancel?: boolean
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
  bundleId?: string
  isPipelineBundle?: boolean
  isMembershipBundle?: boolean
  membershipId?: string
  membershipGpath?: string
  initialEligibility?: EligibilityPreview | null
  initialActiveOrder?: ActiveOrderPayload | null
  initialPaymentPreview?: PaymentPreview | null
  initialExemptionOptions?: ExemptionOptions | null
}>()

const emit = defineEmits<{ "update:open": [value: boolean]; cancelled: [] }>()
const { t } = useTranslation()
const paymentMethod = ref<PaymentMethod>("stripe")
const eligibilityLoading = ref(false)
const dialogStateLoading = ref(false)
const actionLoading = ref(false)
const paymentLoading = ref(false)
const cancelOrderLoading = ref(false)
const credentialApplicationLoadingKey = ref("")
const eligibility = ref<EligibilityPreview | null>(null)
const exemptionOptions = ref<ExemptionOptions | null>(null)
const activeOrder = ref<ActiveOrder | null>(null)
const paymentPreview = ref<PaymentPreview | null>(null)
const previewError = ref("")
const exemptionError = ref("")
const couponInput = ref("")
const appliedCouponCodes = ref<string[]>([])
const couponPreviewLoading = ref(false)
const couponError = ref("")
const selectedExemptionUnitIds = ref<Record<string, boolean>>({})
const resolvedBundleId = ref(props.bundleId || "")
const activePaymentSession = ref<{
  paymentKey?: string
  bizType: string
  bizRefUlid: string
  orderId: string
  source: string
  returnPath: string
  extraReturnParams?: Record<string, string>
  couponCodes?: string[]
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
const shouldUsePipelineEligibility = computed(() => Boolean(props.isPipelineBundle && props.pipelineId))
const canPurchase = computed(() => Boolean(eligibility.value?.can_purchase))
const canUnlock = computed(() => Boolean(shouldUsePipelineEligibility.value && eligibility.value?.can_unlock))
const cannotContinue = computed(() => Boolean(eligibility.value && !canPurchase.value && !canUnlock.value))
const hasInProgressOrder = computed(() => blockers.value.some((blocker) => blocker.blocker_type === "IN_PROGRESS_PURCHASE"))
const exemptionStages = computed(() => exemptionOptions.value?.stages?.filter((stage) => (stage.units?.length || 0) > 0) || [])
const hasExemptionOptions = computed(() => exemptionStages.value.length > 0)
const selectedExemptionCount = computed(() => Object.values(selectedExemptionUnitIds.value).filter(Boolean).length)
const isPreparingOrder = computed(() => Boolean(actionLoading.value && activeOrder.value && !paymentPreview.value && !activePaymentSession.value && !previewError.value))
const isOrderPreviewLoading = computed(() => Boolean(activeOrder.value && !paymentPreview.value && !previewError.value && !activePaymentSession.value))
const canCancelActiveOrder = computed(() => Boolean(activeOrder.value?.orderId && (activeOrder.value?.canCancel || isCancelableOrderStatus(activeOrder.value?.status))))
const pendingCredentialApplications = ref<Record<string, boolean>>({})
const hasPendingCredentialApplication = computed(() => Object.values(pendingCredentialApplications.value).some(Boolean))
const activeCouponCodes = computed(() => appliedCouponCodes.value.map((code) => code.trim()).filter(Boolean))


function normalizeInitialActiveOrder(order?: ActiveOrderPayload | null): ActiveOrder | null {
  const orderId = String(order?.order_id || order?.orderId || "").trim()
  if (!orderId) return null
  return {
    action: order?.action || "purchase",
    orderId,
    status: order?.status,
    payOrderId: order?.pay_order_id || order?.payOrderId,
    canCancel: Boolean(order?.can_cancel || order?.canCancel),
    message: order?.message || copy.value.inProgressPurchaseDesc,
  }
}

function hasInitialPurchaseState() {
  return Boolean(props.initialEligibility || props.initialActiveOrder || props.initialPaymentPreview || props.initialExemptionOptions)
}

function hasInitialActiveOrderState() {
  return Boolean(props.initialActiveOrder || props.initialPaymentPreview)
}

function hydrateFromInitialState() {
  eligibility.value = props.initialEligibility || { can_purchase: true, can_unlock: false, blockers: [] }
  activeOrder.value = normalizeInitialActiveOrder(props.initialActiveOrder)
  paymentPreview.value = props.initialPaymentPreview || null
  exemptionOptions.value = props.initialExemptionOptions || null
  previewError.value = ""
  exemptionError.value = ""
  selectedExemptionUnitIds.value = {}
  couponInput.value = ""
  appliedCouponCodes.value = []
  couponError.value = ""
  pruneSelectedExemptions(exemptionOptions.value)
}

function applyBundlePurchaseState(bundle: any) {
  eligibility.value = bundle?.purchase_state?.eligibility || bundle?.eligibility || { can_purchase: true, can_unlock: false, blockers: [] }
  activeOrder.value = normalizeInitialActiveOrder(bundle?.purchase_state?.active_order || bundle?.active_order)
  paymentPreview.value = bundle?.purchase_state?.payment_preview || bundle?.payment_preview || null
  exemptionOptions.value = bundle?.purchase_state?.exemption_options || bundle?.exemption_options || null
  previewError.value = ""
  exemptionError.value = ""
  activePaymentSession.value = null
  couponError.value = ""
  pruneSelectedExemptions(exemptionOptions.value)
}

async function loadBundlePurchaseState() {
  if (!resolvedBundleId.value) return false
  try {
    const bundle = await apiClient(`/api/mall/bundles/${encodeURIComponent(resolvedBundleId.value)}`)
    applyBundlePurchaseState(bundle)
    return true
  } catch (error) {
    console.error(error)
    return false
  }
}

async function resolveBundleFromCatalog() {
  if (resolvedBundleId.value) return true
  try {
    const res = await apiClient("/api/mall/bundles")
    const bundles = Array.isArray(res?.bundles) ? res.bundles : []
    const found = bundles.find((b: any) =>
      (props.pipelineId && b?.pipeline_id === props.pipelineId) ||
      (props.membershipId && b?.membership_id === props.membershipId) ||
      (props.membershipGpath && b?.membership_gpath === props.membershipGpath),
    )
    if (!found?.bundle_id) return false
    resolvedBundleId.value = found.bundle_id
    applyBundlePurchaseState(found)
    return true
  } catch (error) {
    console.error("Failed to resolve bundle for purchase dialog", error)
    return false
  }
}

async function loadFreshDialogState() {
  dialogStateLoading.value = !eligibility.value && !activeOrder.value && !paymentPreview.value
  if (!resolvedBundleId.value) {
    await resolveBundleFromCatalog()
  }
  try {
    if (await loadBundlePurchaseState()) {
      await refreshPendingCredentialApplications()
      return
    }
    if (hasInitialPurchaseState()) {
      hydrateFromInitialState()
      await refreshPendingCredentialApplications()
      return
    }
    await loadLegacyDialogState()
    await refreshPendingCredentialApplications()
  } finally {
    dialogStateLoading.value = false
  }
}

async function loadLegacyDialogState() {
  if (await resolveBundleFromCatalog()) return
  if (await loadBundlePurchaseState()) return
  eligibility.value = {
    can_purchase: false,
    can_unlock: false,
    blockers: [{ blocker_type: "BUNDLE_NOT_FOUND", description: "product bundle is unavailable" }],
  }
  activeOrder.value = null
  paymentPreview.value = null
  resetExemptionSelection()
}

watch(() => props.open, async (open) => {
  if (open) {
    resolvedBundleId.value = props.bundleId || ""
    if (hasInitialActiveOrderState()) {
      hydrateFromInitialState()
    } else {
      eligibility.value = null
      activeOrder.value = null
      paymentPreview.value = null
      resetExemptionSelection()
      couponInput.value = ""
      appliedCouponCodes.value = []
      couponError.value = ""
    }
    await loadFreshDialogState()
  } else {
    activePaymentSession.value = null
    paymentLoading.value = false
    cancelOrderLoading.value = false
  }
})

function close() {
  activePaymentSession.value = null
  credentialApplicationOrder.value = null
  paymentLoading.value = false
  cancelOrderLoading.value = false
  emit("update:open", false)
}

function normalizedStatus(status: unknown) {
  return String(status || "").trim().toUpperCase()
}

function normalizedOrderStatus(status: unknown) {
  const value = normalizedStatus(status)
  switch (value) {
    case "1":
      return "PENDING_CREATE"
    case "2":
      return "PENDING_PAYMENT"
    case "3":
      return "COMPLETED"
    case "4":
      return "CANCELLED"
    case "5":
      return "FAILED"
    default:
      return value
  }
}

function isCompletedStatus(status: unknown) {
  return normalizedOrderStatus(status).includes("COMPLETED")
}

function isFailedStatus(status: unknown) {
  const value = normalizedOrderStatus(status)
  return value.includes("FAILED") || value.includes("CANCEL") || value.includes("REJECT")
}

function isCancelableOrderStatus(status: unknown) {
  const value = normalizedOrderStatus(status)
  return value.includes("WAIT") || value.includes("PENDING")
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

function normalizeCouponCodes(codes: string[]) {
  return Array.from(new Set(codes.map((code) => String(code || "").trim()).filter(Boolean)))
}

function couponInputCodes() {
  return normalizeCouponCodes(couponInput.value.split(/[\s,，;；]+/))
}

function couponLabel(item: CouponDiscount) {
  return item.name || item.code || copy.value.couponApplied
}

function invalidCouponText(item: InvalidCoupon) {
  const code = item.code || copy.value.couponUnknown
  return item.reason ? `${code}: ${item.reason}` : code
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

function goToCredentialUpload() {
  window.location.assign("/credentials")
}

function qualificationActionDisabled(unit: ExemptionUnit, qual: ExemptionQual) {
  const qualId = String(qual.qual_id || "").trim()
  return Boolean(
    credentialApplicationLoadingKey.value === applicationLoadingKey(unit, qual) ||
    pendingCredentialApplications.value[qualId] ||
    hasPendingCredentialApplication.value,
  )
}

function qualificationActionLabel(qual: ExemptionQual) {
  const qualId = String(qual.qual_id || "").trim()
  if (pendingCredentialApplications.value[qualId]) return copy.value.qualificationUnderReview
  if (hasPendingCredentialApplication.value) return copy.value.qualificationApplicationBlocked
  return copy.value.goApplyQualification
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
  if (!shouldUsePipelineEligibility.value) {
    return JSON.stringify({})
  }

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

  return JSON.stringify({
    [props.pipelineId]: {
      stages
    }
  })
}

async function latestCredentialApplication(qualId: string) {
  try {
    const res = await apiClient(`/api/credentials/applications?cred_def_ulid=${encodeURIComponent(qualId)}`)
    return (res?.applications || [])[0] || null
  } catch (error) {
    console.error(error)
    return null
  }
}

async function refreshPendingCredentialApplications() {
  const qualIds = Array.from(new Set(
    exemptionStages.value
      .flatMap((stage) => stage.units || [])
      .flatMap((unit) => unit.exemption_quals || [])
      .map((qual) => String(qual.qual_id || "").trim())
      .filter(Boolean),
  ))
  const next: Record<string, boolean> = {}
  await Promise.all(qualIds.map(async (qualId) => {
    const app = await latestCredentialApplication(qualId)
    next[qualId] = Boolean(app?.status && isApplicationPendingStatus(app.status))
  }))
  pendingCredentialApplications.value = next
}

async function refreshEligibility() {
  eligibilityLoading.value = true
  try {
    if (!await loadBundlePurchaseState()) await loadLegacyDialogState()
    await refreshPendingCredentialApplications()
  } finally {
    eligibilityLoading.value = false
  }
}

async function refreshPaymentPreviewWithCoupons(codes = activeCouponCodes.value) {
  const orderId = activeOrder.value?.orderId
  if (!orderId || activeOrder.value?.action !== "purchase") return
  couponPreviewLoading.value = true
  couponError.value = ""
  try {
    const preview = await apiClient("/api/mall/payments/preview", {
      method: "POST",
      body: JSON.stringify({
        biz_type: "BUNDLE_PURCHASE",
        biz_ref_ulid: orderId,
        coupon_codes: normalizeCouponCodes(codes),
      }),
    })
    paymentPreview.value = preview
    previewError.value = ""
  } catch (error) {
    console.error(error)
    couponError.value = copy.value.couponPreviewFailed || copy.value.pricePreviewFailed || t.value.common.error
  } finally {
    couponPreviewLoading.value = false
  }
}

async function applyCouponCodes() {
  const nextCodes = couponInputCodes()
  appliedCouponCodes.value = nextCodes
  await refreshPaymentPreviewWithCoupons(nextCodes)
}

async function clearCouponCodes() {
  couponInput.value = ""
  appliedCouponCodes.value = []
  await refreshPaymentPreviewWithCoupons([])
}

async function cancelActiveOrder() {
  const orderId = activeOrder.value?.orderId
  if (!orderId || cancelOrderLoading.value) return
  const confirmed = window.confirm(copy.value.cancelOrderConfirm)
  if (!confirmed) return
  cancelOrderLoading.value = true
  try {
    const res = await apiClient(`/api/orders/${encodeURIComponent(orderId)}/cancel`, { method: "POST" })
    if (res?.success === false) {
      toast.error(copy.value.cancelOrderFailed)
      return
    }
    toast.success(copy.value.cancelOrderSuccess)
    emit("update:open", false)
    activeOrder.value = null
    paymentPreview.value = null
    previewError.value = ""
    activePaymentSession.value = null
    paymentLoading.value = false
    await loadBundlePurchaseState()
    emit("cancelled")
  } catch (error) {
    console.error(error)
    toast.error(copy.value.cancelOrderFailed)
  } finally {
    cancelOrderLoading.value = false
  }
}

async function createBundlePurchaseOrder(bundleOrderUlid = "") {
  const order = await apiClient(`/api/mall/bundles/${resolvedBundleId.value}/purchase`, {
    method: "POST",
    body: JSON.stringify({
      payment_mode: "FULL_PIPELINE",
      selected_exemptions_json: buildSelectedExemptionsJson(),
      bundle_order_ulid: bundleOrderUlid,
    }),
  })
  const orderId = String(order?.bundle_order_ulid || "").trim()
  const orderStatus = String(order?.order_status || "")
  activeOrder.value = {
    action: "purchase",
    orderId,
    status: orderStatus,
    payOrderId: order?.bundle_pay_order_ulid,
    canCancel: isCancelableOrderStatus(orderStatus),
    message: order?.message,
  }
  paymentPreview.value = null
  await loadBundlePurchaseState()
  return { orderId, orderStatus }
}

async function createPurchaseOrder() {
  actionLoading.value = true
  try {
    if (await loadBundlePurchaseState()) {
      if (activeOrder.value?.action === "purchase") return
      if (!eligibility.value?.can_purchase) return
    } else if (!eligibility.value?.can_purchase) {
      return
    }

    const { orderId, orderStatus } = await createBundlePurchaseOrder()
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
    if (orderId && !paymentPreview.value) previewError.value = copy.value.pricePreviewFailed || t.value.common.error
  } catch (error) {
    console.error(error)
  } finally {
    actionLoading.value = false
  }
}



async function createUnlockOrder() {
  actionLoading.value = true
  try {
    if (await loadBundlePurchaseState()) {
      if (!eligibility.value?.can_unlock) return
    } else if (!eligibility.value?.can_unlock) {
      return
    }

    const order = await apiClient(`/api/mall/bundles/${resolvedBundleId.value}/unlock`, {
      method: "POST",
      body: JSON.stringify({
        pipeline_cc_ulid: props.pipelineId
      })
    })
    const orderId = order.pipeline_unlock_order_ulid
    const orderStatus = order.order_status
    activeOrder.value = {
      action: "unlock",
      orderId,
      status: orderStatus,
      payOrderId: order.pay_order_ulid,
      canCancel: isCancelableOrderStatus(orderStatus),
      message: order.message,
    }
    if (isCompletedStatus(orderStatus)) {
      toast.success(copy.value.unlockCompleted)
      await refreshEligibility()
      return
    }
    if (isFailedStatus(orderStatus)) {
      toast.error(copy.value.unlockFailed)
      return
    }
    if (orderId) {
      paymentPreview.value = null
      previewError.value = ""
      // Re-initiate by business order so a reused unlock order never mounts an expired Stripe session.
      activePaymentSession.value = {
        bizType: "PIPELINE_UNLOCK",
        bizRefUlid: orderId,
        orderId,
        source: "unlock",
        returnPath: "/certifications",
        extraReturnParams: {
          pipeline_id: props.pipelineId,
          bundle_id: resolvedBundleId.value,
        },
      }
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
  if (!shouldUsePipelineEligibility.value || !props.pipelineId || !resolvedBundleId.value || !qualId) return
  const loadingKey = applicationLoadingKey(unit, qual)
  credentialApplicationLoadingKey.value = loadingKey
  activePaymentSession.value = null
  try {
      const existingApplication = await latestCredentialApplication(qualId)
    if (existingApplication?.status) {
      if (isApplicationPendingStatus(existingApplication.status)) {
        toast.info(copy.value.qualificationUnderReview)
        window.setTimeout(() => goToCredentialUpload(), 300)
        return
      }
      if (isApplicationApprovedStatus(existingApplication.status)) {
        toast.success(copy.value.qualificationAlreadyApproved)
        await refreshEligibility()
        return
      }
      if (isApplicationResubmitStatus(existingApplication.status)) {
        goToCredentialUpload()
        return
      }
    }

    const qualIds = [qualId]
    let order
    try {
      order = await apiClient("/api/credentials/application-orders", {
        method: "POST",
        suppressErrorToast: true,
        body: JSON.stringify({
          pipeline_cc_ulid: props.pipelineId,
          bundle_ulid: resolvedBundleId.value,
          qual_ulids: qualIds,
        }),
      })
    } catch (error) {
      const message = error instanceof Error ? error.message : ""
      if (message.includes("in-progress credential application") || message.includes("进行中") || message.includes("请先处理")) {
        toast.info(copy.value.qualificationUnderReview)
        window.setTimeout(() => goToCredentialUpload(), 300)
        return
      }
      throw error
    }
    const orderId = String(order?.application_order_ulid || "").trim()
    const orderStatus = String(order?.order_status || "")
    credentialApplicationOrder.value = {
      applicationOrderUlid: orderId,
      orderStatus,
      payOrderUlid: order?.pay_order_ulid,
      paymentKey: order?.payment_key,
      message: order?.message,
      qualIds,
    }

    if (isUploadReadyStatus(orderStatus)) {
      toast.info(copy.value.qualificationUploadReady)
      window.setTimeout(() => goToCredentialUpload(), 300)
      return
    }
    if (isCredentialApplicationUnderReviewStatus(orderStatus)) {
      toast.info(copy.value.qualificationUnderReview)
      window.setTimeout(() => goToCredentialUpload(), 300)
      return
    }
    if (isCredentialApplicationResolvedStatus(orderStatus)) {
      toast.info(copy.value.refreshEligibility)
      await refreshEligibility()
      return
    }
    if (isCredentialApplicationPaymentStatus(orderStatus) || order?.payment_key) {
      activePaymentSession.value = {
        bizType: "CREDENTIAL_APPLICATION",
        bizRefUlid: orderId,
        orderId,
        source: "credential_application",
        returnPath: "/credentials",
        extraReturnParams: { qual_ulids: qualIds.join(",") },
      }
      return
    }
    toast.info(copy.value.qualificationApplicationCreated)
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
    bundleId: resolvedBundleId.value,
  }))
}

function initiatePayment() {
  if (paymentLoading.value || activePaymentSession.value || !activeOrder.value?.orderId) return
  const bizType = activeOrder.value.action === "unlock" ? "PIPELINE_UNLOCK" : "BUNDLE_PURCHASE"
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
      extraReturnParams: {
        pipeline_id: props.pipelineId,
      bundle_id: resolvedBundleId.value,
    },
    couponCodes: activeOrder.value.action === "purchase" ? activeCouponCodes.value : [],
  }
    paymentLoading.value = false
  } catch (error) {
    console.error(error)
    paymentLoading.value = false
  }
}

async function handlePaymentSessionError() {
  paymentLoading.value = false
  activePaymentSession.value = null
  await refreshEligibility()
}
</script>

<template>
  <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="close">
    <div class="flex max-h-[86vh] w-full max-w-[620px] flex-col overflow-hidden rounded-xl bg-card shadow-2xl">
      <div class="shrink-0 border-b border-border px-6 pb-4 pt-6">
        <h2 class="text-xl font-semibold">{{ courseName }}</h2>
        <p v-if="description" class="mt-2 text-sm leading-6 text-muted-foreground">{{ description }}</p>
      </div>

      <div class="min-h-0 flex-1 space-y-5 overflow-y-auto px-6 py-5">


        <div v-if="dialogStateLoading || (eligibilityLoading && !eligibility)" class="rounded-lg border border-border bg-muted/30 p-4">
          <div class="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 class="h-4 w-4 animate-spin" />
            {{ copy.checking }}
          </div>
        </div>
        <div v-else-if="isPreparingOrder" class="rounded-lg border border-blue-200 bg-blue-50 p-4 text-blue-900">
          <div class="flex items-center gap-2 font-semibold">
            <Loader2 class="h-4 w-4 animate-spin" />
            {{ copy.preparingOrderTitle }}
          </div>
          <p class="mt-2 text-sm leading-6 text-blue-800">{{ copy.preparingOrderDesc }}</p>
        </div>
        <div v-else-if="cannotContinue && !hasInProgressOrder" class="rounded-lg border border-amber-200 bg-amber-50 p-4">
          <div class="flex items-center gap-2 font-semibold text-amber-900"><AlertCircle class="h-4 w-4" />{{ copy.blockedTitle }}</div>
          <p class="mt-2 text-sm text-amber-800">{{ copy.blockedDesc }}</p>
        </div>

        <div v-if="blockers.length > 0 && cannotContinue && !hasInProgressOrder" class="rounded-lg border border-amber-200 bg-amber-50/70 p-4">
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

        <div v-if="canPurchase && !activeOrder && (exemptionError || hasExemptionOptions)" class="rounded-lg border border-border bg-muted/20 p-4">
          <div class="mb-3 grid grid-cols-1 items-start gap-3 sm:grid-cols-[minmax(0,1fr)_auto]">
            <div class="max-w-[390px] min-w-0">
              <div class="text-sm font-semibold text-foreground">{{ copy.exemptionsTitle }}</div>
              <p class="mt-1 text-xs leading-5 text-muted-foreground">{{ copy.exemptionsDesc }}</p>
            </div>
            <span v-if="selectedExemptionCount > 0" class="badge shrink-0 whitespace-nowrap border-emerald-200 bg-emerald-50 px-3 text-xs text-emerald-700">
              {{ selectedExemptionCount }} {{ copy.selectedExemptions }}
            </span>
          </div>

          <div v-if="exemptionError" class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
            <div class="flex items-center gap-2 font-semibold"><AlertCircle class="h-4 w-4" />{{ copy.exemptionsLoadFailed }}</div>
            <p class="mt-2">{{ copy.exemptionsFallback }}</p>
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
                    'group flex gap-3 rounded-xl border p-3.5 transition-all duration-200',
                    selectedExemptionUnitIds[unit.unit_id]
                      ? 'border-emerald-300 bg-emerald-50/70 shadow-sm ring-1 ring-emerald-100'
                      : unit.qualified
                        ? 'cursor-pointer border-border bg-background hover:border-primary/30 hover:bg-primary/5'
                        : 'border-border bg-muted/30 opacity-75',
                  ]"
                >
                  <input
                    type="checkbox"
                    class="sr-only"
                    :checked="Boolean(selectedExemptionUnitIds[unit.unit_id])"
                    :disabled="!unit.qualified"
                    @change="onExemptionToggle(unit, $event)"
                  />
                  <span
                    :class="[
                      'mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full border transition-all duration-200',
                      selectedExemptionUnitIds[unit.unit_id]
                        ? 'border-emerald-500 bg-emerald-500 text-white shadow-sm shadow-emerald-200'
                        : unit.qualified
                          ? 'border-slate-300 bg-white text-transparent group-hover:border-primary/50'
                          : 'border-slate-200 bg-muted text-transparent',
                    ]"
                    aria-hidden="true"
                  >
                    <Check v-if="selectedExemptionUnitIds[unit.unit_id]" class="h-[18px] w-[18px] stroke-[3]" />
                    <span v-else class="h-2 w-2 rounded-full bg-current opacity-0" />
                  </span>
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
                        :disabled="qualificationActionDisabled(unit, qual)"
                        @click.prevent="createCredentialApplicationOrder(unit, qual)"
                      >
                        <Loader2 v-if="credentialApplicationLoadingKey === applicationLoadingKey(unit, qual)" class="h-3 w-3 animate-spin" />
                        {{ qualificationActionLabel(qual) }}
                      </button>
                    </div>
                  </div>
                </label>
              </div>
            </div>
          </div>
        </div>


        <div v-if="isOrderPreviewLoading" class="space-y-5">
          <div class="rounded-lg border border-border bg-muted/30 p-4">
            <div class="mb-3 text-sm font-semibold text-foreground">{{ copy.pricePreviewTitle }}</div>
            <div class="space-y-3 text-sm">
              <div class="flex items-center justify-between">
                <span class="text-muted-foreground">{{ copy.subtotal }}</span>
                <span class="h-4 w-24 animate-pulse rounded bg-muted-foreground/15" />
              </div>
              <div class="flex items-center justify-between">
                <span class="text-muted-foreground">{{ copy.discount }}</span>
                <span class="h-4 w-20 animate-pulse rounded bg-muted-foreground/15" />
              </div>
              <div class="flex items-center justify-between">
                <span class="text-muted-foreground">{{ copy.tax }}</span>
                <span class="h-4 w-20 animate-pulse rounded bg-muted-foreground/15" />
              </div>
              <div class="mt-2 flex items-center justify-between border-t border-border pt-3">
                <span class="font-semibold text-foreground">{{ copy.total }}</span>
                <span class="h-6 w-28 animate-pulse rounded bg-muted-foreground/20" />
              </div>
            </div>
          </div>

          <div class="space-y-3">
            <label class="text-sm font-medium text-foreground">{{ t.common.purchaseDialogPaymentMethod }}</label>
            <div class="space-y-2">
              <div class="flex w-full items-center gap-3 rounded-lg border border-border p-3">
                <span class="h-5 w-5 animate-pulse rounded-full bg-muted-foreground/15" />
                <CreditCard class="h-4 w-4 text-primary/60" />
                <span class="h-4 w-32 animate-pulse rounded bg-muted-foreground/15" />
                <span class="ml-auto h-5 w-20 animate-pulse rounded-full bg-muted-foreground/15" />
              </div>
              <div class="flex w-full items-center gap-3 rounded-lg border border-border p-3">
                <span class="h-5 w-5 animate-pulse rounded-full bg-muted-foreground/15" />
                <Building2 class="h-4 w-4 text-muted-foreground/70" />
                <span class="h-4 w-24 animate-pulse rounded bg-muted-foreground/15" />
              </div>
            </div>
            <div class="h-10 rounded-lg border border-amber-200 bg-amber-50/70" />
          </div>
        </div>

        <div v-if="paymentPreview" class="rounded-lg border border-border bg-muted/30 p-4 transition-opacity duration-200">
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
          <div v-if="paymentPreview.breakdown?.length" class="mt-3 space-y-2 rounded-lg border border-emerald-100 bg-emerald-50 p-3 text-xs text-emerald-900">
            <div class="font-semibold">{{ copy.couponApplied }}</div>
            <div v-for="item in paymentPreview.breakdown" :key="`${item.code}-${item.discount}`" class="flex items-start justify-between gap-3">
              <div>
                <div class="font-medium">{{ couponLabel(item) }}</div>
                <div v-if="item.code && item.code !== couponLabel(item)" class="text-emerald-700">{{ item.code }}</div>
              </div>
              <div class="shrink-0 font-semibold">-{{ formatMoney(item.discount || 0, paymentPreview.currency) }}</div>
            </div>
          </div>
          <div v-if="paymentPreview.invalid?.length" class="mt-3 rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-900">
            <div class="font-semibold">{{ copy.couponInvalidTitle }}</div>
            <div v-for="item in paymentPreview.invalid" :key="`${item.code}-${item.reason}`" class="mt-1">{{ invalidCouponText(item) }}</div>
          </div>
        </div>

        <div v-if="activeOrder?.action === 'purchase' && paymentPreview && !activePaymentSession" class="rounded-lg border border-border bg-background p-4">
          <label class="text-sm font-semibold text-foreground" for="purchase-coupon-input">{{ copy.couponTitle }}</label>
          <p class="mt-1 text-xs text-muted-foreground">{{ copy.couponHint }}</p>
          <div class="mt-3 flex flex-col gap-2 sm:flex-row">
            <input
              id="purchase-coupon-input"
              v-model="couponInput"
              class="input flex-1"
              :placeholder="copy.couponPlaceholder"
              :disabled="couponPreviewLoading || paymentLoading"
              @keydown.enter.prevent="applyCouponCodes"
            />
            <button type="button" class="btn btn-outline" :disabled="couponPreviewLoading || paymentLoading" @click="applyCouponCodes">
              <Loader2 v-if="couponPreviewLoading" class="h-4 w-4 animate-spin" />
              {{ copy.applyCoupon }}
            </button>
            <button v-if="activeCouponCodes.length" type="button" class="btn btn-outline" :disabled="couponPreviewLoading || paymentLoading" @click="clearCouponCodes">
              {{ copy.clearCoupon }}
            </button>
          </div>
          <div v-if="activeCouponCodes.length" class="mt-2 flex flex-wrap gap-2">
            <span v-for="code in activeCouponCodes" :key="code" class="rounded-full bg-primary/10 px-2 py-1 text-xs font-semibold text-primary">{{ code }}</span>
          </div>
          <p v-if="couponError" class="mt-2 text-xs text-red-600">{{ couponError }}</p>
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
            {{ t.paymentSession.testHint }}
          </div>
          <PaymentSessionPanel
            :payment-key="activePaymentSession.paymentKey"
            :biz-type="activePaymentSession.bizType"
            :biz-ref-ulid="activePaymentSession.bizRefUlid"
            :order-id="activePaymentSession.orderId"
            :source="activePaymentSession.source"
            :return-path="activePaymentSession.returnPath"
            :extra-return-params="activePaymentSession.extraReturnParams"
            :coupon-codes="activePaymentSession.couponCodes"
            min-height-class="min-h-[420px]"
            @error="handlePaymentSessionError"
          />
        </div>

        <div v-if="activeOrder && paymentPreview && !activePaymentSession" class="space-y-3 transition-opacity duration-200">
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
            {{ t.paymentSession.testHint }}
          </div>
        </div>
      </div>

      <div class="shrink-0 flex items-center justify-end gap-3 border-t border-border bg-muted/30 px-6 py-4">
        <button class="btn btn-outline" @click="close">{{ t.common.cancel }}</button>
        <button v-if="cannotContinue" class="btn btn-outline" :disabled="eligibilityLoading" @click="refreshEligibility">
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
        <button v-if="activeOrder && previewError" class="btn btn-outline" :disabled="actionLoading" @click="refreshEligibility">
          {{ copy.retryPreview }}
        </button>
        <button v-if="canCancelActiveOrder" class="btn border-red-600 bg-red-600 text-white shadow-sm shadow-red-100 hover:border-red-700 hover:bg-red-700 disabled:border-red-300 disabled:bg-red-300 disabled:text-white" :disabled="cancelOrderLoading || actionLoading" @click="cancelActiveOrder">
          <Loader2 v-if="cancelOrderLoading" class="h-4 w-4 animate-spin" />
          {{ copy.cancelOrder }}
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





