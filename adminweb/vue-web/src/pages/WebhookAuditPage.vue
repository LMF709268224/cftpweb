<script setup lang="ts">
import { Check, ChevronLeft, ChevronRight, Copy as CopyIcon, Loader2, RefreshCw, RotateCcw, Search, Webhook, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { copyTextToClipboard } from "@/lib/clipboard"
import { formatDate, humanizeKey, isPrimitive, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { badgeClass, pickFirst } from "@/lib/status"

const PAGE_SIZE = 10

const messages = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const detail = ref<JsonRecord | null>(null)
const loading = ref(false)
const detailLoading = ref(false)
const detailOpen = ref(false)
const reprocessing = ref("")
const copiedJson = ref(false)
const page = ref(1)
const total = ref(0)
const status = ref("")
const { t } = useAdminLanguage()
const copy = computed(() => t.value.webhooks)

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / PAGE_SIZE)))
const activeDetail = computed(() => detail.value || selected.value)
const activeDetailJson = computed(() => JSON.stringify(activeDetail.value || {}, null, 2))
const activeRecord = computed(() => activeDetail.value || selected.value || {})
const summaryFields = computed(() => {
  const record = activeRecord.value
  if (!record || !Object.keys(record).length) return []
  return [
    { label: copy.value.summaryLabels.meaning, value: eventTitle(record) },
    { label: copy.value.summaryLabels.status, value: statusText(messageStatus(record)) },
    { label: copy.value.summaryLabels.confirmationNumber, value: fieldText(record, ["confirmation_number"]) },
    { label: copy.value.summaryLabels.examId, value: fieldText(record, ["exam_ulid", "exam_id"]) },
    { label: copy.value.summaryLabels.eventTime, value: formatDate(fieldText(record, ["event_timestamp"])) || "-" },
    { label: copy.value.summaryLabels.processedTime, value: formatDate(fieldText(record, ["processed_at"])) || "-" },
    { label: copy.value.summaryLabels.webhookMessageId, value: fieldText(record, ["webhook_msg_id", "id"]) },
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
  return copy.value.fieldLabels[key as keyof typeof copy.value.fieldLabels] || humanizeKey(key)
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
    result_created: copy.value.eventTitles.resultCreated,
    appointment_scheduled: copy.value.eventTitles.appointmentScheduled,
    appointment_rescheduled: copy.value.eventTitles.appointmentRescheduled,
    appointment_cancelled: copy.value.eventTitles.appointmentCancelled,
    payment_intent_succeeded: copy.value.eventTitles.paymentSucceeded,
    payment_intent_payment_failed: copy.value.eventTitles.paymentFailed,
    invoice_created: copy.value.eventTitles.invoiceCreated,
  }
  return labels[type] || eventType(message) || copy.value.eventTitles.fallback
}

function eventDescription(message: JsonRecord | null | undefined) {
  const type = eventType(message).toLowerCase()
  if (type === "result_created") return copy.value.eventDescriptions.resultCreated
  if (type === "appointment_scheduled") return copy.value.eventDescriptions.appointmentScheduled
  if (type === "appointment_rescheduled") return copy.value.eventDescriptions.appointmentRescheduled
  if (type === "appointment_cancelled") return copy.value.eventDescriptions.appointmentCancelled
  if (type.includes("payment")) return copy.value.eventDescriptions.payment
  if (type.includes("invoice")) return copy.value.eventDescriptions.invoice
  return copy.value.eventDescriptions.fallback
}

function messageSubtitle(message: JsonRecord) {
  const confirmation = fieldText(message, ["confirmation_number"])
  const examId = fieldText(message, ["exam_ulid", "exam_id"])
  if (confirmation !== "-") return `${copy.value.subtitlePrefix.confirmation}${confirmation}`
  if (examId !== "-") return `${copy.value.subtitlePrefix.exam}${examId}`
  return `${copy.value.subtitlePrefix.message}${fieldText(message, ["webhook_msg_id", "id"])}`
}

function statusText(value: unknown) {
  const statusValue = String(value || "").toUpperCase()
  const labels: Record<string, string> = {
    PROCESSED: copy.value.statuses.processed,
    SKIPPED: copy.value.statuses.skipped,
    FAILED: copy.value.statuses.failed,
    PENDING: copy.value.statuses.pending,
    SUCCESS: copy.value.statuses.success,
    IGNORED: copy.value.statuses.ignored,
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

async function copyActiveDetailJson() {
  try {
    await copyTextToClipboard(activeDetailJson.value)
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
    toast.error(copy.value.toasts.loadFailed)
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
    toast.error(copy.value.toasts.detailLoadFailed)
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
    toast.error(copy.value.toasts.missingWebhookId)
    return
  }
  reprocessing.value = String(id)
  try {
    await apiClient("/api/audit/webhooks/reprocess", {
      method: "POST",
      body: JSON.stringify({ webhook_msg_id: Number(id) }),
    })
    toast.success(copy.value.toasts.reprocessSubmitted)
    await load(page.value)
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.reprocessFailed)
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
        <h1 class="text-4xl font-black tracking-tight">{{ copy.title }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load(page)">
        <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
        {{ copy.refresh }}
      </button>
    </header>

    <form class="grid gap-4 rounded-3xl border border-slate-200 bg-white p-5 shadow-sm md:grid-cols-[1fr_auto]" @submit.prevent="search">
      <select v-model="status" class="rounded-xl border border-slate-200 px-4 py-3">
        <option value="">{{ copy.allStatus }}</option>
        <option value="PROCESSED">{{ copy.statusOptions.processed }}</option>
        <option value="SKIPPED">{{ copy.statusOptions.skipped }}</option>
        <option value="FAILED">{{ copy.statusOptions.failed }}</option>
        <option value="PENDING">{{ copy.statusOptions.pending }}</option>
      </select>
      <button class="inline-flex items-center justify-center gap-2 rounded-xl bg-blue-700 px-5 py-3 font-bold text-white" type="submit">
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
        <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ copy.totalText(total) }}</span>
      </div>
      <div class="grid grid-cols-[minmax(0,1fr)_180px_180px_112px] gap-5 border-b border-slate-200 bg-slate-50 px-5 py-3 text-xs font-black text-slate-500">
        <span>{{ copy.columns.webhook }}</span>
        <span class="text-center">{{ copy.columns.status }}</span>
        <span class="text-right">{{ copy.columns.createdAt }}</span>
        <span class="text-right">{{ copy.columns.action }}</span>
      </div>

      <div v-if="loading" class="p-12 text-center text-slate-500">
        <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
        {{ copy.loading }}
      </div>
      <div v-else-if="!messages.length" class="p-12 text-center text-slate-500">{{ copy.empty }}</div>
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
            <div class="mt-1 truncate text-sm font-semibold text-slate-600">{{ messageSubtitle(message) }}</div>
          </div>
          <div class="min-w-0 text-center">
            <span class="inline-flex max-w-full truncate rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(messageStatus(message))">
              {{ statusText(messageStatus(message)) }}
            </span>
          </div>
          <div class="text-right text-sm font-semibold text-slate-500">{{ formatDate(message.created_at) || copy.noCreatedAt }}</div>
          <div class="text-right">
            <button
              class="text-sm font-bold text-[#1890ff] transition hover:underline"
              type="button"
              @click.stop="loadDetail(message)"
            >
              {{ copy.viewDetails }}
            </button>
          </div>
        </div>
      </div>

      <div class="flex items-center justify-between gap-3 border-t border-slate-200 p-5">
        <span class="text-sm font-bold text-slate-500">{{ copy.pageText(page, totalPages) }}</span>
        <div class="flex gap-3">
          <button
            class="inline-flex items-center gap-2 rounded-xl border px-3 py-2 text-sm font-bold disabled:cursor-not-allowed disabled:opacity-40"
            type="button"
            :disabled="page <= 1 || loading"
            @click="goPage(page - 1)"
          >
            <ChevronLeft class="h-4 w-4" />
            {{ copy.prev }}
          </button>
          <button
            class="inline-flex items-center gap-2 rounded-xl border px-3 py-2 text-sm font-bold disabled:cursor-not-allowed disabled:opacity-40"
            type="button"
            :disabled="page >= totalPages || loading"
            @click="goPage(page + 1)"
          >
            {{ copy.next }}
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
                <h2 class="text-2xl font-black text-slate-950">{{ copy.detailTitle }}</h2>
                <p class="mt-1 text-sm text-slate-500">{{ copy.detailDescription }}</p>
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
                {{ copy.reprocess }}
              </button>
              <button
                class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
                type="button"
                :aria-label="copy.close"
                @click="closeDetail"
              >
                <X class="h-5 w-5" />
              </button>
            </div>
          </div>

          <div v-if="detailLoading" class="p-12 text-center text-slate-500">
            <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
            {{ copy.detailLoading }}
          </div>
          <div v-else class="min-h-0 flex-1 space-y-5 overflow-y-auto p-5">
            <div class="rounded-2xl border border-blue-100 bg-blue-50 p-4">
              <div class="flex items-center gap-2 text-sm font-black text-blue-700">
                <Webhook class="h-4 w-4" />
                {{ copy.currentWebhook }}
              </div>
              <div class="mt-1 break-all text-lg font-black text-slate-950">{{ eventTitle(activeRecord) }}</div>
              <p class="mt-2 text-sm font-semibold leading-6 text-slate-600">{{ eventDescription(activeRecord) }}</p>
              <div class="mt-2 inline-flex rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(messageStatus(activeDetail || selected))">
                {{ statusText(messageStatus(activeDetail || selected)) }}
              </div>
            </div>

            <div class="grid gap-4 md:grid-cols-2">
              <div v-for="field in summaryFields" :key="field.label" class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                <span class="text-xs font-black tracking-wide text-slate-400">{{ field.label }}</span>
                <div class="mt-2 break-words rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-bold text-slate-700">{{ field.value }}</div>
              </div>
            </div>

            <details class="rounded-2xl border border-slate-200">
              <summary class="cursor-pointer px-4 py-3 text-sm font-black">{{ copy.technicalFields }}</summary>
              <div class="border-t border-slate-100 p-4">
                <div class="grid gap-4 md:grid-cols-2">
                  <div v-for="field in detailFields" :key="field.key" class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                    <span class="text-xs font-black tracking-wide text-slate-400">{{ field.label }}</span>
                    <div class="mt-2 break-words rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-bold text-slate-700">{{ field.value }}</div>
                  </div>
                </div>
              </div>
              <details class="border-t border-slate-100 p-4">
                <summary class="cursor-pointer text-sm font-black text-slate-700">{{ copy.rawJson }}</summary>
                <div class="mt-4 overflow-hidden rounded-2xl bg-slate-950">
                  <div class="flex items-center justify-between gap-3 border-b border-white/10 px-4 py-3">
                    <span class="text-xs font-black uppercase text-slate-400">{{ copy.rawJson }}</span>
                    <button class="inline-flex h-8 items-center gap-2 rounded-lg border border-white/10 px-3 text-xs font-bold text-slate-100 transition hover:bg-white/10" type="button" @click="copyActiveDetailJson">
                      <Check v-if="copiedJson" class="h-3.5 w-3.5" />
                      <CopyIcon v-else class="h-3.5 w-3.5" />
                      {{ copiedJson ? copy.copiedJson : copy.copyJson }}
                    </button>
                  </div>
                  <pre class="max-h-[460px] overflow-auto p-5 text-xs leading-6 text-slate-100">{{ activeDetailJson }}</pre>
                </div>
              </details>
            </details>
          </div>
        </section>
      </div>
    </Teleport>
  </section>
</template>
