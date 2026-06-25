<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { RouterLink } from "vue-router"
import { AlertCircle, BookOpen, CheckCircle2, Clock, Lock, ShoppingCart, Users } from "lucide-vue-next"
import { CANDIDATE_PIPELINE_STATUS_LABELS, statusLabel } from "@/lib/status-labels"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"
import PurchaseDialog from "./PurchaseDialog.vue"

type CourseCardStat = { label: string; value: string | number }
type EligibilityBlocker = { blocker_type?: string; description?: string }
type EligibilityPreview = { can_purchase?: boolean; can_unlock?: boolean; blockers?: EligibilityBlocker[] }

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
}>()

const { t, lang } = useTranslation()
const showPurchaseDialog = ref(false)
const eligibility = ref<EligibilityPreview | null>(null)
const eligibilityLoading = ref(false)
const activeMembership = ref<any | null>(null)

const blockers = computed(() => eligibility.value?.blockers || [])
const isPipelineProduct = computed(() => Boolean(props.isPipelineBundle && props.pipelineId))
const isMembershipProduct = computed(() => Boolean(props.isMembershipBundle || props.itemTypes?.some((type) => String(type).includes("membership"))))
const effectivePurchased = computed(() =>
  Boolean(props.isPurchased || activeMembership.value || blockers.value.some((blocker) => blocker.blocker_type === "ALREADY_PURCHASED")),
)
const hasInProgressOrder = computed(() => blockers.value.some((blocker) => blocker.blocker_type === "IN_PROGRESS_PURCHASE"))
const resolvedStatusLabel = computed(() =>
  props.statusValue !== undefined ? statusLabel(t.value, CANDIDATE_PIPELINE_STATUS_LABELS, props.statusValue) : props.statusLabel,
)
const purchasedTarget = computed(() => isMembershipProduct.value ? "/membership" : `/certifications/${encodeURIComponent(props.pipelineId || props.id)}`)

const cardCopy = computed(() => ({
  ready: lang.value === "zh" ? "\u53ef\u8d2d\u4e70\u8ba4\u8bc1" : "Ready to buy",
  readyMembership: lang.value === "zh" ? "\u53ef\u8d2d\u4e70\u4f1a\u5458" : "Ready to subscribe",
  unlock: lang.value === "zh" ? "\u9700\u8981\u5148\u89e3\u9501" : "Unlock required",
  blocked: lang.value === "zh" ? "\u6682\u4e0d\u53ef\u8d2d\u4e70" : "Unavailable",
  checking: lang.value === "zh" ? "\u68c0\u67e5\u4e2d" : "Checking",
  missingQualification: lang.value === "zh" ? "\u7f3a\u5c11\u89e3\u9501\u8d44\u683c" : "Missing unlock qualification",
  alreadyPurchased: lang.value === "zh" ? "\u5df2\u8d2d\u4e70" : "Already purchased",
  inProgressPurchase: lang.value === "zh" ? "\u6709\u672a\u5b8c\u6210\u8ba2\u5355" : "Order in progress",
  pipelineNotFound: lang.value === "zh" ? "\u8ba4\u8bc1\u5df2\u4e0d\u53ef\u7528" : "No longer available",
  estimatedPrice: lang.value === "zh" ? "\u9884\u4f30\u4ef7\u683c" : "Estimated price",
}))

const actionCopy = computed(() => {
  if (effectivePurchased.value) return isMembershipProduct.value ? (lang.value === "zh" ? "\u8fdb\u5165\u4f1a\u5458\u4e2d\u5fc3" : "Membership Center") : (lang.value === "zh" ? "\u8fdb\u5165\u8ba4\u8bc1" : "Enter Certification")
  if (hasInProgressOrder.value) return lang.value === "zh" ? "\u7ee7\u7eed\u652f\u4ed8" : "Continue Payment"
  if (eligibility.value?.can_unlock) return lang.value === "zh" ? "\u53bb\u89e3\u9501" : "Unlock"
  if (eligibility.value?.can_purchase) return lang.value === "zh" ? "\u53bb\u8d2d\u4e70" : "Buy Now"
  if (eligibilityLoading.value && !eligibility.value) return lang.value === "zh" ? "\u68c0\u67e5\u4e2d" : "Checking"
  if (eligibility.value) return lang.value === "zh" ? "\u6682\u4e0d\u53ef\u8d2d\u4e70" : "Unavailable"
  return lang.value === "zh" ? "\u67e5\u770b\u8d2d\u4e70\u72b6\u6001" : "Check Status"
})

