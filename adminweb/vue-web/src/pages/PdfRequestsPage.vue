<script setup lang="ts">
import { ChevronLeft, ChevronRight, FileBadge, Loader2, RefreshCw, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { formatDate, getDisplayTitle, humanizeKey, isPrimitive, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { pickFirst } from "@/lib/status"

const PAGE_SIZE = 10

const requests = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const loading = ref(false)
const detailOpen = ref(false)
const page = ref(1)
const total = ref(0)
const hasMore = ref(false)
const nextCursor = ref("")
const prevCursor = ref("")
const lastPage = ref(1)
const { t } = useAdminLanguage()
const copy = computed(() => t.value.pdfRequests)


const detailFields = computed(() => {
  if (!selected.value) return []
  return Object.entries(selected.value)
    .filter(([, value]) => isPrimitive(value))
    .map(([key, value]) => ({
      key,
      label: fieldLabel(key),
      value: formatFieldValue(key, value),
    }))
})

function fieldLabel(key: string) {
  return copy.value.fieldLabels[key as keyof typeof copy.value.fieldLabels] || humanizeKey(key)
}

function requestUlid(request: JsonRecord) {
  return String(pickFirst(request, ["request_ulid", "request_id", "id", "pdf_request_ulid"]) || "")
}

function statusLabel(value: unknown) {
  const status = Number(value)
  if (status === 1) return copy.value.statuses.pending
  if (status === 2) return copy.value.statuses.generating
  if (status === 3) return copy.value.statuses.success
  if (status === 4) return copy.value.statuses.failed
  return String(value || copy.value.statuses.unknown)
}

function statusBadgeClass(value: unknown) {
  const status = Number(value)
  if (status === 1) return "bg-amber-50 text-amber-700"
  if (status === 2) return "bg-sky-50 text-sky-700"
  if (status === 3) return "bg-emerald-50 text-emerald-700"
  if (status === 4) return "bg-red-50 text-red-700"
  return "bg-slate-100 text-slate-600"
}

function formatFieldValue(key: string, value: unknown) {
  if (key === "status") return statusLabel(value)
  if (key.endsWith("_at") || key.endsWith("_date")) return formatDate(value)
  if (value === null || value === undefined || value === "") return "-"
  return String(value)
}

async function openRequest(request: JsonRecord | null, open = true) {
  selected.value = request
  detailOpen.value = !!request && open
  const id = request ? requestUlid(request) : ""
  if (!id) return
  try {
    const detail = await apiClient<JsonRecord>(`/api/pdf-requests/${encodeURIComponent(id)}/detail`)
    selected.value = { ...request, ...detail }
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.loadFailed))
  }
}

function closeDetail() {
  detailOpen.value = false
}

