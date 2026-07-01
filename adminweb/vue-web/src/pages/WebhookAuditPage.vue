<script setup lang="ts">
import { ChevronLeft, ChevronRight, Loader2, RefreshCw, RotateCcw, Search, Webhook, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, humanizeKey, isPrimitive, type JsonRecord } from "@/lib/display"
import { badgeClass, pickFirst } from "@/lib/status"

const PAGE_SIZE = 10

const messages = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const detail = ref<JsonRecord | null>(null)
const loading = ref(false)
const detailLoading = ref(false)
const detailOpen = ref(false)
const reprocessing = ref("")
const page = ref(1)
const total = ref(0)
const status = ref("")

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / PAGE_SIZE)))
const activeDetail = computed(() => detail.value || selected.value)
const detailFields = computed(() => {
  if (!activeDetail.value) return []
  return Object.entries(activeDetail.value)
    .filter(([, value]) => isPrimitive(value))
    .map(([key, value]) => ({
      key,
      label: fieldLabel(key),
      value: formatFieldValue(key, value),
    }))
})

function fieldLabel(key: string) {
  const labels: Record<string, string> = {
    webhook_msg_id: "Webhook 消息 ID",
    id: "Webhook 消息 ID",
    msg_fp: "消息指纹",
    message_fp: "消息指纹",
    provider: "来源",
    event_type: "事件类型",
    event_timestamp: "事件时间",
    exam_ulid: "考试 ID",
    confirmation_number: "确认编号",
    processed_status: "处理状态",
    status: "状态",
    error_message: "错误信息",
    created_at: "创建时间",
    updated_at: "更新时间",
    processed_at: "处理时间",
  }
  return labels[key] || humanizeKey(key)
}

function msgKey(message: JsonRecord) {
  return String(pickFirst(message, ["msg_fp", "message_fp", "id", "webhook_msg_id"]) || "")
}

function webhookMsgId(message: JsonRecord) {
  return pickFirst(message, ["webhook_msg_id", "id"])
}

function messageStatus(message: JsonRecord) {
  return pickFirst(message, ["processed_status", "status"]) || "-"
}

function messageTitle(message: JsonRecord) {
  return String(pickFirst(message, ["event_type", "provider", "type"]) || "Webhook")
}

function formatFieldValue(key: string, value: unknown) {
  if (key.endsWith("_at")) return formatDate(value)
  if (value === null || value === undefined || value === "") return "-"
  return String(value)
}

async function load(targetPage = page.value) {
  loading.value = true
  try {
    const params = new URLSearchParams({
      page: String(targetPage),
      page_size: String(PAGE_SIZE),
    })
    if (status.value) params.set("status", status.value)

    const data = await apiClient<JsonRecord>(`/api/audit/webhooks?${params}`)
    const rawList = data.webhook_messages || data.messages || data.items
    const list = Array.isArray(rawList) ? rawList : []
    const selectedKey = selected.value ? msgKey(selected.value) : ""

    messages.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    total.value = Number(data.total || messages.value.length) || 0
    page.value = Number(data.page || targetPage)
    const nextSelected = messages.value.find((item) => msgKey(item) === selectedKey) || messages.value[0] || null
    if (nextSelected) {
      await loadDetail(nextSelected, false, detailOpen.value)
    } else {
      selected.value = null
      detail.value = null
      detailOpen.value = false
    }
  } catch (err) {
    console.error(err)
    messages.value = []
    selected.value = null
    detail.value = null
    detailOpen.value = false
    toast.error("Webhook 审计加载失败")
  } finally {
    loading.value = false
  }
}

async function loadDetail(message: JsonRecord, showLoading = true, open = true) {
  selected.value = message
  detail.value = null
  detailOpen.value = open
  const fp = msgKey(message)
  if (!fp) return
  if (showLoading) detailLoading.value = true
  try {
    detail.value = await apiClient<JsonRecord>(`/api/audit/webhooks/detail?msg_fp=${encodeURIComponent(fp)}`)
  } catch (err) {
    console.error(err)
    toast.error("Webhook 详情加载失败")
  } finally {
    detailLoading.value = false
  }
}

function closeDetail() {
  detailOpen.value = false
}

async function reprocess(message: JsonRecord) {
  const id = webhookMsgId(message)
  if (!id) {
    toast.error("缺少 webhook_msg_id，不能重放")
    return
  }
  reprocessing.value = String(id)
  try {
    await apiClient("/api/audit/webhooks/reprocess", {
      method: "POST",
      body: JSON.stringify({ webhook_msg_id: Number(id) }),
    })
    toast.success("已提交重放请求")
    await load(page.value)
  } catch (err) {
    console.error(err)
    toast.error("重放失败")
  } finally {
    reprocessing.value = ""
  }
}

function search() {
  load(1)
}

function goPage(nextPage: number) {
  if (nextPage < 1 || nextPage > totalPages.value || nextPage === page.value) return
  load(nextPage)
}

