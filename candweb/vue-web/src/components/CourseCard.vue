<script setup lang="ts">
import { computed, ref } from "vue"
import { RouterLink, useRouter } from "vue-router"
import { AlertCircle, BookOpen, CheckCircle2, Clock, Lock, ShoppingCart, Users } from "lucide-vue-next"
import { CANDIDATE_PIPELINE_STATUS_LABELS, statusLabel } from "@/lib/status-labels"
import { useTranslation } from "@/lib/language"
import { apiClient } from "@/lib/apiClient"
import PurchaseDialog from "./PurchaseDialog.vue"

type CourseCardStat = { label: string; value: string | number }
type EligibilityBlocker = { blocker_type?: string; description?: string }
type EligibilityPreview = { eligible?: boolean; can_purchase?: boolean; can_unlock?: boolean; blockers?: EligibilityBlocker[] }
type ActiveOrderPreview = { action?: "purchase" | "unlock"; order_id?: string; orderId?: string; status?: string; pay_order_id?: string; payOrderId?: string; message?: string }
type PaymentPreview = { subtotal?: number; discount_total?: number; tax_total?: number; total?: number; currency?: string }
type ExemptionOptions = { stages?: any[] }

const props = defineProps<{
  id: string
  pipelineId?: string
  membershipId?: string
  membershipGpath?: string
  itemTypes?: string[]
  isPipelineBundle?: boolean
  isMembershipBundle?: boolean
  title: string
  description: string
  image?: string
  category?: "course" | "column" | "short"
  provider: string
  duration?: string
  students?: number
  isPurchased?: boolean
  progress?: number
  statusLabel?: string
  statusValue?: string | number
  versionLabel?: string
  priceLabel?: string
  stats?: CourseCardStat[]
  eligibility?: EligibilityPreview | null
  activeOrder?: ActiveOrderPreview | null
  paymentPreview?: PaymentPreview | null
  exemptionOptions?: ExemptionOptions | null
  activeMembership?: Record<string, unknown> | null
}>()

const { t, lang } = useTranslation()
const router = useRouter()
const showPurchaseDialog = ref(false)
const freshBundle = ref<any | null>(null)
const statusRefreshing = ref(false)
const currentEligibility = computed<EligibilityPreview | null>(() => freshBundle.value?.purchase_state?.eligibility || freshBundle.value?.eligibility || props.eligibility || null)
const currentActiveOrder = computed<ActiveOrderPreview | null>(() => freshBundle.value?.purchase_state?.active_order || freshBundle.value?.active_order || props.activeOrder || null)
const currentPaymentPreview = computed<PaymentPreview | null>(() => freshBundle.value?.purchase_state?.payment_preview || freshBundle.value?.payment_preview || props.paymentPreview || null)
const currentExemptionOptions = computed<ExemptionOptions | null>(() => freshBundle.value?.purchase_state?.exemption_options || freshBundle.value?.exemption_options || props.exemptionOptions || null)
const currentActiveMembership = computed<Record<string, unknown> | null>(() => freshBundle.value?.active_membership || props.activeMembership || null)
const blockers = computed(() => currentEligibility.value?.blockers || [])
const isPipelineProduct = computed(() => Boolean(props.isPipelineBundle && props.pipelineId))
const isMembershipProduct = computed(() => Boolean(props.isMembershipBundle || props.itemTypes?.some((type) => String(type).includes("membership"))))
const effectivePurchased = computed(() =>
  Boolean(props.isPurchased || currentActiveMembership.value || blockers.value.some((blocker) => blocker.blocker_type === "ALREADY_PURCHASED")),
)
const hasInProgressOrder = computed(() => Boolean(currentActiveOrder.value) || blockers.value.some((blocker) => blocker.blocker_type === "IN_PROGRESS_PURCHASE"))
const resolvedStatusLabel = computed(() =>
  props.statusValue !== undefined ? statusLabel(t.value, CANDIDATE_PIPELINE_STATUS_LABELS, props.statusValue) : props.statusLabel,
)
const purchasedTarget = computed(() => isMembershipProduct.value ? "/membership" : `/certifications/${encodeURIComponent(props.pipelineId || props.id)}`)

const cardCopy = computed(() => t.value.courseCard)

const actionCopy = computed(() => {
  if (effectivePurchased.value) return isMembershipProduct.value ? cardCopy.value.membershipCenter : cardCopy.value.enterCertification
  if (statusRefreshing.value) return cardCopy.value.checking
  if (hasInProgressOrder.value) return cardCopy.value.continuePayment
  if (currentEligibility.value?.can_unlock) return cardCopy.value.unlockAction
  if (currentEligibility.value?.can_purchase) return cardCopy.value.buyNow
  if (currentEligibility.value) return cardCopy.value.unavailable
  return cardCopy.value.checkStatus
})

