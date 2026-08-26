<script setup lang="ts">
import { Download, Loader2, Plus, RefreshCw, Trash2, Upload, X } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import TranslationsEditor from "@/components/TranslationsEditor.vue"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { sha256Hex, uploadToDirectURL } from "@/lib/directUpload"
import { type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { pickFirst } from "@/lib/status"

type FileConstraint = {
  name: string
  type: number
  is_required: boolean
}

type CredentialAttachment = {
  attachment_id?: string
  name: string
  description?: string
  file_name: string
  file_type: number
  file_ext: string
  file_size: number
  file_hash: string
  download_url?: string
  file_key: string
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
const attachmentName = ref("")
const attachmentDescription = ref("")
const attachmentFileType = ref(0)
const attachmentFile = ref<File | null>(null)
const attachmentInput = ref<HTMLInputElement | null>(null)
const savingAttachment = ref(false)
const { t } = useAdminLanguage()
const copy = computed(() => t.value.credentials)
let listRequestId = 0
let detailRequestId = 0

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
const credentialTranslationFields = computed(() => [
  { key: "name", label: copy.value.labels.name, maxLength: 128 },
  { key: "description", label: copy.value.labels.description, kind: "textarea" as const, maxLength: 1024 },
  { key: "acquisition_method", label: copy.value.labels.acquisitionMethod, maxLength: 256 },
  {
    key: "file_constraint_names",
    label: copy.value.labels.requiredFiles,
    kind: "map" as const,
    mapEntries: fileConstraints(selected.value)
      .map((constraint) => {
        const label = String(constraint.name || "").trim()
        return { key: normalizeTranslationKey(label), label }
      })
      .filter((entry) => entry.key),
  },
])

function definitionUlid(definition: JsonRecord | null | undefined) {
  return String(pickFirst(definition || {}, ["cred_def_ulid", "cred_def_id", "qual_ulid"]) || "")
}

function invalidateDetailRequest() {
  detailRequestId += 1
  detailLoading.value = false
}

function isCurrentDetailRequest(requestId: number, id: string) {
  return requestId === detailRequestId
    && detailOpen.value
    && mode.value === "detail"
    && definitionUlid(selected.value) === id
}

function definitionName(definition: JsonRecord | null | undefined) {
  return String(pickFirst(definition || {}, ["name", "name_hint", "title"]) || copy.value.unnamedCredential)
}

function definitionAcquisitionMethod(definition: JsonRecord | null | undefined) {
  return String(pickFirst(definition || {}, ["acquisition_method", "acquisitionMethod"]) || "")
}

function fileConstraints(definition: JsonRecord | null | undefined) {
  const value = definition?.file_constraints
  return Array.isArray(value) ? value.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item)) : []
}

function credentialAttachments(definition: JsonRecord | null | undefined): CredentialAttachment[] {
  const value = definition?.attachments
  if (!Array.isArray(value)) return []
  return value.filter((item): item is CredentialAttachment => !!item && typeof item === "object" && !Array.isArray(item))
}

