<script setup lang="ts">
import { Loader2, RefreshCw, Search, Trash2, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import {
  badgeClass,
  bizTypeOptions,
  type LabelOption,
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
}

const zhCopy = {
  pageTitle: "订单管理",
  pageDescription: "查看认证、管线、阶段、重考和资格申请订单。",
  refresh: "刷新",
  candidatePlaceholder: "Candidate ULID / 用户关键字",
  allTypes: "全部类型",
  allStatuses: "全部状态",
  allPaymentStatuses: "全部支付状态",
  search: "查询",
  listTitle: "订单列表",
  listDescription: "点击订单查看详情并处理。",
  totalPrefix: "共",
  totalSuffix: "条",
  columns: {
    order: "订单",
    amount: "金额",
    status: "状态",
    createdAt: "创建时间",
    action: "操作",
  },
  orderPrefix: "订单：",
  viewDetails: "查看详情",
  loading: "正在加载...",
  empty: "暂无订单",
  pagePrefix: "第",
  pageSuffix: "页",
  prev: "上一页",
  next: "下一页",
  close: "关闭",
  currentOrder: "当前订单",
  orderAmount: "订单金额",
  tabs: {
    summary: "订单摘要",
    bundleDetail: "套餐详情",
    actions: "支持操作",
    raw: "完整字段",
  },
  fields: {
    productName: "商品名称",
    orderAmount: "订单金额",
    orderStatus: "订单状态",
    paymentStatus: "支付状态",
    bizType: "业务类型",
    bizTypeCode: "业务类型编码",
    currency: "币种",
    rawAmount: "原始金额",
    candidate: "候选人",
    orderId: "订单号",
    payOrderId: "支付订单号",
    bizRefId: "业务关联 ID",
    createdAt: "创建时间",
    bundleOrderId: "套餐订单 ID",
    bundleId: "套餐 ID",
    paymentMode: "支付模式",
  },
  bundleUnsupported: "只有认证套餐订单支持拉取套餐订单详情。",
  bundleLoading: "正在加载套餐订单详情...",
  bundleRaw: "完整套餐字段",
  bundleEmpty: "暂无套餐订单详情",
  actionsTitle: "支持操作",
  actionsDescription: "认证套餐订单可清理关联测试数据；其他订单类型仅查看。",
  purgeAction: "清理认证测试数据",
  rawNote: "系统字段仅用于查看。",
  confirmTitle: "确认清理认证测试数据",
  confirmDescription: "该操作会清理认证套餐订单关联的测试数据。",
  cancel: "取消",
  confirmPurge: "确认清理",
  purging: "清理中...",
  toasts: {
    bundleLoadFailed: "套餐订单详情加载失败",
    ordersLoadFailed: "订单加载失败",
    purgeMissing: "缺少 candidate_ulid 或 bundle_order_ulid，无法清理",
    purgeSuccess: "认证测试数据已清理",
    purgeFailed: "清理失败",
  },
  bizTypes: {
    PIPELINE_PAYMENT: "管线订单",
    STAGE_PAYMENT: "阶段订单",
    COURSE_RETAKE_PAYMENT: "重考订单",
    PIPELINE_UNLOCK: "管线解锁订单",
    CREDENTIAL_APPLICATION: "资格申请订单",
    BUNDLE_PURCHASE: "认证套餐订单",
  },
  orderStatuses: {
    PENDING: "待处理",
    WAIT_PAYMENT: "待支付",
    WAIT_BUNDLE_PAYMENT: "等待套餐支付",
    WAIT_PIPELINE_PAYMENT: "等待管线支付",
    WAIT_PIPELINE_INSTANTIATE: "管线创建中",
    WAIT_EXEMPTION_SELECTION: "等待选择免考",
    WAIT_EXEMPTION_REVIEW: "等待免考审核",
    WAIT_STAGE_PAYMENT: "等待阶段支付",
    WAIT_REVIEW_FEE_PAYMENT: "等待审核费支付",
    WAIT_RETAKE_PAYMENT: "等待重考支付",
    WAIT_UNLOCK_PAYMENT: "等待解锁支付",
    UPLOAD_READY: "可上传材料",
    UNDER_REVIEW: "审核中",
    RESOLVED: "已处理",
    PAID: "已支付",
    COMPLETED: "已完成",
    CANCELLED: "已取消",
    FAILED: "失败",
    EXPIRED: "已过期",
    PENDING_CREATE: "等待创建",
    PENDING_PAYMENT: "等待支付",
  },
  paymentStatuses: {
    WAIT_PAY: "待支付",
    WAIT_PAYMENT: "待支付",
    UNPAID: "待支付",
    PAID: "已支付",
    COMPLETED: "已支付",
    FAILED: "支付失败",
    REFUNDED: "已退款",
    CANCELLED: "已取消",
  },
}

const enCopy: typeof zhCopy = {
  pageTitle: "Order Management",
  pageDescription: "View certification, pipeline, stage, retake, and credential application orders.",
  refresh: "Refresh",
  candidatePlaceholder: "Candidate ULID / user keyword",
  allTypes: "All types",
  allStatuses: "All statuses",
  allPaymentStatuses: "All payment statuses",
  search: "Search",
  listTitle: "Order List",
  listDescription: "Click an order to view details and manage it.",
  totalPrefix: "Total",
  totalSuffix: "items",
  columns: {
    order: "Order",
    amount: "Amount",
    status: "Status",
    createdAt: "Created At",
    action: "Action",
  },
  orderPrefix: "Order:",
  viewDetails: "View Details",
  loading: "Loading...",
  empty: "No orders",
  pagePrefix: "Page",
  pageSuffix: "",
  prev: "Previous",
  next: "Next",
  close: "Close",
  currentOrder: "Current Order",
  orderAmount: "Order Amount",
  tabs: {
    summary: "Order Summary",
    bundleDetail: "Bundle Details",
    actions: "Actions",
    raw: "Raw Fields",
  },
  fields: {
    productName: "Product Name",
    orderAmount: "Order Amount",
    orderStatus: "Order Status",
    paymentStatus: "Payment Status",
    bizType: "Business Type",
    bizTypeCode: "Business Type Code",
    currency: "Currency",
    rawAmount: "Raw Amount",
    candidate: "Candidate",
    orderId: "Order ID",
    payOrderId: "Payment Order ID",
    bizRefId: "Business Ref ID",
    createdAt: "Created At",
    bundleOrderId: "Bundle Order ID",
    bundleId: "Bundle ID",
    paymentMode: "Payment Mode",
  },
  bundleUnsupported: "Only certification bundle orders support fetching bundle order details.",
  bundleLoading: "Loading bundle order details...",
  bundleRaw: "Raw Bundle Fields",
  bundleEmpty: "No bundle order details",
  actionsTitle: "Actions",
  actionsDescription: "Certification bundle orders can purge related test data; other order types are view-only.",
  purgeAction: "Purge Certification Test Data",
  rawNote: "System fields are for viewing only.",
  confirmTitle: "Confirm Test Data Purge",
  confirmDescription: "This will clean test data associated with the certification bundle order.",
  cancel: "Cancel",
  confirmPurge: "Confirm Purge",
  purging: "Purging...",
  toasts: {
    bundleLoadFailed: "Failed to load bundle order details",
    ordersLoadFailed: "Failed to load orders",
    purgeMissing: "Missing candidate_ulid or bundle_order_ulid; unable to purge",
    purgeSuccess: "Certification test data has been purged",
    purgeFailed: "Purge failed",
  },
  bizTypes: {
    PIPELINE_PAYMENT: "Pipeline Order",
    STAGE_PAYMENT: "Stage Order",
    COURSE_RETAKE_PAYMENT: "Retake Order",
    PIPELINE_UNLOCK: "Pipeline Unlock Order",
    CREDENTIAL_APPLICATION: "Credential Application Order",
    BUNDLE_PURCHASE: "Certification Bundle Order",
  },
  orderStatuses: {
    PENDING: "Pending",
    WAIT_PAYMENT: "Awaiting Payment",
    WAIT_BUNDLE_PAYMENT: "Awaiting Bundle Payment",
    WAIT_PIPELINE_PAYMENT: "Awaiting Pipeline Payment",
    WAIT_PIPELINE_INSTANTIATE: "Creating Pipeline",
    WAIT_EXEMPTION_SELECTION: "Awaiting Exemption Selection",
    WAIT_EXEMPTION_REVIEW: "Awaiting Exemption Review",
    WAIT_STAGE_PAYMENT: "Awaiting Stage Payment",
    WAIT_REVIEW_FEE_PAYMENT: "Awaiting Review Fee Payment",
    WAIT_RETAKE_PAYMENT: "Awaiting Retake Payment",
    WAIT_UNLOCK_PAYMENT: "Awaiting Unlock Payment",
    UPLOAD_READY: "Ready to Upload",
    UNDER_REVIEW: "Under Review",
    RESOLVED: "Resolved",
    PAID: "Paid",
    COMPLETED: "Completed",
    CANCELLED: "Cancelled",
    FAILED: "Failed",
    EXPIRED: "Expired",
    PENDING_CREATE: "Pending Creation",
    PENDING_PAYMENT: "Awaiting Payment",
  },
  paymentStatuses: {
    WAIT_PAY: "Awaiting Payment",
    WAIT_PAYMENT: "Awaiting Payment",
    UNPAID: "Unpaid",
    PAID: "Paid",
    COMPLETED: "Paid",
    FAILED: "Payment Failed",
    REFUNDED: "Refunded",
    CANCELLED: "Cancelled",
  },
}

const { lang } = useAdminLanguage()
const copy = computed(() => (lang.value === "en" ? enCopy : zhCopy))

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
const localizedBizTypeOptions = computed(() => localizeOptions(bizTypeOptions, "bizTypes"))
const localizedOrderStatusOptions = computed(() => localizeOptions(orderStatusOptions, "orderStatuses"))
const localizedPaymentStatusOptions = computed(() => localizeOptions(paymentStatusOptions, "paymentStatuses"))
const detailTabs = computed(() => [
  { key: "summary" as const, title: copy.value.tabs.summary, count: selected.value ? 1 : 0 },
  { key: "bundle-detail" as const, title: copy.value.tabs.bundleDetail, count: bundleDetail.value ? 1 : 0 },
  { key: "actions" as const, title: copy.value.tabs.actions, count: isBundlePurchase.value ? 1 : 0 },
  { key: "raw" as const, title: copy.value.tabs.raw, count: 1 },
])
const orderSummaryFields = computed<SummaryField[]>(() => {
  const order = selected.value
  if (!order) return []
  return [
    { label: copy.value.fields.productName, value: productName(order) },
    { label: copy.value.fields.orderAmount, value: amountText(order) },
    { label: copy.value.fields.orderStatus, value: localizedLabelFor("orderStatuses", status(order), orderStatusOptions) },
    { label: copy.value.fields.paymentStatus, value: localizedLabelFor("paymentStatuses", payStatus(order), paymentStatusOptions) },
    { label: copy.value.fields.bizType, value: localizedLabelFor("bizTypes", biz(order), bizTypeOptions) },
    { label: copy.value.fields.bizTypeCode, value: stringValue(biz(order)) },
    { label: copy.value.fields.currency, value: stringValue(pickFirst(order, ["currency_code", "currencyCode", "currency"])) },
    { label: copy.value.fields.rawAmount, value: stringValue(pickFirst(order, ["amount_minor"])) },
    { label: copy.value.fields.candidate, value: candidate(order) },
    { label: copy.value.fields.orderId, value: orderUlid(order) },
    { label: copy.value.fields.payOrderId, value: stringValue(pickFirst(order, ["pay_order_ulid", "payOrderUlid"])) },
    { label: copy.value.fields.bizRefId, value: bizRef(order) || "-" },
    { label: copy.value.fields.createdAt, value: createdAt(order) },
  ]
})
const bundleSummaryFields = computed<SummaryField[]>(() => {
  const detail = bundleDetail.value
  if (!detail) return []
  const source = bundleDetailSource(detail)
  return [
    { label: copy.value.fields.bundleOrderId, value: stringValue(pickFirst(source, ["bundle_order_ulid", "order_ulid"]) || bizRef(selected.value)) },
    { label: copy.value.fields.bundleId, value: stringValue(pickFirst(source, ["bundle_ulid", "bundle_id"])) },
    { label: copy.value.fields.candidate, value: stringValue(pickFirst(source, ["candidate_ulid", "candidate_id"]) || candidate(selected.value)) },
    { label: copy.value.fields.paymentMode, value: stringValue(pickFirst(source, ["payment_mode", "paymentMode"])) },
    { label: copy.value.fields.orderStatus, value: localizedLabelFor("orderStatuses", pickFirst(source, ["order_status", "orderStatus", "status"]), orderStatusOptions) },
    { label: copy.value.fields.createdAt, value: formatDate(String(pickFirst(source, ["created_at", "createdAt"]) || "")) || "-" },
  ]
})

function localizeOptions(options: LabelOption[], group: "bizTypes" | "orderStatuses" | "paymentStatuses") {
  return options.map((option) => ({
    ...option,
    label: copy.value[group][option.value as keyof typeof copy.value[typeof group]] || option.label,
  }))
}

function localizedLabelFor(group: "bizTypes" | "orderStatuses" | "paymentStatuses", value: unknown, fallbackOptions: LabelOption[]) {
  const normalized = normalizeStatus(value)
  if (!normalized) return "-"
  const translated = copy.value[group][normalized as keyof typeof copy.value[typeof group]]
  return translated || labelFor(fallbackOptions, normalized)
}

function orderUlid(order: JsonRecord | null | undefined) {
  return String(pickFirst(order || {}, ["order_ulid", "logical_order_ulid", "biz_order_ulid", "order_id"]) || "")
}

function candidate(order: JsonRecord | null | undefined) {
  return String(pickFirst(order || {}, ["candidate_name", "candidate_email", "candidate_ulid", "candidate_id"]) || "-")
}

function productName(order: JsonRecord | null | undefined) {
  return String(pickFirst(order || {}, ["product_name", "productName", "name", "title"]) || localizedLabelFor("bizTypes", biz(order || {}), bizTypeOptions))
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
    toast.error(copy.value.toasts.bundleLoadFailed)
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
    toast.error(copy.value.toasts.ordersLoadFailed)
  } finally {
    loading.value = false
  }
}

