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
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <p class="text-sm font-black uppercase tracking-[0.2em] text-sky-600">{{ copy.eyebrow }}</p>
        <h1 class="mt-2 text-4xl font-black tracking-tight text-slate-950">{{ copy.title }}</h1>
        <p class="mt-2 max-w-3xl text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" :disabled="loading" @click="load">
        <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
        <RefreshCw v-else class="h-4 w-4" />
        {{ copy.refresh }}
      </button>
    </header>

    <section class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
      <div class="grid gap-3 md:grid-cols-4">
        <label class="text-sm font-bold text-slate-700">
          {{ copy.filters.keyword }}
          <input v-model="filters.keyword" class="mt-2 h-11 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.placeholders.keyword" @keydown.enter="resetAndLoad" />
        </label>
        <label class="text-sm font-bold text-slate-700">
          {{ copy.filters.sourceService }}
          <input v-model="filters.source_service" class="mt-2 h-11 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.placeholders.sourceService" @keydown.enter="resetAndLoad" />
        </label>
        <label class="text-sm font-bold text-slate-700">
          {{ copy.filters.action }}
          <input v-model="filters.action" class="mt-2 h-11 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.placeholders.action" @keydown.enter="resetAndLoad" />
        </label>
        <label class="text-sm font-bold text-slate-700">
          {{ copy.filters.status }}
          <input v-model="filters.status" class="mt-2 h-11 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.placeholders.status" @keydown.enter="resetAndLoad" />
        </label>
        <label class="text-sm font-bold text-slate-700">
          {{ copy.filters.operator }}
          <input v-model="filters.operator_id" class="mt-2 h-11 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.placeholders.operator" @keydown.enter="resetAndLoad" />
        </label>
        <label class="text-sm font-bold text-slate-700">
          {{ copy.filters.resourceType }}
          <input v-model="filters.resource_type" class="mt-2 h-11 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.placeholders.resourceType" @keydown.enter="resetAndLoad" />
        </label>
        <label class="text-sm font-bold text-slate-700">
          {{ copy.filters.resourceId }}
          <input v-model="filters.resource_id" class="mt-2 h-11 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.placeholders.resourceId" @keydown.enter="resetAndLoad" />
        </label>
        <div class="flex items-end">
          <button class="inline-flex h-11 w-full items-center justify-center gap-2 rounded-xl bg-slate-950 px-4 text-sm font-black text-white" type="button" @click="resetAndLoad">
            <Search class="h-4 w-4" />
            {{ copy.search }}
          </button>
        </div>
        <label class="text-sm font-bold text-slate-700 md:col-span-2">
          {{ copy.filters.startTime }}
          <input v-model="filters.start_time" class="mt-2 h-11 w-full rounded-xl border border-slate-200 px-3" type="datetime-local" />
        </label>
        <label class="text-sm font-bold text-slate-700 md:col-span-2">
          {{ copy.filters.endTime }}
          <input v-model="filters.end_time" class="mt-2 h-11 w-full rounded-xl border border-slate-200 px-3" type="datetime-local" />
        </label>
      </div>
    </section>

    <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
      <div class="flex items-center justify-between border-b border-slate-200 p-5">
        <div>
          <h2 class="text-xl font-black">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
        <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-bold text-slate-500">{{ copy.totalText(total) }}</span>
      </div>

      <div v-if="loading" class="p-12 text-center text-slate-500">
        <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
        {{ copy.loading }}
      </div>
      <div v-else-if="!logs.length" class="p-12 text-center text-slate-500">{{ copy.empty }}</div>
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
          class="grid w-full gap-3 border-b border-slate-100 px-5 py-4 text-left transition last:border-b-0 hover:bg-slate-50 lg:grid-cols-[180px_130px_minmax(0,1fr)_180px_120px] lg:items-center lg:gap-4"
          :class="auditId(item) === auditId(selected) ? 'bg-sky-50/70' : ''"
          type="button"
          @click="openDetail(item)"
        >
          <span class="text-sm font-semibold text-slate-600">{{ formatDate(fieldText(item, ["created_at"])) }}</span>
          <span class="font-mono text-xs font-black uppercase text-sky-700">{{ fieldText(item, ["source_service"]) }}</span>
          <span class="min-w-0">
            <span class="block truncate text-base font-black text-slate-950">{{ fieldText(item, ["summary_text", "action"]) }}</span>
            <span class="mt-1 block truncate text-xs font-semibold text-slate-500">{{ fieldText(item, ["resource_display_name", "resource_id", "audit_ulid"]) }}</span>
          </span>
          <span class="truncate text-sm font-bold text-slate-700">{{ fieldText(item, ["operator_name", "operator_id"]) }}</span>
          <span class="justify-self-start rounded-full border border-slate-200 bg-slate-50 px-3 py-1 text-xs font-black uppercase text-slate-600 lg:justify-self-center">
            {{ fieldText(item, ["status"]) }}
          </span>
        </button>
      </div>

      <div class="flex items-center justify-end gap-3 border-t border-slate-200 p-5">
        <span class="mr-auto text-sm font-bold text-slate-500">{{ copy.pageText(page, totalPages) }}</span>
        <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrevious || loading" @click="previousPage">{{ copy.prev }}</button>
        <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext || loading" @click="nextPage">{{ copy.next }}</button>
      </div>
    </section>

    <section v-if="detailOpen" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-6">
      <div class="flex max-h-[88vh] w-full max-w-[980px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
        <div class="flex items-start justify-between gap-4 border-b border-slate-200 p-5">
          <div>
            <h2 class="text-xl font-black">{{ copy.detailTitle }}</h2>
            <p class="mt-1 break-all text-sm text-slate-500">{{ auditId(selected) }}</p>
          </div>
          <button class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="detailOpen = false">
            <X class="h-5 w-5" />
          </button>
        </div>

        <div v-if="detailLoading" class="p-12 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          {{ copy.detailLoading }}
        </div>
        <div v-else class="flex-1 space-y-5 overflow-y-auto p-5">
          <div class="grid gap-4 md:grid-cols-3">
            <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
              <div class="text-xs font-black uppercase text-slate-400">{{ copy.summaryCards.service }}</div>
              <div class="mt-2 break-words text-lg font-black text-slate-950">{{ fieldText(summary(detail), ["source_service"]) }}</div>
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
