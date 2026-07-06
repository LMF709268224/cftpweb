<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { AlertTriangle, ExternalLink, Loader2 } from "lucide-vue-next"
import { apiClient } from "@/lib/apiClient"
import { stripeCheckoutUrl, stripeEmbeddedClientSecret } from "@/lib/payment"
import { useTranslation } from "@/lib/language"

declare global {
  interface Window {
    Stripe: any
  }
}

const props = withDefaults(defineProps<{
  paymentKey?: string
  bizType?: string
  bizRefUlid?: string
  orderId?: string
  source?: string
  returnPath?: string
  extraReturnParams?: Record<string, string | number | boolean | null | undefined>
  couponCodes?: string[]
  autoStart?: boolean
  minHeightClass?: string
}>(), {
  couponCodes: () => [],
  autoStart: true,
  minHeightClass: "min-h-[60vh]",
})

const emit = defineEmits<{
  "status-change": [status: "loading" | "redirecting" | "embedded" | "error"]
  error: [message: string]
}>()

const { t } = useTranslation()
const status = ref<"loading" | "redirecting" | "embedded" | "error">("loading")
const errorMessage = ref("")
const checkoutUrl = ref("")
const clientSecret = ref("")
const embeddedLoading = ref(false)
const showStripeConnectionHint = ref(false)
const checkoutContainerId = `stripe-checkout-${Math.random().toString(36).slice(2)}`
let stripeCheckoutInstance: any = null
let stripeCheckoutMountToken = 0

const copy = computed(() => t.value.paymentSession)

function setStatus(nextStatus: typeof status.value) {
  status.value = nextStatus
  emit("status-change", nextStatus)
}

function clearCheckoutContainer() {
  const container = document.getElementById(checkoutContainerId)
  if (container) container.innerHTML = ""
}

function destroyStripeCheckout(clearSecret = false) {
  stripeCheckoutMountToken += 1
  showStripeConnectionHint.value = false
  const checkout = stripeCheckoutInstance
  stripeCheckoutInstance = null
  if (checkout) {
    try {
      if (typeof checkout.destroy === "function") checkout.destroy()
      else if (typeof checkout.unmount === "function") checkout.unmount()
    } catch (error) {
      console.error("Failed to destroy Stripe embedded checkout", error)
    }
  }
  clearCheckoutContainer()
  if (clearSecret) {
    clientSecret.value = ""
    embeddedLoading.value = false
  }
}

function paymentReturnUrl(paymentStatus: "success" | "cancelled") {
  const returnUrl = new URL(props.returnPath || "/orders", window.location.origin)
  returnUrl.searchParams.set("payment_status", paymentStatus)
  returnUrl.searchParams.set("payment_action", props.source || "payment")
  returnUrl.searchParams.set("order_id", props.orderId || props.bizRefUlid || "")
  Object.entries(props.extraReturnParams || {}).forEach(([key, value]) => {
    if (value !== null && value !== undefined && String(value).trim() !== "") {
      returnUrl.searchParams.set(key, String(value))
    }
  })
  return returnUrl.toString()
}

function fail(message?: string) {
  const finalMessage = message || copy.value.failed
  errorMessage.value = finalMessage
  setStatus("error")
  emit("error", finalMessage)
}

async function mountStripeCheckout(secret: string) {
  destroyStripeCheckout(false)
  embeddedLoading.value = true
  showStripeConnectionHint.value = false
  const mountToken = stripeCheckoutMountToken
  let mounted = false
  try {
    const configRes = await apiClient("/api/public/config")
    const pk = configRes.stripe_publishable_key
    if (!pk) throw new Error("Missing Stripe publishable key")

    await nextTick()
    const stripe = window.Stripe(pk)
    const checkout = await stripe.initEmbeddedCheckout({
      fetchClientSecret: async () => secret,
    })
    if (mountToken !== stripeCheckoutMountToken) {
      if (typeof checkout.destroy === "function") checkout.destroy()
      return
    }
    stripeCheckoutInstance = checkout
    checkout.mount(`#${checkoutContainerId}`)
    mounted = true
    window.setTimeout(() => {
      if (mountToken === stripeCheckoutMountToken) embeddedLoading.value = false
    }, 500)
    window.setTimeout(() => {
      if (mountToken === stripeCheckoutMountToken && status.value === "embedded") showStripeConnectionHint.value = true
    }, 2500)
  } catch (error: any) {
    console.error(error)
    fail(copy.value.stripeConnectionFailed)
  } finally {
    if (!mounted && mountToken === stripeCheckoutMountToken) embeddedLoading.value = false
  }
}

