<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import type { Component } from "vue"
import { Award, BookOpen, CheckCircle2, ClipboardList, PackageOpen, PanelLeft, Receipt } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { isAuthenticated } from "@/lib/authStorage"
import { useTranslation } from "@/lib/language"

const { t } = useTranslation()
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
    },
    ...visibleCards.slice(1),
  ]
})
const featuredCards = computed(() => portalCards.value.filter((card) => card.featured))
const secondaryCards = computed(() => portalCards.value.filter((card) => !card.featured))
const showDashboardSkeleton = computed(() => dashboardLoading.value && !dashboardLoaded.value)

const cardStyles = {
  orange: {
    panel: "bg-[#fffaf0]",
    border: "border-[#f1d69a]",
    hoverBorder: "hover:border-[#d9a52e]",
    text: "text-[#704000]",
    number: "text-[#9b5200]",
    iconPanel: "bg-[#fff0c7]",
    iconText: "text-[#b86600]",
    accent: "#d99a13",
    glow: "rgba(217, 154, 19, 0.24)",
  },
  purple: {
    panel: "bg-[#f8f5ff]",
    border: "border-[#d9cff5]",
    hoverBorder: "hover:border-[#7657c8]",
    text: "text-[#463079]",
    number: "text-[#59399d]",
    iconPanel: "bg-[#ece5ff]",
    iconText: "text-[#6847b7]",
    accent: "#7657c8",
    glow: "rgba(118, 87, 200, 0.22)",
  },
  blue: {
    panel: "bg-[#f3f7ff]",
    border: "border-[#cbd9f3]",
    hoverBorder: "hover:border-[#0957f9]",
    text: "text-[#002a66]",
    number: "text-[#002a66]",
    iconPanel: "bg-[#e3edff]",
    iconText: "text-[#0957f9]",
    accent: "#0957f9",
    glow: "rgba(9, 87, 249, 0.22)",
  },
  teal: {
    panel: "bg-[#f1fbf9]",
    border: "border-[#bfe5dc]",
    hoverBorder: "hover:border-[#168a78]",
    text: "text-[#075e53]",
    number: "text-[#087568]",
    iconPanel: "bg-[#d9f3ed]",
    iconText: "text-[#0a8b79]",
    accent: "#14917d",
    glow: "rgba(20, 145, 125, 0.22)",
  },
  green: {
    panel: "bg-[#f2fbf4]",
    border: "border-[#c3e4ca]",
    hoverBorder: "hover:border-[#318a4a]",
    text: "text-[#175f2b]",
    number: "text-[#1b7133]",
    iconPanel: "bg-[#ddf2e2]",
    iconText: "text-[#278342]",
    accent: "#328b4b",
    glow: "rgba(50, 139, 75, 0.22)",
  },
} as const

async function countFromRequest(endpoint: string, listKey: string) {
  try {
    const res = await apiClient(endpoint)
    return Array.isArray(res?.[listKey]) ? res[listKey].length : 0
  } catch (err) {
    console.error(`Failed to load ${endpoint}:`, err)
    return 0
  }
}

