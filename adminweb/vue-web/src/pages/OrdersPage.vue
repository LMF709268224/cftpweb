<script setup lang="ts">
import { ArrowUpRight, ChevronRight, Inbox, RefreshCw, Search, Trash2, TriangleAlert, X } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import CandidatePicker from "@/components/CandidatePicker.vue"
import UlidChip from "@/components/UlidChip.vue"
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
  kind?: "ulid" | "candidate"
  ulid?: string
  name?: string
}

const { t } = useAdminLanguage()
const copy = computed(() => t.value.orders)

const orders = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const businessDetail = ref<JsonRecord | null>(null)
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
const statusChips = computed(() => [
  { value: "", label: copy.value.allStatuses },
  ...localizedOrderStatusOptions.value,
])
const detailTabs = computed(() => [
  { key: "summary" as const, title: copy.value.tabs.summary, count: selected.value ? 1 : 0 },
  { key: "business-detail" as const, title: copy.value.tabs.bundleDetail, count: businessDetail.value ? 1 : 0 },
  { key: "actions" as const, title: copy.value.tabs.actions, count: isBundlePurchase.value ? 1 : 0 },
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
    {
      label: copy.value.fields.candidate,
      value: candidate(order),
      kind: "candidate" as const,
      ulid: candidateUlidOf(order),
      name: candidateName(order),
    },
    { label: copy.value.fields.orderId, value: orderUlid(order), kind: "ulid" as const, ulid: orderUlid(order) },
    {
      label: copy.value.fields.payOrderId,
      value: stringValue(pickFirst(order, ["pay_order_ulid", "payOrderUlid"])),
      kind: "ulid" as const,
      ulid: String(pickFirst(order, ["pay_order_ulid", "payOrderUlid"]) || ""),
    },
    { label: copy.value.fields.bizRefId, value: bizRef(order) || "-", kind: "ulid" as const, ulid: bizRef(order) },
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

function candidateName(order: JsonRecord | null | undefined) {
  return String(pickFirst(order || {}, ["candidate_name", "candidate_email"]) || "")
}

function candidateUlidOf(order: JsonRecord | null | undefined) {
  return String(pickFirst(order || {}, ["candidate_ulid", "candidateUlid", "candidate_id"]) || "")
}

const ULID_LIKE = /^[0-9A-HJKMNP-TV-Z]{26}$/i

function isUlidLike(value: unknown) {
  return ULID_LIKE.test(String(value ?? "").trim())
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

function isTimeField(key: string) {
  return /(^|_)(created|updated|paid|expired|completed)_at$|At$/i.test(key)
}

function businessFieldLabel(key: string) {
  return copy.value.businessFields?.[key as keyof typeof copy.value.businessFields] || key.replaceAll("_", " ")
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

async function loadBusinessDetail(order: JsonRecord | null) {
  businessDetail.value = null
  if (!order || !orderUlid(order)) return
  detailLoading.value = true
  try {
    const response = await apiClient<JsonRecord>(`/api/mall/orders/${encodeURIComponent(orderUlid(order))}`)
    businessDetail.value = recordValue(response.business_detail)
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
  await loadBusinessDetail(order)
}

function closeDetail() {
  detailOpen.value = false
}

async function load(targetPage = page.value) {
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
      detailOpen.value = false
      total.value = 0
      hasMore.value = false
      nextCursor.value = ""
      prevCursor.value = ""
      return
    }

    const data = await apiClient<JsonRecord>(`/api/mall/orders?${params}`)
    const list = Array.isArray(data.items) ? data.items : Array.isArray(data.orders) ? data.orders : []

    orders.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    total.value = Number(data.total ?? data.total_count ?? data.totalCount ?? orders.value.length) || 0
    const isBackward = page.value < lastPage.value
    hasMore.value = isBackward ? true : Boolean(data.has_more)
    nextCursor.value = String(data.next_cursor || "")
    prevCursor.value = String(data?.prev_cursor || "")

    lastPage.value = targetPage
    page.value = targetPage
    if (orders.value.length) {
      await selectOrder(orders.value[0], detailOpen.value)
    } else {
      selected.value = null
      businessDetail.value = null
      detailOpen.value = false
    }
  } catch (err) {
    console.error(err)
    orders.value = []
    selected.value = null
    businessDetail.value = null
    detailOpen.value = false
    total.value = 0
    hasMore.value = false
    nextCursor.value = ""
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

function setOrderStatus(value: string) {
  orderStatus.value = value
  search()
}

watch(candidateUlid, () => {
  search()
})

onMounted(() => load(1))
</script>

<template>
  <main class="ui-page">
    <header class="flex flex-wrap items-start justify-between gap-4 md:items-center">
      <div class="min-w-0">
        <h1 class="ui-page-title">{{ copy.pageTitle }}</h1>
        <p class="ui-page-subtitle">{{ copy.pageDescription }}</p>
      </div>
      <button class="ui-btn ui-btn-secondary" type="button" @click="load(page)">
        <RefreshCw class="h-4 w-4 text-slate-400" :class="loading ? 'animate-spin' : ''" />
        {{ copy.refresh }}
      </button>
    </header>

    <form class="ui-card mt-6 p-4 md:p-5" @submit.prevent="search">
      <div class="grid gap-4 xl:grid-cols-[minmax(0,1fr)_220px_auto]">
        <div class="grid gap-1.5">
          <span class="ui-label">{{ copy.candidatePlaceholder }}</span>
          <CandidatePicker v-model="candidateUlid" />
        </div>
        <label class="grid gap-1.5">
          <span class="ui-label">{{ copy.fields.bizType }}</span>
          <select v-model="bizType" class="ui-input" @change="search()">
            <option value="">{{ copy.allTypes }}</option>
            <option v-for="option in localizedBizTypeOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </label>
        <button class="ui-btn ui-btn-primary xl:self-end" type="submit">
          <Search class="h-4 w-4" />
          {{ copy.search }}
        </button>
      </div>
      <div class="mt-4 flex flex-wrap items-center gap-2">
        <span class="ui-label mr-1">{{ copy.fields.orderStatus }}</span>
        <button
          v-for="chip in statusChips"
          :key="chip.value"
          type="button"
          class="inline-flex h-8 shrink-0 items-center gap-1.5 whitespace-nowrap rounded-full px-3.5 text-sm font-medium transition-colors duration-150 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary-500/20"
          :class="orderStatus === chip.value
            ? 'bg-primary-600 text-white shadow-sm'
            : 'border border-slate-200 bg-white text-slate-600 hover:border-slate-300 hover:bg-slate-50 hover:text-slate-900'"
          :aria-pressed="orderStatus === chip.value"
          @click="setOrderStatus(chip.value)"
        >
          {{ chip.label }}
        </button>
      </div>
    </form>

    <section class="ui-card mt-4 overflow-hidden">
      <div class="flex flex-wrap items-center justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
        <div class="min-w-0">
          <h2 class="text-base font-semibold text-ink">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
        <span class="tnum ui-badge shrink-0 bg-slate-100 text-slate-600">{{ copy.totalPrefix }} {{ total }} {{ copy.totalSuffix }}</span>
      </div>
      <div class="hidden grid-cols-[minmax(0,1fr)_160px_140px_150px_170px_112px] gap-4 border-b border-edge bg-slate-50/60 px-4 py-3 text-xs font-medium uppercase tracking-wider text-slate-400 md:grid">
        <span>{{ copy.columns.order }}</span>
        <span>{{ copy.columns.candidate }}</span>
        <span class="text-right">{{ copy.columns.amount }}</span>
        <span class="text-center">{{ copy.columns.status }}</span>
        <span>{{ copy.columns.createdAt }}</span>
        <span class="text-right">{{ copy.columns.action }}</span>
      </div>
      <div v-if="loading">
        <div class="space-y-4 px-4 py-4 md:hidden">
          <div v-for="i in 4" :key="`msk-${i}`" class="space-y-2.5">
            <div class="skeleton h-4 w-2/5" />
            <div class="skeleton h-3.5 w-4/5" />
            <div class="skeleton h-3.5 w-3/5" />
          </div>
        </div>
        <div class="hidden md:block">
          <div
            v-for="i in 5"
            :key="`sk-${i}`"
            class="grid grid-cols-[minmax(0,1fr)_160px_140px_150px_170px_112px] items-center gap-4 border-b border-hairline px-4 py-3.5 last:border-b-0"
          >
            <div>
              <div class="skeleton h-4 w-40" />
              <div class="skeleton mt-2 h-3 w-56" />
            </div>
            <div class="skeleton h-4 w-24" />
            <div class="skeleton ml-auto h-4 w-16" />
            <div class="skeleton mx-auto h-5 w-16 rounded-full" />
            <div class="skeleton h-4 w-28" />
            <div class="skeleton ml-auto h-4 w-16" />
          </div>
        </div>
      </div>
      <div v-else-if="orders.length" class="divide-y divide-hairline">
        <div
          v-for="order in orders"
          :key="orderUlid(order)"
          class="ui-table-row flex cursor-pointer flex-col gap-3 px-4 py-4 md:grid md:grid-cols-[minmax(0,1fr)_160px_140px_150px_170px_112px] md:items-center md:gap-4 md:py-3.5"
          :class="orderUlid(selected) === orderUlid(order) ? 'bg-primary-50/60' : ''"
          role="button"
          tabindex="0"
          @click="selectOrder(order)"
          @keydown.enter.prevent="selectOrder(order)"
          @keydown.space.prevent="selectOrder(order)"
        >
          <div class="min-w-0">
            <div class="break-words text-sm font-medium text-ink md:truncate">{{ productName(order) }}</div>
            <div class="mt-1 flex flex-wrap items-center gap-2 text-xs text-slate-400">
              <span>{{ localizedLabelFor("bizTypes", biz(order), bizTypeOptions) }}</span>
              <span class="inline-flex items-center gap-1">
                <span>{{ copy.orderPrefix }}</span>
                <UlidChip :value="orderUlid(order)" />
              </span>
            </div>
          </div>
          <div class="flex min-w-0 items-center justify-between gap-4 border-t border-hairline pt-3 md:block md:border-0 md:pt-0">
            <span class="shrink-0 text-xs font-medium text-slate-400 md:hidden">{{ copy.columns.candidate }}</span>
            <span v-if="candidateName(order)" class="break-all text-right text-sm text-slate-600 md:text-left">{{ candidateName(order) }}</span>
            <UlidChip v-else :value="candidateUlidOf(order)" />
          </div>
          <div class="flex items-center justify-between gap-4 border-t border-hairline pt-3 md:block md:border-0 md:pt-0 md:text-right">
            <span class="shrink-0 text-xs font-medium text-slate-400 md:hidden">{{ copy.columns.amount }}</span>
            <span class="tnum text-sm font-semibold text-ink">{{ amountText(order) }}</span>
          </div>
          <div class="flex items-center justify-between gap-4 border-t border-hairline pt-3 md:justify-center md:border-0 md:pt-0">
            <span class="shrink-0 text-xs font-medium text-slate-400 md:hidden">{{ copy.columns.status }}</span>
            <span class="ui-badge" :class="badgeClass(status(order))">
              {{ localizedLabelFor("orderStatuses", status(order), orderStatusOptions) }}
            </span>
          </div>
          <div class="flex items-center justify-between gap-4 border-t border-hairline pt-3 md:block md:border-0 md:pt-0">
            <span class="shrink-0 text-xs font-medium text-slate-400 md:hidden">{{ copy.columns.createdAt }}</span>
            <span class="tnum text-right text-sm text-slate-500 md:text-left">{{ createdAt(order) }}</span>
          </div>
          <div class="border-t border-hairline pt-3 text-right md:border-0 md:pt-0">
            <button
              class="inline-flex w-full items-center justify-center gap-1 rounded-[10px] border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-primary-700 transition-colors hover:bg-primary-50 md:w-auto md:border-0 md:bg-transparent md:p-0 md:hover:bg-transparent md:hover:text-primary-600"
              type="button"
              @click.stop="selectOrder(order)"
            >
              {{ copy.viewDetails }}
              <ChevronRight class="h-4 w-4" />
            </button>
          </div>
        </div>
      </div>
      <div v-else class="ui-empty">
        <div class="ui-empty-icon">
          <Inbox class="h-6 w-6" />
        </div>
        <p class="ui-empty-title">{{ copy.empty }}</p>
      </div>
      <div class="flex flex-col items-stretch justify-between gap-3 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center md:px-6">
        <span class="tnum text-center text-sm text-slate-500 sm:text-left">{{ copy.pagePrefix }} {{ page }} {{ copy.pageSuffix }}</span>
        <div class="flex flex-col gap-2 sm:flex-row">
          <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canPrev" @click="load(page - 1)">{{ copy.prev }}</button>
          <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canNext" @click="load(page + 1)">{{ copy.next }}</button>
        </div>
      </div>
    </section>

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="detailOpen && selected" class="ui-modal-mask">
          <section class="ui-modal-panel max-w-[1280px]">
            <div class="flex items-start justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
              <div class="min-w-0">
                <h2 class="break-words text-base font-semibold text-ink md:truncate md:text-lg">{{ productName(selected) }}</h2>
                <div class="mt-1">
                  <UlidChip :value="orderUlid(selected)" />
                </div>
              </div>
              <button
                class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
                type="button"
                :aria-label="copy.close"
                @click="closeDetail"
              >
                <X class="h-5 w-5" />
              </button>
            </div>

            <div class="flex shrink-0 gap-5 overflow-x-auto border-b border-hairline px-4 md:px-6">
              <button
                v-for="tab in detailTabs"
                :key="tab.key"
                class="ui-tab"
                :class="activeTab === tab.key ? 'ui-tab-active' : ''"
                type="button"
                @click="activeTab = tab.key"
              >
                <span>{{ tab.title }}</span>
                <span class="tnum ui-badge bg-slate-100 text-slate-600">{{ tab.count }}</span>
              </button>
            </div>

            <main class="min-h-0 flex-1 overflow-y-auto p-4 md:h-[60vh] md:min-h-[360px] md:max-h-[620px] md:p-6">
              <div v-if="activeTab === 'summary'" class="space-y-5">
                <div class="rounded-2xl bg-primary-50 p-4 md:p-5">
                  <div class="flex flex-wrap items-start justify-between gap-4">
                    <div class="min-w-0">
                      <div class="text-xs font-medium text-primary-700">{{ copy.currentOrder }}</div>
                      <div class="mt-1 break-words text-base font-semibold text-ink md:truncate">{{ productName(selected) }}</div>
                      <div class="mt-2 flex flex-wrap items-center gap-2">
                        <span class="ui-badge" :class="badgeClass(status(selected))">
                          {{ localizedLabelFor("orderStatuses", status(selected), orderStatusOptions) }}
                        </span>
                        <span class="ui-badge bg-slate-100 text-slate-600">
                          {{ localizedLabelFor("paymentStatuses", payStatus(selected), paymentStatusOptions) }}
                        </span>
                      </div>
                    </div>
                    <div class="ui-card w-full px-5 py-4 text-left sm:w-auto sm:text-right">
                      <div class="text-xs font-medium text-slate-400">{{ copy.orderAmount }}</div>
                      <div class="tnum mt-1 text-2xl font-bold text-primary-700">{{ amountText(selected) }}</div>
                    </div>
                  </div>
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <div
                    v-for="field in orderSummaryFields"
                    :key="field.label"
                    class="rounded-2xl bg-slate-50 p-4"
                  >
                    <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ field.label }}</div>
                    <div v-if="field.kind === 'ulid'" class="mt-2">
                      <UlidChip :value="field.ulid || ''" />
                    </div>
                    <div v-else-if="field.kind === 'candidate'" class="mt-2 flex flex-wrap items-center gap-2">
                      <RouterLink
                        v-if="field.ulid && field.name"
                        :to="{ path: '/prog', query: { candidate: field.ulid } }"
                        class="inline-flex items-center gap-1 break-all text-sm font-medium text-slate-700 underline decoration-slate-300 decoration-dashed underline-offset-4 transition-colors hover:text-primary-700 hover:decoration-primary-400"
                        @click.stop
                      >
                        {{ field.name }}
                        <ArrowUpRight class="h-3.5 w-3.5 shrink-0 text-slate-400" />
                      </RouterLink>
                      <UlidChip v-if="field.ulid" :value="field.ulid" />
                      <RouterLink
                        v-if="field.ulid && !field.name"
                        :to="{ path: '/prog', query: { candidate: field.ulid } }"
                        class="inline-flex h-6 w-6 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-primary-700"
                        :aria-label="t.common.openLink"
                        :title="t.common.openLink"
                        @click.stop
                      >
                        <ArrowUpRight class="h-3.5 w-3.5" />
                      </RouterLink>
                      <span v-if="!field.ulid" class="tnum break-all text-sm font-medium text-ink">{{ field.value }}</span>
                    </div>
                    <div v-else class="tnum mt-2 break-all text-sm font-medium text-ink">{{ field.value }}</div>
                  </div>
                </div>
              </div>

              <div v-else-if="activeTab === 'business-detail'" class="space-y-4">
                <div v-if="detailLoading" class="grid gap-4 md:grid-cols-2">
                  <div v-for="i in 4" :key="`bd-sk-${i}`" class="skeleton h-20 w-full" />
                </div>
                <div v-else-if="businessDetail" class="space-y-4">
                  <div class="grid gap-4 md:grid-cols-2">
                    <div
                      v-for="field in businessSummaryFields"
                      :key="field.label"
                      class="rounded-2xl bg-slate-50 p-4"
                    >
                      <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ field.label }}</div>
                      <div v-if="isUlidLike(field.value)" class="mt-2">
                        <UlidChip :value="field.value" />
                      </div>
                      <div v-else class="tnum mt-2 break-all text-sm font-medium text-ink">{{ field.value }}</div>
                    </div>
                  </div>
                </div>
                <div v-else class="ui-empty">
                  <div class="ui-empty-icon">
                    <Inbox class="h-6 w-6" />
                  </div>
                  <p class="ui-empty-title">{{ copy.bundleEmpty }}</p>
                </div>
              </div>

              <div v-else-if="activeTab === 'actions'" class="space-y-4">
                <div class="rounded-2xl bg-slate-50 p-5">
                  <div class="text-base font-semibold text-ink">{{ copy.actionsTitle }}</div>
                  <p class="mt-2 text-sm text-slate-600">
                    {{ copy.actionsDescription }}
                  </p>
                </div>
                <section class="rounded-2xl border border-danger-200 bg-danger-50/40 p-4 md:p-5">
                  <h3 class="flex items-center gap-2 text-sm font-semibold text-danger-600">
                    <TriangleAlert class="h-4 w-4" />
                    {{ t.common.dangerZone }}
                  </h3>
                  <div class="mt-3 flex flex-col items-stretch justify-between gap-3 sm:flex-row sm:items-center">
                    <p class="text-sm text-slate-600">{{ copy.confirmDescription }}</p>
                    <button
                      class="ui-btn ui-btn-danger shrink-0"
                      type="button"
                      :disabled="!isBundlePurchase || Boolean(purging)"
                      :title="!isBundlePurchase ? t.common.reasonUnavailable : undefined"
                      @click="showPurgeConfirm = true"
                    >
                      <Trash2 class="h-4 w-4" />
                      {{ copy.purgeAction }}
                    </button>
                  </div>
                </section>
              </div>

            </main>
          </section>
        </div>
      </Transition>
    </Teleport>

    <Transition name="modal">
      <div v-if="showPurgeConfirm" class="ui-modal-mask z-50">
        <section class="ui-modal-panel md:max-w-md">
          <div class="border-b border-hairline px-4 py-4 md:px-6 md:py-5">
            <h2 class="text-base font-semibold text-ink md:text-lg">{{ copy.confirmTitle }}</h2>
          </div>
          <main class="min-h-0 flex-1 overflow-y-auto p-4 md:p-6">
            <p class="text-sm text-slate-600">{{ copy.confirmDescription }}</p>
            <div class="mt-4 rounded-2xl bg-slate-50 p-4">
              <div class="text-sm font-medium text-ink">{{ productName(selected) }}</div>
              <div class="mt-1.5">
                <UlidChip :value="bizRef(selected)" />
              </div>
            </div>
          </main>
          <div class="flex flex-col items-stretch justify-end gap-3 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center md:px-6">
            <button class="ui-btn ui-btn-secondary" type="button" :disabled="Boolean(purging)" @click="showPurgeConfirm = false">{{ copy.cancel }}</button>
            <button class="ui-btn ui-btn-danger" type="button" :disabled="Boolean(purging)" @click="purgeSelected">
              {{ purging ? copy.purging : copy.confirmPurge }}
            </button>
          </div>
        </section>
      </div>
    </Transition>
  </main>
</template>
