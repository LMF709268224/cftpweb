<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue"
import { useRoute } from "vue-router"
import { toast } from "vue-sonner"
import { AlertCircle, ChevronRight, CreditCard, Loader2, Package, Receipt, RefreshCw, X } from "lucide-vue-next"
import { timelineStatusBadgeClassForStatus, timelineStatusLabelWithDiagnostics } from "@/lib/status-labels"
import AppShell from "@/components/AppShell.vue"
import AppPagination from "@/components/AppPagination.vue"
import PaymentSessionDialog from "@/components/PaymentSessionDialog.vue"
import { apiClient } from "@/lib/apiClient"
import { useBodyScrollLock } from "@/lib/bodyScrollLock"
import { useDialogAccessibility } from "@/lib/dialogAccessibility"
import { formatBackendDateMinute } from "@/lib/utils"
import { useTranslation } from "@/lib/language"

type OrderStatus = keyof typeof statusConfig

type OrderItem = {
  id: string
  invoiceOrderId: string
  canViewInvoice: boolean
  items: string[]
  date: string
  amount: string
  currency: string
  bizType: string
  bizRefUlid: string
  status: OrderStatus
  order_status: string
  payment_status?: string
  pipelineId: string
  paymentMethod: string
}

type DetailField = {
  label: string
  value: string
}

type PriceMetric = {
  key: string
  label: string
  value: string
  note: string
  tone: string
}

type OrderPriceItem = {
  item_type?: string
  item_ulid?: string
  title?: string
  unit_price_minor?: number
  quantity?: number
  subtotal_minor?: number
}

type OrderExemption = {
  course_cc_ulid?: string
  credential_ulid?: string
}

type OrderDetail = {
  found?: boolean
  summary?: {
    order_id?: string
    candidate_id?: string
    biz_type?: string
    biz_ref_ulid?: string
    currency?: string
    amount?: number
    amount_minor?: number
    order_status?: string
    payment_status?: string
    created_at?: string
    meta?: {
      product_name?: string
    }
  }
  gpay_order_ulid?: string
  has_payment_key?: boolean
  paid_at?: string
  closed_at?: string
  last_reconciled_at?: string
  version?: number
  updated_at?: string
  order_status_at?: string
  payment_status_at?: string
  pricing?: {
    available?: boolean
    source?: string
    currency_code?: string
    billable_subtotal_minor?: number
    exemption_discount_minor?: number
    promotion_discount_minor?: number
    tax_minor?: number
    total_minor?: number
    amount_paid_minor?: number
    exemption_amount_recorded?: boolean
    items?: OrderPriceItem[]
    coupons?: Array<{ code?: string; name?: string }>
    promo_codes?: string[]
    unavailable_reason?: string
  }
  exemptions?: OrderExemption[]
  business_detail?: Record<string, unknown>
  raw?: unknown
}

const statusConfig = {
  completed: { labelKey: "statusCompleted", statusValue: "SUCCESS" },
  pending: { labelKey: "statusPending", statusValue: "PENDING" },
  processing: { labelKey: "statusProcessing", statusValue: "PROCESSING" },
  cancelled: { labelKey: "statusCancelled", statusValue: "CANCEL" },
} as const

const { t } = useTranslation()

const orders = ref<OrderItem[]>([])
const loading = ref(true)
const loadError = ref(false)
const page = ref(1)
const lastPage = ref(1)
const pageSize = ref(10)
const pageSizeOptions = [10, 30, 50, 100]
const lastPageSize = ref(10)
const totalOrders = ref(0)
const totalPages = ref(0)
const totalLabel = ref("")
const currentCursor = ref("")
const nextCursor = ref("")
const prevCursor = ref("")
const hasMore = ref(false)
const route = useRoute()
const selectedBizType = ref("")
const selectedOrderStatus = ref((route.query.status as string) || "")
const invoiceLoading = ref<string | null>(null)

const cancelLoading = ref<string | null>(null)
const cancelConfirmOrder = ref<OrderItem | null>(null)
const cancelConfirmOpen = computed(() => Boolean(cancelConfirmOrder.value))
const cancelConfirmDialogRef = ref<HTMLElement | null>(null)
useBodyScrollLock(() => cancelConfirmOpen.value)
const detailLoading = ref(false)
const detailLoadingOrderId = ref<string | null>(null)
const detailError = ref("")
const selectedOrderDetail = ref<OrderDetail | null>(null)
const selectedOrderItem = ref<OrderItem | null>(null)
const orderDetailDialogOpen = computed(() => detailLoading.value || Boolean(detailError.value) || Boolean(selectedOrderDetail.value))
const orderDetailDialogRef = ref<HTMLElement | null>(null)
useBodyScrollLock(() => orderDetailDialogOpen.value)
const orderPaymentDialogOpen = ref(false)
const orderPaymentSession = ref<{
  orderId: string
  bizType: string
  bizRefUlid: string
  source: string
  returnPath: string
  couponCodes: string[]
} | null>(null)

const invoiceOpeningLabel = computed(() => t.value.orders.invoiceOpening)
const orderStatusOptions = computed(() => [
  { value: "", label: t.value.orders.allStatuses },
  { value: "WAIT_PAYMENT", label: orderStatusFilterLabel("WAIT_PAYMENT") },
  { value: "PENDING", label: orderStatusFilterLabel("PENDING") },
  { value: "COMPLETED", label: orderStatusFilterLabel("COMPLETED") },
  { value: "CANCELLED", label: orderStatusFilterLabel("CANCELLED") },
  { value: "CLOSED", label: orderStatusFilterLabel("CLOSED") },
])

const actionableOrderStatuses = new Set([
  "WAIT_PAYMENT",
  "PENDING",
  "PENDING_PAYMENT",
  "WAIT_PIPELINE_PAYMENT",
  "WAIT_STAGE_PAYMENT",
  "WAIT_RETAKE_PAYMENT",
  "WAIT_UNLOCK_PAYMENT",
  "WAIT_BUNDLE_PAYMENT",
  "WAIT_REVIEW_FEE_PAYMENT",
])
const paidPaymentStatuses = new Set(["PAID"])
const paymentReturnSyncAttempts = 6
const paymentReturnSyncIntervalMs = 1500
const paymentSyncingOrderId = ref("")
let paymentSyncCancelled = false
let detailRequestSequence = 0
let ordersRequestId = 0

