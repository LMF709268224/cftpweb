<script setup lang="ts">
import { Loader2, RefreshCw, Search, Trash2, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import JsonPreview from "@/components/JsonPreview.vue"
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

type DetailTab = "summary" | "bundle-detail" | "actions" | "raw"
type SummaryField = {
  label: string
  value: string
}

const { t } = useAdminLanguage()
const copy = computed(() => t.value.orders)

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
const hasMore = ref(false)
const nextCursor = ref("")
const prevCursor = ref("")
const lastPage = ref(1)
const activeTab = ref<DetailTab>("summary")

const candidateUlid = ref("")
const bizType = ref("")
const orderStatus = ref("")
const paymentStatus = ref("")

const canPrev = computed(() => page.value > 1)
const canNext = computed(() => hasMore.value)
const isBundlePurchase = computed(() => normalizeStatus(biz(selected.value || {})) === "BUNDLE_PURCHASE")
const localizedBizTypeOptions = computed(() => localizeOptions(bizTypeOptions, "bizTypes"))
const localizedOrderStatusOptions = computed(() => localizeOptions(orderStatusOptions, "orderStatuses"))
const localizedPaymentStatusOptions = computed(() => localizeOptions(paymentStatusOptions, "paymentStatuses"))
const selectedJson = computed(() => JSON.stringify(selected.value || {}, null, 2))
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
    if (paymentStatus.value) params.set("payment_status", paymentStatus.value)

    const data = await apiClient<JsonRecord>(`/api/mall/orders?${params}`)
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
      <button class="inline-flex h-10 items-center justify-center gap-2 rounded-xl bg-blue-700 px-5 text-sm font-bold text-white" type="submit">
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
      <div class="grid grid-cols-[minmax(0,1fr)_160px_140px_150px_170px_112px] gap-4 border-b border-slate-200 bg-slate-50 px-5 py-3 text-xs font-black text-slate-500">
        <span>{{ copy.columns.order }}</span>
        <span>{{ copy.columns.candidate }}</span>
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
          class="grid cursor-pointer grid-cols-[minmax(0,1fr)_160px_140px_150px_170px_112px] items-center gap-4 px-5 py-4 transition hover:bg-sky-50"
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
            </div>
          </div>
          <div class="min-w-0 break-all text-sm font-semibold text-slate-600">{{ candidate(order) }}</div>
          <div class="text-right text-sm font-black">{{ amountText(order) }}</div>
          <div class="flex items-center justify-center gap-2">
            <span class="inline-flex rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(status(order))">
              {{ localizedLabelFor("orderStatuses", status(order), orderStatusOptions) }}
            </span>
          </div>
          <div class="text-sm font-semibold text-slate-500">{{ createdAt(order) }}</div>
          <div class="text-right">
            <button
              class="text-sm font-bold text-[#1890ff] transition hover:underline"
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
                  <JsonPreview
                    :title="copy.bundleRaw"
                    :value="bundleDetail"
                    :copy-label="copy.copyJson"
                    :copied-label="copy.copiedJson"
                    :copied-message="copy.toasts.jsonCopied"
                    :copy-error-message="copy.toasts.jsonCopyFailed"
                    max-height="520px"
                  />
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
                <JsonPreview
                  :title="copy.rawJson"
                  :text="selectedJson"
                  :copy-label="copy.copyJson"
                  :copied-label="copy.copiedJson"
                  :copied-message="copy.toasts.jsonCopied"
                  :copy-error-message="copy.toasts.jsonCopyFailed"
                  max-height="620px"
                />
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
        <div class="mt-6 flex items-center justify-end gap-3">
          <button class="inline-flex h-11 min-w-[96px] items-center justify-center rounded-xl border px-5 text-sm font-bold disabled:opacity-50" type="button" :disabled="Boolean(purging)" @click="showPurgeConfirm = false">{{ copy.cancel }}</button>
          <button class="inline-flex h-11 min-w-[112px] items-center justify-center rounded-xl bg-red-600 px-5 text-sm font-bold text-white disabled:opacity-50" type="button" :disabled="Boolean(purging)" @click="purgeSelected">
            {{ purging ? copy.purging : copy.confirmPurge }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
