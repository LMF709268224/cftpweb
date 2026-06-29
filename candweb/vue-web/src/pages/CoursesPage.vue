<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { BadgeCheck, Boxes, Clock, GraduationCap, Search, ShieldCheck, UserRoundCheck } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import CourseCard from "@/components/CourseCard.vue"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"

const { t, lang } = useTranslation()
type CourseCategoryFilter = "all" | "certification" | "bundle" | "membership"

const searchQuery = ref("")
const activeCategory = ref<CourseCategoryFilter>("all")
const refreshKey = ref(0)
const allCourses = ref<any[]>([])
const loading = ref(false)

const emptyCopy = computed(() => t.value.courses)
const categoryOptions = computed<Array<{ key: CourseCategoryFilter; label: string; icon: any }>>(() => [
  { key: "all", label: t.value.courses.categoryAll, icon: Boxes },
  { key: "certification", label: t.value.courses.categoryCertification, icon: BadgeCheck },
  { key: "bundle", label: t.value.courses.categoryBundle, icon: ShieldCheck },
  { key: "membership", label: t.value.courses.categoryMembership, icon: UserRoundCheck },
])

function courseCategory(course: any): Exclude<CourseCategoryFilter, "all"> | "other" {
  if (course.isPipelineBundle && course.isMembershipBundle) return "bundle"
  if (course.isPipelineBundle) return "certification"
  if (course.isMembershipBundle) return "membership"
  return "other"
}

const filteredCourses = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  return allCourses.value.filter((course) => {
    const matchesCategory = activeCategory.value === "all" || courseCategory(course) === activeCategory.value
    const matchesSearch = !keyword ||
      course.title.toLowerCase().includes(keyword) ||
      course.description.toLowerCase().includes(keyword)
    return matchesCategory && matchesSearch
  })
})

function certificationDisplayName(value?: string) {
  return String(value || "").replace(/\bPipeline\b/g, "Certification").replace(/管线/g, "认证")
}

function formatDisplayAmount(amount: number, currency = "USD") {
  const normalizedCurrency = String(currency || "USD").trim().toUpperCase()
  try {
    return new Intl.NumberFormat(undefined, { style: "currency", currency: normalizedCurrency }).format(amount / 100)
  } catch {
    return `${normalizedCurrency} ${(amount / 100).toLocaleString()}`
  }
}

function bundlePriceLabel(bundle: any) {
  const min = Number(bundle?.display_amount_min || 0)
  const max = Number(bundle?.display_amount_max || 0)
  const currency = String(bundle?.display_currency || "USD").trim()
  if (min <= 0 && max <= 0) return ""
  if (max > 0 && max !== min) return `${formatDisplayAmount(min, currency)} - ${formatDisplayAmount(max, currency)}`
  return formatDisplayAmount(min || max, currency)
}

function normalizeBundleItemType(value: unknown) {
  return String(value || "").trim().toLowerCase().replace(/-/g, "_")
}

function bundleItemTypes(bundle: any) {
  const fromBackend = Array.isArray(bundle?.bundle_item_types)
    ? bundle.bundle_item_types.map(normalizeBundleItemType).filter(Boolean)
    : []
  const fromItemsJson: string[] = []
  try {
    const parsed = JSON.parse(String(bundle?.items_json || ""))
    const items = Array.isArray(parsed) ? parsed : [parsed]
    for (const item of items) {
      if (item && typeof item === "object") {
        const type = normalizeBundleItemType(item.item_type || item.type || item.itemType || item.kind)
        if (type) fromItemsJson.push(type)
      }
    }
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      if (Array.isArray(parsed.pipelines)) fromItemsJson.push("pipeline")
      if (Array.isArray(parsed.memberships)) fromItemsJson.push("membership")
    }
  } catch {
    // items_json is optional and legacy bundles may not have a typed payload.
  }
  return Array.from(new Set([...fromBackend, ...fromItemsJson]))
}

function isPipelineBundle(bundle: any, itemTypes: string[]) {
  if (bundle?.is_pipeline_bundle === true) return true
  return itemTypes.some((type) => type.includes("pipeline"))
}

function isMembershipBundle(bundle: any, itemTypes: string[]) {
  if (bundle?.is_membership_bundle === true) return true
  return itemTypes.some((type) => type.includes("membership"))
}

async function fetchData() {
  loading.value = true
  try {
    const res = await apiClient("/api/mall/bundles")
    const bundles = Array.isArray(res?.bundles) ? res.bundles : []
    allCourses.value = await Promise.all(bundles.map(async (b: any) => {
      const stages = Array.isArray(b?.stages) ? b.stages : []
      const itemTypes = bundleItemTypes(b)
      const pipelineBundle = isPipelineBundle(b, itemTypes)
      const membershipBundle = isMembershipBundle(b, itemTypes)
      const unitCount = stages.reduce((total: number, stage: any) => total + (Array.isArray(stage?.units) ? stage.units.length : 0), 0)
      const finalQualCount = Array.isArray(b?.final_quals) ? b.final_quals.length : 0
      const firstStageNames = stages.slice(0, 2).map((stage: any) => stage?.name).filter(Boolean).join(" / ")
      return {
        id: b.bundle_id,
        pipelineId: pipelineBundle ? b.pipeline_id : "",
        membershipId: membershipBundle ? b.membership_id : "",
        membershipGpath: membershipBundle ? b.membership_gpath : "",
        itemTypes,
        isPipelineBundle: pipelineBundle,
        isMembershipBundle: membershipBundle,
        title: certificationDisplayName(b.name) || t.value.common.unknownCourse,
        description: String(b.description || "").trim() || firstStageNames || `${stages.length} ${t.value.courses.stages} / ${unitCount} ${t.value.courses.units}`,
        provider: b.category_tips || t.value.courses.certificationPath,
        isPurchased: false,
        image: typeof b?.thumbnail_url === "string" ? b.thumbnail_url : "",
        priceLabel: bundlePriceLabel(b),
        students: typeof b.purchase_count === "number" ? b.purchase_count : undefined,
        versionLabel: `${t.value.courses.version} ${b.version || 0}`,
        eligibility: b?.eligibility || null,
        activeOrder: b?.purchase_state?.active_order || b?.active_order || null,
        paymentPreview: b?.purchase_state?.payment_preview || b?.payment_preview || null,
        exemptionOptions: b?.purchase_state?.exemption_options || b?.exemption_options || null,
        activeMembership: b?.active_membership || null,
        stats: [
          { label: t.value.courses.stages, value: stages.length },
          { label: t.value.courses.units, value: unitCount },
          { label: t.value.courses.finalQualifications, value: finalQualCount },
        ],
      }
    }))
  } catch (error) {
    console.error(error)
    toast.error(t.value.common.error)
  } finally {
    loading.value = false
  }
}

