<script setup lang="ts">
import { computed, ref } from "vue"
import { RouterLink, useRouter } from "vue-router"
import { toast } from "vue-sonner"
import { AlertCircle, BookOpen, CheckCircle2, Clock, ShoppingCart, Users } from "lucide-vue-next"
import { CANDIDATE_PIPELINE_STATUS_LABELS, statusLabel } from "@/lib/status-labels"
import { startGfiLogin } from "@/lib/gfiLogin"
import { useTranslation } from "@/lib/language"
import { apiClient } from "@/lib/apiClient"
import { preloadCheckoutWizard } from "@/router"

type CourseCardStat = { label: string; value: string | number }
type EligibilityBlocker = { blocker_type?: string; description?: string; details?: unknown[] }
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
  stages?: any[]
  loginRequired?: boolean
}>()

const { t } = useTranslation()
const router = useRouter()
const freshBundle = ref<any | null>(null)
const statusRefreshing = ref(false)
const currentEligibility = computed<EligibilityPreview | null>(() => freshBundle.value?.purchase_state?.eligibility || freshBundle.value?.eligibility || props.eligibility || null)
const currentActiveOrder = computed<ActiveOrderPreview | null>(() => freshBundle.value?.purchase_state?.active_order || freshBundle.value?.active_order || props.activeOrder || null)
const currentActiveMembership = computed<Record<string, unknown> | null>(() => freshBundle.value?.active_membership || props.activeMembership || null)
const blockers = computed(() => currentEligibility.value?.blockers || [])
const credentialCenterBlockers = computed(() => blockers.value.filter((blocker) => [
  "EXEMPTION_DOCUMENTS_PENDING_UPLOAD",
  "EXEMPTION_UNDER_REVIEW",
].includes(String(blocker.blocker_type || ""))))
const credentialCenterBlocker = computed(() => credentialCenterBlockers.value[0])
const hardBlockers = computed(() => blockers.value.filter((blocker) => [
  "FORBIDDEN_QUALIFICATION",
  "CONFLICT_PIPELINE_IN_PROGRESS",
  "CONFLICT_CHECK_UNAVAILABLE",
].includes(String(blocker.blocker_type || ""))))
const isPipelineProduct = computed(() => Boolean(props.isPipelineBundle && props.pipelineId))
const isMembershipProduct = computed(() => Boolean(props.isMembershipBundle || props.itemTypes?.some((type) => String(type).includes("membership"))))
const isCombinationProduct = computed(() => isPipelineProduct.value && isMembershipProduct.value)
const isMembershipOnlyProduct = computed(() => isMembershipProduct.value && !isPipelineProduct.value)
const alreadyPurchasedBlockers = computed(() => blockers.value.filter((blocker) => blocker.blocker_type === "ALREADY_PURCHASED"))
const hasPurchasedPipeline = computed(() => alreadyPurchasedBlockers.value.some((blocker) => String(blocker.description || "").toLowerCase().includes("pipeline")))
const hasPurchasedMembership = computed(() =>
  Boolean(currentActiveMembership.value) ||
  alreadyPurchasedBlockers.value.some((blocker) => String(blocker.description || "").toLowerCase().includes("membership")),
)
const effectivePurchased = computed(() => {
  if (props.isPurchased) return true
  if (isCombinationProduct.value) return hasPurchasedPipeline.value && hasPurchasedMembership.value
  if (isPipelineProduct.value) return hasPurchasedPipeline.value
  if (isMembershipProduct.value) return hasPurchasedMembership.value
  return alreadyPurchasedBlockers.value.length > 0
})
const hasInProgressOrder = computed(() => Boolean(currentActiveOrder.value) || blockers.value.some((blocker) => blocker.blocker_type === "IN_PROGRESS_PURCHASE"))
const resolvedStatusLabel = computed(() =>
  props.statusValue !== undefined ? statusLabel(t.value, CANDIDATE_PIPELINE_STATUS_LABELS, props.statusValue) : props.statusLabel,
)
const purchasedTarget = computed(() => isPipelineProduct.value ? `/certifications/${encodeURIComponent(props.pipelineId || props.id)}` : "/membership")

const cardCopy = computed(() => t.value.courseCard)

const actionCopy = computed(() => {
  if (props.loginRequired) return cardCopy.value.loginToPurchase
  if (effectivePurchased.value) return isPipelineProduct.value ? cardCopy.value.enterCertification : cardCopy.value.membershipCenter
  if (statusRefreshing.value) return cardCopy.value.checking
  if (hasInProgressOrder.value) return cardCopy.value.continuePayment
  if (currentEligibility.value?.can_purchase || currentEligibility.value?.can_unlock) return cardCopy.value.buyNow
  if (credentialCenterBlocker.value?.blocker_type === "EXEMPTION_DOCUMENTS_PENDING_UPLOAD") return cardCopy.value.uploadExemptionDocuments
  if (credentialCenterBlocker.value?.blocker_type === "EXEMPTION_UNDER_REVIEW") return cardCopy.value.viewExemptionReview
  if (currentEligibility.value) return cardCopy.value.unavailable
  return cardCopy.value.checkStatus
})

