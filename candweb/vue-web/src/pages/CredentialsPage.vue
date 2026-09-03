<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { useRoute } from "vue-router"
import { AlertCircle, Award, CheckCircle, Clock, Download, Eye, FileText, Loader2, X, XCircle } from "lucide-vue-next"
import { getFileConstraintInfo } from "../lib/fileConstraints"
import { CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, CANDIDATE_APPLICATION_STATUS_LABELS, statusEnumNameForStatus, statusLabel } from "@/lib/status-labels"
import AppPagination from "@/components/AppPagination.vue"
import AppShell from "@/components/AppShell.vue"
import CredentialAttachmentList from "@/components/CredentialAttachmentList.vue"
import PageFeedback from "@/components/PageFeedback.vue"
import { apiClient } from "@/lib/apiClient"
import { useBodyScrollLock } from "@/lib/bodyScrollLock"
import { candidateVisibleCredentialDefinitions } from "@/lib/credentialDefinitions"
import { useDialogAccessibility } from "@/lib/dialogAccessibility"
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
const applicationTotalLabel = ref("")
const applicationHasMore = ref(false)
const applicationNextCursor = ref("")
const applicationPrevCursor = ref("")
const lastApplicationPage = ref(1)
const lastApplicationPageSize = ref(applicationPageSize.value)
const loading = ref(true)
const loadError = ref(false)
const applicationsLoading = ref(false)
const selectedDef = ref<any>(null)
const resubmitAppId = ref("")
const isApplyOpen = ref(false)
const applicationDetailDialogOpen = ref(false)
const applicationDetailDialogRef = ref<HTMLElement | null>(null)
const selectedApplicationDetail = ref<any>(null)
const applicationDetailLoading = ref(false)
const applicationDetailError = ref(false)
let applicationDetailRequestID = 0
useBodyScrollLock(() => isApplyOpen.value || applicationDetailDialogOpen.value)
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

async function fetchApplications(options: { showLoading?: boolean; suppressErrorToast?: boolean } = {}) {
  if (options.showLoading) applicationsLoading.value = true
  try {
    const params = new URLSearchParams({
      page_size: String(applicationPageSize.value),
    })
    
    let cursor = ""
    if (applicationPage.value > lastApplicationPage.value) {
      cursor = applicationNextCursor.value
    } else if (applicationPage.value < lastApplicationPage.value) {
      cursor = applicationPrevCursor.value
    }
    
    if (cursor) params.set("cursor", cursor)
    
    const appsRes = await apiClient(`/api/credentials/applications?${params.toString()}`, {
      suppressErrorToast: options.suppressErrorToast,
    })
    const nextApplications = appsRes?.applications || []
    applications.value = nextApplications
    applicationTotal.value = totalFrom(appsRes, nextApplications)
    applicationTotalLabel.value = String(appsRes?.total_label || applicationTotal.value)
    applicationTotalPages.value = totalPagesFrom(appsRes, applicationTotal.value, applicationPageSize.value)
    applicationHasMore.value = Boolean(appsRes?.has_more)
    applicationNextCursor.value = String(appsRes?.next_cursor || "")
    applicationPrevCursor.value = String(appsRes?.prev_cursor || "")
    lastApplicationPage.value = applicationPage.value
  } finally {
    if (options.showLoading) applicationsLoading.value = false
  }
}

async function fetchData(openSingleDefinition = true) {
  loading.value = true
  loadError.value = false
  try {
    const qualIds = String(route.query.qual_ulids || route.query.qual_ids || "").trim()
    const definitionsEndpoint = qualIds ? `/api/credentials/definitions?qual_ulids=${encodeURIComponent(qualIds)}` : "/api/credentials/definitions"
    const defsRes = await apiClient(definitionsEndpoint, { suppressErrorToast: true })
    definitions.value = candidateVisibleCredentialDefinitions(defsRes?.definitions)
    await fetchApplications({ suppressErrorToast: true })
    if (openSingleDefinition && qualIds && definitions.value.length === 1 && !isApplyOpen.value) {
      handleDefinitionAction(definitions.value[0])
    }
  } catch (error) {
    console.error(error)
    loadError.value = true
  } finally {
    loading.value = false
  }
}