function handlePaymentReturn() {
  const url = new URL(window.location.href)
  const paymentStatus = url.searchParams.get("payment_status")
  if (!paymentStatus) return

  const paymentAction = url.searchParams.get("payment_action")
  const purchasedPipelineId = url.searchParams.get("pipeline_id")
  const purchasedBundleId = url.searchParams.get("bundle_id")
  const targetId = purchasedBundleId || purchasedPipelineId
  const isUnlock = paymentAction === "unlock"
  const copy = t.value.paymentReturnHandler || {}

  if (paymentStatus === "success") {
    toast.success(isUnlock ? copy.unlockSuccess : copy.purchaseSuccess)
    if (!isUnlock && purchasedPipelineId && targetId) {
      allCourses.value = allCourses.value.map((course) =>
        course.id === targetId || course.pipelineId === purchasedPipelineId ? { ...course, eligibilityRefreshKey: Date.now() } : course,
      )
    }
  } else if (paymentStatus === "cancelled") {
    toast.warning(copy.cancelled)
  } else if (paymentStatus === "failed") {
    toast.error(copy.failed)
  }

  localStorage.removeItem("pending_mall_payment")
  refreshKey.value += 1
  url.searchParams.delete("payment_status")
  url.searchParams.delete("payment_action")
  url.searchParams.delete("order_id")
  url.searchParams.delete("pipeline_id")
  url.searchParams.delete("bundle_id")
  window.history.replaceState({}, "", `${url.pathname}${url.search}${url.hash}`)
}

watch([refreshKey, lang], () => {
  searchQuery.value = ""
  void fetchData()
})

onMounted(() => {
  handlePaymentReturn()
  void fetchData()
})
</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <GraduationCap class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ t.courses.title }}</span>
      </header>

      <main class="px-5 py-8 md:px-8 lg:px-10">
        <div class="mb-6">
          <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.courses.title }}</h1>
          <p class="mt-2 text-muted-foreground">{{ t.courses.subtitle }}</p>
        </div>

        <div class="mb-4 flex flex-col gap-4 rounded-[16px] bg-white px-5 pt-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)] lg:flex-row lg:items-start lg:justify-between lg:px-6">
          <div class="overflow-x-auto">
            <div class="flex min-w-max gap-x-8 gap-y-2">
              <button
                v-for="option in categoryOptions"
                :key="option.key"
                type="button"
                :class="[
                  'relative inline-flex h-12 cursor-pointer items-center gap-2 whitespace-nowrap px-1 pb-4 text-base font-medium transition-colors duration-200',
                  activeCategory === option.key ? 'text-primary' : 'text-[#111827] hover:text-primary',
                ]"
                @click="activeCategory = option.key"
              >
                <component :is="option.icon" class="h-4 w-4" />
                {{ option.label }}
                <span v-if="activeCategory === option.key" class="absolute bottom-[-1px] left-0 h-0.5 w-full rounded-full bg-primary" />
              </button>
            </div>
          </div>

          <div class="relative w-full pb-4 lg:ml-auto lg:max-w-md">
            <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <input v-model="searchQuery" class="input pl-10" :placeholder="t.courses.searchPlaceholder" />
          </div>
        </div>

        <div v-if="loading && allCourses.length === 0" class="flex items-center justify-center gap-2 rounded-[16px] bg-white py-14 text-muted-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
          <Clock class="h-5 w-5 animate-spin" /> <span>{{ t.common.loading }}</span>
        </div>

        <div v-else-if="filteredCourses.length > 0" class="grid gap-4 sm:grid-cols-2 2xl:grid-cols-3">
          <CourseCard v-for="course in filteredCourses" :key="`${course.id}-${course.eligibilityRefreshKey || 0}`" v-bind="course" />
        </div>

        <div v-else class="flex flex-col items-center justify-center rounded-[16px] bg-white py-16 text-center shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
          <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10">
            <Search class="h-8 w-8 text-primary" />
          </div>
          <h3 class="mb-2 text-lg font-semibold text-foreground">{{ searchQuery.trim() || activeCategory !== 'all' ? emptyCopy.noSearchTitle : emptyCopy.noAvailableTitle }}</h3>
          <p class="mx-auto max-w-md text-sm leading-6 text-muted-foreground">{{ searchQuery.trim() || activeCategory !== 'all' ? emptyCopy.noSearchDesc : emptyCopy.noAvailableDesc }}</p>
          <button v-if="searchQuery.trim() || activeCategory !== 'all'" class="btn btn-primary mt-5 rounded-lg shadow-sm shadow-primary/20" @click="searchQuery = ''; activeCategory = 'all'">
            {{ emptyCopy.clearSearch }}
          </button>
        </div>
      </main>
    </div>
  </AppShell>
</template>