const detailExtraFields = computed<DetailField[]>(() => {
  const detail = selectedOrderDetail.value
  if (!detail) return []
  return [
    { label: t.value.orders.detailClosedAt, value: detail.closed_at || "" },
  ].filter((field) => field.value !== "")
})

const businessDetailFields = computed<DetailField[]>(() => {
  const response = selectedOrderDetail.value?.business_detail
  if (!response || typeof response !== "object" || Array.isArray(response)) return []
  const detail = recordValue(response.detail) || response
  const summary = recordValue(detail.summary)
  const values = {
    ...(summary || {}),
    ...Object.fromEntries(Object.entries(detail).filter(([key]) => key !== "summary")),
  }
  return Object.entries(values)
    .filter(([key, value]) => !hiddenBusinessDetailFields.has(key) && value !== undefined && value !== null && value !== "")
    .map(([key, value]) => ({
      label: businessDetailFieldLabel(key),
      value: displayBusinessValue(value),
    }))
})
const hiddenBusinessDetailFields = new Set([
  "candidate_ulid",
  "amount_minor",
  "currency_code",
  "created_at",
  "candidate_selected_exemptions_json",
  "final_exemptions_json",
  "items_snapshot_json",
])
const orderPricing = computed(() => selectedOrderDetail.value?.pricing || null)
const pricingCurrency = computed(() => String(orderPricing.value?.currency_code || selectedOrderDetail.value?.summary?.currency || ""))
const priceItems = computed(() => Array.isArray(orderPricing.value?.items) ? orderPricing.value!.items : [])
const orderExemptions = computed(() => Array.isArray(selectedOrderDetail.value?.exemptions) ? selectedOrderDetail.value!.exemptions : [])
const promotionLabels = computed(() => {
  const labels: string[] = []
  for (const coupon of orderPricing.value?.coupons || []) {
    const label = String(coupon?.name || coupon?.code || "").trim()
    if (label && !labels.includes(label)) labels.push(label)
  }
  for (const value of orderPricing.value?.promo_codes || []) {
    const label = String(value || "").trim()
    if (label && !labels.includes(label)) labels.push(label)
  }
  return labels
})
const priceMetrics = computed<PriceMetric[]>(() => {
  const pricing = orderPricing.value
  if (!pricing) return []
  const exemptionRecorded = pricing.exemption_amount_recorded === true
  return [
    {
      key: "billable-subtotal",
      label: t.value.orders.pricingBillableSubtotal,
      value: minorAmountText(pricing.billable_subtotal_minor, pricingCurrency.value),
      note: t.value.orders.pricingBillableSubtotalNote,
      tone: "border-slate-200 bg-white text-slate-950",
    },
    {
      key: "exemption-discount",
      label: t.value.orders.pricingExemptionDiscount,
      value: exemptionRecorded ? signedMinorAmountText(pricing.exemption_discount_minor, pricingCurrency.value, "-") : t.value.orders.pricingNotRecorded,
      note: orderExemptions.value.length
        ? t.value.orders.pricingExemptedUnits.replace("{{count}}", String(orderExemptions.value.length))
        : t.value.orders.pricingExemptionNotRecorded,
      tone: "border-amber-200 bg-amber-50 text-amber-950",
    },
    {
      key: "promotion-discount",
      label: t.value.orders.pricingPromotionDiscount,
      value: signedMinorAmountText(pricing.promotion_discount_minor, pricingCurrency.value, "-"),
      note: t.value.orders.pricingPromotionDiscountNote,
      tone: "border-emerald-200 bg-emerald-50 text-emerald-950",
    },
    {
      key: "tax",
      label: t.value.orders.pricingTax,
      value: signedMinorAmountText(pricing.tax_minor, pricingCurrency.value, "+"),
      note: t.value.orders.pricingTaxNote,
      tone: "border-slate-200 bg-slate-50 text-slate-950",
    },
    {
      key: "paid",
      label: t.value.orders.pricingAmountPaid,
      value: minorAmountText(pricing.amount_paid_minor ?? pricing.total_minor, pricingCurrency.value),
      note: t.value.orders.pricingAmountPaidNote,
      tone: "border-blue-200 bg-blue-50 text-blue-950",
    },
  ]
})

function recordValue(value: unknown): Record<string, unknown> | null {
  return value && typeof value === "object" && !Array.isArray(value) ? value as Record<string, unknown> : null
}

