<script setup lang="ts">
import { Loader2, RefreshCw, Search, Trash2, X } from "lucide-vue-next"
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

type DetailTab = "summary" | "bundle-detail" | "actions" | "raw"
type SummaryField = {
  label: string
  value: string
  wide?: boolean
}

const orders = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const bundleDetail = ref<JsonRecord | null>(null)
const loading = ref(false)
const detailLoading = ref(false)
const detailOpen = ref(false)
const purging = ref("")
const showPurgeConfirm = ref(false)
const page = ref(1)
const total = ref(0)
const pageSize = 20
const activeTab = ref<DetailTab>("summary")

const candidateUlid = ref("")
const bizType = ref("")
const orderStatus = ref("")
const paymentStatus = ref("")

const canPrev = computed(() => page.value > 1)
const canNext = computed(() => orders.value.length >= pageSize)
const isBundlePurchase = computed(() => normalizeStatus(biz(selected.value || {})) === "BUNDLE_PURCHASE")
const detailTabs = computed(() => [
  { key: "summary" as const, title: "订单摘要", count: selected.value ? 1 : 0 },
  { key: "bundle-detail" as const, title: "套餐详情", count: bundleDetail.value ? 1 : 0 },
  { key: "actions" as const, title: "支持操作", count: isBundlePurchase.value ? 1 : 0 },
  { key: "raw" as const, title: "完整字段", count: 1 },
])
const orderSummaryFields = computed<SummaryField[]>(() => {
  const order = selected.value
  if (!order) return []
  return [
    { label: "商品名称", value: productName(order), wide: true },
    { label: "订单金额", value: amountText(order) },
    { label: "订单状态", value: labelFor(orderStatusOptions, status(order)) },
    { label: "支付状态", value: labelFor(paymentStatusOptions, payStatus(order)) },
    { label: "业务类型", value: labelFor(bizTypeOptions, biz(order)) },
    { label: "业务类型编码", value: stringValue(biz(order)) },
    { label: "币种", value: stringValue(pickFirst(order, ["currency_code", "currencyCode", "currency"])) },
    { label: "原始金额", value: stringValue(pickFirst(order, ["amount_minor"])) },
    { label: "候选人", value: candidate(order) },
    { label: "订单号", value: orderUlid(order), wide: true },
    { label: "支付订单号", value: stringValue(pickFirst(order, ["pay_order_ulid", "payOrderUlid"])), wide: true },
    { label: "业务关联 ID", value: bizRef(order) || "-", wide: true },
    { label: "创建时间", value: createdAt(order) },
  ]
})
const bundleSummaryFields = computed<SummaryField[]>(() => {
  const detail = bundleDetail.value
  if (!detail) return []
  const source = bundleDetailSource(detail)
  return [
    { label: "套餐订单 ID", value: stringValue(pickFirst(source, ["bundle_order_ulid", "order_ulid"]) || bizRef(selected.value)), wide: true },
    { label: "套餐 ID", value: stringValue(pickFirst(source, ["bundle_ulid", "bundle_id"])) },
    { label: "候选人", value: stringValue(pickFirst(source, ["candidate_ulid", "candidate_id"]) || candidate(selected.value)) },
    { label: "支付模式", value: stringValue(pickFirst(source, ["payment_mode", "paymentMode"])) },
    { label: "订单状态", value: stringValue(pickFirst(source, ["order_status", "orderStatus", "status"])) },
    { label: "创建时间", value: formatDate(String(pickFirst(source, ["created_at", "createdAt"]) || "")) || "-" },
  ]
})

function orderUlid(order: JsonRecord | null | undefined) {
  return String(pickFirst(order || {}, ["order_ulid", "logical_order_ulid", "biz_order_ulid", "order_id"]) || "")
}

function candidate(order: JsonRecord | null | undefined) {
  return String(pickFirst(order || {}, ["candidate_name", "candidate_email", "candidate_ulid", "candidate_id"]) || "-")
}

function productName(order: JsonRecord | null | undefined) {
  return String(pickFirst(order || {}, ["product_name", "productName", "name", "title"]) || labelFor(bizTypeOptions, biz(order || {})))
}

