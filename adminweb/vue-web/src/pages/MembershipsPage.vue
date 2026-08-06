<script setup lang="ts">
import { Loader2, RefreshCw, Settings, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"

import TranslationsEditor from "@/components/TranslationsEditor.vue"
import { apiClient } from "@/lib/apiClient"
import { fetchAllCursorRecords } from "@/lib/cursorPagination"
import type { JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { pickFirst } from "@/lib/status"

type MapEntry = {
  key: string
  label: string
}

const memberships = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const loading = ref(false)
const detailLoading = ref(false)
const dialogOpen = ref(false)
const { t } = useAdminLanguage()
const copy = computed(() => t.value.membershipsAdmin)

const featureEntries = computed(() => collectFeatureEntries(selected.value?.features_json))
const translationFields = computed(() => [
  { key: "name", label: copy.value.fields.name, maxLength: 512 },
  { key: "description", label: copy.value.fields.description, kind: "textarea" as const, maxLength: 2048 },
  { key: "ideal_for", label: copy.value.fields.idealFor, kind: "textarea" as const, maxLength: 2048 },
  {
    key: "feature_categories",
    label: copy.value.fields.featureCategories,
    kind: "map" as const,
    mapEntries: featureEntries.value.categories,
  },
  {
    key: "feature_labels",
    label: copy.value.fields.featureLabels,
    kind: "map" as const,
    mapEntries: featureEntries.value.labels,
  },
])

function membershipId(membership: JsonRecord | null | undefined) {
  return String(pickFirst(membership || {}, ["membership_ulid", "membership_id"]) || "")
}

function membershipName(membership: JsonRecord | null | undefined) {
  return String(pickFirst(membership || {}, ["name", "title"]) || "-")
}

function normalizeTranslationKey(value: string) {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "")
}

function collectFeatureEntries(raw: unknown) {
  const categoryMap = new Map<string, MapEntry>()
  const labelMap = new Map<string, MapEntry>()
  let parsed: unknown = raw

  if (typeof raw === "string" && raw.trim()) {
    try {
      parsed = JSON.parse(raw)
    } catch {
      parsed = null
    }
  }

  const addEntry = (target: Map<string, MapEntry>, value: unknown) => {
    const label = String(value || "").trim()
    const key = normalizeTranslationKey(label)
    if (key && !target.has(key)) target.set(key, { key, label })
  }

  const visit = (value: unknown) => {
    if (Array.isArray(value)) {
      value.forEach(visit)
      return
    }
    if (!value || typeof value !== "object") return
    const record = value as JsonRecord
    addEntry(categoryMap, record.category)
    addEntry(labelMap, record.label)
    Object.values(record).forEach(visit)
  }

  visit(parsed)
  return {
    categories: Array.from(categoryMap.values()),
    labels: Array.from(labelMap.values()),
  }
}

async function load() {
  loading.value = true
  try {
    memberships.value = await fetchAllCursorRecords("/api/memberships/configs", "memberships")
  } catch (err) {
    console.error(err)
    memberships.value = []
    toast.error(copy.value.toasts.loadFailed)
  } finally {
    loading.value = false
  }
}

async function openMembership(membership: JsonRecord) {
  selected.value = membership
  dialogOpen.value = true
  const id = membershipId(membership)
  if (!id) return
  detailLoading.value = true
  try {
    const detail = await apiClient<JsonRecord>(`/api/memberships/${encodeURIComponent(id)}`)
    selected.value = { ...membership, ...detail }
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.detailLoadFailed)
  } finally {
    detailLoading.value = false
  }
}

function closeDialog() {
  if (detailLoading.value) return
  dialogOpen.value = false
  selected.value = null
}

