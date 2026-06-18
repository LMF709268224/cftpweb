<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch, nextTick } from "vue"
import { toast } from "vue-sonner"
import { AlertCircle, Building2, CheckCircle2, CreditCard, Lock, Loader2, ShoppingCart } from "lucide-vue-next"
import { timelineStatusLabelWithDiagnostics, timelineStatusBadgeClassForStatus } from "@/lib/status-labels"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"

declare global {
  interface Window {
    Stripe: any;
  }
}

type PaymentMethod = "stripe" | "bank"
type MallAction = "purchase" | "unlock"

type EligibilityBlocker = {
  blocker_type?: string
  description?: string
  details?: unknown[]
}

type EligibilityPreview = {
  eligible?: boolean
  can_purchase?: boolean
  can_unlock?: boolean
  blockers?: EligibilityBlocker[]
}

type PaymentPreview = {
  subtotal?: number
  discount_total?: number
  tax_total?: number
  total?: number
  currency?: string
  amount_label?: string
  amount?: string | number
  pay_amount_label?: string
  pay_amount?: string | number
}

type ActiveOrder = {
  action: MallAction
  orderId: string
  status?: string
  payOrderId?: string
  message?: string
}

const props = defineProps<{
  open: boolean
  courseName: string
  description?: string
  pipelineId: string
}>()

const emit = defineEmits<{ "update:open": [value: boolean] }>()
const { t } = useTranslation()
const paymentMethod = ref<PaymentMethod>("stripe")
const eligibilityLoading = ref(false)
const actionLoading = ref(false)
const paymentLoading = ref(false)
const eligibility = ref<EligibilityPreview | null>(null)
const activeOrder = ref<ActiveOrder | null>(null)
const paymentPreview = ref<PaymentPreview | null>(null)
const previewError = ref("")
const embeddedClientSecret = ref("")
let stripeCheckoutInstance: any = null
let stripeCheckoutMountToken = 0

const selectedExemptionsJson = JSON.stringify({ stages: [] })
const copy = computed(() => t.value.purchaseDialog || {})
const blockers = computed(() => eligibility.value?.blockers || [])
const canPurchase = computed(() => Boolean(eligibility.value?.can_purchase))
const canUnlock = computed(() => Boolean(eligibility.value?.can_unlock))
const cannotContinue = computed(() => Boolean(eligibility.value && !canPurchase.value && !canUnlock.value))
const hasInProgressOrder = computed(() => blockers.value.some((blocker) => blocker.blocker_type === "IN_PROGRESS_PURCHASE"))

watch(() => props.open, (open) => {
  if (open && props.pipelineId) {
    void loadEligibility()
  } else {
    destroyStripeCheckout(true)
  }
})

function close() {
  destroyStripeCheckout(true)
  emit("update:open", false)
}

onBeforeUnmount(() => {
  destroyStripeCheckout(true)
})

function clearCheckoutContainer() {
  const container = document.getElementById("checkout")
  if (container) container.innerHTML = ""
}

function destroyStripeCheckout(clearClientSecret = false) {
  stripeCheckoutMountToken += 1
  const checkout = stripeCheckoutInstance
  stripeCheckoutInstance = null
  if (checkout) {
    try {
      if (typeof checkout.destroy === "function") {
        checkout.destroy()
      } else if (typeof checkout.unmount === "function") {
        checkout.unmount()
      }
    } catch (error) {
      console.error("Failed to destroy Stripe embedded checkout", error)
    }
  }
  clearCheckoutContainer()
  if (clearClientSecret) embeddedClientSecret.value = ""
}

function normalizedStatus(status: unknown) {
  return String(status || "").trim().toUpperCase()
}

function isCompletedStatus(status: unknown) {
  return normalizedStatus(status).includes("COMPLETED")
}

function isFailedStatus(status: unknown) {
  const value = normalizedStatus(status)
  return value.includes("FAILED") || value.includes("CANCEL") || value.includes("REJECT")
}

