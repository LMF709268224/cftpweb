<script setup lang="ts">
import {
  ArrowUpRight,
  ChevronDown,
  ChevronRight,
  CopyPlus,
  Eye,
  History,
  Info,
  Loader2,
  Package,
  Plus,
  RefreshCw,
  Save,
  Trash2,
  Undo2,
  Upload,
  UploadCloud,
  X,
} from "lucide-vue-next"
import type { Component } from "vue"
import { computed, onMounted, ref, watch } from "vue"
import { RouterLink } from "vue-router"
import { toast } from "vue-sonner"
import RowMenu from "@/components/RowMenu.vue"
import UlidChip from "@/components/UlidChip.vue"
import { ApiError, apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { groupByLineage } from "@/lib/versionGroup"

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
  if (status.includes("DRAFT")) return "bg-amber-50 text-amber-700"
  if (status.includes("DEPRECATED")) return "bg-slate-100 text-slate-600"
  if (status.includes("ACTIVE") || status.includes("PUBLISHED")) return "bg-emerald-50 text-emerald-700"
  return "bg-slate-100 text-slate-600"
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

function isDeprecatedPack(pack: JsonRecord | null) {
  return packStatusKey(pack).includes("DEPRECATED")
}

const lineageGroups = computed(() =>
  groupByLineage(packs.value, (pack) => String(pack.title || ""), (pack) => String(pack.category || "")),
)
const expandedStems = ref<Record<string, boolean>>({})

function groupKey(group: { primary: JsonRecord }) {
  return packId(group.primary)
}

function toggleHistory(group: { primary: JsonRecord }) {
  const key = groupKey(group)
  expandedStems.value[key] = !expandedStems.value[key]
}

type PackMenuItem = { key: string; label: string; icon?: Component; danger?: boolean; disabled?: boolean }

function packMenuItems(pack: JsonRecord): PackMenuItem[] {
  const items: PackMenuItem[] = [
    { key: "detail", label: copy.value.viewDetails, icon: Eye },
    { key: "clone", label: copy.value.copyPack, icon: CopyPlus, disabled: saving.value },
  ]
  if (canPublishPack(pack)) {
    items.push({ key: "publish", label: copy.value.publishPack, icon: Upload, disabled: saving.value })
  }
  if (canRevertPack(pack)) {
    items.push({ key: "revert", label: copy.value.revertPack, icon: Undo2, disabled: saving.value })
  }
  if (canDeletePack(pack)) {
    items.push({ key: "delete", label: copy.value.deletePack, icon: Trash2, danger: true, disabled: saving.value })
  }
  return items
}

function onPackMenu(key: string, pack: JsonRecord) {
  if (key === "detail") openPackDetail(pack)
  if (key === "clone") void duplicatePack(pack)
  if (key === "publish") void runPackAction(pack, "publish")
  if (key === "revert") void runPackAction(pack, "revert-to-draft")
  if (key === "delete") requestDeletePack(pack)
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

onMounted(load)
</script>

<template>
  <main class="ui-page">
    <header class="flex flex-wrap items-start justify-between gap-4 md:items-center">
      <div class="min-w-0">
        <h1 class="ui-page-title">{{ copy.title }}</h1>
        <p class="ui-page-subtitle">{{ copy.subtitle }}</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button class="ui-btn ui-btn-secondary" type="button" :disabled="loading" @click="load">
          <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
          <RefreshCw v-else class="h-4 w-4 text-slate-400" />
          {{ copy.refresh }}
        </button>
        <button class="ui-btn ui-btn-primary" type="button" @click="startCreate">
          <Plus class="h-4 w-4" />
          {{ copy.newPack }}
        </button>
      </div>
    </header>

    <section class="ui-card mt-6 overflow-hidden">
      <div class="flex flex-wrap items-center justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
        <div class="min-w-0">
          <h2 class="text-base font-semibold text-ink">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
        <span class="tnum ui-badge shrink-0 bg-slate-100 text-slate-600">{{ copy.pageSizeText(packs.length, pageSize) }}</span>
      </div>

      <div v-if="loading" class="divide-y divide-hairline">
        <div
          v-for="i in 5"
          :key="`sk-${i}`"
          class="grid gap-3 px-4 py-4 lg:grid-cols-[minmax(0,1fr)_84px_120px_190px] lg:items-center lg:gap-6 lg:py-3.5"
        >
          <div>
            <div class="skeleton h-4 w-2/5" />
            <div class="skeleton mt-2 h-3 w-4/5" />
          </div>
          <div class="skeleton h-4 w-10" />
          <div class="skeleton h-5 w-16 rounded-full" />
          <div class="skeleton h-8 w-3/5 lg:w-full" />
        </div>
      </div>

      <div v-else-if="!packs.length" class="ui-empty">
        <div class="ui-empty-icon">
          <Package class="h-6 w-6" />
        </div>
        <p class="ui-empty-title">{{ copy.empty }}</p>
        <p class="ui-empty-hint">{{ copy.listDescription }}</p>
      </div>

      <div v-else>
        <div class="hidden grid-cols-[minmax(0,1fr)_84px_120px_190px] gap-6 border-b border-edge bg-slate-50/60 px-4 py-3 text-xs font-medium uppercase tracking-wider text-slate-400 lg:grid">
          <span>{{ copy.columns.pack }}</span>
          <span>{{ copy.columns.version }}</span>
          <span class="text-center">{{ copy.columns.status }}</span>
          <span class="text-right">{{ copy.columns.action }}</span>
        </div>
        <div class="divide-y divide-hairline">
          <template v-for="group in lineageGroups" :key="groupKey(group)">
            <div
              class="ui-table-row grid w-full cursor-pointer gap-3 px-4 py-4 text-left lg:grid-cols-[minmax(0,1fr)_84px_120px_190px] lg:items-center lg:gap-6 lg:py-3.5"
              :class="packId(selected) === packId(group.primary) ? 'bg-primary-50/60' : ''"
              role="button"
              tabindex="0"
              @click="openPackDetail(group.primary)"
              @keydown.enter.self.prevent="openPackDetail(group.primary)"
              @keydown.space.self.prevent="openPackDetail(group.primary)"
            >
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <span class="break-words text-sm font-medium" :class="isDeprecatedPack(group.primary) ? 'text-slate-500' : 'text-ink'">{{ packTitle(group.primary) }}</span>
                  <button
                    v-if="group.history.length"
                    type="button"
                    class="tnum inline-flex h-6 shrink-0 items-center gap-1 rounded-full bg-slate-100 px-2 text-xs font-medium text-slate-500 transition-colors hover:bg-slate-200/70 hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30"
                    :aria-expanded="Boolean(expandedStems[groupKey(group)])"
                    :title="t.common.historyVersions"
                    @click.stop="toggleHistory(group)"
                  >
                    <History class="h-3 w-3" />
                    {{ group.history.length }}
                    <ChevronDown class="h-3 w-3 transition-transform duration-150" :class="expandedStems[groupKey(group)] ? 'rotate-180' : ''" />
                  </button>
                </div>
                <div class="mt-0.5 line-clamp-1 text-sm text-slate-500">{{ group.primary.description || "-" }}</div>
                <div class="mt-1.5 flex flex-wrap items-center gap-2">
                  <UlidChip :value="packId(group.primary)" />
                  <RouterLink
                    :to="{ path: '/resource-pack-files', query: { pack: packId(group.primary) } }"
                    class="inline-flex items-center gap-1 text-sm text-slate-600 underline decoration-slate-300 decoration-dashed underline-offset-4 transition-colors hover:text-primary-700 hover:decoration-primary-400"
                    @click.stop
                  >
                    {{ t.common.viewFiles }}
                    <ArrowUpRight class="h-3.5 w-3.5 text-slate-400" />
                  </RouterLink>
                </div>
              </div>
              <div class="flex items-center justify-between gap-4 border-t border-hairline pt-3 text-sm lg:block lg:border-0 lg:pt-0">
                <span class="text-xs font-medium text-slate-400 lg:hidden">{{ copy.fields.version }}</span>
                <span class="tnum text-slate-600">{{ group.primary.version || 0 }}</span>
              </div>
              <div class="flex items-center justify-between gap-4 border-t border-hairline pt-3 lg:block lg:justify-self-center lg:border-0 lg:pt-0">
                <span class="text-xs font-medium text-slate-400 lg:hidden">{{ copy.columns.status }}</span>
                <span class="ui-badge" :class="packStatusClass(group.primary)" :title="String(group.primary.status || '')">
                  {{ packStatusLabel(group.primary) }}
                </span>
              </div>
              <div class="flex items-center justify-between gap-2 border-t border-hairline pt-3 lg:justify-end lg:border-0 lg:pt-0">
                <span class="text-xs font-medium text-slate-400 lg:hidden">{{ copy.columns.action }}</span>
                <div class="flex items-center gap-2">
                  <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" @click.stop="openPackEditor(group.primary)">
                    {{ copy.editPack }}
                  </button>
                  <RowMenu :items="packMenuItems(group.primary)" @select="(key) => onPackMenu(key, group.primary)" />
                  <ChevronRight class="hidden h-4 w-4 shrink-0 text-slate-300 lg:block" />
                </div>
              </div>
            </div>
            <template v-if="expandedStems[groupKey(group)]">
              <div
                v-for="old in group.history"
                :key="packId(old)"
                class="ui-table-row grid w-full cursor-pointer gap-3 px-4 py-3 text-left lg:grid-cols-[minmax(0,1fr)_84px_120px_190px] lg:items-center lg:gap-6 lg:py-2.5"
                :class="packId(selected) === packId(old) ? 'bg-primary-50/60' : 'bg-slate-50/60'"
                role="button"
                tabindex="0"
                @click="openPackDetail(old)"
                @keydown.enter.self.prevent="openPackDetail(old)"
                @keydown.space.self.prevent="openPackDetail(old)"
              >
                <div class="min-w-0 border-l-2 border-slate-200 pl-3">
                  <div class="flex flex-wrap items-center gap-2">
                    <span class="break-words text-sm text-slate-500">{{ packTitle(old) }}</span>
                    <span v-if="isDeprecatedPack(old)" class="ui-badge bg-slate-100 text-slate-500">{{ t.common.deprecatedMuted }}</span>
                  </div>
                  <div class="mt-1 flex flex-wrap items-center gap-2">
                    <UlidChip :value="packId(old)" />
                    <RouterLink
                      :to="{ path: '/resource-pack-files', query: { pack: packId(old) } }"
                      class="inline-flex items-center gap-1 text-sm text-slate-500 underline decoration-slate-300 decoration-dashed underline-offset-4 transition-colors hover:text-primary-700 hover:decoration-primary-400"
                      @click.stop
                    >
                      {{ t.common.viewFiles }}
                      <ArrowUpRight class="h-3.5 w-3.5 text-slate-400" />
                    </RouterLink>
                  </div>
                </div>
                <div class="flex items-center justify-between gap-4 border-t border-hairline pt-3 text-sm lg:block lg:border-0 lg:pt-0">
                  <span class="text-xs font-medium text-slate-400 lg:hidden">{{ copy.fields.version }}</span>
                  <span class="tnum text-slate-500">{{ old.version || 0 }}</span>
                </div>
                <div class="flex items-center justify-between gap-4 border-t border-hairline pt-3 lg:block lg:justify-self-center lg:border-0 lg:pt-0">
                  <span class="text-xs font-medium text-slate-400 lg:hidden">{{ copy.columns.status }}</span>
                  <span class="ui-badge" :class="packStatusClass(old)" :title="String(old.status || '')">
                    {{ packStatusLabel(old) }}
                  </span>
                </div>
                <div class="flex items-center justify-between gap-2 border-t border-hairline pt-3 lg:justify-end lg:border-0 lg:pt-0">
                  <span class="text-xs font-medium text-slate-400 lg:hidden">{{ copy.columns.action }}</span>
                  <div class="flex items-center gap-2">
                    <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" @click.stop="openPackEditor(old)">
                      {{ copy.editPack }}
                    </button>
                    <RowMenu :items="packMenuItems(old)" @select="(key) => onPackMenu(key, old)" />
                    <ChevronRight class="hidden h-4 w-4 shrink-0 text-slate-300 lg:block" />
                  </div>
                </div>
              </div>
            </template>
          </template>
        </div>
      </div>

      <div class="flex flex-col items-stretch justify-between gap-3 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center md:px-6">
        <span class="tnum text-center text-sm text-slate-500 sm:text-left">{{ `${currentPage} / ${Math.max(1, Math.ceil(total / pageSize))}` }}</span>
        <div class="flex flex-col gap-2 sm:flex-row">
          <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canPrevious || loading" @click="previousPage">{{ copy.prev }}</button>
          <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canNext || loading" @click="nextPage">{{ copy.next }}</button>
        </div>
      </div>
    </section>

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="detailOpen" class="ui-modal-mask">
          <section class="ui-modal-panel max-w-[1100px]">
            <div class="flex items-start justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
              <div class="min-w-0">
                <h2 class="break-words text-base font-semibold text-ink md:text-lg">{{ mode === "create" ? copy.createTitle : mode === "edit" ? copy.editTitle : copy.detailTitle }}</h2>
                <p class="mt-1 text-sm text-slate-500">{{ mode === "detail" ? copy.readonlyHint : mode === "create" ? copy.createHint : copy.editHint }}</p>
                <p v-if="detailLoading" class="mt-1 inline-flex items-center gap-2 text-xs font-medium text-primary-700">
                  <Loader2 class="h-3.5 w-3.5 animate-spin" />
                  {{ copy.detailLoading }}
                </p>
              </div>
              <button class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600" type="button" :aria-label="copy.close" @click="closePackDetail">
                <X class="h-5 w-5" />
              </button>
            </div>

            <div class="min-h-0 flex-1 space-y-6 overflow-y-auto p-4 md:p-6">
              <template v-if="mode === 'detail'">
                <div>
                  <div class="mb-3 text-sm font-semibold text-ink">{{ copy.sections.basic }}</div>
                  <dl class="grid gap-4 md:grid-cols-2">
                    <div class="min-w-0">
                      <dt class="text-xs font-medium text-slate-400">{{ copy.fields.title }}</dt>
                      <dd class="mt-1 break-words text-sm text-ink">{{ form.title || "-" }}</dd>
                    </div>
                    <div class="min-w-0">
                      <dt class="text-xs font-medium text-slate-400">{{ copy.fields.category }}</dt>
                      <dd class="mt-1 break-words text-sm text-slate-600">{{ form.category || "-" }}</dd>
                    </div>
                    <div class="min-w-0">
                      <dt class="text-xs font-medium text-slate-400">{{ copy.fields.icon }}</dt>
                      <dd class="mt-1 break-words text-sm text-slate-600">{{ form.icon || "-" }}</dd>
                    </div>
                    <div class="min-w-0 md:col-span-2">
                      <dt class="text-xs font-medium text-slate-400">{{ copy.fields.description }}</dt>
                      <dd class="mt-1 whitespace-pre-wrap break-words text-sm text-slate-600">{{ form.description || "-" }}</dd>
                    </div>
                  </dl>
                </div>

                <div class="border-t border-hairline pt-6">
                  <div class="mb-3 text-sm font-semibold text-ink">{{ copy.sections.pathThumbnail }}</div>
                  <dl class="grid gap-4 md:grid-cols-2">
                    <div class="min-w-0">
                      <dt class="text-xs font-medium text-slate-400">{{ copy.fields.respath }}</dt>
                      <dd class="mt-1"><UlidChip :value="form.respath" :truncate="false" /></dd>
                    </div>
                    <div class="min-w-0">
                      <dt class="text-xs font-medium text-slate-400">{{ copy.fields.thumbnailObjectKey }}</dt>
                      <dd class="mt-1"><UlidChip :value="form.thumbnail_object_key" :truncate="false" /></dd>
                    </div>
                    <div class="min-w-0 md:col-span-2">
                      <dt class="text-xs font-medium text-slate-400">{{ copy.fields.thumbnailFileHash }}</dt>
                      <dd class="mt-1"><UlidChip :value="form.thumbnail_file_hash" :truncate="false" /></dd>
                    </div>
                  </dl>
                </div>

                <div class="border-t border-hairline pt-6">
                  <div class="mb-3 text-sm font-semibold text-ink">{{ copy.sections.system }}</div>
                  <dl class="grid gap-4 md:grid-cols-3">
                    <div class="min-w-0">
                      <dt class="text-xs font-medium text-slate-400">{{ copy.fields.packId }}</dt>
                      <dd class="mt-1"><UlidChip :value="form.pack_id" :truncate="false" /></dd>
                    </div>
                    <div class="min-w-0">
                      <dt class="text-xs font-medium text-slate-400">{{ copy.fields.status }}</dt>
                      <dd class="mt-1">
                        <span class="ui-badge" :class="packStatusClass(form)" :title="String(form.status || '')">
                          {{ packStatusLabel(form) }}
                        </span>
                      </dd>
                    </div>
                    <div class="min-w-0">
                      <dt class="text-xs font-medium text-slate-400">{{ copy.fields.version }}</dt>
                      <dd class="tnum mt-1 text-sm text-slate-600">{{ form.version || 0 }}</dd>
                    </div>
                    <div class="min-w-0">
                      <dt class="text-xs font-medium text-slate-400">{{ copy.fields.createdAt }}</dt>
                      <dd class="tnum mt-1 text-sm text-slate-600">{{ formatDate(form.created_at) || "-" }}</dd>
                    </div>
                    <div class="min-w-0">
                      <dt class="text-xs font-medium text-slate-400">{{ copy.fields.updatedAt }}</dt>
                      <dd class="tnum mt-1 text-sm text-slate-600">{{ formatDate(form.updated_at) || "-" }}</dd>
                    </div>
                  </dl>
                </div>
              </template>

              <template v-else>
                <div>
                  <div class="mb-3 text-sm font-semibold text-ink">{{ copy.sections.basic }}</div>
                  <div class="grid gap-4 md:grid-cols-2">
                    <label class="grid content-start gap-1.5">
                      <span class="ui-label"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.fields.title }}</span>
                      <input v-model="form.title" class="ui-input" :placeholder="copy.placeholders.title" required />
                    </label>
                    <label class="grid content-start gap-1.5">
                      <span class="ui-label flex items-center gap-1.5">
                        {{ copy.fields.category }}
                        <span class="group relative inline-flex cursor-help rounded-full text-slate-400 outline-none transition-colors hover:text-slate-600 focus-visible:text-slate-600 focus-visible:ring-4 focus-visible:ring-primary-500/20" tabindex="0" :aria-label="copy.categoryHint">
                          <Info class="h-4 w-4" aria-hidden="true" />
                          <span role="tooltip" class="pointer-events-none absolute bottom-full left-0 z-30 mb-2 w-72 max-w-[calc(100vw-2rem)] rounded-lg bg-slate-900 px-3 py-2 text-xs font-medium leading-5 text-white opacity-0 shadow-lg transition-opacity group-hover:opacity-100 group-focus:opacity-100">
                            {{ copy.categoryHint }}
                          </span>
                        </span>
                      </span>
                      <input v-model="form.category" class="ui-input" :placeholder="copy.placeholders.category" />
                    </label>
                    <label class="grid content-start gap-1.5">
                      <span class="ui-label flex items-center gap-1.5">
                        {{ copy.fields.icon }}
                        <span class="group relative inline-flex cursor-help rounded-full text-slate-400 outline-none transition-colors hover:text-slate-600 focus-visible:text-slate-600 focus-visible:ring-4 focus-visible:ring-primary-500/20" tabindex="0" :aria-label="copy.iconHint">
                          <Info class="h-4 w-4" aria-hidden="true" />
                          <span role="tooltip" class="pointer-events-none absolute bottom-full left-0 z-30 mb-2 w-72 max-w-[calc(100vw-2rem)] rounded-lg bg-slate-900 px-3 py-2 text-xs font-medium leading-5 text-white opacity-0 shadow-lg transition-opacity group-hover:opacity-100 group-focus:opacity-100">
                            {{ copy.iconHint }}
                          </span>
                        </span>
                      </span>
                      <input v-model="form.icon" class="ui-input" :placeholder="copy.placeholders.icon" />
                    </label>
                    <label class="grid content-start gap-1.5 md:col-span-2">
                      <span class="ui-label">{{ copy.fields.description }}</span>
                      <textarea v-model="form.description" class="ui-textarea" :placeholder="copy.placeholders.description" />
                    </label>
                  </div>
                </div>

                <div class="border-t border-hairline pt-6">
                  <div class="mb-3 text-sm font-semibold text-ink">{{ copy.sections.pathThumbnail }}</div>
                  <div class="grid gap-4 md:grid-cols-2">
                    <label class="grid content-start gap-1.5">
                      <span class="ui-label flex items-center gap-1.5">
                        <span><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.fields.respath }}</span>
                        <span class="group relative inline-flex cursor-help rounded-full text-slate-400 outline-none transition-colors hover:text-slate-600 focus-visible:text-slate-600 focus-visible:ring-4 focus-visible:ring-primary-500/20" tabindex="0" :aria-label="copy.respathHint">
                          <Info class="h-4 w-4" aria-hidden="true" />
                          <span role="tooltip" class="pointer-events-none absolute bottom-full left-0 z-30 mb-2 w-72 max-w-[calc(100vw-2rem)] rounded-lg bg-slate-900 px-3 py-2 text-xs font-medium leading-5 text-white opacity-0 shadow-lg transition-opacity group-hover:opacity-100 group-focus:opacity-100">
                            {{ copy.respathHint }}
                          </span>
                        </span>
                      </span>
                      <input v-model="form.respath" class="ui-input" :placeholder="copy.placeholders.respath" required />
                    </label>
                    <label class="grid content-start gap-1.5">
                      <span class="ui-label flex items-center gap-1.5">
                        {{ copy.fields.thumbnailObjectKey }}
                        <span class="group relative inline-flex cursor-help rounded-full text-slate-400 outline-none transition-colors hover:text-slate-600 focus-visible:text-slate-600 focus-visible:ring-4 focus-visible:ring-primary-500/20" tabindex="0" :aria-label="copy.objectKeyHint">
                          <Info class="h-4 w-4" aria-hidden="true" />
                          <span role="tooltip" class="pointer-events-none absolute bottom-full left-0 z-30 mb-2 w-72 max-w-[calc(100vw-2rem)] rounded-lg bg-slate-900 px-3 py-2 text-xs font-medium leading-5 text-white opacity-0 shadow-lg transition-opacity group-hover:opacity-100 group-focus:opacity-100">
                            {{ copy.objectKeyHint }}
                          </span>
                        </span>
                      </span>
                      <div class="flex flex-col gap-3 sm:flex-row">
                        <input type="file" ref="thumbnailFileInput" class="hidden" accept="image/*" @change="handleThumbnailFileUpload" />
                        <input v-model="form.thumbnail_object_key" class="ui-input" :placeholder="copy.placeholders.thumbnailObjectKey" />
                        <button type="button" class="ui-btn ui-btn-secondary sm:shrink-0" :disabled="uploadingThumbnail || !form.pack_id" @click="thumbnailFileInput?.click()">
                          <Loader2 v-if="uploadingThumbnail" class="h-4 w-4 animate-spin" />
                          <UploadCloud v-else class="h-4 w-4 text-slate-400" />
                          {{ uploadingThumbnail ? (copy as any).uploading : (copy as any).uploadThumbnail }}
                        </button>
                      </div>
                    </label>
                    <label class="grid content-start gap-1.5 md:col-span-2">
                      <span class="ui-label flex items-center gap-1.5">
                        {{ copy.fields.thumbnailFileHash }}
                        <span class="group relative inline-flex cursor-help rounded-full text-slate-400 outline-none transition-colors hover:text-slate-600 focus-visible:text-slate-600 focus-visible:ring-4 focus-visible:ring-primary-500/20" tabindex="0" :aria-label="copy.fileHashHint">
                          <Info class="h-4 w-4" aria-hidden="true" />
                          <span role="tooltip" class="pointer-events-none absolute bottom-full left-0 z-30 mb-2 w-72 max-w-[calc(100vw-2rem)] rounded-lg bg-slate-900 px-3 py-2 text-xs font-medium leading-5 text-white opacity-0 shadow-lg transition-opacity group-hover:opacity-100 group-focus:opacity-100">
                            {{ copy.fileHashHint }}
                          </span>
                        </span>
                      </span>
                      <input v-model="form.thumbnail_file_hash" class="ui-input" :placeholder="copy.placeholders.thumbnailFileHash" />
                    </label>
                  </div>
                </div>

                <div class="border-t border-hairline pt-6">
                  <div class="mb-3 text-sm font-semibold text-ink">{{ copy.sections.system }}</div>
                  <div class="grid gap-4 md:grid-cols-3">
                    <div v-if="mode !== 'create'" class="grid content-start gap-1.5">
                      <span class="ui-label">{{ copy.fields.packId }}</span>
                      <div class="flex min-h-10 items-center"><UlidChip :value="form.pack_id" :truncate="false" /></div>
                    </div>
                    <div class="grid content-start gap-1.5">
                      <span class="ui-label">{{ copy.fields.status }}</span>
                      <div class="readonly-field readonly-field--compact">
                        <span class="ui-badge" :class="packStatusClass(form)" :title="String(form.status || '')">
                          {{ packStatusLabel(form) }}
                        </span>
                      </div>
                    </div>
                    <label class="grid content-start gap-1.5">
                      <span class="ui-label">{{ copy.fields.version }}</span>
                      <input v-model.number="form.version" class="tnum ui-input" type="number" min="0" :disabled="mode === 'create'" />
                    </label>
                    <div v-if="mode !== 'create'" class="grid content-start gap-1.5">
                      <span class="ui-label">{{ copy.fields.createdAt }}</span>
                      <div class="tnum readonly-field readonly-field--compact">{{ formatDate(form.created_at) || "-" }}</div>
                    </div>
                    <div v-if="mode !== 'create'" class="grid content-start gap-1.5">
                      <span class="ui-label">{{ copy.fields.updatedAt }}</span>
                      <div class="tnum readonly-field readonly-field--compact">{{ formatDate(form.updated_at) || "-" }}</div>
                    </div>
                  </div>
                </div>
              </template>
            </div>

            <div v-if="mode !== 'detail'" class="flex shrink-0 flex-col justify-end gap-3 border-t border-hairline px-4 py-4 sm:flex-row md:px-6">
              <button class="ui-btn ui-btn-primary w-full sm:w-auto" type="button" :disabled="saving" @click="savePack">
                <Loader2 v-if="saving" class="h-4 w-4 animate-spin" />
                <Save v-else class="h-4 w-4" />
                {{ mode === "create" ? copy.createPack : copy.savePack }}
              </button>
            </div>
          </section>
        </div>
      </Transition>
    </Teleport>

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="deleteConfirmOpen && selected" class="ui-modal-mask z-50">
          <section class="ui-modal-panel md:max-w-md">
            <div class="border-b border-hairline px-4 py-4 md:px-6 md:py-5">
              <h2 class="text-base font-semibold text-ink md:text-lg">{{ copy.deleteConfirmTitle }}</h2>
            </div>
            <div class="min-h-0 flex-1 overflow-y-auto p-4 md:p-6">
              <p class="text-sm text-slate-600">{{ copy.deleteConfirmDescription }}</p>
              <div class="mt-4 rounded-[10px] bg-slate-50 p-4">
                <div class="break-words text-sm font-semibold text-ink">{{ packTitle(selected) }}</div>
                <div class="mt-1.5"><UlidChip :value="packId(selected)" :truncate="false" /></div>
              </div>
            </div>
            <div class="flex flex-col justify-end gap-3 border-t border-hairline px-4 py-4 sm:flex-row md:px-6">
              <button class="ui-btn ui-btn-secondary" type="button" :disabled="saving" @click="closeDeleteConfirm">{{ copy.cancel }}</button>
              <button class="ui-btn ui-btn-danger" type="button" :disabled="saving" @click="deletePack">
                {{ saving ? copy.deleting : copy.confirmDeleteAction }}
              </button>
            </div>
          </section>
        </div>
      </Transition>
    </Teleport>
  </main>
</template>

<style scoped>
.readonly-field {
  min-height: 2.5rem;
  width: 100%;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  background: #f8fafc;
  padding: 0.5rem 0.75rem;
  color: #475569;
  font-size: 0.875rem;
  line-height: 1.5;
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
  min-height: 6rem;
}

.ui-textarea {
  min-height: 6rem;
  width: 100%;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  background: #ffffff;
  padding: 0.5rem 0.75rem;
  font-size: 0.875rem;
  color: #0f172a;
  transition: border-color 150ms ease, box-shadow 150ms ease;
}

.ui-textarea::placeholder {
  color: #94a3b8;
}

.ui-textarea:focus {
  border-color: #a5b4fc;
  outline: none;
  box-shadow: 0 0 0 4px rgb(99 102 241 / 0.1);
}
</style>