onMounted(() => load(1))
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header class="flex items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">Webhook 审计</h1>
        <p class="mt-2 text-slate-600">查看支付和外部系统回调记录；支持按处理状态筛选、查看详情和重放。</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load(page)">
        <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
        刷新
      </button>
    </header>

    <form class="grid gap-4 rounded-3xl border border-slate-200 bg-white p-5 shadow-sm md:grid-cols-[1fr_auto]" @submit.prevent="search">
      <select v-model="status" class="rounded-xl border border-slate-200 px-4 py-3">
        <option value="">全部处理状态</option>
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

    <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
      <div class="flex items-center justify-between border-b border-slate-200 p-5">
        <div>
          <h2 class="text-xl font-black">Webhook 列表</h2>
          <p class="mt-1 text-sm text-slate-500">每页 10 条；点击查看详情后在弹框中查看完整字段和重放操作。</p>
        </div>
        <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">共 {{ total }} 条</span>
      </div>
      <div class="grid grid-cols-[minmax(0,1fr)_180px_180px_112px] gap-5 border-b border-slate-200 bg-slate-50 px-5 py-3 text-xs font-black text-slate-500">
        <span>Webhook</span>
        <span class="text-center">处理状态</span>
        <span class="text-right">创建时间</span>
        <span class="text-right">操作</span>
      </div>

      <div v-if="loading" class="p-12 text-center text-slate-500">
        <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
        正在加载...
      </div>
      <div v-else-if="!messages.length" class="p-12 text-center text-slate-500">暂无 Webhook 记录</div>
      <div v-else class="divide-y divide-slate-100">
        <div
          v-for="message in messages"
          :key="msgKey(message)"
          class="grid cursor-pointer grid-cols-[minmax(0,1fr)_180px_180px_112px] items-center gap-5 px-5 py-4 transition hover:bg-sky-50"
          :class="msgKey(selected || {}) === msgKey(message) ? 'bg-sky-50' : ''"
          role="button"
          tabindex="0"
          @click="loadDetail(message)"
          @keydown.enter.prevent="loadDetail(message)"
          @keydown.space.prevent="loadDetail(message)"
        >
          <div class="min-w-0">
            <div class="truncate text-base font-black">{{ messageTitle(message) }}</div>
            <div class="mt-1 truncate text-sm font-bold text-blue-700">{{ msgKey(message) || "-" }}</div>
          </div>
          <div class="min-w-0 text-center">
            <span class="inline-flex max-w-full truncate rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(messageStatus(message))">
              {{ messageStatus(message) }}
            </span>
          </div>
          <div class="text-right text-sm font-semibold text-slate-500">{{ formatDate(message.created_at) || "无创建时间" }}</div>
          <div class="text-right">
            <button
              class="rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-black text-[#0b4ea2] shadow-sm transition hover:border-sky-200 hover:bg-sky-50"
              type="button"
              @click.stop="loadDetail(message)"
            >
              查看详情
            </button>
          </div>
        </div>
      </div>

      <div class="flex items-center justify-between gap-3 border-t border-slate-200 p-5">
        <span class="text-sm font-bold text-slate-500">第 {{ page }} / {{ totalPages }} 页</span>
        <div class="flex gap-3">
          <button
            class="inline-flex items-center gap-2 rounded-xl border px-3 py-2 text-sm font-bold disabled:cursor-not-allowed disabled:opacity-40"
            type="button"
            :disabled="page <= 1 || loading"
            @click="goPage(page - 1)"
          >
            <ChevronLeft class="h-4 w-4" />
            上一页
          </button>
          <button
            class="inline-flex items-center gap-2 rounded-xl border px-3 py-2 text-sm font-bold disabled:cursor-not-allowed disabled:opacity-40"
            type="button"
            :disabled="page >= totalPages || loading"
            @click="goPage(page + 1)"
          >
            下一页
            <ChevronRight class="h-4 w-4" />
          </button>
        </div>
      </div>
    </section>

    <Teleport to="body">
      <div v-if="detailOpen && selected" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-6">
        <section class="flex max-h-[88vh] w-full max-w-[1120px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-6 py-5">
            <div class="flex min-w-0 items-start gap-3">
              <span class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-blue-50 text-blue-700">
                <Webhook class="h-5 w-5" />
              </span>
              <div class="min-w-0">
                <h2 class="text-2xl font-black text-slate-950">Webhook 详情</h2>
              </div>
            </div>
            <div class="flex shrink-0 items-center gap-3">
              <button
                class="inline-flex h-10 items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 text-sm font-bold text-slate-700 shadow-sm transition hover:bg-slate-50 disabled:opacity-50"
                type="button"
                :disabled="reprocessing === String(webhookMsgId(selected))"
                @click="reprocess(selected)"
              >
                <RotateCcw class="h-4 w-4" :class="reprocessing === String(webhookMsgId(selected)) ? 'animate-spin' : ''" />
                重放
              </button>
              <button
                class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
                type="button"
                aria-label="关闭"
                @click="closeDetail"
              >
                <X class="h-5 w-5" />
              </button>
            </div>
          </div>

          <div v-if="detailLoading" class="p-12 text-center text-slate-500">
            <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
            正在加载详情...
          </div>
          <div v-else class="min-h-0 flex-1 space-y-5 overflow-y-auto p-5">
            <div class="rounded-2xl border border-blue-100 bg-blue-50 p-4">
              <div class="flex items-center gap-2 text-sm font-black text-blue-700">
                <Webhook class="h-4 w-4" />
                当前 Webhook
              </div>
              <div class="mt-1 break-all text-lg font-black text-slate-950">{{ msgKey(selected) || "-" }}</div>
              <div class="mt-2 inline-flex rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(messageStatus(activeDetail || selected))">
                {{ messageStatus(activeDetail || selected) }}
              </div>
            </div>

            <div class="grid gap-4 md:grid-cols-2">
              <label v-for="field in detailFields" :key="field.key" class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                <span class="text-xs font-black uppercase tracking-wide text-slate-400">{{ field.label }}</span>
                <input class="mt-2 w-full rounded-xl border border-slate-200 bg-slate-100 px-3 py-2 text-sm font-bold text-slate-700" :value="field.value" disabled />
              </label>
            </div>

            <div class="rounded-2xl border border-slate-200">
              <div class="border-b border-slate-100 px-4 py-3 text-sm font-black">完整原始字段</div>
              <pre class="max-h-[460px] overflow-auto rounded-b-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(activeDetail, null, 2) }}</pre>
            </div>
          </div>
        </section>
      </div>
    </Teleport>
  </section>
</template>