async function purgeSelected() {
  if (!selected.value) return
  const candidateUlidValue = String(pickFirst(selected.value, ["candidate_ulid", "candidateUlid", "candidate_id"]) || "")
  const bundleOrderUlid = bizRef(selected.value)
  if (!candidateUlidValue || !bundleOrderUlid) {
    toast.error(copy.value.toasts.purgeMissing)
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
    toast.success(copy.value.toasts.purgeSuccess)
    showPurgeConfirm.value = false
    await load(page.value)
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.purgeFailed)
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
        <h1 class="text-4xl font-black tracking-tight">{{ copy.pageTitle }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.pageDescription }}</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load(page)">
        <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
        {{ copy.refresh }}
      </button>
    </header>

    <form class="grid gap-3 rounded-3xl border border-slate-200 bg-white p-3 shadow-sm lg:grid-cols-[1fr_180px_180px_180px_auto]" @submit.prevent="search">
      <input v-model="candidateUlid" class="h-10 rounded-xl border border-slate-200 px-4 text-sm" :placeholder="copy.candidatePlaceholder" />
      <select v-model="bizType" class="h-10 rounded-xl border border-slate-200 px-4 text-sm">
        <option value="">{{ copy.allTypes }}</option>
        <option v-for="option in localizedBizTypeOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
      </select>
      <select v-model="orderStatus" class="h-10 rounded-xl border border-slate-200 px-4 text-sm">
        <option value="">{{ copy.allStatuses }}</option>
        <option v-for="option in localizedOrderStatusOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
      </select>
      <select v-model="paymentStatus" class="h-10 rounded-xl border border-slate-200 px-4 text-sm">
        <option value="">{{ copy.allPaymentStatuses }}</option>
        <option v-for="option in localizedPaymentStatusOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
      </select>
      <button class="inline-flex h-10 items-center justify-center gap-2 rounded-xl bg-[#0b7bdc] px-5 text-sm font-bold text-white" type="submit">
        <Search class="h-4 w-4" />
        {{ copy.search }}
      </button>
    </form>

    <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
      <div class="flex items-center justify-between border-b border-slate-200 p-5">
        <div>
          <h2 class="text-xl font-black">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
        <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ copy.totalPrefix }} {{ total }} {{ copy.totalSuffix }}</span>
      </div>
      <div class="grid grid-cols-[minmax(0,1fr)_140px_190px_170px_112px] gap-4 border-b border-slate-200 bg-slate-50 px-5 py-3 text-xs font-black text-slate-500">
        <span>{{ copy.columns.order }}</span>
        <span class="text-right">{{ copy.columns.amount }}</span>
        <span class="text-center">{{ copy.columns.status }}</span>
        <span>{{ copy.columns.createdAt }}</span>
        <span class="text-right">{{ copy.columns.action }}</span>
      </div>
      <div v-if="loading" class="p-12 text-center text-slate-500">
        <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
        {{ copy.loading }}
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
              <span>{{ localizedLabelFor("bizTypes", biz(order), bizTypeOptions) }}</span>
              <span class="break-all rounded-full bg-slate-100 px-2 py-1">{{ copy.orderPrefix }} {{ orderUlid(order) || "-" }}</span>
              <span class="break-all rounded-full bg-slate-100 px-2 py-1">{{ candidate(order) }}</span>
            </div>
          </div>
          <div class="text-right text-sm font-black">{{ amountText(order) }}</div>
          <div class="flex items-center justify-center gap-2">
            <span class="inline-flex rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(status(order))">
              {{ localizedLabelFor("orderStatuses", status(order), orderStatusOptions) }}
            </span>
            <span class="inline-flex rounded-full bg-slate-100 px-3 py-1 text-xs font-black text-slate-600">
              {{ localizedLabelFor("paymentStatuses", payStatus(order), paymentStatusOptions) }}
            </span>
          </div>
          <div class="text-sm font-semibold text-slate-500">{{ createdAt(order) }}</div>
          <div class="text-right">
            <button
              class="rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-black text-[#0b4ea2] shadow-sm transition hover:border-sky-200 hover:bg-sky-50"
              type="button"
              @click.stop="selectOrder(order)"
            >
              {{ copy.viewDetails }}
            </button>
          </div>
        </div>
      </div>
      <div v-else class="p-12 text-center text-slate-500">{{ copy.empty }}</div>
      <div class="flex items-center justify-between gap-3 border-t border-slate-200 p-5">
        <span class="text-sm font-bold text-slate-500">{{ copy.pagePrefix }} {{ page }} {{ copy.pageSuffix }}</span>
        <div class="flex gap-3">
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrev" @click="load(page - 1)">{{ copy.prev }}</button>
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext" @click="load(page + 1)">{{ copy.next }}</button>
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
              <button
                class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
                type="button"
                :aria-label="copy.close"
                @click="closeDetail"
              >
                <X class="h-5 w-5" />
              </button>
            </div>
          </div>

          <div class="border-b border-slate-200 px-5 py-4">
            <div class="flex gap-2 overflow-x-auto">
              <button
                v-for="tab in detailTabs"
                :key="tab.key"
                class="inline-flex h-11 shrink-0 items-center gap-3 rounded-2xl border px-4 text-sm font-black transition"
                :class="activeTab === tab.key ? 'border-sky-200 bg-sky-50 text-slate-950' : 'border-slate-100 bg-white text-slate-700 hover:bg-slate-50'"
                type="button"
                @click="activeTab = tab.key"
              >
                <span>{{ tab.title }}</span>
                <span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-black text-slate-600">{{ tab.count }}</span>
              </button>
            </div>
          </div>

          <main class="h-[60vh] min-h-[360px] max-h-[620px] min-w-0 overflow-y-auto p-5">
              <div v-if="activeTab === 'summary'" class="space-y-5">
                <div class="rounded-2xl border border-blue-100 bg-blue-50 p-4">
                  <div class="flex flex-wrap items-start justify-between gap-4">
                    <div class="min-w-0">
                      <div class="text-xs font-black text-blue-600">{{ copy.currentOrder }}</div>
                      <div class="mt-1 truncate text-xl font-black text-slate-950">{{ productName(selected) }}</div>
                      <div class="mt-2 flex flex-wrap items-center gap-2">
                        <span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(status(selected))">
                          {{ localizedLabelFor("orderStatuses", status(selected), orderStatusOptions) }}
                        </span>
                        <span class="rounded-full bg-white px-3 py-1 text-xs font-black text-slate-600">
                          {{ localizedLabelFor("paymentStatuses", payStatus(selected), paymentStatusOptions) }}
                        </span>
                      </div>
                    </div>
                    <div class="rounded-2xl border border-blue-100 bg-white px-5 py-4 text-right shadow-sm">
                      <div class="text-xs font-black text-slate-400">{{ copy.orderAmount }}</div>
                      <div class="mt-1 text-2xl font-black text-[#0b4ea2]">{{ amountText(selected) }}</div>
                    </div>
                  </div>
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <div
                    v-for="field in orderSummaryFields"
                    :key="field.label"
                    class="rounded-2xl border border-slate-200 bg-slate-50 p-4"
                  >
                    <div class="text-xs font-black uppercase text-slate-400">{{ field.label }}</div>
                    <div class="mt-2 break-all text-sm font-black text-slate-800">{{ field.value }}</div>
                  </div>
                </div>
              </div>

              <div v-else-if="activeTab === 'bundle-detail'" class="space-y-4">
                <div v-if="!isBundlePurchase" class="rounded-2xl border border-dashed border-slate-200 p-10 text-center text-slate-500">
                  {{ copy.bundleUnsupported }}
                </div>
                <div v-else-if="detailLoading" class="p-12 text-center text-slate-500">
                  <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
                  {{ copy.bundleLoading }}
                </div>
                <div v-else-if="bundleDetail" class="space-y-4">
                  <div class="grid gap-4 md:grid-cols-2">
                    <div
                      v-for="field in bundleSummaryFields"
                      :key="field.label"
                      class="rounded-2xl border border-slate-200 bg-slate-50 p-4"
                    >
                      <div class="text-xs font-black uppercase text-slate-400">{{ field.label }}</div>
                      <div class="mt-2 break-all text-sm font-black text-slate-800">{{ field.value }}</div>
                    </div>
                  </div>
                  <details class="overflow-hidden rounded-2xl border border-slate-200">
                    <summary class="cursor-pointer bg-slate-50 px-4 py-3 text-sm font-black text-slate-700">{{ copy.bundleRaw }}</summary>
                    <pre class="max-h-[520px] overflow-auto bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(bundleDetail, null, 2) }}</pre>
                  </details>
                </div>
                <div v-else class="rounded-2xl border border-dashed border-slate-200 p-10 text-center text-slate-500">{{ copy.bundleEmpty }}</div>
              </div>

              <div v-else-if="activeTab === 'actions'" class="space-y-4">
                <div class="rounded-2xl border border-slate-200 bg-slate-50 p-5">
                  <div class="text-base font-black text-slate-950">{{ copy.actionsTitle }}</div>
                  <p class="mt-2 text-sm text-slate-600">
                    {{ copy.actionsDescription }}
                  </p>
                </div>
                <button
                  class="inline-flex h-11 items-center gap-2 rounded-xl bg-red-600 px-5 text-sm font-bold text-white shadow-sm shadow-red-200 disabled:opacity-50"
                  type="button"
                  :disabled="!isBundlePurchase || Boolean(purging)"
                  @click="showPurgeConfirm = true"
                >
                  <Trash2 class="h-4 w-4" />
                  {{ copy.purgeAction }}
                </button>
              </div>

              <div v-else-if="activeTab === 'raw'" class="space-y-4">
                <div class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
                  {{ copy.rawNote }}
                </div>
                <pre class="max-h-[620px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selected, null, 2) }}</pre>
              </div>
          </main>
        </section>
      </div>
    </Teleport>

    <div v-if="showPurgeConfirm" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-6">
      <div class="w-full max-w-md rounded-3xl bg-white p-6 shadow-2xl">
        <h2 class="text-2xl font-black">{{ copy.confirmTitle }}</h2>
        <p class="mt-3 text-sm text-slate-600">{{ copy.confirmDescription }}</p>
        <div class="mt-5 rounded-2xl bg-slate-50 p-4">
          <div class="font-black">{{ productName(selected) }}</div>
          <div class="mt-1 break-all text-xs text-slate-500">{{ bizRef(selected) }}</div>
        </div>
        <div class="mt-6 flex justify-end gap-3">
          <button class="rounded-xl border px-5 py-3 font-bold" type="button" :disabled="Boolean(purging)" @click="showPurgeConfirm = false">{{ copy.cancel }}</button>
          <button class="rounded-xl bg-red-600 px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="Boolean(purging)" @click="purgeSelected">
            {{ purging ? copy.purging : copy.confirmPurge }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
