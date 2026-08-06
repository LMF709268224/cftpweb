<script setup lang="ts">
import { Languages, Loader2, Plus, Save, Trash2, X } from "lucide-vue-next"
import { computed, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { useAdminLanguage } from "@/lib/language"

type TranslationRecord = Record<string, unknown>
type TranslationMap = Record<string, TranslationRecord>
type MapEntry = {
  key: string
  label: string
}
type FieldSpec = {
  key: string
  label: string
  kind?: "text" | "textarea" | "map"
  maxLength?: number
  rows?: number
  mapEntries?: MapEntry[]
}

const props = withDefaults(defineProps<{
  endpoint: string
  targetId?: string
  fields: FieldSpec[]
  title?: string
  description?: string
}>(), {
  targetId: "",
  title: "",
  description: "",
})

const { t } = useAdminLanguage()
const copy = computed(() => t.value.translationsEditor)
const translations = ref<TranslationMap>({})
const originalLocales = ref(new Set<string>())
const removedLocales = ref(new Set<string>())
const loading = ref(false)
const saving = ref(false)
const addOpen = ref(false)
const pendingLocale = ref("")
let requestId = 0

const localeOptions = computed(() => [
  { value: "zh-CN", label: copy.value.locales.zhCN },
  { value: "en-US", label: copy.value.locales.enUS },
  { value: "ja-JP", label: copy.value.locales.jaJP },
])
const usedLocales = computed(() => new Set(Object.keys(translations.value)))
const availableLocales = computed(() => localeOptions.value.filter((option) => !usedLocales.value.has(option.value)))
const editorTitle = computed(() => props.title || copy.value.title)
const editorDescription = computed(() => props.description || copy.value.description)

function cloneRecord(value: TranslationRecord): TranslationRecord {
  return JSON.parse(JSON.stringify(value || {})) as TranslationRecord
}

function hasTranslationContent(value: unknown): boolean {
  if (typeof value === "string") return value.trim().length > 0
  if (Array.isArray(value)) return value.some(hasTranslationContent)
  if (value && typeof value === "object") return Object.values(value).some(hasTranslationContent)
  return value !== null && value !== undefined
}

function normalizeTranslations(value: unknown): TranslationMap {
  if (!value || typeof value !== "object" || Array.isArray(value)) return {}
  const next: TranslationMap = {}
  for (const [locale, raw] of Object.entries(value)) {
    if (!raw || typeof raw !== "object" || Array.isArray(raw) || !hasTranslationContent(raw)) continue
    next[locale] = cloneRecord(raw as TranslationRecord)
  }
  return next
}

function localeLabel(locale: string) {
  return localeOptions.value.find((option) => option.value === locale)?.label || locale
}

function inputValue(event: Event) {
  return (event.target as HTMLInputElement | HTMLTextAreaElement | null)?.value || ""
}

function fieldValue(locale: string, key: string) {
  return String(translations.value[locale]?.[key] || "")
}

function updateField(locale: string, key: string, value: string) {
  translations.value = {
    ...translations.value,
    [locale]: {
      ...(translations.value[locale] || {}),
      [key]: value,
    },
  }
}

function mapValue(locale: string, fieldKey: string, mapKey: string) {
  const raw = translations.value[locale]?.[fieldKey]
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return ""
  return String((raw as Record<string, unknown>)[mapKey] || "")
}

function updateMapField(locale: string, fieldKey: string, mapKey: string, value: string) {
  const raw = translations.value[locale]?.[fieldKey]
  const current = raw && typeof raw === "object" && !Array.isArray(raw)
    ? raw as Record<string, unknown>
    : {}
  translations.value = {
    ...translations.value,
    [locale]: {
      ...(translations.value[locale] || {}),
      [fieldKey]: {
        ...current,
        [mapKey]: value,
      },
    },
  }
}

function mapEntries(field: FieldSpec) {
  const entries = new Map((field.mapEntries || []).map((entry) => [entry.key, entry]))
  for (const translation of Object.values(translations.value)) {
    const raw = translation[field.key]
    if (!raw || typeof raw !== "object" || Array.isArray(raw)) continue
    for (const key of Object.keys(raw)) {
      if (!entries.has(key)) entries.set(key, { key, label: key })
    }
  }
  return Array.from(entries.values())
}

function startAdd() {
  if (!availableLocales.value.length) return
  pendingLocale.value = availableLocales.value[0]?.value || ""
  addOpen.value = true
}

function cancelAdd() {
  pendingLocale.value = ""
  addOpen.value = false
}

function addLocale() {
  const locale = pendingLocale.value.trim()
  if (!locale) return
  if (usedLocales.value.has(locale)) {
    toast.error(copy.value.duplicateLocale)
    return
  }
  translations.value = { ...translations.value, [locale]: {} }
  removedLocales.value.delete(locale)
  cancelAdd()
}

function removeLocale(locale: string) {
  const next = { ...translations.value }
  delete next[locale]
  translations.value = next
  if (originalLocales.value.has(locale)) removedLocales.value.add(locale)
}

async function load() {
  requestId += 1
  const currentRequest = requestId
  translations.value = {}
  originalLocales.value = new Set()
  removedLocales.value = new Set()
  cancelAdd()
  if (!props.targetId || !props.endpoint) return
  loading.value = true
  try {
    const response = await apiClient<{ translations?: TranslationMap }>(props.endpoint)
    if (currentRequest !== requestId) return
    const normalized = normalizeTranslations(response?.translations)
    translations.value = normalized
    originalLocales.value = new Set(Object.keys(normalized))
  } catch (err) {
    if (currentRequest !== requestId) return
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.loadFailed))
  } finally {
    if (currentRequest === requestId) loading.value = false
  }
}

