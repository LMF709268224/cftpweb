<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { GraduationCap, Search } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import CourseCard from "@/components/CourseCard.vue"
import PageFeedback from "@/components/PageFeedback.vue"
import { apiClient } from "@/lib/apiClient"
import { formatCurrencyMinorAmount } from "@/lib/display"
import { useTranslation } from "@/lib/language"
import { preloadCheckoutWizard } from "@/router"

const { t, lang } = useTranslation()
type CourseCategoryFilter = "all" | "certification" | "bundle" | "membership"

const searchQuery = ref("")
const activeCategory = ref<CourseCategoryFilter>("all")
const refreshKey = ref(0)
const allCourses = ref<any[]>([])
const loading = ref(false)
const loadError = ref(false)
let paymentPollInterval: number | undefined
let checkoutPreloadTimer: number | undefined

function clearPaymentPollInterval() {
  if (paymentPollInterval === undefined) return
  window.clearInterval(paymentPollInterval)
  paymentPollInterval = undefined
}

const emptyCopy = computed(() => t.value.courses)
const categoryOptions = computed<Array<{ key: CourseCategoryFilter; label: string }>>(() => [
  { key: "all", label: t.value.courses.categoryAll },
  { key: "certification", label: t.value.courses.categoryCertification },
  { key: "bundle", label: t.value.courses.categoryBundle },
  { key: "membership", label: t.value.courses.categoryMembership },
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
  return formatCurrencyMinorAmount(amount, currency) || ""
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
  const showPageError = allCourses.value.length === 0
  loading.value = true
  if (showPageError) loadError.value = false
  try {
    const res = await apiClient("/api/mall/bundles")
    if (!Array.isArray(res?.bundles)) throw new Error("MALL_BUNDLES_INVALID_RESPONSE")
    const bundles = res.bundles
    allCourses.value = await Promise.all(bundles.map(async (b: any) => {
      const stages = Array.isArray(b?.stages) ? b.stages : []
      const itemTypes = bundleItemTypes(b)
      const pipelineBundle = isPipelineBundle(b, itemTypes)
      const membershipBundle = isMembershipBundle(b, itemTypes)
      const unitCount = stages.reduce((total: number, stage: any) => total + (Array.isArray(stage?.units) ? stage.units.length : 0), 0)
      const awardCertCount = Array.isArray(b?.award_certs) ? b.award_certs.length : 0
      const firstStageNames = stages.slice(0, 2).map((stage: any) => stage?.name).filter(Boolean).join(" / ")
      return {
        id: b.bundle_id,
        pipelineId: pipelineBundle ? b.pipeline_id : "",
        membershipId: membershipBundle ? b.membership_id : "",
        membershipGpath: membershipBundle ? b.membership_gpath : "",
        membershipRequiredCredRespaths: membershipBundle && Array.isArray(b?.membership_required_cred_respaths)
          ? b.membership_required_cred_respaths
          : [],
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
        stages,
        stats: [
          { label: t.value.courses.stages, value: stages.length },
          { label: t.value.courses.units, value: unitCount },
          { label: t.value.courses.awardedCertificates, value: awardCertCount },
        ],
      }
    }))
    loadError.value = false
  } catch (error) {
    console.error(error)
    if (showPageError) loadError.value = true
    toast.error(t.value.common.error)
  } finally {
    loading.value = false
  }
}

function handlePaymentReturn() {
  const url = new URL(window.location.href)
  const paymentStatus = url.searchParams.get("payment_status")
  if (!paymentStatus) return

  const purchasedPipelineId = url.searchParams.get("pipeline_id")
  const purchasedBundleId = url.searchParams.get("bundle_id")
  const targetId = purchasedBundleId || purchasedPipelineId
  const copy = t.value.paymentReturnHandler || {}

  if (paymentStatus === "success") {
    toast.success(copy.purchaseSuccess)
    if (purchasedPipelineId && targetId) {
      allCourses.value = allCourses.value.map((course) =>
        course.id === targetId || course.pipelineId === purchasedPipelineId ? { ...course, eligibilityRefreshKey: Date.now() } : course,
      )
    }

    // Start a short polling loop to wait for the Stripe webhook to update the backend order status
    let attempts = 0
    clearPaymentPollInterval()
    paymentPollInterval = window.setInterval(() => {
      attempts++
      void fetchData().then(() => {
        const targetCourse = allCourses.value.find((c) => c.id === targetId || c.pipelineId === purchasedPipelineId)
        // If the active order has cleared, the webhook has finished processing
        if (targetCourse && !targetCourse.activeOrder) {
          clearPaymentPollInterval()
        }
      })
      if (attempts >= 6) clearPaymentPollInterval()
    }, 2500)
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
  void fetchData().finally(() => {
    checkoutPreloadTimer = window.setTimeout(() => void preloadCheckoutWizard(), 500)
  })
})

onBeforeUnmount(() => {
  clearPaymentPollInterval()
  if (checkoutPreloadTimer !== undefined) window.clearTimeout(checkoutPreloadTimer)
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
        <div class="courses-page-intro mb-6">
          <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.courses.title }}</h1>
          <p class="mt-2 text-muted-foreground">{{ t.courses.subtitle }}</p>
        </div>

        <div class="courses-page-controls mb-4 flex flex-col gap-3 rounded-[16px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)] lg:flex-row lg:items-center lg:justify-between">
          <div class="courses-category-scroll overflow-x-auto">
            <div class="courses-category-filter inline-flex min-w-max rounded-lg bg-[#f6fafb] p-1">
              <button
                v-for="option in categoryOptions"
                :key="option.key"
                type="button"
                :class="[
                  'courses-category-option h-9 rounded-md px-3 text-sm font-semibold transition-colors',
                  activeCategory === option.key ? 'bg-primary text-white shadow-sm shadow-primary/20' : 'text-muted-foreground hover:bg-white hover:text-foreground',
                ]"
                @click="activeCategory = option.key"
              >
                {{ option.label }}
              </button>
            </div>
          </div>

          <div class="courses-search relative w-full lg:ml-auto lg:max-w-md">
            <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <input v-model="searchQuery" class="input pl-10" :placeholder="t.courses.searchPlaceholder" />
          </div>
        </div>

        <PageFeedback v-if="loading && allCourses.length === 0" kind="loading" :loading-label="t.common.loading" />

        <PageFeedback
          v-else-if="loadError"
          kind="error"
          :title="emptyCopy.loadFailed"
          :description="emptyCopy.loadFailedDesc"
          :action-label="emptyCopy.retry"
          @action="fetchData"
        />

        <div v-else-if="filteredCourses.length > 0" class="courses-page-grid grid gap-4 sm:grid-cols-2 2xl:grid-cols-3">
          <CourseCard v-for="course in filteredCourses" :key="`${course.id}-${course.eligibilityRefreshKey || 0}`" v-bind="course" />
        </div>

        <PageFeedback
          v-else
          kind="empty"
          :title="searchQuery.trim() || activeCategory !== 'all' ? emptyCopy.noSearchTitle : emptyCopy.noAvailableTitle"
          :description="searchQuery.trim() || activeCategory !== 'all' ? emptyCopy.noSearchDesc : emptyCopy.noAvailableDesc"
        >
          <template #icon><Search class="h-8 w-8" /></template>
          <template v-if="searchQuery.trim() || activeCategory !== 'all'" #action>
            <button class="btn btn-primary mt-5 rounded-lg shadow-sm shadow-primary/20" @click="searchQuery = ''; activeCategory = 'all'">
              {{ emptyCopy.clearSearch }}
            </button>
          </template>
        </PageFeedback>
      </main>
    </div>
  </AppShell>
