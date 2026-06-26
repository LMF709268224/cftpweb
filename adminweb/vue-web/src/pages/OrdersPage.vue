<script setup lang="ts">
import { ChevronRight, Loader2, RefreshCw, Search, Trash2 } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import {
  badgeClass,
  bizTypeOptions,
  labelFor,
  normalizeStatus,
  orderStatusOptions,
  paymentStatusOptions,
  pickFirst,
} from "@/lib/status"

const orders = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const loading = ref(false)
const purging = ref("")
const page = ref(1)
const total = ref(0)
const pageSize = 20

const candidateUlid = ref("")
const bizType = ref("")
const orderStatus = ref("")
const paymentStatus = ref("")

const canPrev = computed(() => page.value > 1)
const canNext = computed(() => orders.value.length >= pageSize)

function orderUlid(order: JsonRecord) {
  return String(pickFirst(order, ["order_ulid", "logical_order_ulid", "biz_order_ulid", "order_id"]) || "-")
}

function candidate(order: JsonRecord) {
  return String(pickFirst(order, ["candidate_name", "candidate_email", "candidate_ulid", "candidate_id"]) || "-")
}

function productName(order: JsonRecord) {
  return String(pickFirst(order, ["product_name", "productName", "name", "title"]) || labelFor(bizTypeOptions, biz(order)))
}

function biz(order: JsonRecord) {
  return pickFirst(order, ["biz_type", "bizType"])
}

function bizRef(order: JsonRecord) {
  return String(pickFirst(order, ["biz_ref_ulid", "bizRefUlid", "bundle_order_ulid", "order_ulid"]) || "")
}

function status(order: JsonRecord) {
  return pickFirst(order, ["order_status", "orderStatus", "status"])
}

function payStatus(order: JsonRecord) {
  return pickFirst(order, ["payment_status", "paymentStatus"])
}

function amount(order: JsonRecord) {
  const direct = pickFirst(order, ["total", "total_amount", "totalAmount"])
  if (direct !== undefined) return Number(direct) || 0
  const minor = pickFirst(order, ["amount_minor", "amountMinor", "total_amount_cents", "totalAmountCents", "amount"])
  return Number(minor || 0) / 100
}

function currency(order: JsonRecord) {
  return String(pickFirst(order, ["currency_code", "currencyCode", "currency"]) || "")
}

function createdAt(order: JsonRecord) {
  const value = pickFirst(order, ["created_at", "createdAt"])
  if (typeof value === "number") {
    const ms = value > 1_000_000_000_000 ? value : value * 1000
    return formatDate(new Date(ms).toISOString())
  }
  return formatDate(String(value || ""))
}

function canPurge(order: JsonRecord) {
  return normalizeStatus(biz(order)) === "BUNDLE_PURCHASE"
}

async function load(targetPage = page.value) {
  loading.value = true
  try {
    const params = new URLSearchParams({
      page: String(targetPage),
      limit: String(pageSize),
      offset: String((targetPage - 1) * pageSize),
    })
    if (candidateUlid.value.trim()) params.set("candidate_ulid", candidateUlid.value.trim())
    if (bizType.value) params.set("biz_type", bizType.value)
    if (orderStatus.value) params.set("order_status", orderStatus.value)
    if (paymentStatus.value) params.set("payment_status", paymentStatus.value)

    const data = await apiClient<JsonRecord>(`/api/mall/orders?${params}`)
    const list = Array.isArray(data.items) ? data.items : Array.isArray(data.orders) ? data.orders : []
    orders.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    total.value = Number(data.total ?? data.total_count ?? data.totalCount ?? orders.value.length) || 0
    selected.value = orders.value[0] || null
    page.value = targetPage
  } catch (err) {
    console.error(err)
    orders.value = []
    selected.value = null
    total.value = 0
    toast.error("订单加载失败")
  } finally {
    loading.value = false
  }
}

async function purge(order: JsonRecord) {
  const candidateUlidValue = String(pickFirst(order, ["candidate_ulid", "candidateUlid", "candidate_id"]) || "")
  const bundleOrderUlid = bizRef(order)
  if (!candidateUlidValue || !bundleOrderUlid) {
    toast.error("缺少 candidate_ulid 或 bundle_order_ulid，无法清理")
    return
  }
  if (!window.confirm("确认清理该认证套餐订单关联的测试数据？")) return

  purging.value = bundleOrderUlid
  try {
    await apiClient("/api/mall/bundle-orders/purge", {
      method: "POST",
      body: JSON.stringify({
        candidate_ulid: candidateUlidValue,
        bundle_order_ulid: bundleOrderUlid,
      }),
    })
    toast.success("认证测试数据已清理")
    await load(page.value)
  } finally {
    purging.value = ""
  }
}

function search() {
  load(1)
}

