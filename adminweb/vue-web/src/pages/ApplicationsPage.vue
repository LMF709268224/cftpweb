<script setup lang="ts">
import { CheckCircle2, ChevronRight, Download, Eye, FileText, Inbox, Loader2, RefreshCw, RotateCcw, X, XCircle } from "lucide-vue-next"
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import JsonPreview from "@/components/JsonPreview.vue"
import UlidChip from "@/components/UlidChip.vue"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { badgeClass, pickFirst } from "@/lib/status"

type DetailTab = "overview" | "files" | "audit"

const applications = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const loading = ref(false)
const detailLoading = ref(false)
const detailOpen = ref(false)
const auditing = ref(false)
const page = ref(1)
const total = ref(0)
const hasMore = ref(false)
const nextCursor = ref("")
const prevCursor = ref("")
const lastPage = ref(1)
const statusFilter = ref("Pending")
const auditRemark = ref("")
const activeTab = ref<DetailTab>("overview")
const pageSize = 20
let detailRequestId = 0
const { t } = useAdminLanguage()
const copy = computed(() => t.value.applications)

// lg 及以上使用左右分栏审核工作台；小屏保持列表 + 弹窗形态
const isDesktop = ref(false)
let desktopQuery: MediaQueryList | null = null

function syncDesktop(query: MediaQueryList | MediaQueryListEvent) {
  isDesktop.value = query.matches
  if (query.matches) detailOpen.value = false
}

const canPrev = computed(() => page.value > 1)
const canNext = computed(() => hasMore.value)
const applicationFieldLabels = computed<Record<string, string>>(() => copy.value.fieldLabels || {})
const applicationIdKeys = new Set(["app_ulid", "app_id", "application_ulid", "application_id"])
const hiddenOverviewFieldKeys = new Set([
  "files",
  "application_files",
  "application_materials",
  "supporting_documents",
  "attachments",
])
const selectedFields = computed(() => {
  const current = selected.value || {}
  let hasApplicationId = false
  const hasDuplicateCredentialName =
    "cred_def_name" in current &&
    "credential_name" in current &&
    String(current.cred_def_name ?? "") === String(current.credential_name ?? "")

  return Object.entries(current)
    .filter(([key, value]) => {
      if (hiddenOverviewFieldKeys.has(key)) return false
      if (hasDuplicateCredentialName && key === "credential_name") return false
      if (!applicationIdKeys.has(key)) return true
      const currentValue = String(value ?? "")
      if (!currentValue) return true
      if (!hasApplicationId) {
        hasApplicationId = true
        return true
      }
      return currentValue !== appUlid(current)
    })
    .map(([key, value]) => ({
      key,
      label: applicationFieldLabels.value[key] || key.replace(/_/g, " "),
      value,
      displayValue: key === "status" || key === "application_status" ? applicationLabel(value) : key.endsWith("_at") || key.endsWith("_time") ? formatDate(String(value || "")) : String(value ?? "-"),
    }))
})
const selectedFiles = computed(() => files(selected.value || {}))
const isApprovedSelected = computed(() => isApprovedApplication(selected.value))
const detailTabs = computed(() => {
  const tabs: Array<{ key: DetailTab; title: string; count: number }> = [
    { key: "overview" as const, title: copy.value.tabs.overview, count: selected.value ? 1 : 0 },
    { key: "files" as const, title: copy.value.tabs.files, count: selectedFiles.value.length },
  ]
  if (!isApprovedSelected.value) {
    tabs.push({ key: "audit" as const, title: copy.value.tabs.audit, count: 3 })
  }
  return tabs
})
const statusOptions = computed(() => [
  { value: "", label: copy.value.statusOptions.all },
  { value: "Pending", label: copy.value.statusOptions.pending },
  { value: "Approved", label: copy.value.statusOptions.approved },
  { value: "Rejected", label: copy.value.statusOptions.rejected },
  { value: "Reupload", label: copy.value.statusOptions.resubmit },
])

function appUlid(app: JsonRecord | null | undefined) {
  return String(pickFirst(app || {}, ["app_ulid", "app_id", "application_ulid", "application_id"]) || "")
}

function candidate(app: JsonRecord | null | undefined) {
  return String(pickFirst(app || {}, ["candidate_name", "candidate_email", "candidate_ulid", "candidate_id"]) || "-")
}

