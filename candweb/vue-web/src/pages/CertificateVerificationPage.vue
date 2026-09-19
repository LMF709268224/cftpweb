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
          >
            <input ref="fileInput" type="file" accept="application/pdf,.pdf" class="sr-only" @change="onFileChange" />
            <FileCheck2 v-if="selectedFile" class="verification-dropzone-icon text-emerald-600" />
            <FileUp v-else class="verification-dropzone-icon text-primary" />
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
.certificate-verification-page { min-height: 100vh; background: #f7f9fc; color: #102044; }
.verification-main { width: min(1120px, calc(100% - 40px)); margin: 0 auto; padding: 72px 0 100px; }
.verification-heading { max-width: 720px; margin-bottom: 38px; }
.verification-eyebrow { display: inline-flex; align-items: center; gap: 8px; margin: 0 0 16px; color: #2a63cf; font-size: 13px; font-weight: 700; letter-spacing: .08em; text-transform: uppercase; }
.verification-heading h1 { margin: 0; color: #102044; font-size: clamp(2.1rem, 5vw, 4rem); line-height: 1.08; letter-spacing: -.03em; }
.verification-heading > p:last-child { max-width: 600px; margin: 18px 0 0; color: #5b6b86; font-size: 17px; line-height: 1.7; }
.verification-layout { display: grid; grid-template-columns: minmax(0, 1.2fr) minmax(320px, .8fr); gap: 22px; align-items: start; }
.verification-upload-panel, .verification-steps-panel, .verification-result { border: 1px solid #e0e7f1; background: #fff; box-shadow: 0 18px 42px rgba(16, 32, 68, .06); }
.verification-upload-panel { padding: 22px; }
.verification-dropzone { display: flex; min-height: 320px; cursor: pointer; flex-direction: column; align-items: center; justify-content: center; border: 1.5px dashed #b9c6da; background: #fbfcfe; padding: 32px; text-align: center; transition: border-color .2s ease, background .2s ease; }
.verification-dropzone:hover, .verification-dropzone--active { border-color: #3f73d8; background: #f2f6ff; }
.verification-dropzone--selected { border-color: #51b887; background: #f3fbf7; }
.verification-dropzone-icon { width: 44px; height: 44px; margin-bottom: 20px; }
.verification-dropzone h2 { max-width: 100%; margin: 0; overflow-wrap: anywhere; color: #162746; font-size: 18px; line-height: 1.4; }
.verification-dropzone p { margin: 9px 0 0; color: #71809a; font-size: 14px; }
.verification-file-meta { color: #277e58 !important; font-weight: 700; }
.verification-privacy { display: flex; align-items: center; gap: 7px; margin: 17px 0 0; color: #6b7b95; font-size: 13px; line-height: 1.5; }
.verification-error { display: flex; gap: 8px; margin: 17px 0 0; border-left: 3px solid #d84b4b; background: #fff5f5; padding: 12px 13px; color: #a73333; font-size: 13px; line-height: 1.55; }
.verification-steps-panel { padding: 22px; }
.verification-panel-heading { display: flex; align-items: center; justify-content: space-between; color: #223453; font-size: 15px; font-weight: 800; }
.verification-status-dot { width: 9px; height: 9px; border-radius: 50%; background: #b6c2d4; }
.verification-status-dot--running { background: #e4a93a; box-shadow: 0 0 0 5px rgba(228, 169, 58, .14); }
.verification-steps { display: grid; gap: 4px; margin: 20px 0 0; padding: 0; list-style: none; }
.verification-step { display: flex; min-height: 52px; align-items: flex-start; gap: 12px; padding: 8px 0; }
.verification-step-index { display: inline-flex; width: 27px; height: 27px; flex: 0 0 27px; align-items: center; justify-content: center; border: 1px solid #d6dfed; border-radius: 50%; color: #8290a6; font-size: 12px; font-weight: 700; }
.verification-step-copy { display: grid; min-width: 0; gap: 3px; padding-top: 2px; }
.verification-step-copy strong { color: #455571; font-size: 14px; line-height: 1.35; }
.verification-step-copy small { overflow-wrap: anywhere; color: #7b8aa2; font-family: ui-monospace, SFMono-Regular, Consolas, monospace; font-size: 11px; line-height: 1.45; }
.verification-step--running .verification-step-index { border-color: #d5962e; color: #b27312; }
.verification-step--running .verification-step-copy strong { color: #a66b0e; }
.verification-step--success .verification-step-index { border-color: #42a976; background: #eefaf4; color: #218250; }
.verification-step--success .verification-step-copy strong { color: #267c52; }
.verification-step--failed .verification-step-index { border-color: #d25151; background: #fff3f3; color: #c13d3d; }
.verification-step--failed .verification-step-copy strong { color: #b13b3b; }
.verification-result { margin-top: 22px; padding: 26px; }
.verification-result-heading { display: flex; gap: 14px; align-items: flex-start; }
.verification-result-heading h2 { margin: 0; color: #1d2e4e; font-size: 23px; }
.verification-result-heading p { margin: 7px 0 0; color: #60718d; font-size: 14px; line-height: 1.6; }
.verification-result--active { border-top: 4px solid #35a36d; }
.verification-result--active .verification-result-heading > svg { color: #2a9562; }
.verification-result--revoked, .verification-result--expired { border-top: 4px solid #d18b24; }
.verification-result--revoked .verification-result-heading > svg, .verification-result--expired .verification-result-heading > svg { color: #bf7911; }
.verification-result--invalid { border-top: 4px solid #c84545; }
.verification-result--invalid .verification-result-heading > svg { color: #c13f3f; }
.verification-result-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 1px; margin-top: 25px; border: 1px solid #e3e9f2; background: #e3e9f2; }
.verification-result-grid > div { min-width: 0; background: #fff; padding: 14px; }
.verification-result-grid span { display: block; margin-bottom: 5px; color: #75849b; font-size: 11px; font-weight: 700; letter-spacing: .04em; text-transform: uppercase; }
.verification-result-grid strong { display: block; overflow-wrap: anywhere; color: #293b5b; font-size: 14px; line-height: 1.45; }
.verification-result-wide { grid-column: 1 / -1; }
.verification-mono { font-family: ui-monospace, SFMono-Regular, Consolas, monospace; font-size: 12px !important; }
@media (max-width: 800px) { .verification-main { width: min(100% - 28px, 680px); padding: 48px 0 68px; } .verification-layout { grid-template-columns: 1fr; } .verification-dropzone { min-height: 260px; } }
@media (max-width: 520px) { .verification-heading h1 { font-size: 2.25rem; } .verification-upload-panel, .verification-steps-panel, .verification-result { padding: 17px; } .verification-result-grid { grid-template-columns: 1fr; } .verification-result-wide { grid-column: auto; } }
</style>
