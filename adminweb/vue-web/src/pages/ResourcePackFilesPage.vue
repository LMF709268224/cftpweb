<script setup lang="ts">
import { Loader2, Plus, RefreshCw, Save, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"

const pageSize = 10

const packs = ref<JsonRecord[]>([])
const files = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const total = ref(0)
const loading = ref(false)
const detailLoading = ref(false)
const saving = ref(false)
const detailOpen = ref(false)
const deleteConfirmOpen = ref(false)
const mode = ref<"create" | "edit" | "detail">("detail")
const pageToken = ref("")
const nextPageToken = ref("")
const previousTokens = ref<string[]>([])
const currentPage = ref(1)
const packFilter = ref("")
let detailRequestId = 0
const { t } = useAdminLanguage()
const copy = computed(() => t.value.resourcePackFilesAdmin)
const fileTypeOptions = computed(() => [
  { value: 1, label: copy.value.fileTypes.video },
  { value: 2, label: copy.value.fileTypes.pdf },
  { value: 3, label: copy.value.fileTypes.zip },
])

const form = ref({
  file_id: "",
  pack_id: "",
  title: "",
  description: "",
  thumbnail_object_key: "",
  thumbnail_file_hash: "",
  file_type: 2,
  file_name: "",
  file_size: 0,
  file_hash: "",
  file_object_key: "",
  video_stream_uid: "",
  sort_order: 1,
  version: 0,
  created_at: "",
  updated_at: "",
})

const canPrevious = computed(() => previousTokens.value.length > 0)
const canNext = computed(() => !!nextPageToken.value)
const selectedPack = computed(() => packById(form.value.pack_id))

function displayValue(value: unknown) {
  const text = String(value ?? "").trim()
  return text || "-"
}

function packFieldText(id: unknown) {
  const value = String(id || "")
  if (!value) return copy.value.selectPack
  const pack = packById(value)
  return copy.value.ownerText(pack ? packTitle(pack) : copy.value.unknownPack, value)
}

function asRecordList(value: unknown) {
  return Array.isArray(value) ? value.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item)) : []
}

function fileId(file: JsonRecord | null) {
  return String(file?.file_id || "")
}

function fileTitle(file: JsonRecord | null) {
  return String(file?.title || file?.file_name || file?.file_id || copy.value.unnamed)
}

function fileVersion(file: JsonRecord | null) {
  return Number(file?.version || 0)
}

function packId(pack: JsonRecord | null) {
  return String(pack?.pack_id || "")
}

function packTitle(pack: JsonRecord | null) {
  return String(pack?.title || pack?.pack_id || copy.value.unknownPack)
}

function packById(id: unknown) {
  const value = String(id || "")
  return packs.value.find((pack) => packId(pack) === value) || null
}

function ownerText(file: JsonRecord | null) {
  const pack = packById(file?.pack_id)
  const id = String(file?.pack_id || "-")
  return copy.value.ownerText(packTitle(pack), id)
}

function fileTypeLabel(value: unknown) {
  const numeric = Number(value || 0)
  return fileTypeOptions.value.find((option) => option.value === numeric)?.label || String(value || "-")
}

function fillForm(file: JsonRecord | null) {
  form.value = {
    file_id: String(file?.file_id || ""),
    pack_id: String(file?.pack_id || packFilter.value || packs.value[0]?.pack_id || ""),
    title: String(file?.title || ""),
    description: String(file?.description || ""),
    thumbnail_object_key: String(file?.thumbnail_object_key || ""),
    thumbnail_file_hash: String(file?.thumbnail_file_hash || ""),
    file_type: Number(file?.file_type || 2),
    file_name: String(file?.file_name || ""),
    file_size: Number(file?.file_size || 0),
    file_hash: String(file?.file_hash || ""),
    file_object_key: String(file?.file_object_key || ""),
    video_stream_uid: String(file?.video_stream_uid || ""),
    sort_order: Number(file?.sort_order || 1),
    version: Number(file?.version || 0),
    created_at: String(file?.created_at || ""),
    updated_at: String(file?.updated_at || ""),
  }
}

