<script setup lang="ts">
import { computed } from "vue"
import { CreditCard, X } from "lucide-vue-next"
import PaymentSessionPanel from "@/components/PaymentSessionPanel.vue"
import { useTranslation } from "@/lib/language"

const props = defineProps<{
  open: boolean
  title: string
  subtitle?: string
  paymentKey?: string
  bizType?: string
  bizRefUlid?: string
  orderId?: string
  source?: string
  returnPath?: string
  extraReturnParams?: Record<string, string | number | boolean | null | undefined>
}>()

const emit = defineEmits<{ "update:open": [value: boolean] }>()
const { lang } = useTranslation()

const copy = computed(() =>
  lang.value === "zh"
    ? {
        paymentTitle: "完成支付",
        paymentDesc: "请在下方完成 Stripe 在线支付。支付完成后页面会自动返回并刷新状态。",
        testHint: "测试环境提示：请使用通用测试信用卡号 4242 4242 4242 4242，任意有效日期和 CVV 进行体验。",
        close: "关闭",
      }
    : {
        paymentTitle: "Complete payment",
        paymentDesc: "Please complete Stripe online payment below. After payment, the page will return and refresh the status.",
        testHint: "Test mode: use card 4242 4242 4242 4242 with any valid expiry date and CVV.",
        close: "Close",
      },
)

function close() {
  emit("update:open", false)
}
</script>

<template>
  <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4 backdrop-blur-sm">
    <div class="flex max-h-[92vh] w-full max-w-3xl flex-col overflow-hidden rounded-2xl bg-white shadow-2xl shadow-slate-950/20">
      <div class="flex items-start justify-between gap-4 border-b border-slate-100 px-6 py-5">
        <div class="flex min-w-0 items-start gap-3">
          <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-emerald-50 text-emerald-600">
            <CreditCard class="h-5 w-5" />
          </div>
          <div class="min-w-0">
            <h2 class="text-xl font-bold text-slate-950">{{ title || copy.paymentTitle }}</h2>
            <p v-if="subtitle" class="mt-1 break-all text-sm text-slate-500">{{ subtitle }}</p>
          </div>
        </div>
        <button type="button" class="rounded-xl p-2 text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-700" :aria-label="copy.close" @click="close">
          <X class="h-5 w-5" />
        </button>
      </div>

      <div class="overflow-y-auto px-6 py-5">
        <div class="mb-4 rounded-xl border border-blue-200 bg-blue-50 p-4 text-sm text-blue-900">
          <div class="font-semibold">{{ copy.paymentTitle }}</div>
          <p class="mt-1">{{ copy.paymentDesc }}</p>
        </div>
        <div class="mb-4 rounded-xl border border-amber-200 bg-amber-50 p-3 text-xs text-amber-800">
          {{ copy.testHint }}
        </div>
        <PaymentSessionPanel
          :payment-key="paymentKey"
          :biz-type="bizType"
          :biz-ref-ulid="bizRefUlid"
          :order-id="orderId"
          :source="source"
          :return-path="returnPath"
          :extra-return-params="extraReturnParams"
          min-height-class="min-h-[420px]"
        />
      </div>
    </div>
  </div>
</template>
