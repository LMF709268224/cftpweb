<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue"
import { useRoute, useRouter } from "vue-router"
import { AlertTriangle, ArrowLeft, ExternalLink, Loader2, Receipt } from "lucide-vue-next"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"

const route = useRoute()
const router = useRouter()
const { t } = useTranslation()

const orderId = computed(() => String(route.query.orderId || ""))
const invoiceUrl = ref("")
const status = ref<"loading" | "redirecting" | "error">("loading")
const errorMessage = ref("")
let redirectTimer: number | undefined

const copy = computed(() => t.value.invoiceRedirect)

function clearRedirectTimer() {
  if (redirectTimer) {
    window.clearTimeout(redirectTimer)
    redirectTimer = undefined
  }
}

function openInvoiceNow() {
  if (!invoiceUrl.value) return
  window.location.assign(invoiceUrl.value)
}

async function loadInvoice() {
  clearRedirectTimer()
  invoiceUrl.value = ""
  errorMessage.value = ""

  if (!orderId.value) {
    status.value = "error"
    errorMessage.value = copy.value.missing
    return
  }

  status.value = "loading"
  try {
    const res = await apiClient(`/api/invoices/${encodeURIComponent(orderId.value)}`)
    if (!res?.invoice_url) throw new Error("invoice_url is empty")

    invoiceUrl.value = res.invoice_url
    status.value = "redirecting"
    redirectTimer = window.setTimeout(openInvoiceNow, 900)
  } catch (err) {
    console.error("Failed to open invoice:", err)
    status.value = "error"
    errorMessage.value = copy.value.failed
  }
}

function goBack() {
  if (window.history.length > 1) router.back()
  else router.push("/orders")
}

onMounted(loadInvoice)
onBeforeUnmount(clearRedirectTimer)
</script>

<template>
  <main class="flex min-h-screen items-center justify-center bg-[#eef8f7] px-4 py-10 text-slate-900">
    <section class="w-full max-w-lg rounded-3xl border border-slate-200 bg-white p-8 text-center shadow-[0_18px_55px_rgba(15,74,82,0.12)]">
      <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-2xl bg-emerald-50 text-emerald-500">
        <Receipt v-if="status !== 'error'" class="h-8 w-8" />
        <AlertTriangle v-else class="h-8 w-8 text-rose-500" />
      </div>

      <h1 class="mt-5 text-2xl font-bold tracking-tight">{{ copy.title }}</h1>
      <p class="mt-3 text-sm leading-6 text-slate-500">
        <template v-if="status === 'loading'">{{ copy.loading }}</template>
        <template v-else-if="status === 'redirecting'">{{ copy.redirecting }}</template>
        <template v-else>{{ errorMessage || copy.failed }}</template>
      </p>

      <div v-if="status !== 'error'" class="mt-6 flex flex-col items-center gap-3">
        <Loader2 class="h-7 w-7 animate-spin text-emerald-500" />
        <p class="text-xs leading-5 text-slate-500">{{ copy.hint }}</p>
      </div>

      <div class="mt-7 flex flex-wrap justify-center gap-3">
        <a
          v-if="status === 'redirecting' && invoiceUrl"
          :href="invoiceUrl"
          class="inline-flex items-center gap-2 rounded-xl bg-emerald-500 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-emerald-600"
          rel="noopener noreferrer"
        >
          <ExternalLink class="h-4 w-4" />
          {{ copy.direct }}
        </a>
        <button
          v-if="status === 'error'"
          class="inline-flex items-center gap-2 rounded-xl bg-emerald-500 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-emerald-600"
          @click="loadInvoice"
        >
          <Loader2 class="h-4 w-4" />
          {{ copy.retry }}
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-700 transition-colors hover:bg-slate-50" @click="goBack">
          <ArrowLeft class="h-4 w-4" />
          {{ copy.back }}
        </button>
      </div>
    </section>
  </main>
</template>
