<template>
  <div class="space-y-4">
    <!-- Price Summary -->
    <div v-if="paymentPreview" class="rounded-lg bg-muted/30 p-4 border border-border">
      <div class="mb-3 text-sm font-semibold">{{ t.checkoutWizard.priceSummary }}</div>
      <div class="space-y-2 text-sm">
        <div class="flex justify-between">
          <span class="text-muted-foreground">{{ t.checkoutWizard.subtotal }}</span>
          <span class="font-medium">{{ paymentPreview.amount_label || formatMoney(paymentPreview.subtotal, paymentPreview.currency) }}</span>
        </div>
        <template v-if="paymentPreview.breakdown && paymentPreview.breakdown.length > 0">
          <div v-for="(item, idx) in paymentPreview.breakdown" :key="idx" class="flex justify-between text-emerald-600 text-sm">
            <span class="text-emerald-600/80">{{ getDiscountLabel(item) }}</span>
            <span class="font-medium">-{{ formatMoney(item.discount, paymentPreview.currency) }}</span>
          </div>
        </template>
        <template v-else-if="paymentPreview.discount_total">
          <div class="flex justify-between text-emerald-600 text-sm">
            <span class="text-emerald-600/80">{{ t.checkoutWizard.discount }}</span>
            <span class="font-medium">-{{ formatMoney(paymentPreview.discount_total, paymentPreview.currency) }}</span>
          </div>
        </template>
        <div class="mt-2 flex justify-between border-t border-border pt-2">
          <span class="font-semibold text-foreground">{{ t.checkoutWizard.total }}</span>
          <span class="text-lg font-bold text-foreground">{{ paymentPreview.pay_amount_label || formatMoney(paymentPreview.total, paymentPreview.currency) }}</span>
        </div>
      </div>
    </div>

    <!-- Coupon Input -->
    <CouponInputBlock
      v-model="couponInput"
      :active-coupon-codes="activeCouponCodes"
      :loading="couponPreviewLoading"
      :error="couponError"
      :cannot-pay-reason="cannotPayReason"
      @apply="applyCouponCodes"
      @clear="clearCouponCodes"
    />

    <!-- Stripe Embedded Checkout -->
    <PaymentSessionPanel
      v-if="bizRefUlid && paymentPreview && !couponPreviewLoading && !couponError && !hasInvalidCouponCodes"
      :biz-type="bizType"
      :biz-ref-ulid="bizRefUlid"
      :order-id="orderId"
      :source="source"
      :return-path="returnPath"
      :extra-return-params="extraReturnParams"
      :redirect-on-complete="redirectOnComplete"
      :auto-start="autoStart"
      :min-height-class="minHeightClass"
      :coupon-codes="activeCouponCodes"
      @complete="emit('complete')"
      @error="emit('error', $event)"
      @status-change="emit('status-change', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"
import CouponInputBlock from "@/components/CouponInputBlock.vue"
import PaymentSessionPanel from "@/components/PaymentSessionPanel.vue"

const props = withDefaults(defineProps<{
  bizType: string
  bizRefUlid: string
  orderId?: string
  source?: string
  returnPath?: string
  extraReturnParams?: Record<string, any>
  redirectOnComplete?: boolean
  autoStart?: boolean
  minHeightClass?: string
  initialPaymentPreview?: any
  initialCouponCodes?: string[]
}>(), {
  redirectOnComplete: true,
  autoStart: true,
  minHeightClass: "min-h-[60vh]",
  initialCouponCodes: () => [],
})

const emit = defineEmits<{
  "status-change": [status: "loading" | "redirecting" | "embedded" | "error"]
  error: [message: string]
  complete: []
}>()

const { t } = useTranslation()

const paymentPreview = ref<any>(props.initialPaymentPreview || null)
const couponInput = ref("")
const activeCouponCodes = ref<string[]>([...props.initialCouponCodes])
const couponPreviewLoading = ref(false)
const couponError = ref("")
let paymentPreviewRequestToken = 0

