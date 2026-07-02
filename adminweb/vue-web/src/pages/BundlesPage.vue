<script setup lang="ts">
import { FileJson, Loader2, Plus, RefreshCw, Save, Send, Trash2, X } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { badgeClass, pickFirst } from "@/lib/status"

type BundleForm = {
  bundle_ulid: string
  bundle_gpath: string
  name: string
  description: string
  items_json: string
  pricing_json: string
  thumbnail_object_key: string
  thumbnail_file_hash: string
}

type DetailTab = "summary" | "meta" | "pricing" | "schema" | "actions" | "raw"
type Mode = "detail" | "create"
type SummaryField = {
  label: string
  value: string
}
type DetailField = {
  key: string
  label: string
  value: unknown
}

const emptyForm: BundleForm = {
  bundle_ulid: "",
  bundle_gpath: "",
  name: "",
  description: "",
  items_json: "[]",
  pricing_json: "{}",
  thumbnail_object_key: "",
  thumbnail_file_hash: "",
}

const bundles = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const form = ref<BundleForm>({ ...emptyForm })
const loading = ref(false)
const saving = ref(false)
const publishing = ref(false)
const deprecating = ref(false)
const deleting = ref(false)
const detailOpen = ref(false)
const statusFilter = ref("")
const offset = ref(0)
const schemas = ref<JsonRecord | null>(null)
const activeTab = ref<DetailTab>("summary")
const mode = ref<Mode>("detail")
const showDeleteConfirm = ref(false)
const limit = 20
const { t } = useAdminLanguage()
const copy = computed(() => t.value.bundlesAdmin)

const canPrev = computed(() => offset.value > 0)
const canNext = computed(() => bundles.value.length >= limit)
const statusActionBusy = computed(() => publishing.value || deprecating.value || deleting.value)
const selectedId = computed(() => selected.value ? bundleUlid(selected.value) : "")
const detailTabs = computed(() => [
  { key: "summary" as const, title: copy.value.tabs.summary, count: selected.value ? 1 : 0 },
  { key: "meta" as const, title: copy.value.tabs.meta, count: 1 },
  { key: "pricing" as const, title: copy.value.tabs.pricing, count: 2 },
  { key: "schema" as const, title: "Schema", count: schemas.value ? 1 : 0 },
  { key: "actions" as const, title: copy.value.tabs.actions, count: 3 },
  { key: "raw" as const, title: copy.value.tabs.raw, count: 1 },
])
const summaryFields = computed<SummaryField[]>(() => {
  const bundle = selected.value
  if (!bundle) return []
  return [
    { label: copy.value.summary.displayPrice, value: displayPrice(bundle) },
    { label: copy.value.summary.version, value: String(bundle.version ?? "-") },
  ]
})
const selectedFields = computed<DetailField[]>(() => {
  if (!selected.value) return []
  return Object.entries(selected.value)
    .filter(([key]) => key !== "version")
    .map(([key, value]) => ({
      key,
      label: copy.value.fieldLabels[key as keyof typeof copy.value.fieldLabels] || key.replaceAll("_", " "),
      value,
    }))
})

function bundleUlid(bundle: JsonRecord | null | undefined) {
  return String(pickFirst(bundle || {}, ["bundle_ulid", "bundle_id"]) || "")
}

function bundleName(bundle: JsonRecord | null | undefined) {
  return String(pickFirst(bundle || {}, ["name", "title"]) || copy.value.unnamed)
}

function bundleStatus(bundle: JsonRecord | null | undefined) {
  return pickFirst(bundle || {}, ["status", "raw_status"])
}

function displayPrice(bundle: JsonRecord | null | undefined) {
  const currency = String(bundle?.display_currency || "")
  const min = Number(bundle?.display_amount_min || 0) / 100
  const max = Number(bundle?.display_amount_max || 0) / 100
  if (!currency || (!min && !max)) return "-"
  if (min === max) return `${currency} ${min.toFixed(2)}`
  return `${currency} ${min.toFixed(2)} - ${max.toFixed(2)}`
}

function parseJson(value: string, field: string) {
  try {
    return JSON.parse(value || "")
  } catch {
    toast.error(copy.value.jsonInvalid(field))
    return null
  }
}

