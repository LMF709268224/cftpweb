<script setup lang="ts">
import { onMounted, ref } from "vue"
import { RouterLink } from "vue-router"
import { Award, Calendar, CheckCircle2, Download, ExternalLink, Eye, Share2 } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { formatBackendDate } from "@/lib/utils"
import { useTranslation } from "@/lib/language"

const { t } = useTranslation()
const certificates = ref<any[]>([])

function openCertificate(url?: string) {
  if (url) window.open(url, "_blank")
}

onMounted(async () => {
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
  }
})
</script>

<template>
  <AppShell>
    <div class="mb-8">
      <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.certificatesPage.title }}</h1>
      <p class="mt-1 text-muted-foreground">{{ t.certificatesPage.subtitle }}</p>
    </div>
    <div class="grid gap-6 lg:grid-cols-2">
      <div v-for="cert in certificates" :key="cert.id" class="group relative overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
        <div class="relative bg-gradient-to-br from-primary via-primary/90 to-primary p-6 text-white">
          <div class="absolute -right-8 -top-8 h-32 w-32 rounded-full bg-white/10" />
          <div class="relative flex items-start justify-between">
            <div>
              <span class="badge mb-3 border-0 bg-white/20 text-white"><CheckCircle2 class="mr-1 h-3 w-3" /> {{ t.certificatesPage.active }}</span>
              <h3 class="mb-1 text-xl font-bold">{{ cert.name }}</h3>
              <p class="text-sm text-white/80">{{ cert.description }}</p>
            </div>
            <div class="flex h-14 w-14 items-center justify-center rounded-full bg-white/20 backdrop-blur-sm"><Award class="h-7 w-7" /></div>
          </div>
        </div>
        <div class="p-6">
          <div class="mb-6 grid grid-cols-2 gap-4">
            <div>
              <p class="mb-1 text-xs text-muted-foreground">{{ t.certificatesPage.issueDate }}</p>
              <p class="flex items-center gap-1.5 font-medium text-card-foreground"><Calendar class="h-4 w-4 text-muted-foreground" /> {{ cert.issueDate }}</p>
            </div>
            <div>
              <p class="mb-1 text-xs text-muted-foreground">{{ t.certificatesPage.expiryDate }}</p>
              <p class="flex items-center gap-1.5 font-medium text-card-foreground"><Calendar class="h-4 w-4 text-muted-foreground" /> {{ cert.expiryDate }}</p>
            </div>
          </div>
          <div class="mb-6 rounded-lg bg-muted/50 p-3">
            <p class="mb-1 text-xs text-muted-foreground">{{ t.certificatesPage.certificateId }}</p>
            <p class="font-mono text-sm text-card-foreground">{{ cert.credentialId }}</p>
          </div>
          <div class="flex gap-3">
            <button class="btn btn-primary flex-1" :disabled="!cert.pdfUrl" @click="openCertificate(cert.pdfUrl)">
              <Download class="h-4 w-4" /> {{ cert.pdfUrl ? t.certificatesPage.downloadCertificate : t.certificatesPage.certificateGenerating }}
            </button>
            <button class="btn btn-outline px-3" disabled><Share2 class="h-4 w-4" /></button>
            <button class="btn btn-outline px-3" :disabled="!cert.pdfUrl" @click="openCertificate(cert.pdfUrl)"><Eye class="h-4 w-4" /></button>
          </div>
        </div>
      </div>
      <div class="flex flex-col items-center justify-center rounded-2xl border-2 border-dashed border-border p-12 text-center">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-muted"><Award class="h-8 w-8 text-muted-foreground" /></div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.certificatesPage.keepLearningTitle }}</h3>
        <p class="mb-4 text-sm text-muted-foreground">{{ t.certificatesPage.keepLearningDesc }}</p>
        <RouterLink to="/courses" class="btn btn-outline"><ExternalLink class="h-4 w-4" /> {{ t.certificatesPage.browseCourses }}</RouterLink>
      </div>
    </div>
  </AppShell>
</template>
