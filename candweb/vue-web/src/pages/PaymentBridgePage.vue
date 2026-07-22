<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useRoute, useRouter } from "vue-router"
import { ArrowLeft, Wallet } from "lucide-vue-next"
import PaymentSessionPanel from "@/components/PaymentSessionPanel.vue"
import { clearPendingPaymentSession, readPendingPaymentSession, sanitizePaymentReturnPath, type PendingPaymentSession } from "@/lib/payment"
import { useTranslation } from "@/lib/language"

const route = useRoute()
const router = useRouter()
const { t } = useTranslation()
const session = ref<PendingPaymentSession>({})
const ready = ref(false)

const copy = computed(() => t.value.paymentBridge)

const orderLabel = computed(() => session.value.orderId || session.value.bizRefUlid || session.value.bizType || copy.value.loading)

function hydrateSession() {
  const stored = readPendingPaymentSession()
  clearPendingPaymentSession()
  const returnPath = sanitizePaymentReturnPath(route.query.returnPath) || sanitizePaymentReturnPath(stored?.returnPath)
  session.value = {
    paymentKey: String(route.query.paymentKey || stored?.paymentKey || "").trim(),
    bizType: String(route.query.bizType || stored?.bizType || "").trim(),
    bizRefUlid: String(route.query.bizRefUlid || stored?.bizRefUlid || "").trim(),
    orderId: String(route.query.orderId || stored?.orderId || stored?.bizRefUlid || "").trim(),
    source: String(route.query.source || stored?.source || "").trim(),
    returnPath,
  }
  ready.value = true
}

function goBack() {
  if (session.value.returnPath) {
    router.push(session.value.returnPath)
    return
  }
  if (window.history.length > 1) router.back()
  else router.push("/orders")
}

onMounted(hydrateSession)
</script>

<template>
  <main class="flex min-h-screen items-center justify-center bg-[#eef8f7] px-4 py-10 text-slate-900">
    <section class="w-full max-w-6xl overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-[0_18px_55px_rgba(15,74,82,0.12)]">
      <div class="flex items-center justify-between gap-4 border-b border-slate-200 px-6 py-5">
        <div class="flex items-center gap-3">
          <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-emerald-50 text-emerald-500">
            <Wallet class="h-6 w-6" />
          </div>
          <div>
            <h1 class="text-2xl font-bold tracking-tight">{{ copy.title }}</h1>
            <p class="mt-1 text-sm text-slate-500">{{ orderLabel }}</p>
          </div>
        </div>
        <button class="inline-flex items-center gap-2 rounded-xl border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-700 transition-colors hover:bg-slate-50" @click="goBack">
          <ArrowLeft class="h-4 w-4" />
          {{ copy.back }}
        </button>
      </div>

      <div class="p-6">
        <div v-if="!ready" class="flex min-h-[64vh] items-center justify-center text-sm text-slate-500">
          {{ copy.loading }}
        </div>
        <PaymentSessionPanel
          v-else
          :payment-key="session.paymentKey"
          :biz-type="session.bizType"
          :biz-ref-ulid="session.bizRefUlid"
          :order-id="session.orderId"
          :source="session.source"
          :return-path="session.returnPath"
          min-height-class="min-h-[64vh]"
        />
      </div>
    </section>
  </main>
</template>
