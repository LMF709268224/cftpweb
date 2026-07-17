<script setup lang="ts">
import { ChevronLeft, ChevronRight, Loader2, RefreshCw, RotateCcw, Search, Webhook, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import JsonPreview from "@/components/JsonPreview.vue"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
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
const page = ref(1)
const total = ref(0)
const hasMore = ref(false)
const nextCursor = ref("")
const prevCursor = ref("")
const lastPage = ref(1)
const status = ref("")
const { t } = useAdminLanguage()
const copy = computed(() => t.value.webhooks)

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / PAGE_SIZE)))
const activeDetail = computed(() => detail.value || selected.value)
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

async function load(targetPage = page.value) {
  loading.value = true
  try {
    const params = new URLSearchParams({
      page_size: String(PAGE_SIZE),
    })

    let cursor = ""

    if (targetPage > lastPage.value) {

      cursor = nextCursor.value

    } else if (targetPage < lastPage.value) {

      cursor = prevCursor.value


    }

    

    if (cursor) params.set("cursor", cursor)


    if (status.value) params.set("status", status.value)

    const data = await apiClient<JsonRecord>(`/api/audit/webhooks?${params}`)
    const rawList = data.webhook_messages || data.messages || data.items
    const list = Array.isArray(rawList) ? rawList : []
    const selectedKey = selected.value ? msgKey(selected.value) : ""



    messages.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    total.value = Number(data.total || messages.value.length) || 0
    const isBackward = page.value < lastPage.value
    hasMore.value = isBackward ? true : Boolean(data.has_more)
    lastPage.value = page.value
nextCursor.value = String(data.next_cursor || "")
    prevCursor.value = String(data?.prev_cursor || "")

    lastPage.value = targetPage
    page.value = targetPage
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
    hasMore.value = false
    nextCursor.value = ""
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
    toast.error(apiErrorMessage(err, copy.value.toasts.reprocessFailed))
  } finally {
    reprocessing.value = ""
  }
}

function search() {
  page.value = 1
  lastPage.value = 1

  prevCursor.value = ""
  nextCursor.value = ""
  hasMore.value = false
  load(1)
}