function formFromBundle(bundle: JsonRecord | null): BundleForm {
  if (!bundle) return { ...emptyForm }
  return {
    bundle_ulid: String(bundle.bundle_ulid || bundle.bundle_id || ""),
    bundle_gpath: String(bundle.bundle_gpath || ""),
    name: String(bundle.name || ""),
    description: String(bundle.description || ""),
    items_json: String(bundle.items_json || "[]"),
    pricing_json: String(bundle.pricing_json || "{}"),
    thumbnail_object_key: String(bundle.thumbnail_object_key || ""),
    thumbnail_file_hash: String(bundle.thumbnail_file_hash || ""),
  }
}

async function load() {
  loading.value = true
  try {
    const params = new URLSearchParams({
      limit: String(limit),
      offset: String(offset.value),
    })
    if (statusFilter.value) params.set("status", statusFilter.value)
    const data = await apiClient<JsonRecord>(`/api/mall/bundles?${params}`)
    const list = Array.isArray(data.bundles) ? data.bundles : []
    bundles.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    if (!selected.value && bundles.value.length) {
      await selectBundle(bundles.value[0], false)
    }
  } catch (err) {
    console.error(err)
    bundles.value = []
    toast.error(copy.value.toasts.loadFailed)
  } finally {
    loading.value = false
  }
}

async function selectBundle(bundle: JsonRecord, open = true) {
  const id = bundleUlid(bundle)
  selected.value = bundle
  detailOpen.value = open
  mode.value = "detail"
  activeTab.value = "summary"
  showDeleteConfirm.value = false
  form.value = formFromBundle(bundle)
  if (!id) return
  try {
    const detail = await apiClient<JsonRecord>(`/api/mall/bundles/${encodeURIComponent(id)}`)
    const actualBundle = (detail.bundle && typeof detail.bundle === "object" ? detail.bundle : detail) as JsonRecord
    selected.value = actualBundle
    form.value = formFromBundle(actualBundle)
  } catch {
    form.value = formFromBundle(bundle)
  }
}

function newBundle() {
  selected.value = null
  detailOpen.value = true
  mode.value = "create"
  activeTab.value = "meta"
  showDeleteConfirm.value = false
  form.value = { ...emptyForm }
}

function closeDetail() {
  detailOpen.value = false
  if (mode.value === "create") mode.value = "detail"
}

async function createBundle() {
  if (!form.value.bundle_ulid.trim() || !form.value.bundle_gpath.trim() || !form.value.name.trim()) {
    toast.error(copy.value.toasts.createRequired)
    return
  }
  if (parseJson(form.value.items_json, "items_json") === null || parseJson(form.value.pricing_json, "pricing_json") === null) {
    return
  }

  saving.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/mall/bundles", {
      method: "POST",
      body: JSON.stringify({
        bundle_ulid: form.value.bundle_ulid.trim(),
        bundle_gpath: form.value.bundle_gpath.trim(),
        name: form.value.name.trim(),
        description: form.value.description.trim(),
        items_json: form.value.items_json.trim(),
        pricing_json: form.value.pricing_json.trim(),
        thumbnail_object_key: form.value.thumbnail_object_key.trim(),
        thumbnail_file_hash: form.value.thumbnail_file_hash.trim(),
      }),
    })
    toast.success(copy.value.toasts.created)
    await load()
    const id = String(data.bundle_ulid || form.value.bundle_ulid)
    const created = bundles.value.find((item) => bundleUlid(item) === id)
    if (created) await selectBundle(created)
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.createFailed)
  } finally {
    saving.value = false
  }
}

async function saveMeta() {
  if (!selectedId.value) return
  saving.value = true
  try {
    await apiClient(`/api/mall/bundles/${encodeURIComponent(selectedId.value)}/meta`, {
      method: "PUT",
      body: JSON.stringify({
        name: form.value.name.trim(),
        description: form.value.description.trim(),
        thumbnail_object_key: form.value.thumbnail_object_key.trim(),
        thumbnail_file_hash: form.value.thumbnail_file_hash.trim(),
      }),
    })
    toast.success(copy.value.toasts.metaSaved)
    await load()
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.saveFailed)
  } finally {
    saving.value = false
  }
}

async function savePricing() {
  if (!selectedId.value) return
  if (parseJson(form.value.items_json, "items_json") === null || parseJson(form.value.pricing_json, "pricing_json") === null) {
    return
  }
  saving.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/mall/bundles/pricing", {
      method: "PUT",
      body: JSON.stringify({
        bundle_ulid: selectedId.value,
        items_json: form.value.items_json.trim(),
        pricing_json: form.value.pricing_json.trim(),
      }),
    })
    toast.success(copy.value.toasts.pricingSaved)
    const actualBundle = (data.bundle && typeof data.bundle === "object" ? data.bundle : data) as JsonRecord
    selected.value = actualBundle
    form.value = formFromBundle(actualBundle)
    await load()
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.saveFailed)
  } finally {
    saving.value = false
  }
}

