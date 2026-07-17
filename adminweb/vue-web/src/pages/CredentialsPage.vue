<script setup lang="ts">
import { Award, Loader2, Plus, RefreshCw, Trash2, X } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { pickFirst } from "@/lib/status"

type FileConstraint = {
  name: string
  type: number
  is_required: boolean
}

type DetailMode = "detail" | "create"

const definitions = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const loading = ref(false)
const detailLoading = ref(false)
const detailOpen = ref(false)
const creating = ref(false)
const mode = ref<DetailMode>("detail")
const name = ref("")
const category = ref("")
const description = ref("")
const respath = ref("")
const acquisitionMethod = ref("")
const constraints = ref<FileConstraint[]>([])
const { t } = useAdminLanguage()
const copy = computed(() => t.value.credentials)

const fileTypes = computed(() => [
  { value: 0, label: copy.value.fileTypes.any },
  { value: 1, label: copy.value.fileTypes.image },
  { value: 2, label: copy.value.fileTypes.pdf },
  { value: 4, label: copy.value.fileTypes.video },
  { value: 8, label: copy.value.fileTypes.text },
])
const categoryOptions = computed(() => [
  { value: "Certification", label: copy.value.categoryOptions.certification },
  { value: "Exemption", label: copy.value.categoryOptions.exemption },
  { value: "Qualification", label: copy.value.categoryOptions.qualification },
  { value: "Academic", label: copy.value.categoryOptions.academic },
])
const categoryValues = computed(() => new Set(categoryOptions.value.map((option) => option.value)))

function definitionUlid(definition: JsonRecord | null | undefined) {
  return String(pickFirst(definition || {}, ["cred_def_ulid", "cred_def_id", "qual_ulid"]) || "")
}

function definitionName(definition: JsonRecord | null | undefined) {
  return String(pickFirst(definition || {}, ["name", "name_hint", "title"]) || copy.value.unnamedCredential)
}

function fileConstraints(definition: JsonRecord | null | undefined) {
  const value = definition?.file_constraints
  return Array.isArray(value) ? value.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item)) : []
}

function fileTypeLabel(type: unknown) {
  return fileTypes.value.find((item) => item.value === Number(type))?.label || String(type || "-")
}

function categoryLabel(value: unknown) {
  const text = String(value ?? "").trim()
  if (!text) return "-"
  const option = categoryOptions.value.find((item) => item.value.toLowerCase() === text.toLowerCase())
  if (option) return option.label
  if (text === "学业类") return copy.value.categoryOptions.academic
  return text
}

function resetForm() {
  name.value = ""
  category.value = ""
  description.value = ""
  respath.value = ""
  acquisitionMethod.value = ""
  constraints.value = []
  respathEdited = false
}

let respathEdited = false

watch(name, (val) => {
  if (mode.value === "create" && !respathEdited) {
    const slug = val.trim().toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "")
    respath.value = slug ? `/gcc/credential/${slug}` : ""
  }
})

function onRespathInput() {
  respathEdited = true
}

function startCreate() {
  mode.value = "create"
  detailOpen.value = true
  resetForm()
}

function closeDetail() {
  detailOpen.value = false
  if (mode.value === "create") mode.value = "detail"
}

async function loadDefinitionDetail(definition: JsonRecord) {
  const id = definitionUlid(definition)
  if (!id) {
    selected.value = definition
    return
  }

  detailLoading.value = true
  try {
    const detail = await apiClient<JsonRecord>(`/api/credentials/definitions/${encodeURIComponent(id)}`)
    const merged = { ...definition, ...detail }
    const index = definitions.value.findIndex((item) => definitionUlid(item) === id)
    if (index >= 0) definitions.value.splice(index, 1, merged)
    selected.value = merged
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.detailLoadFailed)
    selected.value = definition
  } finally {
    detailLoading.value = false
  }
}

async function selectDefinition(definition: JsonRecord) {
  selected.value = definition
  mode.value = "detail"
  detailOpen.value = true
  await loadDefinitionDetail(definition)
}

