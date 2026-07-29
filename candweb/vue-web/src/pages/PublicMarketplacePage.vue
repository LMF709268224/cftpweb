<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { Clock, GraduationCap, Search } from "lucide-vue-next"
import CourseCard from "@/components/CourseCard.vue"
import GfiFooter from "@/components/GfiFooter.vue"
import GfiHeader from "@/components/GfiHeader.vue"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"

type ProductCategory = "all" | "certification" | "bundle" | "membership"

const { t, lang } = useTranslation()
const searchQuery = ref("")
const activeCategory = ref<ProductCategory>("all")
const allCourses = ref<any[]>([])
const loading = ref(false)
const loadError = ref(false)

const categoryOptions = computed<Array<{ key: ProductCategory; label: string }>>(() => [
  { key: "all", label: t.value.courses.categoryAll },
  { key: "certification", label: t.value.courses.categoryCertification },
  { key: "bundle", label: t.value.courses.categoryBundle },
  { key: "membership", label: t.value.courses.categoryMembership },
])

function courseCategory(course: any): Exclude<ProductCategory, "all"> | "other" {
  if (course.isPipelineBundle && course.isMembershipBundle) return "bundle"
  if (course.isPipelineBundle) return "certification"
  if (course.isMembershipBundle) return "membership"
  return "other"
}

