<script setup lang="ts">
import { Loader2, RefreshCw, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { badgeClass, pickFirst } from "@/lib/status"

const invoices = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const detailOpen = ref(false)
const loading = ref(false)
const page = ref(1)
const total = ref(0)
const pageSize = 20

const canPrev = computed(() => page.value > 1)
const canNext = computed(() => page.value * pageSize < total.value || invoices.value.length >= pageSize)

function invoiceId(invoice: JsonRecord) {
  return String(pickFirst(invoice, ["id", "invoice_id", "invoice_ulid"]) || "")
}

function openInvoice(invoice: JsonRecord) {
  selected.value = invoice
  detailOpen.value = true
}

async function load(targetPage = page.value) {
  loading.value = true
  try {
    const data = await apiClient<JsonRecord>(`/api/mall/invoices?page=${targetPage}&page_size=${pageSize}`)
    const list = Array.isArray(data.invoices) ? data.invoices : []
    invoices.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    total.value = Number(data.total || invoices.value.length) || 0
    page.value = targetPage
  } catch (err) {
    console.error(err)
    toast.error("发票加载失败")
  } finally {
    loading.value = false
  }
}

onMounted(() => load(1))
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header class="flex items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">发票管理</h1>
        <p class="mt-2 text-slate-600">查看支付发票和收款记录。</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load(page)">
        <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
        刷新
      </button>
    </header>

    <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
      <table class="w-full text-left text-sm">
        <thead class="bg-slate-50 text-xs uppercase text-slate-500">
          <tr>
            <th class="px-5 py-3">发票</th>
            <th class="px-5 py-3">订单</th>
            <th class="px-5 py-3">金额</th>
            <th class="px-5 py-3">状态</th>
            <th class="px-5 py-3">创建时间</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td class="px-5 py-10 text-center text-slate-500" colspan="5"><Loader2 class="mx-auto mb-2 h-5 w-5 animate-spin" />正在加载...</td>
          </tr>
          <tr v-else-if="!invoices.length">
            <td class="px-5 py-10 text-center text-slate-500" colspan="5">暂无发票</td>
          </tr>
          <tr v-for="invoice in invoices" v-else :key="invoiceId(invoice)" class="cursor-pointer border-t border-slate-100 hover:bg-sky-50" @click="openInvoice(invoice)">
            <td class="px-5 py-4 font-black">{{ invoiceId(invoice) || "-" }}</td>
            <td class="px-5 py-4 text-slate-600">{{ invoice.order_id || invoice.order_ulid || "-" }}</td>
            <td class="px-5 py-4 font-bold">{{ Number(invoice.amount || 0).toFixed(2) }} {{ invoice.currency || "" }}</td>
            <td class="px-5 py-4">
              <span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(invoice.status)">{{ invoice.status || "-" }}</span>
            </td>
            <td class="px-5 py-4 text-slate-600">{{ formatDate(String(invoice.created_at || "")) }}</td>
          </tr>
        </tbody>
      </table>
      <div class="flex justify-end gap-3 border-t border-slate-200 p-5">
        <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrev" @click="load(page - 1)">上一页</button>
        <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext" @click="load(page + 1)">下一页</button>
      </div>
    </section>

    <div v-if="detailOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-6" @click.self="detailOpen = false">
      <section class="w-full max-w-4xl overflow-hidden rounded-3xl bg-white shadow-2xl">
        <header class="flex items-start justify-between gap-4 border-b border-slate-200 p-6">
          <div>
            <h2 class="text-2xl font-black">发票详情</h2>
            <p class="mt-1 text-sm text-slate-500">{{ selected ? invoiceId(selected) : "-" }}</p>
          </div>
          <button class="rounded-xl border border-slate-200 p-2 text-slate-500 hover:bg-slate-50" type="button" @click="detailOpen = false">
            <X class="h-5 w-5" />
          </button>
        </header>
        <div v-if="selected" class="space-y-5 p-6">
          <dl class="grid gap-3 md:grid-cols-2">
            <div class="rounded-2xl bg-slate-50 p-4">
              <dt class="text-xs font-black uppercase text-slate-400">Order</dt>
              <dd class="mt-2 break-all font-bold">{{ selected.order_id || selected.order_ulid || "-" }}</dd>
            </div>
            <div class="rounded-2xl bg-slate-50 p-4">
              <dt class="text-xs font-black uppercase text-slate-400">Amount</dt>
              <dd class="mt-2 font-bold">{{ Number(selected.amount || 0).toFixed(2) }} {{ selected.currency || "" }}</dd>
            </div>
            <div class="rounded-2xl bg-slate-50 p-4">
              <dt class="text-xs font-black uppercase text-slate-400">Status</dt>
              <dd class="mt-2">
                <span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(selected.status)">{{ selected.status || "-" }}</span>
              </dd>
            </div>
            <div class="rounded-2xl bg-slate-50 p-4">
              <dt class="text-xs font-black uppercase text-slate-400">Created At</dt>
              <dd class="mt-2 font-bold">{{ formatDate(String(selected.created_at || "")) }}</dd>
            </div>
          </dl>
          <pre class="max-h-[58vh] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selected, null, 2) }}</pre>
        </div>
      </section>
    </div>
  </section>
</template>
