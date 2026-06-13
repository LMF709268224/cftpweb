<script setup lang="ts">
import { onMounted, ref } from "vue"
import { AlertCircle, Award, CheckCircle, Clock, FileText, XCircle } from "lucide-vue-next"
import { CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, CANDIDATE_APPLICATION_STATUS_LABELS, statusBadgeClassForStatus, statusEnumNameForStatus, statusLabel } from "@/lib/status-labels"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { formatBackendDate } from "@/lib/utils"
import { useTranslation } from "@/lib/language"

const { t } = useTranslation()
const definitions = ref<any[]>([])
const applications = ref<any[]>([])
const loading = ref(true)
const selectedDef = ref<any>(null)
const resubmitAppId = ref("")
const isApplyOpen = ref(false)
const uploadedFiles = ref<Record<string, { name: string; url: string; ext: string; hash: string; size: number }>>({})
const isSubmitting = ref(false)

async function sha256Hex(file: File) {
  const buffer = await file.arrayBuffer()
  const hash = await crypto.subtle.digest("SHA-256", buffer)
  return Array.from(new Uint8Array(hash)).map((byte) => byte.toString(16).padStart(2, "0")).join("")
}

async function fetchData() {
  loading.value = true
  try {
    const [defsRes, appsRes] = await Promise.all([apiClient("/api/credentials/definitions"), apiClient("/api/credentials/applications")])
    definitions.value = defsRes?.definitions || []
    applications.value = appsRes?.applications || []
  } finally {
    loading.value = false
  }
}