async function publish() {
  if (!selectedId.value) return
  if (statusActionBusy.value) return
  publishing.value = true
  try {
    await apiClient(`/api/mall/bundles/${encodeURIComponent(selectedId.value)}/publish`, { method: "POST" })
    toast.success(copy.value.toasts.published)
    await load()
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.publishFailed)
  } finally {
    publishing.value = false
  }
}

async function deprecate() {
  if (!selectedId.value) return
  if (statusActionBusy.value) return
  deprecating.value = true
  try {
    await apiClient(`/api/mall/bundles/${encodeURIComponent(selectedId.value)}/deprecate`, { method: "POST" })
    toast.success(copy.value.toasts.deprecated)
    await load()
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.deprecateFailed)
  } finally {
    deprecating.value = false
  }
}

async function removeBundle() {
  if (!selectedId.value) return
  if (statusActionBusy.value) return
  deleting.value = true
  try {
    await apiClient(`/api/mall/bundles/${encodeURIComponent(selectedId.value)}`, { method: "DELETE" })
    toast.success(copy.value.toasts.deleted)
    selected.value = null
    detailOpen.value = false
    form.value = { ...emptyForm }
    showDeleteConfirm.value = false
    await load()
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.deleteFailed)
  } finally {
    deleting.value = false
  }
}

async function syncDisplayPricing() {
  await apiClient("/api/mall/bundles/sync-display-pricing", {
    method: "POST",
    body: JSON.stringify({ bundle_ulid: selectedId.value || undefined }),
  })
  toast.success(copy.value.toasts.displayPricingSynced)
  await load()
}

async function loadSchemas() {
  schemas.value = await apiClient<JsonRecord>("/api/mall/bundles/schemas")
  activeTab.value = "schema"
}

