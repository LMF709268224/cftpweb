<script setup lang="ts">
import { ArrowLeft, ArrowRight, FileSearch, Loader2, RefreshCw } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { useRoute } from "vue-router"
import { toast } from "vue-sonner"
import JsonPreview from "@/components/JsonPreview.vue"
import { apiClient } from "@/lib/apiClient"
import {
  formatDate,
  getDisplaySubtitle,
  getDisplayTitle,
  getListFields,
  getStatusTone,
  humanizeKey,
  type JsonRecord,
} from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import type { ResourceRouteMeta } from "@/router"

type PageData = JsonRecord | JsonRecord[]

const route = useRoute()
const loading = ref(false)
const page = ref(1)
const rawData = ref<PageData | null>(null)
const items = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const pageSize = 20
const { t } = useAdminLanguage()
const copy = computed(() => t.value.resourceAdmin)

const meta = computed(() => route.meta as ResourceRouteMeta)
const routeCopy = computed(() => copy.value.routes[meta.value.copyKey])

const selectedTitle = computed(() => (selected.value ? getDisplayTitle(selected.value, copy.value.fallbackTitle) : copy.value.selectRecord))
const selectedEntries = computed(() => Object.entries(selected.value || {}))
const hasNext = computed(() => items.value.length >= pageSize)
const hasPrevious = computed(() => page.value > 1)

function buildEndpoint() {
  const url = new URL(meta.value.endpoint, window.location.origin)
  if (meta.value.pagination === "page") {
    url.searchParams.set("page", String(page.value))
    url.searchParams.set("page_size", String(pageSize))
    url.searchParams.set("limit", String(pageSize))
    return `${url.pathname}${url.search}`
  }

  url.searchParams.set("limit", String(pageSize))
  url.searchParams.set("offset", String((page.value - 1) * pageSize))
  return `${url.pathname}${url.search}`
}

function normalizeItems(data: PageData | null) {
  if (!data) return []
  if (Array.isArray(data)) return data.filter(isRecord)

  for (const key of meta.value.itemKeys) {
    const value = data[key]
    if (Array.isArray(value)) {
      return value.filter(isRecord)
    }
  }

  for (const value of Object.values(data)) {
    if (Array.isArray(value) && value.every((item) => typeof item === "object")) {
      return value.filter(isRecord)
    }
  }

  return isRecord(data) ? [data] : []
}

function isRecord(value: unknown): value is JsonRecord {
  return !!value && typeof value === "object" && !Array.isArray(value)
}

function jsonText(value: unknown) {
  return JSON.stringify(value ?? {}, null, 2)
}

async function load() {
  loading.value = true
  try {
    const data = await apiClient<PageData>(buildEndpoint())
    rawData.value = data
    items.value = normalizeItems(data)
    selected.value = items.value[0] || null
  } catch (err) {
    console.error(err)
    items.value = []
    selected.value = null
    toast.error(copy.value.loadFailed)
  } finally {
    loading.value = false
  }
}

function previousPage() {
  if (!hasPrevious.value) return
  page.value -= 1
}

function nextPage() {
  if (!hasNext.value) return
  page.value += 1
}

const isEditing = ref(false)
const draftJson = ref("")

watch(selected, (val) => {
  isEditing.value = false
  draftJson.value = val ? JSON.stringify(val, null, 2) : ""
})

function startEdit() {
  if (!selected.value) return
  draftJson.value = JSON.stringify(selected.value, null, 2)
  isEditing.value = true
}

function cancelEdit() {
  isEditing.value = false
  if (selected.value) {
    draftJson.value = JSON.stringify(selected.value, null, 2)
  }
}

