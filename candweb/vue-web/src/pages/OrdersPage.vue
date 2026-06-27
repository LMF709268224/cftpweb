<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useRouter } from "vue-router"
import { toast } from "vue-sonner"
import { ChevronRight, CreditCard, FileText, Loader2, Package, Receipt, XCircle } from "lucide-vue-next"
import { timelineStatusBadgeClassForStatus, timelineStatusLabelWithDiagnostics } from "@/lib/status-labels"
import AppShell from "@/components/AppShell.vue"
import AppPagination from "@/components/AppPagination.vue"
import PaymentSessionDialog from "@/components/PaymentSessionDialog.vue"
import PurchaseDialog from "@/components/PurchaseDialog.vue"
import { apiClient } from "@/lib/apiClient"
import { formatBackendDateMinute } from "@/lib/utils"
import { useTranslation } from "@/lib/language"
import { usePolling } from "@/lib/polling"

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
  rawStatus: string
  pipelineId: string
  paymentMethod: string
  canCancel: boolean
}

const statusConfig = {
  completed: { labelKey: "statusCompleted", statusValue: "SUCCESS" },
  pending: { labelKey: "statusPending", statusValue: "PENDING" },
  processing: { labelKey: "statusProcessing", statusValue: "PROCESSING" },
  cancelled: { labelKey: "statusCancelled", statusValue: "CANCEL" },
} as const

const { t, lang } = useTranslation()
const router = useRouter()

const orders = ref<OrderItem[]>([])
const loading = ref(true)
const page = ref(1)
const pageSize = ref(10)
const pageSizeOptions = [10, 30, 50, 100]
const totalOrders = ref(0)
const totalPages = ref(0)
const selectedBizType = ref("")
const selectedOrderStatus = ref("")
const invoiceLoading = ref<string | null>(null)
const paymentLoading = ref<string | null>(null)
const cancelLoading = ref<string | null>(null)
const orderPaymentDialogOpen = ref(false)
const orderPaymentSession = ref<{
  orderId: string
  bizType: string
  bizRefUlid: string
  source: string
  returnPath: string
} | null>(null)
const showPurchaseDialog = ref(false)
const selectedCourseName = ref("")
const selectedPipelineId = ref("")

const invoiceOpeningLabel = computed(() => (lang.value === "zh" ? "正在打开发票，请稍候..." : "Opening invoice. Please wait..."))
const orderTypeOptions = computed(() => [
  { value: "", label: lang.value === "zh" ? "\u5168\u90e8\u8ba2\u5355" : "All Orders" },
  { value: "PIPELINE_PAYMENT", label: orderTypeLabel("PIPELINE_PAYMENT") },
  { value: "STAGE_PAYMENT", label: orderTypeLabel("STAGE_PAYMENT") },
  { value: "COURSE_RETAKE_PAYMENT", label: orderTypeLabel("COURSE_RETAKE_PAYMENT") },
  { value: "PIPELINE_UNLOCK", label: orderTypeLabel("PIPELINE_UNLOCK") },
  { value: "CREDENTIAL_APPLICATION", label: orderTypeLabel("CREDENTIAL_APPLICATION") },
  { value: "BUNDLE_PURCHASE", label: orderTypeLabel("BUNDLE_PURCHASE") },
])
const orderStatusOptions = computed(() => [
  { value: "", label: lang.value === "zh" ? "\u5168\u90e8\u72b6\u6001" : "All Statuses" },
  { value: "WAIT_PIPELINE_PAYMENT", label: orderStatusFilterLabel("WAIT_PIPELINE_PAYMENT") },
  { value: "WAIT_STAGE_PAYMENT", label: orderStatusFilterLabel("WAIT_STAGE_PAYMENT") },
  { value: "WAIT_RETAKE_PAYMENT", label: orderStatusFilterLabel("WAIT_RETAKE_PAYMENT") },
  { value: "WAIT_UNLOCK_PAYMENT", label: orderStatusFilterLabel("WAIT_UNLOCK_PAYMENT") },
  { value: "WAIT_BUNDLE_PAYMENT", label: orderStatusFilterLabel("WAIT_BUNDLE_PAYMENT") },
  { value: "WAIT_REVIEW_FEE_PAYMENT", label: orderStatusFilterLabel("WAIT_REVIEW_FEE_PAYMENT") },
  { value: "COMPLETED", label: orderStatusFilterLabel("COMPLETED") },
  { value: "CANCELLED", label: orderStatusFilterLabel("CANCELLED") },
  { value: "FAILED", label: orderStatusFilterLabel("FAILED") },
])