</template>

<style scoped>
@media (min-width: 1024px) and (max-width: 1399px) {
  .courses-page-controls {
    flex-wrap: wrap;
  }

  .courses-category-scroll {
    flex: 0 0 auto;
    min-width: 0;
    overflow-x: visible;
  }

  .courses-category-filter {
    width: auto;
    min-width: 0;
  }

  :global(html[lang="en"] .courses-page-controls .courses-category-option) {
    min-width: 0;
    padding-inline: 8px;
    font-size: 13px;
    white-space: nowrap;
  }

  .courses-search {
    width: clamp(180px, 28vw, 320px);
    max-width: 320px;
    min-width: 180px;
    flex: 0 1 320px;
  }

  :global(html[lang="en"] .courses-page-controls .courses-search) {
    width: 220px;
    max-width: 220px;
    flex-basis: 220px;
  }
}

@media (max-width: 767px) {
  .courses-page-intro {
    margin-bottom: 16px;
  }

  .courses-page-controls {
    gap: 8px;
    margin-bottom: 12px;
    padding: 8px;
  }

  .courses-category-scroll {
    overflow-x: hidden;
  }

  .courses-category-filter {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    width: 100%;
    min-width: 0;
  }

  .courses-category-option {
    min-width: 0;
    height: 32px;
    padding-inline: 4px;
    font-size: 11px;
    line-height: 1;
  }

  .courses-page-grid {
    gap: 12px;
  }
}
</style>
