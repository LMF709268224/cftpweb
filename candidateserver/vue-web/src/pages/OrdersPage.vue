<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { CheckCircle2, ChevronRight, Clock, Package, Receipt, ShoppingCart } from "lucide-vue-next"
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
    <div class="mb-8">
      <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.orders.title }}</h1>
      <p class="mt-1 text-muted-foreground">{{ t.orders.subtitle }}</p>
    </div>
    <div class="mb-8 grid gap-4 sm:grid-cols-3">
      <div class="flex items-center gap-4 rounded-xl border border-border bg-card p-5">
        <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10"><ShoppingCart class="h-6 w-6 text-primary" /></div>
        <div><p class="text-2xl font-bold text-card-foreground">{{ orders.length }}</p><p class="text-sm text-muted-foreground">{{ t.orders.totalOrders }}</p></div>
      </div>
      <div class="flex items-center gap-4 rounded-xl border border-border bg-card p-5">
        <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-blue-100"><CheckCircle2 class="h-6 w-6 text-black" /></div>
        <div><p class="text-2xl font-bold text-card-foreground">{{ completedCount }}</p><p class="text-sm text-muted-foreground">{{ t.orders.completed }}</p></div>
      </div>
      <div class="flex items-center gap-4 rounded-xl border border-border bg-card p-5">
        <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-yellow-100"><Receipt class="h-6 w-6 text-black" /></div>
        <div><p class="text-2xl font-bold text-card-foreground">{{ totalSpentLabel }}</p><p class="text-sm text-muted-foreground">{{ t.orders.totalSpent }}</p></div>
      </div>
    </div>
    <div class="overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
      <div class="flex items-center gap-3 border-b border-border px-6 py-4">
        <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10"><Receipt class="h-4 w-4 text-primary" /></div>
        <h2 class="font-semibold text-card-foreground">{{ t.orders.orderHistory }}</h2>
      </div>
      <div v-if="loading" class="flex items-center justify-center py-20 text-muted-foreground"><Clock class="mr-2 h-5 w-5 animate-spin" /> {{ t.common.loading }}</div>
      <div v-else-if="orders.length === 0" class="flex flex-col items-center justify-center px-6 py-14 text-center">
        <div class="mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-muted"><Package class="h-7 w-7 text-muted-foreground" /></div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.orders.noOrders }}</h3>
        <p class="max-w-md text-sm text-muted-foreground">{{ t.orders.noOrdersDesc }}</p>
      </div>
      <div v-else class="divide-y divide-border">
        <div v-for="order in orders" :key="order.id" class="group flex items-center justify-between p-6 transition-colors hover:bg-muted/50">
          <div class="flex items-center gap-4">
            <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-muted"><Package class="h-6 w-6 text-muted-foreground" /></div>
            <div><h3 class="mb-1 font-medium text-card-foreground">{{ order.items.join(", ") }}</h3><p class="text-sm text-muted-foreground">{{ order.date }}</p></div>
          </div>
          <div class="flex items-center gap-6">
            <span :class="['badge', statusBadgeClassForStatusValue(statusConfig[order.status].statusValue)]">{{ orderStatusLabel(order) }}</span>
            <div class="min-w-[80px] text-right"><p class="text-lg font-semibold text-card-foreground">{{ order.amount }}</p></div>
            <ChevronRight class="h-5 w-5 text-muted-foreground transition-transform group-hover:translate-x-1" />
          </div>
        </div>
      </div>
    </div>
  </AppShell>
</template>