function credential(app: JsonRecord | null | undefined) {
  return String(pickFirst(app || {}, ["cred_def_name", "credential_name", "cred_def_ulid", "cred_def_id"]) || "-")
}

function status(app: JsonRecord | null | undefined) {
  return pickFirst(app || {}, ["status", "application_status"])
}

function files(app: JsonRecord) {
  const value = app.files
  return Array.isArray(value) ? value.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item)) : []
}

function fileName(file: JsonRecord) {
  return String(pickFirst(file, ["file_name", "name", "filename", "file_hash"]) || copy.value.defaults.file)
}

function fileUrl(file: JsonRecord) {
  return String(pickFirst(file, ["view_url", "download_url", "url"]) || "")
}

function previewUrl(file: JsonRecord) {
  const url = fileUrl(file)
  if (!url) return ""
  
  const ext = String(file.file_ext || "").toLowerCase()
  if ([".xlsx", ".xls", ".docx", ".doc", ".pptx", ".ppt"].includes(ext)) {
    return `https://view.officeapps.live.com/op/view.aspx?src=${encodeURIComponent(url)}`
  }
  return url
}

function fileUsage(file: JsonRecord) {
  return String(pickFirst(file, ["file_usage", "usage"]) || copy.value.defaults.usage)
}

function applicationLabel(value: unknown) {
  const normalized = String(value || "").trim().toUpperCase()
  if (normalized.includes("APPROVED") || normalized === "2") return copy.value.statusOptions.approved
  if (normalized.includes("REJECTED") || normalized === "3") return copy.value.statusOptions.rejected
  if (normalized.includes("RESUBMIT") || normalized.includes("REUPLOAD") || normalized === "4") return copy.value.statusOptions.resubmit
  if (normalized.includes("PENDING") || normalized === "1") return copy.value.statusOptions.pending
  return normalized || "-"
}

function isApprovedApplication(app: JsonRecord | null | undefined) {
  return applicationLabel(status(app)) === copy.value.statusOptions.approved
}

function fileHash(file: JsonRecord) {
  return String(file.file_hash || "")
}

