<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { RouterLink } from "vue-router"
import { AlertCircle, BookOpen, CheckCircle2, ChevronRight, Clock, Lock, Play, ShoppingCart, Users } from "lucide-vue-next"
import { CANDIDATE_PIPELINE_STATUS_LABELS, statusLabel } from "@/lib/status-labels"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"
import PurchaseDialog from "./PurchaseDialog.vue"

type CourseCardStat = { label: string; value: string | number }
type EligibilityBlocker = { blocker_type?: string; description?: string }
type EligibilityPreview = { can_purchase?: boolean; can_unlock?: boolean; blockers?: EligibilityBlocker[] }

const props = defineProps<{
  id: string
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
  stats?: CourseCardStat[]
}>()

const { t, lang } = useTranslation()
const showPurchaseDialog = ref(false)
const eligibility = ref<EligibilityPreview | null>(null)
const eligibilityLoading = ref(false)

const blockers = computed(() => eligibility.value?.blockers || [])
const effectivePurchased = computed(() => Boolean(props.isPurchased || blockers.value.some((blocker) => blocker.blocker_type === "ALREADY_PURCHASED")))
const resolvedStatusLabel = computed(() => props.statusValue !== undefined ? statusLabel(t.value, CANDIDATE_PIPELINE_STATUS_LABELS, props.statusValue) : props.statusLabel)
const cardCopy = computed(() => ({
  ready: lang.value === "zh" ? "可购买认证" : "Ready to buy",
  unlock: lang.value === "zh" ? "需要先解锁" : "Unlock required",
  blocked: lang.value === "zh" ? "暂不可购买" : "Unavailable",
  checking: lang.value === "zh" ? "检查中" : "Checking",
  missingQualification: lang.value === "zh" ? "缺少解锁资格" : "Missing unlock qualification",
  alreadyPurchased: lang.value === "zh" ? "已购买" : "Already purchased",
  inProgressPurchase: lang.value === "zh" ? "有未完成订单" : "Order in progress",
  pipelineNotFound: lang.value === "zh" ? "认证已不可用" : "No longer available",
}))

onMounted(async () => {
  if (props.isPurchased || !props.id) return
  eligibilityLoading.value = true
  try {
    eligibility.value = await apiClient(`/api/mall/pipelines/${props.id}/eligibility`)
  } catch {
    eligibility.value = null
  } finally {
    eligibilityLoading.value = false
  }
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
  if (eligibilityLoading.value && !eligibility.value) return { label: cardCopy.value.checking, icon: Clock, className: "border-slate-200 bg-slate-50 text-slate-700", hint: "" }
  if (eligibility.value?.can_purchase) return { label: cardCopy.value.ready, icon: ShoppingCart, className: "border-emerald-200 bg-emerald-50 text-emerald-700", hint: "" }
  if (eligibility.value?.can_unlock) return { label: cardCopy.value.unlock, icon: Lock, className: "border-blue-200 bg-blue-50 text-blue-700", hint: "" }
  if (eligibility.value) return { label: cardCopy.value.blocked, icon: AlertCircle, className: "border-amber-200 bg-amber-50 text-amber-800", hint: blockerText(blockers.value[0]) }
  return null
})
</script>

<template>
  <component
    :is="effectivePurchased ? RouterLink : 'div'"
    :to="effectivePurchased ? `/courses/detail?id=${encodeURIComponent(id)}` : undefined"
    class="group flex h-full flex-col overflow-hidden rounded-[22px] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all duration-300 hover:-translate-y-0.5 hover:shadow-md hover:shadow-primary/10"
    :class="!effectivePurchased && 'cursor-pointer'"
    @click="!effectivePurchased && (showPurchaseDialog = true)"
  >
    <div class="relative h-36 overflow-hidden bg-[#eaf5f7] sm:h-40 xl:h-44">
      <template v-if="image">
        <img :src="image" :alt="title" class="h-full w-full object-cover transition-transform duration-500 group-hover:scale-105" />
        <div class="absolute inset-0 bg-gradient-to-t from-slate-950/45 via-slate-950/5 to-transparent" />
      </template>
      <div v-else class="flex h-full items-center justify-center bg-[#eaf5f7]">
        <div class="flex h-16 w-16 items-center justify-center rounded-2xl bg-white text-primary shadow-sm">
          <BookOpen class="h-9 w-9" />
        </div>
      </div>

      <span v-if="effectivePurchased" class="badge absolute right-3 top-3 gap-1 border-0 bg-emerald-500 text-white shadow-sm">
        <CheckCircle2 class="h-3 w-3" />
        {{ t.courses.purchased }}
      </span>

      <div class="absolute inset-x-3 bottom-3 flex items-center justify-between">
        <span v-if="versionLabel" class="badge border-white/70 bg-white/90 text-foreground shadow-sm backdrop-blur">
          {{ versionLabel }}
        </span>
        <span v-else class="badge border-white/70 bg-white/90 text-primary shadow-sm backdrop-blur">
          {{ t.courses.pipeline }}
        </span>
        <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-white/95 text-primary shadow-sm backdrop-blur transition-transform duration-300 group-hover:scale-105">
          <Play v-if="effectivePurchased" class="h-4 w-4 fill-current" />
          <ShoppingCart v-else class="h-4 w-4" />
        </div>
      </div>
    </div>

    <div class="flex flex-1 flex-col p-5">
      <h3 class="mb-2 line-clamp-1 text-lg font-semibold text-card-foreground transition-colors group-hover:text-primary">{{ title }}</h3>
      <p class="mb-4 line-clamp-2 min-h-10 text-sm leading-5 text-muted-foreground">{{ description }}</p>

      <div class="mb-4 min-h-[58px] space-y-3">
        <div v-if="resolvedStatusLabel" class="flex flex-wrap gap-2">
          <span class="badge border-primary/20 bg-primary/10 text-primary">{{ resolvedStatusLabel }}</span>
        </div>

        <div v-if="accessState" :class="['rounded-xl border px-3 py-2 text-xs', accessState.className]">
          <div class="flex items-center gap-1.5 font-medium">
            <component :is="accessState.icon" class="h-3.5 w-3.5" />
            {{ accessState.label }}
          </div>
          <div v-if="accessState.hint" class="mt-1 text-[11px] opacity-80">{{ accessState.hint }}</div>
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

      <div v-if="stats?.length" class="mb-4 grid grid-cols-3 gap-2 rounded-xl bg-[#f6fafb] p-2 text-center">
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

      <div class="mt-auto flex items-center justify-between border-t border-border pt-4">
        <div class="min-w-0 flex items-center gap-2">
          <div class="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-[10px] font-bold text-primary">CF</div>
          <span class="truncate text-sm text-muted-foreground">{{ provider }}</span>
        </div>
        <ChevronRight class="h-5 w-5 shrink-0 text-muted-foreground transition-transform group-hover:translate-x-1 group-hover:text-primary" />
      </div>
    </div>
  </component>

  <PurchaseDialog v-model:open="showPurchaseDialog" :course-name="title" :pipeline-id="id" />
</template>
