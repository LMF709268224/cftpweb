<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { CheckCircle2, ChevronRight, Loader2, Package, Receipt, ShoppingCart } from "lucide-vue-next"
import { statusBadgeClassForStatusValue } from "@/lib/status-labels"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"

type OrderItem = { id: string; items: string[]; date: string; amount: string; status: keyof typeof statusConfig; paymentMethod: string }

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

function orderStatusLabel(order: OrderItem) {
  const labels = t.value.orders as Record<string, string>
  return labels[statusConfig[order.status].labelKey] || order.status
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
  <AppShell>
    <div class="mb-4 overflow-hidden rounded-3xl bg-card shadow-sm ring-1 ring-border/50">
      <div class="bg-[#eef8fa] p-4">
        <div class="mb-3 inline-flex items-center gap-2 rounded-full border border-primary/20 bg-white px-3 py-1 text-xs font-medium text-primary">
          <ShoppingCart class="h-3.5 w-3.5" />
          {{ t.sidebar.orders }}
        </div>
        <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.orders.title }}</h1>
        <p class="mt-2 text-muted-foreground">{{ t.orders.subtitle }}</p>
      </div>
    </div>

    <div class="mb-4 grid gap-4 sm:grid-cols-3">
      <div class="group relative overflow-hidden rounded-2xl bg-card p-4 shadow-sm ring-1 ring-border/50 transition-all hover:-translate-y-0.5 hover:ring-primary/25 hover:shadow-md hover:shadow-primary/10">
        <div class="absolute left-0 top-0 h-full w-1 bg-primary" />
        <div class="flex items-center gap-4">
          <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10 transition-transform group-hover:scale-105"><ShoppingCart class="h-6 w-6 text-primary" /></div>
          <div><p class="text-2xl font-bold text-card-foreground">{{ orders.length }}</p><p class="text-sm text-muted-foreground">{{ t.orders.totalOrders }}</p></div>
        </div>
      </div>
      <div class="group relative overflow-hidden rounded-2xl bg-card p-4 shadow-sm ring-1 ring-border/50 transition-all hover:-translate-y-0.5 hover:ring-primary/25 hover:shadow-md hover:shadow-primary/10">
        <div class="absolute left-0 top-0 h-full w-1 bg-emerald-500/60" />
        <div class="flex items-center gap-4">
          <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-emerald-100 transition-transform group-hover:scale-105"><CheckCircle2 class="h-6 w-6 text-emerald-600" /></div>
          <div><p class="text-2xl font-bold text-card-foreground">{{ completedCount }}</p><p class="text-sm text-muted-foreground">{{ t.orders.completed }}</p></div>
        </div>
      </div>
      <div class="group relative overflow-hidden rounded-2xl bg-card p-4 shadow-sm ring-1 ring-border/50 transition-all hover:-translate-y-0.5 hover:ring-primary/25 hover:shadow-md hover:shadow-primary/10">
        <div class="absolute left-0 top-0 h-full w-1 bg-amber-500/60" />
        <div class="flex items-center gap-4">
          <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-amber-100 transition-transform group-hover:scale-105"><Receipt class="h-6 w-6 text-amber-600" /></div>
          <div><p class="text-2xl font-bold text-card-foreground">{{ totalSpentLabel }}</p><p class="text-sm text-muted-foreground">{{ t.orders.totalSpent }}</p></div>
        </div>
      </div>
    </div>

    <div class="overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
      <div class="flex items-center gap-3 border-b border-border bg-[#f7fbfc] px-4 py-4">
        <div class="flex h-9 w-9 items-center justify-center rounded-xl bg-primary/10"><Receipt class="h-4 w-4 text-primary" /></div>
        <h2 class="font-semibold text-card-foreground">{{ t.orders.orderHistory }}</h2>
      </div>
      <div v-if="loading" class="flex items-center justify-center gap-2 py-16 text-muted-foreground"><Loader2 class="h-5 w-5 animate-spin" /> {{ t.common.loading }}</div>
      <div v-else-if="orders.length === 0" class="flex flex-col items-center justify-center px-4 py-14 text-center">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-primary/10"><Package class="h-8 w-8 text-primary" /></div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.orders.noOrders }}</h3>
        <p class="max-w-md text-sm text-muted-foreground">{{ t.orders.noOrdersDesc }}</p>
      </div>
      <div v-else class="divide-y divide-border">
        <div v-for="order in orders" :key="order.id" class="group flex items-center justify-between px-4 py-4 transition-colors hover:bg-muted/50">
          <div class="flex items-center gap-4">
            <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10"><Package class="h-6 w-6 text-primary" /></div>
            <div><h3 class="mb-1 font-medium text-card-foreground">{{ order.items.join(", ") }}</h3><p class="text-sm text-muted-foreground">{{ order.date }}</p></div>
          </div>
          <div class="flex items-center gap-4">
            <span :class="['badge', statusBadgeClassForStatusValue(statusConfig[order.status].statusValue)]">{{ orderStatusLabel(order) }}</span>
            <div class="min-w-[80px] text-right"><p class="text-lg font-semibold text-card-foreground">{{ order.amount }}</p></div>
            <ChevronRight class="h-5 w-5 text-muted-foreground transition-transform group-hover:translate-x-1" />
          </div>
        </div>
      </div>
    </div>
  </AppShell>
</template>
