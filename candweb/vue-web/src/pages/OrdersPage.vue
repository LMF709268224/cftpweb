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
  discount_unsupported?: boolean
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
    .filter(([, value]) => value !== undefined && value !== null && value !== "")
    .map(([key, value]) => ({
      label: key,
      value: displayBusinessValue(value),
    }))
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
  if (showLoading) {
    loading.value = true
    loadError.value = false
  }
  try {
    if (page.value > lastPage.value) {
      currentCursor.value = nextCursor.value
    } else if (page.value < lastPage.value) {
      currentCursor.value = prevCursor.value
    }
    
    const params = new URLSearchParams({
      page_size: String(pageSize.value),
    })
    
    if (currentCursor.value) params.set("cursor", currentCursor.value)
    if (selectedBizType.value) params.set("biz_type", selectedBizType.value)
    if (selectedOrderStatus.value) params.set("status", selectedOrderStatus.value)
    const res = await apiClient(`/api/orders?${params.toString()}`, { suppressErrorToast })
    totalOrders.value = Number(res.total_orders || 0)
    totalLabel.value = String(res.total_label || totalOrders.value)
    totalPages.value = Number(res.total_pages || 0)
    
    nextCursor.value = String(res.next_cursor || "")
    prevCursor.value = String(res.prev_cursor || "")
    
    // For cursorMode, hasMore controls the "Next" button.
    // When going backward, we naturally have a next page.
    const isBackward = page.value < lastPage.value
    hasMore.value = isBackward ? true : Boolean(res.has_more)
    lastPage.value = page.value

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
    console.error("Failed to fetch orders:", err)
    if (showLoading && !suppressErrorToast) loadError.value = true
  } finally {
    if (showLoading) loading.value = false
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
        <div class="mb-6">
          <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.orders.title }}</h1>
          <p class="mt-2 text-muted-foreground">{{ t.orders.subtitle }}</p>
        </div>

    <div class="overflow-hidden rounded-[16px] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <div class="flex flex-col gap-3 border-b border-slate-100 bg-white px-4 py-4 lg:flex-row lg:items-center lg:justify-between">
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

      <div v-if="loading" class="flex items-center justify-center gap-2 py-16 text-muted-foreground"><Loader2 class="h-5 w-5 animate-spin" /> {{ t.common.loading }}</div>
      <div v-else-if="loadError" class="flex flex-col items-center justify-center px-4 py-16 text-center">
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
      <div v-else-if="orders.length === 0" class="flex flex-col items-center justify-center px-4 py-14 text-center">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10"><Package class="h-8 w-8 text-primary" /></div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.orders.noOrders }}</h3>
      </div>
      <div v-else>
        <div v-for="order in orders" :key="order.id" @click="openOrderDetail(order)" class="order-row group flex cursor-pointer flex-col gap-3 border-b border-slate-100 px-4 py-4 transition-all duration-200 hover:bg-primary/10 md:flex-row md:items-center md:justify-between">
          <div class="flex min-w-0 items-center gap-4">
            <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-lg bg-primary/10"><Package class="h-6 w-6 text-primary" /></div>
            <div class="min-w-0">
              <div class="mb-1 flex min-w-0 flex-wrap items-center gap-2">
                <h3 class="order-title-mobile min-w-0 max-w-full font-medium text-card-foreground">{{ order.items.join(", ") }}</h3>
                <span class="rounded-full border border-primary/15 bg-primary/5 px-2 py-0.5 text-xs font-semibold text-primary">{{ orderTypeLabel(order.bizType) }}</span>
              </div>
              <p class="text-sm text-muted-foreground">{{ order.date }}</p>
            </div>
          </div>
          <div class="flex w-full flex-wrap items-center justify-end gap-x-3 gap-y-3 pl-16 md:grid md:w-auto md:shrink-0 md:grid-cols-[130px_148px_112px_112px_24px] md:gap-x-5 md:pl-0">
            <div class="mr-auto flex justify-start md:mr-0 md:justify-center">
              <span class="badge text-xs" :class="orderStatusBadgeClass(order)">
                {{ orderStatusLabel(order) }}
              </span>
            </div>
            <div class="text-right">
              <button v-if="canContinuePayment(order)" @click.stop="continueOrderPayment(order)" class="inline-flex h-8 min-w-[148px] items-center justify-center whitespace-nowrap rounded-lg bg-primary/10 px-3 text-sm font-semibold text-primary transition-colors hover:bg-primary/20 md:w-full">
                {{ t.orders.continuePayment }}
              </button>
              <p v-else class="text-lg font-semibold text-card-foreground">{{ order.amount }}</p>
            </div>
            <button v-if="canCancelOrder(order)" type="button" class="inline-flex h-8 w-full items-center justify-center gap-2 whitespace-nowrap rounded-lg border border-red-200 bg-red-50 px-3 text-sm font-semibold text-red-600 transition-colors hover:border-red-300 hover:bg-red-100" @click.stop="openCancelConfirm(order)">
              <Loader2 v-if="cancelLoading === order.bizRefUlid" class="h-4 w-4 animate-spin" />
              {{ t.orders.cancelPayment }}
            </button>
            <span v-else class="h-8 w-[112px]" />
            <button v-if="order.canViewInvoice" @click.stop="viewInvoice(order.invoiceOrderId)" class="inline-flex h-9 w-[112px] items-center justify-center gap-2 whitespace-nowrap rounded-lg bg-primary/10 px-3 text-sm font-semibold text-primary transition-colors hover:bg-primary/20">
              <Loader2 v-if="invoiceLoading === order.invoiceOrderId" class="h-4 w-4 animate-spin" />
              {{ t.orders.viewInvoice }}
            </button>
            <span v-else class="h-9 w-[112px]" />

            <Loader2 v-if="detailLoadingOrderId === order.id" class="h-5 w-5 animate-spin text-muted-foreground" />
            <ChevronRight v-else class="h-5 w-5 text-muted-foreground transition-transform group-hover:translate-x-1" />
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

    <div v-if="cancelConfirmOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/55 p-4 backdrop-blur-sm">
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
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 text-slate-500 transition-colors hover:border-red-200 hover:bg-red-50 hover:text-red-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500/30"
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

    <div v-if="detailLoading || detailError || selectedOrderDetail" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 px-3 py-4 backdrop-blur-[2px] sm:px-4 sm:py-6">
      <div
        ref="orderDetailDialogRef"
        class="flex max-h-[92vh] w-full max-w-3xl flex-col overflow-hidden rounded-[22px] bg-white shadow-[0_28px_90px_rgba(15,23,42,0.28)]"
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
          <button type="button" class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white/90 text-slate-500 transition hover:border-primary/25 hover:text-primary" :aria-label="t.common.close" :title="t.common.close" @click="closeOrderDetail">
            <X class="h-5 w-5" />
          </button>
        </header>

        <div class="overflow-y-auto bg-slate-50/70 px-4 py-4 sm:px-6 sm:py-5">
          <div v-if="detailLoading" class="flex items-center justify-center gap-2 py-16 text-muted-foreground">
            <Loader2 class="h-5 w-5 animate-spin" />
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

            <section v-if="detailExtraFields.length" class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm shadow-slate-200/60">
              <h3 class="border-b border-slate-100 bg-white px-4 py-3 font-semibold text-slate-950 sm:px-5">{{ t.orders.detailPaymentInfo }}</h3>
              <dl class="grid gap-3 p-4 sm:grid-cols-2 sm:p-5">
                <div v-for="field in detailExtraFields" :key="field.label" class="rounded-2xl border border-slate-100 bg-slate-50/80 px-4 py-3">
                  <dt class="text-xs font-semibold text-slate-500">{{ field.label }}</dt>
                  <dd class="mt-1.5 break-words text-sm font-bold leading-snug text-slate-950">{{ field.value }}</dd>
                </div>
              </dl>
            </section>

            <section v-if="false && businessDetailFields.length" class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm shadow-slate-200/60">
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

@media (max-width: 640px) {
  .order-title-mobile {
    font-size: 14px;
    line-height: 1.45;
  }
}
</style>
