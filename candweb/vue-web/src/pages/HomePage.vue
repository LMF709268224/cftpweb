<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { RouterLink } from "vue-router"
import { ArrowRight, BookOpen, FileText, GraduationCap, PanelLeft, Play } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import PageFeedback from "@/components/PageFeedback.vue"
import { apiClient } from "@/lib/apiClient"
import { isAuthenticated } from "@/lib/authStorage"
import { useTranslation } from "@/lib/language"
import { formatBackendDateOnly } from "@/lib/utils"
import { useUser } from "@/lib/user"

type Pipeline = {
  pipeline_ulid?: string
  pipeline_cc_ulid?: string
  pipeline_name?: string
  current_stage_name?: string
  progress?: number
  progress_available?: boolean
  status?: string | number
  started_at?: string
  completed_at?: string
  created_at?: string
}

type ResourcePack = {
  pack_id?: string
  title?: string
  description?: string
  respath?: string
}

type ResourceFile = {
  file_id?: string
  title?: string
  description?: string
  file_type?: number | string
  created_at?: string
}

type Bundle = {
  bundle_id?: string
  pipeline_id?: string
  name?: string
  description?: string
  thumbnail_url?: string
  is_pipeline_bundle?: boolean
  bundle_item_types?: string[]
}

const { t, lang } = useTranslation()
const { currentUser } = useUser()
const pipelines = ref<Pipeline[]>([])
const pipelinesLoaded = ref(false)
const pipelinesLoading = ref(false)
const pipelinesError = ref(false)
const insights = ref<ResourceFile[]>([])
const insightsPack = ref<ResourcePack | null>(null)
const insightsLoading = ref(false)
const insightsError = ref(false)
const products = ref<Bundle[]>([])
const productsLoading = ref(false)
const productsError = ref(false)

const greetingName = computed(() => [
  currentUser.value?.first_name,
  currentUser.value?.display_name,
  currentUser.value?.real_name,
].map((value) => String(value || "").trim()).find(Boolean) || "")

const greeting = computed(() => greetingName.value
  ? t.value.home.portalTitleNamed.replace("{name}", greetingName.value)
  : t.value.home.portalTitle)

const nextStep = computed(() => {
  const status = currentUser.value?.account_status
  if (!currentUser.value) return null
  const courseKnown = Boolean(status?.certification?.available) || (pipelinesLoaded.value && !pipelinesError.value)
  const hasCourse = status?.certification?.available
    ? status.certification.is_candidate
    : pipelinesLoaded.value && !pipelinesError.value && pipelines.value.length > 0
  if (hasCourse) return { label: t.value.home.startYourLearning, href: "/my-certifications" }
  if (courseKnown && status?.membership?.available && !status.membership.is_member) {
    return { label: t.value.home.becomeMember, href: "/membership" }
  }
  return { label: t.value.home.exploreOptions, href: "/certifications" }
})

const continuingPipelines = computed(() => pipelines.value
  .filter((pipeline) => {
    const status = String(pipeline.status || "").toUpperCase()
    return !pipeline.completed_at && status === "PIPELINE_STATUS_RUNNING"
  })
  .sort((a, b) => (Date.parse(b.started_at || b.created_at || "") || 0) - (Date.parse(a.started_at || a.created_at || "") || 0))
  .slice(0, 2))

function pipelineHref(pipeline: Pipeline) {
  const id = pipeline.pipeline_cc_ulid || pipeline.pipeline_ulid
  return id ? `/certifications/${encodeURIComponent(id)}` : "/my-certifications"
}

function productHref(product: Bundle) {
  return product.pipeline_id
    ? `/certifications/${encodeURIComponent(product.pipeline_id)}`
    : "/certifications"
}

function fileHref(file: ResourceFile) {
  if (!file.file_id) return insightsPack.value?.pack_id
    ? `/resource-packs/${encodeURIComponent(insightsPack.value.pack_id)}`
    : "/resource-packs"
  if (Number(file.file_type) === 2) return `/resource-pack-files/${encodeURIComponent(file.file_id)}/preview`
  if (Number(file.file_type) === 1) return `/video-preview/resource-pack-files/${encodeURIComponent(file.file_id)}`
  return insightsPack.value?.pack_id
    ? `/resource-packs/${encodeURIComponent(insightsPack.value.pack_id)}`
    : "/resource-packs"
}

function opensPreview(file: ResourceFile) {
  return Boolean(file.file_id && [1, 2].includes(Number(file.file_type)))
}