const filteredCourses = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  return allCourses.value.filter((course) => {
    const matchesCategory = activeCategory.value === "all" || courseCategory(course) === activeCategory.value
    const matchesSearch = !keyword || course.title.toLowerCase().includes(keyword) || course.description.toLowerCase().includes(keyword)
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
  loadError.value = false
  try {
    const response = await apiClient("/api/public/mall/bundles", { suppressErrorToast: true })
    const bundles = Array.isArray(response?.bundles) ? response.bundles : []
    allCourses.value = bundles.map((bundle: any) => {
      const stages = Array.isArray(bundle?.stages) ? bundle.stages : []
      const itemTypes = bundleItemTypes(bundle)
      const pipelineBundle = isPipelineBundle(bundle, itemTypes)
      const membershipBundle = isMembershipBundle(bundle, itemTypes)
      const unitCount = stages.reduce((total: number, stage: any) => total + (Array.isArray(stage?.units) ? stage.units.length : 0), 0)
      const finalQualCount = Array.isArray(bundle?.final_quals) ? bundle.final_quals.length : 0
      const firstStageNames = stages.slice(0, 2).map((stage: any) => stage?.name).filter(Boolean).join(" / ")
      return {
        id: bundle.bundle_id,
        pipelineId: pipelineBundle ? bundle.pipeline_id : "",
        membershipId: membershipBundle ? bundle.membership_id : "",
        membershipGpath: membershipBundle ? bundle.membership_gpath : "",
        itemTypes,
        isPipelineBundle: pipelineBundle,
        isMembershipBundle: membershipBundle,
        title: certificationDisplayName(bundle.name) || t.value.common.unknownCourse,
        description: String(bundle.description || "").trim() || firstStageNames || `${stages.length} ${t.value.courses.stages} / ${unitCount} ${t.value.courses.units}`,
        provider: bundle.category_tips || t.value.courses.certificationPath,
        isPurchased: false,
        image: typeof bundle?.thumbnail_url === "string" ? bundle.thumbnail_url : "",
        priceLabel: bundlePriceLabel(bundle),
        students: typeof bundle.purchase_count === "number" ? bundle.purchase_count : undefined,
        versionLabel: `${t.value.courses.version} ${bundle.version || 0}`,
        stats: [
          { label: t.value.courses.stages, value: stages.length },
          { label: t.value.courses.units, value: unitCount },
          { label: t.value.courses.finalQualifications, value: finalQualCount },
        ],
      }
    })
  } catch (error) {
    console.error("Failed to load public marketplace products", error)
    allCourses.value = []
    loadError.value = true
  } finally {
    loading.value = false
  }
}

watch(lang, () => {
  searchQuery.value = ""
  void fetchData()
})

onMounted(() => void fetchData())
</script>

<template>
  <div class="public-marketplace">
    <GfiHeader auth-target="/login" :auth-new-tab="false" />

    <main class="public-marketplace-main">
      <div class="page-panel public-marketplace-panel">
        <header class="flex h-16 items-center border-b border-border bg-white px-5">
          <GraduationCap class="mr-4 h-4 w-4 text-slate-700" />
          <span class="text-sm font-medium text-foreground">{{ t.courses.title }}</span>
        </header>

        <main class="px-5 py-8 md:px-8 lg:px-10">
          <div class="mb-6">
            <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.courses.title }}</h1>
            <p class="mt-2 text-muted-foreground">{{ t.courses.subtitle }}</p>
          </div>

          <div class="mb-4 flex flex-col gap-3 rounded-[16px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)] lg:flex-row lg:items-center lg:justify-between">
            <div class="overflow-x-auto">
              <div class="inline-flex min-w-max rounded-lg bg-[#f6fafb] p-1">
                <button
                  v-for="option in categoryOptions"
                  :key="option.key"
                  type="button"
                  :class="[
                    'h-9 rounded-md px-3 text-sm font-semibold transition-colors',
                    activeCategory === option.key ? 'bg-primary text-white shadow-sm shadow-primary/20' : 'text-muted-foreground hover:bg-white hover:text-foreground',
                  ]"
                  @click="activeCategory = option.key"
                >
                  {{ option.label }}
                </button>
              </div>
            </div>

            <div class="relative w-full lg:ml-auto lg:max-w-md">
              <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <input v-model="searchQuery" class="input pl-10" :placeholder="t.courses.searchPlaceholder" />
            </div>
          </div>

          <div v-if="loading && allCourses.length === 0" class="flex items-center justify-center gap-2 rounded-[16px] bg-white py-14 text-muted-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
            <Clock class="h-5 w-5 animate-spin" /> <span>{{ t.common.loading }}</span>
          </div>

          <div v-else-if="loadError" class="flex flex-col items-center justify-center rounded-[16px] bg-white py-16 text-center shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
            <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.common.error }}</h3>
            <button class="btn btn-primary mt-4 rounded-lg shadow-sm shadow-primary/20" type="button" @click="fetchData">
              {{ lang === "zh" ? "重新加载" : "Reload" }}
            </button>
          </div>

          <div v-else-if="filteredCourses.length > 0" class="grid gap-4 sm:grid-cols-2 2xl:grid-cols-3">
            <CourseCard v-for="course in filteredCourses" :key="course.id" v-bind="course" login-required />
          </div>

          <div v-else class="flex flex-col items-center justify-center rounded-[16px] bg-white py-16 text-center shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
            <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10">
              <Search class="h-8 w-8 text-primary" />
            </div>
            <h3 class="mb-2 text-lg font-semibold text-foreground">{{ searchQuery.trim() || activeCategory !== 'all' ? t.courses.noSearchTitle : t.courses.noAvailableTitle }}</h3>
            <p class="mx-auto max-w-md text-sm leading-6 text-muted-foreground">{{ searchQuery.trim() || activeCategory !== 'all' ? t.courses.noSearchDesc : t.courses.noAvailableDesc }}</p>
            <button v-if="searchQuery.trim() || activeCategory !== 'all'" class="btn btn-primary mt-5 rounded-lg shadow-sm shadow-primary/20" @click="searchQuery = ''; activeCategory = 'all'">
              {{ t.courses.clearSearch }}
            </button>
          </div>
        </main>
      </div>
    </main>

    <GfiFooter />
  </div>
</template>

<style scoped>
.public-marketplace {
  min-height: 100vh;
  background: #f9fafb;
}

.public-marketplace-main {
  width: 100%;
  box-sizing: border-box;
  padding: .5rem;
}

.public-marketplace-panel {
  min-height: calc(100vh - 116px);
}

@media (max-width: 1023px) {
  .public-marketplace-main { padding: 0; }
}
</style>
