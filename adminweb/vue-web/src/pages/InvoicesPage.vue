<script setup lang="ts">
import { Loader2, RefreshCw, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { badgeClass, pickFirst } from "@/lib/status"

const invoices = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const loading = ref(false)
const detailOpen = ref(false)
const page = ref(1)
const total = ref(0)
const pageSize = 20

const canPrev = computed(() => page.value > 1)
const canNext = computed(() => page.value * pageSize < total.value || invoices.value.length >= pageSize)
const selectedFields = computed(() => selected.value || {})

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
    toast.error("发票加载失败")
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
        <h1 class="text-4xl font-black tracking-tight">发票管理</h1>
        <p class="mt-2 text-slate-600">查看支付发票和收款记录。</p>
        <p class="mt-2 text-xs font-semibold text-slate-500">已确认接口：list invoices。当前 adminbff 未提供发票详情、更新或下载接口，因此右侧只读展示列表返回字段。</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load(page)">
        <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
        刷新
      </button>
    </header>

    <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
      <div class="flex items-center justify-between border-b border-slate-200 p-5">
        <div>
          <h2 class="text-xl font-black">发票列表</h2>
          <p class="mt-1 text-sm text-slate-500">来自 `/api/mall/invoices`，详情仅展示列表返回字段。</p>
        </div>
        <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">共 {{ total }} 条</span>
      </div>
      <div class="grid grid-cols-[minmax(0,1fr)_120px_240px_180px_112px] gap-5 border-b border-slate-200 bg-slate-50 px-5 py-3 text-xs font-black text-slate-500">
        <span>发票</span>
        <span class="text-right">金额</span>
        <span class="text-center">状态</span>
        <span class="text-right">创建时间</span>
        <span class="text-right">操作</span>
      </div>
      <div v-if="loading" class="p-12 text-center text-slate-500">
        <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
        正在加载...
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
            <div class="mt-1 break-all text-sm text-slate-500">订单：{{ orderId(invoice) }}</div>
          </div>
          <div class="text-right text-sm font-black">{{ amountText(invoice) }}</div>
          <div class="min-w-0 text-center">
            <span class="inline-flex max-w-full truncate rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(invoice.status)">{{ invoice.status || "-" }}</span>
          </div>
          <div class="text-right text-sm font-semibold text-slate-500">{{ formatDate(String(invoice.created_at || "")) }}</div>
          <div class="text-right">
            <button
              class="rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-black text-[#0b4ea2] shadow-sm transition hover:border-sky-200 hover:bg-sky-50"
              type="button"
              @click.stop="openInvoice(invoice)"
            >
              查看详情
            </button>
          </div>
        </div>
      </div>
      <div v-else class="p-12 text-center text-slate-500">暂无发票</div>
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
        <section class="flex max-h-[88vh] w-full max-w-[1120px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-6 py-5">
            <div class="min-w-0">
              <h2 class="text-2xl font-black text-slate-950">发票详情</h2>
              <p class="mt-1 break-all text-sm text-slate-500">{{ invoiceId(selected) || "-" }}</p>
            </div>
            <button
              class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
              type="button"
              aria-label="关闭"
              @click="closeDetail"
            >
              <X class="h-5 w-5" />
            </button>
          </div>
          <div class="min-h-0 flex-1 space-y-5 overflow-y-auto p-5">
            <div class="grid gap-4 md:grid-cols-3">
              <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                <div class="text-xs font-black uppercase text-slate-400">订单</div>
                <div class="mt-2 break-all text-sm font-bold">{{ orderId(selected) }}</div>
              </div>
              <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                <div class="text-xs font-black uppercase text-slate-400">金额</div>
                <div class="mt-2 text-sm font-bold">{{ amountText(selected) }}</div>
              </div>
              <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                <div class="text-xs font-black uppercase text-slate-400">状态</div>
                <div class="mt-2">
                  <span class="inline-flex rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(selected.status)">{{ selected.status || "-" }}</span>
                </div>
              </div>
            </div>
            <div class="grid gap-4 md:grid-cols-2">
              <label v-for="(value, key) in selectedFields" :key="key" class="grid gap-2 text-sm font-bold">
                {{ key }}
                <textarea
                  v-if="Array.isArray(value) || (value && typeof value === 'object')"
                  class="min-h-24 rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-600"
                  disabled
                  :value="JSON.stringify(value, null, 2)"
                />
                <input v-else class="rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-600" disabled :value="String(value ?? '-')" />
              </label>
            </div>
            <pre class="max-h-[520px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selected, null, 2) }}</pre>
          </div>
        </section>
      </div>
    </Teleport>
  </section>
</template>
