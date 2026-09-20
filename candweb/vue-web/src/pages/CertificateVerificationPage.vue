<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from "vue"
import { AlertTriangle, CheckCircle2, FileCheck2, FileUp, Fingerprint, Loader2, ShieldCheck, XCircle } from "lucide-vue-next"
import GfiHeader from "@/components/GfiHeader.vue"
import {
  isActiveLifecycleStatus,
  isExpiredLifecycleStatus,
  isRevokedLifecycleStatus,
  verifyCredentialPdf,
  type CredentialVerificationResult,
  type VerificationStep,
} from "@/lib/credentialVerification"
import { useTranslation } from "@/lib/language"

const { t, lang } = useTranslation()
const fileInput = ref<HTMLInputElement | null>(null)
const selectedFile = ref<File | null>(null)
const isDragging = ref(false)
const isVerifying = ref(false)
const errorMessage = ref("")
const result = ref<CredentialVerificationResult | null>(null)
const steps = ref<VerificationStep[]>([
  { id: "parse", status: "pending", detail: "" },
  { id: "digest", status: "pending", detail: "" },
  { id: "chain", status: "pending", detail: "" },
  { id: "certificate", status: "pending", detail: "" },
  { id: "signature", status: "pending", detail: "" },
  { id: "identity", status: "pending", detail: "" },
  { id: "lifecycle", status: "pending", detail: "" },
])

const copy = computed(() => t.value.certificateVerificationPage)
const stepLabels = computed(() => copy.value.steps)

const verdict = computed(() => {
  if (!result.value) return null
  const status = result.value.lifecycle.status
  if (result.value.lifecycle.is_valid && isActiveLifecycleStatus(status)) return { key: "active", icon: CheckCircle2, title: copy.value.active, desc: copy.value.activeDesc, className: "verification-result--active" }
  if (isRevokedLifecycleStatus(status)) return { key: "revoked", icon: AlertTriangle, title: copy.value.revoked, desc: copy.value.revokedDesc, className: "verification-result--revoked" }
  if (isExpiredLifecycleStatus(status)) return { key: "expired", icon: AlertTriangle, title: copy.value.expired, desc: copy.value.expiredDesc, className: "verification-result--expired" }
  return { key: "invalid", icon: XCircle, title: copy.value.invalid, desc: copy.value.invalidDesc, className: "verification-result--invalid" }
})

function displayDate(value?: string) {
  if (!value) return t.value.common.na
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat(lang.value === "zh" ? "zh-CN" : "en-US", { year: "numeric", month: "short", day: "numeric" }).format(date)
}

function formatFileSize(value: number) {
  if (value < 1024 * 1024) return `${Math.max(1, Math.round(value / 1024))} KB`
  return `${(value / (1024 * 1024)).toFixed(1)} MB`
}

function chooseFile() {
  if (isVerifying.value) return
  if (fileInput.value) fileInput.value.value = ""
  fileInput.value?.click()
}

function setFile(file?: File) {
  if (!file || isVerifying.value) return
  if (file.type && file.type !== "application/pdf" && !file.name.toLowerCase().endsWith(".pdf")) {
    errorMessage.value = copy.value.invalidFile
    return
  }
  selectedFile.value = file
  errorMessage.value = ""
  result.value = null
  steps.value = steps.value.map((step) => ({ ...step, status: "pending", detail: "" }))
  void verify(file)
}

function onFileChange(event: Event) {
  setFile((event.target as HTMLInputElement).files?.[0])
}

function onDrop(event: DragEvent) {
  isDragging.value = false
  setFile(event.dataTransfer?.files?.[0])
}

function updateStep(update: VerificationStep) {
  const index = steps.value.findIndex((step) => step.id === update.id)
  if (index >= 0) steps.value[index] = update
}

async function verify(file: File) {
  isVerifying.value = true
  errorMessage.value = ""
  result.value = null
  try {
    result.value = await verifyCredentialPdf(file, updateStep, copy.value)
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : String(error)
    const running = steps.value.find((step) => step.status === "running")
    if (running) updateStep({ ...running, status: "failed" })
  } finally {
    isVerifying.value = false
  }
}

onBeforeUnmount(() => {
  selectedFile.value = null
})
</script>

