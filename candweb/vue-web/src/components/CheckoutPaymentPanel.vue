<template>
  <div class="space-y-4">
    <!-- Price Summary -->
    <div v-if="paymentPreview" class="rounded-lg bg-muted/30 p-4 border border-border">
      <div class="mb-3 text-sm font-semibold">{{ t.checkoutWizard?.priceSummary || 'Price Summary' }}</div>
      <div class="space-y-2 text-sm">
        <div class="flex justify-between">
          <span class="text-muted-foreground">{{ t.checkoutWizard?.subtotal || 'Subtotal' }}</span>
          <span class="font-medium">{{ paymentPreview.amount_label || formatMoney(paymentPreview.subtotal, paymentPreview.currency) }}</span>
        </div>
        <div v-if="paymentPreview.discount_total" class="flex justify-between text-emerald-600">
          <span class="text-emerald-600/80">{{ t.checkoutWizard?.discount || 'Discount' }}</span>
          <span class="font-medium">-{{ formatMoney(paymentPreview.discount_total, paymentPreview.currency) }}</span>
        </div>
        <div class="mt-2 flex justify-between border-t border-border pt-2">
          <span class="font-semibold text-foreground">{{ t.checkoutWizard?.total || 'Total' }}</span>
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
      v-if="bizRefUlid"
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
import { computed, onMounted, ref, watch } from "vue"
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

const hasInvalidCouponCodes = computed(() => Boolean(paymentPreview.value?.invalid?.length))
const cannotPayReason = computed(() => hasInvalidCouponCodes.value ? (t.value.purchaseDialog?.couponInvalidPaymentBlocked || "Invalid coupon. Cannot proceed.") : "")

function formatMoney(amount?: number, currency = "usd") {
  if (typeof amount !== "number") return "-"
  return new Intl.NumberFormat(undefined, { style: "currency", currency: currency || "usd" }).format(amount / 100)
}

function normalizeCouponCodes(codes: string[]) {
  return Array.from(new Set(codes.map((c) => String(c || "").trim()).filter(Boolean)))
}

function couponInputCodes() {
  return normalizeCouponCodes(couponInput.value.split(/[\s,;，；]+/))
}

async function refreshPaymentPreviewWithCoupons(codes: string[]) {
  if (!props.bizType || !props.bizRefUlid) return
  couponPreviewLoading.value = true
  couponError.value = ""
  try {
    const res = await apiClient("/api/mall/payments/preview", {
      method: "POST",
      body: JSON.stringify({
        biz_type: props.bizType,
        biz_ref_ulid: props.bizRefUlid,
        promo_codes: normalizeCouponCodes(codes),
        coupon_codes: [],
      }),
      suppressErrorToast: true,
    })
    paymentPreview.value = res
  } catch (error: any) {
    console.error("Failed to fetch payment preview:", error)
    couponError.value = error.message || t.value.common?.error || "Error fetching preview"
  } finally {
    couponPreviewLoading.value = false
  }
}

async function applyCouponCodes() {
  const nextCodes = couponInputCodes()
  activeCouponCodes.value = nextCodes
  await refreshPaymentPreviewWithCoupons(nextCodes)
}

async function clearCouponCodes() {
  couponInput.value = ""
  activeCouponCodes.value = []
  await refreshPaymentPreviewWithCoupons([])
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
</script>