function displayBusinessValue(value: unknown) {
  if (typeof value === "string") return value
  if (typeof value === "number" || typeof value === "boolean") return String(value)
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

function businessDetailFieldLabel(key: string) {
  const labels = t.value.orders.businessFieldLabels as Record<string, string>
  return labels[key] || key
}

function minorAmountText(value: unknown, currency: string) {
  if (value === undefined || value === null || value === "") return t.value.orders.pricingUnavailable
  const amount = Number(value)
  if (!Number.isFinite(amount)) return t.value.orders.pricingUnavailable
  return `${currency ? `${currency.toUpperCase()} ` : ""}${(amount / 100).toFixed(2)}`
}

function signedMinorAmountText(value: unknown, currency: string, sign: "+" | "-") {
  const formatted = minorAmountText(value, currency)
  return formatted === t.value.orders.pricingUnavailable ? formatted : `${sign}${formatted}`
}

function pricingSourceLabel(value: unknown) {
  const source = String(value || "")
  const labels = t.value.orders.pricingSources as Record<string, string>
  return labels[source] || source
}

function normalizedStatus(value: unknown) {
  return String(value || "").trim().toUpperCase()
}

function isPaidPaymentStatus(value: unknown) {
  return paidPaymentStatuses.has(normalizedStatus(value))
}

function displayOrderStatus(order: OrderItem) {
  const orderStatus = normalizedStatus(order.order_status)
  if (actionableOrderStatuses.has(orderStatus) && isPaidPaymentStatus(order.payment_status)) {
    return "PAID"
  }
  return orderStatus
}

function orderStatusLabel(order: OrderItem) {
  if (paymentSyncingOrderId.value === order.id) return t.value.orders.statusPaymentSyncing
  return timelineStatusLabelWithDiagnostics(t.value, "MALL_ORDER", displayOrderStatus(order))
}

function orderStatusBadgeClass(order: OrderItem) {
  if (paymentSyncingOrderId.value === order.id) {
    return "border-blue-200 bg-blue-50 text-blue-700"
  }
  const status = displayOrderStatus(order)
  if (status === "COMPLETED" || status === "SUCCESS") {
    return "border-[#6CE9A6] bg-[#ECFDF3] text-[#027A48]"
  }
  return timelineStatusBadgeClassForStatus("MALL_ORDER", status)
}

async function openOrderDetail(order: OrderItem) {
  if (!order.id || detailLoading.value) return
  const requestSequence = ++detailRequestSequence
  detailLoading.value = true
  detailLoadingOrderId.value = order.id
  detailError.value = ""
  selectedOrderDetail.value = null
  selectedOrderItem.value = order
  try {
    const detail = await apiClient(`/api/orders/${encodeURIComponent(order.id)}`)
    if (requestSequence !== detailRequestSequence) return
    selectedOrderDetail.value = detail
    applyOrderDetail(detail, order.id)
  } catch (error) {
    if (requestSequence !== detailRequestSequence) return
    console.error(error)
    detailError.value = t.value.orders.detailLoadFailed
  } finally {
    if (requestSequence === detailRequestSequence) {
      detailLoading.value = false
      detailLoadingOrderId.value = null
    }
  }
}

function closeOrderDetail() {
  detailRequestSequence += 1
  detailLoading.value = false
  detailLoadingOrderId.value = null
  selectedOrderDetail.value = null
  selectedOrderItem.value = null
  detailError.value = ""
}

useDialogAccessibility(() => orderDetailDialogOpen.value, orderDetailDialogRef, closeOrderDetail)

function canContinuePayment(order: OrderItem) {
  if (!order.bizType || !order.bizRefUlid) return false
  if (paymentSyncingOrderId.value === order.id || isPaidPaymentStatus(order.payment_status)) return false
  return actionableOrderStatuses.has(normalizedStatus(order.order_status))
}

function continueOrderPayment(order: OrderItem) {
  if (!canContinuePayment(order)) return
  selectedOrderDetail.value = null
  selectedOrderItem.value = null
  detailError.value = ""
  orderPaymentSession.value = {
    orderId: order.id,
    bizType: order.bizType,
    bizRefUlid: order.bizRefUlid,
    source: "orders",
    returnPath: "/orders",
    couponCodes: [],
  }
  orderPaymentDialogOpen.value = true
}

function handlePaymentDialogOpenChange(open: boolean) {
  orderPaymentDialogOpen.value = open
  if (!open) orderPaymentSession.value = null
}

function canCancelOrder(order: OrderItem) {
  return Boolean(
    order.bizType
    && order.bizRefUlid
    && paymentSyncingOrderId.value !== order.id
    && !isPaidPaymentStatus(order.payment_status)
    && actionableOrderStatuses.has(normalizedStatus(order.order_status))
    && !cancelLoading.value,
  )
}

function openCancelConfirm(order: OrderItem) {
  if (!canCancelOrder(order)) return
  cancelConfirmOrder.value = order
}

function closeCancelConfirm() {
  cancelConfirmOrder.value = null
}

function confirmCancelOrder() {
  const order = cancelConfirmOrder.value
  if (!order) return
  closeCancelConfirm()
  void cancelOrder(order)
}

useDialogAccessibility(() => cancelConfirmOpen.value, cancelConfirmDialogRef, closeCancelConfirm)

async function cancelOrder(order: OrderItem) {
  if (!canCancelOrder(order)) return
  cancelLoading.value = order.bizRefUlid
  try {
    const res = await apiClient("/api/orders/cancel", {
      method: "POST",
      body: JSON.stringify({
        biz_type: order.bizType,
        biz_ref_ulid: order.bizRefUlid,
      }),
    })
    if (res?.success === false) {
      toast.error(t.value.orders.cancelOrderFailed)
      return
    }
    toast.success(t.value.orders.cancelOrderSuccess)
    await fetchOrders(false)
  } catch (error) {
    console.error(error)
    toast.error(t.value.orders.cancelOrderFailed)
  } finally {
    if (cancelLoading.value === order.bizRefUlid) cancelLoading.value = null
  }
}

function orderFromDetail(order: OrderItem, detail: OrderDetail) {
  const summary = detail.summary
  if (!summary) return order

  const orderStatus = normalizedStatus(summary.order_status) || order.order_status
  const paymentStatus = normalizedStatus(summary.payment_status) || order.payment_status
  const amount = Number(summary.amount)
  const currency = normalizedStatus(summary.currency) || order.currency
  return {
    ...order,
    items: summary.meta?.product_name ? [summary.meta.product_name] : order.items,
    amount: Number.isFinite(amount)
      ? orderAmountDisplay(amount, currency, orderStatus, t.value.orders.free)
      : order.amount,
    currency,
    bizType: summary.biz_type || order.bizType,
    bizRefUlid: summary.biz_ref_ulid || order.bizRefUlid,
    order_status: orderStatus,
    payment_status: paymentStatus,
  }
}

function applyOrderDetail(detail: OrderDetail, fallbackOrderId = "") {
  const orderId = String(detail.summary?.order_id || fallbackOrderId).trim()
  if (!orderId) return null

  let latestOrder: OrderItem | null = null
  orders.value = orders.value.map((order) => {
    if (order.id !== orderId) return order
    latestOrder = orderFromDetail(order, detail)
    return latestOrder
  })

  if (selectedOrderItem.value?.id === orderId) {
    latestOrder = orderFromDetail(selectedOrderItem.value, detail)
    selectedOrderItem.value = latestOrder
  }
  return latestOrder
}

async function viewInvoice(orderId: string) {
  if (!orderId || invoiceLoading.value) return
  invoiceLoading.value = orderId
  const redirectUrl = `/invoice-redirect?orderId=${encodeURIComponent(orderId)}`
  window.open(redirectUrl, "_blank", "noopener,noreferrer")
  window.setTimeout(() => {
    if (invoiceLoading.value === orderId) invoiceLoading.value = null
  }, 1200)
}

function formatMoney(amount: number, currency = "USD") {
  const normalizedCurrency = (currency || "USD").toUpperCase()
  try {
    return new Intl.NumberFormat(undefined, {
      style: "currency",
      currency: normalizedCurrency,
    }).format(amount)
  } catch {
    return `${normalizedCurrency} ${amount.toLocaleString()}`
  }
}

function orderAmountDisplay(amount: number, currency: string, orderStatus: string, freeText: string) {
  if (amount > 0) return formatMoney(amount, currency || "USD")
  const r = String(orderStatus || "").toUpperCase()
  if (r === "COMPLETED" || r === "SUCCESS" || r === "PAID" || r.includes("RESOLVED")) {
    return freeText
  }
  return "-"
}

function orderTypeLabel(bizType?: string) {
  const normalized = String(bizType || "").toUpperCase()
  switch (normalized) {
    case "PIPELINE_PAYMENT":
      return t.value.orders.typeCertification
    case "STAGE_PAYMENT":
      return t.value.orders.typeStage
    case "COURSE_RETAKE_PAYMENT":
      return t.value.orders.typeRetake
    case "PIPELINE_UNLOCK":
      return t.value.orders.typeCertificationUnlock
    case "CREDENTIAL_APPLICATION":
      return t.value.orders.typeCredentialApplication
    case "BUNDLE_PURCHASE":
      return t.value.orders.typeBundlePurchase
    default:
      return normalized || t.value.orders.typeOther
  }
}

function orderStatusFilterLabel(status?: string) {
  const normalized = String(status || "").toUpperCase()
  switch (normalized) {
    case "WAIT_PIPELINE_PAYMENT":
      return t.value.orders.filterCertificationPaymentPending
    case "WAIT_STAGE_PAYMENT":
      return t.value.orders.filterStagePaymentPending
    case "WAIT_RETAKE_PAYMENT":
      return t.value.orders.filterRetakePaymentPending
    case "WAIT_UNLOCK_PAYMENT":
      return t.value.orders.filterUnlockPaymentPending
    case "WAIT_BUNDLE_PAYMENT":
      return t.value.orders.filterBundlePaymentPending
    case "WAIT_REVIEW_FEE_PAYMENT":
      return t.value.orders.filterReviewFeePaymentPending
    case "WAIT_PAYMENT":
      return t.value.orders.statusWaitPayment
    case "PENDING":
      return t.value.orders.statusPending
    case "COMPLETED":
      return t.value.orders.statusCompleted
    case "CANCELLED":
      return t.value.orders.statusCancelled
    case "CLOSED":
      return t.value.orders.statusClosed
    default:
      return normalized || t.value.orders.allStatuses
  }
}

async function fetchOrders(showLoading = true, suppressErrorToast = false) {
  const requestId = ++ordersRequestId
  const requestedPage = page.value
  const requestedLastPage = lastPage.value
  const requestedPageSize = pageSize.value
  const requestedBizType = selectedBizType.value
  const requestedOrderStatus = selectedOrderStatus.value

  if (showLoading) {
    loading.value = true
    loadError.value = false
  }
  try {
    if (requestedPage > requestedLastPage) {
      currentCursor.value = nextCursor.value
    } else if (requestedPage < requestedLastPage) {
      currentCursor.value = prevCursor.value
    }
    const requestedCursor = currentCursor.value
    
    const params = new URLSearchParams({
      page_size: String(requestedPageSize),
    })
    
    if (requestedCursor) params.set("cursor", requestedCursor)
    if (requestedBizType) params.set("biz_type", requestedBizType)
    if (requestedOrderStatus) params.set("status", requestedOrderStatus)
    const res = await apiClient(`/api/orders?${params.toString()}`, { suppressErrorToast })
    if (requestId !== ordersRequestId) return

    totalOrders.value = Number(res.total_orders || 0)
    totalLabel.value = String(res.total_label || totalOrders.value)
    totalPages.value = Number(res.total_pages || 0)
    
    nextCursor.value = String(res.next_cursor || "")
    prevCursor.value = String(res.prev_cursor || "")
    
    // For cursorMode, hasMore controls the "Next" button.
    // When going backward, we naturally have a next page.
    const isBackward = requestedPage < requestedLastPage
    hasMore.value = isBackward ? true : Boolean(res.has_more)
    lastPage.value = requestedPage

    if (Array.isArray(res.orders)) {
      orders.value = res.orders.map((o: any) => ({
        id: o.order_id,
        invoiceOrderId: o.pay_order_ulid || o.pipeline_pay_order_ulid || "",
        canViewInvoice: Boolean(o.can_view_invoice) && Number(o.amount || 0) > 0,
        items: [o.product_name || orderTypeLabel(o.biz_type)],
        date: formatBackendDateMinute(o.created_at),
        currency: (o.currency || "USD").toUpperCase(),
        bizType: o.biz_type || "",
        bizRefUlid: o.biz_ref_ulid || "",
        amount: orderAmountDisplay(o.amount || 0, o.currency || "USD", o.order_status || "", t.value.orders.free),
        status: (o.order_status in statusConfig ? o.order_status : "pending") as OrderStatus,
        order_status: o.order_status,
        payment_status: o.payment_status,
        pipelineId: o.pipeline_id,
        paymentMethod: o.payment_method,
      }))
    } else {
      orders.value = []
    }
    loadError.value = false
  } catch (err) {
    if (requestId !== ordersRequestId) return
    console.error("Failed to fetch orders:", err)
    if (showLoading && !suppressErrorToast) loadError.value = true
  } finally {
    if (requestId === ordersRequestId) loading.value = false
  }
}

function consumePaymentReturn() {
  const url = new URL(window.location.href)
  const paymentStatus = normalizedStatus(url.searchParams.get("payment_status"))
  if (!paymentStatus) return null

  const orderId = String(url.searchParams.get("order_id") || "").trim()
  url.searchParams.delete("payment_status")
  url.searchParams.delete("payment_action")
  url.searchParams.delete("order_id")
  window.history.replaceState({}, "", `${url.pathname}${url.search}${url.hash}`)

  if (paymentStatus === "SUCCESS") {
    paymentSyncingOrderId.value = orderId
    toast.success(t.value.orders.paymentReturnSuccess)
  } else if (paymentStatus === "CANCELLED") {
    toast.warning(t.value.paymentReturnHandler.cancelled)
  } else if (paymentStatus === "FAILED") {
    toast.error(t.value.paymentReturnHandler.failed)
  }

  return { paymentStatus, orderId }
}

function waitForPaymentSync() {
  return new Promise<void>((resolve) => window.setTimeout(resolve, paymentReturnSyncIntervalMs))
}

async function syncReturnedOrder(orderId: string) {
  if (!orderId) {
    paymentSyncingOrderId.value = ""
    return
  }

  for (let attempt = 0; attempt < paymentReturnSyncAttempts && !paymentSyncCancelled; attempt += 1) {
    try {
      const detail: OrderDetail = await apiClient(`/api/orders/${encodeURIComponent(orderId)}`, {
        suppressErrorToast: true,
      })
      applyOrderDetail(detail, orderId)
      const orderStatus = normalizedStatus(detail.summary?.order_status)
      if (orderStatus === "COMPLETED" || orderStatus === "SUCCESS" || isPaidPaymentStatus(detail.summary?.payment_status)) {
        paymentSyncingOrderId.value = ""
        return
      }
    } catch (error) {
      console.error("Failed to synchronize returned payment order", error)
    }

    if (attempt < paymentReturnSyncAttempts - 1) await waitForPaymentSync()
  }

  if (!paymentSyncCancelled) toast.warning(t.value.orders.paymentSyncDelayed)
}

async function handleOrderPaymentComplete() {
  const orderId = orderPaymentSession.value?.orderId || ""
  orderPaymentDialogOpen.value = false
  orderPaymentSession.value = null
  selectedOrderDetail.value = null
  selectedOrderItem.value = null

  if (!orderId) {
    await fetchOrders(false, true)
    return
  }

  paymentSyncingOrderId.value = orderId
  toast.success(t.value.orders.paymentReturnSuccess)
  await fetchOrders(false, true)
  await syncReturnedOrder(orderId)
}

function resetCursorPagination() {
  page.value = 1
  lastPage.value = 1
  currentCursor.value = ""
  nextCursor.value = ""
  prevCursor.value = ""
  hasMore.value = false
}

function changeOrderStatus(value: string) {
  selectedOrderStatus.value = value
  resetCursorPagination()
  void fetchOrders()
}

function handlePaginationChange() {
  if (loading.value) return
  if (pageSize.value !== lastPageSize.value) {
    lastPageSize.value = pageSize.value
    resetCursorPagination()
  }
  void fetchOrders()
  window.scrollTo({ top: 0, behavior: "smooth" })
}

onMounted(async () => {
  const paymentReturn = consumePaymentReturn()
  await fetchOrders()
  if (paymentReturn?.paymentStatus === "SUCCESS") {
    await syncReturnedOrder(paymentReturn.orderId)
  }
})

onBeforeUnmount(() => {
  paymentSyncCancelled = true
  ordersRequestId += 1
})
</script>

<template>
  <AppShell content-class="p-0">
    <div v-if="invoiceLoading" class="fixed right-5 top-5 z-50 flex items-center gap-3 rounded-2xl border border-emerald-100 bg-white px-4 py-3 text-sm font-semibold text-slate-700 shadow-[0_16px_40px_rgba(15,74,82,0.14)]">
      <Loader2 class="h-4 w-4 animate-spin text-emerald-500" />
      <span>{{ invoiceOpeningLabel }}</span>
    </div>

    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <Receipt class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ t.orders.title }}</span>
      </header>

      <main class="px-5 py-8 md:px-8 lg:px-10">
        <div class="orders-page-intro mb-6">
          <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.orders.title }}</h1>
          <p class="mt-2 text-muted-foreground">{{ t.orders.subtitle }}</p>
        </div>

    <div class="orders-panel overflow-hidden rounded-[16px] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <div class="orders-toolbar flex flex-col gap-3 border-b border-slate-100 bg-white px-4 py-4 lg:flex-row lg:items-center lg:justify-between">
        <div class="flex items-center">
          <h2 class="font-semibold text-card-foreground">{{ t.orders.orderHistory }}</h2>
        </div>
        <div class="flex flex-col gap-3 sm:items-end">
          <div class="flex w-full flex-col gap-3 sm:w-auto sm:flex-row">
            <label class="sr-only" for="order-status-filter">{{ t.orders.orderStatus }}</label>
            <select
              id="order-status-filter"
              :value="selectedOrderStatus"
              class="h-10 w-full rounded-lg border border-slate-200 bg-white px-3 text-sm font-medium text-slate-700 shadow-sm outline-none transition-colors hover:border-amber-400 focus:border-amber-500 focus:ring-2 focus:ring-amber-500/15 sm:w-48"
              @change="changeOrderStatus(($event.target as HTMLSelectElement).value)"
            >
              <option v-for="option in orderStatusOptions" :key="option.value || 'ALL_STATUSES'" :value="option.value">
                {{ option.label }}
              </option>
            </select>
          </div>
        </div>
      </div>

      <div v-if="loading" class="orders-state flex items-center justify-center gap-2 py-16 text-muted-foreground"><Loader2 class="h-5 w-5 animate-spin text-primary" /> {{ t.common.loading }}</div>
      <div v-else-if="loadError" class="orders-state flex flex-col items-center justify-center px-4 py-16 text-center">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-red-50">
          <AlertCircle class="h-8 w-8 text-red-600" />
        </div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.orders.loadFailed }}</h3>
        <p class="mb-5 max-w-md text-muted-foreground">{{ t.orders.loadFailedDesc }}</p>
        <button type="button" class="btn btn-primary min-w-32 justify-center" @click="fetchOrders()">
          <RefreshCw class="h-4 w-4" />
          {{ t.orders.retry }}
        </button>
      </div>
      <div v-else-if="orders.length === 0" class="orders-state flex flex-col items-center justify-center px-4 py-14 text-center">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10"><Package class="h-8 w-8 text-primary" /></div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.orders.noOrders }}</h3>
      </div>
      <div v-else>
        <div
          v-for="order in orders"
          :key="order.id"
          class="order-row group relative flex cursor-pointer flex-col gap-3 border-b border-slate-100 px-4 py-4 transition-all duration-200 hover:bg-primary/10 md:flex-row md:items-center md:justify-between"
        >
          <button
            type="button"
            :aria-label="`${t.orders.detailTitle}: ${order.items.join(', ')}`"
            class="order-summary flex min-w-0 items-center gap-4 text-left focus-visible:outline-none after:absolute after:inset-0 after:z-0 focus-visible:after:ring-2 focus-visible:after:ring-inset focus-visible:after:ring-primary/50"
            @click="openOrderDetail(order)"
          >
            <div class="order-product-icon flex h-12 w-12 shrink-0 items-center justify-center rounded-lg bg-primary/10"><Package class="h-6 w-6 text-primary" /></div>
            <div class="min-w-0">
              <div class="order-heading-line mb-1 flex min-w-0 flex-wrap items-center gap-2">
                <h3 class="order-title-mobile min-w-0 max-w-full font-medium text-card-foreground">{{ order.items.join(", ") }}</h3>
                <span class="rounded-full border border-primary/15 bg-primary/5 px-2 py-0.5 text-xs font-semibold text-primary">{{ orderTypeLabel(order.bizType) }}</span>
              </div>
              <p class="text-sm text-muted-foreground">{{ order.date }}</p>
            </div>
          </button>
          <div class="order-actions pointer-events-none relative z-10 flex w-full flex-wrap items-center justify-end gap-x-3 gap-y-3 pl-16 md:grid md:w-auto md:shrink-0 md:grid-cols-[130px_148px_112px_112px_24px] md:gap-x-5 md:pl-0">
            <div class="mr-auto flex justify-start md:mr-0 md:justify-center">
              <span class="badge text-xs" :class="orderStatusBadgeClass(order)">
                {{ orderStatusLabel(order) }}
              </span>
            </div>
            <div class="text-right">
              <button v-if="canContinuePayment(order)" @click.stop="continueOrderPayment(order)" class="pointer-events-auto inline-flex h-10 min-w-[148px] items-center justify-center whitespace-nowrap rounded-lg bg-primary/10 px-3 text-sm font-semibold text-primary transition-colors hover:bg-primary/20 md:h-8 md:w-full">
                {{ t.orders.continuePayment }}
              </button>
              <p v-else class="order-amount text-lg font-semibold text-card-foreground">{{ order.amount }}</p>
            </div>
            <button v-if="canCancelOrder(order)" type="button" class="pointer-events-auto inline-flex h-10 w-full items-center justify-center gap-2 whitespace-nowrap rounded-lg border border-red-200 bg-red-50 px-3 text-sm font-semibold text-red-600 transition-colors hover:border-red-300 hover:bg-red-100 md:h-8" @click.stop="openCancelConfirm(order)">
              <Loader2 v-if="cancelLoading === order.bizRefUlid" class="h-4 w-4 animate-spin" />
              {{ t.orders.cancelPayment }}
            </button>
            <span v-else class="order-action-placeholder h-10 w-[112px] md:h-8" />
            <button v-if="order.canViewInvoice" @click.stop="viewInvoice(order.invoiceOrderId)" class="pointer-events-auto inline-flex h-10 w-[112px] items-center justify-center gap-2 whitespace-nowrap rounded-lg bg-primary/10 px-3 text-sm font-semibold text-primary transition-colors hover:bg-primary/20 md:h-9">
              <Loader2 v-if="invoiceLoading === order.invoiceOrderId" class="h-4 w-4 animate-spin" />
              {{ t.orders.viewInvoice }}
            </button>
            <span v-else class="order-action-placeholder h-10 w-[112px] md:h-9" />

            <Loader2 v-if="detailLoadingOrderId === order.id" class="order-detail-indicator h-5 w-5 animate-spin text-muted-foreground" />
            <ChevronRight v-else class="order-detail-indicator h-5 w-5 text-muted-foreground transition-transform group-hover:translate-x-1" />
          </div>
        </div>
        <AppPagination
          v-model:page="page"
          v-model:page-size="pageSize"
          :total="totalOrders"
          :total-pages="totalPages"
          :total-label="totalLabel"
          :page-size-options="pageSizeOptions"
          :disabled="loading"
          cursor-mode
          :has-more="hasMore"
          @page-change="handlePaginationChange"
        />
      </div>
    </div>

      </main>
    </div>

    <div v-if="cancelConfirmOpen" class="app-safe-area-overlay fixed inset-0 z-50 flex items-center justify-center bg-slate-950/55 p-4 backdrop-blur-sm">
      <div
        ref="cancelConfirmDialogRef"
        class="w-full max-w-md overflow-hidden rounded-[16px] bg-white shadow-[0_24px_60px_rgba(15,23,42,0.24)]"
        role="dialog"
        aria-modal="true"
        aria-labelledby="order-cancel-confirm-title"
        aria-describedby="order-cancel-confirm-description"
        tabindex="-1"
      >
        <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-5 py-4">
          <div class="flex min-w-0 items-center gap-3">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-red-50 text-red-600">
              <AlertCircle class="h-5 w-5" />
            </div>
            <h2 id="order-cancel-confirm-title" class="text-lg font-semibold text-slate-950">
              {{ t.orders.cancelOrder }}
            </h2>
          </div>
          <button
            type="button"
            class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full border border-slate-200 text-slate-500 transition-colors hover:border-red-200 hover:bg-red-50 hover:text-red-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500/30 md:h-10 md:w-10"
            :aria-label="t.common.close"
            :title="t.common.close"
            @click="closeCancelConfirm"
          >
            <X class="h-5 w-5" />
          </button>
        </div>

        <div class="px-5 py-5">
          <p id="order-cancel-confirm-description" class="text-sm leading-6 text-slate-600">
            {{ t.orders.cancelOrderConfirm }}
          </p>
        </div>

        <div class="flex justify-end gap-3 border-t border-slate-200 bg-slate-50 px-5 py-4">
          <button type="button" class="btn btn-outline min-w-24 rounded-lg" @click="closeCancelConfirm">
            {{ t.orders.keepOrder }}
          </button>
          <button
            type="button"
            class="btn min-w-24 rounded-lg border-red-600 bg-red-600 text-white shadow-sm shadow-red-200 hover:border-red-700 hover:bg-red-700 focus-visible:ring-red-500/40"
            @click="confirmCancelOrder"
          >
            {{ t.orders.cancelOrder }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="detailLoading || detailError || selectedOrderDetail" class="app-safe-area-overlay-order fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 backdrop-blur-[2px]">
      <div
        ref="orderDetailDialogRef"
        class="app-dialog-viewport flex max-h-[92vh] w-full max-w-5xl flex-col overflow-hidden rounded-[22px] bg-white shadow-[0_28px_90px_rgba(15,23,42,0.28)]"
        role="dialog"
        aria-modal="true"
        aria-labelledby="order-detail-dialog-title"
        tabindex="-1"
      >
        <header class="flex items-start justify-between gap-4 border-b border-slate-100 bg-white px-5 py-4 sm:px-6 sm:py-5">
          <div class="flex min-w-0 items-start gap-3">
            <div class="hidden h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-primary/10 text-primary sm:flex">
              <Receipt class="h-5 w-5" />
            </div>
            <div class="min-w-0">
              <h2 id="order-detail-dialog-title" class="text-xl font-bold text-slate-950">{{ t.orders.detailTitle }}</h2>
              <p class="mt-1 break-all text-sm text-muted-foreground">{{ selectedOrderDetail?.summary?.order_id || t.orders.detailSubtitle }}</p>
            </div>
          </div>
          <button type="button" class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white/90 text-slate-500 transition hover:border-primary/25 hover:text-primary md:h-10 md:w-10" :aria-label="t.common.close" :title="t.common.close" @click="closeOrderDetail">
            <X class="h-5 w-5" />
          </button>
        </header>

        <div class="overflow-y-auto bg-slate-50/70 px-4 py-4 sm:px-6 sm:py-5">
          <div v-if="detailLoading" class="flex items-center justify-center gap-2 py-16 text-muted-foreground">
            <Loader2 class="h-5 w-5 animate-spin text-primary" />
            {{ t.common.loading }}
          </div>
          <div v-else-if="detailError" class="rounded-xl border border-red-100 bg-red-50 px-4 py-3 text-sm font-semibold text-red-700">
            {{ detailError }}
          </div>
          <div v-else-if="selectedOrderDetail" class="space-y-4">
            <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm shadow-slate-200/60">
              <div class="border-b border-primary/10 bg-[#f4f6fa] px-4 py-4 sm:px-5">
                <div class="mb-4 flex items-start justify-between gap-3">
                  <h3 class="font-semibold text-slate-950">{{ t.orders.detailSummary }}</h3>
                  <span v-if="selectedOrderDetail.summary?.order_status" class="badge shrink-0 text-xs" :class="timelineStatusBadgeClassForStatus('MALL_ORDER', selectedOrderDetail.summary.order_status)">
                    {{ timelineStatusLabelWithDiagnostics(t, 'MALL_ORDER', selectedOrderDetail.summary.order_status) }}
                  </span>
                </div>
                <div class="grid gap-4 sm:grid-cols-[1fr_auto] sm:items-end">
                  <div class="min-w-0">
                    <p class="text-xs font-semibold text-slate-500">{{ t.orders.detailProductName }}</p>
                    <h4 class="mt-1 break-words text-lg font-black leading-snug text-slate-950">
                      {{ selectedOrderDetail.summary?.meta?.product_name || "-" }}
                    </h4>
                  </div>
                  <div class="rounded-2xl border border-white/70 bg-white/80 px-4 py-3 shadow-sm sm:min-w-44 sm:text-right">
                    <p class="text-xs font-semibold text-slate-500">{{ t.orders.detailAmount }}</p>
                    <p class="mt-1 text-2xl font-black tracking-tight text-primary">
                      {{ orderAmountDisplay(Number(selectedOrderDetail.summary?.amount || 0), selectedOrderDetail.summary?.currency || "USD", selectedOrderDetail.summary?.order_status || "", t.orders.free) }}
                    </p>
                  </div>
                </div>
              </div>
              <dl class="divide-y divide-slate-100 px-4 sm:px-5">
                <div class="grid gap-1 py-3 sm:grid-cols-[140px_1fr] sm:gap-4">
                  <dt class="text-xs font-semibold text-slate-500">{{ t.orders.detailOrderId }}</dt>
                  <dd class="break-all text-sm font-semibold text-slate-950">{{ selectedOrderDetail.summary?.order_id || "-" }}</dd>
                </div>
                <div class="grid gap-1 py-3 sm:grid-cols-[140px_1fr] sm:gap-4">
                  <dt class="text-xs font-semibold text-slate-500">{{ t.orders.detailType }}</dt>
                  <dd class="break-words text-sm font-semibold text-slate-950">{{ orderTypeLabel(selectedOrderDetail.summary?.biz_type) }}</dd>
                </div>
                <div class="grid gap-1 py-3 sm:grid-cols-[140px_1fr] sm:gap-4">
                  <dt class="text-xs font-semibold text-slate-500">{{ t.orders.detailPaidAt }}</dt>
                  <dd class="break-words text-sm font-semibold text-slate-950">{{ selectedOrderDetail.paid_at || "-" }}</dd>
                </div>
                <div class="grid gap-1 py-3 sm:grid-cols-[140px_1fr] sm:gap-4">
                  <dt class="text-xs font-semibold text-slate-500">{{ t.orders.detailCreatedAt }}</dt>
                  <dd class="break-words text-sm font-semibold text-slate-950">{{ selectedOrderDetail.summary?.created_at || "-" }}</dd>
                </div>
              </dl>
            </section>

            <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm shadow-slate-200/60">
              <div class="flex flex-wrap items-start justify-between gap-3 border-b border-slate-100 px-4 py-4 sm:px-5">
                <div>
                  <h3 class="font-semibold text-slate-950">{{ t.orders.pricingTitle }}</h3>
                  <p class="mt-1 text-xs leading-5 text-slate-500">{{ t.orders.pricingDescription }}</p>
                </div>
                <span v-if="orderPricing?.source" class="rounded-full bg-slate-100 px-3 py-1 text-xs font-bold text-slate-600">
                  {{ pricingSourceLabel(orderPricing.source) }}
                </span>
              </div>

              <template v-if="orderPricing">
                <div class="grid gap-3 p-4 sm:grid-cols-2 sm:p-5 lg:grid-cols-5">
                  <div
                    v-for="metric in priceMetrics"
                    :key="metric.key"
                    class="min-h-28 rounded-lg border p-4"
                    :class="metric.tone"
                  >
                    <div class="text-xs font-black">{{ metric.label }}</div>
                    <div class="mt-2 break-words text-lg font-black">{{ metric.value }}</div>
                    <div class="mt-2 text-xs font-medium leading-5 opacity-70">{{ metric.note }}</div>
                  </div>
                </div>

                <div v-if="orderPricing.unavailable_reason" class="mx-4 mb-4 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm font-medium text-amber-900 sm:mx-5 sm:mb-5">
                  {{ t.orders.pricingPartialUnavailable }}
                </div>

                <div v-if="promotionLabels.length" class="flex flex-wrap items-center gap-2 border-t border-slate-100 px-4 py-4 sm:px-5">
                  <span class="text-xs font-black text-slate-500">{{ t.orders.pricingPromotions }}</span>
                  <span v-for="label in promotionLabels" :key="label" class="rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1 text-xs font-bold text-emerald-800">
                    {{ label }}
                  </span>
                </div>

                <div v-if="orderExemptions.length" class="border-t border-slate-100 px-4 py-4 sm:px-5">
                  <div class="text-xs font-black text-slate-500">{{ t.orders.pricingExemptedItems }}</div>
                  <div class="mt-2 grid gap-2 sm:grid-cols-2">
                    <div v-for="item in orderExemptions" :key="`${item.course_cc_ulid}-${item.credential_ulid}`" class="min-w-0 border-l-2 border-amber-300 bg-amber-50 px-3 py-2">
                      <div class="break-all text-xs font-bold text-amber-950">{{ item.course_cc_ulid }}</div>
                      <div v-if="item.credential_ulid" class="mt-1 break-all text-xs text-amber-700">{{ item.credential_ulid }}</div>
                    </div>
                  </div>
                </div>

                <div v-if="priceItems.length" class="border-t border-slate-100">
                  <div class="bg-slate-50 px-4 py-3 text-sm font-black text-slate-800 sm:px-5">{{ t.orders.pricingItemsTitle }}</div>
                  <div class="overflow-x-auto">
                    <table class="w-full min-w-[620px] text-left text-sm">
                      <thead class="border-y border-slate-200 bg-white text-xs text-slate-500">
                        <tr>
                          <th class="px-4 py-3 font-black sm:px-5">{{ t.orders.pricingItem }}</th>
                          <th class="px-4 py-3 font-black">{{ t.orders.pricingUnitPrice }}</th>
                          <th class="px-4 py-3 text-center font-black">{{ t.orders.pricingQuantity }}</th>
                          <th class="px-4 py-3 text-right font-black sm:px-5">{{ t.orders.pricingSubtotal }}</th>
                        </tr>
                      </thead>
                      <tbody class="divide-y divide-slate-100">
                        <tr v-for="item in priceItems" :key="`${item.item_type}-${item.item_ulid}`">
                          <td class="px-4 py-3 sm:px-5">
                            <div class="font-bold text-slate-900">{{ item.title || item.item_ulid || t.orders.pricingUnnamedItem }}</div>
                            <div class="mt-1 break-all text-xs text-slate-500">{{ item.item_type }}<span v-if="item.item_ulid"> · {{ item.item_ulid }}</span></div>
                          </td>
                          <td class="px-4 py-3 font-semibold text-slate-700">{{ minorAmountText(item.unit_price_minor, pricingCurrency) }}</td>
                          <td class="px-4 py-3 text-center font-semibold text-slate-700">{{ item.quantity }}</td>
                          <td class="px-4 py-3 text-right font-black text-slate-900 sm:px-5">{{ minorAmountText(item.subtotal_minor, pricingCurrency) }}</td>
                        </tr>
                      </tbody>
                    </table>
                  </div>
                </div>
              </template>
              <div v-else class="p-8 text-center text-sm text-slate-500">{{ t.orders.pricingUnavailableDetail }}</div>
            </section>

            <section v-if="detailExtraFields.length" class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm shadow-slate-200/60">
              <h3 class="border-b border-slate-100 bg-white px-4 py-3 font-semibold text-slate-950 sm:px-5">{{ t.orders.detailPaymentInfo }}</h3>
              <dl class="grid gap-3 p-4 sm:grid-cols-2 sm:p-5">
                <div v-for="field in detailExtraFields" :key="field.label" class="rounded-2xl border border-slate-100 bg-slate-50/80 px-4 py-3">
                  <dt class="text-xs font-semibold text-slate-500">{{ field.label }}</dt>
                  <dd class="mt-1.5 break-words text-sm font-bold leading-snug text-slate-950">{{ field.value }}</dd>
                </div>
              </dl>
            </section>

            <section v-if="businessDetailFields.length" class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm shadow-slate-200/60">
              <h3 class="border-b border-slate-100 bg-white px-4 py-3 font-semibold text-slate-950 sm:px-5">{{ t.orders.detailBusinessInfo }}</h3>
              <dl class="grid gap-3 p-4 sm:grid-cols-2 sm:p-5">
                <div v-for="field in businessDetailFields" :key="field.label" class="rounded-2xl border border-slate-100 bg-slate-50/80 px-4 py-3">
                  <dt class="break-all text-xs font-semibold text-slate-500">{{ field.label }}</dt>
                  <dd class="mt-1.5 break-all text-sm font-bold leading-snug text-slate-950">{{ field.value }}</dd>
                </div>
              </dl>
            </section>

          </div>
        </div>
        <div v-if="selectedOrderItem && canContinuePayment(selectedOrderItem)" class="border-t border-slate-100 bg-slate-50 px-5 py-4 sm:px-6">
          <div class="flex justify-end">
            <button type="button" class="btn btn-primary min-w-36 justify-center" @click="continueOrderPayment(selectedOrderItem)">
              <CreditCard class="h-4 w-4" />
              {{ t.orders.continuePayment }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <PaymentSessionDialog
      v-if="orderPaymentSession"
      :open="orderPaymentDialogOpen"
      :title="t.orders.continuePayment"
      :subtitle="orderPaymentSession.orderId"
      :biz-type="orderPaymentSession.bizType"
      :biz-ref-ulid="orderPaymentSession.bizRefUlid"
      :order-id="orderPaymentSession.orderId"
      :source="orderPaymentSession.source"
      :return-path="orderPaymentSession.returnPath"
      :coupon-codes="orderPaymentSession.couponCodes"
      :redirect-on-complete="false"
      @update:open="handlePaymentDialogOpenChange"
      @complete="handleOrderPaymentComplete"
    />
  </AppShell>
</template>

<style scoped>
.order-row {
  box-shadow: inset 0 0 0 1px transparent;
}

.order-row:hover {
  box-shadow: inset 3px 0 0 rgba(37, 99, 235, 0.55);
}

.order-title-mobile {
  overflow-wrap: anywhere;
  word-break: break-word;
}

@media (max-width: 767px) {
  .orders-page-intro {
    margin-bottom: 16px;
  }

  .orders-toolbar {
    gap: 8px;
    padding: 12px;
  }

  .orders-toolbar > :last-child,
  .orders-toolbar > :last-child > div {
    gap: 8px;
  }

  .orders-state {
    padding-block: 32px;
  }

  .order-row {
    gap: 8px;
    padding: 12px;
  }

  .order-summary {
    gap: 12px;
  }

  .order-product-icon {
    width: 40px;
    height: 40px;
  }

  .order-heading-line {
    gap: 6px;
    margin-bottom: 2px;
  }

  .order-title-mobile {
    font-size: 14px;
    line-height: 1.45;
  }

  .order-actions {
    gap: 8px;
    padding-left: 52px;
    padding-right: 28px;
  }

  .order-action-placeholder {
    display: none;
  }

  .order-amount {
    font-size: 16px;
    line-height: 24px;
  }

  .order-detail-indicator {
    position: absolute;
    right: 0;
    bottom: 0;
  }
}
</style>