<template>
  <div class="certificate-verification-page">
    <GfiHeader auth-target="/login" :auth-new-tab="false" />
    <main class="verification-main">
      <section class="verification-heading">
        <p class="verification-eyebrow"><ShieldCheck class="h-4 w-4" /> {{ copy.eyebrow }}</p>
        <h1>{{ copy.title }}</h1>
        <p>{{ copy.subtitle }}</p>
      </section>

      <section class="verification-layout">
        <div class="verification-upload-panel">
          <div
            class="verification-dropzone"
            :class="{ 'verification-dropzone--active': isDragging, 'verification-dropzone--selected': selectedFile }"
            @dragenter.prevent="isDragging = true"
            @dragover.prevent="isDragging = true"
            @dragleave.prevent="isDragging = false"
            @drop.prevent="onDrop"
            @click="chooseFile"
            @keydown.enter.prevent="chooseFile"
            @keydown.space.prevent="chooseFile"
            role="button"
            tabindex="0"
          >
            <input ref="fileInput" type="file" accept="application/pdf,.pdf" class="sr-only" @change="onFileChange" />
            <FileCheck2 v-if="selectedFile" class="verification-dropzone-icon text-[#2f7d4f]" />
            <FileUp v-else class="verification-dropzone-icon text-[#0957f9]" />
            <h2>{{ selectedFile ? selectedFile.name : copy.drop }}</h2>
            <p v-if="selectedFile" class="verification-file-meta">{{ formatFileSize(selectedFile.size) }}</p>
            <p v-else>{{ copy.browse }}</p>
          </div>
          <p class="verification-privacy"><Fingerprint class="h-4 w-4" /> {{ copy.privacy }}</p>
          <p v-if="errorMessage" class="verification-error" role="alert"><XCircle class="h-4 w-4 shrink-0" /> {{ errorMessage }}</p>
        </div>

        <div class="verification-steps-panel">
          <div class="verification-panel-heading"><span>{{ copy.stepsTitle }}</span><span class="verification-status-dot" :class="{ 'verification-status-dot--running': isVerifying }" /></div>
          <ol class="verification-steps">
            <li v-for="(step, index) in steps" :key="step.id" class="verification-step" :class="`verification-step--${step.status}`">
              <span class="verification-step-index"><Loader2 v-if="step.status === 'running'" class="h-3.5 w-3.5 animate-spin" /><CheckCircle2 v-else-if="step.status === 'success'" class="h-4 w-4" /><XCircle v-else-if="step.status === 'failed'" class="h-4 w-4" /><span v-else>{{ index + 1 }}</span></span>
              <span class="verification-step-copy"><strong>{{ stepLabels[step.id] }}</strong><small v-if="step.detail">{{ step.detail }}</small></span>
            </li>
          </ol>
        </div>
      </section>

      <section v-if="verdict && result" class="verification-result" :class="verdict.className">
        <div class="verification-result-heading"><component :is="verdict.icon" class="h-7 w-7" /><div><h2>{{ verdict.title }}</h2><p>{{ verdict.desc }}</p></div></div>
        <div class="verification-result-grid">
          <div><span>{{ copy.certificate }}</span><strong>{{ result.lifecycle.cred_def_name || t.common.na }}</strong></div>
          <div><span>{{ copy.ulid }}</span><strong class="verification-mono">{{ result.ulid }}</strong></div>
          <div><span>{{ copy.issued }}</span><strong>{{ displayDate(result.lifecycle.issue_date) }}</strong></div>
          <div><span>{{ copy.validUntil }}</span><strong>{{ displayDate(result.lifecycle.valid_until) }}</strong></div>
          <div><span>{{ copy.signer }}</span><strong>{{ result.signerSubject }}</strong></div>
          <div><span>{{ copy.crypto }}</span><strong>{{ result.cryptoSuite }}</strong></div>
          <div class="verification-result-wide"><span>{{ copy.fingerprint }}</span><strong class="verification-mono">{{ result.leafFingerprint }}</strong></div>
          <div class="verification-result-wide"><span>{{ copy.root }}</span><strong class="verification-mono">{{ result.rootKeyId || t.common.na }}</strong></div>
          <div v-if="verdict.key === 'revoked' && result.lifecycle.revoke_reason" class="verification-result-wide"><span>{{ copy.revokeReason }}</span><strong>{{ result.lifecycle.revoke_reason }}</strong></div>
        </div>
      </section>
    </main>
  </div>
</template>

