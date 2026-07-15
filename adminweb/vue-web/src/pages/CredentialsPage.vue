<script setup lang="ts">
import { Loader2, Plus, RefreshCw, Trash2, X } from "lucide-vue-next"
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
  return option?.label || text
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
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">{{ copy.pageTitle }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.pageDescription }}</p>
      </div>
      <div class="flex gap-3">
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load">
          <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
          {{ copy.refresh }}
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl bg-blue-700 px-4 py-3 text-sm font-bold text-white shadow-sm" type="button" @click="startCreate">
          <Plus class="h-4 w-4" />
          {{ copy.newCredential }}
        </button>
      </div>
    </header>

    <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
      <div class="flex items-center justify-between border-b border-slate-200 p-5">
        <div>
          <h2 class="text-xl font-black">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
        <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">共 {{ definitions.length }} 条</span>
      </div>
      <div v-if="loading" class="p-12 text-center text-slate-500">
        <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
        {{ copy.loading }}
      </div>
      <div v-else-if="!definitions.length" class="p-12 text-center text-slate-500">{{ copy.empty }}</div>
      <template v-else>
        <div class="grid grid-cols-[minmax(0,1fr)_180px_112px] gap-4 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-500">
          <span>{{ copy.columns.credential }}</span>
          <span class="text-center">{{ copy.columns.category }}</span>
          <span class="text-right">{{ copy.columns.action }}</span>
        </div>
        <div
          v-for="definition in definitions"
          :key="definitionUlid(definition)"
          class="grid w-full cursor-pointer grid-cols-[minmax(0,1fr)_180px_112px] gap-4 border-b border-slate-100 px-5 py-4 text-left transition last:border-b-0 hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-200"
          :class="mode === 'detail' && definitionUlid(selected) === definitionUlid(definition) ? 'bg-sky-50' : ''"
          role="button"
          tabindex="0"
          @click="selectDefinition(definition)"
          @keydown.enter.prevent="selectDefinition(definition)"
          @keydown.space.prevent="selectDefinition(definition)"
        >
          <div class="min-w-0">
            <div class="truncate text-lg font-black">{{ definitionName(definition) }}</div>
            <div class="mt-2 break-all text-xs font-semibold text-slate-500">ID: {{ definitionUlid(definition) || "-" }}</div>
          </div>
          <span class="self-center justify-self-center rounded-full bg-slate-100 px-3 py-1 text-sm font-bold text-slate-600">{{ categoryLabel(definition.category) }}</span>
          <button class="self-center justify-self-end text-sm font-bold text-[#1890ff] transition hover:underline" type="button" @click.stop="selectDefinition(definition)">
            {{ copy.viewDetails }}
          </button>
        </div>
      </template>
    </section>

    <Teleport to="body">
      <div v-if="detailOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/45 p-6">
        <div class="flex max-h-[88vh] w-full max-w-[1180px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-6 py-5">
            <div class="min-w-0">
              <h2 class="truncate text-2xl font-black">{{ mode === "create" ? copy.createTitle : selected ? definitionName(selected) : copy.detailTitle }}</h2>
              <p class="mt-1 break-all text-sm text-slate-500">
                {{ mode === "create" ? copy.createHint : definitionUlid(selected) || copy.selectCredential }}
              </p>
            </div>
            <div class="flex items-center gap-3">
              <Loader2 v-if="detailLoading" class="h-4 w-4 animate-spin text-slate-400" />
              <button class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="closeDetail">
                <X class="h-5 w-5" />
              </button>
            </div>
          </div>

          <section class="flex-1 overflow-y-auto p-6">
            <template v-if="mode === 'create'">
              <div class="grid gap-5">
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="grid gap-2 text-sm font-bold">
                    <span><span class="mr-1 text-red-500">*</span>{{ copy.labels.name }}</span>
                    <input v-model="name" class="rounded-xl border border-slate-200 px-4 py-3" maxlength="120" :placeholder="copy.placeholders.name" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    <span><span class="mr-1 text-red-500">*</span>{{ copy.labels.category }}</span>
                    <select v-model="category" class="rounded-xl border border-slate-200 px-4 py-3">
                      <option value="" disabled>{{ copy.placeholders.category }}</option>
                      <option v-for="option in categoryOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
                    </select>
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    <span><span class="mr-1 text-red-500">*</span>{{ copy.labels.respath }}</span>
                    <input v-model="respath" class="rounded-xl border border-slate-200 px-4 py-3" maxlength="240" :placeholder="copy.placeholders.respath" @input="onRespathInput" />
                    <div class="text-xs font-normal text-slate-500">{{ copy.respathHint }}</div>
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    {{ copy.labels.acquisitionMethod }}
                    <input v-model="acquisitionMethod" class="rounded-xl border border-slate-200 px-4 py-3" maxlength="240" :placeholder="copy.placeholders.optional" />
                  </label>
                </div>
                <label class="grid gap-2 text-sm font-bold">
                  <span><span class="mr-1 text-red-500">*</span>{{ copy.labels.description }}</span>
                  <textarea v-model="description" class="min-h-28 rounded-xl border border-slate-200 px-4 py-3" maxlength="1000" :placeholder="copy.placeholders.description" />
                </label>
                <div class="rounded-2xl border border-slate-200 p-4">
                  <div class="mb-3 flex items-center justify-between gap-4">
                    <div>
                      <div class="font-black">{{ copy.labels.fileConstraints }}</div>
                      <div class="mt-1 text-xs text-slate-500">{{ copy.fileConstraintsHint }}</div>
                    </div>
                    <button class="inline-flex items-center gap-2 rounded-xl border px-3 py-2 text-sm font-bold" type="button" @click="addConstraint">
                      <Plus class="h-4 w-4" />
                      {{ copy.addFile }}
                    </button>
                  </div>
                  <div v-for="(constraint, index) in constraints" :key="index" class="mb-3 grid gap-3 rounded-xl bg-slate-50 p-3 md:grid-cols-[1fr_170px_100px_auto]">
                    <input v-model="constraint.name" class="rounded-lg border border-slate-200 px-3 py-2" maxlength="120" :placeholder="copy.placeholders.filePurpose" />
                    <select v-model.number="constraint.type" class="rounded-lg border border-slate-200 px-3 py-2">
                      <option v-for="type in fileTypes" :key="type.value" :value="type.value">{{ type.label }}</option>
                    </select>
                    <label class="flex items-center gap-2 text-sm font-bold">
                      <input v-model="constraint.is_required" type="checkbox" />
                      {{ copy.required }}
                    </label>
                    <button class="inline-flex items-center justify-center rounded-lg border px-3 py-2 text-red-600" type="button" @click="constraints.splice(index, 1)">
                      <Trash2 class="h-4 w-4" />
                    </button>
                  </div>
                  <div v-if="!constraints.length" class="text-sm text-slate-500">{{ copy.noFileConstraints }}</div>
                </div>
              </div>
            </template>

            <template v-else>
              <div v-if="!selected" class="p-10 text-center text-slate-500">{{ copy.selectCredential }}</div>
              <div v-else class="space-y-5">
                <div class="grid gap-4 md:grid-cols-2">
                  <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                    <div class="text-xs font-black uppercase text-slate-400">{{ copy.labels.name }}</div>
                    <div class="mt-2 break-all text-sm font-bold text-slate-950">{{ definitionName(selected) }}</div>
                  </div>
                  <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                    <div class="text-xs font-black uppercase text-slate-400">{{ copy.labels.category }}</div>
                    <div class="mt-2 break-all text-sm font-bold text-slate-950">{{ categoryLabel(selected.category) }}</div>
                  </div>
                  <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                    <div class="text-xs font-black uppercase text-slate-400">{{ copy.labels.respath }}</div>
                    <div class="mt-2 break-all text-sm font-bold text-slate-950">{{ selected.respath || "-" }}</div>
                  </div>
                  <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                    <div class="text-xs font-black uppercase text-slate-400">{{ copy.labels.description }}</div>
                    <div class="mt-2 break-all text-sm font-bold text-slate-950">{{ selected.description || "-" }}</div>
                  </div>
                </div>

                <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                  <div class="mb-3 text-sm font-black">{{ copy.labels.requiredFiles }}</div>
                  <div v-if="!fileConstraints(selected).length" class="text-sm text-slate-500">{{ copy.noFileConstraints }}</div>
                  <div v-else class="grid gap-3 md:grid-cols-2">
                    <div v-for="constraint in fileConstraints(selected)" :key="String(constraint.name)" class="rounded-xl border border-slate-200 bg-white p-4">
                      <div class="font-bold">{{ constraint.name || copy.unnamedFile }}</div>
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

          <div v-if="mode === 'create'" class="flex shrink-0 justify-end gap-3 border-t border-slate-200 bg-white px-5 py-4">
            <button class="h-10 rounded-xl border border-slate-900 px-5 font-bold text-slate-950 disabled:opacity-50" type="button" :disabled="creating" @click="closeDetail">{{ copy.cancel }}</button>
            <button class="inline-flex h-10 min-w-[180px] items-center justify-center rounded-xl bg-blue-700 px-4 font-bold text-white disabled:opacity-50" type="button" :disabled="creating" @click="createDefinition">
              {{ creating ? copy.creating : copy.create }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </section>
</template>
