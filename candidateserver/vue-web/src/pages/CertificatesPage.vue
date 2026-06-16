<script setup lang="ts">
import { onMounted, ref } from "vue"
import { RouterLink } from "vue-router"
import { Award, Calendar, CheckCircle2, Download, ExternalLink, Eye, Loader2, Share2 } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { formatBackendDate } from "@/lib/utils"
import { useTranslation } from "@/lib/language"

const { t } = useTranslation()
const certificates = ref<any[]>([])
const loading = ref(false)
const CERTIFICATE_PREVIEW_TIMEOUT_MS = 20000

function openCertificate(url?: string) {
  if (url) window.open(url, "_blank")
}

async function previewCertificate(url?: string) {
  if (!url) return
  if (loading.value) return
  loading.value = true
  const controller = new AbortController()
  const timeoutId = window.setTimeout(() => controller.abort(), CERTIFICATE_PREVIEW_TIMEOUT_MS)
  try {
    const response = await fetch(url, { signal: controller.signal })
    if (!response.ok) throw new Error("fetch failed")
    const blob = await response.blob()
    const blobUrl = URL.createObjectURL(blob)
    window.open(blobUrl, "_blank")
  } catch (err) {
    window.open(url, "_blank")
  } finally {
    window.clearTimeout(timeoutId)
    loading.value = false
  }
}

onMounted(async () => {
  loading.value = true
  try {
    const res = await apiClient("/api/certificates")
    if (res?.certificates) {
      certificates.value = res.certificates.map((cert: any) => ({
        id: cert.cred_id || cert.catalog_id,
        name: cert.name,
        description: cert.description || "",
        issueDate: cert.created_at ? formatBackendDate(cert.created_at).split(" ")[0] : t.value.common.na,
        expiryDate: cert.valid_until ? formatBackendDate(cert.valid_until).split(" ")[0] : t.value.common.permanent,
        credentialId: cert.cred_guid || cert.cred_id || t.value.common.na,
        pdfUrl: cert.files?.find((f: any) => f.file_type === 2 || f.file_ext === ".pdf" || f.file_ext === "pdf" || f.file_name?.endsWith(".pdf"))?.view_url || "",
      }))
    }
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <AppShell content-class="p-4">
    <div class="mb-4 overflow-hidden rounded-[16px] bg-white shadow-[0_12px_30px_rgba(15,74,82,0.06)]">
      <div class="bg-gradient-to-r from-[#ecfbf7] via-white to-[#f4fbff] p-4">
        <div class="mb-3 inline-flex items-center gap-2 rounded-full bg-primary/10 px-3 py-1 text-xs font-semibold text-primary">
          <Award class="h-3.5 w-3.5" />
          {{ t.sidebar.certificates }}
        </div>
        <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.certificatesPage.title }}</h1>
        <p class="mt-2 text-muted-foreground">{{ t.certificatesPage.subtitle }}</p>
      </div>
    </div>

    <div v-if="loading" class="flex items-center justify-center gap-2 rounded-[16px] bg-white py-16 text-muted-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <Loader2 class="h-5 w-5 animate-spin" />
      <span>{{ t.common.loading }}</span>
    </div>
    <div v-else class="grid gap-4 lg:grid-cols-2">
      <div v-for="cert in certificates" :key="cert.id" class="group relative overflow-hidden rounded-[16px] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all hover:-translate-y-0.5 hover:border-primary/25 hover:shadow-md hover:shadow-primary/10">
        <div class="relative bg-primary p-4 text-white">
          <div class="relative flex items-start justify-between">
            <div>
              <span class="badge mb-3 border-0 bg-white/20 text-white"><CheckCircle2 class="mr-1 h-3 w-3" /> {{ t.certificatesPage.active }}</span>
              <h3 class="mb-1 text-xl font-bold">{{ cert.name }}</h3>
              <p class="text-sm text-white/80">{{ cert.description }}</p>
            </div>
            <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-white/20 backdrop-blur-sm"><Award class="h-6 w-6" /></div>
          </div>
        </div>
        <div class="p-4">
          <div class="mb-4 grid grid-cols-2 gap-4">
            <div class="rounded-lg bg-[#f7fbfc] p-3">
              <p class="mb-1 text-xs text-muted-foreground">{{ t.certificatesPage.issueDate }}</p>
              <p class="flex items-center gap-1.5 font-medium text-card-foreground"><Calendar class="h-4 w-4 text-muted-foreground" /> {{ cert.issueDate }}</p>
            </div>
            <div class="rounded-lg bg-[#f7fbfc] p-3">
              <p class="mb-1 text-xs text-muted-foreground">{{ t.certificatesPage.expiryDate }}</p>
              <p class="flex items-center gap-1.5 font-medium text-card-foreground"><Calendar class="h-4 w-4 text-muted-foreground" /> {{ cert.expiryDate }}</p>
            </div>
          </div>
          <div class="mb-4 rounded-lg bg-[#f7fbfc] p-3">
            <p class="mb-1 text-xs text-muted-foreground">{{ t.certificatesPage.certificateId }}</p>
            <p class="font-mono text-sm text-card-foreground">{{ cert.credentialId }}</p>
          </div>
          <div class="flex gap-3">
            <button class="btn btn-primary flex-1 rounded-lg shadow-sm shadow-primary/20" :disabled="!cert.pdfUrl" @click="openCertificate(cert.pdfUrl)">
              <Download class="h-4 w-4" /> {{ cert.pdfUrl ? t.certificatesPage.downloadCertificate : t.certificatesPage.certificateGenerating }}
            </button>
            <button class="btn btn-outline rounded-lg px-3" disabled><Share2 class="h-4 w-4" /></button>
            <button class="btn btn-outline rounded-lg px-3" :disabled="!cert.pdfUrl" @click="previewCertificate(cert.pdfUrl)"><Eye class="h-4 w-4" /></button>
          </div>
        </div>
      </div>
      <div class="flex flex-col items-center justify-center rounded-[16px] bg-white p-4 py-14 text-center shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10"><Award class="h-8 w-8 text-primary" /></div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.certificatesPage.keepLearningTitle }}</h3>
        <p class="mb-4 text-sm text-muted-foreground">{{ t.certificatesPage.keepLearningDesc }}</p>
        <RouterLink to="/courses" class="btn btn-outline rounded-lg hover:border-primary/25 hover:bg-primary/10 hover:text-primary"><ExternalLink class="h-4 w-4" /> {{ t.certificatesPage.browseCourses }}</RouterLink>
      </div>
    </div>
  </AppShell>
</template>
