<script setup lang="ts">
import { Receipt, RefreshCw, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { badgeClass, labelFor, normalizeStatus, orderStatusOptions, pickFirst } from "@/lib/status"

const invoices = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const loading = ref(false)
const detailOpen = ref(false)
const page = ref(1)
const total = ref(0)
const pageSize = 20
const hasMore = ref(false)
const nextCursor = ref("")
const prevCursor = ref("")
const lastPage = ref(1)
const { t } = useAdminLanguage()
const copy = computed(() => t.value.invoices)
const summaryFieldKeys = new Set(["order_id", "order_ulid", "status"])

const canPrev = computed(() => page.value > 1)
const canNext = computed(() => hasMore.value)
const selectedFields = computed(() =>
  Object.entries(selected.value || {})
    .filter(([key]) => !summaryFieldKeys.has(key))
    .map(([key, value]) => ({
      key,
      label: copy.value.fieldLabels[key as keyof typeof copy.value.fieldLabels] || key.replace(/_/g, " "),
      value,
      displayValue: key.endsWith("_at") ? formatDate(value) : String(value ?? "-"),
    })),
)

function invoiceId(invoice: JsonRecord | null | undefined) {
  return String(pickFirst(invoice || {}, ["id", "invoice_id", "invoice_ulid"]) || "")
}

function orderId(invoice: JsonRecord | null | undefined) {
  return String(pickFirst(invoice || {}, ["order_id", "order_ulid"]) || "-")
}

function amountText(invoice: JsonRecord | null | undefined) {
  const amount = Number(invoice?.amount || 0)
  return `${Number.isFinite(amount) ? amount.toFixed(2) : "0.00"} ${invoice?.currency || ""}`.trim()
}

function normalizedInvoiceStatus(value: unknown) {
  return normalizeStatus(value)
    .replace(/^ORDER_STATUS_/, "")
    .replace(/^INVOICE_STATUS_/, "")
    .replace(/^PAYMENT_STATUS_/, "")
}

function invoiceStatusLabel(value: unknown) {
  const normalized = normalizedInvoiceStatus(value)
  if (!normalized) return "-"
  return copy.value.statuses[normalized as keyof typeof copy.value.statuses] || labelFor(orderStatusOptions, normalized)
}

function isStructuredValue(value: unknown) {
  return Array.isArray(value) || (!!value && typeof value === "object")
}

function jsonText(value: unknown) {
  return JSON.stringify(value ?? {}, null, 2)
}

function detailFieldText(value: unknown) {
  const text = String(value ?? "").trim()
  return text || "-"
}

function openInvoice(invoice: JsonRecord | null, open = true) {
  selected.value = invoice
  detailOpen.value = open
}

function closeDetail() {
  detailOpen.value = false
}

async function load(targetPage = page.value) {
  loading.value = true
  try {
    const params = new URLSearchParams({ page_size: String(pageSize) })

    let cursor = ""

    if (targetPage > lastPage.value) {

      cursor = nextCursor.value

    } else if (targetPage < lastPage.value) {

      cursor = prevCursor.value


    }

    

    if (cursor) params.set("cursor", cursor)


    const data = await apiClient<JsonRecord>(`/api/mall/invoices?${params}`)
    const list = Array.isArray(data.invoices) ? data.invoices : []

    invoices.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    total.value = Number(data.total || invoices.value.length) || 0
    const isBackward = page.value < lastPage.value
    hasMore.value = isBackward ? true : Boolean(data.has_more)
    lastPage.value = page.value
nextCursor.value = String(data.next_cursor || "")
    prevCursor.value = String(data?.prev_cursor || "")

    lastPage.value = targetPage
    page.value = targetPage
    if (!selected.value || !invoices.value.some((item) => invoiceId(item) === invoiceId(selected.value))) {
      openInvoice(invoices.value[0] || null, false)
    }
    if (!invoices.value.length) detailOpen.value = false
  } catch (err) {
    console.error(err)
    invoices.value = []
    selected.value = null
    detailOpen.value = false
    hasMore.value = false
    nextCursor.value = ""
    toast.error(apiErrorMessage(err, copy.value.toasts.loadFailed))
  } finally {
    loading.value = false
  }
}

onMounted(() => load(1))
</script>

<template>
  <main class="ui-page">
    <header class="flex flex-wrap items-start justify-between gap-4 md:items-center">
      <div class="min-w-0">
        <h1 class="ui-page-title">{{ copy.title }}</h1>
        <p class="ui-page-subtitle">{{ copy.subtitle }}</p>
      </div>
      <button class="ui-btn ui-btn-secondary" type="button" @click="load(page)">
        <RefreshCw class="h-4 w-4 text-slate-400" :class="loading ? 'animate-spin' : ''" />
        {{ copy.refresh }}
      </button>
    </header>

    <section class="ui-card mt-6 overflow-hidden">
      <div class="flex flex-wrap items-center justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
        <div class="min-w-0">
          <h2 class="text-base font-semibold text-ink">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
        <span class="tnum ui-badge shrink-0 bg-slate-100 text-slate-600">{{ copy.totalText(total) }}</span>
      </div>
      <div class="hidden grid-cols-[minmax(0,1fr)_120px_240px_180px_112px] gap-5 border-b border-edge bg-slate-50/60 px-4 py-3 text-xs font-medium uppercase tracking-wider text-slate-400 md:grid">
        <span>{{ copy.columns.invoice }}</span>
        <span class="text-right">{{ copy.columns.amount }}</span>
        <span class="text-center">{{ copy.columns.status }}</span>
        <span class="text-right">{{ copy.columns.createdAt }}</span>
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
            class="grid grid-cols-[minmax(0,1fr)_120px_240px_180px_112px] items-center gap-5 border-b border-hairline px-4 py-3.5 last:border-b-0"
          >
            <div>
              <div class="skeleton h-4 w-48" />
              <div class="skeleton mt-2 h-3 w-64" />
            </div>
            <div class="skeleton ml-auto h-4 w-16" />
            <div class="skeleton mx-auto h-5 w-20 rounded-full" />
            <div class="skeleton ml-auto h-4 w-24" />
            <div class="skeleton ml-auto h-4 w-16" />
          </div>
        </div>
      </div>
      <div v-else-if="invoices.length" class="divide-y divide-hairline">
        <div
          v-for="invoice in invoices"
          :key="invoiceId(invoice)"
          class="ui-table-row flex cursor-pointer flex-col gap-3 px-4 py-4 md:grid md:grid-cols-[minmax(0,1fr)_120px_240px_180px_112px] md:items-center md:gap-5 md:py-3.5"
          :class="invoiceId(selected) === invoiceId(invoice) ? 'bg-primary-50/60' : ''"
          role="button"
          tabindex="0"
          @click="openInvoice(invoice)"
          @keydown.enter.prevent="openInvoice(invoice)"
          @keydown.space.prevent="openInvoice(invoice)"
        >
          <div class="min-w-0">
            <div class="tnum break-all text-sm font-medium text-ink md:truncate">{{ invoiceId(invoice) || "-" }}</div>
            <div class="tnum mt-1 break-all text-xs text-slate-400">{{ copy.orderPrefix }}{{ orderId(invoice) }}</div>
          </div>
          <div class="flex items-center justify-between gap-4 border-t border-hairline pt-3 md:block md:border-0 md:pt-0 md:text-right">
            <span class="shrink-0 text-xs font-medium text-slate-400 md:hidden">{{ copy.columns.amount }}</span>
            <span class="tnum text-sm font-semibold text-ink">{{ amountText(invoice) }}</span>
          </div>
          <div class="flex min-w-0 items-center justify-between gap-4 border-t border-hairline pt-3 md:justify-center md:border-0 md:pt-0">
            <span class="shrink-0 text-xs font-medium text-slate-400 md:hidden">{{ copy.columns.status }}</span>
            <span class="ui-badge max-w-full truncate" :class="badgeClass(normalizedInvoiceStatus(invoice.status))">{{ invoiceStatusLabel(invoice.status) }}</span>
          </div>
          <div class="flex items-center justify-between gap-4 border-t border-hairline pt-3 md:block md:border-0 md:pt-0 md:text-right">
            <span class="shrink-0 text-xs font-medium text-slate-400 md:hidden">{{ copy.columns.createdAt }}</span>
            <span class="tnum text-sm text-slate-500">{{ formatDate(String(invoice.created_at || "")) }}</span>
          </div>
          <div class="border-t border-hairline pt-3 text-right md:border-0 md:pt-0">
            <button
              class="inline-flex w-full items-center justify-center rounded-[10px] border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-primary-700 transition-colors hover:bg-primary-50 md:w-auto md:border-0 md:bg-transparent md:p-0 md:hover:bg-transparent md:hover:text-primary-600"
              type="button"
              @click.stop="openInvoice(invoice)"
            >
              {{ copy.viewDetails }}
            </button>
          </div>
        </div>
      </div>
      <div v-else class="ui-empty">
        <div class="ui-empty-icon">
          <Receipt class="h-6 w-6" />
        </div>
        <p class="ui-empty-title">{{ copy.empty }}</p>
      </div>
      <div class="flex flex-col items-stretch justify-between gap-3 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center md:px-6">
        <span class="tnum text-center text-sm text-slate-500 sm:text-left">{{ copy.pageText(page) }}</span>
        <div class="flex flex-col gap-2 sm:flex-row">
          <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canPrev" @click="load(page - 1)">{{ copy.prev }}</button>
          <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canNext" @click="load(page + 1)">{{ copy.next }}</button>
        </div>
      </div>
    </section>

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="detailOpen && selected" class="ui-modal-mask">
          <section class="ui-modal-panel max-w-[1120px]">
            <div class="flex items-start justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
              <div class="min-w-0">
                <h2 class="text-base font-semibold text-ink md:text-lg">{{ copy.detailTitle }}</h2>
                <p class="tnum mt-1 break-all text-sm text-slate-500">{{ invoiceId(selected) || "-" }}</p>
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
            <div class="min-h-0 flex-1 space-y-5 overflow-y-auto p-4 md:p-6">
              <div class="grid gap-4 md:grid-cols-3">
                <div class="rounded-2xl bg-slate-50 p-4">
                  <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.labels.order }}</div>
                  <div class="tnum mt-2 break-all text-sm font-medium text-ink">{{ orderId(selected) }}</div>
                </div>
                <div class="rounded-2xl bg-slate-50 p-4">
                  <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.labels.amount }}</div>
                  <div class="tnum mt-2 text-sm font-semibold text-ink">{{ amountText(selected) }}</div>
                </div>
                <div class="rounded-2xl bg-slate-50 p-4">
                  <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.labels.status }}</div>
                  <div class="mt-2">
                    <span class="ui-badge" :class="badgeClass(normalizedInvoiceStatus(selected.status))">{{ invoiceStatusLabel(selected.status) }}</span>
                  </div>
                </div>
              </div>
              <div class="grid gap-4 md:grid-cols-2">
                <div v-for="field in selectedFields" :key="field.key" class="grid gap-1.5" :class="isStructuredValue(field.value) ? 'md:col-span-2' : ''">
                  <span class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ field.label }}</span>
                  <pre
                    v-if="isStructuredValue(field.value)"
                    class="tnum max-h-64 overflow-auto whitespace-pre-wrap break-words rounded-[10px] bg-slate-50 px-4 py-3 font-mono text-xs leading-5 text-slate-700"
                  >{{ jsonText(field.value) }}</pre>
                  <div v-else class="tnum break-words rounded-[10px] bg-slate-50 px-4 py-3 text-sm leading-5 text-slate-700">
                    {{ detailFieldText(field.displayValue) }}
                  </div>
                </div>
              </div>
            </div>
          </section>
        </div>
      </Transition>
    </Teleport>
  </main>
</template>