async function save() {
  if (!props.targetId || !props.endpoint) {
    toast.error(copy.value.saveBaseFirst)
    return
  }
  const payload: TranslationMap = {}
  for (const [locale, value] of Object.entries(translations.value)) {
    payload[locale] = cloneRecord(value)
  }
  for (const locale of removedLocales.value) {
    payload[locale] = {}
  }
  saving.value = true
  try {
    await apiClient(props.endpoint, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ translations: payload }),
    })
    originalLocales.value = new Set(Object.keys(translations.value))
    removedLocales.value = new Set()
    toast.success(copy.value.saved)
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.saveFailed))
  } finally {
    saving.value = false
  }
}

watch(
  () => [props.endpoint, props.targetId],
  () => {
    void load()
  },
  { immediate: true },
)
</script>

<template>
  <section class="rounded-2xl border border-sky-200 bg-sky-50/70 p-4 md:p-5">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div class="min-w-0">
        <h3 class="flex items-center gap-2 text-lg font-black text-slate-950">
          <Languages class="h-5 w-5 text-blue-700" />
          {{ editorTitle }}
        </h3>
        <p class="mt-1 text-sm font-semibold leading-6 text-slate-600">{{ editorDescription }}</p>
        <p class="mt-1 text-xs font-semibold text-slate-500">{{ copy.baseLanguageHint }}</p>
      </div>
      <button
        class="inline-flex h-10 items-center justify-center gap-2 rounded-xl border border-blue-200 bg-white px-4 text-sm font-bold text-blue-700 shadow-sm disabled:cursor-not-allowed disabled:opacity-50"
        type="button"
        :disabled="loading || !targetId || !availableLocales.length"
        @click="startAdd"
      >
        <Plus class="h-4 w-4" />
        {{ copy.addLanguage }}
      </button>
    </div>

    <div v-if="!targetId" class="mt-4 rounded-xl border border-dashed border-amber-300 bg-amber-50 px-4 py-3 text-sm font-semibold text-amber-800">
      {{ copy.saveBaseFirst }}
    </div>
    <div v-else-if="loading" class="mt-4 flex items-center justify-center gap-2 rounded-xl border border-slate-200 bg-white p-6 text-sm font-bold text-slate-500">
      <Loader2 class="h-5 w-5 animate-spin" />
      {{ copy.loading }}
    </div>
    <template v-else>
      <div v-if="addOpen" class="mt-4 flex flex-col gap-3 rounded-xl border border-blue-200 bg-white p-4 sm:flex-row sm:items-end">
        <label class="grid flex-1 gap-2 text-sm font-bold">
          {{ copy.language }}
          <select v-model="pendingLocale" class="h-11 rounded-xl border border-slate-200 bg-white px-4">
            <option v-for="option in availableLocales" :key="option.value" :value="option.value">
              {{ option.label }} ({{ option.value }})
            </option>
          </select>
        </label>
        <div class="flex gap-2">
          <button class="inline-flex h-11 flex-1 items-center justify-center rounded-xl bg-blue-700 px-4 text-sm font-bold text-white sm:flex-none" type="button" @click="addLocale">
            {{ copy.confirmAdd }}
          </button>
          <button class="inline-flex h-11 items-center justify-center rounded-xl border border-slate-200 bg-white px-4 text-sm font-bold text-slate-600" type="button" @click="cancelAdd">
            <X class="h-4 w-4" />
          </button>
        </div>
      </div>

      <div v-if="!Object.keys(translations).length" class="mt-4 rounded-xl border border-dashed border-slate-300 bg-white/80 p-5 text-center text-sm font-semibold text-slate-500">
        {{ copy.empty }}
      </div>

      <div v-else class="mt-4 space-y-4">
        <article v-for="locale in Object.keys(translations)" :key="locale" class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
          <div class="mb-4 flex items-center justify-between gap-3 border-b border-slate-100 pb-3">
            <div>
              <div class="font-black text-slate-950">{{ localeLabel(locale) }}</div>
              <div class="mt-0.5 font-mono text-xs font-bold text-blue-700">{{ locale }}</div>
            </div>
            <button class="inline-flex items-center gap-2 rounded-xl px-3 py-2 text-sm font-bold text-red-600 hover:bg-red-50" type="button" @click="removeLocale(locale)">
              <Trash2 class="h-4 w-4" />
              {{ copy.remove }}
            </button>
          </div>

          <div class="grid gap-4">
            <template v-for="field in fields" :key="field.key">
              <label v-if="field.kind !== 'map'" class="grid gap-2 text-sm font-bold">
                {{ field.label }}
                <textarea
                  v-if="field.kind === 'textarea'"
                  :value="fieldValue(locale, field.key)"
                  class="min-h-24 rounded-xl border border-slate-200 px-4 py-3 font-normal"
                  :maxlength="field.maxLength"
                  :rows="field.rows || 4"
                  @input="updateField(locale, field.key, inputValue($event))"
                />
                <input
                  v-else
                  :value="fieldValue(locale, field.key)"
                  class="h-11 rounded-xl border border-slate-200 px-4 font-normal"
                  :maxlength="field.maxLength"
                  @input="updateField(locale, field.key, inputValue($event))"
                />
              </label>

              <div v-else class="rounded-xl border border-slate-200 bg-slate-50 p-4">
                <div class="mb-3 text-sm font-black text-slate-800">{{ field.label }}</div>
                <div v-if="!mapEntries(field).length" class="text-sm font-semibold text-slate-500">{{ copy.noMapEntries }}</div>
                <div v-else class="grid gap-3 md:grid-cols-2">
                  <label v-for="entry in mapEntries(field)" :key="entry.key" class="grid gap-2 text-sm font-bold">
                    <span>{{ entry.label }} <span class="font-mono text-xs text-slate-400">({{ entry.key }})</span></span>
                    <input
                      :value="mapValue(locale, field.key, entry.key)"
                      class="h-10 rounded-lg border border-slate-200 bg-white px-3 font-normal"
                      @input="updateMapField(locale, field.key, entry.key, inputValue($event))"
                    />
                  </label>
                </div>
              </div>
            </template>
          </div>
        </article>
      </div>

      <div class="mt-4 flex justify-end">
        <button
          class="inline-flex h-11 w-full items-center justify-center gap-2 rounded-xl bg-blue-700 px-5 text-sm font-bold text-white disabled:cursor-not-allowed disabled:opacity-50 sm:w-auto"
          type="button"
          :disabled="saving || !targetId"
          @click="save"
        >
          <Loader2 v-if="saving" class="h-4 w-4 animate-spin" />
          <Save v-else class="h-4 w-4" />
          {{ saving ? copy.saving : copy.save }}
        </button>
      </div>
    </template>
  </section>
</template>
