<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useRouter } from "vue-router"
import { AlertCircle, ArrowRight, BookOpen, Clock, Search } from "lucide-vue-next"
import GfiFooter from "@/components/GfiFooter.vue"
import GfiHeader from "@/components/GfiHeader.vue"
import { apiClient } from "@/lib/apiClient"
import { isAuthenticated } from "@/lib/authStorage"
import { useTranslation } from "@/lib/language"

type ProductCategory = "all" | "certification" | "bundle" | "membership"

type PublicCourse = {
  id: string
  title: string
  description: string
  provider: string
  image: string
  priceLabel: string
  isPipelineBundle: boolean
  isMembershipBundle: boolean
}

const router = useRouter()
const { t, lang } = useTranslation()
const searchQuery = ref("")
const activeCategory = ref<ProductCategory>("all")
const allCourses = ref<PublicCourse[]>([])
const loading = ref(false)
const loadError = ref(false)
const authenticated = ref(isAuthenticated())

const pageCopy = computed(() => lang.value === "zh"
  ? {
      eyebrow: "GFI 专业认证",
      title: "通过专业认证，迈向职业新高度",
      subtitle: "获得全球认可的金融科技认证，展现您的专业能力。",
      oneTimeFee: "认证及相关服务费用",
      loginAction: "登录后选购",
      memberAction: "进入商城",
      loadErrorTitle: "商品暂时无法加载",
      loadErrorDesc: "请稍后重试，或登录后进入商城查看。",
      retry: "重新加载",
    }
  : {
      eyebrow: "GFI Professional Credentials",
      title: "Advance Your Career with Professional Certifications",
      subtitle: "Earn globally recognized fintech credentials and demonstrate your expertise.",
      oneTimeFee: "Certification and related service fee",
      loginAction: "Log In to Purchase",
      memberAction: "Enter Marketplace",
      loadErrorTitle: "Products are temporarily unavailable",
      loadErrorDesc: "Please try again shortly or log in to view the member marketplace.",
      retry: "Reload",
    })

const categoryOptions = computed<Array<{ key: ProductCategory; label: string }>>(() => [
  { key: "all", label: t.value.courses.categoryAll },
  { key: "certification", label: t.value.courses.categoryCertification },
  { key: "bundle", label: t.value.courses.categoryBundle },
  { key: "membership", label: t.value.courses.categoryMembership },
])

function courseCategory(course: PublicCourse): Exclude<ProductCategory, "all"> | "other" {
  if (course.isPipelineBundle && course.isMembershipBundle) return "bundle"
  if (course.isPipelineBundle) return "certification"
  if (course.isMembershipBundle) return "membership"
  return "other"
}

