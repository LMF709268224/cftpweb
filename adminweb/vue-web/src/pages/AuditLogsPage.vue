<script setup lang="ts">
import { ChevronDown, FileText, Loader2, RefreshCw, Search, SlidersHorizontal, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import {
  AUDIT_ACTION_SUGGESTIONS,
  AUDIT_RESOURCE_TYPE_SUGGESTIONS,
  AUDIT_SOURCE_SERVICE_LABELS,
  AUDIT_SOURCE_SERVICE_OPTIONS,
  AUDIT_STATUS_OPTIONS,
} from "@/lib/auditFilterOptions"
import { formatDate, humanizeKey, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"

const pageSize = 20

const { t, isZh } = useAdminLanguage()
const copy = computed(() => t.value.auditLogs)
const logs = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const detail = ref<JsonRecord | null>(null)
const loading = ref(false)
const detailLoading = ref(false)
const detailOpen = ref(false)
const advancedOpen = ref(false)
const page = ref(1)
const total = ref(0)
const hasMore = ref(false)
const nextCursor = ref("")
const prevCursor = ref("")
const lastPage = ref(1)
const filters = ref({
  keyword: "",
  source_service: "",
  action: "",
  status: "",
  operator_id: "",
  resource_type: "",
  resource_id: "",
  start_time: "",
  end_time: "",
})

const advancedFilterKeys = ["source_service", "action", "status", "operator_id", "resource_type", "resource_id", "start_time", "end_time"] as const
const filterNames = computed(() => ({
  source_service: copy.value.filters.sourceService,
  action: copy.value.filters.action,
  status: copy.value.filters.status,
  operator_id: copy.value.filters.operator,
  resource_type: copy.value.filters.resourceType,
  resource_id: copy.value.filters.resourceId,
  start_time: copy.value.filters.startTime,
  end_time: copy.value.filters.endTime,
}))
const activeAdvancedFilters = computed(() => advancedFilterKeys.flatMap((key) => {
  const value = filters.value[key].trim()
  if (!value) return []
  return [{ key, label: filterNames.value[key], value: key === "source_service" ? sourceServiceLabel(value) : value }]
}))
const actionSuggestions = computed(() => {
  const source = filters.value.source_service
  if (!source) return AUDIT_ACTION_SUGGESTIONS
  return [...new Set(logs.value
    .filter((log) => String(log.source_service || "") === source)
    .map((log) => String(log.action || "").trim())
    .filter(Boolean))].sort()
})

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const canPrevious = computed(() => page.value > 1)
const canNext = computed(() => hasMore.value)
const detailEntries = computed(() => Object.entries(detail.value || {}))

function asRecordList(value: unknown) {
  return Array.isArray(value) ? value.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item)) : []
}

function fieldText(record: JsonRecord | null, keys: string[], fallback = "-") {
  if (!record) return fallback
  for (const key of keys) {
    const value = record[key]
    if (value !== undefined && value !== null && String(value).trim()) {
      return String(value)
    }
  }
  return fallback
}

function auditId(record: JsonRecord | null) {
  return fieldText(record, ["audit_ulid", "auditUlid", "id"], "")
}

function sourceServiceLabel(value: unknown) {
  const source = String(value || "").trim()
  const labels = AUDIT_SOURCE_SERVICE_LABELS[source as keyof typeof AUDIT_SOURCE_SERVICE_LABELS]
  return labels ? labels[isZh.value ? "zh" : "en"] : source || "-"
}

function toRFC3339Local(value: string) {
  if (!value) return ""
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toISOString()
}

function prettyValue(value: unknown) {
  if (typeof value === "string" && value.trim()) {
    const trimmed = value.trim()
    if (trimmed.startsWith("{") || trimmed.startsWith("[")) {
      try {
        return JSON.stringify(JSON.parse(trimmed), null, 2)
      } catch {
        return value
      }
    }
  }
  if (value && typeof value === "object") return JSON.stringify(value, null, 2)
  return String(value ?? "-")
}

function summary(record: JsonRecord | null) {
  if (!record) return {}
  const value = record.summary
  return value && typeof value === "object" && !Array.isArray(value) ? (value as JsonRecord) : record
}

let listRequestId = 0
let detailRequestId = 0