function formatFileSize(value: unknown) {
  const bytes = Number(value)
  if (!Number.isFinite(bytes) || bytes <= 0) return "-"
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

function attachmentPayload(attachment: CredentialAttachment) {
  return {
    name: String(attachment.name || "").trim(),
    description: String(attachment.description || "").trim(),
    file_name: String(attachment.file_name || "").trim(),
    file_type: Number(attachment.file_type),
    file_ext: String(attachment.file_ext || "").trim().replace(/^\./, ""),
    file_size: Number(attachment.file_size),
    file_hash: String(attachment.file_hash || "").trim(),
    file_key: String(attachment.file_key || "").trim(),
  }
}

function onAttachmentFile(event: Event) {
  attachmentFile.value = (event.target as HTMLInputElement).files?.[0] || null
  if (attachmentFile.value && !attachmentName.value.trim()) attachmentName.value = attachmentFile.value.name
}

function resetAttachmentForm() {
  attachmentName.value = ""
  attachmentDescription.value = ""
  attachmentFileType.value = 0
  attachmentFile.value = null
  if (attachmentInput.value) attachmentInput.value.value = ""
}

async function replaceAttachments(attachments: CredentialAttachment[]) {
  if (!selected.value) return
  const id = definitionUlid(selected.value)
  await apiClient(`/api/credentials/definitions/${encodeURIComponent(id)}/attachments`, {
    method: "PUT",
    body: JSON.stringify({ attachments: attachments.map(attachmentPayload) }),
  })
  await loadDefinitionDetail(selected.value)
}

async function uploadAttachment() {
  const file = attachmentFile.value
  if (!selected.value || !file || !attachmentName.value.trim() || attachmentFileType.value === 0) {
    toast.error(copy.value.toasts.attachmentRequired)
    return
  }
  if (!file.type) {
    toast.error(copy.value.toasts.attachmentContentTypeMissing)
    return
  }
  const fileExt = file.name.includes(".") ? file.name.split(".").pop()?.toLowerCase() || "" : ""
  if (!fileExt) {
    toast.error(copy.value.toasts.attachmentExtensionMissing)
    return
  }

  savingAttachment.value = true
  try {
    const id = definitionUlid(selected.value)
    const fileHash = await sha256Hex(file)
    const upload = await apiClient<JsonRecord>(`/api/credentials/definitions/${encodeURIComponent(id)}/attachments/upload-url`, {
      method: "POST",
      body: JSON.stringify({
        file_name: file.name,
        file_ext: fileExt,
        content_type: file.type,
        file_hash: fileHash,
      }),
    })
    await uploadToDirectURL(file, upload)
    const next: CredentialAttachment = {
      name: attachmentName.value.trim(),
      description: attachmentDescription.value.trim(),
      file_name: file.name,
      file_type: attachmentFileType.value,
      file_ext: fileExt,
      file_size: file.size,
      file_hash: fileHash,
      file_key: String(upload.file_key || "").trim(),
    }
    if (!next.file_key) throw new Error(copy.value.toasts.attachmentFileKeyMissing)
    await replaceAttachments([...credentialAttachments(selected.value), next])
    resetAttachmentForm()
    toast.success(copy.value.toasts.attachmentSaved)
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.attachmentSaveFailed))
  } finally {
    savingAttachment.value = false
  }
}

async function removeAttachment(index: number) {
  if (!selected.value || savingAttachment.value) return
  savingAttachment.value = true
  try {
    const next = credentialAttachments(selected.value).filter((_, current) => current !== index)
    await replaceAttachments(next)
    toast.success(copy.value.toasts.attachmentRemoved)
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.attachmentRemoveFailed))
  } finally {
    savingAttachment.value = false
  }
}

function normalizeTranslationKey(value: string) {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "")
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
  invalidateDetailRequest()
  mode.value = "create"
  detailOpen.value = true
  resetForm()
}

function closeDetail() {
  invalidateDetailRequest()
  detailOpen.value = false
  if (mode.value === "create") mode.value = "detail"
}

async function loadDefinitionDetail(definition: JsonRecord) {
  const id = definitionUlid(definition)
  const requestId = ++detailRequestId
  detailLoading.value = false
  if (!id) return

  detailLoading.value = true
  try {
    const detail = await apiClient<JsonRecord>(`/api/credentials/definitions/${encodeURIComponent(id)}`)
    if (!isCurrentDetailRequest(requestId, id)) return
    const merged = { ...definition, ...detail }
    const index = definitions.value.findIndex((item) => definitionUlid(item) === id)
    if (index >= 0) definitions.value.splice(index, 1, merged)
    selected.value = merged
  } catch (err) {
    if (!isCurrentDetailRequest(requestId, id)) return
    console.error(err)
    toast.error(copy.value.toasts.detailLoadFailed)
    selected.value = definition
  } finally {
    if (isCurrentDetailRequest(requestId, id)) detailLoading.value = false
  }
}

async function selectDefinition(definition: JsonRecord) {
  selected.value = definition
  mode.value = "detail"
  detailOpen.value = true
  await loadDefinitionDetail(definition)
}

