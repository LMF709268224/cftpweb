<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { AlertCircle, CheckCircle2, CreditCard, Lock, Loader2 } from "lucide-vue-next"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"

type EligibilityBlocker = {
  blocker_type?: string
  description?: string
  details?: unknown[]
}

type EligibilityPreview = {
  can_purchase?: boolean
  can_unlock?: boolean
  blockers?: EligibilityBlocker[]
}

type ActiveOrder = {
  action: "purchase" | "unlock"
  orderId: string
  status?: string
  message?: string
}

const props = defineProps<{
  open: boolean
  courseName: string
  pipelineId: string
}>()

const emit = defineEmits<{ "update:open": [value: boolean] }>()
const { t } = useTranslation()
const eligibility = ref<EligibilityPreview | null>(null)
const eligibilityLoading = ref(false)
const actionLoading = ref(false)
const paymentLoading = ref(false)
const activeOrder = ref<ActiveOrder | null>(null)
const paymentPreview = ref<any>(null)
const previewError = ref("")

const copy = computed(() => t.value.purchaseDialog || {})
const blockers = computed(() => eligibility.value?.blockers || [])
const canPurchase = computed(() => Boolean(eligibility.value?.can_purchase))
const canUnlock = computed(() => Boolean(eligibility.value?.can_unlock))
const hasInProgressOrder = computed(() => blockers.value.some((blocker) => blocker.blocker_type === "IN_PROGRESS_PURCHASE"))

watch(() => props.open, (open) => {
  if (open && props.pipelineId) void loadEligibility()
})

function close() {
  emit("update:open", false)
}

function normalizedStatus(status?: string) {
  return String(status || "").toUpperCase()
}

function isCompletedStatus(status?: string) {
  const s = normalizedStatus(status)
  return s.includes("COMPLETED") || s.includes("SUCCESS") || s.includes("PAID")
}

function isFailedStatus(status?: string) {
  const s = normalizedStatus(status)
  return s.includes("FAILED") || s.includes("CANCEL")
}

function blockerTitle(blocker: EligibilityBlocker) {
  if (blocker.blocker_type === "MISSING_UNLOCK_QUALIFICATION") return copy.value.missingQualification
  if (blocker.blocker_type === "ALREADY_PURCHASED") return copy.value.alreadyPurchased
  if (blocker.blocker_type === "IN_PROGRESS_PURCHASE") return copy.value.inProgressPurchase
  if (blocker.blocker_type === "PIPELINE_NOT_FOUND") return copy.value.pipelineNotFound
  return blocker.description || blocker.blocker_type || copy.value.unknownBlocker || t.value.common.unknown
}

function detailText(detail: any) {
  if (!detail) return ""
  if (typeof detail === "string") return detail
  return detail.name || detail.title || detail.description || detail.required_entity_id || ""
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

async function previewPayment(action: "purchase" | "unlock", orderId: string) {
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
  try {
    const order = await apiClient(`/api/mall/pipelines/${props.pipelineId}/active-order`)
    const orderId = orderIdFromDetail(order)
    if (!orderId) return
    activeOrder.value = {
      action: "purchase",
      orderId,
      status: orderStatusFromDetail(order),
      message: copy.value.inProgressPurchaseDesc,
    }
    await previewPayment("purchase", orderId)
  } catch {
    // No active order is a normal eligibility state.
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
      body: JSON.stringify({ payment_mode: "FULL_PIPELINE", candidate_selected_exemptions_json: "" }),
    })
    const orderId = order.pipeline_order_ulid
    const orderStatus = order.order_status
    activeOrder.value = { action: "purchase", orderId, status: orderStatus, message: order.message }
    if (isCompletedStatus(orderStatus)) {
      toast.success(copy.value.purchaseCompleted || t.value.common.purchaseSuccess)
      close()
      window.setTimeout(() => window.location.reload(), 800)
      return
    }
    if (isFailedStatus(orderStatus)) {
      toast.error(copy.value.purchaseFailed || t.value.common.error)
      return
    }
    if (orderId) await previewPayment("purchase", orderId)
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
    const orderStatus = order.order_status
    activeOrder.value = { action: "unlock", orderId, status: orderStatus, message: order.message }
    if (isCompletedStatus(orderStatus)) {
      toast.success(copy.value.unlockCompleted || t.value.common.success)
      await loadEligibility()
      return
    }
    if (isFailedStatus(orderStatus)) {
      toast.error(copy.value.unlockFailed || t.value.common.error)
      return
    }
    if (orderId) await previewPayment("unlock", orderId)
  } finally {
    actionLoading.value = false
  }
}