function invalidateDetailRequest() {
  detailRequestId += 1
  detailLoading.value = false
}

function isCurrentDetailRequest(requestId: number, id: string) {
  return requestId === detailRequestId && detailOpen.value && auditId(selected.value) === id
}

async function load() {
  const requestId = ++listRequestId
  loading.value = true
  try {
    const params = new URLSearchParams()
    params.set("page_size", String(pageSize))

    let cursor = ""

    if (page.value > lastPage.value) {

      cursor = nextCursor.value

    } else if (page.value < lastPage.value) {

      cursor = prevCursor.value


    }

    

    if (cursor) params.set("cursor", cursor)


    for (const [key, value] of Object.entries(filters.value)) {
      const text = String(value || "").trim()
      if (!text) continue
      params.set(key, key === "start_time" || key === "end_time" ? toRFC3339Local(text) : text)
    }
    const data = await apiClient<JsonRecord>(`/api/audit/logs?${params}`)
    if (requestId !== listRequestId) return
    logs.value = asRecordList(data.items)
    total.value = Number(data.total || logs.value.length)
    const isBackward = page.value < lastPage.value
    hasMore.value = isBackward ? true : Boolean(data.has_more)
    lastPage.value = page.value
nextCursor.value = String(data.next_cursor || "")
    prevCursor.value = String(data?.prev_cursor || "")

    lastPage.value = page.value
    if (!logs.value.some((item) => auditId(item) === auditId(selected.value))) {
      invalidateDetailRequest()
      selected.value = logs.value[0] || null
      detail.value = null
      detailOpen.value = false
    }
  } catch (err) {
    if (requestId !== listRequestId) return
    console.error(err)
    logs.value = []
    total.value = 0
    hasMore.value = false
    nextCursor.value = ""
    selected.value = null
    invalidateDetailRequest()
    detail.value = null
    detailOpen.value = false
    toast.error(copy.value.toasts.loadFailed)
  } finally {
    if (requestId === listRequestId) loading.value = false
  }
}

function resetAndLoad() {
  page.value = 1
  lastPage.value = 1

  prevCursor.value = ""
  nextCursor.value = ""
  hasMore.value = false
  void load()
}

function removeAdvancedFilter(key: (typeof advancedFilterKeys)[number]) {
  filters.value[key] = ""
  resetAndLoad()
}

function clearFilters() {
  for (const key of advancedFilterKeys) filters.value[key] = ""
  filters.value.keyword = ""
  resetAndLoad()
}

async function openDetail(record: JsonRecord) {
  const id = auditId(record)
  if (!id) return
  const requestId = ++detailRequestId
  selected.value = record
  detailOpen.value = true
  detailLoading.value = true
  detail.value = null
  try {
    const response = await apiClient<JsonRecord>(`/api/audit/logs/${encodeURIComponent(id)}`)
    if (!isCurrentDetailRequest(requestId, id)) return
    const responseId = auditId(response)
    if (responseId && responseId !== id) throw new Error(`Audit detail response ID does not match ${id}`)
    detail.value = response
  } catch (err) {
    if (!isCurrentDetailRequest(requestId, id)) return
    console.error(err)
    toast.error(copy.value.toasts.detailLoadFailed)
  } finally {
    if (isCurrentDetailRequest(requestId, id)) detailLoading.value = false
  }
}

function closeDetail() {
  invalidateDetailRequest()
  detailOpen.value = false
  detail.value = null
}

function previousPage() {
  if (!canPrevious.value) return
  page.value -= 1
  void load()
}

function nextPage() {
  if (!canNext.value) return
  page.value += 1
  void load()
}

