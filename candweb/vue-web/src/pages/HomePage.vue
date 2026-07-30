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
    panel: "bg-white",
    border: "border-[#ccd7e8]",
    hoverBorder: "hover:border-[#0957f9]",
    text: "text-[#002a66]",
    number: "text-[#002a66]",
    accent: "#0957f9",
    glow: "rgba(9, 87, 249, 0.12)",
  },
  purple: {
    panel: "bg-white",
    border: "border-[#ccd7e8]",
    hoverBorder: "hover:border-[#0957f9]",
    text: "text-[#002a66]",
    number: "text-[#002a66]",
    accent: "#0957f9",
    glow: "rgba(9, 87, 249, 0.12)",
  },
  blue: {
    panel: "bg-white",
    border: "border-[#ccd7e8]",
    hoverBorder: "hover:border-[#0957f9]",
    text: "text-[#002a66]",
    number: "text-[#002a66]",
    accent: "#0957f9",
    glow: "rgba(9, 87, 249, 0.12)",
  },
  teal: {
    panel: "bg-white",
    border: "border-[#ccd7e8]",
    hoverBorder: "hover:border-[#0957f9]",
    text: "text-[#002a66]",
    number: "text-[#002a66]",
    accent: "#0957f9",
    glow: "rgba(9, 87, 249, 0.12)",
  },
  green: {
    panel: "bg-white",
    border: "border-[#ccd7e8]",
    hoverBorder: "hover:border-[#0957f9]",
    text: "text-[#002a66]",
    number: "text-[#002a66]",
    accent: "#0957f9",
    glow: "rgba(9, 87, 249, 0.12)",
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
        <section class="mx-auto w-full max-w-[1180px] border-b border-[#002a66]/15 pb-8 text-left">
          <h1 class="text-[32px] font-bold leading-tight text-[#002a66]">{{ guideCopy.title }}</h1>
          <p class="mt-3 max-w-3xl text-base leading-7 text-[#5b6b87]">{{ guideCopy.subtitle }}</p>
        </section>

        <section class="portal-card-section mx-auto mt-8 w-full max-w-[1180px]">
          <div v-if="showDashboardSkeleton" class="flex flex-col gap-8" role="status" :aria-label="t.common.loading" aria-live="polite">
            <div class="portal-card-row portal-card-featured-row grid grid-cols-1 gap-5 md:grid-cols-2">
              <div
                v-for="item in 2"
                :key="`featured-skeleton-${item}`"
                class="portal-stat-card portal-card-featured portal-card-skeleton min-h-[190px] w-full rounded-lg border border-[#ccd7e8] bg-white p-6"
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
                class="portal-stat-card portal-card-secondary portal-card-skeleton min-h-[190px] w-full rounded-lg border border-[#ccd7e8] bg-white p-6"
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
                  'portal-stat-card portal-card-featured group relative flex min-h-[190px] w-full flex-col items-start rounded-lg border p-6 text-left transition-[border-color,box-shadow] duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30',
                  cardStyles[card.color].panel,
                  cardStyles[card.color].border,
                  cardStyles[card.color].hoverBorder,
                ]"
                :style="{ '--portal-card-accent': cardStyles[card.color].accent, '--portal-card-glow': cardStyles[card.color].glow }"
              >
                <div class="portal-card-icon relative flex h-10 w-10 items-center justify-center rounded-lg bg-[#edf2ff] text-[#0957f9]">
                  <component :is="card.icon" class="h-6 w-6 text-[#0957f9]" :stroke-width="2.1" />
                </div>
                <h2 :class="['relative mt-5 text-base font-semibold', cardStyles[card.color].text]">{{ card.title }}</h2>
                <p :class="['relative mt-auto pt-6 text-4xl font-bold', cardStyles[card.color].number]">
                  <span>{{ card.value }}</span>
                </p>
                <p class="relative mt-2 text-sm font-medium text-[#0957f9] transition-colors group-hover:text-[#002a66]">{{ card.action }} <span aria-hidden="true">→</span></p>
              </RouterLink>
            </div>

            <div class="portal-card-row portal-card-secondary-row grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3">
              <RouterLink
                v-for="card in secondaryCards"
                :key="card.key"
                :to="card.href"
                :class="[
                  'portal-stat-card portal-card-secondary group relative flex min-h-[190px] w-full flex-col items-start rounded-lg border p-6 text-left transition-[border-color,box-shadow] duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30',
                  cardStyles[card.color].panel,
                  cardStyles[card.color].border,
                  cardStyles[card.color].hoverBorder,
                ]"
                :style="{ '--portal-card-accent': cardStyles[card.color].accent, '--portal-card-glow': cardStyles[card.color].glow }"
              >
                <div class="portal-card-icon relative flex h-10 w-10 items-center justify-center rounded-lg bg-[#edf2ff] text-[#0957f9]">
                  <component :is="card.icon" class="h-6 w-6 text-[#0957f9]" :stroke-width="2.1" />
                </div>
                <h2 :class="['relative mt-5 text-base font-semibold', cardStyles[card.color].text]">{{ card.title }}</h2>
                <p :class="['relative mt-auto pt-6 text-4xl font-bold', cardStyles[card.color].number]">
                  <span>{{ card.value }}</span>
                </p>
                <p class="relative mt-2 text-sm font-medium text-[#0957f9] transition-colors group-hover:text-[#002a66]">{{ card.action }} <span aria-hidden="true">→</span></p>
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
  box-shadow: 0 10px 24px -18px var(--portal-card-glow);
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