function goPage(nextPage: number) {
  if (nextPage < 1 || nextPage === page.value) return
  if (nextPage > page.value && !hasMore.value) return
  if (Math.abs(nextPage - page.value) !== 1) return
  load(nextPage)
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
        <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
        <RefreshCw v-else class="h-4 w-4 text-slate-400" />
        {{ copy.refresh }}
      </button>
    </header>

    <form class="ui-card mt-6 grid gap-4 p-4 md:grid-cols-[1fr_auto] md:p-5" @submit.prevent="search">
      <select v-model="status" class="ui-input">
        <option value="">{{ copy.allStatus }}</option>
        <option value="PROCESSED">{{ copy.statusOptions.processed }}</option>
        <option value="SKIPPED">{{ copy.statusOptions.skipped }}</option>
        <option value="FAILED">{{ copy.statusOptions.failed }}</option>
        <option value="PENDING">{{ copy.statusOptions.pending }}</option>
      </select>
      <button class="ui-btn ui-btn-primary" type="submit">
        <Search class="h-4 w-4" />
        {{ copy.search }}
      </button>
    </form>

    <section class="ui-card mt-4 overflow-hidden">
      <div class="flex flex-wrap items-center justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
        <div class="min-w-0">
          <h2 class="text-base font-semibold text-ink">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
        <span class="tnum ui-badge shrink-0 bg-slate-100 text-slate-600">{{ copy.totalText(total) }}</span>
      </div>
      <div class="ui-table-head grid grid-cols-[minmax(0,1fr)_180px_180px_112px] gap-5 px-4 py-3">
        <span>{{ copy.columns.webhook }}</span>
        <span class="text-center">{{ copy.columns.status }}</span>
        <span class="text-right">{{ copy.columns.createdAt }}</span>
        <span class="text-right">{{ copy.columns.action }}</span>
      </div>

      <div v-if="loading" class="divide-y divide-hairline">
        <div v-for="i in 5" :key="`sk-${i}`" class="grid grid-cols-[minmax(0,1fr)_180px_180px_112px] items-center gap-5 px-4 py-3.5">
          <div class="min-w-0">
            <div class="skeleton h-4 w-2/5" />
            <div class="skeleton mt-2 h-3 w-3/5" />
          </div>
          <div class="skeleton h-5 w-16 justify-self-center rounded-full" />
          <div class="skeleton h-4 w-24 justify-self-end" />
          <div class="skeleton h-4 w-14 justify-self-end" />
        </div>
      </div>
      <div v-else-if="!messages.length" class="ui-empty">
        <div class="ui-empty-icon">
          <Webhook class="h-6 w-6" />
        </div>
        <p class="ui-empty-title">{{ copy.empty }}</p>
      </div>
      <div v-else class="divide-y divide-hairline">
        <div
          v-for="message in messages"
          :key="msgKey(message)"
          class="ui-table-row grid cursor-pointer grid-cols-[minmax(0,1fr)_180px_180px_112px] items-center gap-5 px-4 py-3.5"
          :class="msgKey(selected || {}) === msgKey(message) ? 'bg-primary-50/60' : ''"
          role="button"
          tabindex="0"
          @click="loadDetail(message)"
          @keydown.enter.prevent="loadDetail(message)"
          @keydown.space.prevent="loadDetail(message)"
        >
          <div class="min-w-0">
            <div class="truncate text-sm font-medium text-ink">{{ messageTitle(message) }}</div>
            <div class="tnum mt-0.5 truncate text-xs text-slate-400">{{ messageSubtitle(message) }}</div>
          </div>
          <div class="min-w-0 text-center">
            <span class="ui-badge max-w-full truncate" :class="badgeClass(messageStatus(message))">
              {{ statusText(messageStatus(message)) }}
            </span>
          </div>
          <div class="tnum text-right text-sm text-slate-500">{{ formatDate(message.created_at) || copy.noCreatedAt }}</div>
          <div class="text-right">
            <button
              class="text-sm font-medium text-primary-700 transition-colors hover:text-primary-600"
              type="button"
              @click.stop="loadDetail(message)"
            >
              {{ copy.viewDetails }}
            </button>
          </div>
        </div>
      </div>

      <div class="flex flex-col items-stretch justify-between gap-3 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center md:px-6">
        <span class="tnum text-center text-sm text-slate-500 sm:text-left">{{ copy.pageText(page, totalPages) }}</span>
        <div class="flex flex-col gap-2 sm:flex-row">
          <button
            class="ui-btn ui-btn-secondary ui-btn-sm"
            type="button"
            :disabled="page <= 1 || loading"
            @click="goPage(page - 1)"
          >
            <ChevronLeft class="h-4 w-4 text-slate-400" />
            {{ copy.prev }}
          </button>
          <button
            class="ui-btn ui-btn-secondary ui-btn-sm"
            type="button"
            :disabled="!hasMore || loading"
            @click="goPage(page + 1)"
          >
            {{ copy.next }}
            <ChevronRight class="h-4 w-4 text-slate-400" />
          </button>
        </div>
      </div>
    </section>

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="detailOpen && selected" class="ui-modal-mask">
          <section class="ui-modal-panel max-w-[1120px]">
            <div class="flex items-start justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
              <div class="flex min-w-0 items-start gap-3">
                <span class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-[10px] bg-primary-50 text-primary-700">
                  <Webhook class="h-5 w-5" />
                </span>
                <div class="min-w-0">
                  <h2 class="text-base font-semibold text-ink md:text-lg">{{ copy.detailTitle }}</h2>
                  <p class="mt-1 text-sm text-slate-500">{{ copy.detailDescription }}</p>
                </div>
              </div>
              <div class="flex shrink-0 items-center gap-2">
                <button
                  class="ui-btn ui-btn-secondary"
                  type="button"
                  :disabled="reprocessing === String(webhookMsgId(selected))"
                  @click="reprocess(selected)"
                >
                  <RotateCcw class="h-4 w-4 text-slate-400" :class="reprocessing === String(webhookMsgId(selected)) ? 'animate-spin' : ''" />
                  {{ copy.reprocess }}
                </button>
                <button
                  class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
                  type="button"
                  :aria-label="copy.close"
                  @click="closeDetail"
                >
                  <X class="h-5 w-5" />
                </button>
              </div>
            </div>

            <div v-if="detailLoading" class="space-y-3 p-4 md:p-6">
              <div class="skeleton h-24 w-full" />
              <div class="skeleton h-12 w-4/5" />
              <div class="skeleton h-12 w-3/5" />
            </div>
            <div v-else class="min-h-0 flex-1 space-y-5 overflow-y-auto p-4 md:p-6">
              <div class="rounded-2xl bg-primary-50 p-4 md:p-5">
                <div class="flex items-center gap-2 text-sm font-medium text-primary-700">
                  <Webhook class="h-4 w-4" />
                  {{ copy.currentWebhook }}
                </div>
                <div class="mt-1 break-all text-base font-semibold text-ink">{{ eventTitle(activeRecord) }}</div>
                <p class="mt-2 text-sm leading-6 text-slate-600">{{ eventDescription(activeRecord) }}</p>
                <span class="ui-badge mt-3" :class="badgeClass(messageStatus(activeDetail || selected))">
                  {{ statusText(messageStatus(activeDetail || selected)) }}
                </span>
              </div>

              <div class="grid gap-4 md:grid-cols-2">
                <div v-for="field in summaryFields" :key="field.label" class="rounded-2xl bg-slate-50 p-4">
                  <span class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ field.label }}</span>
                  <div class="tnum mt-1.5 break-words text-sm font-medium text-slate-700">{{ field.value }}</div>
                </div>
              </div>

              <details class="overflow-hidden rounded-2xl border border-edge">
                <summary class="cursor-pointer px-4 py-3 text-sm font-medium text-ink transition-colors hover:bg-slate-50">{{ copy.technicalFields }}</summary>
                <div class="border-t border-hairline p-4">
                  <div class="grid gap-4 md:grid-cols-2">
                    <div v-for="field in detailFields" :key="field.key" class="rounded-2xl bg-slate-50 p-4">
                      <span class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ field.label }}</span>
                      <div class="tnum mt-1.5 break-words text-sm font-medium text-slate-700">{{ field.value }}</div>
                    </div>
                  </div>
                </div>
                <div class="border-t border-hairline p-4">
                  <JsonPreview
                    :title="copy.rawJson"
                    :value="activeDetail || selected || {}"
                    :copy-label="copy.copyJson"
                    :copied-label="copy.copiedJson"
                    :copied-message="copy.toasts.jsonCopied"
                    :copy-error-message="copy.toasts.jsonCopyFailed"
                    max-height="460px"
                  />
                </div>
              </details>
            </div>
          </section>
        </div>
      </Transition>
    </Teleport>
  </main>
</template>