const actionClass = computed(() => {
  if (statusRefreshing.value) return "bg-slate-200 text-slate-500"
  if (currentEligibility.value && !effectivePurchased.value && !currentEligibility.value.can_purchase && !currentEligibility.value.can_unlock && !hasInProgressOrder.value) {
    return "bg-slate-200 text-slate-500"
  }
  return "bg-primary text-white shadow-sm shadow-primary/20 group-hover:bg-primary/90"
})

function blockerText(blocker?: EligibilityBlocker) {
  if (!blocker) return ""
  if (blocker.blocker_type === "MISSING_UNLOCK_QUALIFICATION") return cardCopy.value.missingQualification
  if (blocker.blocker_type === "ALREADY_PURCHASED") return cardCopy.value.alreadyPurchased
  if (blocker.blocker_type === "IN_PROGRESS_PURCHASE") return cardCopy.value.inProgressPurchase
  if (blocker.blocker_type === "PIPELINE_NOT_FOUND") return cardCopy.value.pipelineNotFound
  return blocker.description || blocker.blocker_type || ""
}

const accessState = computed(() => {
  if (effectivePurchased.value) return null
  if (statusRefreshing.value) {
    return { label: cardCopy.value.checking, icon: Clock, className: "border-slate-200 bg-slate-50 text-slate-700", hint: "" }
  }
  if (currentEligibility.value?.can_purchase || hasInProgressOrder.value) {
    return { label: isMembershipProduct.value ? cardCopy.value.readyMembership : cardCopy.value.ready, icon: ShoppingCart, className: "border-emerald-200 bg-emerald-50 text-emerald-700", hint: "" }
  }
  if (currentEligibility.value?.can_unlock) {
    return { label: cardCopy.value.unlock, icon: Lock, className: "border-blue-200 bg-blue-50 text-blue-700", hint: "" }
  }
  if (currentEligibility.value) {
    return { label: cardCopy.value.blocked, icon: AlertCircle, className: "border-amber-200 bg-amber-50 text-amber-800", hint: blockerText(blockers.value[0]) }
  }
  return { label: cardCopy.value.checking, icon: Clock, className: "border-slate-200 bg-slate-50 text-slate-700", hint: "" }
})

async function refreshBundleState() {
  if (!props.id || statusRefreshing.value) return false
  statusRefreshing.value = true
  try {
    freshBundle.value = await apiClient(`/api/mall/bundles/${encodeURIComponent(props.id)}`, { suppressErrorToast: true })
    return true
  } catch (error) {
    console.error("Failed to refresh bundle state", error)
    return false
  } finally {
    statusRefreshing.value = false
  }
}

async function handleCardClick() {
  if (effectivePurchased.value || statusRefreshing.value) return
  await refreshBundleState()
  if (effectivePurchased.value) return
  if (hasInProgressOrder.value) {
    router.push({ path: "/orders" })
    return
  }
  showPurchaseDialog.value = true
}
</script>

