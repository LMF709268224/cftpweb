<script setup lang="ts">
import { onMounted, ref } from "vue"
import { useRoute } from "vue-router"
import { AlertCircle, Award, CheckCircle, Clock, FileText, Loader2, X, XCircle } from "lucide-vue-next"
import { CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, CANDIDATE_APPLICATION_STATUS_LABELS, statusEnumNameForStatus, statusLabel } from "@/lib/status-labels"
import AppPagination from "@/components/AppPagination.vue"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { formatBackendDateOnly } from "@/lib/utils"
import { useTranslation } from "@/lib/language"
import { toast } from "vue-sonner"

const { t, lang } = useTranslation()
const route = useRoute()
const definitions = ref<any[]>([])
const applications = ref<any[]>([])
const applicationPage = ref(1)
const applicationPageSize = ref(10)
const applicationPageSizeOptions = [10, 30, 50, 100]
const applicationTotal = ref(0)
const applicationTotalPages = ref(0)
const loading = ref(true)
const applicationsLoading = ref(false)
const selectedDef = ref<any>(null)
const resubmitAppId = ref("")
const isApplyOpen = ref(false)
const uploadedFiles = ref<Record<string, { name: string; url: string; ext: string; hash: string; size: number }>>({})
const isSubmitting = ref(false)
const uploadingConstraintName = ref("")
const UPLOAD_TIMEOUT_MS = 30000

async function sha256Hex(file: File) {
  const buffer = await file.arrayBuffer()
  const hash = await crypto.subtle.digest("SHA-256", buffer)
  return Array.from(new Uint8Array(hash)).map((byte) => byte.toString(16).padStart(2, "0")).join("")
}

async function uploadWithTimeout(url: string, init: RequestInit) {
  const controller = new AbortController()
  const timeoutId = window.setTimeout(() => controller.abort(), UPLOAD_TIMEOUT_MS)
  try {
    return await fetch(url, { ...init, signal: controller.signal })
  } finally {
    window.clearTimeout(timeoutId)
  }
}

function totalFrom(data: any, list: any[]) {
  return Number(data?.total ?? data?.total_count ?? data?.total_items ?? list.length ?? 0) || 0
}

function totalPagesFrom(data: any, total: number, pageSize: number) {
  return Number(data?.total_pages || Math.ceil(total / pageSize) || 0)
}

async function fetchApplications(options: { showLoading?: boolean } = {}) {
  if (options.showLoading) applicationsLoading.value = true
  try {
    const params = new URLSearchParams({
      page: String(applicationPage.value),
      page_size: String(applicationPageSize.value),
    })
    const appsRes = await apiClient(`/api/credentials/applications?${params.toString()}`)
    const nextApplications = appsRes?.applications || []
    applications.value = nextApplications
    applicationTotal.value = totalFrom(appsRes, nextApplications)
    applicationTotalPages.value = totalPagesFrom(appsRes, applicationTotal.value, applicationPageSize.value)
  } finally {
    if (options.showLoading) applicationsLoading.value = false
  }
}

async function fetchData() {
  loading.value = true
  try {
    const qualIds = String(route.query.qual_ulids || route.query.qual_ids || "").trim()
    const definitionsEndpoint = qualIds ? `/api/credentials/definitions?qual_ulids=${encodeURIComponent(qualIds)}` : "/api/credentials/definitions"
    const defsRes = await apiClient(definitionsEndpoint)
    definitions.value = defsRes?.definitions || []
    await fetchApplications()
    if (qualIds && definitions.value.length === 1 && !isApplyOpen.value) {
      handleApplyClick(definitions.value[0])
    }
  } finally {
    loading.value = false
  }
}

async function handleApplicationPageChange() {
  await fetchApplications({ showLoading: true })
}

function handleApplyClick(def: any, appId = "") {
  if (!def) return
  const existing = latestApplicationForDef(credentialDefinitionId(def))
  if (!appId && existing && !canStartNewApplication(existing.status)) return
  resubmitAppId.value = appId
  selectedDef.value = def
  uploadedFiles.value = {}
  isApplyOpen.value = true
}

function onConstraintFileChange(event: Event, constraintName: string) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) void handleFileUpload(constraintName, file)
}

function triggerFileInput(constraintName: string) {
  document.getElementById(`file-${constraintName}`)?.click()
}

