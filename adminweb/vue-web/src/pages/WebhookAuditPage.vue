<script setup lang="ts">
import { ChevronLeft, ChevronRight, Loader2, RefreshCw, RotateCcw, Search, Webhook } from "lucide-vue-next"
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
      await loadDetail(nextSelected, false)
    } else {
      selected.value = null
      detail.value = null
    }
  } catch (err) {
    console.error(err)
    toast.error("Webhook 审计加载失败")
  } finally {
    loading.value = false
  }
}

async function loadDetail(message: JsonRecord, showLoading = true) {
  selected.value = message
  detail.value = null
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

    <div class="grid gap-6 xl:grid-cols-[0.9fr_1.1fr]">
      <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="flex items-center justify-between border-b border-slate-100 px-5 py-4">
          <div>
            <h2 class="text-xl font-black">Webhook 列表</h2>
            <p class="mt-1 text-sm text-slate-500">每页 10 条；左侧选择记录，右侧展示详情和重放操作。</p>
          </div>
          <span class="rounded-full bg-slate-100 px-3 py-1 text-xs font-black text-slate-600">共 {{ total }} 条</span>
        </div>

        <div v-if="loading" class="p-12 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          正在加载...
        </div>
        <div v-else-if="!messages.length" class="p-12 text-center text-slate-500">暂无 Webhook 记录</div>
        <button
          v-for="message in messages"
          v-else
          :key="msgKey(message)"
          class="grid w-full grid-cols-[1fr_auto] gap-4 border-b border-slate-100 px-5 py-4 text-left last:border-b-0 hover:bg-sky-50"
          :class="selected === message ? 'bg-sky-50' : ''"
          type="button"
          @click="loadDetail(message)"
        >
          <div class="min-w-0">
            <div class="truncate text-base font-black">{{ messageTitle(message) }}</div>
            <div class="mt-1 truncate text-sm font-bold text-blue-700">{{ msgKey(message) || "-" }}</div>
            <div class="mt-1 text-xs text-slate-500">{{ formatDate(message.created_at) || "无创建时间" }}</div>
          </div>
          <span class="h-fit rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(messageStatus(message))">{{ messageStatus(message) }}</span>
        </button>

        <div class="flex items-center justify-end gap-3 border-t border-slate-200 p-5">
          <button
            class="inline-flex items-center gap-2 rounded-xl border px-3 py-2 text-sm font-bold disabled:cursor-not-allowed disabled:opacity-40"
            type="button"
            :disabled="page <= 1 || loading"
            @click="goPage(page - 1)"
          >
            <ChevronLeft class="h-4 w-4" />
            上一页
          </button>
          <span class="text-sm font-bold text-slate-500">第 {{ page }} / {{ totalPages }} 页</span>
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
      </section>

      <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="flex items-center justify-between border-b border-slate-100 px-5 py-4">
          <div>
            <h2 class="text-xl font-black">Webhook 详情</h2>
            <p class="mt-1 text-sm text-slate-500">详情接口按消息指纹获取；重放接口需要 webhook_msg_id。</p>
          </div>
          <button
            v-if="selected"
            class="inline-flex items-center gap-2 rounded-xl border px-4 py-2 text-sm font-bold disabled:opacity-50"
            type="button"
            :disabled="reprocessing === String(webhookMsgId(selected))"
            @click="reprocess(selected)"
          >
            <RotateCcw class="h-4 w-4" />
            重放
          </button>
        </div>

        <div v-if="!selected" class="p-12 text-center text-slate-500">请选择一条记录</div>
        <div v-else-if="detailLoading" class="p-12 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          正在加载详情...
        </div>
        <div v-else class="space-y-5 p-5">
          <div class="rounded-2xl bg-blue-50 p-4">
            <div class="flex items-center gap-2 text-sm font-black text-blue-700">
              <Webhook class="h-4 w-4" />
              当前 Webhook
            </div>
            <div class="mt-1 break-all text-lg font-black text-slate-950">{{ msgKey(selected) || "-" }}</div>
            <div class="mt-2 inline-flex rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(messageStatus(activeDetail || selected))">
              {{ messageStatus(activeDetail || selected) }}
            </div>
          </div>

          <div class="grid gap-3 md:grid-cols-2">
            <label v-for="field in detailFields" :key="field.key" class="rounded-2xl bg-slate-50 p-4">
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
  </section>
</template>