watch([statusFilter, offset], () => {
  selected.value = null
  void load()
})
onMounted(load)
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1580px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">{{ copy.title }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-4 py-3 text-sm font-bold text-white shadow-sm" type="button" @click="newBundle">
          <Plus class="h-4 w-4" />
          {{ copy.newBundle }}
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="syncDisplayPricing">
          <RefreshCw class="h-4 w-4" />
          {{ copy.syncDisplayPricing }}
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load">
          <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
          {{ copy.refresh }}
        </button>
      </div>
    </header>

    <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 p-5">
          <div class="flex items-center gap-3">
            <div>
              <h2 class="text-xl font-black">{{ copy.listTitle }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
            </div>
            <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ bundles.length }}</span>
          </div>
          <select v-model="statusFilter" class="h-10 w-full rounded-xl border border-slate-200 px-4 text-sm md:w-64">
            <option value="">{{ copy.allStatus }}</option>
            <option value="Draft">Draft</option>
            <option value="Active">Active</option>
            <option value="Deprecated">Deprecated</option>
          </select>
        </div>
        <div v-if="loading" class="p-12 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          {{ copy.loading }}
        </div>
        <template v-else>
          <div class="grid grid-cols-[minmax(0,1fr)_160px_110px_170px_112px] gap-4 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-500">
            <span>{{ copy.columns.bundle }}</span>
            <span class="text-center">{{ copy.columns.status }}</span>
            <span class="text-center">{{ copy.columns.version }}</span>
            <span class="text-right">{{ copy.columns.updatedAt }}</span>
            <span class="text-right">{{ copy.columns.action }}</span>
          </div>
          <div
            v-for="bundle in bundles"
            :key="bundleUlid(bundle)"
            class="grid w-full cursor-pointer grid-cols-[minmax(0,1fr)_160px_110px_170px_112px] gap-4 border-b border-slate-100 px-5 py-4 text-left transition last:border-b-0 hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-200"
            :class="mode === 'detail' && selectedId === bundleUlid(bundle) ? 'bg-sky-50' : ''"
            role="button"
            tabindex="0"
            @click="selectBundle(bundle)"
            @keydown.enter.prevent="selectBundle(bundle)"
            @keydown.space.prevent="selectBundle(bundle)"
          >
            <div class="min-w-0">
              <div class="truncate text-lg font-black">{{ bundleName(bundle) }}</div>
              <div class="mt-1 line-clamp-1 text-sm text-slate-500">{{ bundle.description || copy.noDescription }}</div>
              <div class="mt-2 flex flex-wrap gap-2 text-xs font-semibold text-slate-500">
                <span class="rounded-full bg-slate-100 px-2 py-1">{{ copy.displayPricePrefix }}{{ displayPrice(bundle) }}</span>
                <span class="rounded-full bg-slate-100 px-2 py-1">ID: {{ bundleUlid(bundle) || "-" }}</span>
              </div>
            </div>
            <span class="self-center justify-self-center rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(bundleStatus(bundle))">{{ bundleStatus(bundle) || "-" }}</span>
            <span class="self-center text-center text-sm font-black text-slate-700">v{{ bundle.version || 0 }}</span>
            <span class="self-center justify-self-end text-sm font-semibold text-slate-500">{{ formatDate(String(bundle.updated_at || bundle.created_at || "")) }}</span>
            <button class="inline-flex h-9 items-center justify-self-end rounded-xl border border-slate-200 bg-white px-3 text-sm font-bold text-blue-700 shadow-sm transition hover:border-blue-200 hover:bg-blue-50" type="button" @click.stop="selectBundle(bundle)">
              {{ copy.viewDetails }}
            </button>
          </div>
        </template>
        <div v-if="!loading && !bundles.length" class="p-12 text-center text-slate-500">{{ copy.empty }}</div>
        <div class="flex justify-end gap-3 border-t border-slate-200 p-5">
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrev" @click="offset = Math.max(0, offset - limit)">{{ copy.prev }}</button>
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext" @click="offset += limit">{{ copy.next }}</button>
        </div>
    </section>

    <Teleport to="body">
      <div v-if="detailOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/45 p-6">
        <section class="flex max-h-[88vh] w-full max-w-[1320px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
        <template v-if="mode === 'create'">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 p-5">
            <div>
              <h2 class="text-2xl font-black">{{ copy.createTitle }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ copy.createDescription }}</p>
            </div>
            <button class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="closeDetail">
              <X class="h-5 w-5" />
            </button>
          </div>
          <div class="space-y-5 overflow-y-auto p-5">
            <div class="grid gap-4 md:grid-cols-2">
              <label class="grid gap-2 text-sm font-bold">
                Bundle ULID
                <input v-model="form.bundle_ulid" class="rounded-xl border border-slate-200 px-4 py-3" />
              </label>
              <label class="grid gap-2 text-sm font-bold">
                Bundle GPath
                <input v-model="form.bundle_gpath" class="rounded-xl border border-slate-200 px-4 py-3" />
              </label>
              <label class="grid gap-2 text-sm font-bold">
                {{ copy.fields.name }}
                <input v-model="form.name" class="rounded-xl border border-slate-200 px-4 py-3" maxlength="160" />
              </label>
              <label class="grid gap-2 text-sm font-bold">
                {{ copy.fields.thumbnailObjectKey }}
                <input v-model="form.thumbnail_object_key" class="rounded-xl border border-slate-200 px-4 py-3" />
              </label>
              <label class="grid gap-2 text-sm font-bold md:col-span-2">
                {{ copy.fields.description }}
                <textarea v-model="form.description" class="min-h-24 rounded-xl border border-slate-200 p-4" maxlength="1200" />
              </label>
              <label class="grid gap-2 text-sm font-bold md:col-span-2">
                Thumbnail File Hash
                <input v-model="form.thumbnail_file_hash" class="rounded-xl border border-slate-200 px-4 py-3" />
              </label>
            </div>
            <div class="grid gap-4 xl:grid-cols-2">
              <label class="grid gap-2 text-sm font-bold">
                items_json
                <textarea v-model="form.items_json" class="min-h-[260px] rounded-xl border border-slate-200 p-4 font-mono text-xs leading-6" />
              </label>
              <label class="grid gap-2 text-sm font-bold">
                pricing_json
                <textarea v-model="form.pricing_json" class="min-h-[260px] rounded-xl border border-slate-200 p-4 font-mono text-xs leading-6" />
              </label>
            </div>
            <div class="flex justify-end">
              <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving" @click="createBundle">
                <Plus class="h-4 w-4" />
                {{ copy.createDraft }}
              </button>
            </div>
          </div>
        </template>

        <div v-else-if="!selected" class="flex items-start justify-between gap-4 p-6">
          <div>
            <h2 class="text-2xl font-black">{{ copy.detailTitle }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.detailDescription }}</p>
          </div>
          <button class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="closeDetail">
            <X class="h-5 w-5" />
          </button>
        </div>

        <template v-else>
          <div class="border-b border-slate-200 p-5">
            <div class="flex flex-wrap items-start justify-between gap-4">
              <div>
                <h2 class="text-2xl font-black">{{ bundleName(selected) }}</h2>
                <p class="mt-1 break-all text-sm text-slate-500">{{ selectedId }}</p>
              </div>
              <div class="flex items-center gap-3">
                <span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(bundleStatus(selected))">{{ bundleStatus(selected) || "-" }}</span>
                <button class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="closeDetail">
                  <X class="h-5 w-5" />
                </button>
              </div>
            </div>
          </div>

          <div class="border-b border-slate-200 p-4">
            <div class="flex gap-2 overflow-x-auto">
              <button
                v-for="tab in detailTabs"
                :key="tab.key"
                class="inline-flex h-11 shrink-0 items-center gap-3 rounded-2xl border px-4 text-sm font-black transition"
                :class="activeTab === tab.key ? 'border-sky-200 bg-sky-50 text-slate-950' : 'border-slate-100 bg-white text-slate-700 hover:bg-slate-50'"
                type="button"
                @click="activeTab = tab.key"
              >
                <span>{{ tab.title }}</span>
                <span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-black text-slate-600">{{ tab.count }}</span>
              </button>
            </div>
          </div>

          <div class="min-h-0 flex-1 overflow-hidden">
            <main class="h-[60vh] min-h-[360px] max-h-[620px] min-w-0 overflow-y-auto p-5">
              <div v-if="activeTab === 'summary'" class="space-y-5">
                <div class="grid gap-4 md:grid-cols-2">
                  <div v-for="field in summaryFields" :key="field.label" class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                    <div class="text-xs font-black uppercase text-slate-400">{{ field.label }}</div>
                    <div class="mt-2 break-all text-sm font-black text-slate-800">{{ field.value }}</div>
                  </div>
                </div>
                <div class="rounded-2xl border border-slate-200 bg-white p-4">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.summary.description }}</div>
                  <p class="mt-2 whitespace-pre-wrap text-sm font-semibold leading-6 text-slate-700">{{ form.description || "-" }}</p>
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <label v-for="field in selectedFields" :key="field.key" class="grid gap-2 text-sm font-bold">
                    {{ field.label }}
                    <textarea
                      v-if="Array.isArray(field.value) || (field.value && typeof field.value === 'object')"
                      class="min-h-24 rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-600"
                      disabled
                      :value="JSON.stringify(field.value, null, 2)"
                    />
                    <input v-else class="rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-600" disabled :value="String(field.value ?? '-')" />
                  </label>
                </div>
              </div>

              <div v-else-if="activeTab === 'meta'" class="space-y-5">
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="grid gap-2 text-sm font-bold">
                    Bundle ULID
                    <input v-model="form.bundle_ulid" disabled class="rounded-xl border border-slate-200 bg-slate-100 px-4 py-3" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    Bundle GPath
                    <input v-model="form.bundle_gpath" disabled class="rounded-xl border border-slate-200 bg-slate-100 px-4 py-3" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    {{ copy.fields.name }}
                    <input v-model="form.name" class="rounded-xl border border-slate-200 px-4 py-3" maxlength="160" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    {{ copy.fields.thumbnailObjectKey }}
                    <input v-model="form.thumbnail_object_key" class="rounded-xl border border-slate-200 px-4 py-3" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold md:col-span-2">
                    {{ copy.fields.description }}
                    <textarea v-model="form.description" class="min-h-28 rounded-xl border border-slate-200 p-4" maxlength="1200" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold md:col-span-2">
                    Thumbnail File Hash
                    <input v-model="form.thumbnail_file_hash" class="rounded-xl border border-slate-200 px-4 py-3" />
                  </label>
                </div>
                <div class="flex justify-end">
                  <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving" @click="saveMeta">
                    <Save class="h-4 w-4" />
                    {{ copy.saveMeta }}
                  </button>
                </div>
              </div>

              <div v-else-if="activeTab === 'pricing'" class="space-y-5">
                <div class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
                  {{ copy.jsonValidateHint }}
                </div>
                <div class="grid gap-4 xl:grid-cols-2">
                  <label class="grid gap-2 text-sm font-bold">
                    items_json
                    <textarea v-model="form.items_json" class="min-h-[420px] rounded-xl border border-slate-200 p-4 font-mono text-xs leading-6" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    pricing_json
                    <textarea v-model="form.pricing_json" class="min-h-[420px] rounded-xl border border-slate-200 p-4 font-mono text-xs leading-6" />
                  </label>
                </div>
                <div class="flex justify-end">
                  <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving" @click="savePricing">
                    <Send class="h-4 w-4" />
                    {{ copy.savePricing }}
                  </button>
                </div>
              </div>

              <div v-else-if="activeTab === 'schema'" class="space-y-4">
                <button class="inline-flex items-center gap-2 rounded-xl border px-4 py-2 text-sm font-bold" type="button" @click="loadSchemas">
                  <FileJson class="h-4 w-4" />
                  {{ copy.loadSchema }}
                </button>
                <pre v-if="schemas" class="max-h-[620px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(schemas, null, 2) }}</pre>
                <div v-else class="rounded-2xl border border-dashed border-slate-200 p-10 text-center text-slate-500">{{ copy.emptySchema }}</div>
              </div>

              <div v-else-if="activeTab === 'actions'" class="space-y-5">
                <div class="rounded-2xl border border-slate-200 bg-slate-50 p-5">
                  <h3 class="font-black">{{ copy.actionsTitle }}</h3>
                  <p class="mt-1 text-sm text-slate-500">{{ copy.actionsDescription }}</p>
                  <div class="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                    <button class="inline-flex h-11 items-center justify-center gap-2 rounded-xl border bg-white px-4 text-sm font-bold shadow-sm transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60" type="button" :disabled="statusActionBusy" @click="publish">
                      <Loader2 v-if="publishing" class="h-4 w-4 animate-spin" />
                      {{ publishing ? copy.publishing : copy.publish }}
                    </button>
                    <button class="inline-flex h-11 items-center justify-center gap-2 rounded-xl border bg-white px-4 text-sm font-bold shadow-sm transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60" type="button" :disabled="statusActionBusy" @click="deprecate">
                      <Loader2 v-if="deprecating" class="h-4 w-4 animate-spin" />
                      {{ deprecating ? copy.deprecating : copy.deprecate }}
                    </button>
                    <button class="inline-flex h-11 items-center justify-center gap-2 rounded-xl bg-red-600 px-4 text-sm font-bold text-white shadow-sm shadow-red-200 transition hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-60" type="button" :disabled="statusActionBusy" @click="showDeleteConfirm = true">
                      <Loader2 v-if="deleting" class="h-4 w-4 animate-spin" />
                      <Trash2 v-else class="h-4 w-4" />
                      {{ deleting ? copy.deleting : copy.delete }}
                    </button>
                  </div>
                </div>
              </div>

              <div v-else-if="activeTab === 'raw'" class="space-y-4">
                <div class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
                  {{ copy.rawReadonlyHint }}
                </div>
                <pre class="max-h-[620px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selected, null, 2) }}</pre>
              </div>
            </main>
          </div>
        </template>
        </section>
      </div>
    </Teleport>

    <div v-if="showDeleteConfirm" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-6">
      <div class="w-full max-w-md rounded-3xl bg-white p-6 shadow-2xl">
        <h2 class="text-2xl font-black">{{ copy.deleteConfirmTitle }}</h2>
        <p class="mt-3 text-sm text-slate-600">{{ copy.deleteConfirmDescription }}</p>
        <div class="mt-5 rounded-2xl bg-slate-50 p-4">
          <div class="font-black">{{ bundleName(selected) }}</div>
          <div class="mt-1 break-all text-xs text-slate-500">{{ selectedId }}</div>
        </div>
        <div class="mt-6 flex justify-end gap-3">
          <button class="rounded-xl border px-5 py-3 font-bold disabled:cursor-not-allowed disabled:opacity-60" type="button" :disabled="deleting" @click="showDeleteConfirm = false">{{ copy.cancel }}</button>
          <button class="inline-flex items-center justify-center gap-2 rounded-xl bg-red-600 px-5 py-3 font-bold text-white disabled:cursor-not-allowed disabled:opacity-60" type="button" :disabled="deleting" @click="removeBundle">
            <Loader2 v-if="deleting" class="h-4 w-4 animate-spin" />
            {{ deleting ? copy.deleting : copy.confirmDelete }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