async function handleApplicationPageChange() {
  if (applicationPageSize.value !== lastApplicationPageSize.value) {
    lastApplicationPageSize.value = applicationPageSize.value
    applicationPage.value = 1
    lastApplicationPage.value = 1
    applicationNextCursor.value = ""
    applicationPrevCursor.value = ""
    applicationHasMore.value = false
  }
  await fetchApplications({ showLoading: true })
  window.scrollTo({ top: 0, behavior: "smooth" })
}

function handleApplyClick(def: any, appId = "") {
  if (!def) return
  const existing = latestApplicationForDef(credentialDefinitionId(def))
  if (!existing || (!isPendingUploadStatus(existing.status) && !canResubmit(existing.status))) return
  resubmitAppId.value = canResubmit(existing.status) ? (appId || applicationId(existing)) : ""
  selectedDef.value = def
  uploadedFiles.value = {}
  isApplyOpen.value = true
}

function onConstraintFileChange(event: Event, constraint: any) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) void handleFileUpload(constraint, file)
}

function uploadSuccessText(fileName: string) {
  return t.value.credentialsPage.uploadSuccess.replace("{{fileName}}", fileName)
}

function getFormatHint(constraint: any) {
  const info = getFileConstraintInfo(constraint.type)
  const extText = info.extLabel === "Any" ? t.value.credentialsPage.anyFileType : info.extLabel
  return t.value.credentialsPage.supportedFormats.replace("{{exts}}", extText).replace("{{limit}}", info.maxLabel)
}

function constraintDisplayName(constraint: any) {
  return String(constraint?.display_name || constraint?.name || "")
}

function triggerFileInput(constraintName: string) {
  document.getElementById(`file-${constraintName}`)?.click()
}

async function handleFileUpload(constraint: any, file: File) {
  if (uploadingConstraintName.value) return
  const constraintName = constraint.name
  
  const info = getFileConstraintInfo(constraint.type)
  const fileExt = file.name.includes(".") ? "." + file.name.split(".").pop()?.toLowerCase() : ""
  
  if (info.maxSize && file.size > info.maxSize) {
    toast.error(t.value.credentialsPage.fileSizeLimitError.replace("{{limit}}", info.maxLabel))
    return
  }
  
  if (info.exts.length > 0 && !info.exts.includes(fileExt || "")) {
    toast.error(t.value.credentialsPage.fileTypeError.replace("{{exts}}", info.extLabel))
    return
  }

  uploadingConstraintName.value = constraintName
  try {
    const fileHash = await sha256Hex(file)
    const contentType = file.type || "application/octet-stream"
    const res = await apiClient("/api/credentials/upload-url", {
      method: "POST",
      body: JSON.stringify({ cred_def_ulid: credentialDefinitionId(selectedDef.value), file_name: file.name, file_ext: fileExt, file_hash: fileHash, content_type: contentType, file_usage: constraintName }),
    })
    const uploadRes = await uploadWithTimeout(res.upload_url, { method: "PUT", headers: new Headers(res.signed_headers || {}), body: file })
    if (!uploadRes.ok) throw new Error(`S3 upload failed: ${uploadRes.status} ${uploadRes.statusText}`)
    uploadedFiles.value = { ...uploadedFiles.value, [constraintName]: { name: file.name, url: res.file_key, ext: fileExt, hash: fileHash, size: file.size } }
  } catch (err: any) {
    toast.error(`${t.value.credentialsPage.uploadFailed}: ${err?.message || err}`)
  } finally {
    uploadingConstraintName.value = ""
  }
}