function stripeCheckoutUrl(paymentKey?: string) {
  if (!paymentKey) return ""
  if (/^https?:\/\//.test(paymentKey)) return paymentKey
  if (paymentKey.startsWith("cs_")) return `https://checkout.stripe.com/c/pay/${paymentKey}`
  return ""
}

async function initiatePayment() {
  if (!activeOrder.value?.orderId) return
  const bizType = activeOrder.value.action === "unlock" ? "PIPELINE_UNLOCK" : "PIPELINE_PAYMENT"
  paymentLoading.value = true
  try {
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
        success_url: `${origin}/courses?${successParams.toString()}`,
        cancel_url: `${origin}/courses?${cancelParams.toString()}`,
        coupon_codes: [],
      }),
    })
    const checkoutUrl = stripeCheckoutUrl(res.payment_key)
    if (checkoutUrl) {
      localStorage.setItem("pending_mall_payment", JSON.stringify({ action: activeOrder.value.action, orderId: activeOrder.value.orderId, pipelineId: props.pipelineId }))
      window.location.href = checkoutUrl
      return
    }
    toast.error(copy.value.unsupportedPaymentKey || copy.value.paymentSessionFailed || t.value.common.error)
  } finally {
    paymentLoading.value = false
  }
}
</script>

<template>
  <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="close">
    <div class="max-h-[86vh] w-full max-w-2xl overflow-hidden rounded-2xl bg-card shadow-2xl">
      <div class="flex items-center justify-between border-b border-border px-6 py-4">
        <h2 class="text-xl font-semibold">{{ copy.title || t.common.purchaseDialogTitle }}: {{ courseName }}</h2>
        <button class="rounded-lg px-2 py-1 text-muted-foreground hover:bg-muted" @click="close">x</button>
      </div>

      <div class="max-h-[70vh] space-y-5 overflow-y-auto px-6 py-5">
        <div class="flex justify-between border-b border-border/50 py-2">
          <span class="text-sm text-muted-foreground">{{ t.common.purchaseDialogCourse }}</span>
          <span class="text-sm font-medium">{{ courseName }}</span>
        </div>

        <div v-if="eligibilityLoading && !eligibility" class="rounded-xl border border-border bg-muted/30 p-4 text-sm text-muted-foreground">
          <Loader2 class="mr-2 inline h-4 w-4 animate-spin" /> {{ copy.checking || t.common.loading }}
        </div>
        <div v-else-if="canPurchase" class="rounded-xl border border-emerald-200 bg-emerald-50 p-4 text-emerald-900">
          <div class="flex items-center gap-2 font-semibold"><CheckCircle2 class="h-4 w-4" /> {{ copy.canPurchaseTitle || t.common.purchaseDialogTitle }}</div>
          <p class="mt-2 text-sm">{{ copy.canPurchaseDesc }}</p>
        </div>
        <div v-else-if="canUnlock" class="rounded-xl border border-blue-200 bg-blue-50 p-4 text-blue-900">
          <div class="flex items-center gap-2 font-semibold"><Lock class="h-4 w-4" /> {{ copy.canUnlockTitle }}</div>
          <p class="mt-2 text-sm">{{ copy.canUnlockDesc }}</p>
        </div>
        <div v-else-if="hasInProgressOrder" class="rounded-xl border border-blue-200 bg-blue-50 p-4 text-blue-900">
          <div class="flex items-center gap-2 font-semibold"><CreditCard class="h-4 w-4" /> {{ copy.inProgressPurchase }}</div>
          <p class="mt-2 text-sm">{{ copy.inProgressPurchaseDesc }}</p>
        </div>
        <div v-else class="rounded-xl border border-amber-200 bg-amber-50 p-4 text-amber-900">
          <div class="flex items-center gap-2 font-semibold"><AlertCircle class="h-4 w-4" /> {{ copy.blockedTitle || t.common.purchaseDialogIneligible }}</div>
          <p class="mt-2 text-sm">{{ copy.blockedDesc }}</p>
        </div>

        <div v-if="blockers.length > 0" class="rounded-xl border border-amber-200 bg-amber-50/70 p-4">
          <div class="mb-3 text-sm font-semibold text-amber-950">{{ copy.blockersTitle }}</div>
          <ul class="space-y-2">
            <li v-for="(blocker, index) in blockers" :key="`${blocker.blocker_type || 'blocker'}-${index}`" class="rounded-lg border border-amber-200 bg-white/80 p-3">
              <div class="font-medium text-amber-950">{{ blockerTitle(blocker) }}</div>
              <div v-if="Array.isArray(blocker.details) && blocker.details.length" class="mt-2 space-y-1">
                <div v-for="(detail, detailIndex) in blocker.details" :key="detailIndex" class="rounded-md bg-amber-100/70 px-2 py-1.5 text-sm font-medium text-amber-950">
                  {{ detailText(detail) }}
                </div>
              </div>
            </li>
          </ul>
        </div>

        <div v-if="activeOrder" class="rounded-xl border border-border bg-muted/30 p-4">
          <div class="mb-2 flex items-center justify-between gap-3">
            <div class="text-sm font-semibold">{{ activeOrder.message === copy.inProgressPurchaseDesc ? copy.activeOrder : copy.orderCreated }}</div>
            <span v-if="activeOrder.status" class="badge">{{ activeOrder.status }}</span>
          </div>
          <div class="break-all text-xs text-muted-foreground">{{ activeOrder.orderId }}</div>
          <p v-if="activeOrder.message" class="mt-2 text-sm text-muted-foreground">{{ activeOrder.message }}</p>
        </div>

        <div v-if="paymentPreview" class="rounded-xl border border-border bg-muted/30 p-4 text-sm">
          <div class="mb-3 font-semibold">{{ copy.pricePreviewTitle || t.common.purchaseDialogPrice }}</div>
          <div class="flex justify-between"><span class="text-muted-foreground">{{ copy.subtotal || t.common.purchaseDialogPrice }}</span><span>{{ paymentPreview.amount_label || paymentPreview.amount || "-" }}</span></div>
          <div v-if="paymentPreview.pay_amount_label || paymentPreview.pay_amount" class="mt-2 flex justify-between font-semibold">
            <span>{{ copy.total || t.common.purchaseDialogPrice }}</span><span>{{ paymentPreview.pay_amount_label || paymentPreview.pay_amount }}</span>
          </div>
        </div>
        <p v-if="previewError" class="text-sm text-destructive">{{ previewError }}</p>
      </div>

      <div class="flex justify-end gap-3 border-t border-border px-6 py-4">
        <button class="btn btn-outline" @click="close">{{ t.common.cancel }}</button>
        <button v-if="canPurchase && !activeOrder" class="btn btn-primary" :disabled="actionLoading" @click="createPurchaseOrder">
          <Loader2 v-if="actionLoading" class="h-4 w-4 animate-spin" /> {{ copy.createPurchaseOrder || t.common.purchaseDialogBankSubmit }}
        </button>
        <button v-if="canUnlock && !activeOrder" class="btn btn-primary" :disabled="actionLoading" @click="createUnlockOrder">
          <Loader2 v-if="actionLoading" class="h-4 w-4 animate-spin" /> {{ copy.createUnlockOrder || t.common.confirm }}
        </button>
        <button v-if="activeOrder" class="btn btn-primary" :disabled="paymentLoading" @click="initiatePayment">
          <Loader2 v-if="paymentLoading" class="h-4 w-4 animate-spin" /> {{ t.common.purchaseDialogStripeSubmit }}
        </button>
      </div>
    </div>
  </div>
</template>