function mergeFileDetail(id: string, detail: JsonRecord) {
  const index = files.value.findIndex((file) => fileId(file) === id)
  const base = index >= 0 ? files.value[index] : selected.value || {}
  const merged = { ...base, ...detail }
  if (index >= 0) {
    files.value.splice(index, 1, merged)
  }
  if (selected.value && fileId(selected.value) === id) {
    selected.value = merged
    fillForm(merged)
  }
}

async function loadFileDetail(file: JsonRecord | null) {
  if (!file) return
  const id = fileId(file)
  if (!id) return

  const requestId = ++detailRequestId
  detailLoading.value = true
  try {
    const detail = await apiClient<JsonRecord>(`/api/lms/resource-pack-files/${encodeURIComponent(id)}`)
    if (requestId !== detailRequestId) return
    if (detail && typeof detail === "object" && !Array.isArray(detail)) {
      mergeFileDetail(id, detail)
    }
  } catch (err) {
    console.error(err)
    if (requestId === detailRequestId) {
      toast.error(copy.value.toasts.detailLoadFailed)
    }
  } finally {
    if (requestId === detailRequestId) {
      detailLoading.value = false
    }
  }
}

async function loadPacks() {
  const data = await apiClient<JsonRecord>("/api/lms/resource-packs?page_size=100")
    total.value = Number(data.total) || 0
  packs.value = asRecordList(data.packs || data.items)
}

async function loadFiles() {
  const url = new URL("/api/lms/resource-pack-files", window.location.origin)
  url.searchParams.set("page_size", String(pageSize))
  if (packFilter.value) url.searchParams.set("pack_id", packFilter.value)
  if (pageToken.value) url.searchParams.set("page_token", pageToken.value)

  const data = await apiClient<JsonRecord>(`${url.pathname}${url.search}`)
  files.value = asRecordList(data.files || data.items)
  nextPageToken.value = String(data.next_page_token || "")
  const nextSelected = files.value.find((file) => fileId(file) === form.value.file_id) || files.value[0] || null
  selectFile(nextSelected)
}

async function load() {
  loading.value = true
  try {
    await loadPacks()
    await loadFiles()
  } catch (err) {
    console.error(err)
    packs.value = []
    files.value = []
    selected.value = null
    fillForm(null)
    toast.error(copy.value.toasts.loadFailed)
  } finally {
    loading.value = false
  }
}

function selectFile(file: JsonRecord | null, openDetail = false) {
  selected.value = file
  mode.value = openDetail ? "detail" : file ? "edit" : "create"
  fillForm(file)
  void loadFileDetail(file)
  if (openDetail) detailOpen.value = true
}

function openFileDetail(file: JsonRecord) {
  selectFile(file, true)
}

function openFileEditor(file: JsonRecord) {
  selectFile(file)
  mode.value = "edit"
  detailOpen.value = true
}

function closeFileDetail() {
  detailOpen.value = false
}

function requestDeleteFile(file: JsonRecord | null = selected.value) {
  if (!file) return
  selected.value = file
  fillForm(file)
  const id = fileId(file)
  const version = fileVersion(file)
  if (!id || version <= 0) {
    toast.error(copy.value.toasts.deleteRequiresVersion)
    return
  }
  deleteConfirmOpen.value = true
}

function closeDeleteConfirm() {
  deleteConfirmOpen.value = false
}

function startCreate() {
  detailRequestId += 1
  detailLoading.value = false
  selected.value = null
  mode.value = "create"
  fillForm(null)
  detailOpen.value = true
}

