<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { CheckCircle2, ChevronRight, Loader2, Package, Receipt, ShoppingCart, FileText } from "lucide-vue-next"
import { timelineStatusLabelWithDiagnostics, timelineStatusBadgeClassForStatus } from "@/lib/status-labels"
import AppShell from "@/components/AppShell.vue"
import PurchaseDialog from "@/components/PurchaseDialog.vue"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"

type OrderItem = { id: string; items: string[]; date: string; amount: string; status: keyof typeof statusConfig; rawStatus: string; pipelineId: string; paymentMethod: string }

const statusConfig = {
  completed: { labelKey: "statusCompleted", statusValue: "SUCCESS" },
  pending: { labelKey: "statusPending", statusValue: "PENDING" },
  processing: { labelKey: "statusProcessing", statusValue: "PROCESSING" },
  cancelled: { labelKey: "statusCancelled", statusValue: "CANCEL" },
} as const

const { t } = useTranslation()
const orders = ref<OrderItem[]>([])
const totalSpent = ref(0)
const completedCount = ref(0)
const loading = ref(true)
const totalSpentLabel = computed(() => `¥${totalSpent.value.toLocaleString()}`)

import { useRouter } from "vue-router"

const showPurchaseDialog = ref(false)
const selectedCourseName = ref("")
const selectedPipelineId = ref("")
const router = useRouter()

function handleOrderClick(order: OrderItem) {
  if (order.status !== "completed" && order.pipelineId) {
    selectedCourseName.value = order.items.join(", ")
    selectedPipelineId.value = order.pipelineId
    showPurchaseDialog.value = true
  } else if (order.status === "completed" && order.pipelineId) {
    router.push(`/courses/detail?id=${encodeURIComponent(order.pipelineId)}`)
  }
}

const invoiceLoading = ref<string | null>(null)

async function viewInvoice(orderId: string) {
  if (invoiceLoading.value) return
  try {
    invoiceLoading.value = orderId
    const res = await apiClient(`/api/invoices/${orderId}`)
    if (res.invoice_url) {
      window.open(res.invoice_url, '_blank')
    } else {
      alert("发票还在生成中或不可用 (Invoice not available yet)")
    }
  } catch (err) {
    console.error("Failed to fetch invoice:", err)
    alert("获取发票失败 (Failed to fetch invoice)")
  } finally {
    invoiceLoading.value = null
  }
}

