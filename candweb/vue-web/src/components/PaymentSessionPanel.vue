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
  autoStart?: boolean
  minHeightClass?: string
}>(), {
  autoStart: true,
  minHeightClass: "min-h-[60vh]",
})

const emit = defineEmits<{
  "status-change": [status: "loading" | "redirecting" | "embedded" | "error"]
  error: [message: string]
}>()

const { lang } = useTranslation()
const status = ref<"loading" | "redirecting" | "embedded" | "error">("loading")
const errorMessage = ref("")
const checkoutUrl = ref("")
const clientSecret = ref("")
const embeddedLoading = ref(false)
const showStripeConnectionHint = ref(false)
const checkoutContainerId = `stripe-checkout-${Math.random().toString(36).slice(2)}`
let stripeCheckoutInstance: any = null
let stripeCheckoutMountToken = 0

const copy = computed(() =>
  lang.value === "zh"
    ? {
        loading: "\u6b63\u5728\u83b7\u53d6\u652f\u4ed8\u4fe1\u606f\uff0c\u8bf7\u7a0d\u5019...",
        redirecting: "\u5df2\u51c6\u5907\u597d\u652f\u4ed8\u94fe\u63a5\uff0c\u6b63\u5728\u6253\u5f00 Stripe \u652f\u4ed8\u9875\u9762...",
        loadingCheckout: "\u6b63\u5728\u52a0\u8f7d Stripe \u652f\u4ed8\u7ec4\u4ef6...",
        stripeConnectionHint: "如果下方显示 “Something went wrong”，通常是浏览器无法连接 Stripe 或证书校验失败。请检查网络、代理/VPN、HTTPS 抓包工具、公司网关或系统时间。",
        missing: "\u7f3a\u5c11\u652f\u4ed8\u4fe1\u606f\uff0c\u65e0\u6cd5\u7ee7\u7eed\u3002",
        failed: "\u6253\u5f00\u652f\u4ed8\u5931\u8d25\uff0c\u8bf7\u7a0d\u540e\u518d\u8bd5\u3002",
        stripeConnectionFailed: "无法连接 Stripe 支付组件。请检查网络、代理/VPN、HTTPS 抓包工具、公司网关、系统时间或证书信任设置后重试。",
        retry: "\u91cd\u8bd5",
        openNow: "\u7acb\u5373\u6253\u5f00",
      }
    : {
        loading: "Fetching payment session. Please wait...",
        redirecting: "Payment link is ready. Opening Stripe checkout...",
        loadingCheckout: "Loading Stripe checkout...",
        stripeConnectionHint: "If the area below shows “Something went wrong”, the browser usually cannot reach Stripe or cannot validate Stripe's certificate. Please check your network, proxy/VPN, HTTPS inspection tools, company gateway, or system time.",
        missing: "Missing payment information. Unable to continue.",
        failed: "Failed to open payment. Please try again later.",
        stripeConnectionFailed: "Unable to connect to Stripe checkout. Please check your network, proxy/VPN, HTTPS inspection tools, company gateway, system time, or certificate trust settings, then try again.",
        retry: "Retry",
        openNow: "Open now",
      },
)

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
        coupon_codes: [],
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
    fail(error?.message)
  }
}

watch(
  () => [props.paymentKey, props.bizType, props.bizRefUlid, props.orderId, props.returnPath, props.source, props.extraReturnParams],
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
      <div v-if="showStripeConnectionHint" class="flex items-start gap-2 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm leading-6 text-amber-800">
        <AlertTriangle class="mt-0.5 h-4 w-4 shrink-0" />
        <span>{{ copy.stripeConnectionHint }}</span>
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