async function handleSubmitApplication() {
  const constraints = selectedDef.value?.file_constraints
  if (!Array.isArray(constraints)) {
    toast.error(t.value.credentialsPage.materialRequirementsUnavailable)
    return
  }

  if (Object.keys(uploadedFiles.value).length === 0) {
    toast.error(t.value.credentialsPage.requiredMaterialsMissing)
    return
  }

  const hasMissingRequiredMaterial = constraints.some((constraint: any) => constraint.is_required && !uploadedFiles.value[constraint.name])
  if (hasMissingRequiredMaterial) {
    toast.error(t.value.credentialsPage.requiredMaterialsMissing)
    return
  }

  isSubmitting.value = true
  const evidenceFiles = Object.keys(uploadedFiles.value).map((k) => ({
    file_name: uploadedFiles.value[k].name,
    file_url: uploadedFiles.value[k].url,
    file_hash: uploadedFiles.value[k].hash,
    file_ext: uploadedFiles.value[k].ext,
    file_size: uploadedFiles.value[k].size,
    file_usage: k,
    file_type: constraints.find((c: any) => c.name === k)?.type || 1,
  }))
  try {
    if (resubmitAppId.value) {
      await apiClient("/api/credentials/update", { method: "PUT", body: JSON.stringify({ app_id: resubmitAppId.value, files: evidenceFiles }) })
    } else {
      await apiClient("/api/credentials/submit", { method: "POST", body: JSON.stringify({ cred_def_ulid: credentialDefinitionId(selectedDef.value), files: evidenceFiles }) })
    }
    toast.success(t.value.credentialsPage.submitSuccess)
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
    case "PENDING_UPLOAD":
    case "APPLICATION_STATUS_PENDING_UPLOAD":
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
  if (["PENDING_UPLOAD", "APPLICATION_STATUS_PENDING_UPLOAD"].includes(s)) return "border-sky-200 bg-sky-50 text-sky-700"
  if (["PENDING", "APPLICATION_STATUS_PENDING"].includes(s)) return "border-blue-200 bg-blue-50 text-blue-700"
  return "border-slate-200 bg-slate-50 text-slate-600"
}

function canResubmit(status: string) {
  const s = statusEnumNameForStatus(CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, status).toUpperCase()
  return ["REUPLOAD", "RESUBMIT", "NEEDS_RESUBMIT", "APPLICATION_STATUS_REUPLOAD", "APPLICATION_STATUS_RESUBMIT"].includes(s)
}

function isPendingUploadStatus(status: string) {
  const s = statusEnumNameForStatus(CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, status).toUpperCase()
  return ["PENDING_UPLOAD", "APPLICATION_STATUS_PENDING_UPLOAD"].includes(s)
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

function applicationFileURL(file: any) {
  const value = String(file?.view_url || "").trim()
  if (!value) return ""
  try {
    const parsed = new URL(value)
    return parsed.protocol === "https:" || parsed.protocol === "http:" ? parsed.toString() : ""
  } catch {
    return ""
  }
}

function formatFileSize(value: unknown) {
  const bytes = Number(value)
  if (!Number.isFinite(bytes) || bytes <= 0) return ""
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

const selectedApplicationFiles = computed(() =>
  (Array.isArray(selectedApplicationDetail.value?.files) ? selectedApplicationDetail.value.files : [])
    .filter((file: any) => applicationFileURL(file)),
)

async function loadApplicationDetail(app: any) {
  const appID = applicationId(app)
  if (!appID) return
  const requestID = ++applicationDetailRequestID
  applicationDetailLoading.value = true
  applicationDetailError.value = false
  try {
    const detail = await apiClient(`/api/credentials/applications/${encodeURIComponent(appID)}`, {
      suppressErrorToast: true,
    })
    if (requestID !== applicationDetailRequestID || !applicationDetailDialogOpen.value) return
    selectedApplicationDetail.value = {
      ...app,
      ...detail,
      audit_remark: String(detail?.audit_remark || "").trim() || app?.audit_remark || "",
    }
  } catch {
    if (requestID === applicationDetailRequestID && applicationDetailDialogOpen.value) {
      applicationDetailError.value = true
    }
  } finally {
    if (requestID === applicationDetailRequestID) applicationDetailLoading.value = false
  }
}

function openApplicationDetail(app: any) {
  selectedApplicationDetail.value = { ...app, files: [] }
  applicationDetailDialogOpen.value = true
  void loadApplicationDetail(app)
}

function retryApplicationDetail() {
  if (selectedApplicationDetail.value) void loadApplicationDetail(selectedApplicationDetail.value)
}

function closeApplicationDetail() {
  applicationDetailRequestID += 1
  applicationDetailDialogOpen.value = false
  applicationDetailLoading.value = false
}

useDialogAccessibility(
  () => applicationDetailDialogOpen.value,
  applicationDetailDialogRef,
  closeApplicationDetail,
)

function latestApplicationForDef(credDefId: string) {
  const normalizedCredDefId = String(credDefId || "").trim()
  const application = applications.value.find((item) => applicationCredentialDefinitionId(item) === normalizedCredDefId)
  if (application) return application
  const definition = definitions.value.find((def) => credentialDefinitionId(def) === normalizedCredDefId)
  // Compatibility for responses produced before applications became the source of truth.
  return definition?.latest_application || null
}

function applicationActionLabel(def: any) {
  const existing = latestApplicationForDef(credentialDefinitionId(def))
  if (!existing) return t.value.credentialsPage.applyDuringCheckout
  if (isPendingUploadStatus(existing.status)) return t.value.credentialsPage.uploadMaterials
  if (isPendingReviewStatus(existing.status)) return t.value.credentialsPage.applicationPendingHint
  if (isApprovedStatus(existing.status)) return t.value.credentialsPage.applicationApprovedHint
  if (canResubmit(existing.status)) return t.value.credentialsPage.resubmitAction
  if (isRejectedStatus(existing.status)) return t.value.credentialsPage.appStatusRejected
  return t.value.credentialsPage.applyNow
}

function isApplicationActionDisabled(def: any) {
  const existing = latestApplicationForDef(credentialDefinitionId(def))
  if (!existing) return true
  return !isPendingUploadStatus(existing.status) && !canResubmit(existing.status)
}

function handleDefinitionAction(def: any) {
  const existing = latestApplicationForDef(credentialDefinitionId(def))
  if (!existing || (!isPendingUploadStatus(existing.status) && !canResubmit(existing.status))) return
  handleApplyClick(def, canResubmit(existing.status) ? applicationId(existing) : "")
}

onMounted(fetchData)

watch(lang, () => {
  void fetchData(false)
})

watch(
  () => String(route.query.qual_ulids || route.query.qual_ids || "").trim(),
  (qualIds, previousQualIds) => {
    if (qualIds === previousQualIds) return
    void fetchData()
  },
)
</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <Award class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ t.credentialsPage.title }}</span>
      </header>

      <main class="px-5 py-8 md:px-8 lg:px-10">
        <div class="credentials-page-intro mb-6">
          <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.credentialsPage.title }}</h1>
          <p class="mt-2 text-muted-foreground">{{ t.credentialsPage.subtitle }}</p>
        </div>

    <PageFeedback v-if="loading" kind="loading" :loading-label="t.common.loading" />
    <PageFeedback
      v-else-if="loadError"
      kind="error"
      :title="t.credentialsPage.loadFailed"
      :description="t.credentialsPage.loadFailedDesc"
      :action-label="t.credentialsPage.retry"
      @action="fetchData()"
    />
    <div v-else class="credentials-page-content space-y-4">
      <section>
        <div class="credential-definitions-grid grid gap-4 md:grid-cols-3">
          <div v-for="def in definitions" :key="credentialDefinitionId(def)" class="credential-definition-card group relative flex flex-col overflow-hidden rounded-[16px] bg-white text-card-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all hover:-translate-y-0.5 hover:bg-[#f4fbfc] hover:shadow-md hover:shadow-primary/10">
            <div class="absolute left-0 top-0 h-full w-1 bg-primary/45" />
            <div class="credential-definition-heading flex flex-col space-y-3 p-4">
              <div class="credential-definition-icon flex h-11 w-11 items-center justify-center rounded-lg bg-primary/10 text-primary transition-transform group-hover:scale-105">
                <Award class="h-5 w-5" />
              </div>
              <h3 class="text-xl font-semibold leading-tight tracking-tight">{{ def.name }}</h3>
              <span class="badge w-fit border-primary/20 bg-primary/10 text-primary">{{ def.category }}</span>
            </div>
            <div class="credential-definition-body flex flex-1 flex-col p-4 pt-0">
              <p class="flex-1 text-sm leading-6 text-muted-foreground">{{ def.description }}</p>
              <div v-if="latestApplicationForDef(credentialDefinitionId(def))" class="credential-definition-status mt-3">
                <span :class="['inline-flex w-fit items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-semibold', applicationStatusPillClass(latestApplicationForDef(credentialDefinitionId(def))?.status)]">
                  <component :is="statusIcon(latestApplicationForDef(credentialDefinitionId(def))?.status)" class="h-3.5 w-3.5" />
                  {{ statusLabel(t, CANDIDATE_APPLICATION_STATUS_LABELS, latestApplicationForDef(credentialDefinitionId(def))?.status, 'credentialsPage.appStatusUnknown') }}
                </span>
              </div>
              <button class="credential-definition-action btn btn-primary mt-4 w-full cursor-pointer rounded-lg shadow-sm shadow-primary/20 disabled:cursor-not-allowed disabled:opacity-60" :disabled="isApplicationActionDisabled(def)" @click="handleDefinitionAction(def)">
                {{ applicationActionLabel(def) }}
              </button>
            </div>
          </div>
        </div>
      </section>

      <section>
        <div class="credentials-applications-header mb-4 flex items-center gap-3 rounded-[16px] bg-white px-4 py-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
          <div class="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <FileText class="h-4 w-4" />
          </div>
          <h2 class="font-semibold text-card-foreground">{{ t.credentialsPage.myApplications }}</h2>
        </div>
        <div v-if="applicationsLoading" class="credentials-applications-state flex items-center justify-center gap-2 rounded-[16px] bg-white py-14 text-muted-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
          <Loader2 class="h-5 w-5 animate-spin text-primary" />
          <span>{{ t.common.loading }}</span>
        </div>
        <div v-else-if="applications.length === 0" class="credentials-applications-state flex flex-col items-center justify-center rounded-[16px] bg-white px-4 py-14 text-center shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
          <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10">
            <FileText class="h-8 w-8 text-primary" />
          </div>
          <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.credentialsPage.noApplications }}</h3>
        </div>
        <div v-else class="overflow-hidden rounded-[16px] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
          <div class="space-y-3 p-3 md:space-y-2 md:p-0">
            <div v-for="app in applications" :key="applicationId(app) || applicationCredentialDefinitionId(app)" class="application-row grid grid-cols-[minmax(0,1fr)_auto] items-start gap-x-3 gap-y-3 rounded-xl border border-slate-100 bg-white px-3 py-4 shadow-sm shadow-slate-100/80 transition-colors hover:bg-primary/10 md:items-center md:rounded-none md:border-0 md:px-4 md:shadow-none md:gap-x-6 lg:grid-cols-[minmax(280px,2.4fr)_minmax(120px,1fr)_104px_auto] lg:gap-x-6">
              <div class="min-w-0 lg:col-span-1">
                <div class="break-words text-base font-semibold leading-6 text-foreground md:truncate md:font-medium" :title="applicationTitle(app)">{{ applicationTitle(app) }}</div>
                <div class="mt-1 break-words text-sm leading-5 text-muted-foreground md:truncate" :title="applicationMeta(app)">{{ applicationMeta(app) }}</div>
              </div>
              <span :class="['inline-flex w-fit min-w-0 items-center justify-center gap-1.5 justify-self-end rounded-full border px-3 py-1 text-xs font-semibold lg:min-w-[88px]', applicationStatusPillClass(app.status)]">
                <component :is="statusIcon(app.status)" class="h-3.5 w-3.5" />
                {{ statusLabel(t, CANDIDATE_APPLICATION_STATUS_LABELS, app.status, 'credentialsPage.appStatusUnknown') }}
              </span>
              <span class="col-span-2 justify-self-start whitespace-nowrap text-sm text-muted-foreground md:col-span-1 lg:col-start-3">{{ formatBackendDateOnly(app.created_at) || t.common.na }}</span>
              <div class="col-span-2 flex w-full flex-wrap items-center justify-end gap-2 md:col-span-1 md:w-auto md:justify-self-end lg:col-start-4">
                <button v-if="isPendingUploadStatus(app.status) || canResubmit(app.status)" class="btn btn-primary h-9 cursor-pointer whitespace-nowrap rounded-lg py-1 text-sm shadow-sm shadow-primary/20" @click="handleApplyClick(definitionForApplication(app), canResubmit(app.status) ? applicationId(app) : '')">{{ isPendingUploadStatus(app.status) ? t.credentialsPage.uploadMaterials : t.credentialsPage.resubmitAction }}</button>
                <button
                  v-if="!isPendingUploadStatus(app.status)"
                  type="button"
                  class="application-details-btn btn btn-outline h-9 cursor-pointer whitespace-nowrap rounded-lg px-3 py-1 text-sm text-primary hover:border-primary/30 hover:bg-primary/10 disabled:cursor-not-allowed disabled:opacity-50"
                  data-testid="application-view-details"
                  :disabled="!applicationId(app)"
                  @click="openApplicationDetail(app)"
                >
                  <Eye class="h-4 w-4" aria-hidden="true" />
                  {{ t.credentialsPage.viewDetails }}
                </button>
              </div>
            </div>
          </div>
          <AppPagination
            v-if="applicationTotal > 0"
            v-model:page="applicationPage"
            v-model:page-size="applicationPageSize"
            :total="applicationTotal"
            :total-pages="applicationTotalPages"
            :total-label="applicationTotalLabel"
            :page-size-options="applicationPageSizeOptions"
            cursor-mode
            :has-more="applicationHasMore"
            @page-change="handleApplicationPageChange"
          />
        </div>
      </section>
    </div>

      </main>
    </div>

    <div
      v-if="applicationDetailDialogOpen"
      class="app-safe-area-overlay fixed inset-0 z-50 flex items-center justify-center bg-slate-950/55 p-4 backdrop-blur-[2px]"
      @click.self="closeApplicationDetail"
    >
      <div
        ref="applicationDetailDialogRef"
        class="app-dialog-viewport-compact flex max-h-[86vh] w-full max-w-2xl flex-col overflow-hidden rounded-[16px] bg-white shadow-[0_24px_70px_rgba(15,23,42,0.28)]"
        role="dialog"
        aria-modal="true"
        aria-labelledby="application-detail-dialog-title"
        tabindex="-1"
        data-testid="application-detail-dialog"
      >
        <div class="flex items-start justify-between gap-4 border-b border-slate-100 px-5 py-5 sm:px-6">
          <div class="min-w-0">
            <h2 id="application-detail-dialog-title" class="text-xl font-bold leading-snug text-slate-950">
              {{ t.credentialsPage.applicationDetails }}
            </h2>
            <p class="mt-1 break-words text-sm text-slate-500">{{ applicationTitle(selectedApplicationDetail) }}</p>
          </div>
          <button
            type="button"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 transition hover:border-primary/25 hover:text-primary"
            :aria-label="t.common.close"
            :title="t.common.close"
            @click="closeApplicationDetail"
          >
            <X class="h-5 w-5" />
          </button>
        </div>
        <div class="min-h-0 overflow-y-auto px-5 py-5 sm:px-6">
          <div v-if="applicationDetailLoading" class="flex min-h-52 items-center justify-center gap-2 text-sm text-slate-500" data-testid="application-detail-loading">
            <Loader2 class="h-5 w-5 animate-spin text-primary" />
            {{ t.common.loading }}
          </div>
          <div v-else-if="applicationDetailError" class="flex min-h-52 flex-col items-center justify-center gap-4 text-center" data-testid="application-detail-error">
            <AlertCircle class="h-9 w-9 text-red-500" />
            <p class="text-sm text-slate-600">{{ t.credentialsPage.detailsLoadFailed }}</p>
            <button type="button" class="btn btn-outline" @click="retryApplicationDetail">{{ t.credentialsPage.retry }}</button>
          </div>
          <div v-else class="space-y-6">
            <div class="grid gap-3 sm:grid-cols-2">
              <div class="rounded-lg bg-slate-50 px-4 py-3">
                <div class="text-xs font-medium text-slate-500">{{ t.credentialsPage.applicationStatus }}</div>
                <div class="mt-1 text-sm font-semibold text-slate-900">
                  {{ statusLabel(t, CANDIDATE_APPLICATION_STATUS_LABELS, selectedApplicationDetail?.status, 'credentialsPage.appStatusUnknown') }}
                </div>
              </div>
              <div class="rounded-lg bg-slate-50 px-4 py-3">
                <div class="text-xs font-medium text-slate-500">{{ t.credentialsPage.applicationSubmittedAt }}</div>
                <div class="mt-1 text-sm font-semibold text-slate-900">{{ formatBackendDateOnly(selectedApplicationDetail?.created_at) || t.common.na }}</div>
              </div>
            </div>

            <section>
              <h3 class="text-sm font-semibold text-slate-950">{{ t.credentialsPage.auditRemark }}</h3>
              <p class="mt-2 whitespace-pre-wrap break-words rounded-lg border border-slate-200 bg-slate-50 px-4 py-3 text-sm leading-7 text-slate-800" data-testid="application-audit-remark-content">{{ String(selectedApplicationDetail?.audit_remark || '').trim() || t.credentialsPage.noAuditRemark }}</p>
            </section>

            <section>
              <h3 class="text-sm font-semibold text-slate-950">{{ t.credentialsPage.uploadedFiles }}</h3>
              <div v-if="selectedApplicationFiles.length" class="mt-2 divide-y divide-slate-100 overflow-hidden rounded-lg border border-slate-200">
                <div v-for="(file, index) in selectedApplicationFiles" :key="file.file_hash || `${file.file_name}-${index}`" class="flex flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
                  <div class="flex min-w-0 items-start gap-3">
                    <FileText class="mt-0.5 h-5 w-5 shrink-0 text-primary" />
                    <div class="min-w-0">
                      <div class="break-words text-sm font-semibold text-slate-900">{{ file.file_name || t.credentialsPage.uploadedFile }}</div>
                      <div class="mt-1 flex flex-wrap gap-x-2 gap-y-1 text-xs text-slate-500">
                        <span v-if="file.file_usage">{{ file.file_usage }}</span>
                        <span v-if="formatFileSize(file.file_size)">{{ formatFileSize(file.file_size) }}</span>
                      </div>
                    </div>
                  </div>
                  <div class="flex shrink-0 flex-wrap gap-2">
                    <a class="btn btn-outline h-9 px-3 text-xs" :href="applicationFileURL(file)" target="_blank" rel="noopener noreferrer">
                      <Eye class="h-4 w-4" /> {{ t.credentialsPage.previewFile }}
                    </a>
                    <a class="btn btn-outline h-9 px-3 text-xs" :href="applicationFileURL(file)" :download="file.file_name || true" target="_blank" rel="noopener noreferrer">
                      <Download class="h-4 w-4" /> {{ t.credentialsPage.downloadFile }}
                    </a>
                  </div>
                </div>
              </div>
              <p v-else class="mt-2 rounded-lg border border-dashed border-slate-200 px-4 py-6 text-center text-sm text-slate-500">{{ t.credentialsPage.noUploadedFiles }}</p>
            </section>
          </div>
        </div>
        <div class="flex justify-end border-t border-slate-100 px-5 py-4 sm:px-6">
          <button type="button" class="btn btn-primary min-w-24 justify-center" data-testid="application-detail-close" @click="closeApplicationDetail">
            {{ t.common.close }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="isApplyOpen" class="credentials-apply-backdrop app-safe-area-overlay fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div class="credentials-apply-dialog w-full max-w-md rounded-[16px] bg-white p-4 shadow-lg shadow-slate-900/20">
        <div class="credentials-apply-header flex items-start justify-between gap-4">
          <h2 class="text-lg font-semibold leading-none tracking-tight">{{ selectedDef?.name }}</h2>
          <button class="credentials-apply-close flex h-10 w-10 shrink-0 cursor-pointer items-center justify-center rounded-full border border-slate-200 bg-white/90 text-slate-500 transition hover:border-primary/25 hover:text-primary" @click="isApplyOpen = false">
            <X class="h-5 w-5" />
          </button>
        </div>
        <div class="credentials-apply-body space-y-4 py-4">
          <p class="text-sm text-muted-foreground">{{ t.credentialsPage.description }}: {{ selectedDef?.description }}</p>
          <CredentialAttachmentList :attachments="selectedDef?.attachments" class="border-t border-border pt-4" />
          <div class="space-y-4 border-t border-border pt-4">
            <h4 class="text-sm font-semibold">{{ t.credentialsPage.uploadMaterials }}</h4>
            <div v-for="constraint in selectedDef?.file_constraints || []" :key="constraint.name" class="space-y-2 rounded-lg bg-muted p-3">
              <div class="flex items-center gap-1">
                <span v-if="constraint.is_required" class="text-sm font-bold text-destructive">*</span>
                <span class="font-medium">{{ constraintDisplayName(constraint) }}</span>
              </div>
              <div class="mt-2 flex items-center gap-2">
                <button type="button" class="btn btn-outline cursor-pointer rounded-lg px-3 py-1.5 text-xs hover:border-primary/25 hover:bg-primary/10 hover:text-primary" :disabled="Boolean(uploadingConstraintName)" @click="triggerFileInput(constraint.name)">
                  <Loader2 v-if="uploadingConstraintName === constraint.name" class="h-4 w-4 animate-spin" />
                  {{ t.credentialsPage.chooseFile }}
                </button>
                <span class="max-w-[200px] truncate text-sm text-muted-foreground" :title="uploadedFiles[constraint.name] ? uploadedFiles[constraint.name].name : t.credentialsPage.noFileChosen">
                  {{ uploadedFiles[constraint.name] ? uploadedFiles[constraint.name].name : t.credentialsPage.noFileChosen }}
                </span>
                <input :id="`file-${constraint.name}`" type="file" class="hidden" :accept="getFileConstraintInfo(constraint.type).acceptStr" @change="onConstraintFileChange($event, constraint)" />
              </div>
              <p class="text-xs text-muted-foreground">
                {{ getFormatHint(constraint) }}
              </p>
              <p v-if="uploadedFiles[constraint.name]" class="flex items-center gap-1 text-xs text-green-600"><CheckCircle class="h-3 w-3" /> {{ uploadSuccessText(uploadedFiles[constraint.name].name) }}</p>
            </div>
          </div>
        </div>
        <div class="credentials-apply-actions flex justify-end gap-3">
          <button class="btn btn-outline cursor-pointer rounded-lg" @click="isApplyOpen = false">{{ t.common.cancel }}</button>
          <button class="btn btn-primary cursor-pointer rounded-lg shadow-sm shadow-primary/20 disabled:cursor-not-allowed" :disabled="isSubmitting || Boolean(uploadingConstraintName)" @click="handleSubmitApplication">
            <Loader2 v-if="isSubmitting" class="h-4 w-4 animate-spin" />
            {{ isSubmitting ? t.credentialsPage.submitting : t.credentialsPage.submitApplication }}
          </button>
        </div>
      </div>
    </div>
  </AppShell>
</template>

<style scoped>
@media (max-width: 767px) {
  .credentials-page-intro {
    margin-bottom: 16px;
  }

  .credentials-page-content > :not(:last-child) {
    margin-block-end: 12px;
  }

  .credentials-applications-header {
    margin-bottom: 12px;
    padding: 12px;
  }

  .credential-definitions-grid {
    gap: 12px;
  }

  .credential-definition-heading {
    padding: 12px;
  }

  .credential-definition-heading > :not(:last-child) {
    margin-block-end: 8px;
  }

  .credential-definition-icon {
    width: 40px;
    height: 40px;
  }

  .credential-definition-body {
    padding: 0 12px 12px;
  }

  .credential-definition-status {
    margin-top: 10px;
  }

  .credential-definition-action {
    margin-top: 12px;
  }

  .credentials-applications-state {
    padding-block: 32px;
  }

  .credentials-apply-backdrop {
    padding-top: max(12px, var(--app-safe-area-top));
    padding-right: max(12px, var(--app-safe-area-right));
    padding-bottom: max(12px, var(--app-safe-area-bottom));
    padding-left: max(12px, var(--app-safe-area-left));
  }

  .credentials-apply-dialog {
    display: flex;
    max-height: calc(var(--app-viewport-height) - var(--app-safe-area-top) - var(--app-safe-area-bottom) - 24px);
    flex-direction: column;
    overflow: hidden;
    border-radius: 12px;
  }

  .credentials-apply-header,
  .credentials-apply-actions {
    flex: 0 0 auto;
  }

  .credentials-apply-close {
    width: 44px;
    height: 44px;
  }

  .credentials-apply-body {
    min-height: 0;
    overflow-y: auto;
    overscroll-behavior: contain;
    padding-right: 4px;
  }

  .credentials-apply-dialog .btn {
    min-height: 42px;
  }

  .credentials-apply-actions {
    border-top: 1px solid var(--border);
    padding-top: 12px;
  }

  .credentials-apply-actions > button {
    min-height: 44px;
  }
}
</style>