async function load() {
  loading.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/credentials/definitions")
    const list = Array.isArray(data.definitions) ? data.definitions : []
    definitions.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    if (!selected.value || !definitions.value.some((item) => definitionUlid(item) === definitionUlid(selected.value))) {
      selected.value = definitions.value[0] || null
    }
    if (!selected.value) mode.value = "create"
    if (selected.value && mode.value === "detail") await loadDefinitionDetail(selected.value)
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.listLoadFailed)
  } finally {
    loading.value = false
  }
}

function addConstraint() {
  constraints.value.push({ name: "", type: 2, is_required: true })
}

async function createDefinition() {
  if (!name.value.trim() || !categoryValues.value.has(category.value.trim()) || !respath.value.trim() || !description.value.trim()) {
    toast.error(copy.value.toasts.requiredCreateFields)
    return
  }
  creating.value = true
  try {
    await apiClient("/api/credentials/definitions", {
      method: "POST",
      body: JSON.stringify({
        name: name.value.trim(),
        category: category.value.trim(),
        description: description.value.trim(),
        respath: respath.value.trim(),
        acquisition_method: acquisitionMethod.value.trim(),
        file_constraints: constraints.value.map((constraint) => ({
          name: constraint.name.trim(),
          type: Number(constraint.type),
          is_required: Boolean(constraint.is_required),
        })),
      }),
    })
    toast.success(copy.value.toasts.createSuccess)
    resetForm()
    mode.value = "detail"
    await load()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.createFailed))
  } finally {
    creating.value = false
  }
}

onMounted(load)
</script>