function rememberInsightsPack() {
  const pack = insightsPack.value
  if (!pack?.pack_id) return
  sessionStorage.setItem(`resource-pack-title:${pack.pack_id}`, pack.title || "")
  sessionStorage.setItem(`resource-pack-description:${pack.pack_id}`, pack.description || "")
  sessionStorage.setItem(`resource-pack-respath:${pack.pack_id}`, pack.respath || "")
}

function rememberFileTitle(file: ResourceFile) {
  if (file.file_id && file.title) {
    sessionStorage.setItem(`resource-pack-file-preview-title:${file.file_id}`, file.title)
  }
  rememberInsightsPack()
}

async function loadPipelines() {
  pipelinesLoading.value = true
  pipelinesError.value = false
  try {
    const response = await apiClient("/api/pipeline", { suppressErrorToast: true })
    if (!Array.isArray(response?.list)) throw new Error("PIPELINE_LIST_INVALID_RESPONSE")
    pipelines.value = response.list
  } catch (error) {
    console.error("Failed to load dashboard learning", error)
    pipelinesError.value = true
  } finally {
    pipelinesLoaded.value = true
    pipelinesLoading.value = false
  }
}

async function loadInsights() {
  insightsLoading.value = true
  insightsError.value = false
  try {
    let pack: ResourcePack | undefined
    let cursor = ""
    const seen = new Set<string>()
    for (let page = 0; page < 100; page += 1) {
      const params = new URLSearchParams({ page_size: "50" })
      if (cursor) params.set("page_token", cursor)
      const response = await apiClient(`/api/resource-packs?${params}`, { suppressErrorToast: true })
      if (!Array.isArray(response?.packs)) throw new Error("RESOURCE_PACKS_INVALID_RESPONSE")
      pack = response.packs.find((item: ResourcePack) =>
        `${item.respath || ""} ${item.title || ""}`.toLowerCase().includes("insight"))
      if (pack) break
      const next = String(response.next_page_token || "")
      if (!next) break
      if (seen.has(next)) throw new Error("RESOURCE_PACK_CURSOR_LOOP")
      seen.add(next)
      cursor = next
    }
    insightsPack.value = pack || null
    if (!pack?.pack_id) {
      insights.value = []
      return
    }

    const files: ResourceFile[] = []
    cursor = ""
    seen.clear()
    let complete = false
    for (let page = 0; page < 100; page += 1) {
      const params = new URLSearchParams({ page_size: "50" })
      if (cursor) params.set("page_token", cursor)
      const response = await apiClient(
        `/api/resource-packs/${encodeURIComponent(pack.pack_id)}/files?${params}`,
        { suppressErrorToast: true },
      )
      if (!Array.isArray(response?.files)) throw new Error("RESOURCE_FILES_INVALID_RESPONSE")
      files.push(...response.files)
      const next = String(response.next_page_token || "")
      if (!next) {
        complete = true
        break
      }
      if (seen.has(next)) throw new Error("RESOURCE_FILE_CURSOR_LOOP")
      seen.add(next)
      cursor = next
    }
    if (!complete) throw new Error("RESOURCE_FILE_PAGE_LIMIT")
    insights.value = files
      .filter((file) => Boolean(file.file_id))
      .sort((a, b) => (Date.parse(b.created_at || "") || 0) - (Date.parse(a.created_at || "") || 0))
      .slice(0, 2)
  } catch (error) {
    console.error("Failed to load dashboard insights", error)
    insightsError.value = true
  } finally {
    insightsLoading.value = false
  }
}

async function loadProducts() {
  productsLoading.value = true
  productsError.value = false
  try {
    const response = await apiClient("/api/mall/bundles", { suppressErrorToast: true })
    if (!Array.isArray(response?.bundles)) throw new Error("MALL_BUNDLES_INVALID_RESPONSE")
    products.value = response.bundles
      .filter((bundle: Bundle) => bundle.is_pipeline_bundle
        || bundle.bundle_item_types?.some((type) => String(type).toLowerCase().includes("pipeline")))
      .slice(0, 3)
  } catch (error) {
    console.error("Failed to load dashboard products", error)
    productsError.value = true
  } finally {
    productsLoading.value = false
  }
}

function reloadDashboard() {
  void Promise.all([loadPipelines(), loadInsights(), loadProducts()])
}