async function load(nextPage = page.value) {
  loading.value = true
  try {
    const params = new URLSearchParams({ page_size: String(PAGE_SIZE) })

    let cursor = ""

    if (nextPage > lastPage.value) {

      cursor = nextCursor.value

    } else if (nextPage < lastPage.value) {

      cursor = prevCursor.value


    }

    

    if (cursor) params.set("cursor", cursor)


    const data = await apiClient<JsonRecord>(`/api/pdf-requests?${params}`)
    const list = Array.isArray(data.requests) ? data.requests : []
    const selectedId = selected.value ? requestUlid(selected.value) : ""

    page.value = nextPage
    total.value = Number(data.total || list.length)
    const isBackward = page.value < lastPage.value
    hasMore.value = isBackward ? true : Boolean(data.has_more)
    lastPage.value = page.value
nextCursor.value = String(data.next_cursor || "")
    prevCursor.value = String(data?.prev_cursor || "")

    lastPage.value = nextPage

    requests.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    selected.value = requests.value.find((item) => requestUlid(item) === selectedId) || requests.value[0] || null
    if (!selected.value) detailOpen.value = false
  } catch (err) {
    console.error(err)
    requests.value = []
    selected.value = null
    detailOpen.value = false
    hasMore.value = false
    nextCursor.value = ""
    toast.error(apiErrorMessage(err, copy.value.toasts.loadFailed))
  } finally {
    loading.value = false
  }
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
      <button class="ui-btn ui-btn-secondary" type="button" @click="load()">
        <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
        <RefreshCw v-else class="h-4 w-4 text-slate-400" />
        {{ copy.refresh }}
      </button>
    </header>

    <section class="ui-card mt-6 overflow-hidden">
      <div class="flex flex-wrap items-center justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
        <div class="min-w-0">
          <h2 class="text-base font-semibold text-ink">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
        <span class="tnum ui-badge shrink-0 bg-slate-100 text-slate-600">{{ copy.totalText(total) }}</span>
      </div>
      <div class="ui-table-head flex flex-wrap gap-x-4 gap-y-2 px-4 py-3 md:grid md:grid-cols-[minmax(0,1fr)_160px_180px_112px] md:gap-5 md:px-5">
        <span>{{ copy.columns.request }}</span>
        <span class="md:text-center">{{ copy.columns.status }}</span>
        <span class="md:text-right">{{ copy.columns.createdAt }}</span>
        <span class="md:text-right">{{ copy.columns.action }}</span>
      </div>

      <div v-if="loading" class="divide-y divide-hairline">
        <div v-for="i in 5" :key="`sk-${i}`" class="grid gap-3 px-4 py-3.5 md:grid-cols-[minmax(0,1fr)_160px_180px_112px] md:items-center md:gap-5 md:px-5">
          <div class="min-w-0">
            <div class="skeleton h-4 w-2/5" />
            <div class="skeleton mt-2 h-3 w-3/5" />
          </div>
          <div class="skeleton h-5 w-16 rounded-full md:justify-self-center" />
          <div class="skeleton h-4 w-24 md:justify-self-end" />
          <div class="skeleton h-4 w-14 md:justify-self-end" />
        </div>
      </div>
      <div v-else-if="!requests.length" class="ui-empty">
        <div class="ui-empty-icon">
          <FileBadge class="h-6 w-6" />
        </div>
        <p class="ui-empty-title">{{ copy.empty }}</p>
      </div>
      <div v-else class="divide-y divide-hairline">
        <div
          v-for="request in requests"
          :key="requestUlid(request)"
          class="ui-table-row grid gap-3 px-4 py-3.5 md:grid-cols-[minmax(0,1fr)_160px_180px_112px] md:items-center md:gap-5 md:px-5"
          :class="detailOpen && requestUlid(selected || {}) === requestUlid(request) ? 'bg-primary-50/60' : ''"
        >
          <div class="min-w-0">
            <div class="break-words text-sm font-medium text-ink md:truncate">{{ getDisplayTitle(request) }}</div>
            <div class="tnum mt-0.5 break-all text-xs text-slate-400 md:truncate">{{ requestUlid(request) || "-" }}</div>
          </div>
          <div class="min-w-0 md:text-center">
            <span class="ui-badge max-w-full truncate" :class="statusBadgeClass(request.status)">
              {{ statusLabel(request.status) }}
            </span>
          </div>
          <div class="tnum text-sm text-slate-500 md:text-right">{{ formatDate(request.created_at) || copy.noCreatedAt }}</div>
          <div class="md:text-right">
            <button
              class="inline-flex w-full items-center justify-center rounded-[10px] border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-primary-700 shadow-sm transition-colors hover:bg-slate-50 md:w-auto md:border-0 md:bg-transparent md:p-0 md:shadow-none md:hover:bg-transparent md:hover:text-primary-600"
              type="button"
              @click.stop="openRequest(request)"
            >
              {{ copy.viewDetails }}
            </button>
          </div>
        </div>
      </div>

      <div class="flex flex-col items-stretch justify-between gap-3 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center md:px-6">
        <span class="tnum text-center text-sm text-slate-500 sm:text-left">{{ copy.pageText(page, Math.max(1, Math.ceil(total / PAGE_SIZE))) }}</span>
        <div class="flex gap-2">
          <button
            class="ui-btn ui-btn-secondary ui-btn-sm flex-1 sm:flex-none"
            type="button"
            :disabled="page <= 1 || loading"
            @click="goPage(page - 1)"
          >
            <ChevronLeft class="h-4 w-4 text-slate-400" />
            {{ copy.prev }}
          </button>
          <button
            class="ui-btn ui-btn-secondary ui-btn-sm flex-1 sm:flex-none"
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
                  <FileBadge class="h-5 w-5" />
                </span>
                <div class="min-w-0">
                  <h2 class="text-base font-semibold text-ink md:text-lg">{{ copy.detailTitle }}</h2>
                  <p class="tnum mt-1 break-all text-sm text-slate-500">{{ requestUlid(selected) || "-" }}</p>
                </div>
              </div>
              <button
                class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
                type="button"
                :aria-label="copy.close"
                @click="closeDetail"
              >
                <X class="h-5 w-5" />
              </button>
            </div>

            <div class="min-h-0 flex-1 space-y-5 overflow-y-auto p-4 md:p-6">
              <div class="rounded-2xl bg-primary-50 p-4 md:p-5">
                <div class="flex items-center gap-2 text-sm font-medium text-primary-700">
                  <FileBadge class="h-4 w-4" />
                  {{ copy.currentRequest }}
                </div>
                <div class="tnum mt-1 break-all text-base font-semibold text-ink">{{ requestUlid(selected) || "-" }}</div>
                <span class="ui-badge mt-3" :class="statusBadgeClass(selected.status)">
                  {{ statusLabel(selected.status) }}
                </span>
              </div>

              <div class="grid gap-4 md:grid-cols-2">
                <div v-for="field in detailFields" :key="field.key" class="rounded-2xl bg-slate-50 p-4">
                  <span class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ field.label }}</span>
                  <div class="tnum mt-1.5 break-words text-sm font-medium text-slate-700">{{ field.value }}</div>
                </div>
              </div>
            </div>
          </section>
        </div>
      </Transition>
    </Teleport>
  </main>
</template>
