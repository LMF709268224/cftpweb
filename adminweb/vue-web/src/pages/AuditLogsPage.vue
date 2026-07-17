<script setup lang="ts">
import { FileText, Loader2, RefreshCw, Search, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, humanizeKey, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"

const pageSize = 20

const { t } = useAdminLanguage()
const copy = computed(() => t.value.auditLogs)
const logs = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const detail = ref<JsonRecord | null>(null)
const loading = ref(false)
const detailLoading = ref(false)
const detailOpen = ref(false)
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

function statusBadgeClass(value: string) {
  const status = value.trim().toUpperCase()
  if (status.includes("SUCCESS") || status.includes("OK") || status.includes("COMPLETED")) return "bg-emerald-50 text-emerald-700"
  if (status.includes("FAIL") || status.includes("ERROR") || status.includes("REJECT")) return "bg-red-50 text-red-700"
  if (status.includes("PENDING") || status.includes("WAIT")) return "bg-amber-50 text-amber-700"
  return "bg-slate-100 text-slate-600"
}

async function load() {
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
    logs.value = asRecordList(data.items)
    total.value = Number(data.total || logs.value.length)
    const isBackward = page.value < lastPage.value
    hasMore.value = isBackward ? true : Boolean(data.has_more)
    lastPage.value = page.value
nextCursor.value = String(data.next_cursor || "")
    prevCursor.value = String(data?.prev_cursor || "")

    lastPage.value = page.value
if (!logs.value.some((item) => auditId(item) === auditId(selected.value))) {
      selected.value = logs.value[0] || null
    }
  } catch (err) {
    console.error(err)
    logs.value = []
    total.value = 0
    hasMore.value = false
    nextCursor.value = ""
    selected.value = null
    toast.error(copy.value.toasts.loadFailed)
  } finally {
    loading.value = false
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

async function openDetail(record: JsonRecord) {
  const id = auditId(record)
  if (!id) return
  selected.value = record
  detailOpen.value = true
  detailLoading.value = true
  detail.value = null
  try {
    detail.value = await apiClient<JsonRecord>(`/api/audit/logs/${encodeURIComponent(id)}`)
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.detailLoadFailed)
  } finally {
    detailLoading.value = false
  }
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
  <main class="ui-page">
    <header class="flex flex-wrap items-start justify-between gap-4 md:items-center">
      <div class="min-w-0">
        <h1 class="ui-page-title">{{ copy.title }}</h1>
        <p class="ui-page-subtitle">{{ copy.subtitle }}</p>
      </div>
      <button class="ui-btn ui-btn-secondary" type="button" :disabled="loading" @click="load">
        <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
        <RefreshCw v-else class="h-4 w-4 text-slate-400" />
        {{ copy.refresh }}
      </button>
    </header>

    <section class="ui-card mt-6 p-4 md:p-5">
      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-5">
        <label class="grid gap-1.5">
          <span class="ui-label">{{ copy.filters.keyword }}</span>
          <input v-model="filters.keyword" class="ui-input" :placeholder="copy.placeholders.keyword" @keydown.enter="resetAndLoad" />
        </label>
        <label class="grid gap-1.5">
          <span class="ui-label">{{ copy.filters.sourceService }}</span>
          <input v-model="filters.source_service" class="ui-input" :placeholder="copy.placeholders.sourceService" @keydown.enter="resetAndLoad" />
        </label>
        <label class="grid gap-1.5">
          <span class="ui-label">{{ copy.filters.action }}</span>
          <input v-model="filters.action" class="ui-input" :placeholder="copy.placeholders.action" @keydown.enter="resetAndLoad" />
        </label>
        <label class="grid gap-1.5">
          <span class="ui-label">{{ copy.filters.status }}</span>
          <input v-model="filters.status" class="ui-input" :placeholder="copy.placeholders.status" @keydown.enter="resetAndLoad" />
        </label>
        <label class="grid gap-1.5">
          <span class="ui-label">{{ copy.filters.operator }}</span>
          <input v-model="filters.operator_id" class="ui-input" :placeholder="copy.placeholders.operator" @keydown.enter="resetAndLoad" />
        </label>
        <label class="grid gap-1.5">
          <span class="ui-label">{{ copy.filters.resourceType }}</span>
          <input v-model="filters.resource_type" class="ui-input" :placeholder="copy.placeholders.resourceType" @keydown.enter="resetAndLoad" />
        </label>
        <label class="grid gap-1.5">
          <span class="ui-label">{{ copy.filters.resourceId }}</span>
          <input v-model="filters.resource_id" class="ui-input" :placeholder="copy.placeholders.resourceId" @keydown.enter="resetAndLoad" />
        </label>
        <label class="grid gap-1.5">
          <span class="ui-label">{{ copy.filters.startTime }}</span>
          <input v-model="filters.start_time" class="ui-input" type="datetime-local" />
        </label>
        <label class="grid gap-1.5">
          <span class="ui-label">{{ copy.filters.endTime }}</span>
          <input v-model="filters.end_time" class="ui-input" type="datetime-local" />
        </label>
        <button class="ui-btn ui-btn-primary w-full self-end" type="button" @click="resetAndLoad">
          <Search class="h-4 w-4" />
          {{ copy.search }}
        </button>
      </div>
    </section>

    <section class="ui-card mt-4 overflow-hidden">
      <div class="flex flex-wrap items-center justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
        <div class="min-w-0">
          <h2 class="text-base font-semibold text-ink">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
        <span class="tnum ui-badge shrink-0 bg-slate-100 text-slate-600">{{ copy.totalText(total) }}</span>
      </div>

      <div v-if="loading" class="divide-y divide-hairline">
        <div v-for="i in 6" :key="`sk-${i}`" class="grid gap-3 px-4 py-3.5 md:px-5 lg:grid-cols-[180px_130px_minmax(0,1fr)_180px_120px] lg:items-center lg:gap-4">
          <div class="skeleton h-4 w-28" />
          <div class="skeleton h-4 w-16" />
          <div class="min-w-0">
            <div class="skeleton h-4 w-3/5" />
            <div class="skeleton mt-2 h-3 w-2/5" />
          </div>
          <div class="skeleton h-4 w-24" />
          <div class="skeleton h-5 w-16 rounded-full lg:justify-self-center" />
        </div>
      </div>
      <div v-else-if="!logs.length" class="ui-empty">
        <div class="ui-empty-icon">
          <FileText class="h-6 w-6" />
        </div>
        <p class="ui-empty-title">{{ copy.empty }}</p>
      </div>
      <div v-else>
        <div class="ui-table-head hidden grid-cols-[180px_130px_minmax(0,1fr)_180px_120px] gap-4 px-5 py-3 lg:grid">
          <span>{{ copy.columns.time }}</span>
          <span>{{ copy.columns.service }}</span>
          <span>{{ copy.columns.summary }}</span>
          <span>{{ copy.columns.operator }}</span>
          <span class="text-center">{{ copy.columns.status }}</span>
        </div>
        <div class="divide-y divide-hairline">
          <button
            v-for="item in logs"
            :key="auditId(item)"
            class="ui-table-row grid w-full gap-3 px-4 py-3.5 text-left md:px-5 lg:grid-cols-[180px_130px_minmax(0,1fr)_180px_120px] lg:items-center lg:gap-4"
            :class="auditId(item) === auditId(selected) ? 'bg-primary-50/60' : ''"
            type="button"
            @click="openDetail(item)"
          >
            <span class="tnum text-sm text-slate-500">{{ formatDate(fieldText(item, ["created_at"])) }}</span>
            <span class="font-mono text-xs font-medium uppercase text-primary-700">{{ fieldText(item, ["source_service"]) }}</span>
            <span class="min-w-0">
              <span class="block break-words text-sm font-medium text-ink lg:truncate">{{ fieldText(item, ["summary_text", "action"]) }}</span>
              <span class="tnum mt-0.5 block break-all text-xs text-slate-400 lg:truncate">{{ fieldText(item, ["resource_display_name", "resource_id", "audit_ulid"]) }}</span>
            </span>
            <span class="break-all text-sm text-slate-600 lg:truncate">{{ fieldText(item, ["operator_name", "operator_id"]) }}</span>
            <span class="ui-badge justify-self-start uppercase lg:justify-self-center" :class="statusBadgeClass(fieldText(item, ['status']))">
              {{ fieldText(item, ["status"]) }}
            </span>
          </button>
        </div>
      </div>

      <div class="flex flex-col items-stretch justify-between gap-3 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center md:px-6">
        <span class="tnum text-center text-sm text-slate-500 sm:text-left">{{ copy.pageText(page, totalPages) }}</span>
        <div class="flex flex-col gap-2 sm:flex-row">
          <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canPrevious || loading" @click="previousPage">{{ copy.prev }}</button>
          <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canNext || loading" @click="nextPage">{{ copy.next }}</button>
        </div>
      </div>
    </section>

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="detailOpen" class="ui-modal-mask">
          <section class="ui-modal-panel max-w-[980px]">
            <div class="flex items-start justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
              <div class="min-w-0">
                <h2 class="text-base font-semibold text-ink md:text-lg">{{ copy.detailTitle }}</h2>
                <p class="tnum mt-1 break-all text-sm text-slate-500">{{ auditId(selected) }}</p>
              </div>
              <button class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600" type="button" :aria-label="copy.close" @click="detailOpen = false">
                <X class="h-5 w-5" />
              </button>
            </div>

            <div v-if="detailLoading" class="space-y-3 p-4 md:p-6">
              <div class="skeleton h-20 w-full" />
              <div class="skeleton h-12 w-4/5" />
              <div class="skeleton h-12 w-3/5" />
            </div>
            <div v-else class="min-h-0 flex-1 space-y-5 overflow-y-auto p-4 md:p-6">
              <div class="grid gap-4 md:grid-cols-3">
                <div class="rounded-2xl bg-slate-50 p-4">
                  <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.summaryCards.service }}</div>
                  <div class="mt-1.5 break-words text-base font-semibold text-ink">{{ fieldText(summary(detail), ["source_service"]) }}</div>
                </div>
                <div class="rounded-2xl bg-slate-50 p-4">
                  <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.summaryCards.action }}</div>
                  <div class="mt-1.5 break-words text-base font-semibold text-ink">{{ fieldText(summary(detail), ["action"]) }}</div>
                </div>
                <div class="rounded-2xl bg-slate-50 p-4">
                  <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.summaryCards.resource }}</div>
                  <div class="mt-1.5 break-words text-base font-semibold text-ink">{{ fieldText(summary(detail), ["resource_display_name", "resource_id"]) }}</div>
                </div>
              </div>

              <div class="overflow-hidden rounded-2xl border border-edge">
                <div class="flex items-center gap-2 border-b border-hairline bg-slate-50/60 px-4 py-3 text-sm font-semibold text-ink">
                  <FileText class="h-4 w-4 text-slate-400" />
                  {{ copy.detailFields }}
                </div>
                <div class="divide-y divide-hairline px-4">
                  <div v-for="[key, value] in detailEntries" :key="key" class="grid gap-2 py-3 text-sm md:grid-cols-[170px_1fr]">
                    <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ humanizeKey(key) }}</div>
                    <pre class="tnum whitespace-pre-wrap break-words font-sans text-slate-600">{{ prettyValue(value) }}</pre>
                  </div>
                </div>
              </div>
            </div>
          </section>
        </div>
      </Transition>
    </Teleport>
  </main>
</template>
