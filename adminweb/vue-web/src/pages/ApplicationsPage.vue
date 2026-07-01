<script setup lang="ts">
import { CheckCircle2, Download, Eye, FileText, Loader2, RefreshCw, RotateCcw, XCircle } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { applicationStatusLabel, applicationStatusOptions, badgeClass, pickFirst } from "@/lib/status"

type DetailTab = "overview" | "files" | "audit" | "raw"

const applications = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const loading = ref(false)
const detailLoading = ref(false)
const auditing = ref(false)
const page = ref(1)
const total = ref(0)
const statusFilter = ref("0")
const auditRemark = ref("")
const activeTab = ref<DetailTab>("overview")
const pageSize = 20
let detailRequestId = 0

const canPrev = computed(() => page.value > 1)
const canNext = computed(() => applications.value.length >= pageSize)
const selectedFields = computed(() => selected.value || {})
const selectedFiles = computed(() => files(selected.value || {}))
const detailTabs = computed(() => [
  { key: "overview" as const, title: "概览", count: selected.value ? 1 : 0 },
  { key: "files" as const, title: "申请材料", count: selectedFiles.value.length },
  { key: "audit" as const, title: "审核操作", count: 3 },
  { key: "raw" as const, title: "完整字段", count: 1 },
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
  return String(pickFirst(file, ["file_name", "name", "filename", "file_hash"]) || "文件")
}

function fileUrl(file: JsonRecord) {
  return String(pickFirst(file, ["view_url", "download_url", "url"]) || "")
}

function fileUsage(file: JsonRecord) {
  return String(pickFirst(file, ["file_usage", "usage"]) || "申请材料")
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

function mergeApplicationDetail(appID: string, detail: JsonRecord) {
  const index = applications.value.findIndex((app) => appUlid(app) === appID)
  const base = index >= 0 ? applications.value[index] : selected.value || {}
  const merged = { ...base, ...detail }
  if (index >= 0) {
    applications.value.splice(index, 1, merged)
  }
  if (selected.value && appUlid(selected.value) === appID) {
    selected.value = merged
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
      toast.error("申请材料加载失败")
    }
  } finally {
    if (requestId === detailRequestId) {
      detailLoading.value = false
    }
  }
}

function selectApplication(app: JsonRecord) {
  selected.value = app
  auditRemark.value = ""
  activeTab.value = "overview"
  void loadApplicationDetail(app)
}

async function load(targetPage = page.value) {
  loading.value = true
  try {
    const data = await apiClient<JsonRecord>(`/api/applications?page_number=${targetPage}&page_size=${pageSize}&status=${statusFilter.value}`)
    const list = Array.isArray(data.applications) ? data.applications : []
    applications.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    total.value = Number(data.total || applications.value.length) || 0
    selected.value = applications.value[0] || null
    activeTab.value = "overview"
    page.value = targetPage
    void loadApplicationDetail(selected.value)
  } catch (err) {
    console.error(err)
    applications.value = []
    selected.value = null
    total.value = 0
    toast.error("申请列表加载失败")
  } finally {
    loading.value = false
  }
}

async function audit(action: "approve" | "reject" | "resubmit") {
  if (!selected.value) return
  if ((action === "reject" || action === "resubmit") && !auditRemark.value.trim()) {
    toast.error("拒绝或要求补交时需要填写审核备注")
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
    toast.success("审核已提交")
    auditRemark.value = ""
    await load(page.value)
  } catch (err) {
    console.error(err)
    toast.error("审核提交失败")
  } finally {
    auditing.value = false
  }
}

watch(statusFilter, () => load(1))
onMounted(() => load(1))
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1520px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">审核中心</h1>
        <p class="mt-2 text-slate-600">审核考生提交的资格申请材料。</p>
        <p class="mt-2 text-xs font-semibold text-slate-500">
          已确认接口：list/get detail/audit。文件预览与下载使用申请详情返回的 view_url。
        </p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load(page)">
        <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
        刷新
      </button>
    </header>

    <div class="grid gap-6 xl:grid-cols-[460px_minmax(0,1fr)]">
      <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="space-y-4 border-b border-slate-200 p-5">
          <div class="flex items-center justify-between gap-3">
            <div>
              <h2 class="text-xl font-black">申请列表</h2>
              <p class="mt-1 text-sm text-slate-500">来自 `/api/applications`。</p>
            </div>
            <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">共 {{ total }} 条</span>
          </div>
          <select v-model="statusFilter" class="w-full rounded-xl border border-slate-200 px-4 py-3">
            <option v-for="option in applicationStatusOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </div>

        <div v-if="loading" class="p-12 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          正在加载...
        </div>
        <div v-else-if="!applications.length" class="p-12 text-center text-slate-500">暂无申请</div>
        <button
          v-for="app in applications"
          v-else
          :key="appUlid(app)"
          class="w-full border-b border-slate-100 px-5 py-4 text-left last:border-b-0 hover:bg-sky-50"
          :class="appUlid(selected) === appUlid(app) ? 'bg-sky-50' : ''"
          type="button"
          @click="selectApplication(app)"
        >
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="truncate font-black text-slate-950">{{ credential(app) }}</div>
              <div class="mt-1 break-all text-sm text-slate-500">{{ candidate(app) }}</div>
            </div>
            <span class="h-fit shrink-0 rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(status(app))">
              {{ applicationStatusLabel(status(app)) }}
            </span>
          </div>
          <div class="mt-2 break-all text-xs font-semibold text-slate-500">申请 ID：{{ appUlid(app) || "-" }}</div>
          <div class="mt-1 text-xs text-slate-400">{{ formatDate(String(app.created_at || "")) }}</div>
        </button>

        <div class="flex justify-end gap-3 border-t border-slate-200 p-5">
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrev" @click="load(page - 1)">上一页</button>
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext" @click="load(page + 1)">下一页</button>
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div v-if="!selected" class="p-10 text-center text-slate-500">请选择一条申请</div>
        <template v-else>
          <div class="border-b border-slate-200 p-5">
            <div class="flex flex-wrap items-start justify-between gap-4">
              <div>
                <h2 class="text-2xl font-black">{{ credential(selected) }}</h2>
                <p class="mt-1 break-all text-sm text-slate-500">{{ appUlid(selected) }}</p>
              </div>
              <span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(status(selected))">
                {{ applicationStatusLabel(status(selected)) }}
              </span>
            </div>
          </div>

          <div class="grid min-h-[720px] lg:grid-cols-[240px_minmax(0,1fr)]">
            <aside class="border-b border-slate-200 p-4 lg:border-b-0 lg:border-r">
              <div class="space-y-2">
                <button
                  v-for="tab in detailTabs"
                  :key="tab.key"
                  class="w-full rounded-2xl border px-4 py-3 text-left"
                  :class="activeTab === tab.key ? 'border-sky-200 bg-sky-50' : 'border-slate-100 hover:bg-slate-50'"
                  type="button"
                  @click="activeTab = tab.key"
                >
                  <div class="flex items-center justify-between gap-3">
                    <span class="font-black">{{ tab.title }}</span>
                    <span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-black text-slate-600">{{ tab.count }}</span>
                  </div>
                </button>
              </div>
            </aside>

            <main class="min-w-0 p-5">
              <div v-if="activeTab === 'overview'" class="grid gap-4 md:grid-cols-2">
                <label v-for="(value, key) in selectedFields" :key="key" class="grid gap-2 text-sm font-bold">
                  {{ key }}
                  <textarea
                    v-if="Array.isArray(value) || (value && typeof value === 'object')"
                    class="min-h-24 rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-600"
                    disabled
                    :value="JSON.stringify(value, null, 2)"
                  />
                  <input
                    v-else
                    class="rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-600"
                    disabled
                    :value="key === 'status' ? applicationStatusLabel(value) : String(value ?? '-')"
                  />
                </label>
              </div>

              <div v-else-if="activeTab === 'files'" class="space-y-4">
                <div v-if="detailLoading" class="flex items-center gap-2 rounded-2xl border border-slate-200 bg-slate-50 p-5 text-sm text-slate-500">
                  <Loader2 class="h-4 w-4 animate-spin" />
                  正在加载申请材料...
                </div>
                <div v-else-if="!selectedFiles.length" class="rounded-xl border border-dashed border-slate-200 bg-white p-5 text-sm text-slate-500">
                  暂无文件。若考生已上传材料，请检查微服务申请详情接口是否返回 files / view_url。
                </div>
                <div v-for="file in selectedFiles" v-else :key="String(file.file_hash || file.file_name || file.name)" class="rounded-2xl border border-slate-200 bg-white p-4">
                  <div class="flex items-start justify-between gap-4">
                    <div class="min-w-0">
                      <div class="flex items-center gap-2 font-black text-slate-900">
                        <FileText class="h-4 w-4 shrink-0 text-blue-600" />
                        <span class="truncate">{{ fileName(file) }}</span>
                      </div>
                      <div class="mt-2 flex flex-wrap gap-2 text-xs font-bold text-slate-500">
                        <span class="rounded-full bg-slate-100 px-2 py-1">用途：{{ fileUsage(file) }}</span>
                        <span v-if="fileSize(file)" class="rounded-full bg-slate-100 px-2 py-1">大小：{{ fileSize(file) }}</span>
                        <span v-if="file.file_ext" class="rounded-full bg-slate-100 px-2 py-1">格式：{{ file.file_ext }}</span>
                      </div>
                      <div v-if="fileHash(file)" class="mt-2 break-all text-xs text-slate-400">SHA256：{{ fileHash(file) }}</div>
                    </div>
                    <div class="flex shrink-0 gap-2">
                      <a
                        v-if="fileUrl(file)"
                        class="inline-flex items-center gap-1 rounded-xl border border-blue-200 bg-blue-50 px-3 py-2 text-xs font-black text-blue-700 hover:bg-blue-100"
                        :href="fileUrl(file)"
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        <Eye class="h-4 w-4" />
                        预览
                      </a>
                      <a
                        v-if="fileUrl(file)"
                        class="inline-flex items-center gap-1 rounded-xl border border-slate-200 px-3 py-2 text-xs font-black text-slate-700 hover:bg-slate-50"
                        :href="fileUrl(file)"
                        :download="fileName(file)"
                      >
                        <Download class="h-4 w-4" />
                        下载
                      </a>
                      <span v-else class="rounded-xl border border-amber-200 bg-amber-50 px-3 py-2 text-xs font-black text-amber-700">缺少链接</span>
                    </div>
                  </div>
                </div>
              </div>

              <div v-else-if="activeTab === 'audit'" class="space-y-4">
                <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
                  审核接口支持：通过、打回重提、最终拒绝。打回重提或最终拒绝时必须填写备注。
                </div>
                <textarea
                  v-model="auditRemark"
                  class="min-h-32 w-full rounded-2xl border border-slate-200 p-4 text-sm"
                  placeholder="审核备注：打回重提或最终拒绝时必填。需要用户重新提交材料时请选择“打回重提”。"
                />
                <div class="grid gap-3 md:grid-cols-3">
                  <button class="inline-flex items-center justify-center gap-2 rounded-xl bg-emerald-600 px-4 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="auditing" @click="audit('approve')">
                    <CheckCircle2 class="h-4 w-4" />
                    通过
                  </button>
                  <button class="inline-flex items-center justify-center gap-2 rounded-xl bg-amber-500 px-4 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="auditing" @click="audit('resubmit')">
                    <RotateCcw class="h-4 w-4" />
                    打回重提
                  </button>
                  <button class="inline-flex items-center justify-center gap-2 rounded-xl bg-red-600 px-4 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="auditing" @click="audit('reject')">
                    <XCircle class="h-4 w-4" />
                    最终拒绝
                  </button>
                </div>
              </div>

              <div v-else-if="activeTab === 'raw'" class="space-y-4">
                <div class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
                  完整字段只读展示，方便核对微服务返回。
                </div>
                <pre class="max-h-[560px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selected, null, 2) }}</pre>
              </div>
            </main>
          </div>
        </template>
      </section>
    </div>
  </section>
</template>
