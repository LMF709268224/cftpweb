<script setup lang="ts">
import { CheckCircle2, Copy, Download, Eye, FileText, Loader2, RefreshCw, RotateCcw, X, XCircle } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { copyTextToClipboard } from "@/lib/clipboard"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { badgeClass, pickFirst } from "@/lib/status"

type DetailTab = "overview" | "files" | "audit" | "raw"

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
const statusFilter = ref("0")
const auditRemark = ref("")
const activeTab = ref<DetailTab>("overview")
const copiedRawJson = ref(false)
const pageSize = 20
let detailRequestId = 0
const { t } = useAdminLanguage()
const copy = computed(() => t.value.applications)

const canPrev = computed(() => page.value > 1)
const canNext = computed(() => hasMore.value)
const applicationFieldLabels = computed<Record<string, string>>(() => copy.value.fieldLabels || {})
const applicationIdKeys = new Set(["app_ulid", "app_id", "application_ulid", "application_id"])
const selectedFields = computed(() => {
  const current = selected.value || {}
  let hasApplicationId = false
  const hasDuplicateCredentialName =
    "cred_def_name" in current &&
    "credential_name" in current &&
    String(current.cred_def_name ?? "") === String(current.credential_name ?? "")

  return Object.entries(current)
    .filter(([key, value]) => {
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
const selectedJson = computed(() => JSON.stringify(selected.value || {}, null, 2))
const isApprovedSelected = computed(() => isApprovedApplication(selected.value))
const detailTabs = computed(() => {
  const tabs: Array<{ key: DetailTab; title: string; count: number }> = [
    { key: "overview" as const, title: copy.value.tabs.overview, count: selected.value ? 1 : 0 },
    { key: "files" as const, title: copy.value.tabs.files, count: selectedFiles.value.length },
  ]
  if (!isApprovedSelected.value) {
    tabs.push({ key: "audit" as const, title: copy.value.tabs.audit, count: 3 })
  }
  tabs.push({ key: "raw" as const, title: copy.value.tabs.raw, count: 1 })
  return tabs
})
const statusOptions = computed(() => [
  { value: "0", label: copy.value.statusOptions.all },
  { value: "1", label: copy.value.statusOptions.pending },
  { value: "2", label: copy.value.statusOptions.approved },
  { value: "3", label: copy.value.statusOptions.rejected },
  { value: "4", label: copy.value.statusOptions.resubmit },
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

function jsonText(value: unknown) {
  return JSON.stringify(value ?? {}, null, 2)
}

async function copyRawJson() {
  try {
    await copyTextToClipboard(selectedJson.value)
    copiedRawJson.value = true
    toast.success(copy.value.toasts.jsonCopied)
    window.setTimeout(() => {
      copiedRawJson.value = false
    }, 1600)
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.jsonCopyFailed)
  }
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
  detailOpen.value = true
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
    hasMore.value = Boolean(data.has_more)
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

watch(statusFilter, () => {
  resetCursorPagination()
  void load(1)
})
onMounted(() => load(1))
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1520px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">{{ copy.title }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load(page)">
        <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
        {{ copy.refresh }}
      </button>
    </header>

    <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 p-5">
          <div class="flex items-center gap-3">
            <div>
              <h2 class="text-xl font-black">{{ copy.listTitle }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
            </div>
            <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ copy.totalText(total) }}</span>
          </div>
          <select v-model="statusFilter" class="h-10 w-full rounded-xl border border-slate-200 px-4 text-sm md:w-64">
            <option v-for="option in statusOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </div>

        <div v-if="loading" class="p-12 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          {{ copy.loading }}
        </div>
        <div v-else-if="!applications.length" class="p-12 text-center text-slate-500">{{ copy.empty }}</div>
        <template v-else>
          <div class="grid grid-cols-[minmax(0,1fr)_160px_170px_112px] gap-4 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-500">
            <span>{{ copy.columns.application }}</span>
            <span class="text-center">{{ copy.columns.status }}</span>
            <span class="text-right">{{ copy.columns.submittedAt }}</span>
            <span class="text-right">{{ copy.columns.action }}</span>
          </div>
          <div
            v-for="app in applications"
            :key="appUlid(app)"
            class="grid w-full cursor-pointer grid-cols-[minmax(0,1fr)_160px_170px_112px] gap-4 border-b border-slate-100 px-5 py-4 text-left transition last:border-b-0 hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-200"
            :class="appUlid(selected) === appUlid(app) ? 'bg-sky-50' : ''"
            role="button"
            tabindex="0"
            @click="selectApplication(app)"
            @keydown.enter.prevent="selectApplication(app)"
            @keydown.space.prevent="selectApplication(app)"
          >
            <div class="min-w-0">
              <div class="truncate text-lg font-black text-slate-950">{{ credential(app) }}</div>
              <div class="mt-1 break-all text-sm text-slate-500">{{ copy.candidatePrefix }}{{ candidate(app) }}</div>
              <div class="mt-2 break-all text-xs font-semibold text-slate-500">{{ copy.appIdPrefix }}{{ appUlid(app) || "-" }}</div>
            </div>
            <span class="self-center justify-self-center rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(status(app))">
              {{ applicationLabel(status(app)) }}
            </span>
            <span class="self-center justify-self-end text-sm font-semibold text-slate-500">{{ formatDate(String(app.created_at || "")) }}</span>
            <button class="self-center justify-self-end text-sm font-bold text-[#1890ff] transition hover:underline" type="button" @click.stop="selectApplication(app)">
              {{ copy.viewDetails }}
            </button>
          </div>
        </template>

        <div class="flex justify-end gap-3 border-t border-slate-200 p-5">
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrev" @click="load(page - 1)">{{ copy.prev }}</button>
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext" @click="load(page + 1)">{{ copy.next }}</button>
        </div>
    </section>

    <Teleport to="body">
      <div v-if="detailOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/45 p-6">
        <section class="flex max-h-[88vh] w-full max-w-[1280px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
        <div v-if="!selected" class="flex items-start justify-between gap-4 p-6">
          <div>
            <h2 class="text-2xl font-black">{{ copy.detailTitle }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.selectApplication }}</p>
          </div>
          <button class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="closeDetail">
            <X class="h-5 w-5" />
          </button>
        </div>
        <template v-else>
          <div class="border-b border-slate-200 p-5">
            <div class="flex flex-wrap items-start justify-between gap-4">
              <div>
                <h2 class="text-2xl font-black">{{ credential(selected) }}</h2>
                <p class="mt-1 break-all text-sm text-slate-500">{{ appUlid(selected) }}</p>
              </div>
              <div class="flex items-center gap-3">
                <Loader2 v-if="detailLoading" class="h-4 w-4 animate-spin text-slate-400" />
                <button class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="closeDetail">
                  <X class="h-5 w-5" />
                </button>
              </div>
            </div>
          </div>

          <div class="border-b border-slate-200 px-5 py-4">
            <div class="flex gap-2 overflow-x-auto">
                <button
                  v-for="tab in detailTabs"
                  :key="tab.key"
                  class="inline-flex h-11 shrink-0 items-center gap-3 rounded-2xl border px-4 text-left text-sm font-black transition"
                  :class="activeTab === tab.key ? 'border-sky-200 bg-sky-50' : 'border-slate-100 hover:bg-slate-50'"
                  type="button"
                  @click="activeTab = tab.key"
                >
                  <span>{{ tab.title }}</span>
                  <span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-black text-slate-600">{{ tab.count }}</span>
                </button>
            </div>
          </div>

          <main class="h-[60vh] min-h-[360px] max-h-[620px] min-w-0 overflow-y-auto p-5">
              <div v-if="activeTab === 'overview'" class="grid gap-4 md:grid-cols-2">
                <div v-for="field in selectedFields" :key="field.key" class="grid gap-2 text-sm font-bold" :class="isStructuredValue(field.value) ? 'md:col-span-2' : ''">
                  <span class="text-xs font-black uppercase text-slate-400">{{ field.label }}</span>
                  <pre
                    v-if="isStructuredValue(field.value)"
                    class="max-h-64 overflow-auto whitespace-pre-wrap break-words rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 font-mono text-xs leading-5 text-slate-700"
                  >{{ jsonText(field.value) }}</pre>
                  <div v-else class="min-h-11 break-words rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm font-bold leading-5 text-slate-700">
                    {{ field.displayValue }}
                  </div>
                </div>
              </div>

              <div v-else-if="activeTab === 'files'" class="space-y-4">
                <div v-if="detailLoading" class="flex items-center gap-2 rounded-2xl border border-slate-200 bg-slate-50 p-5 text-sm text-slate-500">
                  <Loader2 class="h-4 w-4 animate-spin" />
                  {{ copy.filesLoading }}
                </div>
                <div v-else-if="!selectedFiles.length" class="rounded-xl border border-dashed border-slate-200 bg-white p-5 text-sm text-slate-500">
                  {{ copy.noFiles }}
                </div>
                <div v-for="file in selectedFiles" v-else :key="String(file.file_hash || file.file_name || file.name)" class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
                  <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                    <div class="min-w-0">
                      <div class="flex items-center gap-2 text-sm font-black text-slate-900">
                        <FileText class="h-4 w-4 shrink-0 text-blue-600" />
                        <span class="truncate">{{ fileName(file) }}</span>
                      </div>
                      <div class="mt-3 grid gap-2 text-xs font-bold text-slate-600 sm:grid-cols-3 lg:max-w-2xl">
                        <span class="min-w-0 truncate rounded-xl border border-slate-200 bg-slate-50 px-3 py-2">{{ copy.fileMeta.usage }}{{ fileUsage(file) }}</span>
                        <span v-if="fileSize(file)" class="min-w-0 truncate rounded-xl border border-slate-200 bg-slate-50 px-3 py-2">{{ copy.fileMeta.size }}{{ fileSize(file) }}</span>
                        <span v-if="file.file_ext" class="min-w-0 truncate rounded-xl border border-slate-200 bg-slate-50 px-3 py-2">{{ copy.fileMeta.ext }}{{ file.file_ext }}</span>
                      </div>
                      <div v-if="fileHash(file)" class="mt-3 rounded-xl border border-slate-200 bg-slate-50 px-3 py-2">
                        <div class="text-[11px] font-black uppercase text-slate-400">{{ copy.fileMeta.sha256 }}</div>
                        <div class="mt-1 break-all font-mono text-xs leading-5 text-slate-600">{{ fileHash(file) }}</div>
                      </div>
                    </div>
                    <div class="flex shrink-0 flex-wrap gap-2 sm:justify-end">
                      <a
                        v-if="fileUrl(file)"
                        class="inline-flex h-9 items-center gap-1 rounded-xl border border-blue-200 bg-blue-50 px-3 text-xs font-black text-blue-700 hover:bg-blue-100"
                        :href="fileUrl(file)"
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        <Eye class="h-4 w-4" />
                        {{ copy.preview }}
                      </a>
                      <a
                        v-if="fileUrl(file)"
                        class="inline-flex h-9 items-center gap-1 rounded-xl border border-slate-200 px-3 text-xs font-black text-slate-700 hover:bg-slate-50"
                        :href="fileUrl(file)"
                        :download="fileName(file)"
                      >
                        <Download class="h-4 w-4" />
                        {{ copy.download }}
                      </a>
                      <span v-else class="rounded-xl border border-amber-200 bg-amber-50 px-3 py-2 text-xs font-black text-amber-700">{{ copy.missingLink }}</span>
                    </div>
                  </div>
                </div>
              </div>

              <div v-else-if="activeTab === 'audit'" class="space-y-4">
                <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
                  {{ copy.auditHint }}
                </div>
                <textarea
                  v-model="auditRemark"
                  class="min-h-32 w-full rounded-2xl border border-slate-200 p-4 text-sm"
                  :placeholder="copy.auditRemarkPlaceholder"
                />
                <div class="grid gap-3 md:grid-cols-3">
                  <button class="inline-flex items-center justify-center gap-2 rounded-xl bg-emerald-600 px-4 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="auditing" @click="audit('approve')">
                    <CheckCircle2 class="h-4 w-4" />
                    {{ copy.approve }}
                  </button>
                  <button class="inline-flex items-center justify-center gap-2 rounded-xl bg-amber-500 px-4 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="auditing" @click="audit('resubmit')">
                    <RotateCcw class="h-4 w-4" />
                    {{ copy.resubmit }}
                  </button>
                  <button class="inline-flex items-center justify-center gap-2 rounded-xl bg-red-600 px-4 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="auditing" @click="audit('reject')">
                    <XCircle class="h-4 w-4" />
                    {{ copy.reject }}
                  </button>
                </div>
              </div>

              <div v-else-if="activeTab === 'raw'" class="space-y-4">
                <div class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
                  {{ copy.rawHint }}
                </div>
                <details class="rounded-2xl border border-slate-200 bg-white p-4">
                  <summary class="cursor-pointer text-sm font-black text-slate-700">{{ copy.rawJson }}</summary>
                  <div class="mt-4 overflow-hidden rounded-2xl bg-slate-950">
                    <div class="flex items-center justify-between gap-3 border-b border-white/10 px-4 py-3">
                      <span class="text-xs font-black uppercase text-slate-400">{{ copy.rawJson }}</span>
                      <button class="inline-flex h-8 items-center gap-2 rounded-lg border border-white/10 px-3 text-xs font-bold text-slate-100 transition hover:bg-white/10" type="button" @click="copyRawJson">
                        <CheckCircle2 v-if="copiedRawJson" class="h-3.5 w-3.5" />
                        <Copy v-else class="h-3.5 w-3.5" />
                        {{ copiedRawJson ? copy.copiedJson : copy.copyJson }}
                      </button>
                    </div>
                    <pre class="max-h-[520px] overflow-auto p-5 text-xs leading-6 text-slate-100">{{ selectedJson }}</pre>
                  </div>
                </details>
              </div>
          </main>
        </template>
        </section>
      </div>
    </Teleport>
  </section>
</template>
