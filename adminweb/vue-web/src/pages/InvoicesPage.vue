<script setup lang="ts">
import { Loader2, RefreshCw } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { badgeClass, pickFirst } from "@/lib/status"

const invoices = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const loading = ref(false)
const page = ref(1)
const total = ref(0)
const pageSize = 20

const canPrev = computed(() => page.value > 1)
const canNext = computed(() => page.value * pageSize < total.value || invoices.value.length >= pageSize)

function invoiceId(invoice: JsonRecord) {
  return String(pickFirst(invoice, ["id", "invoice_id", "invoice_ulid"]) || "")
}

async function load(targetPage = page.value) {
  loading.value = true
  try {
    const data = await apiClient<JsonRecord>(`/api/mall/invoices?page=${targetPage}&page_size=${pageSize}`)
    const list = Array.isArray(data.invoices) ? data.invoices : []
    invoices.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    total.value = Number(data.total || invoices.value.length) || 0
    selected.value = invoices.value[0] || null
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

    <div class="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
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
            <tr v-for="invoice in invoices" v-else :key="invoiceId(invoice)" class="border-t border-slate-100 hover:bg-sky-50" @click="selected = invoice">
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
      <section class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <div v-if="!selected" class="p-10 text-center text-slate-500">请选择一张发票</div>
        <pre v-else class="max-h-[720px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selected, null, 2) }}</pre>
      </section>
    </div>
  </section>
</template>