const payableOrderStatuses = new Set([
  "WAIT_PIPELINE_PAYMENT",
  "WAIT_STAGE_PAYMENT",
  "WAIT_RETAKE_PAYMENT",
  "WAIT_UNLOCK_PAYMENT",
  "WAIT_BUNDLE_PAYMENT",
  "WAIT_REVIEW_FEE_PAYMENT",
  "PENDING_PAYMENT",
  "WAIT_PAY",
  "UNPAID",
])

function orderStatusBadgeClass(order: OrderItem) {
  if (order.status === "completed" || order.rawStatus === "COMPLETED") {
    return "border-[#6CE9A6] bg-[#ECFDF3] text-[#027A48]"
  }
  return timelineStatusBadgeClassForStatus("MALL_ORDER", order.rawStatus)
}

function handleOrderClick(order: OrderItem) {
  if (order.bizType !== "PIPELINE_PAYMENT" || !order.pipelineId) return
  if (order.status !== "completed") {
    selectedCourseName.value = order.items.join(", ")
    selectedPipelineId.value = order.pipelineId
    showPurchaseDialog.value = true
  } else {
    router.push(`/certifications/${encodeURIComponent(order.pipelineId)}`)
  }
}

function canContinuePayment(order: OrderItem) {
  if (!order.bizType || !order.bizRefUlid) return false
  const rawStatus = String(order.rawStatus || "").toUpperCase()
  if (order.status === "completed" || rawStatus.includes("COMPLETED")) return false
  return payableOrderStatuses.has(rawStatus)
}

function canCancelOrder(order: OrderItem) {
  return Boolean(order.canCancel && order.id && !cancelLoading.value)
}

async function cancelOrder(order: OrderItem) {
  if (!canCancelOrder(order)) return
  const confirmed = window.confirm(t.value.orders.cancelOrderConfirm)
  if (!confirmed) return
  cancelLoading.value = order.id
  try {
    const res = await apiClient(`/api/orders/${encodeURIComponent(order.id)}/cancel`, { method: "POST" })
    if (res?.success === false) {
      toast.error(res?.message || t.value.orders.cancelOrderFailed)
      return
    }
    toast.success(res?.message || t.value.orders.cancelOrderSuccess)
    await fetchOrders(false)
  } catch (error) {
    console.error(error)
    toast.error(t.value.orders.cancelOrderFailed)
  } finally {
    if (cancelLoading.value === order.id) cancelLoading.value = null
  }
}

