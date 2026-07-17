<script setup lang="ts">
import { Info, Loader2, Plus, RefreshCw, Save, X, UploadCloud } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { ApiError, apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"

const pageSize = 10

const packs = ref<JsonRecord[]>([])
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
let detailRequestId = 0
const { t } = useAdminLanguage()
const copy = computed(() => t.value.resourcePacksAdmin)

const uploadingThumbnail = ref(false)
const thumbnailFileInput = ref<HTMLInputElement | null>(null)

const form = ref({
  pack_id: "",
  title: "",
  description: "",
  thumbnail_object_key: "",
  thumbnail_file_hash: "",
  respath: "",
  status: "Active",
  version: 0,
  icon: "",
  category: "",
  created_at: "",
  updated_at: "",
})

const canPrevious = computed(() => previousTokens.value.length > 0)
const canNext = computed(() => !!nextPageToken.value)

function asRecordList(value: unknown) {
  return Array.isArray(value) ? value.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item)) : []
}

function packId(pack: JsonRecord | null) {
  return String(pack?.pack_id || "")
}

function packTitle(pack: JsonRecord | null) {
  return String(pack?.title || pack?.pack_id || copy.value.unnamed)
}

function packVersion(pack: JsonRecord | null) {
  return Number(pack?.version || 0)
}

function packStatusKey(pack: JsonRecord | null) {
  return String(pack?.status || "").trim().toUpperCase()
}

function packStatusLabel(pack: JsonRecord | null) {
  const status = packStatusKey(pack)
  if (status.includes("DRAFT")) return copy.value.statusLabels.draft
  if (status.includes("DEPRECATED")) return copy.value.statusLabels.deprecated
  if (status.includes("ACTIVE") || status.includes("PUBLISHED")) return copy.value.statusLabels.active
  return String(pack?.status || copy.value.statusLabels.unknown)
}

function packStatusClass(pack: JsonRecord | null) {
  const status = packStatusKey(pack)
  if (status.includes("DRAFT")) return "border-amber-200 bg-amber-50 text-amber-700"
  if (status.includes("DEPRECATED")) return "border-slate-200 bg-slate-100 text-slate-600"
  if (status.includes("ACTIVE") || status.includes("PUBLISHED")) return "border-emerald-200 bg-emerald-50 text-emerald-700"
  return "border-slate-200 bg-white text-slate-600"
}

function canPublishPack(pack: JsonRecord | null) {
  const status = packStatusKey(pack)
  return status.includes("DRAFT")
}

function canRevertPack(pack: JsonRecord | null) {
  const status = packStatusKey(pack)
  return status.includes("ACTIVE") || status.includes("PUBLISHED")
}

function canDeletePack(pack: JsonRecord | null) {
  return !!pack && !packStatusKey(pack).includes("DEPRECATED")
}

function fillForm(pack: JsonRecord | null) {
  form.value = {
    pack_id: String(pack?.pack_id || ""),
    title: String(pack?.title || ""),
    description: String(pack?.description || ""),
    thumbnail_object_key: String(pack?.thumbnail_object_key || ""),
    thumbnail_file_hash: String(pack?.thumbnail_file_hash || ""),
    respath: String(pack?.respath || ""),
    status: String(pack?.status || (mode.value === "create" ? "Draft" : "Active")),
    version: Number(pack?.version || 0),
    icon: String(pack?.icon || ""),
      category: String(pack?.category || (mode.value === "create" ? "public" : "")),
      created_at: String(pack?.created_at || ""),
      updated_at: String(pack?.updated_at || ""),
  }
}

function mergePackDetail(id: string, detail: JsonRecord) {
  const index = packs.value.findIndex((pack) => packId(pack) === id)
  const base = index >= 0 ? packs.value[index] : selected.value || {}
  const merged = { ...base, ...detail }
  if (index >= 0) {
    packs.value.splice(index, 1, merged)
  }
  if (selected.value && packId(selected.value) === id) {
    selected.value = merged
    fillForm(merged)
  }
}