function categoryLabel(course: PublicCourse) {
  const category = courseCategory(course)
  return categoryOptions.value.find((option) => option.key === category)?.label || course.provider
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
      const firstStageNames = stages.slice(0, 2).map((stage: any) => stage?.name).filter(Boolean).join(" / ")
      return {
        id: bundle.bundle_id,
        title: certificationDisplayName(bundle.name) || t.value.common.unknownCourse,
        description: String(bundle.description || "").trim() || firstStageNames || `${stages.length} ${t.value.courses.stages} / ${unitCount} ${t.value.courses.units}`,
        provider: bundle.category_tips || t.value.courses.certificationPath,
        image: typeof bundle?.thumbnail_url === "string" ? bundle.thumbnail_url : "",
        priceLabel: bundlePriceLabel(bundle),
        isPipelineBundle: pipelineBundle,
        isMembershipBundle: membershipBundle,
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

function syncAuthentication() {
  authenticated.value = isAuthenticated()
}

function openCourse() {
  syncAuthentication()
  void router.push(authenticated.value ? "/certifications" : "/login")
}

watch(lang, () => {
  searchQuery.value = ""
  void fetchData()
})

onMounted(() => {
  syncAuthentication()
  window.addEventListener("storage", syncAuthentication)
  window.addEventListener("focus", syncAuthentication)
  void fetchData()
})

onBeforeUnmount(() => {
  window.removeEventListener("storage", syncAuthentication)
  window.removeEventListener("focus", syncAuthentication)
})
</script>

<template>
  <div class="public-marketplace">
    <GfiHeader auth-target="/login" :auth-new-tab="false" />

    <main>
      <section class="marketplace-hero">
        <div class="marketplace-hero-inner">
          <p class="marketplace-eyebrow">{{ pageCopy.eyebrow }}</p>
          <h1>{{ pageCopy.title }}</h1>
          <p class="marketplace-subtitle">{{ pageCopy.subtitle }}</p>
        </div>
      </section>

      <section class="marketplace-catalog" :aria-label="t.courses.title">
        <div class="catalog-toolbar">
          <div class="category-filter" role="group" :aria-label="t.courses.title">
            <button
              v-for="option in categoryOptions"
              :key="option.key"
              type="button"
              :class="{ active: activeCategory === option.key }"
              @click="activeCategory = option.key"
            >
              {{ option.label }}
            </button>
          </div>

          <label class="catalog-search">
            <Search aria-hidden="true" />
            <input v-model="searchQuery" :placeholder="t.courses.searchPlaceholder" />
          </label>
        </div>

        <div v-if="loading && allCourses.length === 0" class="product-grid" aria-live="polite">
          <article v-for="index in 2" :key="index" class="product-card product-card--loading">
            <div class="skeleton product-visual-skeleton" />
            <div class="product-body">
              <div class="skeleton skeleton-line skeleton-line--short" />
              <div class="skeleton skeleton-line skeleton-line--title" />
              <div class="skeleton skeleton-line" />
              <div class="skeleton skeleton-line" />
              <div class="skeleton skeleton-button" />
            </div>
          </article>
        </div>

        <div v-else-if="loadError" class="catalog-state" role="alert">
          <span class="catalog-state-icon"><AlertCircle /></span>
          <div>
            <h2>{{ pageCopy.loadErrorTitle }}</h2>
            <p>{{ pageCopy.loadErrorDesc }}</p>
          </div>
          <button type="button" @click="fetchData">
            <Clock /> {{ pageCopy.retry }}
          </button>
        </div>

        <div v-else-if="filteredCourses.length > 0" class="product-grid">
          <article v-for="course in filteredCourses" :key="course.id" class="product-card">
            <div class="product-visual">
              <span class="product-category">{{ categoryLabel(course) }}</span>
              <img v-if="course.image" :src="course.image" :alt="course.title" />
              <div v-else class="product-placeholder" aria-hidden="true">
                <BookOpen />
                <span>GFI</span>
              </div>
            </div>

            <div class="product-body">
              <div class="product-copy">
                <h2>{{ course.title }}</h2>
                <p>{{ course.description }}</p>
              </div>
              <div class="product-price">
                <strong>{{ course.priceLabel || t.courseCard.free }}</strong>
                <span>{{ pageCopy.oneTimeFee }}</span>
              </div>
              <button class="product-action" type="button" @click="openCourse">
                <span>{{ authenticated ? pageCopy.memberAction : pageCopy.loginAction }}</span>
                <ArrowRight />
              </button>
            </div>
          </article>
        </div>

        <div v-else class="catalog-state catalog-state--empty">
          <span class="catalog-state-icon"><Search /></span>
          <div>
            <h2>{{ searchQuery.trim() || activeCategory !== "all" ? t.courses.noSearchTitle : t.courses.noAvailableTitle }}</h2>
            <p>{{ searchQuery.trim() || activeCategory !== "all" ? t.courses.noSearchDesc : t.courses.noAvailableDesc }}</p>
          </div>
          <button v-if="searchQuery.trim() || activeCategory !== 'all'" type="button" @click="searchQuery = ''; activeCategory = 'all'">
            {{ t.courses.clearSearch }}
          </button>
        </div>
      </section>
    </main>

    <GfiFooter />
  </div>
</template>

<style scoped>
.public-marketplace {
  min-height: 100vh;
  overflow-x: hidden;
  color: #101828;
  background: #f6f8fb;
  font-family: Inter, "PingFang SC", "Microsoft YaHei", Arial, sans-serif;
}

.public-marketplace * {
  box-sizing: border-box;
}

.marketplace-hero {
  border-bottom: 1px solid #e4e8ef;
  background: #fff;
}

.marketplace-hero-inner {
  width: min(1120px, calc(100% - 64px));
  margin: 0 auto;
  padding: 54px 0 38px;
  text-align: center;
}

.marketplace-eyebrow {
  margin: 0 0 12px;
  color: #225edf;
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0;
  text-transform: uppercase;
}

.marketplace-hero h1 {
  max-width: 820px;
  margin: 0 auto;
  overflow-wrap: anywhere;
  color: #0d1730;
  font-size: 34px;
  line-height: 1.25;
  letter-spacing: 0;
}

.marketplace-subtitle {
  max-width: 720px;
  margin: 16px auto 0;
  overflow-wrap: anywhere;
  color: #5c667a;
  font-size: 16px;
  line-height: 1.75;
}

.marketplace-catalog {
  width: min(1120px, calc(100% - 64px));
  min-height: 540px;
  margin: 0 auto;
  padding: 30px 0 68px;
}

.catalog-toolbar {
  display: flex;
  margin-bottom: 28px;
  padding-bottom: 20px;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  border-bottom: 1px solid #dfe4ec;
}

.category-filter {
  display: inline-flex;
  min-width: 0;
  padding: 3px;
  overflow-x: auto;
  border: 1px solid #d7deea;
  border-radius: 6px;
  background: #fff;
}

.category-filter button {
  min-height: 36px;
  padding: 0 16px;
  border: 0;
  border-radius: 4px;
  color: #44506a;
  background: transparent;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
  white-space: nowrap;
  transition: color .2s ease, background .2s ease;
}

.category-filter button:hover {
  color: #174eb9;
  background: #eef4ff;
}

.category-filter button.active {
  color: #fff;
  background: #1e55b3;
}

.catalog-search {
  display: flex;
  width: min(340px, 100%);
  height: 44px;
  flex: 0 1 340px;
  align-items: center;
  gap: 10px;
  padding: 0 14px;
  border: 1px solid #ccd5e3;
  border-radius: 6px;
  background: #fff;
  transition: border-color .2s ease, box-shadow .2s ease;
}

.catalog-search:focus-within {
  border-color: #2864d7;
  box-shadow: 0 0 0 3px rgba(40, 100, 215, .12);
}

.catalog-search svg {
  width: 18px;
  height: 18px;
  flex: none;
  color: #69758a;
}

.catalog-search input {
  width: 100%;
  min-width: 0;
  border: 0;
  outline: 0;
  color: #17213a;
  background: transparent;
  font-size: 14px;
}

.product-grid {
  display: grid;
  width: min(920px, 100%);
  margin: 0 auto;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 30px;
}

.product-card {
  display: flex;
  min-width: 0;
  min-height: 570px;
  overflow: hidden;
  border: 1px solid #d9dee7;
  border-radius: 6px;
  flex-direction: column;
  background: #fff;
  box-shadow: 0 2px 5px rgba(18, 38, 75, .06);
  transition: transform .2s ease, border-color .2s ease, box-shadow .2s ease;
}

.product-card:not(.product-card--loading):hover {
  border-color: #aebbd0;
  box-shadow: 0 12px 30px rgba(18, 38, 75, .1);
  transform: translateY(-2px);
}

.product-visual {
  position: relative;
  display: flex;
  height: 254px;
  padding: 38px;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-bottom: 1px solid #e5e8ee;
  background: #fff;
}

.product-visual img {
  width: min(320px, 86%);
  max-height: 135px;
  object-fit: contain;
}

.product-category {
  position: absolute;
  top: 18px;
  left: 20px;
  max-width: calc(100% - 40px);
  overflow: hidden;
  color: #41608f;
  font-size: 12px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.product-placeholder {
  display: flex;
  align-items: center;
  gap: 16px;
  color: #174f9f;
  font-size: 42px;
  font-weight: 700;
}

.product-placeholder svg {
  width: 62px;
  height: 62px;
  stroke-width: 1.5;
}

.product-body {
  display: flex;
  min-height: 0;
  padding: 25px 25px 24px;
  flex: 1;
  flex-direction: column;
}

.product-copy h2 {
  margin: 0 0 12px;
  overflow-wrap: anywhere;
  color: #101828;
  font-size: 20px;
  line-height: 1.35;
  letter-spacing: 0;
}

.product-copy p {
  display: -webkit-box;
  min-height: 66px;
  margin: 0;
  overflow: hidden;
  color: #566176;
  font-size: 14px;
  line-height: 1.6;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
}

.product-price {
  margin-top: auto;
  padding: 22px 0 18px;
}

.product-price strong,
.product-price span {
  display: block;
}

.product-price strong {
  color: #0b1223;
  font-size: 30px;
  line-height: 1.2;
}

.product-price span {
  margin-top: 4px;
  color: #677287;
  font-size: 13px;
}

.product-action,
.catalog-state > button {
  display: inline-flex;
  min-height: 44px;
  align-items: center;
  justify-content: center;
  gap: 9px;
  border: 1px solid #2058b5;
  border-radius: 5px;
  color: #fff;
  background: #2058b5;
  cursor: pointer;
  font-size: 14px;
  font-weight: 700;
  transition: background .2s ease, border-color .2s ease;
}

.product-action:hover,
.catalog-state > button:hover {
  border-color: #174796;
  background: #174796;
}

.product-action svg,
.catalog-state > button svg {
  width: 17px;
  height: 17px;
}

.catalog-state {
  display: grid;
  width: 100%;
  min-height: 148px;
  margin: 42px 0 0;
  padding: 26px 28px;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 20px;
  border: 1px solid #d8dee8;
  border-radius: 6px;
  background: #fff;
}

.catalog-state-icon {
  display: inline-flex;
  width: 46px;
  height: 46px;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  color: #a2483d;
  background: #faecea;
}

.catalog-state-icon svg {
  width: 22px;
  height: 22px;
}

.catalog-state h2 {
  margin: 0;
  color: #17213a;
  font-size: 18px;
}

.catalog-state p {
  margin: 6px 0 0;
  overflow-wrap: anywhere;
  color: #647086;
  font-size: 14px;
  line-height: 1.55;
}

.catalog-state > button {
  min-width: 116px;
  padding: 0 18px;
}

.catalog-state--empty .catalog-state-icon {
  color: #2058b5;
  background: #edf3ff;
}

.skeleton {
  border-radius: 4px;
  background: linear-gradient(90deg, #eef1f5 25%, #f7f8fa 50%, #eef1f5 75%);
  background-size: 200% 100%;
  animation: skeleton-loading 1.4s ease infinite;
}

.product-visual-skeleton {
  height: 254px;
  border-radius: 0;
}

.skeleton-line {
  height: 14px;
  margin-top: 12px;
}

.skeleton-line--short {
  width: 30%;
  height: 11px;
  margin-top: 0;
}

.skeleton-line--title {
  width: 72%;
  height: 21px;
  margin: 18px 0 20px;
}

.skeleton-button {
  height: 44px;
  margin-top: auto;
}

@keyframes skeleton-loading {
  from { background-position: 200% 0; }
  to { background-position: -200% 0; }
}

@media (max-width: 760px) {
  .marketplace-hero-inner,
  .marketplace-catalog {
    width: calc(100% - 32px);
  }

  .marketplace-hero-inner {
    padding: 40px 0 30px;
  }

  .marketplace-hero h1 {
    font-size: 29px;
  }

  .marketplace-subtitle {
    font-size: 15px;
  }

  .catalog-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .category-filter {
    width: 100%;
  }

  .catalog-search {
    width: 100%;
    flex-basis: 44px;
  }

  .product-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .product-card {
    min-height: 530px;
  }

  .product-visual {
    height: 220px;
  }

  .catalog-state {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .catalog-state > button {
    grid-column: 1 / -1;
  }
}

@media (prefers-reduced-motion: reduce) {
  .product-card,
  .category-filter button,
  .product-action,
  .skeleton {
    transition: none;
    animation: none;
  }
}
</style>