function continuePayment(order: OrderItem) {
  if (!canContinuePayment(order) || paymentLoading.value) return
  paymentLoading.value = order.id
  orderPaymentSession.value = {
    orderId: order.id,
    bizType: order.bizType,
    bizRefUlid: order.bizRefUlid,
    source: "orders",
    returnPath: "/orders",
  }
  orderPaymentDialogOpen.value = true
  window.setTimeout(() => {
    if (paymentLoading.value === order.id) paymentLoading.value = null
  }, 300)
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

function orderTypeLabel(bizType?: string) {
  const normalized = String(bizType || "").toUpperCase()
  const zh = lang.value === "zh"
  switch (normalized) {
    case "PIPELINE_PAYMENT":
      return zh ? "\u7ba1\u7ebf\u8ba2\u5355" : "Pipeline Order"
    case "STAGE_PAYMENT":
      return zh ? "\u9636\u6bb5\u8ba2\u5355" : "Stage Order"
    case "COURSE_RETAKE_PAYMENT":
      return zh ? "\u91cd\u8003\u8ba2\u5355" : "Retake Order"
    case "PIPELINE_UNLOCK":
      return zh ? "\u7ba1\u7ebf\u89e3\u9501\u8ba2\u5355" : "Pipeline Unlock Order"
    case "CREDENTIAL_APPLICATION":
      return zh ? "\u8d44\u683c\u7533\u8bf7\u8ba2\u5355" : "Credential Application Order"
    case "BUNDLE_PURCHASE":
      return zh ? "\u8ba4\u8bc1\u5957\u9910\u8ba2\u5355" : "Bundle Purchase Order"
    default:
      return normalized || (zh ? "\u5176\u4ed6\u8ba2\u5355" : "Other Order")
  }
}

function orderStatusFilterLabel(status?: string) {
  const normalized = String(status || "").toUpperCase()
  const zh = lang.value === "zh"
  switch (normalized) {
    case "WAIT_PIPELINE_PAYMENT":
      return zh ? "\u7ba1\u7ebf\u5f85\u652f\u4ed8" : "Pipeline Payment Pending"
    case "WAIT_STAGE_PAYMENT":
      return zh ? "\u9636\u6bb5\u5f85\u652f\u4ed8" : "Stage Payment Pending"
    case "WAIT_RETAKE_PAYMENT":
      return zh ? "\u91cd\u8003\u5f85\u652f\u4ed8" : "Retake Payment Pending"
    case "WAIT_UNLOCK_PAYMENT":
      return zh ? "\u89e3\u9501\u5f85\u652f\u4ed8" : "Unlock Payment Pending"
    case "WAIT_BUNDLE_PAYMENT":
      return zh ? "\u5957\u9910\u5f85\u652f\u4ed8" : "Bundle Payment Pending"
    case "WAIT_REVIEW_FEE_PAYMENT":
      return zh ? "\u5ba1\u6838\u8d39\u5f85\u652f\u4ed8" : "Review Fee Payment Pending"
    case "COMPLETED":
      return zh ? "\u5df2\u5b8c\u6210" : "Completed"
    case "CANCELLED":
      return zh ? "\u5df2\u53d6\u6d88" : "Cancelled"
    case "FAILED":
      return zh ? "\u5931\u8d25" : "Failed"
    default:
      return normalized || (zh ? "\u5168\u90e8\u72b6\u6001" : "All Statuses")
  }
}

async function fetchOrders(showLoading = true, suppressErrorToast = false) {
  if (showLoading) loading.value = true
  try {
    const params = new URLSearchParams({
      page: String(page.value),
      page_size: String(pageSize.value),
    })
    if (selectedBizType.value) params.set("biz_type", selectedBizType.value)
    if (selectedOrderStatus.value) params.set("status", selectedOrderStatus.value)
    const res = await apiClient(`/api/orders?${params.toString()}`, { suppressErrorToast })
    totalOrders.value = Number(res.total_orders || 0)
    totalPages.value = Number(res.total_pages || 0)
    if (Array.isArray(res.orders)) {
      orders.value = res.orders.map((o: any) => ({
        id: o.order_id,
        invoiceOrderId: o.pay_order_ulid || o.pipeline_pay_order_ulid || "",
        canViewInvoice: Boolean(o.pay_order_ulid || o.pipeline_pay_order_ulid || o.can_view_invoice),
        items: [o.product_name || orderTypeLabel(o.biz_type)],
        date: formatBackendDateMinute(o.created_at),
        currency: (o.currency || "USD").toUpperCase(),
        bizType: o.biz_type || "",
        bizRefUlid: o.biz_ref_ulid || "",
        amount: o.amount > 0 ? formatMoney(o.amount, o.currency || "USD") : (lang.value === "zh" ? "免费" : "Free"),
        status: (o.status in statusConfig ? o.status : "pending") as OrderStatus,
        rawStatus: o.raw_status,
        pipelineId: o.pipeline_id,
        paymentMethod: o.payment_method,
        canCancel: Boolean(o.can_cancel),
      }))
    } else {
      orders.value = []
    }
  } catch (err) {
    console.error("Failed to fetch orders:", err)
    orders.value = []
    totalOrders.value = 0
    totalPages.value = 0
  } finally {
    if (showLoading) loading.value = false
  }
}

function changeOrderType(value: string) {
  selectedBizType.value = value
  page.value = 1
  void fetchOrders()
}

function changeOrderStatus(value: string) {
  selectedOrderStatus.value = value
  page.value = 1
  void fetchOrders()
}

function handlePaginationChange() {
  if (loading.value) return
  void fetchOrders()
}

const ordersPolling = usePolling(() => fetchOrders(false, true))

onMounted(() => {
  void fetchOrders()
  ordersPolling.start()
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
            <label class="sr-only" for="order-type-filter">{{ lang === 'zh' ? '订单类型' : 'Order type' }}</label>
            <select
              id="order-type-filter"
              :value="selectedBizType"
              class="h-10 w-full rounded-lg border border-slate-200 bg-white px-3 text-sm font-medium text-slate-700 shadow-sm outline-none transition-colors hover:border-primary/40 focus:border-primary focus:ring-2 focus:ring-primary/15 sm:w-48"
              @change="changeOrderType(($event.target as HTMLSelectElement).value)"
            >
              <option v-for="option in orderTypeOptions" :key="option.value || 'ALL_TYPES'" :value="option.value">
                {{ option.label }}
              </option>
            </select>
            <label class="sr-only" for="order-status-filter">{{ lang === 'zh' ? '订单状态' : 'Order status' }}</label>
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
      <div v-else-if="orders.length === 0" class="flex flex-col items-center justify-center px-4 py-14 text-center">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10"><Package class="h-8 w-8 text-primary" /></div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.orders.noOrders }}</h3>
        <p class="max-w-md text-sm text-muted-foreground">{{ t.orders.noOrdersDesc }}</p>
      </div>
      <div v-else>
        <div v-for="order in orders" :key="order.id" @click="handleOrderClick(order)" class="order-row group flex cursor-pointer flex-col gap-3 border-b border-slate-100 px-4 py-4 transition-all duration-200 hover:bg-primary/10 md:flex-row md:items-center md:justify-between">
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
          <div class="grid w-full grid-cols-[1fr_auto_auto_auto_auto_auto] items-center gap-x-3 gap-y-3 pl-16 md:w-auto md:shrink-0 md:grid-cols-[96px_96px_36px_36px_36px_20px] md:gap-x-5 md:pl-0">
            <div class="flex justify-start md:justify-center">
              <span class="badge text-xs" :class="orderStatusBadgeClass(order)">
                {{ timelineStatusLabelWithDiagnostics(t, 'MALL_ORDER', order.rawStatus) }}
              </span>
            </div>
            <div class="text-right"><p class="text-lg font-semibold text-card-foreground">{{ order.amount }}</p></div>
            <button v-if="canContinuePayment(order)" @click.stop="continuePayment(order)" class="flex h-9 w-9 items-center justify-center rounded-lg text-primary transition-colors hover:bg-primary/10" :title="lang === 'zh' ? '继续支付' : 'Continue payment'">
              <Loader2 v-if="paymentLoading === order.id" class="h-4 w-4 animate-spin" />
              <CreditCard v-else class="h-4 w-4" />
              <span class="sr-only">{{ lang === 'zh' ? '继续支付' : 'Continue payment' }}</span>
            </button>
            <span v-else class="h-9 w-9" />
            <button v-if="canCancelOrder(order)" @click.stop="cancelOrder(order)" class="flex h-9 w-9 items-center justify-center rounded-lg text-red-600 transition-colors hover:bg-red-50" :title="t.orders.cancelOrder">
              <Loader2 v-if="cancelLoading === order.id" class="h-4 w-4 animate-spin" />
              <XCircle v-else class="h-4 w-4" />
              <span class="sr-only">{{ t.orders.cancelOrder }}</span>
            </button>
            <span v-else class="h-9 w-9" />
            <button v-if="order.canViewInvoice" @click.stop="viewInvoice(order.invoiceOrderId)" class="flex h-9 w-9 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-primary/10 hover:text-primary" title="View Invoice">
              <Loader2 v-if="invoiceLoading === order.invoiceOrderId" class="h-4 w-4 animate-spin text-primary" />
              <FileText v-else class="h-4 w-4" />
              <span class="sr-only">{{ invoiceOpeningLabel }}</span>
            </button>
            <span v-else class="h-9 w-9" />

            <ChevronRight class="h-5 w-5 text-muted-foreground transition-transform group-hover:translate-x-1" />
          </div>
        </div>
        <AppPagination
          v-model:page="page"
          v-model:page-size="pageSize"
          :total="totalOrders"
          :total-pages="totalPages"
          :page-size-options="pageSizeOptions"
          :disabled="loading"
          :locale="lang"
          @page-change="handlePaginationChange"
        />
      </div>
    </div>

      </main>
    </div>

    <PurchaseDialog
      v-if="showPurchaseDialog"
      v-model:open="showPurchaseDialog"
      :course-name="selectedCourseName"
      :pipeline-id="selectedPipelineId"
    />
    <PaymentSessionDialog
      v-if="orderPaymentSession"
      v-model:open="orderPaymentDialogOpen"
      :title="lang === 'zh' ? '继续支付' : 'Continue payment'"
      :subtitle="orderPaymentSession.orderId"
      :biz-type="orderPaymentSession.bizType"
      :biz-ref-ulid="orderPaymentSession.bizRefUlid"
      :order-id="orderPaymentSession.orderId"
      :source="orderPaymentSession.source"
      :return-path="orderPaymentSession.returnPath"
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