async function loadPackDetail(pack: JsonRecord | null) {
  if (!pack) return
  const id = packId(pack)
  if (!id) return

  const requestId = ++detailRequestId
  detailLoading.value = true
  try {
    const detail = await apiClient<JsonRecord>(`/api/lms/resource-packs/${encodeURIComponent(id)}`)
    if (requestId !== detailRequestId) return
    if (detail && typeof detail === "object" && !Array.isArray(detail)) {
      mergePackDetail(id, detail)
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

async function load() {
  loading.value = true
  try {
    const url = new URL("/api/lms/resource-packs", window.location.origin)
    url.searchParams.set("page_size", String(pageSize))
    if (pageToken.value) url.searchParams.set("page_token", pageToken.value)

    const data = await apiClient<JsonRecord>(`${url.pathname}${url.search}`)
    total.value = Number(data.total) || 0
    packs.value = asRecordList(data.packs || data.items)
    nextPageToken.value = String(data.next_page_token || "")
    const nextSelected = packs.value.find((pack) => packId(pack) === form.value.pack_id) || packs.value[0] || null
    selectPack(nextSelected)
  } catch (err) {
    console.error(err)
    packs.value = []
    selected.value = null
    fillForm(null)
    toast.error(copy.value.toasts.listLoadFailed)
  } finally {
    loading.value = false
  }
}

function selectPack(pack: JsonRecord | null, openDetail = false) {
  selected.value = pack
  mode.value = openDetail ? "detail" : pack ? "edit" : "create"
  fillForm(pack)
  void loadPackDetail(pack)
  if (openDetail) detailOpen.value = true
}

watch(() => mode.value, (m) => {
  if (m === "create" && !form.value.status) form.value.status = "Draft"
})

watch(() => form.value.title, (newTitle, oldTitle) => {
  if (mode.value !== "create") return
  const cat = form.value.category || "public"
  const generatePath = (t: string) => t ? `/res-packages/${cat}/${t.trim().toLowerCase().replace(/\s+/g, '_')}` : ''
  const newPath = generatePath(newTitle)
  const oldPath = generatePath(oldTitle)
  if (!form.value.respath || form.value.respath === oldPath) {
    form.value.respath = newPath
  }
})

watch(() => form.value.category, (newCat, oldCat) => {
  if (mode.value !== "create") return
  const t = form.value.title
  if (!t) return
  const generatePath = (c: string) => c ? `/res-packages/${c}/${t.trim().toLowerCase().replace(/\s+/g, '_')}` : ''
  const newPath = generatePath(newCat)
  const oldPath = generatePath(oldCat)
  if (!form.value.respath || form.value.respath === oldPath) {
    form.value.respath = newPath
  }
})

function openPackDetail(pack: JsonRecord) {
  selectPack(pack, true)
}

function openPackEditor(pack: JsonRecord) {
  selectPack(pack)
  mode.value = "edit"
  detailOpen.value = true
}

function closePackDetail() {
  detailOpen.value = false
}

function requestDeletePack(pack: JsonRecord | null = selected.value) {
  if (!pack) return
  if (!canDeletePack(pack)) return
  selected.value = pack
  fillForm(pack)
  const id = packId(pack)
  const version = packVersion(pack)
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

function validatePackForm() {
  if (!form.value.title.trim()) {
    toast.error(copy.value.toasts.titleRequired)
    return false
  }
  if (!form.value.respath.trim()) {
    toast.error(copy.value.toasts.respathRequired)
    return false
  }
  if (mode.value === "edit" && (!form.value.pack_id || form.value.version <= 0)) {
    toast.error(copy.value.toasts.updateRequiresVersion)
    return false
  }
  return true
}

function packSaveErrorMessage(err: unknown) {
  if (!(err instanceof ApiError)) return copy.value.toasts.saveFailed

  const message = String(err.message || "").toLowerCase()
  if (err.status === 409 || (message.includes("resource pack") && message.includes("already exists"))) {
    return copy.value.toasts.duplicatePack
  }
  if (err.status === 400 || message.includes("invalid_request")) {
    return copy.value.toasts.invalidRequest
  }
  return copy.value.toasts.saveFailed
}

function packActionErrorMessage(err: unknown, action: "publish" | "revert-to-draft") {
  if (!(err instanceof ApiError)) return copy.value.toasts.actionFailed

  const message = String(err.message || "").toLowerCase()
  if (action === "publish" && message.includes("thumbnail")) {
    return copy.value.toasts.publishThumbnailRequired
  }
  if (message.includes("precondition") || err.status === 409 || err.status === 412) {
    return copy.value.toasts.preconditionFailed
  }
  if (err.status === 400 || message.includes("invalid_request")) {
    return copy.value.toasts.invalidRequest
  }
  return copy.value.toasts.actionFailed
}

function packDuplicateErrorMessage(err: unknown) {
  if (!(err instanceof ApiError)) return copy.value.toasts.duplicateFailed

  const message = String(err.message || "").toLowerCase()
  if (err.status === 409 || message.includes("already exists")) {
    return copy.value.toasts.duplicatePack
  }
  if (err.status === 400 || message.includes("invalid_request")) {
    return copy.value.toasts.invalidRequest
  }
  return copy.value.toasts.duplicateFailed
}

async function savePack() {
  if (!validatePackForm()) return

  saving.value = true
  try {
    const body: JsonRecord = {
      title: form.value.title.trim(),
      description: form.value.description.trim(),
      thumbnail_object_key: form.value.thumbnail_object_key.trim(),
      thumbnail_file_hash: form.value.thumbnail_file_hash.trim(),
      respath: form.value.respath.trim(),
      icon: form.value.icon.trim(),
      category: form.value.category.trim(),
    }

    if (mode.value === "create") {
      await apiClient("/api/lms/resource-packs", {
        method: "POST",
        body: JSON.stringify(body),
      })
      toast.success(copy.value.toasts.created)
    } else {
      body.version = form.value.version
      await apiClient(`/api/lms/resource-packs/${encodeURIComponent(form.value.pack_id)}`, {
        method: "PUT",
        body: JSON.stringify(body),
      })
      toast.success(copy.value.toasts.saved)
    }

    await load()
    detailOpen.value = false
    mode.value = "detail"
  } catch (err) {
    console.error(err)
    toast.error(packSaveErrorMessage(err))
  } finally {
    saving.value = false
  }
}

async function runPackAction(pack: JsonRecord | null, action: "publish" | "revert-to-draft") {
  const id = packId(pack)
  const version = packVersion(pack)
  if (!id || version <= 0) {
    toast.error(copy.value.toasts.actionRequiresVersion)
    return
  }

  saving.value = true
  try {
    await apiClient(`/api/lms/resource-packs/${encodeURIComponent(id)}/${action}`, {
      method: "POST",
      body: JSON.stringify({ version }),
    })
    if (action === "publish") toast.success(copy.value.toasts.published)
    if (action === "revert-to-draft") toast.success(copy.value.toasts.reverted)
    await load()
  } catch (err) {
    console.error(err)
    toast.error(packActionErrorMessage(err, action))
  } finally {
    saving.value = false
  }
}

async function duplicatePack(pack: JsonRecord | null) {
  const id = packId(pack)
  if (!id) {
    toast.error(copy.value.toasts.duplicateRequiresId)
    return
  }

  saving.value = true
  try {
    await apiClient(`/api/lms/resource-packs/${encodeURIComponent(id)}/duplicate`, {
      method: "POST",
      body: JSON.stringify({ title: copy.value.duplicateTitle(packTitle(pack)) }),
    })
    toast.success(copy.value.toasts.duplicated)
    await load()
  } catch (err) {
    console.error(err)
    toast.error(packDuplicateErrorMessage(err))
  } finally {
    saving.value = false
  }
}

async function deletePack() {
  if (!selected.value) return
  const id = packId(selected.value)
  const version = packVersion(selected.value)
  if (!id || version <= 0) {
    toast.error(copy.value.toasts.deleteRequiresVersion)
    return
  }

  saving.value = true
  try {
    await apiClient(`/api/lms/resource-packs/${encodeURIComponent(id)}?version=${version}`, {
      method: "DELETE",
    })
    toast.success(copy.value.toasts.deleted)
    deleteConfirmOpen.value = false
    detailOpen.value = false
    await load()
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.deleteFailed)
  } finally {
    saving.value = false
  }
}

async function handleThumbnailFileUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (!form.value.pack_id) return

  uploadingThumbnail.value = true
  try {
    const arrayBuffer = await file.arrayBuffer()
    const hashBuffer = await crypto.subtle.digest('SHA-256', arrayBuffer)
    const hashArray = Array.from(new Uint8Array(hashBuffer))
    const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('')

    const uploadUrlReq = {
      upload_type: 5,
      pack_id: form.value.pack_id,
      file_name: file.name,
      content_type: file.type || "application/octet-stream",
      file_hash: hashHex
    }
    const uploadRes = await apiClient<JsonRecord>("/api/lms/upload-url", { method: "POST", body: JSON.stringify(uploadUrlReq) })
    
    if (!uploadRes.upload_url) throw new Error("Missing upload URL")
    const uploadResponse = await fetch(String(uploadRes.upload_url), {
      method: "PUT",
      body: file,
      headers: uploadRes.signed_headers as Record<string, string> || {}
    })
    if (!uploadResponse.ok) {
      throw new Error(`Upload failed: ${uploadResponse.status}`)
    }

    form.value.thumbnail_object_key = String(uploadRes.object_key)
    form.value.thumbnail_file_hash = hashHex
    toast.success(copy.value.toasts.thumbnailUploadSuccess)
  } catch (err: any) {
    toast.error(copy.value.toasts.thumbnailUploadFailed(err.message || String(err)))
  } finally {
    uploadingThumbnail.value = false
    if (input) input.value = ""
  }
}

function previousPage() {
  if (!canPrevious.value) return
  pageToken.value = previousTokens.value.pop() || ""
  currentPage.value = Math.max(1, currentPage.value - 1)
  void load()
}

function nextPage() {
  if (!canNext.value) return
  previousTokens.value.push(pageToken.value)
  pageToken.value = nextPageToken.value
  currentPage.value += 1
  void load()
}

watch(() => form.value.title, (newTitle) => {
  if (mode.value !== 'create') return
  if (newTitle && !form.value.respath) {
    form.value.respath = `/res-packages/${newTitle.toLowerCase().replace(/[^a-z0-9]/g, "-").replace(/-+/g, "-").replace(/^-|-$/g, "")}`
  }
})

onMounted(load)
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-5 px-4 py-5 md:gap-6 md:px-8 md:py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div class="min-w-0">
        <h1 class="text-3xl font-black tracking-tight text-slate-950 md:text-4xl">{{ copy.title }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" :disabled="loading" @click="load">
          <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
          <RefreshCw v-else class="h-4 w-4" />
          {{ copy.refresh }}
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl bg-blue-700 px-4 py-3 text-sm font-bold text-white shadow-sm" type="button" @click="startCreate">
          <Plus class="h-4 w-4" />
          {{ copy.newPack }}
        </button>
      </div>
    </header>

    <div class="grid gap-6">
      <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 px-4 py-4 md:p-5">
          <div class="min-w-0">
            <h2 class="text-xl font-black">{{ copy.listTitle }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
          </div>
          
          </div>

        <div v-if="loading" class="px-4 py-10 text-center text-slate-500 md:p-12">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          {{ copy.loading }}
        </div>
        <div v-else-if="!packs.length" class="px-4 py-10 text-center text-slate-500 md:p-12">{{ copy.empty }}</div>
        <div v-else>
          <div class="hidden grid-cols-[minmax(0,1fr)_84px_120px_300px] gap-6 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-400 lg:grid">
            <span>{{ copy.columns.pack }}</span>
            <span>{{ copy.columns.version }}</span>
            <span class="text-center">{{ copy.columns.status }}</span>
            <span class="text-center">{{ copy.columns.action }}</span>
          </div>
          <div
            v-for="pack in packs"
            :key="packId(pack)"
            class="grid w-full gap-3 border-b border-slate-100 px-4 py-4 text-left transition last:border-b-0 hover:bg-slate-50 md:px-5 md:py-3 lg:grid-cols-[minmax(0,1fr)_84px_120px_300px] lg:items-center lg:gap-6"
            :class="packId(selected) === packId(pack) ? 'bg-sky-50/70' : ''"
          >
            <div class="min-w-0">
              <div class="break-words text-base font-black text-slate-950 lg:truncate">{{ packTitle(pack) }}</div>
              <div class="mt-1 line-clamp-1 text-sm text-slate-500">{{ pack.description || "-" }}</div>
              <div class="mt-1 break-all font-mono text-xs text-slate-500 lg:truncate">{{ copy.fields.idPrefix }}{{ pack.pack_id }}</div>
            </div>
            <div class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2 text-sm font-bold text-slate-700 lg:block lg:rounded-none lg:bg-transparent lg:p-0">
              <span class="text-xs font-bold text-slate-400 lg:hidden">{{ copy.fields.version }}</span>
              <span>{{ pack.version || 0 }}</span>
            </div>
            <span class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2 lg:block lg:justify-self-center lg:rounded-none lg:bg-transparent lg:p-0">
              <span class="text-xs font-bold text-slate-400 lg:hidden">{{ copy.columns.status }}</span>
              <span class="inline-flex rounded-full border px-3 py-1 text-xs font-black" :class="packStatusClass(pack)" :title="String(pack.status || '')">
                {{ packStatusLabel(pack) }}
              </span>
            </span>
            <div class="grid grid-cols-2 gap-3 sm:flex sm:flex-wrap sm:items-center sm:justify-start lg:justify-center">
              <button class="inline-flex items-center justify-center rounded-xl border border-blue-100 bg-blue-50 px-3 py-2 text-sm font-bold text-blue-700 transition hover:underline lg:border-0 lg:bg-transparent lg:px-0 lg:py-0" type="button" @click="openPackDetail(pack)">
                {{ copy.viewDetails }}
              </button>
              <button class="inline-flex items-center justify-center rounded-xl border border-amber-100 bg-amber-50 px-3 py-2 text-sm font-bold text-amber-600 transition hover:underline lg:border-0 lg:bg-transparent lg:px-0 lg:py-0" type="button" @click="openPackEditor(pack)">
                {{ copy.editPack }}
              </button>
              <button class="inline-flex items-center justify-center rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-bold text-slate-700 transition hover:underline disabled:opacity-50 lg:border-0 lg:bg-transparent lg:px-0 lg:py-0" type="button" :disabled="saving" @click.stop="duplicatePack(pack)">
                {{ copy.copyPack }}
              </button>
              <button v-if="canPublishPack(pack)" class="inline-flex items-center justify-center rounded-xl border border-emerald-100 bg-emerald-50 px-3 py-2 text-sm font-bold text-emerald-700 transition hover:underline disabled:opacity-50 lg:border-0 lg:bg-transparent lg:px-0 lg:py-0" type="button" :disabled="saving" @click.stop="runPackAction(pack, 'publish')">
                {{ copy.publishPack }}
              </button>
              <button v-if="canRevertPack(pack)" class="inline-flex items-center justify-center rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-bold text-slate-700 transition hover:underline disabled:opacity-50 lg:border-0 lg:bg-transparent lg:px-0 lg:py-0" type="button" :disabled="saving" @click.stop="runPackAction(pack, 'revert-to-draft')">
                {{ copy.revertPack }}
              </button>
              <button v-if="canDeletePack(pack)" class="inline-flex items-center justify-center rounded-xl border border-red-100 bg-red-50 px-3 py-2 text-sm font-bold text-red-600 transition hover:underline lg:border-0 lg:bg-transparent lg:px-0 lg:py-0" type="button" @click="requestDeletePack(pack)">
                {{ copy.deletePack }}
              </button>
            </div>
          </div>
        </div>

        <div class="flex flex-col items-stretch justify-end gap-3 border-t border-slate-200 px-4 py-4 sm:flex-row sm:items-center md:p-5">
          <span class="text-center text-sm font-bold text-slate-500 sm:mr-auto sm:text-left">{{ `${currentPage} / ${Math.max(1, Math.ceil(total / pageSize))}` }}</span>
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrevious || loading" @click="previousPage">{{ copy.prev }}</button>
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext || loading" @click="nextPage">{{ copy.next }}</button>
        </div>
      </section>

      <section v-if="detailOpen" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-0 md:p-6">
        <div class="flex h-full max-h-none w-full max-w-[1100px] flex-col overflow-hidden rounded-none bg-white shadow-2xl md:h-auto md:max-h-[88vh] md:rounded-3xl">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-4 py-4 md:p-5">
            <div class="min-w-0">
              <h2 class="break-words text-xl font-black">{{ mode === "create" ? copy.createTitle : mode === "edit" ? copy.editTitle : copy.detailTitle }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ mode === "detail" ? copy.readonlyHint : mode === "create" ? copy.createHint : copy.editHint }}</p>
              <p v-if="detailLoading" class="mt-1 inline-flex items-center gap-2 text-xs font-bold text-blue-600">
                <Loader2 class="h-3.5 w-3.5 animate-spin" />
                {{ copy.detailLoading }}
              </p>
            </div>
            <button class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="closePackDetail">
              <X class="h-5 w-5" />
            </button>
          </div>

          <div class="min-h-0 flex-1 space-y-5 overflow-y-auto p-4 md:p-5">
            <template v-if="mode === 'detail'">
              <div>
                <div class="mb-3 text-sm font-black text-slate-950">{{ copy.sections.basic }}</div>
                <div class="grid gap-4 md:grid-cols-2">
                  <div class="text-sm font-bold">
                    {{ copy.fields.title }}
                    <div class="mt-2 min-h-10 break-words rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 font-semibold text-slate-700">{{ form.title || "-" }}</div>
                  </div>
                  <div class="text-sm font-bold">
                    {{ copy.fields.category }}
                    <div class="mt-2 min-h-10 break-words rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 font-semibold text-slate-700">{{ form.category || "-" }}</div>
                  </div>
                  <div class="text-sm font-bold">
                    {{ copy.fields.icon }}
                    <div class="mt-2 min-h-10 break-words rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 font-semibold text-slate-700">{{ form.icon || "-" }}</div>
                  </div>
                  <div class="text-sm font-bold md:col-span-2">
                    {{ copy.fields.description }}
                    <div class="mt-2 min-h-24 whitespace-pre-wrap break-words rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 font-semibold text-slate-700">{{ form.description || "-" }}</div>
                  </div>
                </div>
              </div>

              <div class="border-t border-slate-100 pt-5">
                <div class="mb-3 text-sm font-black text-slate-950">{{ copy.sections.pathThumbnail }}</div>
                <div class="grid gap-4 md:grid-cols-2">
                  <div class="text-sm font-bold">
                    {{ copy.fields.respath }}
                    <div class="readonly-long-field">{{ form.respath || "-" }}</div>
                  </div>
                  <div class="text-sm font-bold">
                    {{ copy.fields.thumbnailObjectKey }}
                    <div class="readonly-long-field">{{ form.thumbnail_object_key || "-" }}</div>
                  </div>
                  <div class="text-sm font-bold md:col-span-2">
                    {{ copy.fields.thumbnailFileHash }}
                    <div class="readonly-long-field">{{ form.thumbnail_file_hash || "-" }}</div>
                  </div>
                </div>
              </div>

              <div class="border-t border-slate-100 pt-5">
                <div class="mb-3 text-sm font-black text-slate-950">{{ copy.sections.system }}</div>
                <div class="grid gap-4 md:grid-cols-3">
                  <div class="text-sm font-bold">
                    {{ copy.fields.packId }}
                    <div class="mt-2 min-h-10 break-all rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 font-semibold text-slate-700">{{ form.pack_id || "-" }}</div>
                  </div>
                  <div class="text-sm font-bold">
                    {{ copy.fields.status }}
                    <div class="mt-2 min-h-10 break-words rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 font-semibold text-slate-700">
                      <span class="inline-flex rounded-full border px-2 py-0.5 text-xs font-black" :class="packStatusClass(form)" :title="String(form.status || '')">
                        {{ packStatusLabel(form) }}
                      </span>
                    </div>
                  </div>
                  <div class="text-sm font-bold">
                    {{ copy.fields.version }}
                    <div class="mt-2 min-h-10 break-words rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 font-semibold text-slate-700">{{ form.version || 0 }}</div>
                  </div>
                  <div class="text-sm font-bold">
                    {{ copy.fields.createdAt }}
                    <div class="mt-2 min-h-10 break-words rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 font-semibold text-slate-700">{{ formatDate(form.created_at) || "-" }}</div>
                  </div>
                  <div class="text-sm font-bold">
                    {{ copy.fields.updatedAt }}
                    <div class="mt-2 min-h-10 break-words rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 font-semibold text-slate-700">{{ formatDate(form.updated_at) || "-" }}</div>
                  </div>
                </div>
              </div>
            </template>

            <template v-else>
              <div>
                <div class="mb-3 text-sm font-black text-slate-950">{{ copy.sections.basic }}</div>
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="block">
                    <span class="text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.fields.title }}</span>
                    <input v-model="form.title" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2" :placeholder="copy.placeholders.title" required />
                  </label>
                  <label class="block">
                    <span class="flex items-center gap-1.5 text-sm font-bold">
                      {{ copy.fields.category }}
                      <span class="group relative inline-flex cursor-help rounded-full text-slate-600 outline-none transition-colors hover:text-slate-900 focus-visible:text-slate-900 focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-1" tabindex="0" :aria-label="copy.categoryHint">
                        <Info class="h-4 w-4" aria-hidden="true" />
                        <span role="tooltip" class="pointer-events-none absolute bottom-full left-0 z-30 mb-2 w-72 max-w-[calc(100vw-2rem)] rounded-md bg-slate-900 px-3 py-2 text-xs font-medium leading-5 text-white opacity-0 shadow-lg transition-opacity group-hover:opacity-100 group-focus:opacity-100">
                          {{ copy.categoryHint }}
                        </span>
                      </span>
                    </span>
                    <input v-model="form.category" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2" :placeholder="copy.placeholders.category" />
                  </label>
                  <label class="block">
                    <span class="flex items-center gap-1.5 text-sm font-bold">
                      {{ copy.fields.icon }}
                      <span class="group relative inline-flex cursor-help rounded-full text-slate-600 outline-none transition-colors hover:text-slate-900 focus-visible:text-slate-900 focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-1" tabindex="0" :aria-label="copy.iconHint">
                        <Info class="h-4 w-4" aria-hidden="true" />
                        <span role="tooltip" class="pointer-events-none absolute bottom-full left-0 z-30 mb-2 w-72 max-w-[calc(100vw-2rem)] rounded-md bg-slate-900 px-3 py-2 text-xs font-medium leading-5 text-white opacity-0 shadow-lg transition-opacity group-hover:opacity-100 group-focus:opacity-100">
                          {{ copy.iconHint }}
                        </span>
                      </span>
                    </span>
                    <input v-model="form.icon" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2" :placeholder="copy.placeholders.icon" />
                  </label>
                  <label class="text-sm font-bold md:col-span-2">
                    {{ copy.fields.description }}
                    <textarea v-model="form.description" class="mt-2 min-h-24 w-full rounded-xl border border-slate-200 px-3 py-2" :placeholder="copy.placeholders.description" />
                  </label>
                </div>
              </div>

              <div class="border-t border-slate-100 pt-5">
                <div class="mb-3 text-sm font-black text-slate-950">{{ copy.sections.pathThumbnail }}</div>
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="block">
                    <span class="flex items-center gap-1.5 text-sm font-bold">
                      <span><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.fields.respath }}</span>
                      <span class="group relative inline-flex cursor-help rounded-full text-slate-600 outline-none transition-colors hover:text-slate-900 focus-visible:text-slate-900 focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-1" tabindex="0" :aria-label="copy.respathHint">
                        <Info class="h-4 w-4" aria-hidden="true" />
                        <span role="tooltip" class="pointer-events-none absolute bottom-full left-0 z-30 mb-2 w-72 max-w-[calc(100vw-2rem)] rounded-md bg-slate-900 px-3 py-2 text-xs font-medium leading-5 text-white opacity-0 shadow-lg transition-opacity group-hover:opacity-100 group-focus:opacity-100">
                          {{ copy.respathHint }}
                        </span>
                      </span>
                    </span>
                    <input v-model="form.respath" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2" :placeholder="copy.placeholders.respath" required />
                  </label>
                  <label class="block">
                    <span class="flex items-center gap-1.5 text-sm font-bold">
                      {{ copy.fields.thumbnailObjectKey }}
                      <span class="group relative inline-flex cursor-help rounded-full text-slate-600 outline-none transition-colors hover:text-slate-900 focus-visible:text-slate-900 focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-1" tabindex="0" :aria-label="copy.objectKeyHint">
                        <Info class="h-4 w-4" aria-hidden="true" />
                        <span role="tooltip" class="pointer-events-none absolute bottom-full left-0 z-30 mb-2 w-72 max-w-[calc(100vw-2rem)] rounded-md bg-slate-900 px-3 py-2 text-xs font-medium leading-5 text-white opacity-0 shadow-lg transition-opacity group-hover:opacity-100 group-focus:opacity-100">
                          {{ copy.objectKeyHint }}
                        </span>
                      </span>
                    </span>
                    <div class="mt-2 flex flex-col gap-3 sm:flex-row">
                      <input type="file" ref="thumbnailFileInput" class="hidden" accept="image/*" @change="handleThumbnailFileUpload" />
                      <input v-model="form.thumbnail_object_key" class="w-full rounded-xl border border-slate-200 px-3 py-2" :placeholder="copy.placeholders.thumbnailObjectKey" />
                      <button type="button" class="flex items-center justify-center gap-2 rounded-xl border border-blue-500 bg-blue-50 px-4 py-2 font-bold text-blue-700 shadow-sm transition hover:bg-blue-100 disabled:opacity-50 sm:shrink-0" :disabled="uploadingThumbnail || !form.pack_id" @click="thumbnailFileInput?.click()">
                        <Loader2 v-if="uploadingThumbnail" class="h-4 w-4 animate-spin" />
                        <UploadCloud v-else class="h-4 w-4" />
                        {{ uploadingThumbnail ? (copy as any).uploading : (copy as any).uploadThumbnail }}
                      </button>
                    </div>
                  </label>
                  <label class="block md:col-span-2">
                    <span class="flex items-center gap-1.5 text-sm font-bold">
                      {{ copy.fields.thumbnailFileHash }}
                      <span class="group relative inline-flex cursor-help rounded-full text-slate-600 outline-none transition-colors hover:text-slate-900 focus-visible:text-slate-900 focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-1" tabindex="0" :aria-label="copy.fileHashHint">
                        <Info class="h-4 w-4" aria-hidden="true" />
                        <span role="tooltip" class="pointer-events-none absolute bottom-full left-0 z-30 mb-2 w-72 max-w-[calc(100vw-2rem)] rounded-md bg-slate-900 px-3 py-2 text-xs font-medium leading-5 text-white opacity-0 shadow-lg transition-opacity group-hover:opacity-100 group-focus:opacity-100">
                          {{ copy.fileHashHint }}
                        </span>
                      </span>
                    </span>
                    <input v-model="form.thumbnail_file_hash" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2" :placeholder="copy.placeholders.thumbnailFileHash" />
                  </label>
                </div>
              </div>

              <div class="border-t border-slate-100 pt-5">
                <div class="mb-3 text-sm font-black text-slate-950">{{ copy.sections.system }}</div>
                <div class="grid gap-4 md:grid-cols-3">
                  <div v-if="mode !== 'create'" class="text-sm font-bold">
                    {{ copy.fields.packId }}
                    <div class="mt-2 min-h-10 break-all rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 font-semibold text-slate-700">{{ form.pack_id || "-" }}</div>
                  </div>
                  <div class="text-sm font-bold">
                    {{ copy.fields.status }}
                    <div class="mt-2 min-h-10 break-words rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 font-semibold text-slate-700">
                      <span class="inline-flex rounded-full border px-2 py-0.5 text-xs font-black" :class="packStatusClass(form)" :title="String(form.status || '')">
                        {{ packStatusLabel(form) }}
                      </span>
                    </div>
                  </div>
                  <label class="text-sm font-bold">
                    {{ copy.fields.version }}
                    <input v-model.number="form.version" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2 disabled:bg-slate-100" type="number" min="0" :disabled="mode === 'create'" />
                  </label>
                  <div v-if="mode !== 'create'" class="text-sm font-bold">
                    {{ copy.fields.createdAt }}
                    <div class="mt-2 min-h-10 break-words rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 font-semibold text-slate-700">{{ formatDate(form.created_at) || "-" }}</div>
                  </div>
                  <div v-if="mode !== 'create'" class="text-sm font-bold">
                    {{ copy.fields.updatedAt }}
                    <div class="mt-2 min-h-10 break-words rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 font-semibold text-slate-700">{{ formatDate(form.updated_at) || "-" }}</div>
                  </div>
                </div>
              </div>

            </template>
          </div>

          <div v-if="mode !== 'detail'" class="flex shrink-0 flex-col justify-end border-t border-slate-200 bg-white px-4 py-4 sm:flex-row md:px-5">
            <button class="inline-flex h-10 w-full min-w-[180px] items-center justify-center gap-2 rounded-xl bg-blue-700 px-4 font-bold text-white disabled:opacity-50 sm:w-auto" type="button" :disabled="saving" @click="savePack">
              <Loader2 v-if="saving" class="h-4 w-4 animate-spin" />
              <Save v-else class="h-4 w-4" />
              {{ mode === "create" ? copy.createPack : copy.savePack }}
            </button>
          </div>
        </div>
      </section>

      <Teleport to="body">
        <div v-if="deleteConfirmOpen && selected" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4 md:p-6">
          <section class="w-full max-w-[460px] rounded-2xl bg-white p-4 shadow-2xl md:rounded-3xl md:p-6">
            <h2 class="text-xl font-black text-slate-950 md:text-2xl">{{ copy.deleteConfirmTitle }}</h2>
            <p class="mt-3 text-sm font-semibold text-slate-500">{{ copy.deleteConfirmDescription }}</p>
            <div class="mt-5 rounded-2xl bg-slate-50 p-4">
              <div class="break-words font-black text-slate-950">{{ packTitle(selected) }}</div>
              <div class="mt-1 break-all text-sm font-semibold text-slate-500">{{ packId(selected) }}</div>
            </div>
            <div class="mt-6 flex flex-col justify-end gap-3 sm:flex-row">
              <button class="rounded-xl border border-slate-900 px-5 py-3 font-bold text-slate-950 disabled:opacity-50" type="button" :disabled="saving" @click="closeDeleteConfirm">{{ copy.cancel }}</button>
              <button class="rounded-xl bg-red-600 px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving" @click="deletePack">
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
.readonly-long-field {
  min-height: 4.75rem;
  margin-top: 0.5rem;
  width: 100%;
  border: 1px solid #e2e8f0;
  border-radius: 0.75rem;
  background: #f8fafc;
  padding: 0.625rem 0.75rem;
  color: #334155;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 0.8125rem;
  font-weight: 700;
  line-height: 1.45;
  overflow-wrap: anywhere;
  word-break: break-all;
  white-space: pre-wrap;
}
</style>