onMounted(() => {
  if (isAuthenticated()) reloadDashboard()
})
watch(lang, () => {
  if (isAuthenticated()) reloadDashboard()
})
</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <PanelLeft class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ t.sidebar.home }}</span>
      </header>

      <main class="dashboard-main mx-auto w-full max-w-[1180px] px-5 py-8 md:px-8 lg:px-10">
        <div class="dashboard-intro">
          <h1 class="text-3xl font-bold text-[#002a66]">{{ greeting }}</h1>
          <p class="mt-2 max-w-3xl text-sm leading-6 text-slate-600">{{ t.home.portalSubtitle }}</p>
          <RouterLink v-if="nextStep" data-testid="dashboard-next-step" :to="nextStep.href" class="btn btn-primary mt-5 inline-flex min-h-10 items-center gap-2 px-4">
            {{ nextStep.label }}
            <ArrowRight class="h-4 w-4" aria-hidden="true" />
          </RouterLink>
        </div>

        <section class="dashboard-section mt-9 border-t border-slate-200 pt-6" :aria-label="t.home.continueLearning">
          <div class="mb-4 flex items-center justify-between gap-4">
            <h2 class="text-lg font-semibold text-[#002a66]">{{ t.home.continueLearning }}</h2>
            <RouterLink to="/my-certifications" class="inline-flex items-center gap-1 text-sm font-medium text-primary hover:underline">
              {{ t.home.viewLearning }} <ArrowRight class="h-4 w-4" aria-hidden="true" />
            </RouterLink>
          </div>
          <PageFeedback v-if="pipelinesError" kind="error" :title="t.home.learningLoadFailed" :action-label="t.home.retry" @action="loadPipelines" />
          <div v-else-if="pipelinesLoading && !pipelinesLoaded" class="h-32 animate-pulse rounded-md bg-white" role="status" :aria-label="t.common.loading" />
          <div v-else-if="continuingPipelines.length" class="grid gap-3 md:grid-cols-2">
            <RouterLink
              v-for="pipeline in continuingPipelines"
              :key="pipeline.pipeline_ulid || pipeline.pipeline_cc_ulid"
              data-testid="dashboard-learning-card"
              :to="pipelineHref(pipeline)"
              class="dashboard-item flex min-w-0 flex-col rounded-md border border-slate-200 bg-white p-4 hover:border-primary"
            >
              <div class="flex items-start gap-3">
                <BookOpen class="mt-0.5 h-5 w-5 shrink-0 text-primary" aria-hidden="true" />
                <div class="min-w-0 flex-1">
                  <h3 class="break-words text-base font-semibold text-[#002a66]">{{ pipeline.pipeline_name || t.common.unknownCourse }}</h3>
                  <p v-if="pipeline.current_stage_name" class="mt-1 text-sm text-slate-600">{{ pipeline.current_stage_name }}</p>
                </div>
                <ArrowRight class="h-4 w-4 shrink-0 text-primary" aria-hidden="true" />
              </div>
              <div v-if="pipeline.progress_available && Number.isFinite(Number(pipeline.progress))" class="mt-5">
                <div class="mb-1.5 flex justify-between text-xs text-slate-600">
                  <span>{{ t.home.learningProgress }}</span>
                  <span>{{ Math.round(Math.max(0, Math.min(100, Number(pipeline.progress)))) }}%</span>
                </div>
                <div class="h-2 overflow-hidden rounded-full bg-slate-100">
                  <div class="h-full rounded-full bg-primary" :style="{ width: `${Math.max(0, Math.min(100, Number(pipeline.progress)))}%` }" />
                </div>
              </div>
            </RouterLink>
          </div>
          <div v-else class="flex flex-col items-start gap-3 border-l-2 border-primary bg-white px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 class="text-sm font-semibold text-[#002a66]">{{ pipelines.length ? t.home.learningCompleteTitle : t.home.noLearningTitle }}</h3>
              <p class="mt-1 text-sm text-slate-600">{{ pipelines.length ? t.home.learningCompleteDesc : t.home.noLearningDesc }}</p>
            </div>
            <RouterLink to="/certifications" class="inline-flex min-h-9 shrink-0 items-center gap-1 text-sm font-semibold text-primary hover:underline">
              {{ t.home.exploreLearning }} <ArrowRight class="h-4 w-4" aria-hidden="true" />
            </RouterLink>
          </div>
        </section>

        <section class="dashboard-section mt-9 border-t border-slate-200 pt-6" :aria-label="t.home.latestInsights">
          <div class="mb-4 flex items-center justify-between gap-4">
            <h2 class="text-lg font-semibold text-[#002a66]">{{ t.home.latestInsights }}</h2>
            <RouterLink
              v-if="insightsPack?.pack_id"
              :to="`/resource-packs/${encodeURIComponent(insightsPack.pack_id)}`"
              class="inline-flex items-center gap-1 text-sm font-medium text-primary hover:underline"
              @click="rememberInsightsPack"
            >
              {{ t.home.viewAll }} <ArrowRight class="h-4 w-4" aria-hidden="true" />
            </RouterLink>
          </div>
          <PageFeedback v-if="insightsError" kind="error" :title="t.home.insightsLoadFailed" :action-label="t.home.retry" @action="loadInsights" />
          <div v-else-if="insightsLoading" class="h-28 animate-pulse rounded-md bg-white" role="status" :aria-label="t.common.loading" />
          <div v-else-if="insights.length" class="grid gap-3 md:grid-cols-2">
            <a
              v-for="file in insights"
              :key="file.file_id"
              data-testid="dashboard-insight-card"
              :href="fileHref(file)"
              :target="opensPreview(file) ? '_blank' : undefined"
              :rel="opensPreview(file) ? 'noopener noreferrer' : undefined"
              class="dashboard-item flex min-w-0 flex-col rounded-md border border-slate-200 bg-white p-4 hover:border-primary"
              @click="rememberFileTitle(file)"
            >
              <div class="flex items-start gap-3">
                <component :is="Number(file.file_type) === 1 ? Play : FileText" class="mt-0.5 h-5 w-5 shrink-0 text-[#2f7d4f]" aria-hidden="true" />
                <div class="min-w-0 flex-1">
                  <h3 class="break-words text-base font-semibold text-[#002a66]">{{ file.title || t.home.untitledInsight }}</h3>
                  <p v-if="file.description" class="mt-1 line-clamp-2 text-sm leading-5 text-slate-600">{{ file.description }}</p>
                </div>
              </div>
              <div class="mt-auto flex items-center justify-between gap-3 pt-4 text-xs text-slate-500">
                <span>{{ file.created_at ? formatBackendDateOnly(file.created_at, lang) : "" }}</span>
                <span class="inline-flex items-center gap-1 font-semibold text-primary">
                  {{ Number(file.file_type) === 1 ? t.home.watchInsight : t.home.readInsight }}
                  <ArrowRight class="h-4 w-4" aria-hidden="true" />
                </span>
              </div>
            </a>
          </div>
          <p v-else class="py-5 text-sm text-slate-600">{{ t.home.noInsights }}</p>
        </section>

        <section class="dashboard-section mt-9 border-t border-slate-200 pt-6" :aria-label="t.home.featuredLearning">
          <div class="mb-4 flex items-center justify-between gap-4">
            <h2 class="text-lg font-semibold text-[#002a66]">{{ t.home.featuredLearning }}</h2>
            <RouterLink to="/certifications" class="inline-flex items-center gap-1 text-sm font-medium text-primary hover:underline">
              {{ t.home.viewAll }} <ArrowRight class="h-4 w-4" aria-hidden="true" />
            </RouterLink>
          </div>
          <PageFeedback v-if="productsError" kind="error" :title="t.home.productsLoadFailed" :action-label="t.home.retry" @action="loadProducts" />
          <div v-else-if="productsLoading" class="h-40 animate-pulse rounded-md bg-white" role="status" :aria-label="t.common.loading" />
          <div v-else-if="products.length" class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            <RouterLink
              v-for="product in products"
              :key="product.bundle_id"
              data-testid="dashboard-product-card"
              :to="productHref(product)"
              class="dashboard-item flex min-w-0 flex-col overflow-hidden rounded-md border border-slate-200 bg-white hover:border-primary"
            >
              <div class="flex aspect-[16/7] items-center justify-center bg-slate-50">
                <img v-if="product.thumbnail_url" :src="product.thumbnail_url" :alt="product.name || ''" class="h-full w-full object-contain p-3" />
                <GraduationCap v-else class="h-8 w-8 text-primary" aria-hidden="true" />
              </div>
              <div class="flex flex-1 flex-col p-4">
                <h3 class="break-words text-base font-semibold text-[#002a66]">{{ product.name || t.common.unknownCourse }}</h3>
                <p v-if="product.description" class="mt-1 line-clamp-2 text-sm leading-5 text-slate-600">{{ product.description }}</p>
                <span class="mt-auto inline-flex items-center gap-1 pt-4 text-sm font-semibold text-primary">
                  {{ t.home.viewCourse }} <ArrowRight class="h-4 w-4" aria-hidden="true" />
                </span>
              </div>
            </RouterLink>
          </div>
          <p v-else class="py-5 text-sm text-slate-600">{{ t.home.noProducts }}</p>
        </section>
      </main>
    </div>
  </AppShell>
</template>

<style scoped>
.dashboard-item {
  transition: border-color 150ms ease, box-shadow 150ms ease;
}

.dashboard-item:hover {
  box-shadow: 0 4px 14px rgb(0 42 102 / 7%);
}

@media (max-width: 640px) {
  .dashboard-main {
    padding-bottom: 40px;
  }
}
</style>