async function saveEdit() {
  if (!selected.value) return
  try {
    const parsed = JSON.parse(draftJson.value)
    
    let idValue = ""
    const possibleIdKeys = ["id", "ulid", "file_id", "pack_id", "course_id", "pipeline_cc_ulid", "bundle_id"]
    for (const k of Object.keys(parsed)) {
      if (possibleIdKeys.includes(k) || k.endsWith("_id") || k.endsWith("_ulid")) {
        idValue = parsed[k]
        break
      }
    }
    
    if (!idValue) {
      toast.error(copy.value.missingId)
      return
    }

    loading.value = true
    const endpoint = meta.value.endpoint
    await apiClient(`${endpoint}/${idValue}`, {
      method: "PUT",
      body: JSON.stringify(parsed)
    })
    
    toast.success(copy.value.saveSuccess)
    isEditing.value = false
    load()
  } catch (err: any) {
    console.error(err)
    toast.error(copy.value.saveFailed(err.message || String(err)))
  } finally {
    loading.value = false
  }
}

watch(
  () => route.path,
  () => {
    page.value = 1
    load()
  },
)

watch(page, () => load())

onMounted(load)
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight text-slate-950">{{ routeCopy.title }}</h1>
        <p class="mt-2 text-base text-slate-600">{{ routeCopy.subtitle }}</p>
      </div>
      <button
        class="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 py-3 text-sm font-bold text-slate-700 shadow-sm hover:bg-slate-50 disabled:opacity-60"
        type="button"
        :disabled="loading"
        @click="load"
      >
        <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
        <RefreshCw v-else class="h-4 w-4" />
        {{ copy.refresh }}
      </button>
    </header>

    <div class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
      <div class="flex items-center justify-between gap-4">
        <div>
          <h2 class="text-xl font-black">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
        <div class="rounded-full bg-slate-100 px-3 py-1 text-sm font-bold text-slate-600">
          {{ copy.pageSummary(page, items.length) }}
        </div>
      </div>

      <div v-if="loading" class="mt-6 flex min-h-[320px] items-center justify-center rounded-2xl border border-dashed border-slate-200 text-slate-500">
        <Loader2 class="mr-2 h-5 w-5 animate-spin" />
        {{ copy.loading }}
      </div>

      <div v-else-if="!items.length" class="mt-6 flex min-h-[320px] flex-col items-center justify-center rounded-2xl border border-dashed border-slate-200 text-slate-500">
        <FileSearch class="mb-3 h-10 w-10 text-slate-300" />
        {{ copy.empty }}
      </div>

      <div v-else class="mt-6 overflow-hidden rounded-2xl border border-slate-200">
        <button
          v-for="item in items"
          :key="JSON.stringify(item).slice(0, 120)"
          class="grid w-full grid-cols-[minmax(240px,1.2fr)_1.8fr_auto] items-center gap-5 border-b border-slate-200 bg-white px-5 py-4 text-left transition last:border-b-0 hover:bg-sky-50"
          :class="selected === item ? 'bg-sky-50' : ''"
          type="button"
          @click="selected = item"
        >
          <div class="min-w-0">
            <div class="truncate text-lg font-black text-slate-950">{{ getDisplayTitle(item, copy.fallbackTitle) }}</div>
            <div class="mt-1 truncate text-sm text-slate-500">{{ getDisplaySubtitle(item, copy.fallbackSubtitle) }}</div>
          </div>
          <div class="grid grid-cols-2 gap-3 xl:grid-cols-3">
            <div v-for="field in getListFields(item)" :key="field.key" class="min-w-0">
              <div class="text-[11px] font-bold uppercase tracking-wide text-slate-400">{{ field.label }}</div>
              <div class="truncate text-sm font-semibold text-slate-700">{{ field.value }}</div>
            </div>
          </div>
          <span
            v-if="item.status || item.raw_status || item.payment_status"
            class="rounded-full border px-3 py-1 text-xs font-black"
            :class="{
              'border-emerald-200 bg-emerald-50 text-emerald-700': getStatusTone(item.status || item.raw_status || item.payment_status) === 'success',
              'border-amber-200 bg-amber-50 text-amber-700': getStatusTone(item.status || item.raw_status || item.payment_status) === 'warning',
              'border-red-200 bg-red-50 text-red-700': getStatusTone(item.status || item.raw_status || item.payment_status) === 'danger',
              'border-slate-200 bg-slate-50 text-slate-600': getStatusTone(item.status || item.raw_status || item.payment_status) === 'neutral',
            }"
          >
            {{ item.status || item.raw_status || item.payment_status }}
          </span>
          <span v-else class="text-sm font-bold text-blue-700 transition hover:underline">{{ copy.viewDetails }}</span>
        </button>
      </div>

      <div class="mt-5 flex items-center justify-end gap-3">
        <button
          class="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 py-2 text-sm font-bold text-slate-600 disabled:cursor-not-allowed disabled:opacity-40"
          type="button"
          :disabled="!hasPrevious || loading"
          @click="previousPage"
        >
          <ArrowLeft class="h-4 w-4" />
          {{ copy.prev }}
        </button>
        <button
          class="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 py-2 text-sm font-bold text-slate-600 disabled:cursor-not-allowed disabled:opacity-40"
          type="button"
          :disabled="!hasNext || loading"
          @click="nextPage"
        >
          {{ copy.next }}
          <ArrowRight class="h-4 w-4" />
        </button>
      </div>
    </div>

    <div class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
      <div class="mb-5 flex items-center justify-between gap-4">
        <div>
          <h2 class="text-xl font-black">{{ selectedTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.detailDescription }}</p>
        </div>
      </div>

      <div v-if="!selected" class="rounded-2xl border border-dashed border-slate-200 p-12 text-center text-slate-500">
        {{ copy.selectFromList }}
      </div>

      <div v-else class="grid gap-5 xl:grid-cols-[1fr_1.1fr]">
        <div class="grid gap-3 md:grid-cols-2">
          <div v-for="[key, value] in selectedEntries" :key="key" class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
            <div class="mb-2 text-xs font-black uppercase tracking-wide text-slate-400">{{ humanizeKey(key) }}</div>
            <div v-if="typeof value === 'string' && key.endsWith('_at')" class="break-words text-sm font-semibold text-slate-800">
              {{ formatDate(value) }}
            </div>
            <div v-else-if="typeof value !== 'object' || value === null" class="break-words text-sm font-semibold text-slate-800">
              {{ value ?? "-" }}
            </div>
            <pre v-else class="max-h-44 overflow-auto whitespace-pre-wrap break-words rounded-xl bg-white p-3 font-mono text-xs leading-5 text-slate-700">{{ jsonText(value) }}</pre>
          </div>
        </div>
        <div class="flex flex-col gap-3">
          <div class="flex items-center justify-between">
            <h3 class="text-sm font-bold text-slate-700">{{ copy.rawJson }}</h3>
            <div v-if="!isEditing" class="flex gap-2">
              <button class="rounded-lg bg-slate-900 px-3 py-1.5 text-xs font-bold text-white hover:bg-slate-800" @click="startEdit">{{ copy.editRawJson }}</button>
            </div>
            <div v-else class="flex gap-2">
              <button class="rounded-lg border border-slate-300 px-3 py-1.5 text-xs font-bold text-slate-600 hover:bg-slate-50" @click="cancelEdit">{{ copy.cancel }}</button>
              <button class="rounded-lg bg-blue-700 px-3 py-1.5 text-xs font-bold text-white hover:bg-blue-800" @click="saveEdit">{{ copy.saveAndSubmit }}</button>
            </div>
          </div>
          <JsonPreview
            v-if="!isEditing"
            :title="copy.rawJson"
            :value="selected || {}"
            :copy-label="copy.copyJson"
            :copied-label="copy.copiedJson"
            :copied-message="copy.jsonCopied"
            :copy-error-message="copy.jsonCopyFailed"
            max-height="720px"
          />
          <textarea v-else v-model="draftJson" class="min-h-[500px] w-full rounded-2xl bg-slate-950 p-5 font-mono text-xs leading-6 text-slate-100 outline-none focus:ring-2 focus:ring-blue-500"></textarea>
        </div>
      </div>
    </div>
  </section>
</template>