const actionClass = computed(() => {
  if (props.loginRequired) return "bg-primary text-white shadow-sm shadow-primary/20 group-hover:bg-primary/90"
  if (statusRefreshing.value) return "bg-slate-200 text-slate-500"
  if (credentialCenterBlocker.value) return "bg-primary text-white shadow-sm shadow-primary/20 group-hover:bg-primary/90"
  if (currentEligibility.value && !effectivePurchased.value && !currentEligibility.value.can_purchase && !currentEligibility.value.can_unlock && !hasInProgressOrder.value) {
    return "bg-slate-200 text-slate-500"
  }
  return "bg-primary text-white shadow-sm shadow-primary/20 group-hover:bg-primary/90"
})

function blockerText(blocker?: EligibilityBlocker) {
  if (!blocker) return ""
  if (isCombinationProduct.value && (hasPurchasedPipeline.value !== hasPurchasedMembership.value)) return cardCopy.value.partiallyOwnedBundle
  if (blocker.blocker_type === "MISSING_UNLOCK_QUALIFICATION") return cardCopy.value.missingQualification
  if (blocker.blocker_type === "ALREADY_PURCHASED") return cardCopy.value.alreadyPurchased
  if (blocker.blocker_type === "IN_PROGRESS_PURCHASE") return cardCopy.value.inProgressPurchase
  if (blocker.blocker_type === "PIPELINE_NOT_FOUND") return cardCopy.value.pipelineNotFound
  if (blocker.blocker_type === "FORBIDDEN_QUALIFICATION") return cardCopy.value.forbiddenQualification
  if (blocker.blocker_type === "CONFLICT_PIPELINE_IN_PROGRESS") return cardCopy.value.conflictPipelineInProgress
  if (blocker.blocker_type === "CONFLICT_CHECK_UNAVAILABLE") return cardCopy.value.conflictCheckUnavailable
  if (blocker.blocker_type === "EXEMPTION_DOCUMENTS_PENDING_UPLOAD") return cardCopy.value.exemptionDocumentsPendingUpload
  if (blocker.blocker_type === "EXEMPTION_UNDER_REVIEW") return cardCopy.value.exemptionUnderReview
  return blocker.description || blocker.blocker_type || ""
}

function blockerQualificationIDs(blocker: EligibilityBlocker) {
  const detailIDs = new Set((blocker.details || []).map((detail) => String(detail || "").trim()).filter(Boolean))
  const stages = Array.isArray(freshBundle.value?.stages) ? freshBundle.value.stages : (props.stages || [])
  const IDs = stages.flatMap((stage: any) => Array.isArray(stage?.units) ? stage.units : [])
    .filter((unit: any) => detailIDs.size === 0 || detailIDs.has(String(unit?.unit_id || unit?.unit_ulid || "")))
    .flatMap((unit: any) => Array.isArray(unit?.exemption_quals) ? unit.exemption_quals : [])
    .map((qualification: any) => typeof qualification === "string"
      ? qualification
      : String(qualification?.qual_ulid || qualification?.qual_id || qualification?.cred_def_ulid || ""))
    .filter(Boolean)
  return Array.from(new Set(IDs))
}

async function openCredentialCenter() {
  const qualificationIDs = Array.from(new Set(
    credentialCenterBlockers.value.flatMap(blockerQualificationIDs),
  ))
  await router.push({
    path: "/credentials",
    query: qualificationIDs.length > 0 ? { qual_ids: qualificationIDs.join(",") } : undefined,
  })
}

