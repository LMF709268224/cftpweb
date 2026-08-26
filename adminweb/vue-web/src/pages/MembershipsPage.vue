<script setup lang="ts">
import { Archive, Loader2, Pencil, Plus, RefreshCw, Save, Settings, TriangleAlert, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"

import TranslationsEditor from "@/components/TranslationsEditor.vue"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { fetchAllCursorRecords } from "@/lib/cursorPagination"
import type { JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { pickFirst } from "@/lib/status"

type MapEntry = {
  key: string
  label: string
}

type MembershipFormMode = "create" | "edit"

type MembershipForm = {
  membership_ulid: string
  membership_gpath: string
  name: string
  description: string
  features_json: string
  ideal_for: string
  duration_in_months: number
  casdoor_role_name: string
  tier_level: number
  course_discount_coupon: string
}

const ULID_ALPHABET = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
const ULID_PATTERN = /^[0-7][0-9A-HJKMNP-TV-Z]{25}$/
const GPATH_PATTERN = /^\/[a-z0-9_-]+(?:\/[a-z0-9_-]+)*$/

const memberships = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const loading = ref(false)
let listRequestId = 0
const detailLoading = ref(false)
const dialogOpen = ref(false)
const editorOpen = ref(false)
const formMode = ref<MembershipFormMode>("create")
const saving = ref(false)
const deprecateConfirmOpen = ref(false)
const deprecating = ref(false)
const form = ref<MembershipForm>(emptyMembershipForm())
const { t } = useAdminLanguage()
const copy = computed(() => t.value.membershipsAdmin)

const featureEntries = computed(() => collectFeatureEntries(selected.value?.features_json))
const translationFields = computed(() => [
  { key: "name", label: copy.value.fields.name, maxLength: 512 },
  { key: "description", label: copy.value.fields.description, kind: "textarea" as const, maxLength: 2048 },
  { key: "ideal_for", label: copy.value.fields.idealFor, kind: "textarea" as const, maxLength: 2048 },
  {
    key: "feature_labels",
    label: copy.value.fields.featureLabels,
    kind: "map" as const,
    mapEntries: featureEntries.value.labels,
  },
  {
    key: "feature_categories",
    label: copy.value.fields.featureCategories,
    kind: "map" as const,
    mapEntries: featureEntries.value.categories,
  },
])
const translationDefaultValues = computed(() => ({
  feature_categories: Object.fromEntries(
    featureEntries.value.categories.map((entry) => [entry.key, entry.label]),
  ),
}))
const isSelectedDeprecated = computed(() => isDeprecated(selected.value))
const editorTitle = computed(() => (
  formMode.value === "create" ? copy.value.createTitle : copy.value.editTitle
))
const editorDescription = computed(() => (
  formMode.value === "create" ? copy.value.createDescription : copy.value.editDescription
))

function emptyMembershipForm(): MembershipForm {
  return {
    membership_ulid: "",
    membership_gpath: "",
    name: "",
    description: "",
    features_json: "[]",
    ideal_for: "",
    duration_in_months: 12,
    casdoor_role_name: "",
    tier_level: 1,
    course_discount_coupon: "",
  }
}

function membershipId(membership: JsonRecord | null | undefined) {
  return String(pickFirst(membership || {}, ["membership_ulid", "membership_id"]) || "")
}

function membershipName(membership: JsonRecord | null | undefined) {
  return String(pickFirst(membership || {}, ["name", "title"]) || "-")
}

function membershipGpath(membership: JsonRecord | null | undefined) {
  return String(membership?.membership_gpath || "")
}

function membershipTier(membership: JsonRecord | null | undefined) {
  return Number(membership?.tier_level || 0)
}

function isDeprecated(membership: JsonRecord | null | undefined) {
  return String(membership?.status || "").trim().toLowerCase() === "deprecated"
}

function encodeUlidTime(timestamp: number) {
  let value = timestamp
  let encoded = ""
  for (let index = 0; index < 10; index += 1) {
    encoded = ULID_ALPHABET[value % 32] + encoded
    value = Math.floor(value / 32)
  }
  return encoded
}

function encodeUlidRandom(bytes: Uint8Array) {
  let encoded = ""
  let buffer = 0
  let bitCount = 0
  for (const byte of bytes) {
    buffer = (buffer << 8) | byte
    bitCount += 8
    while (bitCount >= 5) {
      bitCount -= 5
      encoded += ULID_ALPHABET[(buffer >> bitCount) & 31]
      buffer &= (1 << bitCount) - 1
    }
  }
  if (bitCount > 0) encoded += ULID_ALPHABET[(buffer << (5 - bitCount)) & 31]
  return encoded.slice(0, 16).padEnd(16, "0")
}

function generateUlid() {
  const random = new Uint8Array(10)
  globalThis.crypto.getRandomValues(random)
  return `${encodeUlidTime(Date.now())}${encodeUlidRandom(random)}`
}

function expectedTierForGpath(gpath: string) {
  const normalized = gpath.trim()
  const maxTier = memberships.value.reduce((maximum, membership) => {
    if (membershipGpath(membership) !== normalized) return maximum
    return Math.max(maximum, membershipTier(membership))
  }, 0)
  return maxTier + 1
}

function syncCreateTier() {
  if (formMode.value !== "create") return
  form.value.tier_level = expectedTierForGpath(form.value.membership_gpath)
}

function formatFeaturesJson(value: unknown) {
  if (typeof value !== "string") return JSON.stringify(value ?? [], null, 2)
  const trimmed = value.trim()
  if (!trimmed) return "[]"
  try {
    return JSON.stringify(JSON.parse(trimmed), null, 2)
  } catch {
    return value
  }
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
  const requestId = ++listRequestId
  loading.value = true
  try {
    const nextMemberships = await fetchAllCursorRecords("/api/memberships/configs", "memberships")
    if (requestId !== listRequestId) return
    memberships.value = nextMemberships
  } catch (err) {
    if (requestId !== listRequestId) return
    console.error(err)
    memberships.value = []
    toast.error(copy.value.toasts.loadFailed)
  } finally {
    if (requestId === listRequestId) loading.value = false
  }
}

async function refreshSelected(id: string) {
  const summary = memberships.value.find((membership) => membershipId(membership) === id)
  if (!summary) return
  try {
    const detail = await apiClient<JsonRecord>(`/api/memberships/${encodeURIComponent(id)}`)
    selected.value = { ...summary, ...detail }
  } catch (err) {
    console.error(err)
    selected.value = summary
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
  if (detailLoading.value || editorOpen.value || deprecateConfirmOpen.value) return
  dialogOpen.value = false
  selected.value = null
}

function openCreateEditor() {
  formMode.value = "create"
  form.value = {
    ...emptyMembershipForm(),
    membership_ulid: generateUlid(),
  }
  editorOpen.value = true
}

function openEditEditor() {
  if (!selected.value) return
  formMode.value = "edit"
  form.value = {
    membership_ulid: membershipId(selected.value),
    membership_gpath: membershipGpath(selected.value),
    name: membershipName(selected.value) === "-" ? "" : membershipName(selected.value),
    description: String(selected.value.description || ""),
    features_json: formatFeaturesJson(selected.value.features_json),
    ideal_for: String(selected.value.ideal_for || ""),
    duration_in_months: Number(selected.value.duration_in_months || 0),
    casdoor_role_name: String(selected.value.casdoor_role_name || ""),
    tier_level: membershipTier(selected.value),
    course_discount_coupon: String(selected.value.course_discount_coupon || ""),
  }
  editorOpen.value = true
}

function closeEditor() {
  if (saving.value) return
  editorOpen.value = false
  form.value = emptyMembershipForm()
}

function validateForm() {
  const value = form.value
  if (!ULID_PATTERN.test(value.membership_ulid.trim())) {
    toast.error(copy.value.validation.membershipIdInvalid)
    return false
  }
  if (!GPATH_PATTERN.test(value.membership_gpath.trim())) {
    toast.error(copy.value.validation.membershipGpathInvalid)
    return false
  }
  if (!value.name.trim()) {
    toast.error(copy.value.validation.nameRequired)
    return false
  }
  if (Array.from(value.name.trim()).length > 255) {
    toast.error(copy.value.validation.nameTooLong)
    return false
  }
  if (!value.casdoor_role_name.trim()) {
    toast.error(copy.value.validation.casdoorRoleRequired)
    return false
  }
  if (Array.from(value.casdoor_role_name.trim()).length > 255) {
    toast.error(copy.value.validation.casdoorRoleTooLong)
    return false
  }
  if (!Number.isInteger(value.tier_level) || value.tier_level <= 0) {
    toast.error(copy.value.validation.tierInvalid)
    return false
  }
  if (!Number.isInteger(value.duration_in_months) || value.duration_in_months < 0) {
    toast.error(copy.value.validation.durationInvalid)
    return false
  }
  if (Array.from(value.course_discount_coupon.trim()).length > 255) {
    toast.error(copy.value.validation.couponTooLong)
    return false
  }
  if (formMode.value === "create") {
    const expectedTier = expectedTierForGpath(value.membership_gpath)
    if (value.tier_level !== expectedTier) {
      toast.error(copy.value.validation.tierSequenceInvalid(expectedTier))
      return false
    }
  }
  try {
    JSON.parse(value.features_json.trim() || "[]")
  } catch {
    toast.error(copy.value.validation.featuresJsonInvalid)
    return false
  }
  return true
}

function membershipPayload() {
  const value = form.value
  return {
    membership_gpath: value.membership_gpath.trim(),
    name: value.name.trim(),
    description: value.description,
    features_json: value.features_json.trim() || "[]",
    ideal_for: value.ideal_for,
    duration_in_months: value.duration_in_months,
    casdoor_role_name: value.casdoor_role_name.trim(),
    tier_level: value.tier_level,
    course_discount_coupon: value.course_discount_coupon.trim(),
  }
}

async function saveMembership() {
  if (!validateForm()) return
  saving.value = true
  const currentId = form.value.membership_ulid.trim()
  try {
    const body = formMode.value === "create"
      ? { membership_ulid: currentId, ...membershipPayload() }
      : membershipPayload()
    await apiClient("/api/memberships", {
      method: formMode.value === "create" ? "POST" : "PUT",
      body: JSON.stringify(body),
    })
    toast.success(
      formMode.value === "create"
        ? copy.value.toasts.created
        : copy.value.toasts.updated,
    )
    editorOpen.value = false
    await load()
    if (formMode.value === "edit") await refreshSelected(currentId)
    form.value = emptyMembershipForm()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(
      err,
      formMode.value === "create"
        ? copy.value.toasts.createFailed
        : copy.value.toasts.updateFailed,
    ))
  } finally {
    saving.value = false
  }
}

function openDeprecateConfirm() {
  if (!selected.value || isSelectedDeprecated.value) return
  deprecateConfirmOpen.value = true
}

function closeDeprecateConfirm() {
  if (deprecating.value) return
  deprecateConfirmOpen.value = false
}

async function deprecateMembership() {
  if (!selected.value) return
  const id = membershipId(selected.value)
  const gpath = membershipGpath(selected.value)
  if (!id || !gpath) return
  deprecating.value = true
  try {
    await apiClient(
      `/api/memberships/${encodeURIComponent(id)}/deprecate?membership_gpath=${encodeURIComponent(gpath)}`,
      { method: "POST" },
    )
    toast.success(copy.value.toasts.deprecated)
    deprecateConfirmOpen.value = false
    dialogOpen.value = false
    selected.value = null
    await load()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.deprecateFailed))
  } finally {
    deprecating.value = false
  }
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
      <div class="flex flex-wrap items-center gap-3">
        <button class="inline-flex items-center gap-2 rounded-xl bg-blue-700 px-4 py-3 text-sm font-bold text-white shadow-sm transition hover:bg-blue-800" type="button" @click="openCreateEditor">
          <Plus class="h-4 w-4" />
          {{ copy.createMembership }}
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm disabled:opacity-50" type="button" :disabled="loading" @click="load">
          <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
          {{ copy.refresh }}
        </button>
      </div>
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
              <div class="flex flex-wrap items-center gap-3">
                <h2 class="truncate text-xl font-black md:text-2xl">{{ membershipName(selected) }}</h2>
                <TranslationsEditor
                  :endpoint="`/api/memberships/${encodeURIComponent(membershipId(selected))}/translations`"
                  :target-id="membershipId(selected)"
                  :fields="translationFields"
                  :default-values="translationDefaultValues"
                />
              </div>
              <p class="mt-1 break-all font-mono text-xs font-bold text-blue-700">{{ membershipId(selected) }}</p>
            </div>
            <div class="flex shrink-0 flex-wrap items-center justify-end gap-2">
              <button v-if="!isSelectedDeprecated" class="inline-flex h-10 items-center gap-2 rounded-xl border border-blue-200 bg-blue-50 px-3 text-sm font-bold text-blue-700 disabled:opacity-50" type="button" :disabled="detailLoading" @click="openEditEditor">
                <Pencil class="h-4 w-4" />
                {{ copy.editMembership }}
              </button>
              <button v-if="!isSelectedDeprecated" class="inline-flex h-10 items-center gap-2 rounded-xl border border-amber-300 bg-amber-50 px-3 text-sm font-bold text-amber-800 disabled:opacity-50" type="button" :disabled="detailLoading" @click="openDeprecateConfirm">
                <Archive class="h-4 w-4" />
                {{ copy.deprecateMembership }}
              </button>
              <button class="rounded-full border border-slate-200 p-2 text-slate-500" type="button" :aria-label="copy.close" :disabled="detailLoading" @click="closeDialog">
                <X class="h-5 w-5" />
              </button>
            </div>
          </header>

          <main class="min-h-0 flex-1 overflow-y-auto p-4 md:p-5">
            <div v-if="detailLoading" class="flex items-center justify-center gap-2 p-12 text-sm font-bold text-slate-500">
              <Loader2 class="h-6 w-6 animate-spin" />
              {{ copy.loading }}
            </div>
            <div v-else class="space-y-6">
              <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.fields.name }}</div>
                  <div class="mt-2 text-sm font-bold">{{ membershipName(selected) }}</div>
                </div>
                <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4 md:col-span-2">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.fields.description }}</div>
                  <div class="mt-2 whitespace-pre-wrap text-sm font-bold">{{ selected.description || copy.noDescription }}</div>
                </div>
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
                <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.fields.casdoorRoleName }}</div>
                  <div class="mt-2 break-all text-sm font-bold">{{ selected.casdoor_role_name || "-" }}</div>
                </div>
                <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4 md:col-span-2">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.fields.courseDiscountCoupon }}</div>
                  <div class="mt-2 break-all text-sm font-bold">{{ selected.course_discount_coupon || "-" }}</div>
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

            </div>
          </main>
        </div>
      </section>
    </Teleport>

    <Teleport to="body">
      <section v-if="editorOpen" class="fixed inset-0 z-[70] flex items-center justify-center bg-slate-950/60 p-0 md:p-6">
        <form v-modal-dialog="closeEditor" class="flex h-full w-full max-w-[1040px] flex-col overflow-hidden bg-white shadow-2xl md:h-auto md:max-h-[92vh] md:rounded-3xl" @submit.prevent="saveMembership">
          <header class="flex items-start justify-between gap-4 border-b border-slate-200 p-4 md:p-6">
            <div>
              <h2 class="text-xl font-black md:text-2xl">{{ editorTitle }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ editorDescription }}</p>
            </div>
            <button class="rounded-full border border-slate-200 p-2 text-slate-500 disabled:opacity-50" type="button" :aria-label="copy.close" :disabled="saving" @click="closeEditor">
              <X class="h-5 w-5" />
            </button>
          </header>

          <main class="min-h-0 flex-1 space-y-5 overflow-y-auto p-4 md:p-6">
            <div class="rounded-2xl border border-blue-200 bg-blue-50 p-4 text-sm text-blue-900">
              <div class="font-black">{{ copy.form.contractTitle }}</div>
              <p class="mt-1 leading-6">{{ formMode === "create" ? copy.form.createContractHint : copy.form.editContractHint }}</p>
            </div>

            <div class="grid gap-5 md:grid-cols-2">
              <label class="space-y-2">
                <span class="text-sm font-black text-slate-800">{{ copy.fields.membershipId }}</span>
                <input v-model="form.membership_ulid" class="h-11 w-full rounded-xl border border-slate-300 bg-slate-100 px-3 font-mono text-sm text-slate-600 outline-none" type="text" readonly>
                <span class="block text-xs text-slate-500">{{ copy.form.idGeneratedHint }}</span>
              </label>

              <label class="space-y-2">
                <span class="text-sm font-black text-slate-800"><span class="text-red-500">*</span> {{ copy.fields.membershipGpath }}</span>
                <input v-model="form.membership_gpath" class="h-11 w-full rounded-xl border border-slate-300 px-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100 disabled:bg-slate-100 disabled:text-slate-500" type="text" :placeholder="copy.form.gpathPlaceholder" :disabled="formMode === 'edit'" @input="syncCreateTier">
                <span class="block text-xs text-slate-500">{{ formMode === "create" ? copy.form.gpathHint : copy.form.immutableHint }}</span>
              </label>

              <label class="space-y-2 md:col-span-2">
                <span class="text-sm font-black text-slate-800"><span class="text-red-500">*</span> {{ copy.fields.name }}</span>
                <input v-model="form.name" class="h-11 w-full rounded-xl border border-slate-300 px-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100" type="text" maxlength="255">
              </label>

              <label class="space-y-2 md:col-span-2">
                <span class="text-sm font-black text-slate-800">{{ copy.fields.description }}</span>
                <textarea v-model="form.description" class="min-h-28 w-full rounded-xl border border-slate-300 px-3 py-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100" />
              </label>

              <label class="space-y-2">
                <span class="text-sm font-black text-slate-800">{{ copy.fields.duration }}</span>
                <input v-model.number="form.duration_in_months" class="h-11 w-full rounded-xl border border-slate-300 px-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100" type="number" min="0" step="1">
              </label>

              <label class="space-y-2">
                <span class="text-sm font-black text-slate-800"><span class="text-red-500">*</span> {{ copy.fields.tierLevel }}</span>
                <input v-model.number="form.tier_level" class="h-11 w-full rounded-xl border border-slate-300 px-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100 disabled:bg-slate-100 disabled:text-slate-500" type="number" min="1" step="1" disabled>
                <span class="block text-xs text-slate-500">{{ formMode === "create" ? copy.form.tierHint(form.tier_level) : copy.form.immutableHint }}</span>
              </label>

              <label class="space-y-2">
                <span class="text-sm font-black text-slate-800"><span class="text-red-500">*</span> {{ copy.fields.casdoorRoleName }}</span>
                <input v-model="form.casdoor_role_name" class="h-11 w-full rounded-xl border border-slate-300 px-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100" type="text" maxlength="255">
              </label>

              <label class="space-y-2">
                <span class="text-sm font-black text-slate-800">{{ copy.fields.courseDiscountCoupon }}</span>
                <input v-model="form.course_discount_coupon" class="h-11 w-full rounded-xl border border-slate-300 px-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100" type="text" maxlength="255" :placeholder="copy.form.couponPlaceholder">
              </label>

              <label class="space-y-2 md:col-span-2">
                <span class="text-sm font-black text-slate-800">{{ copy.fields.idealFor }}</span>
                <textarea v-model="form.ideal_for" class="min-h-24 w-full rounded-xl border border-slate-300 px-3 py-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100" />
              </label>

              <label class="space-y-2 md:col-span-2">
                <span class="text-sm font-black text-slate-800">{{ copy.fields.featuresJson }}</span>
                <textarea v-model="form.features_json" class="min-h-56 w-full rounded-xl border border-slate-300 px-3 py-3 font-mono text-xs leading-5 outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100" spellcheck="false" />
                <span class="block text-xs text-slate-500">{{ copy.form.featuresJsonHint }}</span>
              </label>
            </div>
          </main>

          <footer class="flex flex-wrap justify-end gap-3 border-t border-slate-200 p-4 md:px-6">
            <button class="h-11 rounded-xl border border-slate-300 bg-white px-5 text-sm font-bold text-slate-700 disabled:opacity-50" type="button" :disabled="saving" @click="closeEditor">
              {{ copy.cancel }}
            </button>
            <button class="inline-flex h-11 min-w-[140px] items-center justify-center gap-2 rounded-xl bg-blue-700 px-5 text-sm font-bold text-white disabled:cursor-not-allowed disabled:opacity-60" type="submit" :disabled="saving">
              <Loader2 v-if="saving" class="h-4 w-4 animate-spin" />
              <Save v-else class="h-4 w-4" />
              {{ saving ? copy.saving : (formMode === "create" ? copy.saveCreate : copy.saveChanges) }}
            </button>
          </footer>
        </form>
      </section>
    </Teleport>

    <Teleport to="body">
      <section v-if="deprecateConfirmOpen && selected" class="fixed inset-0 z-[80] flex items-center justify-center bg-slate-950/65 p-4 md:p-6">
        <div v-modal-dialog="closeDeprecateConfirm" class="w-full max-w-lg rounded-3xl bg-white p-5 shadow-2xl md:p-6">
          <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-amber-100 text-amber-700">
            <TriangleAlert class="h-6 w-6" />
          </div>
          <h2 class="mt-5 text-xl font-black md:text-2xl">{{ copy.deprecateConfirmTitle }}</h2>
          <p class="mt-3 text-sm leading-6 text-slate-600">{{ copy.deprecateConfirmDescription(membershipGpath(selected)) }}</p>
          <div class="mt-5 rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm font-bold text-amber-900">
            {{ copy.deprecateScopeWarning }}
          </div>
          <div class="mt-6 flex flex-wrap justify-end gap-3">
            <button class="h-11 rounded-xl border border-slate-300 bg-white px-5 text-sm font-bold text-slate-700 disabled:opacity-50" type="button" :disabled="deprecating" @click="closeDeprecateConfirm">
              {{ copy.cancel }}
            </button>
            <button class="inline-flex h-11 min-w-[140px] items-center justify-center gap-2 rounded-xl bg-amber-600 px-5 text-sm font-bold text-white disabled:cursor-not-allowed disabled:opacity-60" type="button" :disabled="deprecating" @click="deprecateMembership">
              <Loader2 v-if="deprecating" class="h-4 w-4 animate-spin" />
              <Archive v-else class="h-4 w-4" />
              {{ deprecating ? copy.deprecating : copy.confirmDeprecate }}
            </button>
          </div>
        </div>
      </section>
    </Teleport>
  </section>
</template>