async function startPayment() {
  destroyStripeCheckout(true)
  errorMessage.value = ""
  checkoutUrl.value = ""
  clientSecret.value = ""
  showStripeConnectionHint.value = false
  setStatus("loading")

  const paymentKey = String(props.paymentKey || "").trim()
  const bizType = String(props.bizType || "").trim()
  const bizRefUlid = String(props.bizRefUlid || "").trim()

  if (!paymentKey && (!bizType || !bizRefUlid)) {
    fail(copy.value.missing)
    return
  }

  const hosted = stripeCheckoutUrl(paymentKey)
  if (hosted) {
    checkoutUrl.value = hosted
    setStatus("redirecting")
    window.setTimeout(() => window.location.assign(hosted), 600)
    return
  }

  const secret = stripeEmbeddedClientSecret(paymentKey)
  if (secret) {
    clientSecret.value = secret
    setStatus("embedded")
    await mountStripeCheckout(secret)
    return
  }

  try {
    const res = await apiClient("/api/mall/payments/initiate", {
      method: "POST",
      body: JSON.stringify({
        biz_type: bizType,
        biz_ref_ulid: bizRefUlid,
        success_url: paymentReturnUrl("success"),
        cancel_url: paymentReturnUrl("cancelled"),
        promo_codes: props.couponCodes.map((code) => String(code || "").trim()).filter(Boolean),
      }),
    })
    const nextKey = String(res?.payment_key || "").trim()
    if (!nextKey) throw new Error("payment_key is empty")

    const nextHosted = stripeCheckoutUrl(nextKey)
    if (nextHosted) {
      checkoutUrl.value = nextHosted
      setStatus("redirecting")
      window.setTimeout(() => window.location.assign(nextHosted), 600)
      return
    }

    const nextSecret = stripeEmbeddedClientSecret(nextKey)
    if (nextSecret) {
      clientSecret.value = nextSecret
      setStatus("embedded")
      await mountStripeCheckout(nextSecret)
      return
    }

    throw new Error("unsupported payment key")
  } catch (error: any) {
    console.error(error)
    fail(copy.value.failed)
  }
}

watch(
  () => [props.paymentKey, props.bizType, props.bizRefUlid, props.orderId, props.returnPath, props.source, props.extraReturnParams, props.couponCodes],
  () => {
    if (props.autoStart) void startPayment()
  },
)

onMounted(() => {
  if (props.autoStart) void startPayment()
})

onBeforeUnmount(() => destroyStripeCheckout(true))
</script>

<template>
  <div>
    <div v-if="status === 'loading'" :class="['flex flex-col items-center justify-center gap-4 text-center text-slate-600', minHeightClass]">
      <Loader2 class="h-8 w-8 animate-spin text-emerald-500" />
      <p>{{ copy.loading }}</p>
    </div>

    <div v-else-if="status === 'redirecting'" :class="['flex flex-col items-center justify-center gap-4 text-center text-slate-600', minHeightClass]">
      <Loader2 class="h-8 w-8 animate-spin text-emerald-500" />
      <p>{{ copy.redirecting }}</p>
      <a
        v-if="checkoutUrl"
        :href="checkoutUrl"
        class="inline-flex items-center gap-2 rounded-xl bg-emerald-500 px-4 py-2 text-sm font-semibold text-white hover:bg-emerald-600"
        rel="noopener noreferrer"
      >
        <ExternalLink class="h-4 w-4" />
        {{ copy.openNow }}
      </a>
    </div>

    <div v-else-if="status === 'embedded'" class="space-y-4">
      <div v-if="embeddedLoading" class="flex items-center gap-2 rounded-xl border border-emerald-100 bg-emerald-50 px-4 py-3 text-sm text-emerald-700">
        <Loader2 class="h-4 w-4 animate-spin" />
        {{ copy.loadingCheckout }}
      </div>

      <div :id="checkoutContainerId" :class="['rounded-2xl border border-slate-200 bg-white', minHeightClass]" />
    </div>

    <div v-else :class="['flex flex-col items-center justify-center gap-4 text-center text-slate-600', minHeightClass]">
      <div class="flex h-16 w-16 items-center justify-center rounded-2xl bg-rose-50 text-rose-500">
        <AlertTriangle class="h-8 w-8" />
      </div>
      <p class="max-w-lg text-sm leading-6 text-slate-600">{{ errorMessage || copy.failed }}</p>
      <button class="inline-flex items-center gap-2 rounded-xl bg-emerald-500 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-emerald-600" @click="startPayment">
        <Loader2 class="h-4 w-4" />
        {{ copy.retry }}
      </button>
    </div>
  </div>
</template>
