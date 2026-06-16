<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue"
import { useRoute, useRouter } from "vue-router"
import { AlertTriangle, ArrowLeft, ExternalLink, Loader2, Receipt } from "lucide-vue-next"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"

const route = useRoute()
const router = useRouter()
const { lang } = useTranslation()

const orderId = computed(() => String(route.query.orderId || ""))
const invoiceUrl = ref("")
const status = ref<"loading" | "redirecting" | "error">("loading")
const errorMessage = ref("")
let redirectTimer: number | undefined

const copy = computed(() =>
  lang.value === "zh"
    ? {
        title: "正在打开发票",
        loading: "正在获取发票链接，请稍候...",
        redirecting: "已获取发票链接，正在打开 Stripe 发票页面...",
        hint: "外部页面加载可能需要一点时间，请不要关闭此页面。",
        direct: "立即打开",
        retry: "重试",
        back: "返回订单",
        missing: "缺少订单编号，无法打开发票。",
        failed: "获取发票失败，请稍后重试。",
      }
    : {
        title: "Opening invoice",
        loading: "Fetching the invoice link. Please wait...",
        redirecting: "Invoice link is ready. Opening the Stripe invoice page...",
        hint: "The external page may take a moment to load. Please keep this tab open.",
        direct: "Open now",
        retry: "Retry",
        back: "Back to orders",
        missing: "Missing order ID. Unable to open invoice.",
        failed: "Failed to fetch the invoice. Please try again later.",
      },
)

function clearRedirectTimer() {
  if (redirectTimer) {
    window.clearTimeout(redirectTimer)
    redirectTimer = undefined
  }
}

function openInvoiceNow() {
  if (!invoiceUrl.value) return
  window.location.replace(invoiceUrl.value)
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
        <button
          v-if="status === 'redirecting'"
          class="inline-flex items-center gap-2 rounded-xl bg-emerald-500 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-emerald-600"
          @click="openInvoiceNow"
        >
          <ExternalLink class="h-4 w-4" />
          {{ copy.direct }}
        </button>
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
