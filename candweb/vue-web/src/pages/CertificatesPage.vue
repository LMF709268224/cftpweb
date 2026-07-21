<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { RouterLink } from "vue-router"
import { Award, Calendar, CheckCircle2, ClipboardCheck, Download, Eye, Loader2, ShieldCheck, Sparkles, X } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import rewGif from "@/assets/rew.gif"
import { apiClient } from "@/lib/apiClient"
import { useBodyScrollLock } from "@/lib/bodyScrollLock"
import { formatBackendDateOnly } from "@/lib/utils"
import { useTranslation } from "@/lib/language"
import { usePolling } from "@/lib/polling"

const { t } = useTranslation()
const certificates = ref<any[]>([])
const loading = ref(false)
const celebrationVisible = ref(false)
const CERTIFICATE_PREVIEW_TIMEOUT_MS = 20000
const CERTIFICATE_CELEBRATED_IDS_KEY = "cftp-certificates-celebrated-ids"

const featuredCertificate = computed(() => certificates.value[0] ?? null)
useBodyScrollLock(() => celebrationVisible.value && Boolean(featuredCertificate.value))

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

const SOURCE_CONFIG: Record<string, { labelKey: "sourceApplication" | "sourceSystem"; fallback: string; icon: typeof ClipboardCheck | typeof ShieldCheck; cls: string; iconCls: string; accent: string }> = {
  application: {
    labelKey: "sourceApplication",
    fallback: "Application",
    icon: ClipboardCheck,
    cls: "border-amber-200 text-amber-800",
    iconCls: "bg-amber-100 text-amber-700",
    accent: "bg-amber-300",
  },
  pdf_cert: {
    labelKey: "sourceSystem",
    fallback: "System Issued",
    icon: ShieldCheck,
    cls: "border-cyan-200 text-cyan-800",
    iconCls: "bg-cyan-100 text-cyan-700",
    accent: "bg-cyan-300",
  },
}