function biz(order: JsonRecord | null | undefined) {
  return pickFirst(order || {}, ["biz_type", "bizType"])
}

function bizRef(order: JsonRecord | null | undefined) {
  return String(pickFirst(order || {}, ["biz_ref_ulid", "bizRefUlid", "bundle_order_ulid"]) || "")
}

function status(order: JsonRecord | null | undefined) {
  return pickFirst(order || {}, ["order_status", "orderStatus", "status"])
}

function payStatus(order: JsonRecord | null | undefined) {
  return pickFirst(order || {}, ["payment_status", "paymentStatus"])
}

function amountText(order: JsonRecord | null | undefined) {
  const minor = pickFirst(order || {}, ["amount_minor"])
  const currency = String(pickFirst(order || {}, ["currency_code", "currencyCode", "currency"]) || "")
  if (minor === undefined || minor === null || minor === "") return "-"
  const amount = Number(minor)
  if (!Number.isFinite(amount)) return "-"
  return `${currency ? `${currency} ` : ""}${(amount / 100).toFixed(2)}`
}

function createdAt(order: JsonRecord | null | undefined) {
  const value = pickFirst(order || {}, ["created_at", "createdAt"])
  if (typeof value === "number") {
    const ms = value > 1_000_000_000_000 ? value : value * 1000
    return formatDate(new Date(ms).toISOString())
  }
  return formatDate(String(value || ""))
}

function stringValue(value: unknown) {
  if (value === undefined || value === null || value === "") return "-"
  return String(value)
}

function bundleDetailSource(detail: JsonRecord) {
  const nestedDetail = detail.detail
  if (nestedDetail && typeof nestedDetail === "object" && !Array.isArray(nestedDetail)) {
    const summary = (nestedDetail as JsonRecord).summary
    if (summary && typeof summary === "object" && !Array.isArray(summary)) return summary as JsonRecord
    return nestedDetail as JsonRecord
  }
  const summary = detail.summary
  if (summary && typeof summary === "object" && !Array.isArray(summary)) return summary as JsonRecord
  return detail
}

function canPurge(order: JsonRecord | null | undefined) {
  return normalizeStatus(biz(order || {})) === "BUNDLE_PURCHASE"
}

async function loadBundleDetail(order: JsonRecord | null) {
  bundleDetail.value = null
  if (!order || !canPurge(order) || !bizRef(order)) return
  detailLoading.value = true
  try {
    bundleDetail.value = await apiClient<JsonRecord>(`/api/mall/bundle-orders/${encodeURIComponent(bizRef(order))}`)
  } catch (err) {
    console.error(err)
    toast.error("套餐订单详情加载失败")
  } finally {
    detailLoading.value = false
  }
}

async function selectOrder(order: JsonRecord, open = true) {
  selected.value = order
  activeTab.value = "summary"
  showPurgeConfirm.value = false
  detailOpen.value = open
  await loadBundleDetail(order)
}

function closeDetail() {
  detailOpen.value = false
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
    page.value = targetPage
    if (orders.value.length) {
      await selectOrder(orders.value[0], detailOpen.value)
    } else {
      selected.value = null
      bundleDetail.value = null
      detailOpen.value = false
    }
  } catch (err) {
    console.error(err)
    orders.value = []
    selected.value = null
    bundleDetail.value = null
    detailOpen.value = false
    total.value = 0
    toast.error("订单加载失败")
  } finally {
    loading.value = false
  }
}

async function purgeSelected() {
  if (!selected.value) return
  const candidateUlidValue = String(pickFirst(selected.value, ["candidate_ulid", "candidateUlid", "candidate_id"]) || "")
  const bundleOrderUlid = bizRef(selected.value)
  if (!candidateUlidValue || !bundleOrderUlid) {
    toast.error("缺少 candidate_ulid 或 bundle_order_ulid，无法清理")
    return
  }

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
    showPurgeConfirm.value = false
    await load(page.value)
  } catch (err) {
    console.error(err)
    toast.error("清理失败")
  } finally {
    purging.value = ""
  }
}

function search() {
  void load(1)
}