<template>
  <component
    :is="effectivePurchased ? RouterLink : 'div'"
    :to="effectivePurchased ? purchasedTarget : undefined"
    class="group flex h-full flex-col overflow-hidden rounded-[16px] border-2 border-[#dfe4ea] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all duration-300 hover:-translate-y-0.5 hover:border-primary hover:shadow-[0_18px_42px_rgba(16,30,67,0.16)]"
    :class="!effectivePurchased && 'cursor-pointer'"
    @click="handleCardClick"
  >
    <div class="relative h-32 overflow-hidden bg-white sm:h-36 xl:h-40">
      <template v-if="image">
        <img :src="image" :alt="title" class="h-full w-full scale-[1.65] object-contain px-2 py-3 transition-transform duration-500 group-hover:scale-[1.72]" />
      </template>
      <div v-else class="flex h-full items-center justify-center bg-white">
        <div class="flex h-16 w-16 items-center justify-center rounded-xl bg-white text-primary shadow-sm">
          <BookOpen class="h-9 w-9" />
        </div>
      </div>

      <span v-if="effectivePurchased" class="badge absolute right-3 top-3 gap-1 border-0 bg-emerald-500 text-white shadow-sm">
        <CheckCircle2 class="h-3 w-3" />
        {{ t.courses.purchased }}
      </span>

    </div>

    <div class="flex flex-1 flex-col p-4 sm:p-5">
      <div class="mb-3 min-h-[78px]">
        <h3 class="mb-2 line-clamp-1 text-lg font-semibold text-card-foreground transition-colors group-hover:text-primary">{{ title }}</h3>
        <p class="line-clamp-2 min-h-10 text-sm leading-5 text-muted-foreground">{{ description }}</p>
      </div>

      <div class="mb-3 flex min-h-[132px] flex-col justify-start space-y-2.5">
        <div v-if="resolvedStatusLabel" class="flex flex-wrap gap-2">
          <span class="badge border-primary/20 bg-primary/10 text-primary">{{ resolvedStatusLabel }}</span>
        </div>

        <div class="min-h-[52px]">
          <div v-if="accessState" :class="['rounded-lg border px-3 py-2 text-xs', accessState.className]">
            <div class="flex items-center gap-1.5 font-medium">
              <component :is="accessState.icon" class="h-3.5 w-3.5" />
              {{ accessState.label }}
            </div>
            <div v-if="accessState.hint" class="mt-1 text-[11px] opacity-80">{{ accessState.hint }}</div>
          </div>
        </div>

        <div class="min-h-[52px]">
          <div v-if="priceLabel" class="space-y-0.5">
            <div class="text-sm font-medium leading-5 text-[#4a4f59]">
              {{ cardCopy.estimatedPrice }}
            </div>
            <div class="whitespace-nowrap text-[22px] font-bold leading-tight tracking-tight text-[#101114] xl:text-[24px] 2xl:text-[26px]">
              {{ priceLabel }}
            </div>
          </div>
          <div v-else class="space-y-0.5">
            <div class="text-sm font-medium leading-5 text-transparent select-none">
              -
            </div>
            <div class="whitespace-nowrap text-[22px] font-bold leading-tight tracking-tight text-emerald-500 xl:text-[24px] 2xl:text-[26px]">
              {{ cardCopy.free }}
            </div>
          </div>
        </div>

        <div v-if="effectivePurchased && progress !== undefined">
          <div class="mb-1.5 flex items-center justify-between text-xs">
            <span class="text-muted-foreground">{{ t.courses.courseProgress }}</span>
            <span class="font-medium text-primary">{{ progress }}%</span>
          </div>
          <div class="h-1.5 w-full overflow-hidden rounded-full bg-muted">
            <div class="h-full rounded-full bg-primary transition-all" :style="{ width: `${progress}%` }" />
          </div>
        </div>
      </div>

      <div v-if="stats?.length" class="mb-3 grid grid-cols-3 gap-2 rounded-lg bg-[#f6fafb] p-2 text-center">
        <div v-for="stat in stats" :key="stat.label" class="rounded-lg bg-white px-2 py-2">
          <div class="text-sm font-semibold text-foreground">{{ stat.value }}</div>
          <div class="truncate text-[11px] text-muted-foreground">{{ stat.label }}</div>
        </div>
      </div>

      <div v-if="duration || students !== undefined" class="mb-4 flex items-center justify-between text-sm text-muted-foreground">
        <div class="flex items-center gap-4">
          <div v-if="duration" class="flex items-center gap-1.5">
            <Clock class="h-4 w-4" />
            <span>{{ duration }}</span>
          </div>
          <div v-if="students !== undefined" class="flex items-center gap-1.5">
            <Users class="h-4 w-4" />
            <span>{{ students.toLocaleString() }}</span>
          </div>
        </div>
      </div>

      <div class="mt-auto border-t border-border pt-3">
        <div
          :class="[
            'flex h-10 w-full items-center justify-center rounded-lg px-4 text-sm font-semibold transition-all duration-300',
            actionClass,
          ]"
        >
          <span>{{ actionCopy }}</span>
        </div>
      </div>
    </div>
  </component>

  <PurchaseDialog
    v-model:open="showPurchaseDialog"
    :course-name="title"
    :description="description"
    :pipeline-id="pipelineId || ''"
    :bundle-id="id"
    :is-pipeline-bundle="isPipelineProduct"
    :is-membership-bundle="isMembershipProduct"
    :membership-id="membershipId || ''"
    :membership-gpath="membershipGpath || ''"
    :initial-eligibility="currentEligibility || null"
    :initial-active-order="currentActiveOrder || null"
    :initial-payment-preview="currentPaymentPreview || null"
    :initial-exemption-options="currentExemptionOptions || null"
    @cancelled="refreshBundleState"
  />
</template>