const actionClass = computed(() => {
  if (eligibility.value && !effectivePurchased.value && !eligibility.value.can_purchase && !eligibility.value.can_unlock && !hasInProgressOrder.value) {
    return "bg-slate-200 text-slate-500"
  }
  return "bg-primary text-white shadow-sm shadow-primary/20 group-hover:bg-primary/90"
})

onMounted(async () => {
  if (props.isPurchased) return
  if (isMembershipProduct.value) {
    eligibilityLoading.value = true
    try {
      activeMembership.value = await loadActiveMembership()
      eligibility.value = activeMembership.value
        ? { can_purchase: false, can_unlock: false, blockers: [{ blocker_type: "ALREADY_PURCHASED" }] }
        : { can_purchase: true, can_unlock: false, blockers: [] }
    } catch {
      eligibility.value = { can_purchase: true, can_unlock: false, blockers: [] }
    } finally {
      eligibilityLoading.value = false
    }
    return
  }
  if (!isPipelineProduct.value) {
    eligibility.value = { can_purchase: true, can_unlock: false, blockers: [] }
    return
  }
  
  const targetPipelineId = props.pipelineId
  if (!targetPipelineId) return
  
  eligibilityLoading.value = true
  try {
    eligibility.value = await apiClient(`/api/mall/pipelines/${targetPipelineId}/eligibility`)
  } catch {
    eligibility.value = null
  } finally {
    eligibilityLoading.value = false
  }
})

function isActiveMembershipStatus(status: unknown) {
  return ["active", "membership_status_active"].includes(String(status || "").trim().toLowerCase())
}

function membershipRecords(payload: any) {
  for (const key of ["user_memberships", "memberships", "records", "items", "history"]) {
    if (Array.isArray(payload?.[key])) return payload[key]
  }
  return []
}

async function loadActiveMembership() {
  const membershipGpath = String(props.membershipGpath || "").trim()
  const membershipId = String(props.membershipId || "").trim()
  if (!membershipGpath && !membershipId) return null
  const history = await apiClient("/api/membership/history?page=1&page_size=50", { suppressErrorToast: true })
  const matchingActive = membershipRecords(history).find((record: any) =>
    isActiveMembershipStatus(record?.status) &&
    ((membershipGpath && String(record?.membership_gpath || "").trim() === membershipGpath) ||
      (membershipId && String(record?.membership_ulid || "").trim() === membershipId)),
  )
  if (!matchingActive) return null
  if (!membershipGpath) return matchingActive
  const active = await apiClient(`/api/membership/active?membership_gpath=${encodeURIComponent(membershipGpath)}`, { suppressErrorToast: true })
  return active?.membership || matchingActive
}

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
  if (eligibilityLoading.value && !eligibility.value) {
    return { label: cardCopy.value.checking, icon: Clock, className: "border-slate-200 bg-slate-50 text-slate-700", hint: "" }
  }
  if (eligibility.value?.can_purchase) {
    return { label: isMembershipProduct.value ? cardCopy.value.readyMembership : cardCopy.value.ready, icon: ShoppingCart, className: "border-emerald-200 bg-emerald-50 text-emerald-700", hint: "" }
  }
  if (eligibility.value?.can_unlock) {
    return { label: cardCopy.value.unlock, icon: Lock, className: "border-blue-200 bg-blue-50 text-blue-700", hint: "" }
  }
  if (eligibility.value) {
    return { label: cardCopy.value.blocked, icon: AlertCircle, className: "border-amber-200 bg-amber-50 text-amber-800", hint: blockerText(blockers.value[0]) }
  }
  return null
})
</script>

<template>
  <component
    :is="effectivePurchased ? RouterLink : 'div'"
    :to="effectivePurchased ? purchasedTarget : undefined"
    class="group flex h-full flex-col overflow-hidden rounded-[16px] border-2 border-[#dfe4ea] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all duration-300 hover:-translate-y-0.5 hover:border-primary hover:shadow-[0_18px_42px_rgba(16,30,67,0.16)]"
    :class="!effectivePurchased && 'cursor-pointer'"
    @click="!effectivePurchased && (showPurchaseDialog = true)"
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
  />
</template>