onMounted(async () => {
  if (!isAuthenticated()) {
    return
  }

  dashboardLoading.value = true
  try {
    const [certifications, certificates, exams, resourcePacks, orders] = await Promise.all([
      countFromRequest("/api/pipeline", "list"),
      countFromRequest("/api/certificates", "certificates"),
      countFromRequest("/api/exams?page=1&page_size=50", "exams"),
      countFromRequest("/api/resource-packs?page_size=50", "packs"),
      countFromRequest("/api/orders?page=1&page_size=50", "orders"),
    ])

    counts.value = { certifications, certificates, courses: 0, exams, resourcePacks, orders }
  } finally {
    dashboardLoaded.value = true
    dashboardLoading.value = false
  }
})
</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <PanelLeft class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ t.sidebar.home }}</span>
      </header>

      <main class="dashboard-main px-5 py-10 md:px-8 lg:px-12">
        <section class="mx-auto w-full max-w-[1280px] border-b border-[#002a66]/15 pb-8 text-center">
          <span class="mx-auto mb-4 block h-1 w-12 rounded-full bg-[#0957f9]" aria-hidden="true" />
          <h1 class="text-[34px] font-bold leading-tight text-[#002a66] md:text-[38px]">{{ guideCopy.title }}</h1>
          <p class="mx-auto mt-4 max-w-4xl text-base leading-7 text-[#5b6b87] md:text-lg">{{ guideCopy.subtitle }}</p>
        </section>

        <section class="portal-card-section mx-auto mt-9 w-full max-w-[1280px]">
          <div v-if="showDashboardSkeleton" class="flex flex-col gap-8" role="status" :aria-label="t.common.loading" aria-live="polite">
            <div class="portal-card-row portal-card-featured-row grid grid-cols-1 gap-5 md:grid-cols-2">
              <div
                v-for="item in 2"
                :key="`featured-skeleton-${item}`"
                class="portal-stat-card portal-card-featured portal-card-skeleton min-h-[220px] w-full rounded-lg border border-[#ccd7e8] bg-white p-7"
              >
                <div class="mx-auto h-9 w-9 rounded-full bg-slate-100" />
                <div class="mx-auto mt-6 h-5 w-32 rounded-full bg-slate-100" />
                <div class="mx-auto mt-12 h-9 w-16 rounded-full bg-slate-100" />
                <div class="mx-auto mt-4 h-4 w-28 rounded-full bg-slate-100" />
              </div>
            </div>

            <div class="portal-card-row portal-card-secondary-row grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3">
              <div
                v-for="item in 3"
                :key="`secondary-skeleton-${item}`"
                class="portal-stat-card portal-card-secondary portal-card-skeleton min-h-[205px] w-full rounded-lg border border-[#ccd7e8] bg-white p-7"
              >
                <div class="mx-auto h-9 w-9 rounded-full bg-slate-100" />
                <div class="mx-auto mt-6 h-5 w-28 rounded-full bg-slate-100" />
                <div class="mx-auto mt-12 h-9 w-14 rounded-full bg-slate-100" />
                <div class="mx-auto mt-4 h-4 w-24 rounded-full bg-slate-100" />
              </div>
            </div>
          </div>

          <div v-else class="flex flex-col gap-5">
            <div class="portal-card-row portal-card-featured-row grid grid-cols-1 gap-5 md:grid-cols-2">
              <RouterLink
                v-for="card in featuredCards"
                :key="card.key"
                :to="card.href"
                :class="[
                  'portal-stat-card portal-card-featured group relative flex min-h-[220px] w-full flex-col items-center overflow-hidden rounded-lg border p-7 text-center shadow-[0_8px_20px_rgba(0,42,102,0.07)] transition-[transform,border-color,box-shadow] duration-300 hover:-translate-y-1 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30',
                  cardStyles[card.color].panel,
                  cardStyles[card.color].border,
                  cardStyles[card.color].hoverBorder,
                ]"
                :style="{ '--portal-card-accent': cardStyles[card.color].accent, '--portal-card-glow': cardStyles[card.color].glow }"
              >
                <span class="portal-card-accent absolute inset-x-0 top-0 h-1 origin-left" aria-hidden="true" />
                <div :class="['portal-card-icon relative flex h-12 w-12 items-center justify-center rounded-lg transition-transform duration-300 group-hover:scale-105', cardStyles[card.color].iconPanel]">
                  <component :is="card.icon" :class="['h-6 w-6', cardStyles[card.color].iconText]" :stroke-width="2.1" />
                </div>
                <h2 :class="['relative mt-5 text-lg font-semibold', cardStyles[card.color].text]">{{ card.title }}</h2>
                <p :class="['relative mt-auto pt-5 text-5xl font-bold tracking-tight', cardStyles[card.color].number]">
                  <span>{{ card.value }}</span>
                </p>
                <p :class="['relative mt-3 text-sm font-medium transition-opacity group-hover:opacity-75', cardStyles[card.color].text]">{{ card.action }} <span aria-hidden="true">→</span></p>
              </RouterLink>
            </div>

            <div class="portal-card-row portal-card-secondary-row grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3">
              <RouterLink
                v-for="card in secondaryCards"
                :key="card.key"
                :to="card.href"
                :class="[
                  'portal-stat-card portal-card-secondary group relative flex min-h-[205px] w-full flex-col items-center overflow-hidden rounded-lg border p-7 text-center shadow-[0_8px_20px_rgba(0,42,102,0.06)] transition-[transform,border-color,box-shadow] duration-300 hover:-translate-y-1 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30',
                  cardStyles[card.color].panel,
                  cardStyles[card.color].border,
                  cardStyles[card.color].hoverBorder,
                ]"
                :style="{ '--portal-card-accent': cardStyles[card.color].accent, '--portal-card-glow': cardStyles[card.color].glow }"
              >
                <span class="portal-card-accent absolute inset-x-0 top-0 h-1 origin-left" aria-hidden="true" />
                <div :class="['portal-card-icon relative flex h-12 w-12 items-center justify-center rounded-lg transition-transform duration-300 group-hover:scale-105', cardStyles[card.color].iconPanel]">
                  <component :is="card.icon" :class="['h-6 w-6', cardStyles[card.color].iconText]" :stroke-width="2.1" />
                </div>
                <h2 :class="['relative mt-5 text-base font-semibold', cardStyles[card.color].text]">{{ card.title }}</h2>
                <p :class="['relative mt-auto pt-4 text-[42px] font-bold tracking-tight', cardStyles[card.color].number]">
                  <span>{{ card.value }}</span>
                </p>
                <p :class="['relative mt-3 text-sm font-medium transition-opacity group-hover:opacity-75', cardStyles[card.color].text]">{{ card.action }} <span aria-hidden="true">→</span></p>
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
  line-height: 1.5;
}

.portal-stat-card p:first-of-type {
  line-height: 1;
}

.portal-stat-card p:last-of-type {
  line-height: 16px;
}

.portal-stat-card {
  --portal-card-accent: var(--gfi-vivid, #0957f9);
  --portal-card-glow: rgba(9, 87, 249, 0.16);
}

.portal-stat-card:hover {
  box-shadow: 0 18px 34px -18px var(--portal-card-glow), 0 12px 26px rgba(0, 42, 102, 0.1);
}

.portal-card-accent {
  background: var(--portal-card-accent);
  transform: scaleX(0.35);
  transition: transform 300ms ease;
}

.portal-stat-card:hover .portal-card-accent {
  transform: scaleX(1);
}

.portal-card-icon {
  box-shadow: 0 8px 18px -12px var(--portal-card-glow);
}

.portal-card-skeleton {
  overflow: hidden;
  position: relative;
}

.portal-card-skeleton::after {
  content: "";
  position: absolute;
  inset: 0;
  transform: translateX(-100%);
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.72), transparent);
  animation: portal-card-skeleton-shimmer 1.25s ease-in-out infinite;
}

@keyframes portal-card-skeleton-shimmer {
  100% {
    transform: translateX(100%);
  }
}

@media (max-width: 767px) {
  .dashboard-main {
    padding-top: 2rem !important;
  }
}
</style>
