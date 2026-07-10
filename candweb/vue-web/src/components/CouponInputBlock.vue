<script setup lang="ts">
import { computed } from "vue"
import { Loader2 } from "lucide-vue-next"
import { useTranslation } from "@/lib/language"

const props = defineProps<{
  modelValue: string
  activeCouponCodes: string[]
  loading?: boolean
  disabled?: boolean
  error?: string
  cannotPayReason?: string
}>()

const emit = defineEmits<{
  "update:modelValue": [value: string]
  "apply": []
  "clear": []
}>()

const { t } = useTranslation()
const copy = computed(() => t.value.purchaseDialog || {})

function onApply() {
  emit("apply")
}
function onClear() {
  emit("clear")
}
</script>

<template>
  <div class="rounded-lg border border-blue-100 bg-blue-50/70 p-4">
    <label class="text-sm font-semibold text-foreground" for="purchase-coupon-input">{{ copy.couponTitle }}</label>
    <p class="mt-1 text-xs text-muted-foreground">{{ copy.couponHint }}</p>
    <div class="mt-3 grid gap-2 sm:grid-cols-[minmax(0,1fr)_auto]">
      <input
        id="purchase-coupon-input"
        :value="modelValue"
        @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
        class="input h-10 bg-white shadow-sm shadow-blue-100/40"
        :placeholder="copy.couponPlaceholder"
        :disabled="loading || disabled"
        @keydown.enter.prevent="onApply"
      />
      <button type="button" class="inline-flex h-10 min-w-[112px] shrink-0 items-center justify-center gap-2 rounded-lg border border-blue-100 bg-white px-4 text-sm font-semibold text-slate-800 shadow-sm shadow-blue-100/50 transition-colors hover:border-blue-200 hover:bg-blue-50 hover:text-slate-950 disabled:cursor-not-allowed disabled:opacity-50" :disabled="loading || disabled" @click="onApply">
        <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
        {{ copy.applyCoupon }}
      </button>
      <button v-if="activeCouponCodes.length" type="button" class="inline-flex h-10 shrink-0 items-center justify-center rounded-lg border border-blue-200 bg-blue-50 px-4 text-sm font-semibold text-blue-700 transition-colors hover:border-blue-300 hover:bg-blue-100 hover:text-blue-900 disabled:cursor-not-allowed disabled:opacity-50 sm:col-start-2" :disabled="loading || disabled" @click="onClear">
        {{ copy.clearCoupon }}
      </button>
    </div>
    <div v-if="activeCouponCodes.length" class="mt-2 flex flex-wrap gap-2">
      <span v-for="code in activeCouponCodes" :key="code" class="rounded-full bg-primary/10 px-2 py-1 text-xs font-semibold text-primary">{{ code }}</span>
    </div>
    <p v-if="error" class="mt-2 text-xs text-red-600">{{ error }}</p>
    <p v-if="cannotPayReason" class="mt-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs font-semibold text-amber-800">
      {{ cannotPayReason }}
    </p>
  </div>
</template>