<style scoped>
.certificate-verification-page { min-height: 100vh; background: #edeef2; color: #1a2233; font-family: "DM Sans GFI", "Noto Sans SC", "Microsoft YaHei", system-ui, sans-serif; }
.verification-main { width: min(1120px, calc(100% - 40px)); margin: 0 auto; padding: 48px 0; }
.verification-heading { max-width: 720px; margin-bottom: 32px; }
.verification-eyebrow { display: inline-flex; align-items: center; gap: 8px; margin: 0 0 12px; color: #0957f9; font-size: 13px; font-weight: 700; letter-spacing: 0; text-transform: uppercase; }
.verification-heading h1 { margin: 0; color: #002a66; font-family: "Syne GFI", "DM Sans GFI", "Noto Sans SC", sans-serif; font-size: 30px; line-height: 1.2; letter-spacing: 0; }
.verification-heading > p:last-child { max-width: 600px; margin: 16px 0 0; color: #525e70; font-size: 15px; line-height: 1.6; }
.verification-layout { display: grid; grid-template-columns: minmax(0, 1.2fr) minmax(320px, .8fr); gap: 24px; align-items: start; }
.verification-upload-panel, .verification-steps-panel, .verification-result { border: 1px solid rgba(0, 42, 102, .14); border-radius: 14px; background: #fff; }
.verification-upload-panel { padding: 24px; }
.verification-dropzone { display: flex; min-height: 320px; cursor: pointer; flex-direction: column; align-items: center; justify-content: center; border: 1.5px dashed rgba(0, 42, 102, .32); border-radius: 8px; background: #fff; padding: 32px; text-align: center; transition: border-color 160ms ease, background-color 160ms ease; }
.verification-dropzone:hover, .verification-dropzone--active { border-color: #0957f9; background: #edeef2; }
.verification-dropzone:focus-visible { outline: 2px solid #0957f9; outline-offset: 2px; }
.verification-dropzone--selected { border-color: #2f7d4f; background: #ecf5f0; }
.verification-dropzone-icon { width: 44px; height: 44px; margin-bottom: 20px; }
.verification-dropzone h2 { max-width: 100%; margin: 0; overflow-wrap: anywhere; color: #002a66; font-size: 18px; line-height: 1.4; }
.verification-dropzone p { margin: 8px 0 0; color: #525e70; font-size: 13px; }
.verification-file-meta { color: #2f7d4f !important; font-weight: 700; }
.verification-privacy { display: flex; align-items: center; gap: 8px; margin: 16px 0 0; color: #5b6b87; font-size: 13px; line-height: 1.5; }
.verification-error { display: flex; gap: 8px; margin: 16px 0 0; border-left: 3px solid #b3372f; background: #fbedeb; padding: 12px; color: #b3372f; font-size: 13px; line-height: 1.5; }
.verification-steps-panel { padding: 24px; }
.verification-panel-heading { display: flex; align-items: center; justify-content: space-between; color: #002a66; font-size: 15px; font-weight: 700; }
.verification-status-dot { width: 8px; height: 8px; border-radius: 50%; background: #c9cdd6; }
.verification-status-dot--running { background: #c9962e; }
.verification-steps { display: grid; gap: 4px; margin: 20px 0 0; padding: 0; list-style: none; }
.verification-step { display: flex; min-height: 52px; align-items: flex-start; gap: 12px; padding: 8px 0; }
.verification-step-index { display: inline-flex; width: 28px; height: 28px; flex: 0 0 28px; align-items: center; justify-content: center; border: 1px solid rgba(0, 42, 102, .22); border-radius: 50%; color: #5b6b87; font-size: 12px; font-weight: 700; }
.verification-step-copy { display: grid; min-width: 0; gap: 3px; padding-top: 2px; }
.verification-step-copy strong { color: #1a2233; font-size: 14px; line-height: 1.35; }
.verification-step-copy small { overflow-wrap: anywhere; color: #5b6b87; font-family: ui-monospace, SFMono-Regular, Consolas, monospace; font-size: 12px; line-height: 1.45; }
.verification-step--running .verification-step-index { border-color: #c9962e; color: #a6600c; }
.verification-step--running .verification-step-copy strong { color: #a6600c; }
.verification-step--success .verification-step-index { border-color: #2f7d4f; background: #ecf5f0; color: #2f7d4f; }
.verification-step--success .verification-step-copy strong { color: #2f7d4f; }
.verification-step--failed .verification-step-index { border-color: #b3372f; background: #fbedeb; color: #b3372f; }
.verification-step--failed .verification-step-copy strong { color: #b3372f; }
.verification-result { margin-top: 24px; padding: 24px; }
.verification-result-heading { display: flex; gap: 14px; align-items: flex-start; }
.verification-result-heading h2 { margin: 0; color: #002a66; font-size: 20px; }
.verification-result-heading p { margin: 8px 0 0; color: #525e70; font-size: 14px; line-height: 1.6; }
.verification-result--active { border-top: 4px solid #2f7d4f; }
.verification-result--active .verification-result-heading > svg { color: #2f7d4f; }
.verification-result--revoked, .verification-result--expired { border-top: 4px solid #c9962e; }
.verification-result--revoked .verification-result-heading > svg, .verification-result--expired .verification-result-heading > svg { color: #a6600c; }
.verification-result--invalid { border-top: 4px solid #b3372f; }
.verification-result--invalid .verification-result-heading > svg { color: #b3372f; }
.verification-result-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 1px; margin-top: 24px; border: 1px solid rgba(0, 42, 102, .14); background: rgba(0, 42, 102, .14); }
.verification-result-grid > div { min-width: 0; background: #fff; padding: 14px; }
.verification-result-grid span { display: block; margin-bottom: 4px; color: #5b6b87; font-size: 12px; font-weight: 700; letter-spacing: 0; text-transform: uppercase; }
.verification-result-grid strong { display: block; overflow-wrap: anywhere; color: #1a2233; font-size: 14px; line-height: 1.45; }
.verification-result-wide { grid-column: 1 / -1; }
.verification-mono { font-family: ui-monospace, SFMono-Regular, Consolas, monospace; font-size: 12px !important; }
@media (max-width: 800px) { .verification-main { width: min(100% - 32px, 680px); padding: 32px 0 48px; } .verification-layout { grid-template-columns: 1fr; } .verification-dropzone { min-height: 264px; } }
@media (max-width: 520px) { .verification-heading h1 { font-size: 24px; } .verification-upload-panel, .verification-steps-panel, .verification-result { padding: 16px; } .verification-result-grid { grid-template-columns: 1fr; } .verification-result-wide { grid-column: auto; } }
</style>
