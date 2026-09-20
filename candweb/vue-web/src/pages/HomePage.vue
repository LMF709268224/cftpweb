<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import type { Component } from "vue"
import { RouterLink } from "vue-router"
import { AlertCircle, Award, BookOpen, CheckCircle2, ClipboardList, Crown, GraduationCap, Loader2, PackageOpen, PanelLeft, Receipt, RefreshCw, ShieldCheck } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import PageFeedback from "@/components/PageFeedback.vue"
import { apiClient } from "@/lib/apiClient"
import { isAuthenticated } from "@/lib/authStorage"
import { useTranslation } from "@/lib/language"
import { useUser } from "@/lib/user"

const { t } = useTranslation()
const { currentUser, isLoading: userLoading } = useUser()
const dashboardLoading = ref(false)
const dashboardLoaded = ref(false)
const counts = ref({
  certifications: 0,
  certificates: 0,
  courses: 0,
  exams: 0,
  resourcePacks: 0,
  orders: 0,
})
type DashboardCountKey = "certifications" | "certificates" | "exams" | "resourcePacks" | "orders"
const countErrors = ref<Record<DashboardCountKey, boolean>>({
  certifications: false,
  certificates: false,
  exams: false,
  resourcePacks: false,
  orders: false,
})


const guideCopy = computed(() => ({
  title: t.value.home.portalTitle,
  subtitle: t.value.home.portalSubtitle,
}))

type CardColor = "orange" | "purple" | "blue" | "teal" | "green"
type PortalCard = {
  key: string
  title: string
  value: number
  action: string
  href: string
  icon: Component
  color: CardColor
  featured: boolean
  unavailable: boolean
}

const portalCards = computed<PortalCard[]>(() => {
  const cards: PortalCard[] = [
    {
      key: "certifications",
      title: t.value.home.completedCertifications,
      value: counts.value.certifications,
      action: t.value.home.viewCertifications,
      href: "/certifications",
      icon: Award,
      color: "orange",
      featured: true,
      unavailable: countErrors.value.certifications,
    },
    {
      key: "courses",
      title: t.value.home.coursesInProgress,
      value: counts.value.courses,
      action: t.value.home.viewCourses,
      href: "/certifications",
      icon: BookOpen,
      color: "purple",
      featured: true,
      unavailable: false,
    },
    {
      key: "exams",
      title: t.value.home.examCount,
      value: counts.value.exams,
      action: t.value.home.viewExams,
      href: "/exams",
      icon: ClipboardList,
      color: "blue",
      featured: false,
      unavailable: countErrors.value.exams,
    },
    {
      key: "resourcePacks",
      title: t.value.home.resourcePackCount,
      value: counts.value.resourcePacks,
      action: t.value.home.viewResourcePacks,
      href: "/resource-packs",
      icon: PackageOpen,
      color: "teal",
      featured: false,
      unavailable: countErrors.value.resourcePacks,
    },
    {
      key: "orders",
      title: t.value.home.orderCount,
      value: counts.value.orders,
      action: t.value.home.viewOrders,
      href: "/orders",
      icon: Receipt,
      color: "green",
      featured: false,
      unavailable: countErrors.value.orders,
    },
  ]

  const visibleCards = cards
    .filter((card) => card.key !== "courses")
    .map((card) => {
      switch (card.key) {
        case "certifications":
          return { ...card, title: t.value.home.purchasedCertifications, action: t.value.home.viewCertifications, href: "/my-certifications" }
        case "exams":
          return { ...card, title: t.value.home.examCount, action: t.value.home.viewExams }
        case "resourcePacks":
          return { ...card, title: t.value.home.resourcePackCount, action: t.value.home.viewResourcePacks }
        case "orders":
          return { ...card, title: t.value.home.orderCount, action: t.value.home.viewOrders }
        default:
          return card
      }
    })

  return [
    visibleCards[0],
    {
      key: "certificates",
      title: t.value.home.earnedCertificates,
      value: counts.value.certificates,
      action: t.value.home.viewCertificates,
      href: "/certificates",
      icon: CheckCircle2,
      color: "purple",
      featured: true,
      unavailable: countErrors.value.certificates,
    },
    ...visibleCards.slice(1),
  ]
})
const featuredCards = computed(() => portalCards.value.filter((card) => card.featured))
const secondaryCards = computed(() => portalCards.value.filter((card) => !card.featured))
const showDashboardSkeleton = computed(() => dashboardLoading.value && !dashboardLoaded.value)
const dashboardErrorCount = computed(() => Object.values(countErrors.value).filter(Boolean).length)
const dashboardLoadFailed = computed(() => dashboardLoaded.value && dashboardErrorCount.value === 5)
const dashboardPartialLoadFailed = computed(() => dashboardLoaded.value && dashboardErrorCount.value > 0 && !dashboardLoadFailed.value)

