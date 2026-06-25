<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { RouterLink } from "vue-router"
import { Award, Calendar, CheckCircle2, Download, ExternalLink, Eye, Loader2, Share2, Sparkles, X } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import rewGif from "@/assets/rew.gif"
import { apiClient } from "@/lib/apiClient"
import { formatBackendDateOnly } from "@/lib/utils"
import { useTranslation } from "@/lib/language"

const { t } = useTranslation()
const certificates = ref<any[]>([])
const loading = ref(false)
const celebrationVisible = ref(false)
const CERTIFICATE_PREVIEW_TIMEOUT_MS = 20000
const CERTIFICATE_CELEBRATED_IDS_KEY = "cftp-certificates-celebrated-ids"

type CertificatesModalTexts = {
  celebrationModalTitle?: string
  celebrationModalHeadline?: string
  celebrationModalDesc?: string
  celebrationModalDownload?: string
  celebrationModalShare?: string
  celebrationModalDismiss?: string
  shareText?: string
}

const certificateTexts = computed(() => {
  const page = t.value.certificatesPage as typeof t.value.certificatesPage & CertificatesModalTexts
  return {
    celebrationModalTitle: page.celebrationModalTitle ?? "恭喜！",
    celebrationModalHeadline: page.celebrationModalHeadline ?? "您已获得 CFtP 专业认证证书",
    celebrationModalDesc:
      page.celebrationModalDesc ??
      "您的学习成果已经正式转化为专业证书。现在可以下载荣誉证书，或一键分享这一值得庆祝的成就。",
    celebrationModalDownload: page.celebrationModalDownload ?? "下载您的荣誉证书",
    celebrationModalShare: page.celebrationModalShare ?? "一键分享",
    celebrationModalDismiss: page.celebrationModalDismiss ?? "稍后再说",
    shareText: page.shareText ?? "我已获得 CFtP 专业认证，欢迎查看我的学习成果。",
  }
})

const featuredCertificate = computed(() => certificates.value[0] ?? null)

function openCertificate(url?: string) {
  if (url) window.open(url, "_blank")
}