onMounted(load)
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-5 px-4 py-5 md:gap-6 md:px-8 md:py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div class="min-w-0">
        <h1 class="text-3xl font-black tracking-tight md:text-4xl">{{ copy.title }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm disabled:opacity-50" type="button" :disabled="loading" @click="load">
        <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
        {{ copy.refresh }}
      </button>
    </header>

    <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm md:rounded-3xl">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 p-4 md:p-5">
        <div>
          <h2 class="text-xl font-black">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
        <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ copy.total(memberships.length) }}</span>
      </div>
      <div v-if="loading" class="flex items-center justify-center gap-2 p-12 text-sm font-bold text-slate-500">
        <Loader2 class="h-6 w-6 animate-spin" />
        {{ copy.loading }}
      </div>
      <div v-else-if="!memberships.length" class="p-12 text-center text-slate-500">{{ copy.empty }}</div>
      <div v-else class="divide-y divide-slate-100">
        <article v-for="membership in memberships" :key="membershipId(membership)" class="grid gap-4 p-4 transition hover:bg-slate-50 md:grid-cols-[minmax(0,1fr)_140px_110px_160px] md:items-center md:px-5">
          <div class="min-w-0">
            <div class="truncate text-lg font-black text-slate-950">{{ membershipName(membership) }}</div>
            <p class="mt-1 line-clamp-2 text-sm text-slate-500">{{ membership.description || copy.noDescription }}</p>
            <div class="mt-2 break-all font-mono text-xs font-bold text-blue-700">{{ membershipId(membership) }}</div>
          </div>
          <div class="text-sm font-bold text-slate-600">{{ membership.status || "-" }}</div>
          <div class="text-sm font-bold text-slate-600">v{{ membership.version || 0 }}</div>
          <button class="inline-flex h-10 items-center justify-center gap-2 rounded-xl border border-blue-200 bg-blue-50 px-4 text-sm font-bold text-blue-700" type="button" @click="openMembership(membership)">
            <Settings class="h-4 w-4" />
            {{ copy.viewDetails }}
          </button>
        </article>
      </div>
    </section>

    <Teleport to="body">
      <section v-if="dialogOpen && selected" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-0 md:p-6">
        <div v-modal-dialog="closeDialog" class="flex h-full w-full max-w-[1120px] flex-col overflow-hidden bg-white shadow-2xl md:h-auto md:max-h-[90vh] md:rounded-3xl">
          <header class="flex items-start justify-between gap-4 border-b border-slate-200 p-4 md:p-5">
            <div class="min-w-0">
              <h2 class="truncate text-xl font-black md:text-2xl">{{ membershipName(selected) }}</h2>
              <p class="mt-1 break-all font-mono text-xs font-bold text-blue-700">{{ membershipId(selected) }}</p>
            </div>
            <button class="rounded-full border border-slate-200 p-2 text-slate-500" type="button" :aria-label="copy.close" :disabled="detailLoading" @click="closeDialog">
              <X class="h-5 w-5" />
            </button>
          </header>

          <main class="min-h-0 flex-1 overflow-y-auto p-4 md:p-5">
            <div v-if="detailLoading" class="flex items-center justify-center gap-2 p-12 text-sm font-bold text-slate-500">
              <Loader2 class="h-6 w-6 animate-spin" />
              {{ copy.loading }}
            </div>
            <div v-else class="space-y-6">
              <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.fields.membershipGpath }}</div>
                  <div class="mt-2 break-all text-sm font-bold">{{ selected.membership_gpath || "-" }}</div>
                </div>
                <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.fields.duration }}</div>
                  <div class="mt-2 text-sm font-bold">{{ copy.fields.durationValue(Number(selected.duration_in_months || 0)) }}</div>
                </div>
                <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.fields.tierLevel }}</div>
                  <div class="mt-2 text-sm font-bold">{{ selected.tier_level ?? "-" }}</div>
                </div>
                <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4 md:col-span-2 xl:col-span-3">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.fields.idealFor }}</div>
                  <div class="mt-2 whitespace-pre-wrap text-sm font-bold">{{ selected.ideal_for || "-" }}</div>
                </div>
                <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4 md:col-span-2 xl:col-span-3">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.fields.featuresJson }}</div>
                  <pre class="mt-2 max-h-64 overflow-auto whitespace-pre-wrap break-words font-mono text-xs leading-5 text-slate-700">{{ selected.features_json || "{}" }}</pre>
                </div>
              </div>

              <TranslationsEditor
                :endpoint="`/api/memberships/${encodeURIComponent(membershipId(selected))}/translations`"
                :target-id="membershipId(selected)"
                :fields="translationFields"
              />
            </div>
          </main>
        </div>
      </section>
    </Teleport>
  </section>
</template>
