<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useRouter } from "vue-router"
import { CheckCircle2, ChevronRight, FileText, Loader2, Package, Receipt, ShoppingCart } from "lucide-vue-next"
import { timelineStatusBadgeClassForStatus, timelineStatusLabelWithDiagnostics } from "@/lib/status-labels"
import AppShell from "@/components/AppShell.vue"
import PurchaseDialog from "@/components/PurchaseDialog.vue"
import { apiClient } from "@/lib/apiClient"
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
  status: OrderStatus
  rawStatus: string
  pipelineId: string
  paymentMethod: string
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
const totalSpent = ref(0)
const completedCount = ref(0)
const loading = ref(true)
const page = ref(1)
const pageSize = 10
const totalOrders = ref(0)
const totalPages = ref(0)
const invoiceLoading = ref<string | null>(null)
const showPurchaseDialog = ref(false)
const selectedCourseName = ref("")
const selectedPipelineId = ref("")

const displayCurrency = computed(() => orders.value.find((order) => order.currency)?.currency || "USD")
const totalSpentLabel = computed(() => formatMoney(totalSpent.value, displayCurrency.value))
const invoiceOpeningLabel = computed(() => (lang.value === "zh" ? "正在打开发票，请稍候..." : "Opening invoice. Please wait..."))
const orderRangeLabel = computed(() => {
  if (totalOrders.value === 0) return "0 / 0"
  const start = (page.value - 1) * pageSize + 1
  const end = Math.min(page.value * pageSize, totalOrders.value)
  return `${start}-${end} / ${totalOrders.value}`
})