<template>
  <main class="ui-page">
    <header class="flex flex-wrap items-start justify-between gap-4 md:items-center">
      <div class="min-w-0">
        <h1 class="ui-page-title">{{ copy.pageTitle }}</h1>
        <p class="ui-page-subtitle">{{ copy.pageDescription }}</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button class="ui-btn ui-btn-secondary" type="button" @click="load">
          <RefreshCw class="h-4 w-4 text-slate-400" :class="loading ? 'animate-spin' : ''" />
          {{ copy.refresh }}
        </button>
        <button class="ui-btn ui-btn-primary" type="button" @click="startCreate">
          <Plus class="h-4 w-4" />
          {{ copy.newCredential }}
        </button>
      </div>
    </header>

    <section class="ui-card mt-6 overflow-hidden">
      <div class="flex flex-wrap items-center justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
        <div class="min-w-0">
          <h2 class="text-base font-semibold text-ink">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
        <span class="tnum ui-badge shrink-0 bg-slate-100 text-slate-600">{{ copy.total(definitions.length) }}</span>
      </div>
      <div v-if="loading" class="divide-y divide-hairline">
        <div v-for="i in 5" :key="`sk-${i}`" class="grid gap-3 px-4 py-3.5 md:grid-cols-[minmax(0,1fr)_180px_112px] md:items-center md:gap-4">
          <div class="space-y-2">
            <div class="skeleton h-4 w-2/5" />
            <div class="skeleton h-3 w-3/5" />
          </div>
          <div class="skeleton h-5 w-16 rounded-full md:justify-self-center" />
          <div class="skeleton h-8 w-20 rounded-lg md:justify-self-end" />
        </div>
      </div>
      <div v-else-if="!definitions.length" class="ui-empty">
        <div class="ui-empty-icon">
          <Award class="h-6 w-6" />
        </div>
        <p class="ui-empty-title">{{ copy.empty }}</p>
      </div>
      <template v-else>
        <div class="hidden grid-cols-[minmax(0,1fr)_180px_112px] gap-4 border-b border-edge bg-slate-50/60 px-4 py-3 text-xs font-medium uppercase tracking-wider text-slate-400 md:grid">
          <span>{{ copy.columns.credential }}</span>
          <span class="text-center">{{ copy.columns.category }}</span>
          <span class="text-right">{{ copy.columns.action }}</span>
        </div>
        <div class="divide-y divide-hairline">
          <div
            v-for="definition in definitions"
            :key="definitionUlid(definition)"
            class="ui-table-row flex w-full cursor-pointer flex-col gap-3 px-4 py-3.5 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary-500/20 md:grid md:grid-cols-[minmax(0,1fr)_180px_112px] md:items-center md:gap-4"
            :class="mode === 'detail' && definitionUlid(selected) === definitionUlid(definition) ? 'bg-primary-50/60' : ''"
            role="button"
            tabindex="0"
            @click="selectDefinition(definition)"
            @keydown.enter.prevent="selectDefinition(definition)"
            @keydown.space.prevent="selectDefinition(definition)"
          >
            <div class="min-w-0">
              <div class="break-words text-sm font-medium text-ink md:truncate">{{ definitionName(definition) }}</div>
              <div class="tnum mt-0.5 break-all text-xs text-slate-400">ID: {{ definitionUlid(definition) || "-" }}</div>
            </div>
            <span class="flex items-center justify-between gap-3 md:block md:justify-self-center md:text-center">
              <span class="text-xs font-medium text-slate-400 md:hidden">{{ copy.columns.category }}</span>
              <span class="ui-badge bg-slate-100 text-slate-600">{{ categoryLabel(definition.category) }}</span>
            </span>
            <button class="ui-btn ui-btn-secondary ui-btn-sm w-full md:w-auto md:justify-self-end" type="button" @click.stop="selectDefinition(definition)">
              {{ copy.viewDetails }}
            </button>
          </div>
        </div>
      </template>
    </section>

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="detailOpen" class="ui-modal-mask">
          <section class="ui-modal-panel max-w-[1180px]">
            <div class="flex items-start justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
              <div class="min-w-0">
                <h2 class="break-words text-base font-semibold text-ink md:truncate md:text-lg">{{ mode === "create" ? copy.createTitle : selected ? definitionName(selected) : copy.detailTitle }}</h2>
                <p class="tnum mt-1 break-all text-sm text-slate-500">
                  {{ mode === "create" ? copy.createHint : definitionUlid(selected) || copy.selectCredential }}
                </p>
              </div>
              <div class="flex shrink-0 items-center gap-3">
                <Loader2 v-if="detailLoading" class="h-4 w-4 animate-spin text-slate-400" />
                <button class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600" type="button" :aria-label="copy.close" @click="closeDetail">
                  <X class="h-5 w-5" />
                </button>
              </div>
            </div>

            <section class="min-h-0 flex-1 overflow-y-auto p-4 md:p-6">
              <template v-if="mode === 'create'">
                <div class="grid gap-5">
                  <div class="grid items-start gap-4 md:grid-cols-2">
                    <label class="grid gap-1.5">
                      <span class="ui-label"><span class="mr-1 text-red-500">*</span>{{ copy.labels.name }}</span>
                      <input v-model="name" class="ui-input" maxlength="120" :placeholder="copy.placeholders.name" />
                    </label>
                    <label class="grid gap-1.5">
                      <span class="ui-label"><span class="mr-1 text-red-500">*</span>{{ copy.labels.category }}</span>
                      <select v-model="category" class="ui-input">
                        <option value="" disabled>{{ copy.placeholders.category }}</option>
                        <option v-for="option in categoryOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
                      </select>
                    </label>
                    <label class="grid gap-1.5">
                      <span class="ui-label"><span class="mr-1 text-red-500">*</span>{{ copy.labels.respath }}</span>
                      <input v-model="respath" class="ui-input" maxlength="240" :placeholder="copy.placeholders.respath" @input="onRespathInput" />
                      <span class="text-xs text-slate-400">{{ copy.respathHint }}</span>
                    </label>
                    <label class="grid gap-1.5">
                      <span class="ui-label">{{ copy.labels.acquisitionMethod }}</span>
                      <input v-model="acquisitionMethod" class="ui-input" maxlength="240" :placeholder="copy.placeholders.optional" />
                    </label>
                  </div>
                  <label class="grid gap-1.5">
                    <span class="ui-label"><span class="mr-1 text-red-500">*</span>{{ copy.labels.description }}</span>
                    <textarea v-model="description" class="ui-input min-h-28 py-2.5" maxlength="1000" :placeholder="copy.placeholders.description" />
                  </label>
                  <div class="rounded-2xl border border-edge p-4">
                    <div class="mb-3 flex flex-wrap items-center justify-between gap-4">
                      <div class="min-w-0">
                        <div class="text-sm font-semibold text-ink">{{ copy.labels.fileConstraints }}</div>
                        <div class="mt-1 text-xs text-slate-400">{{ copy.fileConstraintsHint }}</div>
                      </div>
                      <button class="ui-btn ui-btn-secondary ui-btn-sm w-full sm:w-auto" type="button" @click="addConstraint">
                        <Plus class="h-4 w-4 text-slate-400" />
                        {{ copy.addFile }}
                      </button>
                    </div>
                    <div v-for="(constraint, index) in constraints" :key="index" class="mb-3 grid gap-3 rounded-2xl bg-slate-50 p-3 md:grid-cols-[1fr_170px_100px_auto] md:items-center">
                      <input v-model="constraint.name" class="ui-input" maxlength="120" :placeholder="copy.placeholders.filePurpose" />
                      <select v-model.number="constraint.type" class="ui-input">
                        <option v-for="type in fileTypes" :key="type.value" :value="type.value">{{ type.label }}</option>
                      </select>
                      <label class="flex items-center gap-2 text-sm font-medium text-slate-600">
                        <input v-model="constraint.is_required" type="checkbox" class="h-4 w-4 accent-primary-600" />
                        {{ copy.required }}
                      </label>
                      <button class="ui-btn ui-btn-ghost ui-btn-sm text-red-600 hover:bg-red-50 hover:text-red-700" type="button" @click="constraints.splice(index, 1)">
                        <Trash2 class="h-4 w-4" />
                      </button>
                    </div>
                    <div v-if="!constraints.length" class="text-sm text-slate-500">{{ copy.noFileConstraints }}</div>
                  </div>
                </div>
              </template>

              <template v-else>
                <div v-if="!selected" class="ui-empty">
                  <div class="ui-empty-icon">
                    <Award class="h-6 w-6" />
                  </div>
                  <p class="ui-empty-title">{{ copy.selectCredential }}</p>
                </div>
                <div v-else class="space-y-5">
                  <div class="grid gap-4 md:grid-cols-2">
                    <div class="rounded-2xl bg-slate-50 p-4">
                      <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.labels.name }}</div>
                      <div class="mt-2 break-all text-sm font-medium text-ink">{{ definitionName(selected) }}</div>
                    </div>
                    <div class="rounded-2xl bg-slate-50 p-4">
                      <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.labels.category }}</div>
                      <div class="mt-2 break-all text-sm font-medium text-ink">{{ categoryLabel(selected.category) }}</div>
                    </div>
                    <div class="rounded-2xl bg-slate-50 p-4">
                      <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.labels.respath }}</div>
                      <div class="mt-2 break-all text-sm font-medium text-ink">{{ selected.respath || "-" }}</div>
                    </div>
                    <div class="rounded-2xl bg-slate-50 p-4">
                      <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.labels.description }}</div>
                      <div class="mt-2 break-all text-sm font-medium text-ink">{{ selected.description || "-" }}</div>
                    </div>
                  </div>

                  <div class="rounded-2xl bg-slate-50 p-4">
                    <div class="mb-3 text-sm font-semibold text-ink">{{ copy.labels.requiredFiles }}</div>
                    <div v-if="!fileConstraints(selected).length" class="text-sm text-slate-500">{{ copy.noFileConstraints }}</div>
                    <div v-else class="grid gap-3 md:grid-cols-2">
                      <div v-for="constraint in fileConstraints(selected)" :key="String(constraint.name)" class="rounded-[10px] border border-edge bg-white p-4">
                        <div class="text-sm font-medium text-ink">{{ constraint.name || copy.unnamedFile }}</div>
                        <div class="mt-1 text-sm text-slate-500">
                          {{ copy.fileTypePrefix }} {{ fileTypeLabel(constraint.type) }}
                          · {{ constraint.is_required ? copy.required : copy.optional }}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </template>
            </section>

            <div v-if="mode === 'create'" class="flex shrink-0 flex-col justify-end gap-3 border-t border-hairline bg-white px-4 py-4 sm:flex-row md:px-6">
              <button class="ui-btn ui-btn-secondary" type="button" :disabled="creating" @click="closeDetail">{{ copy.cancel }}</button>
              <button class="ui-btn ui-btn-primary sm:min-w-[180px]" type="button" :disabled="creating" @click="createDefinition">
                <Loader2 v-if="creating" class="h-4 w-4 animate-spin" />
                {{ creating ? copy.creating : copy.create }}
              </button>
            </div>
          </section>
        </div>
      </Transition>
    </Teleport>
  </main>
</template>