onMounted(load)
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-5 px-4 py-5 md:gap-6 md:px-8 md:py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div class="min-w-0">
        <p class="text-sm font-black uppercase tracking-[0.2em] text-sky-600">{{ copy.eyebrow }}</p>
        <h1 class="mt-2 text-3xl font-black tracking-tight text-slate-950 md:text-4xl">{{ copy.title }}</h1>
        <p class="mt-2 max-w-3xl text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" :disabled="loading" @click="load">
        <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
        <RefreshCw v-else class="h-4 w-4" />
        {{ copy.refresh }}
      </button>
    </header>

    <section class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm md:p-5">
      <div class="flex flex-col gap-3 md:flex-row md:items-end">
        <label class="grid min-w-0 flex-1 gap-2 text-sm font-bold">
          {{ copy.filters.keyword }}
          <input v-model="filters.keyword" class="h-11 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.placeholders.keyword" @keydown.enter="resetAndLoad" />
        </label>
        <div class="flex gap-2">
          <button class="inline-flex h-11 flex-1 items-center justify-center gap-2 rounded-xl border border-slate-200 px-4 text-sm font-bold md:flex-none" type="button" :aria-expanded="advancedOpen" aria-controls="audit-advanced-filters" @click="advancedOpen = !advancedOpen">
            <SlidersHorizontal class="h-4 w-4" />
            {{ copy.advancedFilters }}
            <span v-if="activeAdvancedFilters.length" class="rounded-full bg-slate-100 px-1.5 text-xs">{{ activeAdvancedFilters.length }}</span>
            <ChevronDown class="h-4 w-4 transition-transform" :class="advancedOpen ? 'rotate-180' : ''" />
          </button>
          <button class="inline-flex h-11 flex-1 items-center justify-center gap-2 rounded-xl bg-blue-700 px-5 text-sm font-black text-white md:flex-none" type="button" @click="resetAndLoad">
            <Search class="h-4 w-4" />
            {{ copy.search }}
          </button>
        </div>
      </div>
      <div v-if="activeAdvancedFilters.length" class="mt-3 flex flex-wrap items-center gap-2">
        <button v-for="filter in activeAdvancedFilters" :key="filter.key" class="inline-flex max-w-full items-center gap-1.5 rounded border border-slate-200 bg-slate-50 px-2 py-1 text-xs font-semibold text-slate-700" type="button" :title="copy.removeFilter" @click="removeAdvancedFilter(filter.key)">
          <span class="truncate">{{ filter.label }}: {{ filter.value }}</span>
          <X class="h-3.5 w-3.5 shrink-0" />
        </button>
        <button class="text-xs font-bold text-blue-700 hover:underline" type="button" @click="clearFilters">{{ copy.clearFilters }}</button>
      </div>
      <div v-if="advancedOpen" id="audit-advanced-filters" class="mt-4 grid gap-4 border-t border-slate-200 pt-4 md:grid-cols-2 xl:grid-cols-4">
        <label class="grid gap-2 text-sm font-bold">
          {{ copy.filters.sourceService }}
          <select v-model="filters.source_service" class="h-11 w-full rounded-xl border border-slate-200 bg-white px-3" @change="filters.action = ''">
            <option value="">{{ copy.options.allSources }}</option>
            <option v-for="source in AUDIT_SOURCE_SERVICE_OPTIONS" :key="source" :value="source">{{ sourceServiceLabel(source) }} ({{ source }})</option>
          </select>
        </label>
        <label class="grid gap-2 text-sm font-bold">
          {{ copy.filters.action }}
          <input v-model="filters.action" list="audit-action-options" class="h-11 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.placeholders.action" @keydown.enter="resetAndLoad" />
          <datalist id="audit-action-options">
            <option v-for="action in actionSuggestions" :key="action" :value="action" />
          </datalist>
        </label>
        <label class="grid gap-2 text-sm font-bold">
          {{ copy.filters.status }}
          <select v-model="filters.status" class="h-11 w-full rounded-xl border border-slate-200 bg-white px-3">
            <option value="">{{ copy.options.allStatuses }}</option>
            <option v-for="status in AUDIT_STATUS_OPTIONS" :key="status" :value="status">{{ copy.options.statuses[status] }}</option>
          </select>
        </label>
        <label class="grid gap-2 text-sm font-bold">
          {{ copy.filters.operator }}
          <input v-model="filters.operator_id" class="h-11 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.placeholders.operator" @keydown.enter="resetAndLoad" />
        </label>
        <label class="grid gap-2 text-sm font-bold">
          {{ copy.filters.resourceType }}
          <input v-model="filters.resource_type" list="audit-resource-type-options" class="h-11 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.placeholders.resourceType" @keydown.enter="resetAndLoad" />
          <datalist id="audit-resource-type-options">
            <option v-for="resourceType in AUDIT_RESOURCE_TYPE_SUGGESTIONS" :key="resourceType" :value="resourceType" />
          </datalist>
        </label>
        <label class="grid gap-2 text-sm font-bold">
          {{ copy.filters.resourceId }}
          <input v-model="filters.resource_id" class="h-11 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.placeholders.resourceId" @keydown.enter="resetAndLoad" />
        </label>
        <label class="grid gap-2 text-sm font-bold">
          {{ copy.filters.startTime }}
          <span class="relative">
            <input
              v-model="filters.start_time"
              class="h-11 w-full rounded-xl border border-slate-200 px-3"
              :class="!isZh && !filters.start_time ? 'audit-datetime-empty' : ''"
              type="datetime-local"
              :lang="isZh ? 'zh-CN' : 'en-US'"
            />
            <span v-if="!isZh && !filters.start_time" class="pointer-events-none absolute inset-y-0 left-3 flex items-center text-sm font-semibold text-slate-400">
              {{ copy.placeholders.dateTime }}
            </span>
          </span>
        </label>
        <label class="grid gap-2 text-sm font-bold">
          {{ copy.filters.endTime }}
          <span class="relative">
            <input
              v-model="filters.end_time"
              class="h-11 w-full rounded-xl border border-slate-200 px-3"
              :class="!isZh && !filters.end_time ? 'audit-datetime-empty' : ''"
              type="datetime-local"
              :lang="isZh ? 'zh-CN' : 'en-US'"
            />
            <span v-if="!isZh && !filters.end_time" class="pointer-events-none absolute inset-y-0 left-3 flex items-center text-sm font-semibold text-slate-400">
              {{ copy.placeholders.dateTime }}
            </span>
          </span>
        </label>
      </div>
    </section>

    <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm md:rounded-3xl">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 px-4 py-4 md:p-5">
        <div class="min-w-0">
          <h2 class="text-xl font-black">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
        <span class="shrink-0 rounded-full bg-slate-100 px-3 py-1 text-sm font-bold text-slate-500">{{ copy.totalText(total) }}</span>
      </div>

      <div v-if="loading" class="px-4 py-10 text-center text-slate-500 md:p-12">
        <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
        {{ copy.loading }}
      </div>
      <div v-else-if="!logs.length" class="px-4 py-10 text-center text-slate-500 md:p-12">{{ copy.empty }}</div>
      <div v-else>
        <div class="hidden grid-cols-[180px_130px_minmax(0,1fr)_180px_120px] gap-4 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-400 lg:grid">
          <span>{{ copy.columns.time }}</span>
          <span>{{ copy.columns.service }}</span>
          <span>{{ copy.columns.summary }}</span>
          <span>{{ copy.columns.operator }}</span>
          <span class="text-center">{{ copy.columns.status }}</span>
        </div>
        <button
          v-for="item in logs"
          :key="auditId(item)"
          class="grid w-full gap-3 border-b border-slate-100 px-4 py-4 text-left transition last:border-b-0 hover:bg-slate-50 md:px-5 lg:grid-cols-[180px_130px_minmax(0,1fr)_180px_120px] lg:items-center lg:gap-4"
          :class="auditId(item) === auditId(selected) ? 'bg-sky-50/70' : ''"
          type="button"
          @click="openDetail(item)"
        >
          <span class="text-sm font-semibold text-slate-600">{{ formatDate(fieldText(item, ["created_at"])) }}</span>
          <span class="text-xs font-black text-sky-700" :title="fieldText(item, ['source_service'])">{{ sourceServiceLabel(item.source_service) }}</span>
          <span class="min-w-0">
            <span class="block break-words text-base font-black text-slate-950 lg:truncate">{{ fieldText(item, ["summary_text", "action"]) }}</span>
            <span class="mt-1 block break-all text-xs font-semibold text-slate-500 lg:truncate">{{ fieldText(item, ["resource_display_name", "resource_id", "audit_ulid"]) }}</span>
          </span>
          <span class="break-all text-sm font-bold text-slate-700 lg:truncate">{{ fieldText(item, ["operator_name", "operator_id"]) }}</span>
          <span class="justify-self-start rounded-full border border-slate-200 bg-slate-50 px-3 py-1 text-xs font-black uppercase text-slate-600 lg:justify-self-center">
            {{ fieldText(item, ["status"]) }}
          </span>
        </button>
      </div>

      <div class="flex flex-wrap items-center justify-end gap-3 border-t border-slate-200 px-4 py-4 md:p-5">
        <span class="w-full text-sm font-bold text-slate-500 md:mr-auto md:w-auto">{{ copy.pageText(page, totalPages) }}</span>
        <button class="flex-1 rounded-xl border px-4 py-2 font-bold disabled:opacity-40 md:flex-none" type="button" :disabled="!canPrevious || loading" @click="previousPage">{{ copy.prev }}</button>
        <button class="flex-1 rounded-xl border px-4 py-2 font-bold disabled:opacity-40 md:flex-none" type="button" :disabled="!canNext || loading" @click="nextPage">{{ copy.next }}</button>
      </div>
    </section>

    <section v-if="detailOpen" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-0 md:p-6">
      <div v-modal-dialog="closeDetail" class="flex h-full max-h-none w-full max-w-[980px] flex-col overflow-hidden rounded-none bg-white shadow-2xl md:h-auto md:max-h-[88vh] md:rounded-3xl">
        <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-4 py-4 md:p-5">
          <div class="min-w-0">
            <h2 class="text-xl font-black">{{ copy.detailTitle }}</h2>
            <p class="mt-1 break-all text-sm text-slate-500">{{ auditId(selected) }}</p>
          </div>
          <button class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="closeDetail">
            <X class="h-5 w-5" />
          </button>
        </div>

        <div v-if="detailLoading" class="px-4 py-10 text-center text-slate-500 md:p-12">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          {{ copy.detailLoading }}
        </div>
        <div v-else class="min-h-0 flex-1 space-y-5 overflow-y-auto p-4 md:p-5">
          <div class="grid gap-4 md:grid-cols-3">
            <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
              <div class="text-xs font-black uppercase text-slate-400">{{ copy.summaryCards.service }}</div>
              <div class="mt-2 break-words text-lg font-black text-slate-950" :title="fieldText(summary(detail), ['source_service'])">{{ sourceServiceLabel(summary(detail).source_service) }}</div>
            </div>
            <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
              <div class="text-xs font-black uppercase text-slate-400">{{ copy.summaryCards.action }}</div>
              <div class="mt-2 break-words text-lg font-black text-slate-950">{{ fieldText(summary(detail), ["action"]) }}</div>
            </div>
            <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
              <div class="text-xs font-black uppercase text-slate-400">{{ copy.summaryCards.resource }}</div>
              <div class="mt-2 break-words text-lg font-black text-slate-950">{{ fieldText(summary(detail), ["resource_display_name", "resource_id"]) }}</div>
            </div>
          </div>

          <div class="rounded-2xl border border-slate-200 bg-slate-50">
            <div class="flex items-center gap-2 border-b border-slate-200 px-4 py-3 text-sm font-black">
              <FileText class="h-4 w-4 text-blue-700" />
              {{ copy.detailFields }}
            </div>
            <div class="divide-y divide-slate-200 px-4">
              <div v-for="[key, value] in detailEntries" :key="key" class="grid gap-2 py-3 text-sm md:grid-cols-[170px_1fr]">
                <div class="text-[11px] font-black uppercase text-slate-400">{{ humanizeKey(key) }}</div>
                <pre class="whitespace-pre-wrap break-words font-sans font-semibold text-slate-700">{{ prettyValue(value) }}</pre>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  </section>
</template>

<style scoped>
.audit-datetime-empty::-webkit-datetime-edit,
.audit-datetime-empty::-webkit-datetime-edit-text,
.audit-datetime-empty::-webkit-datetime-edit-month-field,
.audit-datetime-empty::-webkit-datetime-edit-day-field,
.audit-datetime-empty::-webkit-datetime-edit-year-field,
.audit-datetime-empty::-webkit-datetime-edit-hour-field,
.audit-datetime-empty::-webkit-datetime-edit-minute-field,
.audit-datetime-empty::-webkit-datetime-edit-ampm-field {
  color: transparent;
}
</style>