function certificateSourceLabel(source?: string) {
  const cfg = SOURCE_CONFIG[source ?? ""]
  if (!cfg) return ""
  return (t.value.certificatesPage as any)[cfg.labelKey] || cfg.fallback
}
function certificateSourceIcon(source?: string) {
  return SOURCE_CONFIG[source ?? ""]?.icon ?? ShieldCheck
}
function certificateSourceClass(source?: string) {
  return SOURCE_CONFIG[source ?? ""]?.cls ?? "border-slate-200 text-slate-700"
}
function certificateSourceIconClass(source?: string) {
  return SOURCE_CONFIG[source ?? ""]?.iconCls ?? "bg-slate-100 text-slate-600"
}
function certificateSourceAccentClass(source?: string) {
  return SOURCE_CONFIG[source ?? ""]?.accent ?? "bg-white/20"
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

function normalizeCertificates(list: any[]) {
  return list
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
}

async function loadCertificates(showLoading = true, showCelebration = false, suppressErrorToast = false) {
  if (showLoading) loading.value = true
  try {
    const res = await apiClient("/api/certificates", { suppressErrorToast })
    if (res?.certificates) {
      certificates.value = normalizeCertificates(res.certificates)

      if (showCelebration && certificates.value.length) {
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
    if (showLoading) loading.value = false
  }
}

const certificatesPolling = usePolling(() => loadCertificates(false, false, true))

onMounted(async () => {
  await loadCertificates(true, true)
  certificatesPolling.start()
})
</script>

<template>
  <AppShell content-class="p-0">
    <div
      v-if="celebrationVisible && featuredCertificate"
      class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 px-4 py-6 backdrop-blur-sm"
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
              {{ t.certificatesPage.celebrationModalTitle }}
            </span>

            <h2 class="mt-4 text-3xl font-bold tracking-tight text-foreground md:text-4xl">
              {{ t.certificatesPage.celebrationModalHeadline }}
            </h2>
            <p class="mx-auto mt-4 max-w-[420px] text-sm leading-7 text-muted-foreground md:text-base">
              {{ t.certificatesPage.celebrationModalDesc }}
            </p>

            <div class="mt-6 rounded-[18px] border border-primary/15 bg-[linear-gradient(180deg,#f8fbff_0%,#edf6ff_100%)] px-5 py-4 text-left shadow-[inset_0_1px_0_rgba(255,255,255,0.9),0_10px_28px_rgba(16,30,67,0.08)]">
              <div class="flex items-start gap-3">
                <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-primary text-white shadow-[0_8px_18px_rgba(16,30,67,0.2)]">
                  <Award class="h-5 w-5" />
                </div>
                <div class="min-w-0 flex-1">
                  <p class="text-xs font-semibold uppercase tracking-[0.14em] text-primary/70">{{ t.certificatesPage.certificateName }}</p>
                  <p class="mt-1 break-words text-xl font-bold leading-snug text-slate-950 md:text-2xl">{{ featuredCertificate.name }}</p>
                  <p class="mt-2 flex flex-wrap items-center gap-2 text-sm font-medium text-slate-600">
                    <span class="inline-flex items-center gap-1.5">
                      <Calendar class="h-4 w-4 text-primary/70" />
                      {{ featuredCertificate.issueDate }}
                    </span>
                    <span class="text-slate-300">|</span>
                    <span>{{ featuredCertificate.expiryDate }}</span>
                  </p>
                </div>
              </div>
            </div>

            <div class="mt-6">
              <button
                class="certificate-modal-primary inline-flex w-full items-center justify-center gap-2 rounded-[14px] px-5 py-3 text-sm font-semibold text-white shadow-[0_12px_24px_rgba(16,30,67,0.24)] disabled:cursor-not-allowed disabled:opacity-50"
                :disabled="!featuredCertificate.pdfUrl"
                @click="downloadFeaturedCertificate"
              >
                <Download class="h-4 w-4" />
                {{ t.certificatesPage.celebrationModalDownload }}
              </button>
            </div>

            <button class="mt-4 text-sm text-muted-foreground transition hover:text-foreground" @click="closeCelebrationModal">
              {{ t.certificatesPage.celebrationModalDismiss }}
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
        class="certificate-card group relative overflow-hidden rounded-[16px] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all duration-300 ease-out hover:-translate-y-1 focus-within:ring-2 focus-within:ring-primary/20"
      >
        <span class="certificate-card-sheen pointer-events-none absolute left-0 top-0 z-20 h-1 w-full" />
        <span class="certificate-card-orb pointer-events-none absolute -right-12 -top-12 z-10 h-36 w-36 rounded-full opacity-0 transition-opacity duration-300 group-hover:opacity-100" />
        <div class="absolute inset-x-0 top-0 z-10 h-1" :class="certificateSourceAccentClass(cert.source)" />
        <div class="relative bg-[linear-gradient(135deg,rgb(11,31,69)_0%,rgb(27,69,141)_55%,rgb(58,111,192)_100%)] p-4 text-white">
          <div class="absolute inset-0 bg-[radial-gradient(circle_at_18%_12%,rgba(255,255,255,0.22),transparent_32%)] opacity-70" />
          <div class="relative flex items-start justify-between">
            <div class="min-w-0">
              <div class="mb-3 flex flex-wrap items-center gap-2">
                <span class="badge border-0 bg-white/20 text-white"><CheckCircle2 class="mr-1 h-3 w-3" /> {{ t.certificatesPage.active }}</span>
                <span
                  v-if="certificateSourceLabel(cert.source)"
                  :class="['inline-flex h-7 max-w-full items-center overflow-hidden rounded-full border bg-white text-xs font-semibold shadow-sm', certificateSourceClass(cert.source)]"
                >
                  <span :class="['flex h-full items-center px-2', certificateSourceIconClass(cert.source)]">
                    <component :is="certificateSourceIcon(cert.source)" class="h-3.5 w-3.5" />
                  </span>
                  <span class="flex min-w-0 items-center px-2.5">
                    <span class="truncate">{{ certificateSourceLabel(cert.source) }}</span>
                  </span>
                </span>
              </div>
              <h3 class="mb-1 text-xl font-bold">{{ cert.name }}</h3>
              <p class="text-sm text-white/80">{{ cert.description }}</p>
            </div>
            <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-white/20 ring-1 ring-white/15 backdrop-blur-sm transition-transform duration-300 group-hover:scale-105"><Award class="h-6 w-6" /></div>
          </div>
        </div>
        <div class="relative p-4">
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
          <div class="flex gap-3">
            <button class="btn btn-primary flex-1 rounded-lg shadow-sm shadow-primary/20 transition-all duration-300 group-hover:shadow-primary/30" :disabled="!cert.pdfUrl" @click="openCertificate(cert.pdfUrl)">
              <Download class="h-4 w-4" /> {{ cert.pdfUrl ? t.certificatesPage.downloadCertificate : t.certificatesPage.certificateGenerating }}
            </button>
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
        {{ t.certificatesPage.browseCourses }}
      </RouterLink>
    </div>
      </main>
    </div>
  </AppShell>
</template>

<style scoped>
.certificate-card {
  --certificate-card-accent: #38bdf8;
  --certificate-card-glow: rgba(37, 99, 235, 0.2);
}

.certificate-card:hover {
  box-shadow: 0 18px 34px -18px var(--certificate-card-glow), 0 12px 28px rgba(15, 23, 42, 0.1);
}

.certificate-card-sheen {
  background: linear-gradient(90deg, transparent, var(--certificate-card-accent), transparent);
  opacity: 0.78;
  transform: translateX(-105%);
  transition: transform 0.65s ease;
}

.certificate-card:hover .certificate-card-sheen {
  transform: translateX(105%);
}

.certificate-card-orb {
  background: radial-gradient(circle, rgba(56, 189, 248, 0.2), transparent 68%);
}

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