async function saveFile() {
  if (!form.value.title.trim()) {
    toast.error(copy.value.toasts.titleRequired)
    return
  }
  if (!form.value.file_type) {
    toast.error(copy.value.toasts.typeRequired)
    return
  }
  if (mode.value === "create" && !form.value.pack_id) {
    toast.error(copy.value.toasts.packRequired)
    return
  }
  if (mode.value === "edit" && (!form.value.file_id || form.value.version <= 0)) {
    toast.error(copy.value.toasts.updateRequiresVersion)
    return
  }

  saving.value = true
  try {
    const body: JsonRecord = {
      title: form.value.title.trim(),
      description: form.value.description,
      thumbnail_object_key: form.value.thumbnail_object_key,
      thumbnail_file_hash: form.value.thumbnail_file_hash,
      file_type: form.value.file_type,
      file_name: form.value.file_name,
      file_size: Number(form.value.file_size || 0),
      file_hash: form.value.file_hash,
      file_object_key: form.value.file_object_key,
      video_stream_uid: form.value.video_stream_uid,
      sort_order: Number(form.value.sort_order || 0),
    }

    if (mode.value === "create") {
      await apiClient(`/api/lms/resource-packs/${encodeURIComponent(form.value.pack_id)}/files`, {
        method: "POST",
        body: JSON.stringify(body),
      })
      toast.success(copy.value.toasts.created)
    } else {
      body.version = form.value.version
      await apiClient(`/api/lms/resource-pack-files/${encodeURIComponent(form.value.file_id)}`, {
        method: "PUT",
        body: JSON.stringify(body),
      })
      toast.success(copy.value.toasts.saved)
    }

    await loadFiles()
    detailOpen.value = false
    mode.value = "detail"
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.saveFailed))
  } finally {
    saving.value = false
  }
}

async function deleteFile() {
  if (!selected.value) return
  const id = fileId(selected.value)
  const version = fileVersion(selected.value)
  if (!id || version <= 0) {
    toast.error(copy.value.toasts.deleteRequiresVersion)
    return
  }

  saving.value = true
  try {
    await apiClient(`/api/lms/resource-pack-files/${encodeURIComponent(id)}?version=${version}`, {
      method: "DELETE",
    })
    toast.success(copy.value.toasts.deleted)
    deleteConfirmOpen.value = false
    detailOpen.value = false
    await loadFiles()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.deleteFailed))
  } finally {
    saving.value = false
  }
}

function resetPaging() {
  pageToken.value = ""
  nextPageToken.value = ""
  previousTokens.value = []
  currentPage.value = 1
}

function changePackFilter() {
  resetPaging()
  void loadFiles()
}

function previousPage() {
  if (!canPrevious.value) return
  pageToken.value = previousTokens.value.pop() || ""
  currentPage.value = Math.max(1, currentPage.value - 1)
  void loadFiles()
}

function nextPage() {
  if (!canNext.value) return
  previousTokens.value.push(pageToken.value)
  pageToken.value = nextPageToken.value
  currentPage.value += 1
  void loadFiles()
}