onMounted(() => load(1))
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header class="flex items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">订单管理</h1>
        <p class="mt-2 text-slate-600">查看认证、管线、阶段、重考和资格申请订单。</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load(page)">
        <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
        刷新
      </button>
    </header>

    <form class="grid gap-4 rounded-3xl border border-slate-200 bg-white p-5 shadow-sm lg:grid-cols-[1fr_180px_180px_180px_auto]" @submit.prevent="search">
      <input v-model="candidateUlid" class="rounded-xl border border-slate-200 px-4 py-3" placeholder="Candidate ULID / 用户关键字" />
      <select v-model="bizType" class="rounded-xl border border-slate-200 px-4 py-3">
        <option value="">全部类型</option>
        <option v-for="option in bizTypeOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
      </select>
      <select v-model="orderStatus" class="rounded-xl border border-slate-200 px-4 py-3">
        <option value="">全部状态</option>
        <option v-for="option in orderStatusOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
      </select>
      <select v-model="paymentStatus" class="rounded-xl border border-slate-200 px-4 py-3">
        <option value="">全部支付状态</option>
        <option v-for="option in paymentStatusOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
      </select>
      <button class="inline-flex items-center justify-center gap-2 rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white" type="submit">
        <Search class="h-4 w-4" />
        查询
      </button>
    </form>

    <div class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
      <div class="flex items-center justify-between border-b border-slate-200 p-5">
        <h2 class="text-xl font-black">订单列表</h2>
        <span class="text-sm font-bold text-slate-500">共 {{ total }} 条</span>
      </div>
      <div v-if="loading" class="p-12 text-center text-slate-500">
        <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
        正在加载...
      </div>
      <table v-else class="w-full text-left text-sm">
        <thead class="bg-slate-50 text-xs uppercase tracking-wide text-slate-500">
          <tr>
            <th class="px-5 py-3">商品 / 业务</th>
            <th class="px-5 py-3">候选人</th>
            <th class="px-5 py-3">金额</th>
            <th class="px-5 py-3">״̬</th>
            <th class="px-5 py-3">支付状态</th>
            <th class="px-5 py-3">创建时间</th>
            <th class="px-5 py-3 text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="!orders.length">
            <td class="px-5 py-10 text-center text-slate-500" colspan="7">暂无订单</td>
          </tr>
          <tr v-for="order in orders" :key="orderUlid(order)" class="border-t border-slate-100 hover:bg-sky-50">
            <td class="px-5 py-4">
              <button class="text-left" type="button" @click="selected = order">
                <div class="font-black text-slate-950">{{ productName(order) }}</div>
                <div class="text-xs text-slate-500">{{ labelFor(bizTypeOptions, biz(order)) }}</div>
              </button>
            </td>
            <td class="px-5 py-4 text-slate-700">{{ candidate(order) }}</td>
            <td class="px-5 py-4 font-bold">{{ amount(order).toFixed(2) }} {{ currency(order) }}</td>
            <td class="px-5 py-4">
              <span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(status(order))">{{ labelFor(orderStatusOptions, status(order)) }}</span>
            </td>
            <td class="px-5 py-4">
              <span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(payStatus(order))">{{ labelFor(paymentStatusOptions, payStatus(order)) }}</span>
            </td>
            <td class="px-5 py-4 text-slate-600">{{ createdAt(order) }}</td>
            <td class="px-5 py-4">
              <div class="flex justify-end gap-2">
                <button class="rounded-lg border px-3 py-2 text-xs font-bold" type="button" @click="selected = order">
                  详情
                </button>
                <button
                  v-if="canPurge(order)"
                  class="inline-flex items-center gap-1 rounded-lg bg-red-600 px-3 py-2 text-xs font-bold text-white disabled:opacity-50"
                  type="button"
                  :disabled="purging === bizRef(order)"
                  @click="purge(order)"
                >
                  <Loader2 v-if="purging === bizRef(order)" class="h-3 w-3 animate-spin" />
                  <Trash2 v-else class="h-3 w-3" />
                  清理认证数据
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>

      <div class="flex justify-end gap-3 border-t border-slate-200 p-5">
        <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrev" @click="load(page - 1)">上一页</button>
        <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext" @click="load(page + 1)">下一页</button>
      </div>
    </div>

    <section v-if="selected" class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
      <div class="mb-4 flex items-center justify-between gap-4">
        <div>
          <h2 class="text-xl font-black">{{ productName(selected) }}</h2>
          <p class="text-sm text-slate-500">完整订单详情，包含排查需要的 ULID 字段。</p>
        </div>
        <ChevronRight class="h-5 w-5 text-slate-400" />
      </div>
      <pre class="max-h-[560px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selected, null, 2) }}</pre>
    </section>
  </section>
</template>