async function handleFileUpload(constraintName: string, file: File) {
  if (uploadingConstraintName.value) return
  uploadingConstraintName.value = constraintName
  const fileExt = file.name.includes(".") ? "." + file.name.split(".").pop() : ""
  try {
    const fileHash = await sha256Hex(file)
    const contentType = file.type || "application/octet-stream"
    const res = await apiClient("/api/credentials/upload-url", {
      method: "POST",
      body: JSON.stringify({ cred_def_ulid: credentialDefinitionId(selectedDef.value), file_name: file.name, file_ext: fileExt, file_hash: fileHash, content_type: contentType, file_usage: constraintName }),
    })
    const uploadRes = await uploadWithTimeout(res.upload_url, { method: "PUT", headers: new Headers(res.signed_headers || {}), body: file })
    if (!uploadRes.ok) throw new Error("S3 upload failed")
    uploadedFiles.value = { ...uploadedFiles.value, [constraintName]: { name: file.name, url: res.file_key, ext: fileExt, hash: fileHash, size: file.size } }
  } catch {
    toast.error(t.value.credentialsPage.uploadFailed)
  } finally {
    uploadingConstraintName.value = ""
  }
}

async function handleSubmitApplication() {
  isSubmitting.value = true
  const evidenceFiles = Object.keys(uploadedFiles.value).map((k) => ({
    file_name: uploadedFiles.value[k].name,
    file_url: uploadedFiles.value[k].url,
    file_hash: uploadedFiles.value[k].hash,
    file_ext: uploadedFiles.value[k].ext,
    file_size: uploadedFiles.value[k].size,
    file_usage: k,
    file_type: selectedDef.value.file_constraints.find((c: any) => c.name === k)?.type || 1,
  }))
  try {
    if (resubmitAppId.value) {
      await apiClient("/api/credentials/update", { method: "PUT", body: JSON.stringify({ app_id: resubmitAppId.value, files: evidenceFiles }) })
    } else {
      await apiClient("/api/credentials/submit", { method: "POST", body: JSON.stringify({ cred_def_ulid: credentialDefinitionId(selectedDef.value), files: evidenceFiles }) })
    }
    isApplyOpen.value = false
    applicationPage.value = 1
    await fetchData()
  } catch {
    toast.error(t.value.credentialsPage.submitFailed)
  } finally {
    isSubmitting.value = false
  }
}

function statusIcon(status: string) {
  const s = statusEnumNameForStatus(CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, status).toUpperCase()
  switch (s) {
    case "PENDING":
    case "APPLICATION_STATUS_PENDING":
      return Clock
    case "APPROVED":
    case "APPLICATION_STATUS_APPROVED":
      return CheckCircle
    case "REJECTED":
    case "APPLICATION_STATUS_REJECTED":
      return XCircle
    case "NEEDS_RESUBMIT":
    case "RESUBMIT":
    case "REUPLOAD":
    case "APPLICATION_STATUS_RESUBMIT":
    case "APPLICATION_STATUS_REUPLOAD":
      return AlertCircle
    default:
      return FileText
  }
}

function applicationStatusPillClass(status: string) {
  const s = statusEnumNameForStatus(CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, status).toUpperCase()
  if (["APPROVED", "APPLICATION_STATUS_APPROVED"].includes(s)) return "border-emerald-200 bg-emerald-50 text-emerald-700"
  if (["REJECTED", "APPLICATION_STATUS_REJECTED"].includes(s)) return "border-red-200 bg-red-50 text-red-700"
  if (["NEEDS_RESUBMIT", "RESUBMIT", "REUPLOAD", "APPLICATION_STATUS_RESUBMIT", "APPLICATION_STATUS_REUPLOAD"].includes(s)) return "border-amber-200 bg-amber-50 text-amber-700"
  if (["PENDING", "APPLICATION_STATUS_PENDING"].includes(s)) return "border-blue-200 bg-blue-50 text-blue-700"
  return "border-slate-200 bg-slate-50 text-slate-600"
}

function canResubmit(status: string) {
  const s = statusEnumNameForStatus(CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, status).toUpperCase()
  return ["REUPLOAD", "RESUBMIT", "NEEDS_RESUBMIT", "APPLICATION_STATUS_REUPLOAD", "APPLICATION_STATUS_RESUBMIT"].includes(s)
}

function isRejectedStatus(status: string) {
  const s = statusEnumNameForStatus(CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, status).toUpperCase()
  return ["REJECTED", "APPLICATION_STATUS_REJECTED"].includes(s)
}

function isPendingReviewStatus(status: string) {
  const s = statusEnumNameForStatus(CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, status).toUpperCase()
  return ["PENDING", "APPLICATION_STATUS_PENDING"].includes(s)
}