function handleOrderClick(order: OrderItem) {
  if (order.status !== "completed" && order.pipelineId) {
    selectedCourseName.value = order.items.join(", ")
    selectedPipelineId.value = order.pipelineId
    showPurchaseDialog.value = true
  } else if (order.status === "completed" && order.pipelineId) {
    router.push(`/certifications/${encodeURIComponent(order.pipelineId)}`)
  }
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

async function fetchOrders() {
  loading.value = true
  try {
    const params = new URLSearchParams({
      page: String(page.value),
      page_size: String(pageSize),
    })
    const res = await apiClient(`/api/orders?${params.toString()}`)
    totalSpent.value = Number(res.total_amount || 0)
    completedCount.value = Number(res.completed || 0)
    totalOrders.value = Number(res.total_orders || 0)
    totalPages.value = Number(res.total_pages || 0)
    if (Array.isArray(res.orders)) {
      orders.value = res.orders.map((o: any) => ({
        id: o.order_id,
        invoiceOrderId: o.pipeline_pay_order_ulid || "",
        canViewInvoice: Boolean(o.can_view_invoice && o.pipeline_pay_order_ulid),
        items: [o.product_name],
        date: o.created_at,
        currency: (o.currency || "USD").toUpperCase(),
        amount: o.amount > 0 ? formatMoney(o.amount, o.currency || "USD") : "-",
        status: (o.status in statusConfig ? o.status : "pending") as OrderStatus,
        rawStatus: o.raw_status,
        pipelineId: o.pipeline_id,
        paymentMethod: o.payment_method,
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
    loading.value = false
  }
}

function goToPage(nextPage: number) {
  if (loading.value) return
  if (nextPage < 1 || (totalPages.value > 0 && nextPage > totalPages.value)) return
  page.value = nextPage
  fetchOrders()
}

onMounted(() => {
  fetchOrders()
})
</script>

<template>
  <AppShell content-class="p-4">
    <div v-if="invoiceLoading" class="fixed right-5 top-5 z-50 flex items-center gap-3 rounded-2xl border border-emerald-100 bg-white px-4 py-3 text-sm font-semibold text-slate-700 shadow-[0_16px_40px_rgba(15,74,82,0.14)]">
      <Loader2 class="h-4 w-4 animate-spin text-emerald-500" />
      <span>{{ invoiceOpeningLabel }}</span>
    </div>

    <div class="mb-4 px-1 py-3 md:py-5">
      <div class="flex items-start gap-3">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-accent text-primary">
          <Receipt class="h-6 w-6" />
        </div>
        <div>
          <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.orders.title }}</h1>
          <p class="mt-2 text-muted-foreground">{{ t.orders.subtitle }}</p>
        </div>
      </div>
    </div>

    <div class="mb-4 grid gap-4 sm:grid-cols-3">
      <div class="group relative overflow-hidden rounded-[16px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all hover:-translate-y-0.5 hover:ring-primary/25 hover:shadow-md hover:shadow-primary/10">
        <div class="absolute left-0 top-0 h-full w-1 bg-primary" />
        <div class="flex items-center gap-4">
          <div class="flex h-12 w-12 items-center justify-center rounded-lg bg-primary/10 transition-transform group-hover:scale-105"><ShoppingCart class="h-6 w-6 text-primary" /></div>
          <div><p class="text-2xl font-bold text-card-foreground">{{ totalOrders }}</p><p class="text-sm text-muted-foreground">{{ t.orders.totalOrders }}</p></div>
        </div>
      </div>
      <div class="group relative overflow-hidden rounded-[16px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all hover:-translate-y-0.5 hover:ring-primary/25 hover:shadow-md hover:shadow-primary/10">
        <div class="absolute left-0 top-0 h-full w-1 bg-emerald-500/60" />
        <div class="flex items-center gap-4">
          <div class="flex h-12 w-12 items-center justify-center rounded-lg bg-emerald-100 transition-transform group-hover:scale-105"><CheckCircle2 class="h-6 w-6 text-emerald-600" /></div>
          <div><p class="text-2xl font-bold text-card-foreground">{{ completedCount }}</p><p class="text-sm text-muted-foreground">{{ t.orders.completed }}</p></div>
        </div>
      </div>
      <div class="group relative overflow-hidden rounded-[16px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all hover:-translate-y-0.5 hover:ring-primary/25 hover:shadow-md hover:shadow-primary/10">
        <div class="absolute left-0 top-0 h-full w-1 bg-amber-500/60" />
        <div class="flex items-center gap-4">
          <div class="flex h-12 w-12 items-center justify-center rounded-lg bg-amber-100 transition-transform group-hover:scale-105"><Receipt class="h-6 w-6 text-amber-600" /></div>
          <div><p class="text-2xl font-bold text-card-foreground">{{ totalSpentLabel }}</p><p class="text-sm text-muted-foreground">{{ t.orders.totalSpent }}</p></div>
        </div>
      </div>
    </div>

    <div class="overflow-hidden rounded-[16px] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <div class="flex flex-col gap-3 bg-white px-4 py-4 sm:flex-row sm:items-center sm:justify-between">
        <div class="flex items-center gap-3">
          <div class="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10"><Receipt class="h-4 w-4 text-primary" /></div>
          <h2 class="font-semibold text-card-foreground">{{ t.orders.orderHistory }}</h2>
        </div>
        <div class="text-sm text-muted-foreground">{{ orderRangeLabel }}</div>
      </div>
      <div v-if="loading" class="flex items-center justify-center gap-2 py-16 text-muted-foreground"><Loader2 class="h-5 w-5 animate-spin" /> {{ t.common.loading }}</div>
      <div v-else-if="orders.length === 0" class="flex flex-col items-center justify-center px-4 py-14 text-center">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10"><Package class="h-8 w-8 text-primary" /></div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.orders.noOrders }}</h3>
        <p class="max-w-md text-sm text-muted-foreground">{{ t.orders.noOrdersDesc }}</p>
      </div>
      <div v-else class="space-y-2">
        <div v-for="order in orders" :key="order.id" @click="handleOrderClick(order)" class="group flex items-center justify-between px-4 py-4 transition-colors hover:bg-primary/10 cursor-pointer">
          <div class="flex items-center gap-4">
            <div class="flex h-12 w-12 items-center justify-center rounded-lg bg-primary/10"><Package class="h-6 w-6 text-primary" /></div>
            <div><h3 class="mb-1 font-medium text-card-foreground">{{ order.items.join(", ") }}</h3><p class="text-sm text-muted-foreground">{{ order.date }}</p></div>
          </div>
          <div class="grid shrink-0 grid-cols-[96px_86px_36px_20px] items-center gap-3">
            <div class="flex justify-center">
              <span class="badge text-xs" :class="timelineStatusBadgeClassForStatus('MALL_ORDER', order.rawStatus)">
                {{ timelineStatusLabelWithDiagnostics(t, 'MALL_ORDER', order.rawStatus) }}
              </span>
            </div>
            <div class="text-right"><p class="text-lg font-semibold text-card-foreground">{{ order.amount }}</p></div>
            <button v-if="order.canViewInvoice" @click.stop="viewInvoice(order.invoiceOrderId)" class="flex h-9 w-9 items-center justify-center rounded-lg hover:bg-primary/10 hover:text-primary transition-colors text-muted-foreground" title="查看发票 / View Invoice">
              <Loader2 v-if="invoiceLoading === order.invoiceOrderId" class="h-4 w-4 animate-spin text-primary" />
              <FileText v-else class="h-4 w-4" />
              <span class="sr-only">{{ invoiceOpeningLabel }}</span>
            </button>
            <span v-else class="h-9 w-9" />

            <ChevronRight class="h-5 w-5 text-muted-foreground transition-transform group-hover:translate-x-1" />
          </div>
        </div>
        <div class="flex items-center justify-between border-t border-slate-100 px-4 py-3 text-sm text-muted-foreground">
          <span>{{ orderRangeLabel }}</span>
          <div class="flex items-center gap-2">
            <button class="rounded-lg border border-slate-200 px-3 py-1.5 font-medium transition-colors hover:border-primary hover:text-primary disabled:cursor-not-allowed disabled:opacity-50" :disabled="page <= 1 || loading" @click="goToPage(page - 1)">
              {{ lang === "zh" ? "上一页" : "Previous" }}
            </button>
            <span class="min-w-20 text-center">{{ page }} / {{ totalPages || 1 }}</span>
            <button class="rounded-lg border border-slate-200 px-3 py-1.5 font-medium transition-colors hover:border-primary hover:text-primary disabled:cursor-not-allowed disabled:opacity-50" :disabled="totalPages === 0 || page >= totalPages || loading" @click="goToPage(page + 1)">
              {{ lang === "zh" ? "下一页" : "Next" }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <PurchaseDialog
      v-if="showPurchaseDialog"
      v-model:open="showPurchaseDialog"
      :course-name="selectedCourseName"
      :pipeline-id="selectedPipelineId"
    />
  </AppShell>
</template>
