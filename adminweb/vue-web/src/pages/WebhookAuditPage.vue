<script setup lang="ts">
import { Loader2, RefreshCw, RotateCcw, Search } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { badgeClass, pickFirst } from "@/lib/status"

const messages = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const detail = ref<JsonRecord | null>(null)
const loading = ref(false)
const detailLoading = ref(false)
const reprocessing = ref("")
const page = ref(1)
const total = ref(0)
const provider = ref("")
const status = ref("")
const pageSize = 20

const canPrev = computed(() => page.value > 1)
const canNext = computed(() => page.value * pageSize < total.value || messages.value.length >= pageSize)

function msgKey(message: JsonRecord) {
  return String(pickFirst(message, ["msg_fp", "message_fp", "id", "webhook_msg_id"]) || "")
}

async function load(targetPage = page.value) {
  loading.value = true
  try {
    const params = new URLSearchParams({
      page: String(targetPage),
      page_size: String(pageSize),
    })
    if (provider.value.trim()) params.set("provider", provider.value.trim())
    if (status.value) params.set("status", status.value)
    const data = await apiClient<JsonRecord>(`/api/audit/webhooks?${params}`)
    const list = Array.isArray(data.messages) ? data.messages : []
    messages.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    total.value = Number(data.total || messages.value.length) || 0
    selected.value = messages.value[0] || null
    detail.value = null
    page.value = targetPage
  } catch (err) {
    console.error(err)
    toast.error("Webhook 审计加载失败")
  } finally {
    loading.value = false
  }
}

async function loadDetail(message: JsonRecord) {
  selected.value = message
  detail.value = null
  const fp = msgKey(message)
  if (!fp) return
  detailLoading.value = true
  try {
    detail.value = await apiClient<JsonRecord>(`/api/audit/webhooks/detail?msg_fp=${encodeURIComponent(fp)}`)
  } catch (err) {
    console.error(err)
    toast.error("Webhook 详情加载失败")
  } finally {
    detailLoading.value = false
  }
}

async function reprocess(message: JsonRecord) {
  const id = pickFirst(message, ["webhook_msg_id", "id"])
  if (!id) {
    toast.error("缺少 webhook_msg_id")
    return
  }
  reprocessing.value = String(id)
  try {
    await apiClient("/api/audit/webhooks/reprocess", {
      method: "POST",
      body: JSON.stringify({ webhook_msg_id: id }),
    })
    toast.success("已提交重放请求")
    await load(page.value)
  } finally {
    reprocessing.value = ""
  }
}

function search() {
  load(1)
}

onMounted(() => load(1))
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header class="flex items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">Webhook 审计</h1>
        <p class="mt-2 text-slate-600">查看支付和外部系统回调记录，并支持重放。</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load(page)">
        <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
        刷新
      </button>
    </header>

    <form class="grid gap-4 rounded-3xl border border-slate-200 bg-white p-5 shadow-sm md:grid-cols-[1fr_180px_auto]" @submit.prevent="search">
      <input v-model="provider" class="rounded-xl border border-slate-200 px-4 py-3" placeholder="Provider，例如 stripe / pearson" />
      <select v-model="status" class="rounded-xl border border-slate-200 px-4 py-3">
        <option value="">全部状态</option>
        <option value="SUCCESS">SUCCESS</option>
        <option value="FAILED">FAILED</option>
        <option value="IGNORED">IGNORED</option>
        <option value="PENDING">PENDING</option>
      </select>
      <button class="inline-flex items-center justify-center gap-2 rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white" type="submit">
        <Search class="h-4 w-4" />
        查询
      </button>
    </form>

    <div class="grid gap-6 xl:grid-cols-[1.05fr_0.95fr]">
      <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div v-if="loading" class="p-12 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          正在加载...
        </div>
        <button
          v-for="message in messages"
          v-else
          :key="msgKey(message)"
          class="grid w-full grid-cols-[1fr_auto] gap-4 border-b border-slate-100 px-5 py-4 text-left last:border-b-0 hover:bg-sky-50"
          :class="selected === message ? 'bg-sky-50' : ''"
          type="button"
          @click="loadDetail(message)"
        >
          <div>
            <div class="font-black">{{ message.event_type || message.provider || "Webhook" }}</div>
            <div class="mt-1 text-sm text-slate-500">{{ msgKey(message) || "-" }}</div>
            <div class="mt-1 text-xs text-slate-400">{{ formatDate(String(message.created_at || "")) }}</div>
          </div>
          <span class="h-fit rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(message.status)">{{ message.status || "-" }}</span>
        </button>
        <div v-if="!loading && !messages.length" class="p-12 text-center text-slate-500">暂无 Webhook 记录</div>
        <div class="flex justify-end gap-3 border-t border-slate-200 p-5">
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrev" @click="load(page - 1)">上一页</button>
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext" @click="load(page + 1)">下一页</button>
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <div v-if="!selected" class="p-10 text-center text-slate-500">请选择一条记录</div>
        <template v-else>
          <div class="mb-4 flex items-center justify-between gap-4">
            <h2 class="text-xl font-black">Webhook 详情</h2>
            <button
              class="inline-flex items-center gap-2 rounded-xl border px-4 py-2 text-sm font-bold disabled:opacity-50"
              type="button"
              :disabled="reprocessing === String(selected.webhook_msg_id || selected.id)"
              @click="reprocess(selected)"
            >
              <RotateCcw class="h-4 w-4" />
              重放
            </button>
          </div>
          <div v-if="detailLoading" class="p-10 text-center text-slate-500">
            <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
            正在加载详情...
          </div>
          <pre v-else class="max-h-[720px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(detail || selected, null, 2) }}</pre>
        </template>
      </section>
    </div>
  </section>
</template>