async function load() {
  const requestId = ++listRequestId
  loading.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/credentials/definitions")
    if (requestId !== listRequestId) return
    const list = Array.isArray(data.definitions) ? data.definitions : []
    definitions.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    const selectedId = definitionUlid(selected.value)
    selected.value = definitions.value.find((item) => definitionUlid(item) === selectedId) || definitions.value[0] || null
    if (!selected.value) {
      invalidateDetailRequest()
      mode.value = "create"
    }
    if (selected.value && mode.value === "detail" && detailOpen.value) void loadDefinitionDetail(selected.value)
  } catch (err) {
    if (requestId !== listRequestId) return
    console.error(err)
    toast.error(copy.value.toasts.listLoadFailed)
  } finally {
    if (requestId === listRequestId) loading.value = false
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
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-5 px-4 py-5 md:gap-6 md:px-8 md:py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-3xl font-black tracking-tight md:text-4xl">{{ copy.pageTitle }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.pageDescription }}</p>
      </div>
      <div class="flex flex-wrap gap-3">
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

    <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm md:rounded-3xl">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 p-4 md:p-5">
        <div class="min-w-0">
          <h2 class="text-xl font-black">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
        <span class="shrink-0 rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ copy.total(definitions.length) }}</span>
      </div>
      <div v-if="loading" class="p-8 text-center text-slate-500 md:p-12">
        <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
        {{ copy.loading }}
      </div>
      <div v-else-if="!definitions.length" class="p-8 text-center text-slate-500 md:p-12">{{ copy.empty }}</div>
      <template v-else>
        <div class="hidden grid-cols-[minmax(0,1fr)_180px_112px] gap-4 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-500 md:grid">
          <span>{{ copy.columns.credential }}</span>
          <span class="text-center">{{ copy.columns.category }}</span>
          <span class="text-right">{{ copy.columns.action }}</span>
        </div>
        <div
          v-for="definition in definitions"
          :key="definitionUlid(definition)"
          class="flex w-full flex-col gap-3 border-b border-slate-100 px-4 py-4 text-left transition last:border-b-0 hover:bg-slate-50 md:grid md:grid-cols-[minmax(0,1fr)_180px_112px] md:gap-4 md:px-5"
          :class="mode === 'detail' && definitionUlid(selected) === definitionUlid(definition) ? 'bg-sky-50' : ''"
        >
          <div class="min-w-0">
            <div class="break-words text-lg font-black md:truncate">{{ definitionName(definition) }}</div>
            <div class="mt-2 break-all text-xs font-semibold text-slate-500">ID: {{ definitionUlid(definition) || "-" }}</div>
          </div>
          <span class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2 md:block md:self-center md:justify-self-center md:rounded-none md:bg-transparent md:p-0">
            <span class="text-xs font-black text-slate-400 md:hidden">{{ copy.columns.category }}</span>
            <span class="inline-flex rounded-full bg-slate-100 px-3 py-1 text-sm font-bold text-slate-600">{{ categoryLabel(definition.category) }}</span>
          </span>
          <button class="inline-flex w-full items-center justify-center rounded-xl border border-blue-100 bg-blue-50 px-3 py-2 text-sm font-bold text-blue-700 transition hover:underline md:w-auto md:self-center md:justify-self-end md:border-0 md:bg-transparent md:px-0 md:py-0" type="button" @click.stop="selectDefinition(definition)">
            {{ copy.viewDetails }}
          </button>
        </div>
      </template>
    </section>

    <Teleport to="body">
      <div v-if="detailOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/45 p-0 md:p-6">
        <div v-modal-dialog="closeDetail" class="flex h-full max-h-none w-full max-w-[1180px] flex-col overflow-hidden rounded-none bg-white shadow-2xl md:h-auto md:max-h-[88vh] md:rounded-3xl">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-4 py-4 md:px-6 md:py-5">
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-3">
                <h2 class="break-words text-xl font-black md:truncate md:text-2xl">{{ mode === "create" ? copy.createTitle : selected ? definitionName(selected) : copy.detailTitle }}</h2>
                <TranslationsEditor
                  v-if="mode !== 'create' && selected"
                  :endpoint="`/api/credentials/definitions/${encodeURIComponent(definitionUlid(selected))}/translations`"
                  :target-id="definitionUlid(selected)"
                  :fields="credentialTranslationFields"
                />
              </div>
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

          <section class="min-h-0 flex-1 overflow-y-auto p-4 md:p-6">
            <template v-if="mode === 'create'">
              <div class="grid gap-5">
                <div class="grid items-start gap-4 md:grid-cols-2">
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
                  <div class="mb-3 flex flex-wrap items-center justify-between gap-4">
                    <div>
                      <div class="font-black">{{ copy.labels.fileConstraints }}</div>
                      <div class="mt-1 text-xs text-slate-500">{{ copy.fileConstraintsHint }}</div>
                    </div>
                    <button class="inline-flex w-full items-center justify-center gap-2 rounded-xl border px-3 py-2 text-sm font-bold sm:w-auto" type="button" @click="addConstraint">
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
                  <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4 md:col-span-2">
                    <div class="text-xs font-black uppercase text-slate-400">{{ copy.labels.acquisitionMethod }}</div>
                    <div class="mt-2 whitespace-pre-wrap break-words text-sm font-bold text-slate-950">{{ definitionAcquisitionMethod(selected) || "-" }}</div>
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

                <section class="border-t border-slate-200 pt-5">
                  <div class="flex flex-wrap items-start justify-between gap-3">
                    <div>
                      <h3 class="text-base font-black">{{ copy.attachments.title }}</h3>
                      <p class="mt-1 text-sm text-slate-500">{{ copy.attachments.description }}</p>
                    </div>
                    <span class="rounded-full bg-slate-100 px-3 py-1 text-xs font-bold text-slate-600">{{ copy.attachments.count(credentialAttachments(selected).length) }}</span>
                  </div>

                  <div v-if="credentialAttachments(selected).length" class="mt-4 divide-y divide-slate-200 border-y border-slate-200">
                    <div v-for="(attachment, index) in credentialAttachments(selected)" :key="attachment.attachment_id || attachment.file_key || `${attachment.file_name}-${index}`" class="grid gap-3 py-4 md:grid-cols-[minmax(0,1fr)_auto] md:items-center">
                      <div class="min-w-0">
                        <div class="break-words font-bold text-slate-950">{{ attachment.name || attachment.file_name }}</div>
                        <p v-if="attachment.description" class="mt-1 whitespace-pre-wrap break-words text-sm text-slate-600">{{ attachment.description }}</p>
                        <div class="mt-2 flex flex-wrap gap-x-3 gap-y-1 text-xs font-semibold text-slate-500">
                          <span>{{ attachment.file_name }}</span>
                          <span>{{ fileTypeLabel(attachment.file_type) }}</span>
                          <span>{{ formatFileSize(attachment.file_size) }}</span>
                        </div>
                      </div>
                      <div class="flex items-center gap-2">
                        <a v-if="attachment.download_url" class="inline-flex h-10 items-center gap-2 rounded-lg border border-slate-300 px-3 text-sm font-bold text-blue-700" :href="attachment.download_url" target="_blank" rel="noopener noreferrer">
                          <Download class="h-4 w-4" /> {{ copy.attachments.download }}
                        </a>
                        <button class="inline-flex h-10 w-10 items-center justify-center rounded-lg border border-red-200 text-red-600 disabled:opacity-50" type="button" :aria-label="copy.attachments.remove" :disabled="savingAttachment" @click="removeAttachment(index)">
                          <Trash2 class="h-4 w-4" />
                        </button>
                      </div>
                    </div>
                  </div>
                  <div v-else class="mt-4 border-y border-slate-200 py-6 text-center text-sm text-slate-500">{{ copy.attachments.empty }}</div>

                  <div class="mt-5 grid gap-4 border border-slate-200 bg-slate-50 p-4 md:grid-cols-2">
                    <label class="grid gap-2 text-sm font-bold">
                      {{ copy.attachments.name }}
                      <input v-model="attachmentName" class="rounded-lg border border-slate-200 bg-white px-3 py-2.5" maxlength="120" :placeholder="copy.attachments.namePlaceholder" />
                    </label>
                    <label class="grid gap-2 text-sm font-bold">
                      {{ copy.attachments.fileType }}
                      <select v-model.number="attachmentFileType" class="rounded-lg border border-slate-200 bg-white px-3 py-2.5">
                        <option :value="0">{{ copy.attachments.selectFileType }}</option>
                        <option v-for="option in fileTypes.filter((item) => item.value !== 0)" :key="option.value" :value="option.value">{{ option.label }}</option>
                      </select>
                    </label>
                    <label class="grid gap-2 text-sm font-bold md:col-span-2">
                      {{ copy.attachments.instructions }}
                      <textarea v-model="attachmentDescription" class="min-h-20 rounded-lg border border-slate-200 bg-white px-3 py-2.5" maxlength="500" :placeholder="copy.attachments.instructionsPlaceholder" />
                    </label>
                    <label class="grid gap-2 text-sm font-bold md:col-span-2">
                      {{ copy.attachments.file }}
                      <input ref="attachmentInput" class="rounded-lg border border-slate-200 bg-white px-3 py-2.5 text-sm" type="file" @change="onAttachmentFile" />
                    </label>
                    <div class="flex justify-end md:col-span-2">
                      <button class="inline-flex h-10 items-center gap-2 rounded-lg bg-blue-700 px-4 text-sm font-bold text-white disabled:opacity-50" type="button" :disabled="savingAttachment" @click="uploadAttachment">
                        <Loader2 v-if="savingAttachment" class="h-4 w-4 animate-spin" />
                        <Upload v-else class="h-4 w-4" />
                        {{ savingAttachment ? copy.attachments.saving : copy.attachments.upload }}
                      </button>
                    </div>
                  </div>
                </section>
              </div>
            </template>
          </section>

          <div v-if="mode === 'create'" class="flex shrink-0 flex-col justify-end gap-3 border-t border-slate-200 bg-white px-4 py-4 sm:flex-row md:px-5">
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
