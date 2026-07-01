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
const activeRecord = computed(() => activeDetail.value || selected.value || {})
const summaryFields = computed(() => {
  const record = activeRecord.value
  if (!record || !Object.keys(record).length) return []
  return [
    { label: "这是什么", value: eventTitle(record) },
    { label: "处理状态", value: statusText(messageStatus(record)) },
    { label: "确认编号", value: fieldText(record, ["confirmation_number"]) },
    { label: "考试 ID", value: fieldText(record, ["exam_ulid", "exam_id"]) },
    { label: "事件时间", value: formatDate(fieldText(record, ["event_timestamp"])) || "-" },
    { label: "处理时间", value: formatDate(fieldText(record, ["processed_at"])) || "-" },
    { label: "Webhook 消息 ID", value: fieldText(record, ["webhook_msg_id", "id"]) },
  ]
})
const detailFields = computed(() => {
  if (!activeDetail.value) return []
  return Object.entries(activeDetail.value)
    .filter(([, value]) => isPrimitive(value))
    .filter(([key]) => !["payload_json", "msg_fp", "message_fp"].includes(key))
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
  return eventTitle(message)
}

function fieldText(record: JsonRecord | null | undefined, keys: string[]) {
  const value = pickFirst(record || {}, keys)
  if (value === null || value === undefined || value === "") return "-"
  return String(value)
}

function eventType(message: JsonRecord | null | undefined) {
  return String(pickFirst(message || {}, ["event_type", "type", "provider"]) || "")
}

function eventTitle(message: JsonRecord | null | undefined) {
  const type = eventType(message).toLowerCase()
  const labels: Record<string, string> = {
    result_created: "考试结果回调",
    appointment_scheduled: "考试预约回调",
    appointment_rescheduled: "考试改期回调",
    appointment_cancelled: "考试取消回调",
    payment_intent_succeeded: "支付成功回调",
    payment_intent_payment_failed: "支付失败回调",
    invoice_created: "发票创建回调",
  }
  return labels[type] || eventType(message) || "Webhook 回调"
}

function eventDescription(message: JsonRecord | null | undefined) {
  const type = eventType(message).toLowerCase()
  if (type === "result_created") return "外部考试系统通知我们：某个考生的考试结果已经生成。系统会据此同步成绩、推进管线状态。"
  if (type === "appointment_scheduled") return "外部考试系统通知我们：某个考试预约已经创建。系统会记录预约信息，方便后续同步考试结果。"
  if (type === "appointment_rescheduled") return "外部考试系统通知我们：某个考试预约时间发生变更。"
  if (type === "appointment_cancelled") return "外部考试系统通知我们：某个考试预约已取消。"
  if (type.includes("payment")) return "支付服务通知我们一笔支付状态变化。"
  if (type.includes("invoice")) return "外部系统通知我们一笔发票相关事件。"
  return "这是外部系统发给平台的一条回调消息，系统会按事件类型自动处理。"
}

function messageSubtitle(message: JsonRecord) {
  const confirmation = fieldText(message, ["confirmation_number"])
  const examId = fieldText(message, ["exam_ulid", "exam_id"])
  if (confirmation !== "-") return `确认编号：${confirmation}`
  if (examId !== "-") return `考试 ID：${examId}`
  return `消息 ID：${fieldText(message, ["webhook_msg_id", "id"])}`
}

function statusText(value: unknown) {
  const statusValue = String(value || "").toUpperCase()
  const labels: Record<string, string> = {
    PROCESSED: "已处理",
    SKIPPED: "已跳过",
    FAILED: "处理失败",
    PENDING: "待处理",
    SUCCESS: "成功",
    IGNORED: "已忽略",
  }
  return labels[statusValue] || String(value || "-")
}

function formatFieldValue(key: string, value: unknown) {
  if (key.endsWith("_at")) return formatDate(value)
  if (key === "event_type") return eventTitle({ event_type: value })
  if (key === "processed_status" || key === "status") return statusText(value)
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
        <p class="mt-2 text-slate-600">查看外部系统发来的回调记录。这里主要用于排查考试预约、考试结果、支付等异步通知是否处理成功。</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load(page)">
        <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
        刷新
      </button>
    </header>

    <form class="grid gap-4 rounded-3xl border border-slate-200 bg-white p-5 shadow-sm md:grid-cols-[1fr_auto]" @submit.prevent="search">
      <select v-model="status" class="rounded-xl border border-slate-200 px-4 py-3">
        <option value="">全部处理状态</option>
        <option value="PROCESSED">已处理 / PROCESSED</option>
        <option value="SKIPPED">已跳过 / SKIPPED</option>
        <option value="FAILED">FAILED</option>
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
            <div class="mt-1 truncate text-sm font-semibold text-slate-600">{{ messageSubtitle(message) }}</div>
            <div class="mt-1 text-xs text-slate-500">{{ formatDate(message.created_at) || "无创建时间" }}</div>
          </div>
          <span class="h-fit rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(messageStatus(message))">{{ statusText(messageStatus(message)) }}</span>
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
            <p class="mt-1 text-sm text-slate-500">先看业务含义；消息指纹、Payload JSON 等技术字段收在下方。</p>
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
              当前回调
            </div>
            <div class="mt-1 break-all text-lg font-black text-slate-950">{{ eventTitle(activeRecord) }}</div>
            <p class="mt-2 text-sm font-semibold leading-6 text-slate-600">{{ eventDescription(activeRecord) }}</p>
            <div class="mt-2 inline-flex rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(messageStatus(activeDetail || selected))">
              {{ statusText(messageStatus(activeDetail || selected)) }}
            </div>
          </div>

          <div class="grid gap-3 md:grid-cols-2">
            <label v-for="field in summaryFields" :key="field.label" class="rounded-2xl bg-slate-50 p-4">
              <span class="text-xs font-black tracking-wide text-slate-400">{{ field.label }}</span>
              <input class="mt-2 w-full rounded-xl border border-slate-200 bg-slate-100 px-3 py-2 text-sm font-bold text-slate-700" :value="field.value" disabled />
            </label>
          </div>

          <details class="rounded-2xl border border-slate-200">
            <summary class="cursor-pointer px-4 py-3 text-sm font-black">技术字段：消息指纹、Payload JSON、完整原始字段</summary>
            <div class="border-t border-slate-100 p-4">
              <div class="grid gap-3 md:grid-cols-2">
                <label v-for="field in detailFields" :key="field.key" class="rounded-2xl bg-slate-50 p-4">
                  <span class="text-xs font-black tracking-wide text-slate-400">{{ field.label }}</span>
                  <input class="mt-2 w-full rounded-xl border border-slate-200 bg-slate-100 px-3 py-2 text-sm font-bold text-slate-700" :value="field.value" disabled />
                </label>
              </div>
            </div>
            <pre class="max-h-[460px] overflow-auto rounded-b-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(activeDetail, null, 2) }}</pre>
          </details>
        </div>
      </section>
    </div>
  </section>
</template>