function handleApplyClick(def: any, appId = "") {
  if (!def) return
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
  const fileExt = file.name.includes(".") ? "." + file.name.split(".").pop() : ""
  try {
    const fileHash = await sha256Hex(file)
    const contentType = file.type || "application/octet-stream"
    const res = await apiClient("/api/credentials/upload-url", {
      method: "POST",
      body: JSON.stringify({ cred_def_id: selectedDef.value.cred_def_id, file_name: file.name, file_ext: fileExt, file_hash: fileHash, content_type: contentType, file_usage: constraintName }),
    })
    const uploadRes = await fetch(res.upload_url, { method: "PUT", headers: new Headers(res.signed_headers || {}), body: file })
    if (!uploadRes.ok) throw new Error("S3 upload failed")
    uploadedFiles.value = { ...uploadedFiles.value, [constraintName]: { name: file.name, url: res.file_key, ext: fileExt, hash: fileHash, size: file.size } }
  } catch {
    alert(t.value.credentialsPage.uploadFailed)
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

onMounted(fetchData)
</script>

<template>
  <AppShell>
    <div class="mb-8">
      <h1 class="flex items-center gap-2 text-3xl font-bold tracking-tight text-foreground"><Award class="h-8 w-8 text-primary" /> {{ t.credentialsPage.title }}</h1>
      <p class="mt-2 text-muted-foreground">{{ t.credentialsPage.subtitle }}</p>
    </div>
    <div v-if="loading">{{ t.common.loading }}</div>
    <div v-else class="space-y-10">
      <section>
        <h2 class="mb-4 flex items-center gap-2 text-xl font-semibold"><Award class="h-5 w-5" /> {{ t.credentialsPage.availableQualifications }}</h2>
        <div class="grid gap-6 md:grid-cols-3">
          <div v-for="def in definitions" :key="def.cred_def_id" class="flex flex-col rounded-xl border bg-card text-card-foreground shadow-sm">
            <div class="flex flex-col space-y-1.5 p-6">
              <h3 class="text-xl font-semibold leading-none tracking-tight">{{ def.name }}</h3>
              <span class="badge w-fit border-transparent bg-secondary text-secondary-foreground">{{ def.category }}</span>
            </div>
            <div class="flex flex-1 flex-col p-6 pt-0">
              <p class="mb-4 flex-1 text-sm text-muted-foreground">{{ def.description }}</p>
              <button class="btn btn-primary mt-4 w-full" @click="handleApplyClick(def)">{{ t.credentialsPage.applyNow }}</button>
            </div>
          </div>
        </div>
      </section>
      <hr />
      <section>
        <h2 class="mb-4 flex items-center gap-2 text-xl font-semibold"><FileText class="h-5 w-5" /> {{ t.credentialsPage.myApplications }}</h2>
        <div v-if="applications.length === 0" class="rounded-lg border border-dashed p-8 text-center text-muted-foreground">{{ t.credentialsPage.noApplications }}</div>
        <div v-else class="overflow-hidden rounded-md border bg-card">
          <div class="divide-y">
            <div v-for="app in applications" :key="app.app_id" class="grid grid-cols-[minmax(180px,1.5fr)_minmax(220px,2fr)_minmax(120px,1fr)_auto] items-center gap-4 px-4 py-3 text-sm">
              <div class="min-w-0">
                <div class="truncate font-medium text-foreground">{{ definitions.find((d) => d.cred_def_id === app.cred_def_id)?.name || t.common.unknown }}</div>
                <div class="truncate text-xs text-muted-foreground">{{ app.app_id }}</div>
              </div>
              <div class="min-w-0 truncate text-muted-foreground">{{ app.audit_remark ? `${t.credentialsPage.auditRemark}: ${app.audit_remark}` : t.common.na }}</div>
              <span :class="['badge w-fit gap-1', statusBadgeClassForStatus(CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, app.status)]">
                <component :is="statusIcon(app.status)" class="h-5 w-5 text-black" />
                {{ statusLabel(t, CANDIDATE_APPLICATION_STATUS_LABELS, app.status, 'credentialsPage.appStatusUnknown') }}
              </span>
              <button v-if="canResubmit(app.status)" class="btn btn-primary py-1 text-xs" @click="handleApplyClick(definitions.find((d) => d.cred_def_id === app.cred_def_id), app.app_id)">{{ t.credentialsPage.appStatusResubmit }}</button>
              <span v-else class="text-xs text-muted-foreground">{{ formatBackendDate(app.created_at).split(" ")[0] || t.common.na }}</span>
            </div>
          </div>
        </div>
      </section>
    </div>

    <div v-if="isApplyOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="isApplyOpen = false">
      <div class="w-full max-w-md rounded-xl border bg-card p-6 shadow-lg">
        <h2 class="text-lg font-semibold leading-none tracking-tight">{{ selectedDef?.name }}</h2>
        <div class="space-y-4 py-4">
          <p class="text-sm text-muted-foreground">{{ t.credentialsPage.description }}: {{ selectedDef?.description }}</p>
          <div class="space-y-4 border-t pt-4">
            <h4 class="text-sm font-semibold">{{ t.credentialsPage.uploadMaterials }}</h4>
            <div v-for="constraint in selectedDef?.file_constraints || []" :key="constraint.name" class="space-y-2 rounded-lg bg-muted p-3">
              <div class="flex justify-between">
                <span class="font-medium">{{ constraint.name }}</span>
                <span :class="['badge border-transparent', constraint.is_required ? 'bg-destructive text-destructive-foreground' : 'bg-secondary text-secondary-foreground']">{{ constraint.is_required ? t.credentialsPage.required : t.credentialsPage.optional }}</span>
              </div>
              <div class="mt-2 flex items-center gap-2">
                <button type="button" class="btn btn-outline px-3 py-1.5 text-xs hover:border-emerald-500 hover:bg-emerald-500 hover:text-white" @click="triggerFileInput(constraint.name)">
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
          <button class="btn btn-outline" @click="isApplyOpen = false">{{ t.common.cancel }}</button>
          <button class="btn btn-primary" :disabled="isSubmitting || !(selectedDef?.file_constraints?.every((c: any) => !c.is_required || uploadedFiles[c.name]) && selectedDef?.file_constraints?.length > 0)" @click="handleSubmitApplication">
            {{ isSubmitting ? t.credentialsPage.submitting : t.credentialsPage.submitApplication }}
          </button>
        </div>
      </div>
    </div>
  </AppShell>
</template>