const hasInvalidCouponCodes = computed(() => Boolean(paymentPreview.value?.invalid?.length))
const cannotPayReason = computed(() => hasInvalidCouponCodes.value ? t.value.purchaseDialog.couponInvalidPaymentBlocked : "")

function formatMoney(amount?: number, currency = "usd") {
  if (typeof amount !== "number") return "-"
  return new Intl.NumberFormat(undefined, { style: "currency", currency: currency || "usd" }).format(amount / 100)
}

function getDiscountLabel(item: any) {
  let label = item.name || item.description || item.code || t.value.checkoutWizard.discount
  // Remove backend's unformatted minor units from label if present (e.g. ": -75000 usd")
  return label.replace(/:\s*-\d+(\.\d+)?\s*[a-zA-Z]*$/i, '')
}

function normalizeCouponCodes(codes: string[]) {
  return Array.from(new Set(codes.map((c) => String(c || "").trim()).filter(Boolean)))
}

function couponInputCodes() {
  return normalizeCouponCodes(couponInput.value.split(/[\s,;，；]+/))
}

function sameCouponCodes(left: string[], right: string[]) {
  return left.length === right.length && left.every((code, index) => code === right[index])
}

function invalidCouponMessage(invalid: any[]) {
  const details = invalid
    .map((item) => {
      const code = String(item?.code || "").trim()
      const reason = String(item?.reason || "").trim()
      if (code && reason) return `${code}: ${reason}`
      return code || reason
    })
    .filter(Boolean)
    .join("; ")
  return details || t.value.purchaseDialog.couponInvalidPaymentBlocked
}

async function refreshPaymentPreviewWithCoupons(codes: string[], commitCodes = false) {
  if (!props.bizType || !props.bizRefUlid) return false
  const normalizedCodes = normalizeCouponCodes(codes)
  const requestToken = ++paymentPreviewRequestToken
  couponPreviewLoading.value = true
  couponError.value = ""
  try {
    const res = await apiClient("/api/mall/payments/preview", {
      method: "POST",
      body: JSON.stringify({
        biz_type: props.bizType,
        biz_ref_ulid: props.bizRefUlid,
        promo_codes: normalizedCodes,
        coupon_codes: [],
      }),
      suppressErrorToast: true,
    })
    if (requestToken !== paymentPreviewRequestToken) return false
    const invalid = Array.isArray(res?.invalid) ? res.invalid : []
    if (invalid.length > 0) {
      couponError.value = invalidCouponMessage(invalid)
      return false
    }
    paymentPreview.value = res
    if (commitCodes && !sameCouponCodes(activeCouponCodes.value, normalizedCodes)) {
      activeCouponCodes.value = normalizedCodes
    }
    return true
  } catch (error) {
    if (requestToken !== paymentPreviewRequestToken) return false
    console.error("Failed to fetch payment preview:", error)
    couponError.value = t.value.purchaseDialog.couponPreviewFailed
    return false
  } finally {
    if (requestToken === paymentPreviewRequestToken) {
      couponPreviewLoading.value = false
    }
  }
}

async function applyCouponCodes() {
  const nextCodes = couponInputCodes()
  await refreshPaymentPreviewWithCoupons(nextCodes, true)
}

async function clearCouponCodes() {
  couponInput.value = ""
  await refreshPaymentPreviewWithCoupons([], true)
}

watch(() => [props.bizType, props.bizRefUlid], () => {
  if (props.bizType && props.bizRefUlid) {
    void refreshPaymentPreviewWithCoupons(activeCouponCodes.value)
  }
})

onMounted(() => {
  if (!paymentPreview.value && props.bizType && props.bizRefUlid) {
    void refreshPaymentPreviewWithCoupons(activeCouponCodes.value)
  }
})

onBeforeUnmount(() => {
  paymentPreviewRequestToken += 1
})
</script>
