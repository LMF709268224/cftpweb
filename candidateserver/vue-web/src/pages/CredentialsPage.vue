<script setup lang="ts">
import { onMounted, ref } from "vue"
import { useRoute } from "vue-router"
import { AlertCircle, Award, CheckCircle, Clock, FileText, Loader2, XCircle } from "lucide-vue-next"
import { CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, CANDIDATE_APPLICATION_STATUS_LABELS, statusBadgeClassForStatus, statusEnumNameForStatus, statusLabel } from "@/lib/status-labels"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { formatBackendDate } from "@/lib/utils"
import { useTranslation } from "@/lib/language"

const { t } = useTranslation()
const route = useRoute()
const definitions = ref<any[]>([])
const applications = ref<any[]>([])
const loading = ref(true)
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

async function fetchData() {
  loading.value = true
  try {
    const qualIds = String(route.query.qual_ids || "").trim()
    const definitionsEndpoint = qualIds ? `/api/credentials/definitions?qual_ids=${encodeURIComponent(qualIds)}` : "/api/credentials/definitions"
    const [defsRes, appsRes] = await Promise.all([apiClient(definitionsEndpoint), apiClient("/api/credentials/applications")])
    definitions.value = defsRes?.definitions || []
    applications.value = appsRes?.applications || []
    if (qualIds && definitions.value.length === 1 && !isApplyOpen.value) {
      handleApplyClick(definitions.value[0])
    }
  } finally {
    loading.value = false
  }
}