function isApprovedStatus(status: string) {
  const s = statusEnumNameForStatus(CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, status).toUpperCase()
  return ["APPROVED", "APPLICATION_STATUS_APPROVED"].includes(s)
}

function canStartNewApplication(status: string) {
  const s = statusEnumNameForStatus(CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, status).toUpperCase()
  return !["PENDING", "APPLICATION_STATUS_PENDING", "APPROVED", "APPLICATION_STATUS_APPROVED", "REJECTED", "APPLICATION_STATUS_REJECTED"].includes(s)
}

function credentialDefinitionId(def: any) {
  return String(def?.cred_def_id || def?.cred_def_ulid || "").trim()
}

function applicationId(app: any) {
  return String(app?.app_id || app?.app_ulid || "").trim()
}

function applicationCredentialDefinitionId(app: any) {
  return String(app?.cred_def_id || app?.cred_def_ulid || "").trim()
}

function definitionForApplication(app: any) {
  const credDefId = applicationCredentialDefinitionId(app)
  return definitions.value.find((def) => credentialDefinitionId(def) === credDefId) || null
}

function applicationTitle(app: any) {
  return app?.credential_name || definitionForApplication(app)?.name || t.value.credentialsPage.applicationRecord
}

function applicationMeta(app: any) {
  const parts = [
    app?.credential_category,
    app?.created_at ? `${t.value.credentialsPage.submittedAt} ${formatBackendDateOnly(app.created_at)}` : "",
  ].filter(Boolean)
  return parts.join(" · ") || t.value.credentialsPage.application
}

function latestApplicationForDef(credDefId: string) {
  const normalizedCredDefId = String(credDefId || "").trim()
  const matches = applications.value.filter((app) => applicationCredentialDefinitionId(app) === normalizedCredDefId)
  return matches[0] || null
}

function applicationActionLabel(def: any) {
  const existing = latestApplicationForDef(credentialDefinitionId(def))
  if (!existing) return t.value.credentialsPage.applyNow
  if (isPendingReviewStatus(existing.status)) return t.value.credentialsPage.applicationPendingHint
  if (isApprovedStatus(existing.status)) return t.value.credentialsPage.applicationApprovedHint
  if (canResubmit(existing.status)) return t.value.credentialsPage.appStatusResubmit
  if (isRejectedStatus(existing.status)) return t.value.credentialsPage.appStatusRejected
  return t.value.credentialsPage.applyNow
}

function isApplicationActionDisabled(def: any) {
  const existing = latestApplicationForDef(credentialDefinitionId(def))
  return Boolean(existing && !canStartNewApplication(existing.status) && !canResubmit(existing.status))
}

function handleDefinitionAction(def: any) {
  const existing = latestApplicationForDef(credentialDefinitionId(def))
  if (existing && canResubmit(existing.status)) {
    handleApplyClick(def, applicationId(existing))
    return
  }
  handleApplyClick(def)
}