type AccountStatusItem = {
  key: string
  label: string
  value: string
  active: boolean
  icon: Component
}

function countLabel(template: string, count: number) {
  return template.replace("{count}", String(count))
}

const accountStatusItems = computed<AccountStatusItem[]>(() => {
  const status = currentUser.value?.account_status
  const items: AccountStatusItem[] = []

  if (status?.membership?.available) {
    items.push({
      key: "membership",
      label: t.value.home.accountMembership,
      value: status.membership.is_member
        ? status.membership.plan_name || t.value.home.accountActiveMember
        : t.value.home.accountNotMember,
      active: status.membership.is_member,
      icon: Crown,
    })
  }

  if (status?.certification?.available) {
    const programNames = status.certification.programs
      .map((program) => String(program.name || "").trim())
      .filter(Boolean)
    items.push({
      key: "certification",
      label: t.value.home.accountCertification,
      value: status.certification.is_candidate
        ? programNames.join(" · ") || countLabel(t.value.home.accountPurchasedPrograms, status.certification.purchase_count)
        : t.value.home.accountNoCertification,
      active: status.certification.is_candidate,
      icon: GraduationCap,
    })
  }

  if (status?.qualification?.available) {
    items.push({
      key: "qualification",
      label: t.value.home.accountQualification,
      value: status.qualification.has_qualification
        ? countLabel(t.value.home.accountQualificationsHeld, status.qualification.credential_count)
        : t.value.home.accountNoQualification,
      active: status.qualification.has_qualification,
      icon: ShieldCheck,
    })
  }

  return items
})
const accountStatusGridClass = computed(() => {
  if (accountStatusItems.value.length >= 3) return "md:grid-cols-3"
  if (accountStatusItems.value.length === 2) return "md:grid-cols-2"
  return "md:grid-cols-1"
})

const cardStyles = {
  orange: {
    panel: "bg-white",
    border: "border-[#c9962e]",
    hoverBorder: "hover:border-[#c9962e]",
    text: "text-[#002a66]",
    number: "text-[#002a66]",
    accent: "#c9962e",
  },
  purple: {
    panel: "bg-white",
    border: "border-[#c1cef6]",
    hoverBorder: "hover:border-[#0957f9]",
    text: "text-[#002a66]",
    number: "text-[#002a66]",
    accent: "#0957f9",
  },
  blue: {
    panel: "bg-white",
    border: "border-[#c1cef6]",
    hoverBorder: "hover:border-[#0957f9]",
    text: "text-[#002a66]",
    number: "text-[#002a66]",
    accent: "#0957f9",
  },
  teal: {
    panel: "bg-white",
    border: "border-[#c1cef6]",
    hoverBorder: "hover:border-[#0957f9]",
    text: "text-[#002a66]",
    number: "text-[#002a66]",
    accent: "#0957f9",
  },
  green: {
    panel: "bg-white",
    border: "border-[#c1cef6]",
    hoverBorder: "hover:border-[#0957f9]",
    text: "text-[#002a66]",
    number: "text-[#002a66]",
    accent: "#0957f9",
  },
} as const

async function countFromRequest(endpoint: string, listKey: string, totalKey?: string): Promise<number | null> {
  try {
    const res = await apiClient(endpoint, { suppressErrorToast: true })
    const list = res?.[listKey]
    if (!Array.isArray(list)) {
      console.error(`Invalid dashboard response from ${endpoint}: expected array at ${listKey}`)
      return null
    }
    if (totalKey && res?.[totalKey] !== null && res?.[totalKey] !== undefined) {
      const total = Number(res[totalKey])
      if (Number.isFinite(total) && total >= 0) return total
    }
    return list.length
  } catch (err) {
    console.error(`Failed to load ${endpoint}:`, err)
    return null
  }
}