const accessState = computed(() => {
  if (props.loginRequired) {
    return { label: cardCopy.value.loginRequired, icon: ShoppingCart, className: "border-primary/20 bg-primary/10 text-primary", hint: "" }
  }
  if (effectivePurchased.value) return null
  if (statusRefreshing.value) {
    return { label: cardCopy.value.checking, icon: Clock, className: "border-slate-200 bg-slate-50 text-slate-700", hint: "" }
  }
  if (currentEligibility.value?.can_purchase || currentEligibility.value?.can_unlock || hasInProgressOrder.value) {
    return { label: isMembershipOnlyProduct.value ? cardCopy.value.readyMembership : cardCopy.value.ready, icon: ShoppingCart, className: "border-emerald-200 bg-emerald-50 text-emerald-700", hint: "" }
  }
  if (currentEligibility.value) {
    return {
      label: credentialCenterBlocker.value?.blocker_type === "EXEMPTION_DOCUMENTS_PENDING_UPLOAD"
        ? cardCopy.value.exemptionDocumentsRequired
        : credentialCenterBlocker.value?.blocker_type === "EXEMPTION_UNDER_REVIEW"
          ? cardCopy.value.exemptionReviewPending
          : cardCopy.value.blocked,
      icon: AlertCircle,
      className: "border-amber-200 bg-amber-50 text-amber-800",
      hint: blockerText(credentialCenterBlocker.value || blockers.value[0]),
    }
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
  if (props.loginRequired) {
    try {
      await startGfiLogin()
    } catch (error) {
      console.error("Unable to start GFI login:", error)
      toast.error(t.value.loginPage.errorTitle)
    }
    return
  }
  if (effectivePurchased.value || statusRefreshing.value) return
  void preloadCheckoutWizard()
  await refreshBundleState()
  if (effectivePurchased.value) return
  if (hasInProgressOrder.value) {
    toast.info(t.value.purchaseDialog?.inProgressPurchaseDesc || cardCopy.value.inProgressPurchase)
    router.push({ path: "/orders" })
    return
  }
  if (credentialCenterBlocker.value) {
    await openCredentialCenter()
    return
  }
  if (hardBlockers.value.length) {
    toast.info(blockerText(hardBlockers.value[0]))
    return
  }
  router.push(`/checkout/${encodeURIComponent(props.id)}`)
}
</script>

<template>
  <component
    :is="effectivePurchased ? RouterLink : 'div'"
    :to="effectivePurchased ? purchasedTarget : undefined"
    data-testid="certification-card"
    :data-bundle-id="id"
    :data-pipeline-id="pipelineId || ''"
    class="group flex h-full flex-col overflow-hidden rounded-lg border border-[#ccd7e8] bg-white transition-colors duration-200 hover:border-primary"
    :class="!effectivePurchased && 'cursor-pointer'"
    @click="handleCardClick"
    @pointerenter="!effectivePurchased && preloadCheckoutWizard()"
    @focusin="!effectivePurchased && preloadCheckoutWizard()"
  >
    <div class="course-card-media relative h-32 overflow-hidden bg-white sm:h-36 xl:h-40">
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

    <div class="course-card-body flex flex-1 flex-col p-4 sm:p-5">
      <div class="course-card-summary mb-3 min-h-[104px]">
        <h3 class="course-card-title mb-2 min-h-12 break-words text-lg font-semibold leading-6 text-card-foreground transition-colors group-hover:text-primary">{{ title }}</h3>
        <p class="line-clamp-2 min-h-10 text-sm leading-5 text-muted-foreground">{{ description }}</p>
      </div>

      <div class="course-card-purchase-details mb-3 flex min-h-[132px] flex-col justify-start space-y-2.5">
        <div v-if="resolvedStatusLabel" class="flex flex-wrap gap-2">
          <span class="badge border-primary/20 bg-primary/10 text-primary">{{ resolvedStatusLabel }}</span>
        </div>

        <div class="course-card-detail-slot min-h-[52px]">
          <div v-if="accessState" :class="['rounded-lg border px-3 py-2 text-xs', accessState.className]">
            <div class="flex items-center gap-1.5 font-medium">
              <component :is="accessState.icon" class="h-3.5 w-3.5" />
              {{ accessState.label }}
            </div>
            <div v-if="accessState.hint" class="mt-1 text-[11px] opacity-80">{{ accessState.hint }}</div>
          </div>
        </div>

        <div class="course-card-detail-slot min-h-[52px]">
          <div v-if="priceLabel" class="space-y-0.5">
            <div class="text-sm font-medium leading-5 text-muted-foreground">
              {{ cardCopy.estimatedPrice }}
            </div>
            <div class="whitespace-nowrap text-[22px] font-bold leading-tight text-[#002a66] xl:text-[24px] 2xl:text-[26px]">
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

      <div v-if="stats?.length" class="course-card-stats mb-3 grid grid-cols-3 gap-2 rounded-lg border border-border bg-muted p-2 text-center">
        <div v-for="stat in stats" :key="stat.label" class="course-card-stat rounded-lg bg-white px-2 py-2">
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

      <div class="course-card-footer mt-auto border-t border-border pt-3">
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
</template>

<style scoped>
@media (max-width: 767px) {
  .course-card-media {
    height: 112px;
  }

  .course-card-body {
    padding: 12px;
  }

  .course-card-summary,
  .course-card-title,
  .course-card-purchase-details,
  .course-card-detail-slot {
    min-height: 0;
  }

  .course-card-summary,
  .course-card-purchase-details {
    margin-bottom: 10px;
  }

  .course-card-title {
    margin-bottom: 6px;
  }

  .course-card-stats {
    margin-bottom: 10px;
    padding: 6px;
  }

  .course-card-stat {
    padding-block: 6px;
  }

  .course-card-footer {
    padding-top: 8px;
  }
}
</style>