function handleApplyClick(def: any, appId = "") {
  if (!def) return
  const existing = latestApplicationForDef(def.cred_def_id)
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
      body: JSON.stringify({ cred_def_id: selectedDef.value.cred_def_id, file_name: file.name, file_ext: fileExt, file_hash: fileHash, content_type: contentType, file_usage: constraintName }),
    })
    const uploadRes = await uploadWithTimeout(res.upload_url, { method: "PUT", headers: new Headers(res.signed_headers || {}), body: file })
    if (!uploadRes.ok) throw new Error("S3 upload failed")
    uploadedFiles.value = { ...uploadedFiles.value, [constraintName]: { name: file.name, url: res.file_key, ext: fileExt, hash: fileHash, size: file.size } }
  } catch {
    alert(t.value.credentialsPage.uploadFailed)
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
      await apiClient("/api/credentials/submit", { method: "POST", body: JSON.stringify({ cred_def_id: selectedDef.value.cred_def_id, files: evidenceFiles }) })
    }
    isApplyOpen.value = false
    await fetchData()
  } catch {
    alert(t.value.credentialsPage.submitFailed)
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

function canResubmit(status: string) {
  const s = statusEnumNameForStatus(CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, status).toUpperCase()
  return ["REUPLOAD", "RESUBMIT", "NEEDS_RESUBMIT", "APPLICATION_STATUS_REUPLOAD", "APPLICATION_STATUS_RESUBMIT"].includes(s)
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
  return !["PENDING", "APPLICATION_STATUS_PENDING", "APPROVED", "APPLICATION_STATUS_APPROVED"].includes(s)
}

function latestApplicationForDef(credDefId: string) {
  const matches = applications.value.filter((app) => app.cred_def_id === credDefId)
  return matches[0] || null
}

function applicationActionLabel(def: any) {
  const existing = latestApplicationForDef(def.cred_def_id)
  if (!existing) return t.value.credentialsPage.applyNow
  if (isPendingReviewStatus(existing.status)) return t.value.credentialsPage.applicationPendingHint
  if (isApprovedStatus(existing.status)) return t.value.credentialsPage.applicationApprovedHint
  if (canResubmit(existing.status)) return t.value.credentialsPage.appStatusResubmit
  return t.value.credentialsPage.applyNow
}

function isApplicationActionDisabled(def: any) {
  const existing = latestApplicationForDef(def.cred_def_id)
  return Boolean(existing && !canStartNewApplication(existing.status) && !canResubmit(existing.status))
}

function handleDefinitionAction(def: any) {
  const existing = latestApplicationForDef(def.cred_def_id)
  if (existing && canResubmit(existing.status)) {
    handleApplyClick(def, existing.app_id)
    return
  }
  handleApplyClick(def)
}

onMounted(fetchData)
</script>

<template>
  <AppShell content-class="p-0">
    <div class="min-h-screen bg-white lg:m-4 lg:overflow-hidden lg:rounded-xl lg:border lg:border-border">
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
          <div v-for="def in definitions" :key="def.cred_def_id" class="group relative flex flex-col overflow-hidden rounded-[16px] bg-white text-card-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all hover:-translate-y-0.5 hover:bg-[#f4fbfc] hover:shadow-md hover:shadow-primary/10">
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
              <div v-if="latestApplicationForDef(def.cred_def_id)" class="mt-3">
                <span :class="['badge w-fit gap-1', statusBadgeClassForStatus(CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, latestApplicationForDef(def.cred_def_id)?.status)]">
                  <component :is="statusIcon(latestApplicationForDef(def.cred_def_id)?.status)" class="h-4 w-4 text-black" />
                  {{ statusLabel(t, CANDIDATE_APPLICATION_STATUS_LABELS, latestApplicationForDef(def.cred_def_id)?.status, 'credentialsPage.appStatusUnknown') }}
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
        <div v-if="applications.length === 0" class="flex flex-col items-center justify-center rounded-[16px] bg-white px-4 py-14 text-center shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
          <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10">
            <FileText class="h-8 w-8 text-primary" />
          </div>
          <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.credentialsPage.noApplications }}</h3>
        </div>
        <div v-else class="overflow-hidden rounded-[16px] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
          <div class="space-y-2">
            <div v-for="app in applications" :key="app.app_id" class="grid grid-cols-[minmax(180px,1.5fr)_minmax(220px,2fr)_minmax(120px,1fr)_auto] items-center gap-4 px-4 py-4 text-sm transition-colors hover:bg-primary/10">
              <div class="min-w-0">
                <div class="truncate font-medium text-foreground">{{ definitions.find((d) => d.cred_def_id === app.cred_def_id)?.name || t.common.unknown }}</div>
                <div class="truncate text-xs text-muted-foreground">{{ app.app_id }}</div>
              </div>
              <div class="min-w-0 truncate text-muted-foreground">{{ app.audit_remark ? `${t.credentialsPage.auditRemark}: ${app.audit_remark}` : t.common.na }}</div>
              <span :class="['badge w-fit gap-1', statusBadgeClassForStatus(CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, app.status)]">
                <component :is="statusIcon(app.status)" class="h-5 w-5 text-black" />
                {{ statusLabel(t, CANDIDATE_APPLICATION_STATUS_LABELS, app.status, 'credentialsPage.appStatusUnknown') }}
              </span>
              <button v-if="canResubmit(app.status)" class="btn btn-primary cursor-pointer rounded-lg py-1 text-xs shadow-sm shadow-primary/20" @click="handleApplyClick(definitions.find((d) => d.cred_def_id === app.cred_def_id), app.app_id)">{{ t.credentialsPage.appStatusResubmit }}</button>
              <span v-else class="text-xs text-muted-foreground">{{ formatBackendDate(app.created_at).split(" ")[0] || t.common.na }}</span>
            </div>
          </div>
        </div>
      </section>
    </div>

      </main>
    </div>

    <div v-if="isApplyOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="isApplyOpen = false">
      <div class="w-full max-w-md rounded-[16px] bg-white p-4 shadow-lg shadow-slate-900/20">
        <div class="flex items-start justify-between gap-4">
          <h2 class="text-lg font-semibold leading-none tracking-tight">{{ selectedDef?.name }}</h2>
          <button class="flex h-8 w-8 cursor-pointer items-center justify-center rounded-full text-base leading-none text-muted-foreground transition-colors hover:bg-muted hover:text-foreground" @click="isApplyOpen = false">X</button>
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
