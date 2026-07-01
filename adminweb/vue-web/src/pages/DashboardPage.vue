<script setup lang="ts">
import { BarChart3, CreditCard, Loader2, RefreshCw, Users } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { badgeClass } from "@/lib/status"

type StageBucket = {
  stage_id: string
  status: string
  count: number
}

type RevenueItem = {
  currency: string
  amount_minor: number
  order_count: number
}

type DashboardData = {
  candidate_total: number
  stage_buckets: StageBucket[]
  today_revenue: RevenueItem[]
  generated_at: string
}

const loading = ref(false)
const data = ref<DashboardData | null>(null)

const totalPipelines = computed(() => data.value?.stage_buckets.reduce((sum, item) => sum + Number(item.count || 0), 0) || 0)
const totalPaidOrders = computed(() => data.value?.today_revenue.reduce((sum, item) => sum + Number(item.order_count || 0), 0) || 0)
const revenueText = computed(() => {
  const items = data.value?.today_revenue || []
  if (!items.length) return "暂无收款"
  return items.map((item) => `${item.currency} ${(Number(item.amount_minor || 0) / 100).toFixed(2)}`).join(" / ")
})

function stageLabel(stageId: string) {
  return stageId === "未进入阶段" ? stageId : stageId
}

async function loadDashboard() {
  loading.value = true
  try {
    data.value = await apiClient<DashboardData>("/api/dashboard/ops")
  } catch (err) {
    console.error(err)
    toast.error("运营看板加载失败")
  } finally {
    loading.value = false
  }
}

onMounted(loadDashboard)
</script>

<template>
  <main class="mx-auto max-w-7xl px-6 py-10">
    <header class="mb-8 flex flex-wrap items-center justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">运营看板</h1>
        <p class="mt-2 text-slate-500">查看考生总数、阶段分布和今日收款金额。</p>
      </div>
      <button
        class="inline-flex items-center gap-2 rounded-2xl border border-slate-300 bg-white px-5 py-3 font-bold shadow-sm transition hover:border-slate-500 disabled:opacity-50"
        type="button"
        :disabled="loading"
        @click="loadDashboard"
      >
        <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
        <RefreshCw v-else class="h-4 w-4" />
        刷新
      </button>
    </header>

    <section class="grid gap-5 md:grid-cols-3">
      <article class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm">
        <div class="flex items-center justify-between">
          <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-sky-50 text-sky-700">
            <Users class="h-6 w-6" />
          </div>
          <span class="rounded-full bg-slate-100 px-3 py-1 text-xs font-bold text-slate-500">Candidates</span>
        </div>
        <p class="mt-6 text-sm font-bold text-slate-500">考生总数</p>
        <p class="mt-2 text-4xl font-black">{{ data?.candidate_total ?? "-" }}</p>
      </article>

      <article class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm">
        <div class="flex items-center justify-between">
          <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-indigo-50 text-indigo-700">
            <BarChart3 class="h-6 w-6" />
          </div>
          <span class="rounded-full bg-slate-100 px-3 py-1 text-xs font-bold text-slate-500">Pipelines</span>
        </div>
        <p class="mt-6 text-sm font-bold text-slate-500">管线实例数</p>
        <p class="mt-2 text-4xl font-black">{{ totalPipelines }}</p>
      </article>

      <article class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm">
        <div class="flex items-center justify-between">
          <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-emerald-50 text-emerald-700">
            <CreditCard class="h-6 w-6" />
          </div>
          <span class="rounded-full bg-slate-100 px-3 py-1 text-xs font-bold text-slate-500">Today</span>
        </div>
        <p class="mt-6 text-sm font-bold text-slate-500">今日收款金额</p>
        <p class="mt-2 text-3xl font-black">{{ revenueText }}</p>
        <p class="mt-2 text-sm text-slate-500">已支付订单 {{ totalPaidOrders }} 笔</p>
      </article>
    </section>

    <section class="mt-6 grid gap-6 lg:grid-cols-[1.15fr_0.85fr]">
      <article class="rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="border-b border-slate-100 p-6">
          <h2 class="text-xl font-black">阶段分布</h2>
          <p class="mt-1 text-sm text-slate-500">按当前阶段和管线状态统计。</p>
        </div>
        <div v-if="loading" class="flex h-48 items-center justify-center text-slate-400">
          <Loader2 class="mr-2 h-5 w-5 animate-spin" />
          加载中
        </div>
        <div v-else-if="!data?.stage_buckets.length" class="flex h-48 items-center justify-center text-slate-400">
          暂无阶段数据
        </div>
        <div v-else class="divide-y divide-slate-100">
          <div v-for="item in data.stage_buckets" :key="`${item.stage_id}-${item.status}`" class="flex items-center justify-between gap-4 p-5">
            <div class="min-w-0">
              <p class="break-all text-base font-black">{{ stageLabel(item.stage_id) }}</p>
              <span class="mt-2 inline-flex rounded-full border px-3 py-1 text-xs font-bold" :class="badgeClass(item.status)">
                {{ item.status || "未知状态" }}
              </span>
            </div>
            <div class="text-right">
              <p class="text-3xl font-black">{{ item.count }}</p>
              <p class="text-xs text-slate-500">人/管线</p>
            </div>
          </div>
        </div>
      </article>

      <article class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm">
        <h2 class="text-xl font-black">数据口径</h2>
        <div class="mt-4 space-y-4 text-sm leading-6 text-slate-600">
          <p>考生总数：来自 Casdoor 用户列表，并通过 gmid 映射到平台 ULID 后统计。</p>
          <p>阶段分布：来自 gprog 管线实例列表，按当前阶段和状态聚合。</p>
          <p>今日收款：来自 gmall 订单列表，只统计今天创建且支付状态为 PAID 的订单。</p>
        </div>
      </article>
    </section>
  </main>
</template>