async function loadDashboardStats() {
  if (dashboardLoading.value) return
  dashboardLoading.value = true
  try {
    const [certifications, certificates, exams, resourcePacks, orders] = await Promise.all([
      countFromRequest("/api/pipeline", "list"),
      countFromRequest("/api/certificates", "certificates"),
      countFromRequest("/api/exams?page=1&page_size=50", "exams", "total"),
      countFromRequest("/api/resource-packs?page_size=20", "packs"),
      countFromRequest("/api/orders?page=1&page_size=50", "orders", "total_orders"),
    ])

    countErrors.value = {
      certifications: certifications === null,
      certificates: certificates === null,
      exams: exams === null,
      resourcePacks: resourcePacks === null,
      orders: orders === null,
    }
    counts.value = {
      certifications: certifications ?? counts.value.certifications,
      certificates: certificates ?? counts.value.certificates,
      courses: counts.value.courses,
      exams: exams ?? counts.value.exams,
      resourcePacks: resourcePacks ?? counts.value.resourcePacks,
      orders: orders ?? counts.value.orders,
    }
  } finally {
    dashboardLoaded.value = true
    dashboardLoading.value = false
  }
}

onMounted(() => {
  if (!isAuthenticated()) return
  void loadDashboardStats()
})
</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <PanelLeft class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ t.sidebar.home }}</span>
      </header>

      <main class="px-5 py-10 md:px-8 lg:px-10">
        <section class="w-full text-center">
          <h1 class="font-bold text-[#002a66]">{{ guideCopy.title }}</h1>
          <p class="mx-auto mt-4 max-w-3xl text-[15px] leading-6 text-[#525e70]">{{ guideCopy.subtitle }}</p>
        </section>

        <section
          v-if="userLoading || accountStatusItems.length > 0"
          data-testid="dashboard-account-status"
          class="mx-auto mt-8 w-full max-w-[1120px] overflow-hidden rounded-lg border border-slate-200 bg-white shadow-[0_8px_24px_rgba(15,23,42,0.06)]"
          :aria-label="t.home.accountStatusTitle"
        >
          <div v-if="userLoading && accountStatusItems.length === 0" class="grid min-h-[112px] grid-cols-1 divide-y divide-slate-200 md:grid-cols-3 md:divide-x md:divide-y-0" role="status">
            <div v-for="item in 3" :key="`account-status-skeleton-${item}`" class="flex items-center gap-4 px-6 py-5 sm:px-7">
              <div class="h-11 w-11 shrink-0 animate-pulse rounded-lg bg-slate-100" />
              <div class="min-w-0 flex-1 space-y-2">
                <div class="h-3 w-20 animate-pulse rounded bg-slate-100" />
                <div class="h-4 w-32 animate-pulse rounded bg-slate-100" />
              </div>
            </div>
          </div>
          <div v-else :class="['grid min-h-[112px] grid-cols-1 divide-y divide-slate-200 md:divide-x md:divide-y-0', accountStatusGridClass]">
            <div
              v-for="item in accountStatusItems"
              :key="item.key"
              :data-testid="`dashboard-account-status-${item.key}`"
              class="flex min-w-0 items-center gap-4 px-6 py-5 text-left sm:px-7"
            >
              <div :class="['flex h-11 w-11 shrink-0 items-center justify-center rounded-lg border', item.active ? 'border-[#2f7d4f]/25 bg-[#ecf5f0] text-[#2f7d4f]' : 'border-slate-200 bg-slate-50 text-slate-500']">
                <component :is="item.icon" class="h-5 w-5" />
              </div>
              <div class="min-w-0">
                <div class="text-xs font-medium leading-5 text-slate-500">{{ item.label }}</div>
                <div class="mt-0.5 break-words text-sm font-semibold leading-6 text-slate-900">{{ item.value }}</div>
              </div>
            </div>
          </div>
        </section>

        <section class="portal-card-section mx-auto mt-12 w-full max-w-[1120px]">
          <PageFeedback
            v-if="dashboardLoadFailed"
            kind="error"
            :title="t.home.statsLoadFailed"
            :description="t.home.statsLoadFailedDesc"
          >
            <template #action>
              <button type="button" class="btn btn-primary mt-5 min-w-32 justify-center" :disabled="dashboardLoading" @click="loadDashboardStats">
                <Loader2 v-if="dashboardLoading" class="h-4 w-4 animate-spin" />
                <RefreshCw v-else class="h-4 w-4" />
                {{ t.home.statsRetry }}
              </button>
            </template>
          </PageFeedback>

          <div v-else-if="showDashboardSkeleton" class="flex flex-col gap-8" role="status" :aria-label="t.common.loading" aria-live="polite">
            <div class="portal-card-row portal-card-featured-row flex flex-col items-center justify-center gap-6 lg:flex-row">
              <div
                v-for="item in 2"
                :key="`featured-skeleton-${item}`"
                class="portal-stat-card portal-card-featured portal-card-skeleton h-[214px] w-full rounded-[16px] border border-slate-100 bg-white p-8 shadow-[0_2px_8px_rgba(15,23,42,0.08)] lg:basis-[34%] lg:grow-0 lg:shrink-0"
              >
                <div class="mx-auto h-9 w-9 rounded-full bg-slate-100" />
                <div class="portal-card-skeleton-title mx-auto mt-6 h-5 w-32 rounded-full bg-slate-100" />
                <div class="portal-card-skeleton-value mx-auto mt-12 h-9 w-16 rounded-full bg-slate-100" />
                <div class="portal-card-skeleton-action mx-auto mt-4 h-4 w-28 rounded-full bg-slate-100" />
              </div>
            </div>

            <div class="portal-card-row portal-card-secondary-row flex flex-col items-center justify-center gap-6 lg:flex-row">
              <div
                v-for="item in 3"
                :key="`secondary-skeleton-${item}`"
                class="portal-stat-card portal-card-secondary portal-card-skeleton h-[214px] w-full rounded-[16px] border border-slate-100 bg-white p-8 shadow-[0_2px_8px_rgba(15,23,42,0.08)] lg:basis-[29%] lg:grow-0 lg:shrink-0"
              >
                <div class="mx-auto h-9 w-9 rounded-full bg-slate-100" />
                <div class="portal-card-skeleton-title mx-auto mt-6 h-5 w-28 rounded-full bg-slate-100" />
                <div class="portal-card-skeleton-value mx-auto mt-12 h-9 w-14 rounded-full bg-slate-100" />
                <div class="portal-card-skeleton-action mx-auto mt-4 h-4 w-24 rounded-full bg-slate-100" />
              </div>
            </div>
          </div>

          <div v-else class="flex flex-col gap-8">
            <div
              v-if="dashboardPartialLoadFailed"
              class="flex flex-col items-start justify-between gap-3 rounded-lg border border-[#c9962e]/40 bg-[#fdf6ec] px-4 py-3 text-left sm:flex-row sm:items-center"
              role="alert"
            >
              <div class="flex items-start gap-2 text-sm text-[#a6600c]">
                <AlertCircle class="mt-0.5 h-4 w-4 shrink-0" />
                <span>{{ t.home.statsPartialLoadFailed }}</span>
              </div>
              <button
                type="button"
                class="btn btn-outline h-10 shrink-0 border-[#c9962e] bg-white px-4 text-[#a6600c] hover:bg-[#fdf6ec]"
                :disabled="dashboardLoading"
                @click="loadDashboardStats"
              >
                <Loader2 v-if="dashboardLoading" class="h-4 w-4 animate-spin" />
                <RefreshCw v-else class="h-4 w-4" />
                {{ t.home.statsRetry }}
              </button>
            </div>
            <div class="portal-card-row portal-card-featured-row flex flex-col items-center justify-center gap-6 lg:flex-row">
              <RouterLink
                v-for="card in featuredCards"
                :key="card.key"
                :data-testid="`dashboard-card-${card.key}`"
                :to="card.href"
                :class="[
                  'portal-stat-card portal-card-featured group relative flex h-[214px] w-full flex-col items-center justify-center overflow-hidden rounded-[14px] border p-8 text-center transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30 lg:basis-[34%] lg:grow-0 lg:shrink-0',
                  cardStyles[card.color].panel,
                  cardStyles[card.color].border,
                  cardStyles[card.color].hoverBorder,
                ]"
                :style="{ '--portal-card-accent': cardStyles[card.color].accent }"
              >
                <span class="portal-card-sheen pointer-events-none absolute left-0 top-0 h-1 w-full" />
                <div class="portal-card-icon relative flex h-11 w-11 items-center justify-center rounded-lg bg-[#edeef2]">
                  <component :is="card.icon" :class="['h-6 w-6', cardStyles[card.color].text]" :stroke-width="1.5" />
                </div>
                <h2 :class="['relative mt-6 text-lg font-semibold', cardStyles[card.color].text]">{{ card.title }}</h2>
                <p :class="['relative mt-12 text-5xl font-bold tracking-tight', cardStyles[card.color].number]">
                  <span>{{ card.unavailable ? "--" : card.value }}</span>
                </p>
                <p :class="['relative mt-3 text-base', cardStyles[card.color].text]">{{ card.action }}</p>
              </RouterLink>
            </div>

            <div class="portal-card-row portal-card-secondary-row flex flex-col items-center justify-center gap-6 lg:flex-row">
              <RouterLink
                v-for="card in secondaryCards"
                :key="card.key"
                :data-testid="`dashboard-card-${card.key}`"
                :to="card.href"
                :class="[
                  'portal-stat-card portal-card-secondary group relative flex h-[214px] w-full flex-col items-center justify-center overflow-hidden rounded-[14px] border p-8 text-center transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30 lg:basis-[29%] lg:grow-0 lg:shrink-0',
                  cardStyles[card.color].panel,
                  cardStyles[card.color].border,
                  cardStyles[card.color].hoverBorder,
                ]"
                :style="{ '--portal-card-accent': cardStyles[card.color].accent }"
              >
                <span class="portal-card-sheen pointer-events-none absolute left-0 top-0 h-1 w-full" />
                <div class="portal-card-icon relative flex h-11 w-11 items-center justify-center rounded-lg bg-[#edeef2]">
                  <component :is="card.icon" :class="['h-6 w-6', cardStyles[card.color].text]" :stroke-width="1.5" />
                </div>
                <h2 :class="['relative mt-6 text-lg font-semibold', cardStyles[card.color].text]">{{ card.title }}</h2>
                <p :class="['relative mt-12 text-5xl font-bold tracking-tight', cardStyles[card.color].number]">
                  <span>{{ card.unavailable ? "--" : card.value }}</span>
                </p>
                <p :class="['relative mt-3 text-base', cardStyles[card.color].text]">{{ card.action }}</p>
              </RouterLink>
            </div>
          </div>
        </section>
      </main>
    </div>
  </AppShell>
