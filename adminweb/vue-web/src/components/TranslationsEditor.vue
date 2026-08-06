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
  defaultValues?: TranslationRecord
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
const loadFailed = ref(false)
const saving = ref(false)
const dialogOpen = ref(false)
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

function translationDefaults(): TranslationRecord {
  const allowedKeys = new Set(props.fields.map((field) => field.key))
  return Object.fromEntries(
    Object.entries(props.defaultValues || {})
      .filter(([key]) => allowedKeys.has(key)),
  )
}

function withTranslationDefaults(value: TranslationRecord): TranslationRecord {
  return {
    ...cloneRecord(translationDefaults()),
    ...cloneRecord(value),
  }
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
    next[locale] = withTranslationDefaults(raw as TranslationRecord)
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
  translations.value = {
    ...translations.value,
    [locale]: withTranslationDefaults({}),
  }
  removedLocales.value.delete(locale)
  cancelAdd()
}

function removeLocale(locale: string) {
  const next = { ...translations.value }
  delete next[locale]
  translations.value = next
  if (originalLocales.value.has(locale)) removedLocales.value.add(locale)
}

function openDialog() {
  if (!props.targetId || !props.endpoint) {
    toast.error(copy.value.saveBaseFirst)
    return
  }
  dialogOpen.value = true
  void load()
}

function closeDialog() {
  if (saving.value) return
  dialogOpen.value = false
  cancelAdd()
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
  loadFailed.value = false
  try {
    const response = await apiClient<{ translations?: TranslationMap }>(props.endpoint)
    if (currentRequest !== requestId) return
    const normalized = normalizeTranslations(response?.translations)
    translations.value = normalized
    originalLocales.value = new Set(Object.keys(normalized))
  } catch (err) {
    if (currentRequest !== requestId) return
    loadFailed.value = true
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
    dialogOpen.value = false
    cancelAdd()
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
    translations.value = {}
    originalLocales.value = new Set()
    removedLocales.value = new Set()
    loadFailed.value = false
    cancelAdd()
    if (dialogOpen.value) void load()
  },
)
</script>

<template>
  <span class="inline-flex shrink-0">
    <button
      class="inline-flex h-8 items-center justify-center gap-1.5 rounded-lg border border-blue-200 bg-blue-50 px-3 text-xs font-bold text-blue-700 transition hover:border-blue-300 hover:bg-blue-100 disabled:cursor-not-allowed disabled:border-slate-200 disabled:bg-slate-100 disabled:text-slate-400"
      type="button"
      :disabled="!targetId"
      :title="targetId ? copy.optionalHint : copy.saveBaseFirst"
      @click="openDialog"
    >
      <Languages class="h-4 w-4" />
      {{ copy.configure }}
    </button>

    <Teleport to="body">
      <section
        v-if="dialogOpen"
        class="fixed inset-0 z-[80] flex items-center justify-center bg-slate-950/55 p-0 backdrop-blur-[2px] md:p-6"
        @mousedown.self="closeDialog"
      >
        <div
          v-modal-dialog="closeDialog"
          class="flex h-full w-full max-w-[980px] flex-col overflow-hidden bg-white shadow-2xl md:h-auto md:max-h-[90vh] md:rounded-3xl"
        >
          <header class="flex items-start justify-between gap-4 border-b border-slate-200 bg-white p-4 md:p-5">
            <div class="min-w-0">
              <h2 class="flex items-center gap-2 text-xl font-black text-slate-950 md:text-2xl">
                <Languages class="h-6 w-6 text-blue-700" />
                {{ editorTitle }}
              </h2>
              <p class="mt-1 text-sm font-semibold leading-6 text-slate-600">{{ editorDescription }}</p>
              <p class="mt-1 text-xs font-semibold text-slate-500">{{ copy.baseLanguageHint }}</p>
            </div>
            <button
              class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900 disabled:opacity-50"
              type="button"
              :aria-label="copy.close"
              :disabled="saving"
              @click="closeDialog"
            >
              <X class="h-5 w-5" />
            </button>
          </header>

          <main class="min-h-0 flex-1 overflow-y-auto bg-slate-50/70 p-4 md:p-5">
            <div class="rounded-2xl border border-sky-200 bg-sky-50/70 p-4 md:p-5">
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <h3 class="font-black text-slate-950">{{ copy.languagesTitle }}</h3>
                  <p class="mt-1 text-sm font-semibold text-slate-600">{{ copy.optionalHint }}</p>
                </div>
                <button
                  class="inline-flex h-10 items-center justify-center gap-2 rounded-xl border border-blue-200 bg-white px-4 text-sm font-bold text-blue-700 shadow-sm disabled:cursor-not-allowed disabled:opacity-50"
                  type="button"
                  :disabled="loading || !availableLocales.length"
                  @click="startAdd"
                >
                  <Plus class="h-4 w-4" />
                  {{ copy.addLanguage }}
                </button>
              </div>

              <div v-if="loading" class="mt-4 flex items-center justify-center gap-2 rounded-xl border border-slate-200 bg-white p-8 text-sm font-bold text-slate-500">
                <Loader2 class="h-5 w-5 animate-spin" />
                {{ copy.loading }}
              </div>
              <div v-else-if="loadFailed" class="mt-4 rounded-xl border border-red-200 bg-red-50 p-5 text-center text-sm font-bold text-red-700">
                {{ copy.loadFailed }}
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
                    <button class="inline-flex h-11 items-center justify-center rounded-xl border border-slate-200 bg-white px-4 text-sm font-bold text-slate-600" type="button" :aria-label="copy.cancel" @click="cancelAdd">
                      <X class="h-4 w-4" />
                    </button>
                  </div>
                </div>

                <div v-if="!Object.keys(translations).length" class="mt-4 rounded-xl border border-dashed border-slate-300 bg-white/80 p-8 text-center text-sm font-semibold text-slate-500">
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
              </template>
            </div>
          </main>

          <footer class="flex shrink-0 flex-col-reverse justify-end gap-3 border-t border-slate-200 bg-white p-4 sm:flex-row md:px-5">
            <button
              class="inline-flex h-11 items-center justify-center rounded-xl border border-slate-200 bg-white px-5 text-sm font-bold text-slate-700 disabled:opacity-50"
              type="button"
              :disabled="saving"
              @click="closeDialog"
            >
              {{ copy.cancel }}
            </button>
            <button
              class="inline-flex h-11 items-center justify-center gap-2 rounded-xl bg-blue-700 px-5 text-sm font-bold text-white disabled:cursor-not-allowed disabled:opacity-50"
              type="button"
              :disabled="saving || loading || loadFailed || !targetId"
              @click="save"
            >
              <Loader2 v-if="saving" class="h-4 w-4 animate-spin" />
              <Save v-else class="h-4 w-4" />
              {{ saving ? copy.saving : copy.save }}
            </button>
          </footer>
        </div>
      </section>
    </Teleport>
  </span>
</template>