onMounted(load)
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight text-slate-950">{{ copy.title }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <div class="flex gap-3">
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" :disabled="loading" @click="load">
          <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
          <RefreshCw v-else class="h-4 w-4" />
          {{ copy.refresh }}
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl bg-blue-700 px-4 py-3 text-sm font-bold text-white shadow-sm" type="button" @click="startCreate">
          <Plus class="h-4 w-4" />
          {{ copy.newFile }}
        </button>
      </div>
    </header>

    <div class="rounded-2xl border border-slate-200 bg-white px-5 py-4 shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h2 class="text-lg font-black">{{ copy.filterTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.filterDescription }}</p>
        </div>
        <select v-model="packFilter" class="h-10 min-w-[360px] rounded-xl border border-slate-200 px-3 text-sm font-bold" @change="changePackFilter">
          <option value="">{{ copy.allPacks }}</option>
          <option v-for="pack in packs" :key="packId(pack)" :value="packId(pack)">{{ copy.ownerText(packTitle(pack), packId(pack)) }}</option>
        </select>
      </div>
    </div>

    <div class="grid gap-6">
      <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
        <div class="flex items-center justify-between border-b border-slate-200 px-5 py-4">
          <div>
            <h2 class="text-xl font-black">{{ copy.listTitle }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
          </div>
          
          </div>

        <div v-if="loading" class="px-6 py-10 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          {{ copy.loading }}
        </div>
        <div v-else-if="!files.length" class="px-6 py-10 text-center text-slate-500">{{ copy.empty }}</div>
        <div v-else class="grid grid-cols-[minmax(0,1fr)_72px_130px_280px] gap-5 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-500">
          <span>{{ copy.columns.file }}</span>
          <span class="text-center">{{ copy.columns.sort }}</span>
          <span class="text-center">{{ copy.columns.type }}</span>
          <span class="text-center">{{ copy.columns.action }}</span>
        </div>
        <div
          v-for="file in files"
          :key="fileId(file)"
          class="grid w-full grid-cols-[minmax(0,1fr)_72px_130px_280px] gap-5 border-b border-slate-100 px-5 py-4 text-left transition last:border-b-0 hover:bg-slate-50"
          :class="fileId(selected) === fileId(file) ? 'bg-sky-50' : ''"
        >
          <button class="min-w-0 text-left" type="button" @click="openFileDetail(file)">
            <div class="truncate text-lg font-black text-slate-950">{{ fileTitle(file) }}</div>
            <div class="mt-1 line-clamp-2 text-sm text-slate-500">{{ file.description || "-" }}</div>
            <div class="mt-2 flex flex-wrap items-center gap-2 text-xs font-bold text-slate-500">
              <span class="max-w-full truncate rounded-full bg-blue-50 px-2 py-1 text-blue-700">{{ copy.ownerPrefix }}{{ ownerText(file) }}</span>
              <span class="rounded-full bg-slate-100 px-2 py-1">Version: {{ file.version || 0 }}</span>
            </div>
          </button>
          <span class="self-center text-center text-sm font-black text-slate-700">{{ file.sort_order || 0 }}</span>
          <span class="h-fit self-center justify-self-center whitespace-nowrap rounded-full border border-slate-200 bg-white px-3 py-1 text-xs font-black text-slate-700">
            {{ fileTypeLabel(file.file_type) }}
          </span>
          <div class="flex min-w-0 items-center justify-end gap-4 whitespace-nowrap">
            <button class="text-sm font-bold text-[#1890ff] transition hover:underline" type="button" @click="openFileDetail(file)">
              {{ copy.viewDetails }}
            </button>
            <button class="text-sm font-bold text-[#ffba00] transition hover:underline" type="button" @click="openFileEditor(file)">
              {{ copy.editFile }}
            </button>
            <button class="text-sm font-bold text-[#ff4949] transition hover:underline" type="button" @click="requestDeleteFile(file)">
              {{ copy.deleteFile }}
            </button>
          </div>
        </div>

        <div class="flex items-center justify-end gap-3 border-t border-slate-200 px-5 py-4">
          <span class="mr-auto text-sm font-bold text-slate-500">{{ `${currentPage} / ${Math.max(1, Math.ceil(total / pageSize))}` }}</span>
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrevious || loading" @click="previousPage">{{ copy.prev }}</button>
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext || loading" @click="nextPage">{{ copy.next }}</button>
        </div>
      </section>

      <section v-if="detailOpen" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-6">
        <div class="flex max-h-[88vh] w-full max-w-[1100px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
        <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-5 py-4">
          <div>
            <h2 class="text-xl font-black">{{ mode === "create" ? copy.createTitle : mode === "edit" ? copy.editTitle : copy.detailTitle }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ mode === "detail" ? copy.readonlyHint : mode === "create" ? copy.createHint : copy.editHint }}</p>
            <p v-if="detailLoading" class="mt-1 inline-flex items-center gap-2 text-xs font-bold text-blue-600">
              <Loader2 class="h-3.5 w-3.5 animate-spin" />
              {{ copy.detailLoading }}
            </p>
          </div>
          <button class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="closeFileDetail">
            <X class="h-5 w-5" />
          </button>
        </div>

        <div class="flex-1 overflow-y-auto p-5">
          <div class="mb-5 rounded-2xl border border-blue-100 bg-blue-50/70 p-4">
            <div class="text-xs font-black uppercase text-blue-600">{{ copy.ownerCardTitle }}</div>
            <div class="mt-1 text-lg font-black text-slate-950">{{ selectedPack ? packTitle(selectedPack) : copy.selectPack }}</div>
            <div class="mt-1 break-all text-sm font-bold text-blue-700">{{ form.pack_id || "-" }}</div>
          </div>

          <div class="mb-3 text-sm font-black text-slate-700">{{ copy.sections.basic }}</div>
          <div class="grid gap-3 md:grid-cols-2">
          <div v-if="mode !== 'create'" class="text-sm font-bold">
            {{ copy.fields.fileId }}
            <div class="readonly-field readonly-field--compact">{{ displayValue(form.file_id) }}</div>
          </div>
          <label class="block">
            <span class="block text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.fields.pack }}</span>
            <div v-if="mode === 'detail'" class="mt-2 readonly-field">{{ packFieldText(form.pack_id) }}</div>
            <select v-else v-model="form.pack_id" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3 disabled:bg-slate-100" :disabled="mode === 'edit'">
              <option value="">{{ copy.selectPack }}</option>
              <option v-for="pack in packs" :key="packId(pack)" :value="packId(pack)">{{ copy.ownerText(packTitle(pack), packId(pack)) }}</option>
            </select>
          </label>
          <label class="block">
            <span class="block text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.fields.title }}</span>
            <div v-if="mode === 'detail'" class="mt-2 readonly-field">{{ displayValue(form.title) }}</div>
            <input v-else v-model="form.title" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.placeholders.title" />
          </label>
          <label class="block">
            <span class="block text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.fields.fileType }}</span>
            <div v-if="mode === 'detail'" class="mt-2 readonly-field readonly-field--compact">{{ fileTypeLabel(form.file_type) }}</div>
            <select v-else v-model.number="form.file_type" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3">
              <option v-for="option in fileTypeOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
            </select>
          </label>
          <label class="md:col-span-2 text-sm font-bold">
            {{ copy.fields.description }}
            <div v-if="mode === 'detail'" class="readonly-field readonly-field--textarea">{{ displayValue(form.description) }}</div>
            <textarea v-else v-model="form.description" class="mt-2 min-h-20 w-full rounded-xl border border-slate-200 px-3 py-2" :placeholder="copy.placeholders.description" />
          </label>
          </div>

          <div class="mb-3 mt-5 border-t border-slate-100 pt-5 text-sm font-black text-slate-700">{{ copy.sections.fileThumbnail }}</div>
          <div class="grid gap-3 md:grid-cols-2">
          <label class="text-sm font-bold">
            {{ copy.fields.fileName }}
            <div v-if="mode === 'detail'" class="readonly-field">{{ displayValue(form.file_name) }}</div>
            <input v-else v-model="form.file_name" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.placeholders.fileName" />
          </label>
          <label class="text-sm font-bold">
            {{ copy.fields.fileSize }}
            <div v-if="mode === 'detail'" class="readonly-field readonly-field--compact">{{ displayValue(form.file_size) }}</div>
            <input v-else v-model.number="form.file_size" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" type="number" min="0" />
          </label>
          <label class="text-sm font-bold">
            {{ copy.fields.fileObjectKey }}
            <div v-if="mode === 'detail'" class="readonly-field readonly-field--long">{{ displayValue(form.file_object_key) }}</div>
            <input v-else v-model="form.file_object_key" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.placeholders.fileObjectKey" />
          </label>
          <label class="text-sm font-bold">
            {{ copy.fields.fileHash }}
            <div v-if="mode === 'detail'" class="readonly-field readonly-field--long">{{ displayValue(form.file_hash) }}</div>
            <input v-else v-model="form.file_hash" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.placeholders.fileHash" />
          </label>
          <label class="text-sm font-bold">
            {{ copy.fields.thumbnailObjectKey }}
            <div v-if="mode === 'detail'" class="readonly-field readonly-field--long">{{ displayValue(form.thumbnail_object_key) }}</div>
            <input v-else v-model="form.thumbnail_object_key" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.placeholders.thumbnailObjectKey" />
          </label>
          <label class="text-sm font-bold">
            {{ copy.fields.thumbnailHash }}
            <div v-if="mode === 'detail'" class="readonly-field readonly-field--long">{{ displayValue(form.thumbnail_file_hash) }}</div>
            <input v-else v-model="form.thumbnail_file_hash" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.placeholders.thumbnailHash" />
          </label>
          </div>

          <div class="mb-3 mt-5 border-t border-slate-100 pt-5 text-sm font-black text-slate-700">{{ copy.sections.orderVersion }}</div>
          <div class="grid gap-3 md:grid-cols-3">
          <label class="text-sm font-bold">
            {{ copy.fields.videoStreamUid }}
            <div v-if="mode === 'detail'" class="readonly-field readonly-field--compact">{{ displayValue(form.video_stream_uid) }}</div>
            <input v-else v-model="form.video_stream_uid" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.placeholders.videoStreamUid" />
          </label>
          <label class="text-sm font-bold">
            {{ copy.fields.sort }}
            <div v-if="mode === 'detail'" class="readonly-field readonly-field--compact">{{ displayValue(form.sort_order) }}</div>
            <input v-else v-model.number="form.sort_order" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" type="number" min="0" />
          </label>
          <label class="text-sm font-bold">
            {{ copy.fields.version }}
            <div v-if="mode === 'detail'" class="readonly-field readonly-field--compact">{{ displayValue(form.version) }}</div>
            <input v-else v-model.number="form.version" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3 disabled:bg-slate-100" type="number" min="0" :disabled="mode === 'create'" />
          </label>
          <div v-if="mode !== 'create'" class="text-sm font-bold">
            {{ copy.fields.createdAt }}
            <div class="readonly-field readonly-field--compact">{{ formatDate(form.created_at) || "-" }}</div>
          </div>
          <div v-if="mode !== 'create'" class="text-sm font-bold">
            {{ copy.fields.updatedAt }}
            <div class="readonly-field readonly-field--compact">{{ formatDate(form.updated_at) || "-" }}</div>
          </div>
          </div>
        </div>

        <div v-if="mode !== 'detail'" class="flex shrink-0 justify-end border-t border-slate-200 bg-white px-5 py-4">
          <button class="inline-flex h-10 min-w-[180px] items-center justify-center gap-2 rounded-xl bg-blue-700 px-4 font-bold text-white disabled:opacity-50" type="button" :disabled="saving" @click="saveFile">
            <Loader2 v-if="saving" class="h-4 w-4 animate-spin" />
            <Save v-else class="h-4 w-4" />
            {{ mode === "create" ? copy.createFile : copy.saveFile }}
          </button>
        </div>
        </div>
      </section>

      <Teleport to="body">
        <div v-if="deleteConfirmOpen && selected" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-6">
          <section class="w-full max-w-[460px] rounded-3xl bg-white p-6 shadow-2xl">
            <h2 class="text-2xl font-black text-slate-950">{{ copy.deleteConfirmTitle }}</h2>
            <p class="mt-3 text-sm font-semibold text-slate-500">{{ copy.deleteConfirmDescription }}</p>
            <div class="mt-5 rounded-2xl bg-slate-50 p-4">
              <div class="break-words font-black text-slate-950">{{ fileTitle(selected) }}</div>
              <div class="mt-1 break-all text-sm font-semibold text-slate-500">{{ fileId(selected) }}</div>
            </div>
            <div class="mt-6 flex justify-end gap-3">
              <button class="rounded-xl border border-slate-900 px-5 py-3 font-bold text-slate-950 disabled:opacity-50" type="button" :disabled="saving" @click="closeDeleteConfirm">{{ copy.cancel }}</button>
              <button class="rounded-xl bg-red-600 px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving" @click="deleteFile">
                {{ saving ? copy.deleting : copy.confirmDeleteAction }}
              </button>
            </div>
          </section>
        </div>
      </Teleport>
    </div>
  </section>
</template>

<style scoped>
.readonly-field {
  min-height: 2.5rem;
  margin-top: 0.5rem;
  width: 100%;
  border: 1px solid #e2e8f0;
  border-radius: 0.75rem;
  background: #f1f5f9;
  padding: 0.625rem 0.75rem;
  color: #0f172a;
  font-weight: 700;
  line-height: 1.45;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.readonly-field--compact {
  display: flex;
  align-items: center;
}

.readonly-field--long {
  min-height: 4.75rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 0.8125rem;
  line-height: 1.45;
  word-break: break-all;
}

.readonly-field--textarea {
  min-height: 5rem;
}
</style>