function fileSize(file: JsonRecord) {
  const size = Number(file.file_size || 0)
  if (!Number.isFinite(size) || size <= 0) return ""
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

function isStructuredValue(value: unknown) {
  return Array.isArray(value) || (!!value && typeof value === "object")
}

function isUlidField(key: string) {
  return key === "id" || key.endsWith("_ulid") || key.endsWith("_id")
}

function mergeApplicationDetail(appID: string, detail: JsonRecord) {
  const index = applications.value.findIndex((app) => appUlid(app) === appID)
  const base = index >= 0 ? applications.value[index] : selected.value || {}
  const merged = { ...base, ...detail }
  for (const key of ["audit_remark", "auditor_ulid", "audit_at"]) {
    if (Object.prototype.hasOwnProperty.call(base, key)) {
      merged[key] = base[key]
    }
  }
  if (index >= 0) {
    applications.value.splice(index, 1, merged)
  }
  if (selected.value && appUlid(selected.value) === appID) {
    selected.value = merged
    if (isApprovedApplication(merged) && activeTab.value === "audit") {
      activeTab.value = "overview"
    }
  }
}

async function loadApplicationDetail(app: JsonRecord | null) {
  if (!app) return
  const appID = appUlid(app)
  if (!appID) return

  const requestId = ++detailRequestId
  detailLoading.value = true
  try {
    const detail = await apiClient<JsonRecord>(`/api/applications/${encodeURIComponent(appID)}`)
    if (requestId !== detailRequestId) return
    if (detail && typeof detail === "object" && !Array.isArray(detail)) {
      mergeApplicationDetail(appID, detail)
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

function selectApplication(app: JsonRecord) {
  selected.value = app
  detailOpen.value = !isDesktop.value
  auditRemark.value = ""
  activeTab.value = "overview"
  void loadApplicationDetail(app)
}

function closeDetail() {
  detailOpen.value = false
}

async function load(targetPage = page.value) {
  loading.value = true
  try {
    const params = new URLSearchParams({
      page_size: String(pageSize),
      status: statusFilter.value,
    })

    let cursor = ""

    if (targetPage > lastPage.value) {

      cursor = nextCursor.value

    } else if (targetPage < lastPage.value) {

      cursor = prevCursor.value


    }

    

    if (cursor) params.set("cursor", cursor)


    const data = await apiClient<JsonRecord>(`/api/applications?${params}`)
    const list = Array.isArray(data.applications) ? data.applications : []

    applications.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    total.value = Number(data.total || applications.value.length) || 0
    const isBackward = page.value < lastPage.value
    hasMore.value = isBackward ? true : Boolean(data.has_more)
    lastPage.value = page.value
nextCursor.value = String(data.next_cursor || "")
    prevCursor.value = String(data?.prev_cursor || "")

    lastPage.value = targetPage
    selected.value = applications.value[0] || null
    activeTab.value = "overview"
    page.value = targetPage
    void loadApplicationDetail(selected.value)
  } catch (err) {
    console.error(err)
    applications.value = []
    selected.value = null
    total.value = 0
    hasMore.value = false
    nextCursor.value = ""
    toast.error(copy.value.toasts.listLoadFailed)
  } finally {
    loading.value = false
  }
}

async function audit(action: "approve" | "reject" | "resubmit") {
  if (!selected.value) return
  if ((action === "reject" || action === "resubmit") && !auditRemark.value.trim()) {
    toast.error(copy.value.toasts.remarkRequired)
    return
  }

  auditing.value = true
  try {
    await apiClient("/api/applications/audit", {
      method: "POST",
      body: JSON.stringify({
        application_id: appUlid(selected.value),
        approved: action === "approve",
        reject_reason: auditRemark.value,
        require_resubmit: action === "resubmit",
      }),
    })
    toast.success(copy.value.toasts.auditSubmitted)
    auditRemark.value = ""
    await load(page.value)
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.auditSubmitFailed))
  } finally {
    auditing.value = false
  }
}

function resetCursorPagination() {
  page.value = 1
  lastPage.value = 1

  prevCursor.value = ""
  nextCursor.value = ""
  hasMore.value = false
}

const downloadingFile = ref<string | null>(null)

async function handleDownload(file: JsonRecord) {
  const url = fileUrl(file)
  if (!url) return
  const name = fileName(file)
  const hash = String(file.file_hash || file.file_name || file.name)
  
  downloadingFile.value = hash
  try {
    const res = await fetch(url)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const blob = await res.blob()
    const blobUrl = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = blobUrl
    a.download = name
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    setTimeout(() => URL.revokeObjectURL(blobUrl), 1000)
  } catch (err) {
    console.error("Fetch download failed (likely CORS or network error), falling back to window.open:", err)
    window.open(url, "_blank")
  } finally {
    downloadingFile.value = null
  }
}

watch(statusFilter, () => {
  resetCursorPagination()
  void load(1)
})

onMounted(() => {
  desktopQuery = window.matchMedia("(min-width: 1024px)")
  syncDesktop(desktopQuery)
  desktopQuery.addEventListener("change", syncDesktop)
  void load(1)
})

onBeforeUnmount(() => {
  desktopQuery?.removeEventListener("change", syncDesktop)
})
</script>

<template>
  <main class="ui-page">
    <!-- Page header -->
    <header class="flex flex-wrap items-start justify-between gap-4 md:items-center">
      <div class="min-w-0">
        <h1 class="ui-page-title">{{ copy.title }}</h1>
        <p class="ui-page-subtitle">{{ copy.subtitle }}</p>
      </div>
      <button class="ui-btn ui-btn-secondary" type="button" @click="load(page)">
        <RefreshCw class="h-4 w-4 text-slate-400" :class="loading ? 'animate-spin' : ''" />
        {{ copy.refresh }}
      </button>
    </header>

    <!-- Status chips: click to query immediately (replaces the status select) -->
    <div class="mt-6 flex flex-wrap items-center gap-2">
      <button
        v-for="option in statusOptions"
        :key="option.value || 'all'"
        type="button"
        class="inline-flex h-8 shrink-0 items-center gap-1.5 whitespace-nowrap rounded-full px-3.5 text-sm font-medium transition-colors duration-150 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary-500/20"
        :class="statusFilter === option.value
          ? 'bg-primary-600 text-white shadow-sm'
          : 'border border-slate-200 bg-white text-slate-600 hover:border-slate-300 hover:bg-slate-50 hover:text-slate-900'"
        :aria-pressed="statusFilter === option.value"
        @click="statusFilter = option.value"
      >
        {{ option.label }}
      </button>
    </div>

    <!-- Desktop (lg+): split-view review workbench -->
    <section class="mt-4 hidden gap-6 lg:grid lg:grid-cols-[minmax(320px,2fr)_minmax(0,3fr)] lg:items-start">
      <!-- Left: review queue -->
      <section class="ui-card overflow-hidden">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-hairline px-4 py-4 md:px-5">
          <div class="min-w-0">
            <h2 class="text-base font-semibold text-ink">{{ copy.listTitle }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
          </div>
          <span class="tnum ui-badge shrink-0 bg-slate-100 text-slate-600">{{ copy.totalText(total) }}</span>
        </div>

        <div v-if="loading" class="divide-y divide-hairline">
          <div v-for="i in 6" :key="`qsk-${i}`" class="space-y-2 px-4 py-4">
            <div class="skeleton h-4 w-3/5" />
            <div class="skeleton h-3.5 w-4/5" />
            <div class="skeleton h-3 w-2/5" />
          </div>
        </div>
        <div v-else-if="!applications.length" class="ui-empty">
          <div class="ui-empty-icon">
            <CheckCircle2 class="h-6 w-6" />
          </div>
          <p class="ui-empty-title">{{ copy.empty }}</p>
        </div>
        <div v-else class="divide-y divide-hairline">
          <div
            v-for="app in applications"
            :key="`queue-${appUlid(app)}`"
            class="ui-table-row group flex w-full cursor-pointer items-start gap-3 px-4 py-3.5 text-left focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary-500/20"
            :class="appUlid(selected) === appUlid(app) ? 'bg-primary-50/60' : ''"
            role="button"
            tabindex="0"
            @click="selectApplication(app)"
            @keydown.enter.prevent="selectApplication(app)"
            @keydown.space.prevent="selectApplication(app)"
          >
            <div class="min-w-0 flex-1">
              <div class="truncate text-sm font-semibold text-ink">{{ credential(app) }}</div>
              <div class="mt-0.5 truncate text-sm text-slate-600">{{ copy.candidatePrefix }}{{ candidate(app) }}</div>
              <div class="tnum mt-1 text-xs text-slate-400">{{ formatDate(String(app.created_at || "")) }}</div>
            </div>
            <div class="flex shrink-0 flex-col items-end gap-2">
              <span class="ui-badge" :class="badgeClass(status(app))">{{ applicationLabel(status(app)) }}</span>
              <ChevronRight class="h-4 w-4 text-slate-300 transition-colors duration-150 group-hover:text-slate-400" />
            </div>
          </div>
        </div>

        <div class="flex items-center justify-end gap-2 border-t border-hairline px-4 py-3 md:px-5">
          <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canPrev" @click="load(page - 1)">{{ copy.prev }}</button>
          <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canNext" @click="load(page + 1)">{{ copy.next }}</button>
        </div>
      </section>

      <!-- Right: detail panel with sticky audit bar -->
      <section class="ui-card flex min-h-[420px] flex-col overflow-hidden lg:sticky lg:top-6 lg:max-h-[calc(100vh-3rem)]">
        <div v-if="!selected" class="ui-empty flex-1">
          <div class="ui-empty-icon">
            <Inbox class="h-6 w-6" />
          </div>
          <p class="ui-empty-title">{{ copy.detailTitle }}</p>
          <p class="ui-empty-hint">{{ copy.selectApplication }}</p>
        </div>
        <template v-else>
          <div class="flex items-start justify-between gap-4 border-b border-hairline px-4 py-4 md:px-5">
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <h2 class="break-words text-base font-semibold text-ink">{{ credential(selected) }}</h2>
                <span class="ui-badge shrink-0" :class="badgeClass(status(selected))">{{ applicationLabel(status(selected)) }}</span>
              </div>
              <p class="mt-1 break-all text-sm text-slate-600">{{ copy.candidatePrefix }}{{ candidate(selected) }}</p>
              <div class="mt-1.5 flex min-w-0 items-center gap-1 text-xs text-slate-400">
                <span class="shrink-0">{{ copy.appIdPrefix }}</span>
                <UlidChip :value="appUlid(selected)" />
              </div>
            </div>
            <Loader2 v-if="detailLoading" class="mt-1 h-4 w-4 shrink-0 animate-spin text-slate-400" />
          </div>

          <div class="min-h-0 flex-1 overflow-y-auto px-4 py-4 md:px-5 md:py-5">
            <!-- Candidate & application info -->
            <h3 class="text-sm font-semibold text-ink">{{ copy.tabs.overview }}</h3>
            <div class="mt-3 grid gap-x-6 gap-y-4 xl:grid-cols-2">
              <div
                v-for="field in selectedFields"
                :key="`panel-${field.key}`"
                class="grid content-start gap-1"
                :class="isStructuredValue(field.value) ? 'xl:col-span-2' : ''"
              >
                <span class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ field.label }}</span>
                <JsonPreview
                  v-if="isStructuredValue(field.value)"
                  :title="field.label"
                  :value="field.value"
                  :copy-label="copy.copyJson"
                  :copied-label="copy.copiedJson"
                  :copied-message="copy.toasts.jsonCopied"
                  :copy-error-message="copy.toasts.jsonCopyFailed"
                  max-height="256px"
                />
                <div v-else-if="isUlidField(field.key)">
                  <UlidChip :value="String(field.value ?? '')" />
                </div>
                <div v-else class="break-words text-sm font-medium text-ink">{{ field.displayValue }}</div>
              </div>
            </div>

            <!-- Materials & preview links -->
            <div class="mt-6 border-t border-hairline pt-5">
              <div class="flex items-center gap-2">
                <h3 class="text-sm font-semibold text-ink">{{ copy.tabs.files }}</h3>
                <span class="tnum ui-badge bg-slate-100 text-slate-600">{{ selectedFiles.length }}</span>
              </div>
              <div v-if="detailLoading && !selectedFiles.length" class="mt-3 flex items-center gap-2 rounded-2xl bg-slate-50 p-5 text-sm text-slate-500">
                <Loader2 class="h-4 w-4 animate-spin" />
                {{ copy.filesLoading }}
              </div>
              <div v-else-if="!selectedFiles.length" class="ui-empty py-8">
                <div class="ui-empty-icon">
                  <FileText class="h-6 w-6" />
                </div>
                <p class="ui-empty-title">{{ copy.noFiles }}</p>
              </div>
              <div v-else class="mt-3 space-y-3">
                <div v-for="file in selectedFiles" :key="`panel-${String(file.file_hash || file.file_name || file.name)}`" class="rounded-2xl border border-edge p-4">
                  <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
                    <div class="min-w-0">
                      <div class="flex items-center gap-2">
                        <FileText class="h-4 w-4 shrink-0 text-primary-600" />
                        <span class="truncate text-sm font-semibold text-ink">{{ fileName(file) }}</span>
                      </div>
                      <div class="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500">
                        <span class="min-w-0 break-all">{{ copy.fileMeta.usage }}{{ fileUsage(file) }}</span>
                        <span v-if="fileSize(file)" class="tnum">{{ copy.fileMeta.size }}{{ fileSize(file) }}</span>
                        <span v-if="file.file_ext">{{ copy.fileMeta.ext }}{{ file.file_ext }}</span>
                      </div>
                      <div v-if="fileHash(file)" class="tnum mt-2 break-all font-mono text-xs text-slate-400">{{ copy.fileMeta.sha256 }}{{ fileHash(file) }}</div>
                    </div>
                    <div class="flex shrink-0 flex-wrap gap-2 xl:justify-end">
                      <a
                        v-if="previewUrl(file)"
                        class="ui-btn ui-btn-secondary ui-btn-sm"
                        :href="previewUrl(file)"
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        <Eye class="h-4 w-4 text-slate-400" />
                        {{ copy.preview }}
                      </a>
                      <button
                        v-if="fileUrl(file)"
                        class="ui-btn ui-btn-secondary ui-btn-sm"
                        type="button"
                        :disabled="downloadingFile === String(file.file_hash || file.file_name || file.name)"
                        @click="handleDownload(file)"
                      >
                        <Loader2 v-if="downloadingFile === String(file.file_hash || file.file_name || file.name)" class="h-4 w-4 animate-spin" />
                        <Download v-else class="h-4 w-4 text-slate-400" />
                        {{ copy.download }}
                      </button>
                      <span v-else class="ui-badge bg-amber-50 text-amber-700">{{ copy.missingLink }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Sticky audit action bar -->
          <div v-if="!isApprovedSelected" class="shrink-0 border-t border-hairline bg-white/95 px-4 py-3.5 backdrop-blur md:px-5">
            <p class="text-xs text-slate-500">{{ copy.auditHint }}</p>
            <textarea
              v-model="auditRemark"
              class="ui-input mt-2 h-auto min-h-16 py-2.5"
              :placeholder="copy.auditRemarkPlaceholder"
            />
            <div class="mt-3 flex flex-wrap items-center justify-end gap-3">
              <button
                class="ui-btn ui-btn-secondary text-danger-600 hover:border-danger-200 hover:bg-danger-50"
                type="button"
                :disabled="auditing"
                @click="audit('reject')"
              >
                <XCircle class="h-4 w-4" />
                {{ copy.reject }}
              </button>
              <button class="ui-btn ui-btn-secondary" type="button" :disabled="auditing" @click="audit('resubmit')">
                <RotateCcw class="h-4 w-4 text-slate-400" />
                {{ copy.resubmit }}
              </button>
              <button class="ui-btn ui-btn-primary" type="button" :disabled="auditing" @click="audit('approve')">
                <Loader2 v-if="auditing" class="h-4 w-4 animate-spin" />
                <CheckCircle2 v-else class="h-4 w-4" />
                {{ copy.approve }}
              </button>
            </div>
          </div>
        </template>
      </section>
    </section>

    <!-- Mobile / tablet (<lg): existing list + dialog form -->
    <section class="ui-card mt-4 overflow-hidden lg:hidden">
      <div class="flex flex-wrap items-center justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
        <div class="flex min-w-0 flex-wrap items-center gap-3">
          <div class="min-w-0">
            <h2 class="text-base font-semibold text-ink">{{ copy.listTitle }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
          </div>
          <span class="tnum ui-badge shrink-0 bg-slate-100 text-slate-600">{{ copy.totalText(total) }}</span>
        </div>
      </div>

      <div v-if="loading" class="divide-y divide-hairline">
        <div v-for="i in 5" :key="`sk-${i}`" class="grid gap-3 px-4 py-3.5 md:grid-cols-[minmax(0,1fr)_160px_170px_112px] md:items-center md:gap-4">
          <div class="space-y-2">
            <div class="skeleton h-4 w-2/5" />
            <div class="skeleton h-3 w-3/5" />
          </div>
          <div class="skeleton h-5 w-16 rounded-full md:justify-self-center" />
          <div class="skeleton h-4 w-24 md:justify-self-end" />
          <div class="skeleton h-4 w-14 md:justify-self-end" />
        </div>
      </div>
      <div v-else-if="!applications.length" class="ui-empty">
        <div class="ui-empty-icon">
          <Inbox class="h-6 w-6" />
        </div>
        <p class="ui-empty-title">{{ copy.empty }}</p>
      </div>
      <template v-else>
        <div class="ui-table-head hidden grid-cols-[minmax(0,1fr)_160px_170px_112px] gap-4 px-4 py-3 md:grid">
          <span>{{ copy.columns.application }}</span>
          <span class="text-center">{{ copy.columns.status }}</span>
          <span class="text-right">{{ copy.columns.submittedAt }}</span>
          <span class="text-right">{{ copy.columns.action }}</span>
        </div>
        <div class="divide-y divide-hairline">
          <div
            v-for="app in applications"
            :key="appUlid(app)"
            class="ui-table-row flex w-full cursor-pointer flex-col px-4 py-4 text-left focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary-500/20 md:grid md:grid-cols-[minmax(0,1fr)_160px_170px_112px] md:items-center md:gap-4 md:py-3.5"
            :class="appUlid(selected) === appUlid(app) ? 'bg-primary-50/60' : ''"
            role="button"
            tabindex="0"
            @click="selectApplication(app)"
            @keydown.enter.prevent="selectApplication(app)"
            @keydown.space.prevent="selectApplication(app)"
          >
            <div class="min-w-0">
              <div class="break-words text-sm font-semibold text-ink md:truncate">{{ credential(app) }}</div>
              <div class="mt-1 break-all text-sm text-slate-600">{{ copy.candidatePrefix }}{{ candidate(app) }}</div>
              <div class="mt-1 flex min-w-0 items-center gap-1 text-xs text-slate-400">
                <span class="shrink-0">{{ copy.appIdPrefix }}</span>
                <UlidChip :value="appUlid(app)" />
              </div>
            </div>
            <div class="mt-3 flex items-center justify-between gap-4 border-t border-hairline py-2 md:mt-0 md:block md:border-0 md:py-0 md:text-center">
              <span class="text-xs font-medium text-slate-400 md:hidden">{{ copy.columns.status }}</span>
              <span class="ui-badge" :class="badgeClass(status(app))">{{ applicationLabel(status(app)) }}</span>
            </div>
            <div class="flex items-center justify-between gap-4 border-t border-hairline py-2 md:block md:border-0 md:py-0 md:text-right">
              <span class="text-xs font-medium text-slate-400 md:hidden">{{ copy.columns.submittedAt }}</span>
              <span class="tnum text-sm text-slate-600">{{ formatDate(String(app.created_at || "")) }}</span>
            </div>
            <div class="mt-1 md:mt-0 md:text-right">
              <button class="ui-btn ui-btn-secondary ui-btn-sm w-full md:w-auto" type="button" @click.stop="selectApplication(app)">
                {{ copy.viewDetails }}
              </button>
            </div>
          </div>
        </div>
      </template>

      <!-- Pagination -->
      <div class="flex flex-col items-stretch justify-end gap-3 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center md:px-6">
        <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canPrev" @click="load(page - 1)">{{ copy.prev }}</button>
        <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!canNext" @click="load(page + 1)">{{ copy.next }}</button>
      </div>
    </section>

    <!-- Detail dialog -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="detailOpen" class="ui-modal-mask">
          <section class="ui-modal-panel max-w-[1280px]">
            <div v-if="!selected" class="flex items-start justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
              <div class="min-w-0">
                <h2 class="text-base font-semibold text-ink md:text-lg">{{ copy.detailTitle }}</h2>
                <p class="mt-1 text-sm text-slate-500">{{ copy.selectApplication }}</p>
              </div>
              <button
                class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
                type="button"
                :aria-label="copy.close"
                @click="closeDetail"
              >
                <X class="h-5 w-5" />
              </button>
            </div>
            <template v-else>
              <div class="flex items-start justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
                <div class="min-w-0">
                  <h2 class="break-words text-base font-semibold text-ink md:text-lg">{{ credential(selected) }}</h2>
                  <div class="mt-1"><UlidChip :value="appUlid(selected)" /></div>
                </div>
                <div class="flex shrink-0 items-center gap-2">
                  <Loader2 v-if="detailLoading" class="h-4 w-4 animate-spin text-slate-400" />
                  <button
                    class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
                    type="button"
                    :aria-label="copy.close"
                    @click="closeDetail"
                  >
                    <X class="h-5 w-5" />
                  </button>
                </div>
              </div>

              <div class="flex shrink-0 gap-5 overflow-x-auto border-b border-hairline px-4 md:px-6">
                <button
                  v-for="tab in detailTabs"
                  :key="tab.key"
                  class="ui-tab"
                  :class="activeTab === tab.key ? 'ui-tab-active' : ''"
                  type="button"
                  @click="activeTab = tab.key"
                >
                  {{ tab.title }}
                  <span class="tnum ui-badge bg-slate-100 text-slate-600">{{ tab.count }}</span>
                </button>
              </div>

              <main class="min-h-0 flex-1 overflow-y-auto p-4 md:h-[60vh] md:max-h-[620px] md:min-h-[360px] md:p-6">
                <div v-if="activeTab === 'overview'" class="grid gap-4 md:grid-cols-2">
                  <div v-for="field in selectedFields" :key="field.key" class="grid content-start gap-1.5" :class="isStructuredValue(field.value) ? 'md:col-span-2' : ''">
                    <span class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ field.label }}</span>
                    <JsonPreview
                      v-if="isStructuredValue(field.value)"
                      :title="field.label"
                      :value="field.value"
                      :copy-label="copy.copyJson"
                      :copied-label="copy.copiedJson"
                      :copied-message="copy.toasts.jsonCopied"
                      :copy-error-message="copy.toasts.jsonCopyFailed"
                      max-height="256px"
                    />
                    <div v-else-if="isUlidField(field.key)">
                      <UlidChip :value="String(field.value ?? '')" />
                    </div>
                    <div v-else class="break-words text-sm font-medium text-ink">
                      {{ field.displayValue }}
                    </div>
                  </div>
                </div>

                <div v-else-if="activeTab === 'files'" class="space-y-4">
                  <div v-if="detailLoading" class="flex items-center gap-2 rounded-2xl bg-slate-50 p-5 text-sm text-slate-500">
                    <Loader2 class="h-4 w-4 animate-spin" />
                    {{ copy.filesLoading }}
                  </div>
                  <div v-else-if="!selectedFiles.length" class="ui-empty">
                    <div class="ui-empty-icon">
                      <FileText class="h-6 w-6" />
                    </div>
                    <p class="ui-empty-title">{{ copy.noFiles }}</p>
                  </div>
                  <div v-for="file in selectedFiles" v-else :key="String(file.file_hash || file.file_name || file.name)" class="rounded-2xl border border-edge p-4 md:p-5">
                    <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                      <div class="min-w-0">
                        <div class="flex items-center gap-2">
                          <FileText class="h-4 w-4 shrink-0 text-primary-600" />
                          <span class="truncate text-sm font-semibold text-ink">{{ fileName(file) }}</span>
                        </div>
                        <div class="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500">
                          <span class="min-w-0 break-all">{{ copy.fileMeta.usage }}{{ fileUsage(file) }}</span>
                          <span v-if="fileSize(file)" class="tnum">{{ copy.fileMeta.size }}{{ fileSize(file) }}</span>
                          <span v-if="file.file_ext">{{ copy.fileMeta.ext }}{{ file.file_ext }}</span>
                        </div>
                        <div v-if="fileHash(file)" class="tnum mt-2 break-all font-mono text-xs text-slate-400">{{ copy.fileMeta.sha256 }}{{ fileHash(file) }}</div>
                      </div>
                      <div class="flex shrink-0 flex-wrap gap-2 sm:justify-end">
                        <a
                          v-if="previewUrl(file)"
                          class="ui-btn ui-btn-secondary ui-btn-sm w-full sm:w-auto"
                          :href="previewUrl(file)"
                          target="_blank"
                          rel="noopener noreferrer"
                        >
                          <Eye class="h-4 w-4 text-slate-400" />
                          {{ copy.preview }}
                        </a>
                        <button
                          v-if="fileUrl(file)"
                          class="ui-btn ui-btn-secondary ui-btn-sm w-full sm:w-auto"
                          type="button"
                          :disabled="downloadingFile === String(file.file_hash || file.file_name || file.name)"
                          @click="handleDownload(file)"
                        >
                          <Loader2 v-if="downloadingFile === String(file.file_hash || file.file_name || file.name)" class="h-4 w-4 animate-spin" />
                          <Download v-else class="h-4 w-4 text-slate-400" />
                          {{ copy.download }}
                        </button>
                        <span v-else class="ui-badge w-full justify-center bg-amber-50 text-amber-700 sm:w-auto">{{ copy.missingLink }}</span>
                      </div>
                    </div>
                  </div>
                </div>

                <div v-else-if="activeTab === 'audit'" class="space-y-4">
                  <div class="rounded-2xl bg-slate-50 p-4 text-sm text-slate-600">
                    {{ copy.auditHint }}
                  </div>
                  <textarea
                    v-model="auditRemark"
                    class="ui-input h-auto min-h-32 py-2.5"
                    :placeholder="copy.auditRemarkPlaceholder"
                  />
                  <div class="grid gap-3 md:grid-cols-3">
                    <button
                      class="ui-btn ui-btn-secondary text-danger-600 hover:border-danger-200 hover:bg-danger-50"
                      type="button"
                      :disabled="auditing"
                      @click="audit('reject')"
                    >
                      <XCircle class="h-4 w-4" />
                      {{ copy.reject }}
                    </button>
                    <button class="ui-btn ui-btn-secondary" type="button" :disabled="auditing" @click="audit('resubmit')">
                      <RotateCcw class="h-4 w-4 text-slate-400" />
                      {{ copy.resubmit }}
                    </button>
                    <button class="ui-btn ui-btn-primary" type="button" :disabled="auditing" @click="audit('approve')">
                      <Loader2 v-if="auditing" class="h-4 w-4 animate-spin" />
                      <CheckCircle2 v-else class="h-4 w-4" />
                      {{ copy.approve }}
                    </button>
                  </div>
                </div>
              </main>
            </template>
          </section>
        </div>
      </Transition>
    </Teleport>
  </main>
</template>
