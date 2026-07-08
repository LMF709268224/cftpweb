<script setup lang="ts">
import { Check, Copy as CopyIcon, Loader2, RefreshCw, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { copyTextToClipboard } from "@/lib/clipboard"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { badgeClass, labelFor, normalizeStatus, orderStatusOptions, pickFirst } from "@/lib/status"

const invoices = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const loading = ref(false)
const detailOpen = ref(false)
const copiedJson = ref(false)
const page = ref(1)
const total = ref(0)
const pageSize = 20
const { t } = useAdminLanguage()
const copy = computed(() => t.value.invoices)

const canPrev = computed(() => page.value > 1)
const canNext = computed(() => page.value * pageSize < total.value || invoices.value.length >= pageSize)
const selectedJson = computed(() => JSON.stringify(selected.value || {}, null, 2))
const selectedFields = computed(() =>
  Object.entries(selected.value || {}).map(([key, value]) => ({
    key,
    label: copy.value.fieldLabels[key as keyof typeof copy.value.fieldLabels] || key.replace(/_/g, " "),
    value,
    displayValue: key === "status" ? invoiceStatusLabel(value) : key.endsWith("_at") ? formatDate(value) : String(value ?? "-"),
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

async function copySelectedJson() {
  try {
    await copyTextToClipboard(selectedJson.value)
    copiedJson.value = true
    toast.success(copy.value.toasts.jsonCopied)
    window.setTimeout(() => {
      copiedJson.value = false
    }, 1600)
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.jsonCopyFailed)
  }
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
    const data = await apiClient<JsonRecord>(`/api/mall/invoices?page=${targetPage}&page_size=${pageSize}`)
    const list = Array.isArray(data.invoices) ? data.invoices : []
    invoices.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    total.value = Number(data.total || invoices.value.length) || 0
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
    toast.error(copy.value.toasts.loadFailed)
  } finally {
    loading.value = false
  }
}

onMounted(() => load(1))
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">{{ copy.title }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load(page)">
        <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
        {{ copy.refresh }}
      </button>
    </header>

    <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
      <div class="flex items-center justify-between border-b border-slate-200 p-5">
        <div>
          <h2 class="text-xl font-black">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
        <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ copy.totalText(total) }}</span>
      </div>
      <div class="grid grid-cols-[minmax(0,1fr)_120px_240px_180px_112px] gap-5 border-b border-slate-200 bg-slate-50 px-5 py-3 text-xs font-black text-slate-500">
        <span>{{ copy.columns.invoice }}</span>
        <span class="text-right">{{ copy.columns.amount }}</span>
        <span class="text-center">{{ copy.columns.status }}</span>
        <span class="text-right">{{ copy.columns.createdAt }}</span>
        <span class="text-right">{{ copy.columns.action }}</span>
      </div>
      <div v-if="loading" class="p-12 text-center text-slate-500">
        <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
        {{ copy.loading }}
      </div>
      <div v-else-if="invoices.length" class="divide-y divide-slate-100">
        <div
          v-for="invoice in invoices"
          :key="invoiceId(invoice)"
          class="grid cursor-pointer grid-cols-[minmax(0,1fr)_120px_240px_180px_112px] items-center gap-5 px-5 py-4 transition hover:bg-sky-50"
          :class="invoiceId(selected) === invoiceId(invoice) ? 'bg-sky-50' : ''"
          role="button"
          tabindex="0"
          @click="openInvoice(invoice)"
          @keydown.enter.prevent="openInvoice(invoice)"
          @keydown.space.prevent="openInvoice(invoice)"
        >
          <div class="min-w-0">
            <div class="truncate font-black text-slate-950">{{ invoiceId(invoice) || "-" }}</div>
            <div class="mt-1 break-all text-sm text-slate-500">{{ copy.orderPrefix }}{{ orderId(invoice) }}</div>
          </div>
          <div class="text-right text-sm font-black">{{ amountText(invoice) }}</div>
          <div class="min-w-0 text-center">
            <span class="inline-flex max-w-full truncate rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(normalizedInvoiceStatus(invoice.status))">{{ invoiceStatusLabel(invoice.status) }}</span>
          </div>
          <div class="text-right text-sm font-semibold text-slate-500">{{ formatDate(String(invoice.created_at || "")) }}</div>
          <div class="text-right">
            <button
              class="text-sm font-bold text-[#1890ff] transition hover:underline"
              type="button"
              @click.stop="openInvoice(invoice)"
            >
              {{ copy.viewDetails }}
            </button>
          </div>
        </div>
      </div>
      <div v-else class="p-12 text-center text-slate-500">{{ copy.empty }}</div>
      <div class="flex items-center justify-between gap-3 border-t border-slate-200 p-5">
        <span class="text-sm font-bold text-slate-500">{{ copy.pageText(page) }}</span>
        <div class="flex gap-3">
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrev" @click="load(page - 1)">{{ copy.prev }}</button>
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext" @click="load(page + 1)">{{ copy.next }}</button>
        </div>
      </div>
    </section>

    <Teleport to="body">
      <div v-if="detailOpen && selected" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-6">
        <section class="flex max-h-[88vh] w-full max-w-[1120px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-6 py-5">
            <div class="min-w-0">
              <h2 class="text-2xl font-black text-slate-950">{{ copy.detailTitle }}</h2>
              <p class="mt-1 break-all text-sm text-slate-500">{{ invoiceId(selected) || "-" }}</p>
            </div>
            <button
              class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
              type="button"
              :aria-label="copy.close"
              @click="closeDetail"
            >
              <X class="h-5 w-5" />
            </button>
          </div>
          <div class="min-h-0 flex-1 space-y-5 overflow-y-auto p-5">
            <div class="grid gap-4 md:grid-cols-3">
              <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                <div class="text-xs font-black uppercase text-slate-400">{{ copy.labels.order }}</div>
                <div class="mt-2 break-all text-sm font-bold">{{ orderId(selected) }}</div>
              </div>
              <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                <div class="text-xs font-black uppercase text-slate-400">{{ copy.labels.amount }}</div>
                <div class="mt-2 text-sm font-bold">{{ amountText(selected) }}</div>
              </div>
              <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                <div class="text-xs font-black uppercase text-slate-400">{{ copy.labels.status }}</div>
                <div class="mt-2">
                  <span class="inline-flex rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(normalizedInvoiceStatus(selected.status))">{{ invoiceStatusLabel(selected.status) }}</span>
                </div>
              </div>
            </div>
            <div class="grid gap-4 md:grid-cols-2">
              <div v-for="field in selectedFields" :key="field.key" class="grid gap-2 text-sm font-bold" :class="isStructuredValue(field.value) ? 'md:col-span-2' : ''">
                <span class="text-xs font-black uppercase text-slate-400">{{ field.label }}</span>
                <pre
                  v-if="isStructuredValue(field.value)"
                  class="max-h-64 overflow-auto whitespace-pre-wrap break-words rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 font-mono text-xs leading-5 text-slate-700"
                >{{ jsonText(field.value) }}</pre>
                <div v-else class="min-h-11 break-words rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm font-bold leading-5 text-slate-700">
                  {{ detailFieldText(field.displayValue) }}
                </div>
              </div>
            </div>
            <details class="rounded-2xl border border-slate-200 bg-white p-4">
              <summary class="cursor-pointer text-sm font-black text-slate-700">{{ copy.rawJson }}</summary>
              <div class="mt-4 overflow-hidden rounded-2xl bg-slate-950">
                <div class="flex items-center justify-between gap-3 border-b border-white/10 px-4 py-3">
                  <span class="text-xs font-black uppercase text-slate-400">{{ copy.rawJson }}</span>
                  <button class="inline-flex h-8 items-center gap-2 rounded-lg border border-white/10 px-3 text-xs font-bold text-slate-100 transition hover:bg-white/10" type="button" @click="copySelectedJson">
                    <Check v-if="copiedJson" class="h-3.5 w-3.5" />
                    <CopyIcon v-else class="h-3.5 w-3.5" />
                    {{ copiedJson ? copy.copiedJson : copy.copyJson }}
                  </button>
                </div>
                <pre class="max-h-[520px] overflow-auto p-5 text-xs leading-6 text-slate-100">{{ selectedJson }}</pre>
              </div>
            </details>
          </div>
        </section>
      </div>
    </Teleport>
  </section>
</template>