</template>

<style scoped>
.portal-stat-card :deep(svg) {
  height: 24px !important;
  width: 24px !important;
}

.portal-stat-card h2 {
  margin-top: 16px;
  font-size: 14px;
  line-height: 20px;
}

.portal-stat-card p:first-of-type {
  margin-top: 36px;
  font-size: 30px;
  line-height: 1;
}

.portal-stat-card p:last-of-type {
  margin-top: 12px;
  font-size: 12px;
  line-height: 16px;
}

.portal-stat-card {
  --portal-card-accent: #0957f9;
}

.portal-card-sheen {
  background: var(--portal-card-accent);
}

.portal-card-icon {
  box-shadow: none;
}

.portal-card-skeleton {
  overflow: hidden;
  position: relative;
}

.portal-card-skeleton::after { content: none; }

@media (max-width: 767px) {
  .portal-card-section {
    margin-top: 24px;
  }

  .portal-card-section > div {
    gap: 16px;
  }

  .portal-card-row {
    gap: 16px;
  }

  .portal-stat-card {
    height: 184px;
    padding: 20px;
  }

  .portal-card-icon {
    height: 40px;
    width: 40px;
  }

  .portal-stat-card h2 {
    margin-top: 10px;
  }

  .portal-stat-card p:first-of-type {
    margin-top: 18px;
  }

  .portal-stat-card p:last-of-type {
    margin-top: 8px;
  }

  .portal-card-skeleton-title {
    margin-top: 10px;
  }

  .portal-card-skeleton-value {
    margin-top: 18px;
  }

  .portal-card-skeleton-action {
    margin-top: 8px;
  }
}

</style>