onMounted(() => load(1))
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1580px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">订单管理</h1>
        <p class="mt-2 text-slate-600">查看认证、管线、阶段、重考和资格申请订单。</p>
        <p class="mt-2 text-xs font-semibold text-slate-500">
          已确认接口：list orders、bundle order detail、bundle order purge。金额只展示列表接口返回的 amount_minor，不做前端兜底推算。
        </p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load(page)">
        <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
        刷新
      </button>
    </header>

    <form class="grid gap-3 rounded-3xl border border-slate-200 bg-white p-3 shadow-sm lg:grid-cols-[1fr_180px_180px_180px_auto]" @submit.prevent="search">
      <input v-model="candidateUlid" class="h-10 rounded-xl border border-slate-200 px-4 text-sm" placeholder="Candidate ULID / 用户关键字" />
      <select v-model="bizType" class="h-10 rounded-xl border border-slate-200 px-4 text-sm">
        <option value="">全部类型</option>
        <option v-for="option in bizTypeOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
      </select>
      <select v-model="orderStatus" class="h-10 rounded-xl border border-slate-200 px-4 text-sm">
        <option value="">全部状态</option>
        <option v-for="option in orderStatusOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
      </select>
      <select v-model="paymentStatus" class="h-10 rounded-xl border border-slate-200 px-4 text-sm">
        <option value="">全部支付状态</option>
        <option v-for="option in paymentStatusOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
      </select>
      <button class="inline-flex h-10 items-center justify-center gap-2 rounded-xl bg-[#0b7bdc] px-5 text-sm font-bold text-white" type="submit">
        <Search class="h-4 w-4" />
        查询
      </button>
    </form>

    <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
      <div class="flex items-center justify-between border-b border-slate-200 p-5">
        <div>
          <h2 class="text-xl font-black">订单列表</h2>
          <p class="mt-1 text-sm text-slate-500">来自 `/api/mall/orders`，点击查看详情后在弹框中处理。</p>
        </div>
        <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">共 {{ total }} 条</span>
      </div>
      <div class="grid grid-cols-[minmax(0,1fr)_140px_190px_170px_112px] gap-4 border-b border-slate-200 bg-slate-50 px-5 py-3 text-xs font-black text-slate-500">
        <span>订单</span>
        <span class="text-right">金额</span>
        <span class="text-center">状态</span>
        <span>创建时间</span>
        <span class="text-right">操作</span>
      </div>
      <div v-if="loading" class="p-12 text-center text-slate-500">
        <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
        正在加载...
      </div>
      <div v-else-if="orders.length" class="divide-y divide-slate-100">
        <div
          v-for="order in orders"
          :key="orderUlid(order)"
          class="grid cursor-pointer grid-cols-[minmax(0,1fr)_140px_190px_170px_112px] items-center gap-4 px-5 py-4 transition hover:bg-sky-50"
          :class="orderUlid(selected) === orderUlid(order) ? 'bg-sky-50' : ''"
          role="button"
          tabindex="0"
          @click="selectOrder(order)"
          @keydown.enter.prevent="selectOrder(order)"
          @keydown.space.prevent="selectOrder(order)"
        >
          <div class="min-w-0">
            <div class="truncate font-black text-slate-950">{{ productName(order) }}</div>
            <div class="mt-1 flex flex-wrap items-center gap-2 text-xs font-semibold text-slate-500">
              <span>{{ labelFor(bizTypeOptions, biz(order)) }}</span>
              <span class="break-all rounded-full bg-slate-100 px-2 py-1">订单：{{ orderUlid(order) || "-" }}</span>
              <span class="break-all rounded-full bg-slate-100 px-2 py-1">{{ candidate(order) }}</span>
            </div>
          </div>
          <div class="text-right text-sm font-black">{{ amountText(order) }}</div>
          <div class="flex items-center justify-center gap-2">
            <span class="inline-flex rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(status(order))">
              {{ labelFor(orderStatusOptions, status(order)) }}
            </span>
            <span class="inline-flex rounded-full bg-slate-100 px-3 py-1 text-xs font-black text-slate-600">
              {{ labelFor(paymentStatusOptions, payStatus(order)) }}
            </span>
          </div>
          <div class="text-sm font-semibold text-slate-500">{{ createdAt(order) }}</div>
          <div class="text-right">
            <button
              class="rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-black text-[#0b4ea2] shadow-sm transition hover:border-sky-200 hover:bg-sky-50"
              type="button"
              @click.stop="selectOrder(order)"
            >
              查看详情
            </button>
          </div>
        </div>
      </div>
      <div v-else class="p-12 text-center text-slate-500">暂无订单</div>
      <div class="flex items-center justify-between gap-3 border-t border-slate-200 p-5">
        <span class="text-sm font-bold text-slate-500">第 {{ page }} 页</span>
        <div class="flex gap-3">
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrev" @click="load(page - 1)">上一页</button>
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext" @click="load(page + 1)">下一页</button>
        </div>
      </div>
    </section>

    <Teleport to="body">
      <div v-if="detailOpen && selected" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-6">
        <section class="flex max-h-[88vh] w-full max-w-[1280px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-6 py-5">
            <div class="min-w-0">
              <h2 class="truncate text-2xl font-black text-slate-950">{{ productName(selected) }}</h2>
              <p class="mt-1 break-all text-sm text-slate-500">{{ orderUlid(selected) }}</p>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <span class="hidden rounded-full border px-3 py-1 text-xs font-black sm:inline-flex" :class="badgeClass(status(selected))">
                {{ labelFor(orderStatusOptions, status(selected)) }}
              </span>
              <span class="hidden rounded-full bg-slate-100 px-3 py-1 text-xs font-black text-slate-600 sm:inline-flex">
                {{ labelFor(paymentStatusOptions, payStatus(selected)) }}
              </span>
              <button
                class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
                type="button"
                aria-label="关闭"
                @click="closeDetail"
              >
                <X class="h-5 w-5" />
              </button>
            </div>
          </div>

          <div class="grid min-h-0 flex-1 overflow-hidden lg:grid-cols-[240px_minmax(0,1fr)]">
            <aside class="border-b border-slate-200 p-4 lg:border-b-0 lg:border-r">
              <div class="space-y-2">
                <button
                  v-for="tab in detailTabs"
                  :key="tab.key"
                  class="w-full rounded-2xl border px-4 py-3 text-left"
                  :class="activeTab === tab.key ? 'border-sky-200 bg-sky-50' : 'border-slate-100 hover:bg-slate-50'"
                  type="button"
                  @click="activeTab = tab.key"
                >
                  <div class="flex items-center justify-between gap-3">
                    <span class="font-black">{{ tab.title }}</span>
                    <span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-black text-slate-600">{{ tab.count }}</span>
                  </div>
                </button>
              </div>
            </aside>

            <main class="min-w-0 overflow-y-auto p-5">
              <div v-if="activeTab === 'summary'" class="space-y-5">
                <div class="rounded-2xl border border-blue-100 bg-blue-50 p-4">
                  <div class="flex flex-wrap items-start justify-between gap-4">
                    <div class="min-w-0">
                      <div class="text-xs font-black text-blue-600">当前订单</div>
                      <div class="mt-1 truncate text-xl font-black text-slate-950">{{ productName(selected) }}</div>
                      <div class="mt-2 flex flex-wrap items-center gap-2">
                        <span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(status(selected))">
                          {{ labelFor(orderStatusOptions, status(selected)) }}
                        </span>
                        <span class="rounded-full bg-white px-3 py-1 text-xs font-black text-slate-600">
                          {{ labelFor(paymentStatusOptions, payStatus(selected)) }}
                        </span>
                      </div>
                    </div>
                    <div class="rounded-2xl border border-blue-100 bg-white px-5 py-4 text-right shadow-sm">
                      <div class="text-xs font-black text-slate-400">订单金额</div>
                      <div class="mt-1 text-2xl font-black text-[#0b4ea2]">{{ amountText(selected) }}</div>
                    </div>
                  </div>
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <div
                    v-for="field in orderSummaryFields"
                    :key="field.label"
                    class="rounded-2xl border border-slate-200 bg-slate-50 p-4"
                    :class="field.wide ? 'md:col-span-2' : ''"
                  >
                    <div class="text-xs font-black uppercase text-slate-400">{{ field.label }}</div>
                    <div class="mt-2 break-all text-sm font-black text-slate-800">{{ field.value }}</div>
                  </div>
                </div>
              </div>

              <div v-else-if="activeTab === 'bundle-detail'" class="space-y-4">
                <div v-if="!isBundlePurchase" class="rounded-2xl border border-dashed border-slate-200 p-10 text-center text-slate-500">
                  只有认证套餐订单支持拉取套餐订单详情。
                </div>
                <div v-else-if="detailLoading" class="p-12 text-center text-slate-500">
                  <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
                  正在加载套餐订单详情...
                </div>
                <div v-else-if="bundleDetail" class="space-y-4">
                  <div class="grid gap-4 md:grid-cols-2">
                    <div
                      v-for="field in bundleSummaryFields"
                      :key="field.label"
                      class="rounded-2xl border border-slate-200 bg-slate-50 p-4"
                      :class="field.wide ? 'md:col-span-2' : ''"
                    >
                      <div class="text-xs font-black uppercase text-slate-400">{{ field.label }}</div>
                      <div class="mt-2 break-all text-sm font-black text-slate-800">{{ field.value }}</div>
                    </div>
                  </div>
                  <details class="overflow-hidden rounded-2xl border border-slate-200">
                    <summary class="cursor-pointer bg-slate-50 px-4 py-3 text-sm font-black text-slate-700">完整套餐字段</summary>
                    <pre class="max-h-[520px] overflow-auto bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(bundleDetail, null, 2) }}</pre>
                  </details>
                </div>
                <div v-else class="rounded-2xl border border-dashed border-slate-200 p-10 text-center text-slate-500">暂无套餐订单详情</div>
              </div>

              <div v-else-if="activeTab === 'actions'" class="space-y-4">
                <div class="rounded-2xl border border-slate-200 bg-slate-50 p-5">
                  <div class="text-base font-black text-slate-950">支持操作</div>
                  <p class="mt-2 text-sm text-slate-600">
                    当前 adminbff 只提供认证套餐订单的测试数据清理接口。其他订单类型只读展示。
                  </p>
                </div>
                <button
                  class="inline-flex h-11 items-center gap-2 rounded-xl bg-red-600 px-5 text-sm font-bold text-white shadow-sm shadow-red-200 disabled:opacity-50"
                  type="button"
                  :disabled="!isBundlePurchase || Boolean(purging)"
                  @click="showPurgeConfirm = true"
                >
                  <Trash2 class="h-4 w-4" />
                  清理认证测试数据
                </button>
              </div>

              <div v-else-if="activeTab === 'raw'" class="space-y-4">
                <div class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
                  完整字段只读展示，包含列表接口原始字段。
                </div>
                <pre class="max-h-[620px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selected, null, 2) }}</pre>
              </div>
            </main>
          </div>
        </section>
      </div>
    </Teleport>

    <div v-if="showPurgeConfirm" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-6">
      <div class="w-full max-w-md rounded-3xl bg-white p-6 shadow-2xl">
        <h2 class="text-2xl font-black">确认清理认证测试数据</h2>
        <p class="mt-3 text-sm text-slate-600">该操作会调用 `/api/mall/bundle-orders/purge`，用于清理认证套餐订单关联的测试数据。</p>
        <div class="mt-5 rounded-2xl bg-slate-50 p-4">
          <div class="font-black">{{ productName(selected) }}</div>
          <div class="mt-1 break-all text-xs text-slate-500">{{ bizRef(selected) }}</div>
        </div>
        <div class="mt-6 flex justify-end gap-3">
          <button class="rounded-xl border px-5 py-3 font-bold" type="button" :disabled="Boolean(purging)" @click="showPurgeConfirm = false">取消</button>
          <button class="rounded-xl bg-red-600 px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="Boolean(purging)" @click="purgeSelected">
            {{ purging ? "清理中..." : "确认清理" }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