onMounted(async () => {
  try {
    const res = await apiClient("/api/orders")
    totalSpent.value = res.total_amount || 0
    completedCount.value = res.completed || 0
    if (Array.isArray(res.orders)) {
      orders.value = res.orders.map((o: any) => ({
        id: o.order_id,
        items: [o.product_name],
        date: o.created_at,
        amount: o.amount > 0 ? `¥${o.amount.toLocaleString()}` : "-",
        status: (o.status in statusConfig ? o.status : "pending") as keyof typeof statusConfig,
        rawStatus: o.raw_status,
        pipelineId: o.pipeline_id,
        paymentMethod: o.payment_method,
      }))
    }
  } catch (err) {
    console.error("Failed to fetch orders:", err)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <AppShell content-class="p-4">
    <div class="mb-4 overflow-hidden rounded-[22px] bg-white shadow-[0_12px_30px_rgba(15,74,82,0.06)]">
      <div class="bg-gradient-to-r from-[#ecfbf7] via-white to-[#f4fbff] p-4">
        <div class="mb-3 inline-flex items-center gap-2 rounded-full bg-primary/10 px-3 py-1 text-xs font-semibold text-primary">
          <ShoppingCart class="h-3.5 w-3.5" />
          {{ t.sidebar.orders }}
        </div>
        <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.orders.title }}</h1>
        <p class="mt-2 text-muted-foreground">{{ t.orders.subtitle }}</p>
      </div>
    </div>

    <div class="mb-4 grid gap-4 sm:grid-cols-3">
      <div class="group relative overflow-hidden rounded-[22px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all hover:-translate-y-0.5 hover:ring-primary/25 hover:shadow-md hover:shadow-primary/10">
        <div class="absolute left-0 top-0 h-full w-1 bg-primary" />
        <div class="flex items-center gap-4">
          <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10 transition-transform group-hover:scale-105"><ShoppingCart class="h-6 w-6 text-primary" /></div>
          <div><p class="text-2xl font-bold text-card-foreground">{{ orders.length }}</p><p class="text-sm text-muted-foreground">{{ t.orders.totalOrders }}</p></div>
        </div>
      </div>
      <div class="group relative overflow-hidden rounded-[22px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all hover:-translate-y-0.5 hover:ring-primary/25 hover:shadow-md hover:shadow-primary/10">
        <div class="absolute left-0 top-0 h-full w-1 bg-emerald-500/60" />
        <div class="flex items-center gap-4">
          <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-emerald-100 transition-transform group-hover:scale-105"><CheckCircle2 class="h-6 w-6 text-emerald-600" /></div>
          <div><p class="text-2xl font-bold text-card-foreground">{{ completedCount }}</p><p class="text-sm text-muted-foreground">{{ t.orders.completed }}</p></div>
        </div>
      </div>
      <div class="group relative overflow-hidden rounded-[22px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all hover:-translate-y-0.5 hover:ring-primary/25 hover:shadow-md hover:shadow-primary/10">
        <div class="absolute left-0 top-0 h-full w-1 bg-amber-500/60" />
        <div class="flex items-center gap-4">
          <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-amber-100 transition-transform group-hover:scale-105"><Receipt class="h-6 w-6 text-amber-600" /></div>
          <div><p class="text-2xl font-bold text-card-foreground">{{ totalSpentLabel }}</p><p class="text-sm text-muted-foreground">{{ t.orders.totalSpent }}</p></div>
        </div>
      </div>
    </div>

    <div class="overflow-hidden rounded-[22px] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <div class="flex items-center gap-3 bg-white px-4 py-4">
        <div class="flex h-9 w-9 items-center justify-center rounded-xl bg-primary/10"><Receipt class="h-4 w-4 text-primary" /></div>
        <h2 class="font-semibold text-card-foreground">{{ t.orders.orderHistory }}</h2>
      </div>
      <div v-if="loading" class="flex items-center justify-center gap-2 py-16 text-muted-foreground"><Loader2 class="h-5 w-5 animate-spin" /> {{ t.common.loading }}</div>
      <div v-else-if="orders.length === 0" class="flex flex-col items-center justify-center px-4 py-14 text-center">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-primary/10"><Package class="h-8 w-8 text-primary" /></div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.orders.noOrders }}</h3>
        <p class="max-w-md text-sm text-muted-foreground">{{ t.orders.noOrdersDesc }}</p>
      </div>
      <div v-else class="space-y-2">
        <div v-for="order in orders" :key="order.id" @click="handleOrderClick(order)" class="group flex items-center justify-between px-4 py-4 transition-colors hover:bg-primary/10 cursor-pointer">
          <div class="flex items-center gap-4">
            <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10"><Package class="h-6 w-6 text-primary" /></div>
            <div><h3 class="mb-1 font-medium text-card-foreground">{{ order.items.join(", ") }}</h3><p class="text-sm text-muted-foreground">{{ order.date }}</p></div>
          </div>
          <div class="flex items-center gap-4">
            <span class="badge text-xs" :class="timelineStatusBadgeClassForStatus('MALL_ORDER', order.rawStatus)">
              {{ timelineStatusLabelWithDiagnostics(t, 'MALL_ORDER', order.rawStatus) }}
            </span>
            <div class="min-w-[80px] text-right"><p class="text-lg font-semibold text-card-foreground">{{ order.amount }}</p></div>
            
            <button v-if="order.status === 'completed'" @click.stop="viewInvoice(order.id)" class="flex h-9 w-9 items-center justify-center rounded-lg hover:bg-primary/10 hover:text-primary transition-colors text-muted-foreground" title="查看发票 / View Invoice">
              <Loader2 v-if="invoiceLoading === order.id" class="h-4 w-4 animate-spin text-primary" />
              <FileText v-else class="h-4 w-4" />
            </button>
            
            <ChevronRight class="h-5 w-5 text-muted-foreground transition-transform group-hover:translate-x-1" />
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