function getCelebrationCertificateKey(cert?: { id?: string; credentialId?: string }) {
  return cert?.credentialId || cert?.id || ""
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

function downloadFeaturedCertificate() {
  openCertificate(featuredCertificate.value?.pdfUrl)
}

function closeCelebrationModal() {
  celebrationVisible.value = false
  try {
    const certificateKey = getCelebrationCertificateKey(featuredCertificate.value)
    if (!certificateKey) return
    const storedIds = JSON.parse(window.localStorage.getItem(CERTIFICATE_CELEBRATED_IDS_KEY) || "[]") as string[]
    if (!storedIds.includes(certificateKey)) {
      storedIds.push(certificateKey)
      window.localStorage.setItem(CERTIFICATE_CELEBRATED_IDS_KEY, JSON.stringify(storedIds))
    }
  } catch (error) {
    console.warn("Failed to persist certificate celebration state", error)
  }
}

async function shareFeaturedCertificate() {
  const cert = featuredCertificate.value
  if (!cert?.pdfUrl) return

  const shareTitle = cert.name || t.value.certificatesPage.title
  const shareText = certificateTexts.value.shareText

  if (navigator.share) {
    try {
      await navigator.share({
        title: shareTitle,
        text: shareText,
        url: cert.pdfUrl,
      })
      return
    } catch (error) {
      if ((error as DOMException)?.name === "AbortError") return
    }
  }

  const linkedInUrl = `https://www.linkedin.com/sharing/share-offsite/?url=${encodeURIComponent(cert.pdfUrl)}`
  window.open(linkedInUrl, "_blank", "noopener,noreferrer")
}

onMounted(async () => {
  loading.value = true
  try {
    const res = await apiClient("/api/certificates")
    if (res?.certificates) {
      certificates.value = res.certificates
        .map((cert: any) => ({
          id: cert.cred_id || cert.catalog_id,
          credGuid: cert.cred_guid || "",
          createdAt: cert.created_at || "",
          createdAtMs: cert.created_at ? new Date(cert.created_at).getTime() : 0,
          validUntil: cert.valid_until || "",
          validUntilMs: cert.valid_until ? new Date(cert.valid_until).getTime() : 0,
          name: cert.name,
          description: cert.description || "",
          issueDate: cert.created_at ? formatBackendDateOnly(cert.created_at) : t.value.common.na,
          expiryDate: cert.valid_until ? formatBackendDateOnly(cert.valid_until) : t.value.common.permanent,
          credentialId: cert.cred_guid || cert.cred_id || t.value.common.na,
          source: cert.source || "",
          pdfUrl:
            cert.files?.find(
              (f: any) => f.file_type === 2 || f.file_ext === ".pdf" || f.file_ext === "pdf" || f.file_name?.endsWith(".pdf"),
            )?.view_url || "",
        }))
        .sort((a: any, b: any) => {
          if (b.createdAtMs !== a.createdAtMs) return b.createdAtMs - a.createdAtMs
          return b.validUntilMs - a.validUntilMs
        })

      if (certificates.value.length) {
        try {
          const latestCertificateKey = getCelebrationCertificateKey(certificates.value[0])
          const storedIds = JSON.parse(window.localStorage.getItem(CERTIFICATE_CELEBRATED_IDS_KEY) || "[]") as string[]
          celebrationVisible.value = !!latestCertificateKey && !storedIds.includes(latestCertificateKey)
        } catch (error) {
          console.warn("Failed to read certificate celebration state", error)
          celebrationVisible.value = true
        }
      }
    }
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <AppShell content-class="p-0">
    <div
      v-if="celebrationVisible && featuredCertificate"
      class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 px-4 py-6 backdrop-blur-sm"
      @click.self="closeCelebrationModal"
    >
      <div class="relative w-full max-w-[560px] overflow-hidden rounded-[20px] bg-white shadow-[0_24px_60px_rgba(16,30,67,0.28)]">
        <button
          class="absolute right-4 top-4 z-20 flex h-10 w-10 items-center justify-center rounded-full border border-slate-200 bg-white/90 text-slate-500 transition hover:border-primary/25 hover:text-primary"
          @click="closeCelebrationModal"
        >
          <X class="h-5 w-5" />
        </button>

        <div class="relative overflow-hidden px-6 pb-6 pt-7 text-center md:px-8 md:pb-8">
          <div class="pointer-events-none absolute inset-x-0 top-0 h-28 bg-[radial-gradient(circle_at_top,rgba(16,30,67,0.14),transparent_70%)]" />
          <div class="pointer-events-none absolute inset-x-0 top-10 flex justify-center opacity-95">
            <img :src="rewGif" alt="" class="h-[240px] w-[240px] object-contain md:h-[300px] md:w-[300px]" />
          </div>

          <div class="relative z-10 mt-2">
            <span class="inline-flex items-center gap-1 rounded-full bg-primary/8 px-3 py-1 text-sm font-semibold text-primary">
              <Sparkles class="h-4 w-4" />
              {{ certificateTexts.celebrationModalTitle }}
            </span>

            <h2 class="mt-4 text-3xl font-bold tracking-tight text-foreground md:text-4xl">
              {{ certificateTexts.celebrationModalHeadline }}
            </h2>
            <p class="mx-auto mt-4 max-w-[420px] text-sm leading-7 text-muted-foreground md:text-base">
              {{ certificateTexts.celebrationModalDesc }}
            </p>

            <div class="mt-6 rounded-[18px] bg-[#f7fbfc] px-4 py-4 shadow-inner shadow-primary/5">
              <p class="text-xs uppercase tracking-[0.22em] text-primary/70">{{ featuredCertificate.name }}</p>
              <p class="mt-2 text-sm text-muted-foreground">
                {{ t.certificatesPage.certificateId }}: <span class="font-mono text-card-foreground">{{ featuredCertificate.credentialId }}</span>
              </p>
            </div>

            <div class="mt-6 grid gap-3 sm:grid-cols-2">
              <button
                class="certificate-modal-primary inline-flex items-center justify-center gap-2 rounded-[14px] px-5 py-3 text-sm font-semibold text-white shadow-[0_12px_24px_rgba(16,30,67,0.24)] disabled:cursor-not-allowed disabled:opacity-50"
                :disabled="!featuredCertificate.pdfUrl"
                @click="downloadFeaturedCertificate"
              >
                <Download class="h-4 w-4" />
                {{ certificateTexts.celebrationModalDownload }}
              </button>
              <button
                class="inline-flex items-center justify-center gap-2 rounded-[14px] border border-primary/18 bg-primary/[0.04] px-5 py-3 text-sm font-semibold text-primary transition hover:bg-primary/[0.08] disabled:cursor-not-allowed disabled:border-slate-200 disabled:bg-slate-100 disabled:text-slate-400"
                disabled
                @click="shareFeaturedCertificate"
              >
                <Share2 class="h-4 w-4" />
                {{ certificateTexts.celebrationModalShare }}
              </button>
            </div>

            <button class="mt-4 text-sm text-muted-foreground transition hover:text-foreground" @click="closeCelebrationModal">
              {{ certificateTexts.celebrationModalDismiss }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <Award class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ t.certificatesPage.title }}</span>
      </header>

      <main class="px-5 py-8 md:px-8 lg:px-10">
        <div class="mb-6">
          <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.certificatesPage.title }}</h1>
          <p class="mt-2 text-muted-foreground">{{ t.certificatesPage.subtitle }}</p>
        </div>

    <div v-if="loading" class="flex items-center justify-center gap-2 rounded-[16px] bg-white py-16 text-muted-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <Loader2 class="h-5 w-5 animate-spin" />
      <span>{{ t.common.loading }}</span>
    </div>
    <div v-else-if="certificates.length" class="grid gap-4 lg:grid-cols-2">
      <div
        v-for="cert in certificates"
        :key="cert.id"
        class="group relative overflow-hidden rounded-[16px] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all hover:-translate-y-0.5 hover:border-primary/25 hover:shadow-md hover:shadow-primary/10"
      >
        <div class="relative bg-[linear-gradient(135deg,rgb(11,31,69)_0%,rgb(27,69,141)_55%,rgb(58,111,192)_100%)] p-4 text-white">
          <div class="relative flex items-start justify-between">
            <div>
              <span class="badge mb-3 border-0 bg-white/20 text-white"><CheckCircle2 class="mr-1 h-3 w-3" /> {{ t.certificatesPage.active }}</span>
              <span v-if="cert.source === 'application'" class="badge mb-3 ml-2 border-0 bg-white/20 text-white">{{ t.certificatesPage.sourceApplication || 'Application' }}</span>
              <span v-else-if="cert.source === 'pdf_cert'" class="badge mb-3 ml-2 border-0 bg-white/20 text-white">{{ t.certificatesPage.sourceSystem || 'System Issued' }}</span>
              <h3 class="mb-1 text-xl font-bold">{{ cert.name }}</h3>
              <p class="text-sm text-white/80">{{ cert.description }}</p>
            </div>
            <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-white/20 backdrop-blur-sm"><Award class="h-6 w-6" /></div>
          </div>
        </div>
        <div class="p-4">
          <div class="mb-4 grid grid-cols-2 gap-3">
            <div class="rounded-[14px] bg-[#eef3f8] p-3">
              <p class="mb-1 text-xs text-muted-foreground">{{ t.certificatesPage.issueDate }}</p>
              <p class="flex items-center gap-1.5 font-medium text-card-foreground"><Calendar class="h-4 w-4 text-muted-foreground" /> {{ cert.issueDate }}</p>
            </div>
            <div class="rounded-[14px] bg-[#eef3f8] p-3">
              <p class="mb-1 text-xs text-muted-foreground">{{ t.certificatesPage.expiryDate }}</p>
              <p class="flex items-center gap-1.5 font-medium text-card-foreground"><Calendar class="h-4 w-4 text-muted-foreground" /> {{ cert.expiryDate }}</p>
            </div>
          </div>
          <div class="mb-4 flex items-center gap-2 px-1 text-sm text-muted-foreground">
            <span class="text-base leading-none text-muted-foreground">#</span>
            <span>{{ t.certificatesPage.certificateId }}</span>
            <span class="font-mono text-xs font-semibold text-card-foreground">{{ cert.credentialId }}</span>
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
    </div>

    <div v-else class="flex min-h-[320px] flex-col items-center justify-center rounded-[16px] bg-white p-6 text-center shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10">
        <Award class="h-8 w-8 text-primary" />
      </div>
      <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.certificatesPage.keepLearningTitle }}</h3>
      <p class="mb-5 max-w-md text-sm leading-6 text-muted-foreground">{{ t.certificatesPage.keepLearningDesc }}</p>
      <RouterLink to="/certifications" class="btn btn-primary rounded-lg shadow-sm shadow-primary/20">
        <ExternalLink class="h-4 w-4" />
        {{ t.certificatesPage.browseCourses }}
      </RouterLink>
    </div>
      </main>
    </div>
  </AppShell>
</template>

<style scoped>
.certificate-modal-primary {
  position: relative;
  overflow: hidden;
  isolation: isolate;
  background:
    radial-gradient(circle at 18% 20%, rgba(255, 255, 255, 0.24), transparent 22%),
    linear-gradient(120deg, rgba(16, 30, 67, 1) 0%, rgba(24, 46, 96, 1) 42%, rgba(39, 88, 182, 1) 58%, rgba(16, 30, 67, 1) 100%);
}

.certificate-modal-primary::before {
  content: "";
  position: absolute;
  inset: 0;
  z-index: -1;
  background: linear-gradient(120deg, transparent 12%, rgba(255, 255, 255, 0.42) 36%, rgba(255, 255, 255, 0.16) 48%, transparent 68%);
  transform: translateX(-140%);
  animation: certificateShine 2.8s ease-in-out infinite;
}

.certificate-modal-primary::after {
  content: "";
  position: absolute;
  inset: 1px;
  border-radius: inherit;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.14), transparent 32%);
  pointer-events: none;
}

@keyframes certificateShine {
  0%,
  22% {
    transform: translateX(-140%);
  }

  44%,
  100% {
    transform: translateX(140%);
  }
}
</style>
