<script setup lang="ts">
import { Loader2, RefreshCw, Search, Trash2, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
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

type DetailTab = "summary" | "business-detail" | "actions"
type SummaryField = {
  label: string
  value: string
}
type PriceMetric = {
  key: string
  label: string
  value: string
  note: string
  tone: string
}

const { t, isZh } = useAdminLanguage()
const copy = computed(() => t.value.orders)

const orders = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const businessDetail = ref<JsonRecord | null>(null)
const pricing = ref<JsonRecord | null>(null)
const exemptions = ref<JsonRecord[]>([])
const loading = ref(false)
const detailLoading = ref(false)
const detailOpen = ref(false)
const purging = ref("")
const showPurgeConfirm = ref(false)
const page = ref(1)
const total = ref(0)
const pageSize = 20
const hasMore = ref(false)
const nextCursor = ref("")
const prevCursor = ref("")
const lastPage = ref(1)
const activeTab = ref<DetailTab>("summary")

const candidateUlid = ref("")
const bizType = ref("")
const orderStatus = ref("")

const canPrev = computed(() => page.value > 1)
const canNext = computed(() => hasMore.value)
const isBundlePurchase = computed(() => normalizeStatus(biz(selected.value || {})) === "BUNDLE_PURCHASE")

const localizedBizTypeOptions = computed(() => localizeOptions(bizTypeOptions, "bizTypes"))
const localizedOrderStatusOptions = computed(() => localizeOptions(orderStatusOptions, "orderStatuses"))
const detailTabs = computed(() => [
  { key: "summary" as const, title: copy.value.tabs.summary, count: selected.value ? 1 : 0 },
  { key: "business-detail" as const, title: copy.value.tabs.bundleDetail, count: businessDetail.value ? 1 : 0 },
  { key: "actions" as const, title: copy.value.tabs.actions, count: isBundlePurchase.value ? 1 : 0 },
])
const pricingCurrency = computed(() => String(pricing.value?.currency_code || pickFirst(selected.value || {}, ["currency_code", "currencyCode", "currency"]) || ""))
const priceMetrics = computed<PriceMetric[]>(() => {
  const detail = pricing.value
  if (!detail) return []
  const exemptionRecorded = detail.exemption_amount_recorded === true
  return [
    {
      key: "billable-subtotal",
      label: copy.value.pricing.billableSubtotal,
      value: minorAmountText(detail.billable_subtotal_minor, pricingCurrency.value),
      note: copy.value.pricing.billableSubtotalNote,
      tone: "border-slate-200 bg-white text-slate-950",
    },
    {
      key: "exemption-discount",
      label: copy.value.pricing.exemptionDiscount,
      value: exemptionRecorded ? signedMinorAmountText(detail.exemption_discount_minor, pricingCurrency.value, "-") : copy.value.pricing.notRecorded,
      note: exemptions.value.length
        ? copy.value.pricing.exemptedUnits(exemptions.value.length)
        : copy.value.pricing.exemptionNotRecorded,
      tone: "border-amber-200 bg-amber-50 text-amber-900",
    },
    {
      key: "promotion-discount",
      label: copy.value.pricing.promotionDiscount,
      value: signedMinorAmountText(detail.promotion_discount_minor, pricingCurrency.value, "-"),
      note: copy.value.pricing.promotionDiscountNote,
      tone: "border-emerald-200 bg-emerald-50 text-emerald-900",
    },
    {
      key: "tax",
      label: copy.value.pricing.tax,
      value: signedMinorAmountText(detail.tax_minor, pricingCurrency.value, "+"),
      note: copy.value.pricing.taxNote,
      tone: "border-slate-200 bg-slate-50 text-slate-900",
    },
    {
      key: "paid",
      label: copy.value.pricing.amountPaid,
      value: minorAmountText(detail.amount_paid_minor ?? detail.total_minor, pricingCurrency.value),
      note: copy.value.pricing.amountPaidNote,
      tone: "border-blue-200 bg-blue-50 text-blue-900",
    },
  ]
})
const priceItems = computed<JsonRecord[]>(() => Array.isArray(pricing.value?.items)
  ? pricing.value!.items.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
  : [])
const promotionLabels = computed(() => {
  const labels: string[] = []
  const coupons = Array.isArray(pricing.value?.coupons) ? pricing.value!.coupons : []
  for (const value of coupons) {
    const coupon = recordValue(value)
    const label = String(coupon?.name || coupon?.code || "").trim()
    if (label && !labels.includes(label)) labels.push(label)
  }
  const promoCodes = Array.isArray(pricing.value?.promo_codes) ? pricing.value!.promo_codes : []
  for (const value of promoCodes) {
    const label = String(value || "").trim()
    if (label && !labels.includes(label)) labels.push(label)
  }
  return labels
})
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
const businessSummaryFields = computed<SummaryField[]>(() => {
  const detail = businessDetail.value
  if (!detail) return []
  const source = businessDetailSource(detail)
  const summary = recordValue(source.summary)
  const values = {
    ...(summary || {}),
    ...Object.fromEntries(Object.entries(source).filter(([key]) => key !== "summary")),
  }
  return Object.entries(values)
    .filter(([, value]) => value !== undefined && value !== null && value !== "")
    .map(([key, value]) => ({ label: businessFieldLabel(key), value: displayBusinessValue(key, value) }))
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

function minorAmountText(value: unknown, currency: string) {
  if (value === undefined || value === null || value === "") return copy.value.pricing.unavailable
  const amount = Number(value)
  if (!Number.isFinite(amount)) return copy.value.pricing.unavailable
  return `${currency ? `${currency.toUpperCase()} ` : ""}${(amount / 100).toFixed(2)}`
}

function signedMinorAmountText(value: unknown, currency: string, sign: "+" | "-") {
  const formatted = minorAmountText(value, currency)
  return formatted === copy.value.pricing.unavailable ? formatted : `${sign}${formatted}`
}

function pricingSourceLabel(value: unknown) {
  const source = String(value || "")
  const labels = copy.value.pricing.sources as Record<string, string>
  return labels[source] || source
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

function isTimeField(key: string) {
  return /(^|_)(created|updated|paid|expired|completed)_at$|At$/i.test(key)
}

function businessFieldLabel(key: string) {
  return copy.value.businessFields?.[key as keyof typeof copy.value.businessFields] || key.replaceAll("_", " ")
}

function paymentModeLabel(value: unknown) {
  const raw = String(value || "").trim()
  const normalized = normalizeStatus(raw)
  const labels = copy.value.paymentModes as Record<string, string>
  return labels[normalized] || raw || "-"
}

function dateValue(value: unknown) {
  if (typeof value === "number") {
    const ms = value > 1_000_000_000_000 ? value : value * 1000
    return formatDate(new Date(ms).toISOString())
  }
  return formatDate(String(value || ""))
}

function recordValue(value: unknown): JsonRecord | null {
  return value && typeof value === "object" && !Array.isArray(value) ? value as JsonRecord : null
}

function displayBusinessValue(key: string, value: unknown) {
  if (key === "payment_mode") return paymentModeLabel(value)
  if (isTimeField(key)) return dateValue(value)
  if (typeof value === "string") return value
  if (typeof value === "number" || typeof value === "boolean") return String(value)
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

function businessDetailSource(detail: JsonRecord) {
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

let detailRequestId = 0

async function loadBusinessDetail(order: JsonRecord | null) {
  const id = orderUlid(order)
  const requestId = ++detailRequestId
  businessDetail.value = null
  pricing.value = null
  exemptions.value = []
  if (!order || !id) return
  detailLoading.value = true
  try {
    const response = await apiClient<JsonRecord>(`/api/mall/orders/${encodeURIComponent(id)}`)
    if (requestId !== detailRequestId || orderUlid(selected.value) !== id) return
    businessDetail.value = recordValue(response.business_detail)
    pricing.value = recordValue(response.pricing)
    exemptions.value = Array.isArray(response.exemptions)
      ? response.exemptions.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
      : []
  } catch (err) {
    if (requestId !== detailRequestId || orderUlid(selected.value) !== id) return
    console.error(err)
    toast.error(copy.value.toasts.bundleLoadFailed)
  } finally {
    if (requestId === detailRequestId && orderUlid(selected.value) === id) detailLoading.value = false
  }
}

async function selectOrder(order: JsonRecord, open = true) {
  detailRequestId += 1
  selected.value = order
  activeTab.value = "summary"
  showPurgeConfirm.value = false
  detailOpen.value = open
  businessDetail.value = null
  pricing.value = null
  exemptions.value = []
  detailLoading.value = false
  if (!open) {
    return
  }
  await loadBusinessDetail(order)
}

function closeDetail() {
  detailOpen.value = false
}

function closePurgeConfirm() {
  if (purging.value) return
  showPurgeConfirm.value = false
}

let listRequestId = 0

async function load(targetPage = page.value) {
  const requestId = ++listRequestId
  loading.value = true
  try {
    const params = new URLSearchParams({
      page_size: String(pageSize),
    })

    let cursor = ""

    if (targetPage > lastPage.value) {

      cursor = nextCursor.value

    } else if (targetPage < lastPage.value) {

      cursor = prevCursor.value


    }

    

    if (cursor) params.set("cursor", cursor)


    if (candidateUlid.value.trim()) params.set("candidate_ulid", candidateUlid.value.trim())
    if (bizType.value) params.set("biz_type", bizType.value)
    if (orderStatus.value) params.set("order_status", orderStatus.value)

    const isValidUlid = (id: string) => /^[0-7][0-9A-HJKMNP-TV-Z]{25}$/i.test(id)
    if (candidateUlid.value.trim() && !isValidUlid(candidateUlid.value.trim())) {
      toast.error(copy.value.invalidCandidateUlid)
      orders.value = []
      selected.value = null
      businessDetail.value = null
      pricing.value = null
      exemptions.value = []
      detailOpen.value = false
      total.value = 0
      hasMore.value = false
      nextCursor.value = ""
      prevCursor.value = ""
      return
    }

    const data = await apiClient<JsonRecord>(`/api/mall/orders?${params}`)
    if (requestId !== listRequestId) return
    const list = Array.isArray(data.items) ? data.items : Array.isArray(data.orders) ? data.orders : []

    orders.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    total.value = Number(data.total ?? data.total_count ?? data.totalCount ?? orders.value.length) || 0
    const isBackward = page.value < lastPage.value
    hasMore.value = isBackward ? true : Boolean(data.has_more)
    lastPage.value = page.value
nextCursor.value = String(data.next_cursor || "")
    prevCursor.value = String(data?.prev_cursor || "")

    lastPage.value = targetPage
    page.value = targetPage
    if (orders.value.length) {
      void selectOrder(orders.value[0], detailOpen.value)
    } else {
      selected.value = null
      businessDetail.value = null
      pricing.value = null
      exemptions.value = []
      detailOpen.value = false
    }
  } catch (err) {
    if (requestId !== listRequestId) return
    console.error(err)
    orders.value = []
    selected.value = null
    businessDetail.value = null
    pricing.value = null
    exemptions.value = []
    detailOpen.value = false
    total.value = 0
    hasMore.value = false
    nextCursor.value = ""
    toast.error(copy.value.toasts.ordersLoadFailed)
  } finally {
    if (requestId === listRequestId) loading.value = false
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
    toast.error(apiErrorMessage(err, copy.value.toasts.purgeFailed))
  } finally {
    purging.value = ""
  }
}

function search() {
  page.value = 1
  lastPage.value = 1

  prevCursor.value = ""
  nextCursor.value = ""
  hasMore.value = false
  void load(1)
}

function clearCandidateSearch() {
  if (!candidateUlid.value) return
  candidateUlid.value = ""
  search()
}

onMounted(() => load(1))
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1580px] flex-col gap-5 px-4 py-5 md:gap-6 md:px-8 md:py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-3xl font-black tracking-tight md:text-4xl">{{ copy.pageTitle }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.pageDescription }}</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm disabled:cursor-not-allowed disabled:opacity-50" type="button" :disabled="loading" @click="load(page)">
        <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
        {{ copy.refresh }}
      </button>
    </header>

    <form class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm md:p-5" @submit.prevent="search">
      <div
        class="grid gap-4"
        :class="isZh ? 'xl:grid-cols-[minmax(0,1fr)_180px_180px_auto]' : 'xl:grid-cols-[minmax(0,1fr)_220px_220px_auto]'"
      >
        <label class="grid gap-2 text-sm font-bold">
          {{ copy.candidatePlaceholder }}
          <div class="relative">
            <input v-model="candidateUlid" class="h-11 w-full rounded-xl border border-slate-200 px-3 pr-10" :placeholder="copy.candidatePlaceholder" />
            <button
              v-if="candidateUlid"
              class="absolute right-2 top-1/2 inline-flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-full text-slate-400 transition hover:bg-slate-100 hover:text-slate-700"
              type="button"
              :aria-label="copy.close"
              :title="copy.close"
              @click="clearCandidateSearch"
            >
              <X class="h-4 w-4" />
            </button>
          </div>
        </label>
        <label class="grid gap-2 text-sm font-bold">
          {{ copy.fields.bizType }}
          <select v-model="bizType" class="h-11 rounded-xl border border-slate-200 px-3" :class="isZh ? '' : 'min-w-0 bg-white xl:pr-10'">
            <option value="">{{ copy.allTypes }}</option>
            <option v-for="option in localizedBizTypeOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </label>
        <label class="grid gap-2 text-sm font-bold">
          {{ copy.fields.orderStatus }}
          <select v-model="orderStatus" class="h-11 rounded-xl border border-slate-200 px-3" :class="isZh ? '' : 'min-w-0 bg-white xl:pr-10'">
            <option value="">{{ copy.allStatuses }}</option>
            <option v-for="option in localizedOrderStatusOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </label>
        <button class="mt-0 inline-flex h-11 items-center justify-center gap-2 rounded-xl bg-blue-700 px-5 text-sm font-black text-white shadow-sm xl:mt-7" type="submit">
          <Search class="h-4 w-4" />
          {{ copy.search }}
        </button>
      </div>
    </form>

    <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm md:rounded-3xl">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 p-4 md:p-5">
        <div class="min-w-0">
          <h2 class="text-xl font-black">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
        <span class="shrink-0 rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ copy.totalPrefix }} {{ total }} {{ copy.totalSuffix }}</span>
      </div>
      <div class="hidden grid-cols-[minmax(0,1fr)_160px_140px_150px_170px_112px] gap-4 border-b border-slate-200 bg-slate-50 px-5 py-3 text-xs font-black text-slate-500 md:grid">
        <span>{{ copy.columns.order }}</span>
        <span>{{ copy.columns.candidate }}</span>
        <span class="text-right">{{ copy.columns.amount }}</span>
        <span class="text-center">{{ copy.columns.status }}</span>
        <span>{{ copy.columns.createdAt }}</span>
        <span class="text-right">{{ copy.columns.action }}</span>
      </div>
      <div v-if="loading" class="p-8 text-center text-slate-500 md:p-12">
        <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
        {{ copy.loading }}
      </div>
      <div v-else-if="orders.length" class="divide-y divide-slate-100">
        <div
          v-for="order in orders"
          :key="orderUlid(order)"
          class="flex flex-col gap-3 px-4 py-4 transition hover:bg-sky-50 md:grid md:grid-cols-[minmax(0,1fr)_160px_140px_150px_170px_112px] md:items-center md:gap-4 md:px-5"
          :class="orderUlid(selected) === orderUlid(order) ? 'bg-sky-50' : ''"
        >
          <div class="min-w-0">
            <div class="break-words font-black text-slate-950 md:truncate">{{ productName(order) }}</div>
            <div class="mt-1 flex flex-wrap items-center gap-2 text-xs font-semibold text-slate-500">
              <span>{{ localizedLabelFor("bizTypes", biz(order), bizTypeOptions) }}</span>
              <span class="break-all rounded-full bg-slate-100 px-2 py-1">{{ copy.orderPrefix }} {{ orderUlid(order) || "-" }}</span>
            </div>
          </div>
          <div class="flex min-w-0 items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2 md:block md:rounded-none md:bg-transparent md:p-0">
            <span class="text-xs font-black text-slate-400 md:hidden">{{ copy.columns.candidate }}</span>
            <span class="break-all text-right text-sm font-semibold text-slate-600 md:text-left">{{ candidate(order) }}</span>
          </div>
          <div class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2 md:block md:rounded-none md:bg-transparent md:p-0">
            <span class="text-xs font-black text-slate-400 md:hidden">{{ copy.columns.amount }}</span>
            <span class="text-right text-sm font-black">{{ amountText(order) }}</span>
          </div>
          <div class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2 md:justify-center md:rounded-none md:bg-transparent md:p-0">
            <span class="text-xs font-black text-slate-400 md:hidden">{{ copy.columns.status }}</span>
            <span class="inline-flex rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(status(order))">
              {{ localizedLabelFor("orderStatuses", status(order), orderStatusOptions) }}
            </span>
          </div>
          <div class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2 md:block md:rounded-none md:bg-transparent md:p-0">
            <span class="text-xs font-black text-slate-400 md:hidden">{{ copy.columns.createdAt }}</span>
            <span class="text-right text-sm font-semibold text-slate-500 md:text-left">{{ createdAt(order) }}</span>
          </div>
          <div class="text-right">
            <button
              class="inline-flex w-full items-center justify-center rounded-xl border border-blue-100 bg-blue-50 px-3 py-2 text-sm font-bold text-blue-700 transition hover:underline md:w-auto md:border-0 md:bg-transparent md:px-0 md:py-0"
              type="button"
              @click.stop="selectOrder(order)"
            >
              {{ copy.viewDetails }}
            </button>
          </div>
        </div>
      </div>
      <div v-else class="p-8 text-center text-slate-500 md:p-12">{{ copy.empty }}</div>
      <div class="flex flex-col items-stretch justify-between gap-3 border-t border-slate-200 p-4 sm:flex-row sm:items-center md:p-5">
        <span class="text-center text-sm font-bold text-slate-500 sm:text-left">{{ copy.pagePrefix }} {{ page }} {{ copy.pageSuffix }}</span>
        <div class="flex flex-col gap-3 sm:flex-row">
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="loading || !canPrev" @click="load(page - 1)">{{ copy.prev }}</button>
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="loading || !canNext" @click="load(page + 1)">{{ copy.next }}</button>
        </div>
      </div>
    </section>

    <Teleport to="body">
      <div v-if="detailOpen && selected" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-0 md:p-6">
        <section v-modal-dialog="closeDetail" class="flex h-full max-h-none w-full max-w-[1280px] flex-col overflow-hidden rounded-none bg-white shadow-2xl md:h-auto md:max-h-[88vh] md:rounded-3xl">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-4 py-4 md:px-6 md:py-5">
            <div class="min-w-0">
              <h2 class="break-words text-xl font-black text-slate-950 md:truncate md:text-2xl">{{ productName(selected) }}</h2>
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

          <div class="border-b border-slate-200 px-4 py-3 md:px-5 md:py-4">
            <div class="flex gap-2 overflow-x-auto">
              <button
                v-for="tab in detailTabs"
                :key="tab.key"
                class="inline-flex h-11 shrink-0 items-center gap-2 rounded-2xl border px-3 text-sm font-black transition md:gap-3 md:px-4"
                :class="activeTab === tab.key ? 'border-sky-200 bg-sky-50 text-slate-950' : 'border-slate-100 bg-white text-slate-700 hover:bg-slate-50'"
                type="button"
                @click="activeTab = tab.key"
              >
                <span>{{ tab.title }}</span>
                <span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-black text-slate-600">{{ tab.count }}</span>
              </button>
            </div>
          </div>

          <main class="min-h-0 flex-1 overflow-y-auto p-4 md:h-[60vh] md:min-h-[360px] md:max-h-[620px] md:p-5">
              <div v-if="activeTab === 'summary'" class="space-y-5">
                <div class="rounded-2xl border border-blue-100 bg-blue-50 p-4">
                  <div class="flex flex-wrap items-start justify-between gap-4">
                    <div class="min-w-0">
                      <div class="text-xs font-black text-blue-600">{{ copy.currentOrder }}</div>
                      <div class="mt-1 break-words text-xl font-black text-slate-950 md:truncate">{{ productName(selected) }}</div>
                      <div class="mt-2 flex flex-wrap items-center gap-2">
                        <span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(status(selected))">
                          {{ localizedLabelFor("orderStatuses", status(selected), orderStatusOptions) }}
                        </span>
                        <span class="rounded-full bg-white px-3 py-1 text-xs font-black text-slate-600">
                          {{ localizedLabelFor("paymentStatuses", payStatus(selected), paymentStatusOptions) }}
                        </span>
                      </div>
                    </div>
                    <div class="w-full rounded-2xl border border-blue-100 bg-white px-5 py-4 text-left shadow-sm sm:w-auto sm:text-right">
                      <div class="text-xs font-black text-slate-400">{{ copy.orderAmount }}</div>
                      <div class="mt-1 text-2xl font-black text-blue-800">{{ amountText(selected) }}</div>
                    </div>
                  </div>
                </div>

                <section class="border-y border-slate-200 py-5">
                  <div class="flex flex-wrap items-start justify-between gap-3">
                    <div>
                      <h3 class="text-base font-black text-slate-950">{{ copy.pricing.title }}</h3>
                      <p class="mt-1 text-sm text-slate-500">{{ copy.pricing.description }}</p>
                    </div>
                    <span v-if="pricing?.source" class="rounded-full bg-slate-100 px-3 py-1 text-xs font-bold text-slate-600">
                      {{ pricingSourceLabel(pricing.source) }}
                    </span>
                  </div>

                  <div v-if="detailLoading" class="flex min-h-28 items-center justify-center text-sm text-slate-500">
                    <Loader2 class="mr-2 h-5 w-5 animate-spin" />
                    {{ copy.pricing.loading }}
                  </div>
                  <template v-else-if="pricing">
                    <div class="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
                      <div
                        v-for="metric in priceMetrics"
                        :key="metric.key"
                        class="min-h-28 rounded-lg border p-4"
                        :class="metric.tone"
                      >
                        <div class="text-xs font-black">{{ metric.label }}</div>
                        <div class="mt-2 break-words text-lg font-black">{{ metric.value }}</div>
                        <div class="mt-2 text-xs font-medium opacity-70">{{ metric.note }}</div>
                      </div>
                    </div>

                    <div v-if="pricing.unavailable_reason" class="mt-3 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm font-medium text-amber-900">
                      {{ copy.pricing.partialUnavailable }}
                    </div>

                    <div v-if="promotionLabels.length" class="mt-5 flex flex-wrap items-center gap-2">
                      <span class="text-xs font-black text-slate-500">{{ copy.pricing.promotions }}</span>
                      <span
                        v-for="label in promotionLabels"
                        :key="label"
                        class="rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1 text-xs font-bold text-emerald-800"
                      >
                        {{ label }}
                      </span>
                    </div>

                    <div v-if="exemptions.length" class="mt-5">
                      <div class="text-xs font-black text-slate-500">{{ copy.pricing.exemptedItems }}</div>
                      <div class="mt-2 grid gap-2 sm:grid-cols-2">
                        <div v-for="item in exemptions" :key="String(item.course_cc_ulid)" class="min-w-0 border-l-2 border-amber-300 bg-amber-50 px-3 py-2">
                          <div class="break-all text-xs font-bold text-amber-950">{{ item.course_cc_ulid }}</div>
                          <div v-if="item.credential_ulid" class="mt-1 break-all text-xs text-amber-700">{{ item.credential_ulid }}</div>
                        </div>
                      </div>
                    </div>

                    <div v-if="priceItems.length" class="mt-5 overflow-hidden border-y border-slate-200">
                      <div class="bg-slate-50 px-4 py-3 text-sm font-black text-slate-800">{{ copy.pricing.itemsTitle }}</div>
                      <div class="overflow-x-auto">
                        <table class="w-full min-w-[620px] text-left text-sm">
                          <thead class="border-y border-slate-200 bg-white text-xs text-slate-500">
                            <tr>
                              <th class="px-4 py-3 font-black">{{ copy.pricing.item }}</th>
                              <th class="px-4 py-3 font-black">{{ copy.pricing.unitPrice }}</th>
                              <th class="px-4 py-3 text-center font-black">{{ copy.pricing.quantity }}</th>
                              <th class="px-4 py-3 text-right font-black">{{ copy.pricing.subtotal }}</th>
                            </tr>
                          </thead>
                          <tbody class="divide-y divide-slate-100">
                            <tr v-for="item in priceItems" :key="`${item.item_type}-${item.item_ulid}`">
                              <td class="px-4 py-3">
                                <div class="font-bold text-slate-900">{{ item.title || item.item_ulid || copy.pricing.unnamedItem }}</div>
                                <div class="mt-1 break-all text-xs text-slate-500">{{ item.item_type }}<span v-if="item.item_ulid"> · {{ item.item_ulid }}</span></div>
                              </td>
                              <td class="px-4 py-3 font-semibold text-slate-700">{{ minorAmountText(item.unit_price_minor, pricingCurrency) }}</td>
                              <td class="px-4 py-3 text-center font-semibold text-slate-700">{{ item.quantity }}</td>
                              <td class="px-4 py-3 text-right font-black text-slate-900">{{ minorAmountText(item.subtotal_minor, pricingCurrency) }}</td>
                            </tr>
                          </tbody>
                        </table>
                      </div>
                    </div>
                  </template>
                  <div v-else class="mt-4 rounded-lg border border-dashed border-slate-300 p-8 text-center text-sm text-slate-500">
                    {{ copy.pricing.unavailableDetail }}
                  </div>
                </section>

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

              <div v-else-if="activeTab === 'business-detail'" class="space-y-4">
                <div v-if="detailLoading" class="p-12 text-center text-slate-500">
                  <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
                  {{ copy.bundleLoading }}
                </div>
                <div v-else-if="businessDetail" class="space-y-4">
                  <div class="grid gap-4 md:grid-cols-2">
                    <div
                      v-for="field in businessSummaryFields"
                      :key="field.label"
                      class="rounded-2xl border border-slate-200 bg-slate-50 p-4"
                    >
                      <div class="text-xs font-black uppercase text-slate-400">{{ field.label }}</div>
                      <div class="mt-2 break-all text-sm font-black text-slate-800">{{ field.value }}</div>
                    </div>
                  </div>
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
                  class="inline-flex h-11 w-full items-center justify-center gap-2 rounded-xl bg-red-600 px-5 text-sm font-bold text-white shadow-sm shadow-red-200 disabled:opacity-50 sm:w-auto"
                  type="button"
                  :disabled="!isBundlePurchase || Boolean(purging)"
                  @click="showPurgeConfirm = true"
                >
                  <Trash2 class="h-4 w-4" />
                  {{ copy.purgeAction }}
                </button>
              </div>

            </main>
        </section>
      </div>
    </Teleport>

    <div v-if="showPurgeConfirm" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4 md:p-6">
      <div v-modal-dialog="closePurgeConfirm" class="w-full max-w-md rounded-2xl bg-white p-4 shadow-2xl md:rounded-3xl md:p-6">
        <h2 class="text-xl font-black md:text-2xl">{{ copy.confirmTitle }}</h2>
        <p class="mt-3 text-sm text-slate-600">{{ copy.confirmDescription }}</p>
        <div class="mt-5 rounded-2xl bg-slate-50 p-4">
          <div class="font-black">{{ productName(selected) }}</div>
          <div class="mt-1 break-all text-xs text-slate-500">{{ bizRef(selected) }}</div>
        </div>
        <div class="mt-6 flex flex-col items-stretch justify-end gap-3 sm:flex-row sm:items-center">
          <button data-dialog-initial-focus class="inline-flex h-11 min-w-[96px] items-center justify-center rounded-xl border px-5 text-sm font-bold disabled:opacity-50" type="button" :disabled="Boolean(purging)" @click="closePurgeConfirm">{{ copy.cancel }}</button>
          <button class="inline-flex h-11 min-w-[112px] items-center justify-center rounded-xl bg-red-600 px-5 text-sm font-bold text-white disabled:opacity-50" type="button" :disabled="Boolean(purging)" @click="purgeSelected">
            {{ purging ? copy.purging : copy.confirmPurge }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