onMounted(fetchData)
</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <Award class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ t.credentialsPage.title }}</span>
      </header>

      <main class="px-5 py-8 md:px-8 lg:px-10">
        <div class="mb-6">
          <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.credentialsPage.title }}</h1>
          <p class="mt-2 text-muted-foreground">{{ t.credentialsPage.subtitle }}</p>
        </div>

    <div v-if="loading" class="flex items-center justify-center gap-2 rounded-[16px] bg-white py-16 text-muted-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <Loader2 class="h-5 w-5 animate-spin" />
      <span>{{ t.common.loading }}</span>
    </div>
    <div v-else class="space-y-4">
      <section>
        <div class="mb-4 flex items-center gap-3 rounded-[16px] bg-white px-4 py-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
          <div class="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <Award class="h-4 w-4" />
          </div>
          <h2 class="font-semibold text-card-foreground">{{ t.credentialsPage.availableQualifications }}</h2>
        </div>
        <div class="grid gap-4 md:grid-cols-3">
          <div v-for="def in definitions" :key="credentialDefinitionId(def)" class="group relative flex flex-col overflow-hidden rounded-[16px] bg-white text-card-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all hover:-translate-y-0.5 hover:bg-[#f4fbfc] hover:shadow-md hover:shadow-primary/10">
            <div class="absolute left-0 top-0 h-full w-1 bg-primary/45" />
            <div class="flex flex-col space-y-3 p-4">
              <div class="flex h-11 w-11 items-center justify-center rounded-lg bg-primary/10 text-primary transition-transform group-hover:scale-105">
                <Award class="h-5 w-5" />
              </div>
              <h3 class="text-xl font-semibold leading-tight tracking-tight">{{ def.name }}</h3>
              <span class="badge w-fit border-primary/20 bg-primary/10 text-primary">{{ def.category }}</span>
            </div>
            <div class="flex flex-1 flex-col p-4 pt-0">
              <p class="flex-1 text-sm leading-6 text-muted-foreground">{{ def.description }}</p>
              <div v-if="latestApplicationForDef(credentialDefinitionId(def))" class="mt-3">
                <span :class="['inline-flex w-fit items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-semibold', applicationStatusPillClass(latestApplicationForDef(credentialDefinitionId(def))?.status)]">
                  <component :is="statusIcon(latestApplicationForDef(credentialDefinitionId(def))?.status)" class="h-3.5 w-3.5" />
                  {{ statusLabel(t, CANDIDATE_APPLICATION_STATUS_LABELS, latestApplicationForDef(credentialDefinitionId(def))?.status, 'credentialsPage.appStatusUnknown') }}
                </span>
              </div>
              <button class="btn btn-primary mt-4 w-full cursor-pointer rounded-lg shadow-sm shadow-primary/20 disabled:cursor-not-allowed disabled:opacity-60" :disabled="isApplicationActionDisabled(def)" @click="handleDefinitionAction(def)">
                {{ applicationActionLabel(def) }}
              </button>
            </div>
          </div>
        </div>
      </section>

      <section>
        <div class="mb-4 flex items-center gap-3 rounded-[16px] bg-white px-4 py-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
          <div class="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <FileText class="h-4 w-4" />
          </div>
          <h2 class="font-semibold text-card-foreground">{{ t.credentialsPage.myApplications }}</h2>
        </div>
        <div v-if="applicationsLoading" class="flex items-center justify-center gap-2 rounded-[16px] bg-white py-14 text-muted-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
          <Loader2 class="h-5 w-5 animate-spin" />
          <span>{{ t.common.loading }}</span>
        </div>
        <div v-else-if="applications.length === 0" class="flex flex-col items-center justify-center rounded-[16px] bg-white px-4 py-14 text-center shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
          <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10">
            <FileText class="h-8 w-8 text-primary" />
          </div>
          <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.credentialsPage.noApplications }}</h3>
        </div>
        <div v-else class="overflow-hidden rounded-[16px] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
          <div class="space-y-3 p-3 md:space-y-2 md:p-0">
            <div v-for="app in applications" :key="applicationId(app) || applicationCredentialDefinitionId(app)" class="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-x-3 gap-y-3 rounded-xl border border-slate-100 bg-white px-3 py-4 shadow-sm shadow-slate-100/80 transition-colors hover:bg-primary/10 md:items-center md:rounded-none md:border-0 md:px-4 md:shadow-none md:gap-x-6 lg:grid-cols-[minmax(320px,2.4fr)_minmax(160px,1fr)_minmax(128px,auto)_minmax(104px,auto)] lg:gap-x-8">
              <div class="min-w-0 lg:col-span-1">
                <div class="break-words text-base font-semibold leading-6 text-foreground md:truncate md:font-medium" :title="applicationTitle(app)">{{ applicationTitle(app) }}</div>
                <div class="mt-1 break-words text-sm leading-5 text-muted-foreground md:truncate" :title="applicationMeta(app)">{{ applicationMeta(app) }}</div>
              </div>
              <span :class="['inline-flex w-fit min-w-0 items-center justify-center gap-1.5 justify-self-end rounded-full border px-3 py-1 text-xs font-semibold lg:min-w-[88px]', applicationStatusPillClass(app.status)]">
                <component :is="statusIcon(app.status)" class="h-3.5 w-3.5" />
                {{ statusLabel(t, CANDIDATE_APPLICATION_STATUS_LABELS, app.status, 'credentialsPage.appStatusUnknown') }}
              </span>
              <div class="col-span-2 min-w-0 rounded-lg bg-slate-50 px-3 py-2 text-sm leading-5 text-muted-foreground md:col-span-2 md:bg-transparent md:px-0 md:py-0 md:truncate lg:col-span-1" :title="app.audit_remark ? `${t.credentialsPage.auditRemark}: ${app.audit_remark}` : t.common.na">{{ app.audit_remark ? `${t.credentialsPage.auditRemark}: ${app.audit_remark}` : t.common.na }}</div>
              <button v-if="canResubmit(app.status)" class="btn btn-primary col-span-2 h-9 w-full cursor-pointer rounded-lg py-1 text-sm shadow-sm shadow-primary/20 md:col-span-1 md:w-auto md:justify-self-end" @click="handleApplyClick(definitionForApplication(app), applicationId(app))">{{ t.credentialsPage.appStatusResubmit }}</button>
              <span v-else class="col-span-2 justify-self-start whitespace-nowrap text-sm text-muted-foreground md:col-span-1 md:justify-self-end">{{ formatBackendDateOnly(app.created_at) || t.common.na }}</span>
            </div>
          </div>
          <AppPagination
            v-if="applicationTotal > 0"
            v-model:page="applicationPage"
            v-model:page-size="applicationPageSize"
            :total="applicationTotal"
            :total-pages="applicationTotalPages"
            :page-size-options="applicationPageSizeOptions"
            :locale="lang"
            @page-change="handleApplicationPageChange"
          />
        </div>
      </section>
    </div>

      </main>
    </div>

    <div v-if="isApplyOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div class="w-full max-w-md rounded-[16px] bg-white p-4 shadow-lg shadow-slate-900/20">
        <div class="flex items-start justify-between gap-4">
          <h2 class="text-lg font-semibold leading-none tracking-tight">{{ selectedDef?.name }}</h2>
          <button class="flex h-10 w-10 shrink-0 cursor-pointer items-center justify-center rounded-full border border-slate-200 bg-white/90 text-slate-500 transition hover:border-primary/25 hover:text-primary" @click="isApplyOpen = false">
            <X class="h-5 w-5" />
          </button>
        </div>
        <div class="space-y-4 py-4">
          <p class="text-sm text-muted-foreground">{{ t.credentialsPage.description }}: {{ selectedDef?.description }}</p>
          <div class="space-y-4 border-t border-border pt-4">
            <h4 class="text-sm font-semibold">{{ t.credentialsPage.uploadMaterials }}</h4>
            <div v-for="constraint in selectedDef?.file_constraints || []" :key="constraint.name" class="space-y-2 rounded-lg bg-muted p-3">
              <div class="flex justify-between">
                <span class="font-medium">{{ constraint.name }}</span>
                <span :class="['badge border-transparent', constraint.is_required ? 'bg-destructive text-destructive-foreground' : 'bg-secondary text-secondary-foreground']">{{ constraint.is_required ? t.credentialsPage.required : t.credentialsPage.optional }}</span>
              </div>
              <div class="mt-2 flex items-center gap-2">
                <button type="button" class="btn btn-outline cursor-pointer rounded-lg px-3 py-1.5 text-xs hover:border-primary/25 hover:bg-primary/10 hover:text-primary" :disabled="Boolean(uploadingConstraintName)" @click="triggerFileInput(constraint.name)">
                  <Loader2 v-if="uploadingConstraintName === constraint.name" class="h-4 w-4 animate-spin" />
                  {{ t.credentialsPage.chooseFile }}
                </button>
                <span class="max-w-[200px] truncate text-sm text-muted-foreground" :title="uploadedFiles[constraint.name] ? uploadedFiles[constraint.name].name : t.credentialsPage.noFileChosen">
                  {{ uploadedFiles[constraint.name] ? uploadedFiles[constraint.name].name : t.credentialsPage.noFileChosen }}
                </span>
                <input :id="`file-${constraint.name}`" type="file" class="hidden" @change="onConstraintFileChange($event, constraint.name)" />
              </div>
              <p v-if="uploadedFiles[constraint.name]" class="flex items-center gap-1 text-xs text-green-600"><CheckCircle class="h-3 w-3" /> {{ uploadedFiles[constraint.name].name }} uploaded</p>
            </div>
          </div>
        </div>
        <div class="flex justify-end gap-3">
          <button class="btn btn-outline cursor-pointer rounded-lg" @click="isApplyOpen = false">{{ t.common.cancel }}</button>
          <button class="btn btn-primary cursor-pointer rounded-lg shadow-sm shadow-primary/20 disabled:cursor-not-allowed" :disabled="isSubmitting || !(selectedDef?.file_constraints?.every((c: any) => !c.is_required || uploadedFiles[c.name]) && selectedDef?.file_constraints?.length > 0)" @click="handleSubmitApplication">
            <Loader2 v-if="isSubmitting" class="h-4 w-4 animate-spin" />
            {{ isSubmitting ? t.credentialsPage.submitting : t.credentialsPage.submitApplication }}
          </button>
        </div>
      </div>
    </div>
  </AppShell>
</template>