function stripeCheckoutUrl(paymentKey: unknown) {
  if (typeof paymentKey !== "string") return ""
  const value = paymentKey.trim()
  if (!value) return ""
  if (/^https:\/\/checkout\.stripe\.com\//i.test(value)) return value
  if (value.startsWith("/c/pay/")) return `https://checkout.stripe.com${value}`
  return ""
}

function stripeEmbeddedClientSecret(paymentKey: unknown) {
  if (typeof paymentKey !== "string") return ""
  const value = paymentKey.trim()
  return value.startsWith("cs_") ? value : ""
}

function formatMoney(amount?: number, currency = "usd") {
  if (typeof amount !== "number") return "-"
  return new Intl.NumberFormat(undefined, { style: "currency", currency: currency || "usd" }).format(amount / 100)
}

function detailText(detail: unknown) {
  if (typeof detail === "string") return detail
  if (detail && typeof detail === "object") {
    const record = detail as Record<string, unknown>
    return String(record.name || record.title || record.label || record.description || "")
  }
  return String(detail || "")
}

function blockerTitle(blocker: EligibilityBlocker) {
  if (blocker.blocker_type === "MISSING_UNLOCK_QUALIFICATION") return copy.value.missingQualification
  if (blocker.blocker_type === "ALREADY_PURCHASED") return copy.value.alreadyPurchased
  if (blocker.blocker_type === "IN_PROGRESS_PURCHASE") return copy.value.inProgressPurchase
  if (blocker.blocker_type === "PIPELINE_NOT_FOUND") return copy.value.pipelineNotFound
  return blocker.description || blocker.blocker_type || copy.value.unknownBlocker || t.value.common.unknown
}

function orderIdFromDetail(order: any) {
  return order?.pipeline_order_ulid || order?.summary?.pipeline_order_ulid || ""
}

function orderStatusFromDetail(order: any) {
  return order?.order_status || order?.summary?.order_status || ""
}

async function loadEligibility() {
  eligibilityLoading.value = true
  activeOrder.value = null
  paymentPreview.value = null
  previewError.value = ""
  destroyStripeCheckout(true)
  try {
    const res: EligibilityPreview = await apiClient(`/api/mall/pipelines/${props.pipelineId}/eligibility`)
    eligibility.value = res
    if (res.blockers?.some((blocker) => blocker.blocker_type === "IN_PROGRESS_PURCHASE")) {
      await loadActiveOrder()
    }
  } finally {
    eligibilityLoading.value = false
  }
}

async function previewPayment(action: MallAction, orderId: string) {
  previewError.value = ""
  const bizType = action === "unlock" ? "PIPELINE_UNLOCK" : "PIPELINE_PAYMENT"
  try {
    paymentPreview.value = await apiClient("/api/mall/payments/preview", {
      method: "POST",
      body: JSON.stringify({ biz_type: bizType, biz_ref_ulid: orderId, coupon_codes: [] }),
    })
  } catch {
    paymentPreview.value = null
    previewError.value = copy.value.pricePreviewFailed || t.value.common.error
  }
}

async function loadActiveOrder() {
  previewError.value = ""
  paymentPreview.value = null
  try {
    const order = await apiClient(`/api/mall/pipelines/${props.pipelineId}/active-order`)
    const orderId = orderIdFromDetail(order)
    if (!orderId) return
    activeOrder.value = {
      action: "purchase",
      orderId,
      status: orderStatusFromDetail(order),
      payOrderId: order.pipeline_pay_order_ulid,
      message: copy.value.inProgressPurchaseDesc,
    }
    await previewPayment("purchase", orderId)
  } catch (error) {
    console.error(error)
  }
}

async function createPurchaseOrder() {
  actionLoading.value = true
  try {
    const latest: EligibilityPreview = await apiClient(`/api/mall/pipelines/${props.pipelineId}/eligibility`)
    eligibility.value = latest
    if (!latest.can_purchase) return

    const order = await apiClient(`/api/mall/pipelines/${props.pipelineId}/purchase`, {
      method: "POST",
      body: JSON.stringify({
        payment_mode: "FULL_PIPELINE",
        candidate_selected_exemptions_json: selectedExemptionsJson,
      }),
    })
    const orderId = order.pipeline_order_ulid
    const orderStatus = order.order_status
    activeOrder.value = {
      action: "purchase",
      orderId,
      status: orderStatus,
      payOrderId: order.pipeline_pay_order_ulid,
      message: order.message,
    }
    if (isCompletedStatus(orderStatus)) {
      toast.success(copy.value.purchaseCompleted)
      close()
      window.setTimeout(() => window.location.reload(), 800)
      return
    }
    if (isFailedStatus(orderStatus)) {
      toast.error(copy.value.purchaseFailed)
      return
    }
    if (orderId) await previewPayment("purchase", orderId)
  } catch (error) {
    console.error(error)
  } finally {
    actionLoading.value = false
  }
}

async function createUnlockOrder() {
  actionLoading.value = true
  try {
    const latest: EligibilityPreview = await apiClient(`/api/mall/pipelines/${props.pipelineId}/eligibility`)
    eligibility.value = latest
    if (!latest.can_unlock) return

    const order = await apiClient(`/api/mall/pipelines/${props.pipelineId}/unlock`, { method: "POST" })
    const orderId = order.pipeline_unlock_order_ulid
    const paymentKey = order.payment_key
    const orderStatus = order.order_status
    activeOrder.value = {
      action: "unlock",
      orderId,
      status: orderStatus,
      payOrderId: order.pay_order_ulid,
      message: order.message,
    }
    if (isCompletedStatus(orderStatus)) {
      toast.success(copy.value.unlockCompleted)
      await loadEligibility()
      return
    }
    if (isFailedStatus(orderStatus)) {
      toast.error(copy.value.unlockFailed)
      return
    }
    if (orderId && (paymentKey || order.pay_order_ulid || normalizedStatus(orderStatus).includes("PAYMENT"))) {
      await previewPayment("unlock", orderId)
    } else {
      toast.info(copy.value.refreshEligibility)
    }
  } catch (error) {
    console.error(error)
  } finally {
    actionLoading.value = false
  }
}

function rememberPendingMallPayment() {
  if (!activeOrder.value?.orderId) return
  localStorage.setItem("pending_mall_payment", JSON.stringify({
    action: activeOrder.value.action,
    orderId: activeOrder.value.orderId,
    pipelineId: props.pipelineId,
  }))
}

async function mountStripeCheckout(clientSecret: string) {
  destroyStripeCheckout(false)
  const mountToken = stripeCheckoutMountToken
  try {
    const configRes = await apiClient("/api/public/config")
    const pk = configRes.stripe_publishable_key
    if (!pk) {
      toast.error(copy.value.stripePublishableKeyMissing || "Missing Stripe publishable key")
      return
    }

    await nextTick()
    const stripe = window.Stripe(pk)
    const checkout = await stripe.initEmbeddedCheckout({
      fetchClientSecret: async () => clientSecret
    })
    if (mountToken !== stripeCheckoutMountToken) {
      if (typeof checkout.destroy === "function") checkout.destroy()
      return
    }
    stripeCheckoutInstance = checkout
    checkout.mount("#checkout")
  } catch (err: any) {
    console.error(err)
    toast.error(err.message || String(err))
  }
}

async function initiatePayment() {
  if (!activeOrder.value?.orderId) return
  const bizType = activeOrder.value.action === "unlock" ? "PIPELINE_UNLOCK" : "PIPELINE_PAYMENT"
  paymentLoading.value = true
  try {
    destroyStripeCheckout(true)
    const origin = window.location.origin
    const successParams = new URLSearchParams({
      payment_status: "success",
      payment_action: activeOrder.value.action,
      order_id: activeOrder.value.orderId,
      pipeline_id: props.pipelineId,
    })
    const cancelParams = new URLSearchParams({
      payment_status: "cancelled",
      payment_action: activeOrder.value.action,
      order_id: activeOrder.value.orderId,
      pipeline_id: props.pipelineId,
    })
    const res = await apiClient("/api/mall/payments/initiate", {
      method: "POST",
      body: JSON.stringify({
        biz_type: bizType,
        biz_ref_ulid: activeOrder.value.orderId,
        success_url: `${origin}/certifications?${successParams.toString()}`,
        cancel_url: `${origin}/certifications?${cancelParams.toString()}`,
        coupon_codes: [],
      }),
    })
    const paymentKey = res.payment_key
    if (!paymentKey) {
      toast.error(copy.value.paymentSessionFailed)
      return
    }
    const checkoutUrl = stripeCheckoutUrl(paymentKey)
    if (paymentMethod.value === "stripe" && checkoutUrl) {
      rememberPendingMallPayment()
      window.location.href = checkoutUrl
      return
    }
    const clientSecret = stripeEmbeddedClientSecret(paymentKey)
    if (paymentMethod.value === "stripe" && clientSecret) {
      rememberPendingMallPayment()
      embeddedClientSecret.value = clientSecret
      mountStripeCheckout(clientSecret)
      return
    }
    toast.error(copy.value.unsupportedPaymentKey)
  } catch (error) {
    console.error(error)
  } finally {
    paymentLoading.value = false
  }
}
</script>

<template>
  <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="close">
    <div class="flex max-h-[86vh] w-full max-w-[620px] flex-col overflow-hidden rounded-xl bg-card shadow-2xl">
      <div class="shrink-0 border-b border-border px-6 pb-4 pt-6">
        <h2 class="text-xl font-semibold">{{ copy.title }}: {{ courseName }}</h2>
        <p v-if="description" class="mt-2 text-sm leading-6 text-muted-foreground">{{ description }}</p>
      </div>

      <div class="min-h-0 flex-1 space-y-5 overflow-y-auto px-6 py-5">
        <div class="flex items-center justify-between border-b border-border/50 py-2">
          <span class="text-sm text-muted-foreground">{{ t.common.purchaseDialogCourse }}</span>
          <span class="text-sm font-medium text-foreground">{{ courseName }}</span>
        </div>

        <div v-if="eligibilityLoading && !eligibility" class="rounded-lg border border-border bg-muted/30 p-4">
          <div class="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 class="h-4 w-4 animate-spin" />
            {{ copy.checking }}
          </div>
        </div>
        <div v-else-if="canPurchase" class="rounded-lg border border-emerald-200 bg-emerald-50 p-4">
          <div class="flex items-center gap-2 font-semibold text-emerald-900"><CheckCircle2 class="h-4 w-4" />{{ copy.canPurchaseTitle }}</div>
          <p class="mt-2 text-sm text-emerald-800">{{ copy.canPurchaseDesc }}</p>
        </div>
        <div v-else-if="canUnlock" class="rounded-lg border border-blue-200 bg-blue-50 p-4">
          <div class="flex items-center gap-2 font-semibold text-blue-900"><Lock class="h-4 w-4" />{{ copy.canUnlockTitle }}</div>
          <p class="mt-2 text-sm text-blue-800">{{ copy.canUnlockDesc }}</p>
        </div>
        <div v-else-if="cannotContinue && hasInProgressOrder" class="rounded-lg border border-blue-200 bg-blue-50 p-4">
          <div class="flex items-center gap-2 font-semibold text-blue-900"><CreditCard class="h-4 w-4" />{{ copy.inProgressPurchase }}</div>
          <p class="mt-2 text-sm text-blue-800">{{ copy.inProgressPurchaseDesc }}</p>
        </div>
        <div v-else-if="cannotContinue" class="rounded-lg border border-amber-200 bg-amber-50 p-4">
          <div class="flex items-center gap-2 font-semibold text-amber-900"><AlertCircle class="h-4 w-4" />{{ copy.blockedTitle }}</div>
          <p class="mt-2 text-sm text-amber-800">{{ copy.blockedDesc }}</p>
        </div>

        <div v-if="blockers.length > 0" class="rounded-lg border border-amber-200 bg-amber-50/70 p-4">
          <div class="mb-3 text-sm font-semibold text-amber-950">{{ copy.blockersTitle }}</div>
          <ul class="space-y-2">
            <li v-for="(blocker, index) in blockers" :key="`${blocker.blocker_type || 'blocker'}-${index}`" class="rounded-lg border border-amber-200 bg-white/80 p-3">
              <div class="font-medium text-amber-950">{{ blockerTitle(blocker) }}</div>
              <div v-if="Array.isArray(blocker.details) && blocker.details.map(detailText).filter(Boolean).length > 0" class="mt-2">
                <div class="mb-1 text-xs font-medium text-muted-foreground">{{ copy.requiredItems }}</div>
                <ul class="space-y-1">
                  <li v-for="(detail, detailIndex) in blocker.details.map(detailText).filter(Boolean)" :key="`${detail}-${detailIndex}`" class="flex items-center gap-2 rounded-md bg-amber-100/70 px-2 py-1.5 text-sm font-medium text-amber-950">
                    <AlertCircle class="h-3.5 w-3.5 shrink-0 text-amber-600" />
                    <span>{{ detail }}</span>
                  </li>
                </ul>
              </div>
            </li>
          </ul>
        </div>

        <div v-if="activeOrder" class="rounded-lg border border-border bg-muted/30 p-4">
          <div class="mb-2 flex items-center justify-between gap-3">
            <div class="text-sm font-semibold text-foreground">{{ activeOrder.message === copy.inProgressPurchaseDesc ? copy.activeOrder : copy.orderCreated }}</div>
            <span v-if="activeOrder.status" class="badge text-xs" :class="timelineStatusBadgeClassForStatus('MALL_ORDER', activeOrder.status)">
              {{ timelineStatusLabelWithDiagnostics(t, 'MALL_ORDER', activeOrder.status) }}
            </span>
          </div>
          <div class="break-all text-xs text-muted-foreground">{{ activeOrder.orderId }}</div>
          <p v-if="activeOrder.message" class="mt-2 text-sm text-muted-foreground">{{ activeOrder.message }}</p>
        </div>

        <div v-if="paymentPreview" class="rounded-lg border border-border bg-muted/30 p-4">
          <div class="mb-3 text-sm font-semibold text-foreground">{{ copy.pricePreviewTitle }}</div>
          <div class="space-y-2 text-sm">
            <div class="flex justify-between">
              <span class="text-muted-foreground">{{ copy.subtotal }}</span>
              <span class="font-medium">{{ paymentPreview.amount_label || formatMoney(paymentPreview.subtotal, paymentPreview.currency) }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-muted-foreground">{{ copy.discount }}</span>
              <span class="font-medium">-{{ formatMoney(paymentPreview.discount_total || 0, paymentPreview.currency) }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-muted-foreground">{{ copy.tax }}</span>
              <span class="font-medium">{{ formatMoney(paymentPreview.tax_total || 0, paymentPreview.currency) }}</span>
            </div>
            <div class="mt-2 flex justify-between border-t border-border pt-2">
              <span class="font-semibold text-foreground">{{ copy.total }}</span>
              <span class="text-lg font-bold text-foreground">{{ paymentPreview.pay_amount_label || formatMoney(paymentPreview.total, paymentPreview.currency) }}</span>
            </div>
          </div>
        </div>

        <div v-if="activeOrder && previewError" class="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
          <div class="flex items-center gap-2 font-semibold"><AlertCircle class="h-4 w-4" />{{ copy.pricePreviewTitle }}</div>
          <p class="mt-2">{{ previewError }}</p>
        </div>

        <div v-if="embeddedClientSecret" class="space-y-3">
          <div class="rounded-lg border border-blue-200 bg-blue-50 p-4 text-sm text-blue-900">
            <div class="flex items-center gap-2 font-semibold"><CreditCard class="h-4 w-4" />{{ copy.embeddedCheckoutTitle }}</div>
            <p class="mt-2">{{ copy.embeddedCheckoutDesc }}</p>
          </div>
          <div class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-800">
            <strong>⚠️ 测试环境提示：</strong> 当前为测试环境，请使用通用测试信用卡号 <code>4242 4242 4242 4242</code>，任意有效日期和CVV进行体验。
          </div>
          <div class="rounded-lg border bg-white p-4 text-sm text-muted-foreground min-h-[400px]">
            <div id="checkout"></div>
          </div>
        </div>

        <div v-if="activeOrder && paymentPreview && !embeddedClientSecret" class="space-y-3">
          <label class="text-sm font-medium text-foreground">{{ t.common.purchaseDialogPaymentMethod }}</label>
          <div class="space-y-2">
            <button
              type="button"
              :class="[
                'flex w-full items-center gap-3 rounded-lg border p-3 transition-all',
                paymentMethod === 'stripe' ? 'border-primary bg-primary/5' : 'border-border hover:border-primary/50',
              ]"
              @click="paymentMethod = 'stripe'"
            >
              <div :class="['flex h-5 w-5 items-center justify-center rounded-full border-2 transition-colors', paymentMethod === 'stripe' ? 'border-primary' : 'border-muted-foreground/30']">
                <div v-if="paymentMethod === 'stripe'" class="h-2.5 w-2.5 rounded-full bg-primary" />
              </div>
              <CreditCard class="h-4 w-4 text-primary" />
              <span class="text-sm font-medium text-foreground">{{ copy.stripe }}</span>
              <span class="badge ml-auto border-0 bg-amber-500/10 text-xs text-amber-700">{{ t.common.purchaseDialogStripeBadge }}</span>
            </button>
            <button
              type="button"
              :class="[
                'flex w-full items-center gap-3 rounded-lg border p-3 transition-all',
                paymentMethod === 'bank' ? 'border-primary bg-primary/5' : 'border-border hover:border-primary/50',
              ]"
              @click="paymentMethod = 'bank'"
            >
              <div :class="['flex h-5 w-5 items-center justify-center rounded-full border-2 transition-colors', paymentMethod === 'bank' ? 'border-primary' : 'border-muted-foreground/30']">
                <div v-if="paymentMethod === 'bank'" class="h-2.5 w-2.5 rounded-full bg-primary" />
              </div>
              <Building2 class="h-4 w-4 text-muted-foreground" />
              <span class="text-sm font-medium text-foreground">{{ copy.bank }}</span>
            </button>
          </div>
          <div v-if="paymentMethod === 'stripe'" class="mt-4 rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-800">
            <strong>⚠️ 测试环境提示：</strong> 当前为测试环境，请使用通用测试信用卡号 <code>4242 4242 4242 4242</code>，任意有效日期和CVV进行体验。
          </div>
        </div>
      </div>

      <div class="shrink-0 flex items-center justify-end gap-3 border-t border-border bg-muted/30 px-6 py-4">
        <button class="btn btn-outline" @click="close">{{ t.common.cancel }}</button>
        <button v-if="cannotContinue" class="btn btn-outline" :disabled="eligibilityLoading" @click="loadEligibility">
          <Loader2 v-if="eligibilityLoading" class="h-4 w-4 animate-spin" />
          {{ copy.refreshEligibility }}
        </button>
        <button v-if="canUnlock && !activeOrder" class="btn btn-primary" :disabled="actionLoading" @click="createUnlockOrder">
          <Loader2 v-if="actionLoading" class="h-4 w-4 animate-spin" />
          <Lock v-else class="h-4 w-4" />
          {{ copy.createUnlockOrder }}
        </button>
        <button v-if="canPurchase && !activeOrder" class="btn btn-primary" :disabled="actionLoading" @click="createPurchaseOrder">
          <Loader2 v-if="actionLoading" class="h-4 w-4 animate-spin" />
          <ShoppingCart v-else class="h-4 w-4" />
          {{ copy.createPurchaseOrder }}
        </button>
        <button v-if="activeOrder && previewError" class="btn btn-outline" :disabled="actionLoading" @click="previewPayment(activeOrder.action, activeOrder.orderId)">
          {{ copy.retryPreview }}
        </button>
        <button v-if="activeOrder && paymentPreview && !embeddedClientSecret" class="btn btn-primary" :disabled="paymentLoading" @click="initiatePayment">
          <Loader2 v-if="paymentLoading" class="h-4 w-4 animate-spin" />
          <CreditCard v-else class="h-4 w-4" />
          {{ copy.payNow }}
        </button>
      </div>
    </div>
  </div>
</template>
