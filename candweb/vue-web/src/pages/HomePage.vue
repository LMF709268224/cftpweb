<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import type { Component } from "vue"
import { Award, BookOpen, CheckCircle2, ClipboardList, PackageOpen, PanelLeft, Receipt } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { getAccessToken } from "@/lib/authStorage"
import { useTranslation } from "@/lib/language"

const { t, lang } = useTranslation()
const userName = ref("...")
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


const guideCopy = computed(() => lang.value === "zh"
  ? {
      title: "欢迎来到门户",
      subtitle: "探索我们的认证、课程、网络研讨会、洞察和报告，持续提升你的专业知识。",
    }
  : {
      title: "Welcome to Portal",
      subtitle: "Explore our collection of certifications, courses, webinars, insights and reports to advance your knowledge.",
    },
)

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
  const zh = lang.value === "zh"
  const cards: PortalCard[] = [
    {
      key: "certifications",
      title: zh ? "已完成认证" : "Completed Certifications",
      value: counts.value.certifications,
      action: zh ? "点击查看认证" : "Click to explore certifications",
      href: "/certifications",
      icon: Award,
      color: "orange",
      featured: true,
    },
    {
      key: "courses",
      title: zh ? "课程进行中" : "Courses in Progress",
      value: counts.value.courses,
      action: zh ? "点击查看课程资料" : "Click to explore courses",
      href: "/certifications",
      icon: BookOpen,
      color: "purple",
      featured: true,
    },
    {
      key: "exams",
      title: zh ? "考试数" : "Available Exams",
      value: counts.value.exams,
      action: zh ? "点击查看考试" : "Click to explore exams",
      href: "/exams",
      icon: ClipboardList,
      color: "blue",
      featured: false,
    },
    {
      key: "resourcePacks",
      title: zh ? "资源包数" : "Resource Packs",
      value: counts.value.resourcePacks,
      action: zh ? "点击查看资源包" : "Click to explore resource packs",
      href: "/resource-packs",
      icon: PackageOpen,
      color: "teal",
      featured: false,
    },
    {
      key: "orders",
      title: zh ? "订单数" : "Orders",
      value: counts.value.orders,
      action: zh ? "点击查看订单" : "Click to view orders",
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
          return { ...card, title: t.value.home.purchasedCertifications, action: t.value.home.viewCertifications }
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
    panel: "from-[#fffdf2] to-[#fff3b8]",
    border: "border-[#f4e6b8]",
    hoverBorder: "hover:border-[#f6c85a]",
    text: "text-[#c55a00]",
    number: "text-[#934000]",
    accent: "#f59e0b",
    glow: "rgba(245, 158, 11, 0.22)",
  },
  purple: {
    panel: "from-[#fbf5ff] to-[#f0e0ff]",
    border: "border-[#e5d8f3]",
    hoverBorder: "hover:border-[#c084fc]",
    text: "text-[#8b22ff]",
    number: "text-[#6514c7]",
    accent: "#a855f7",
    glow: "rgba(168, 85, 247, 0.22)",
  },
  blue: {
    panel: "from-[#f4f9ff] to-[#dbeafe]",
    border: "border-[#dbe4f0]",
    hoverBorder: "hover:border-[#93c5fd]",
    text: "text-[#2563ff]",
    number: "text-[#1e40af]",
    accent: "#38bdf8",
    glow: "rgba(56, 189, 248, 0.22)",
  },
  teal: {
    panel: "from-[#effdfa] to-[#ccfbef]",
    border: "border-[#cae9e3]",
    hoverBorder: "hover:border-[#5eead4]",
    text: "text-[#0f8d7e]",
    number: "text-[#0f766e]",
    accent: "#14b8a6",
    glow: "rgba(20, 184, 166, 0.2)",
  },
  green: {
    panel: "from-[#f0fdf4] to-[#dcfce7]",
    border: "border-[#d7eadc]",
    hoverBorder: "hover:border-[#86efac]",
    text: "text-[#16a34a]",
    number: "text-[#166534]",
    accent: "#22c55e",
    glow: "rgba(34, 197, 94, 0.2)",
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
  const token = getAccessToken()
  if (!token) {
    const localName = localStorage.getItem("user_name")
    if (localName) userName.value = localName
    return
  }

  dashboardLoading.value = true
  try {
    const payload = await apiClient("/api/user/me")
    const nameToSet = payload?.display_name || payload?.name
    if (nameToSet) {
      userName.value = nameToSet
      localStorage.setItem("user_name", nameToSet)
    }
  } catch {
    const localName = localStorage.getItem("user_name")
    if (localName) userName.value = localName
  }

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

      <main class="px-5 py-10 md:px-8 lg:px-10">
        <section class="w-full text-center">
          <h1 class="text-[36px] font-bold leading-tight tracking-tight text-[#6847ff]">{{ guideCopy.title }}</h1>
          <p class="mx-auto mt-4 max-w-5xl text-lg leading-8 text-[#4a4f59]">{{ guideCopy.subtitle }}</p>
        </section>

        <section class="portal-card-section mx-auto mt-12 w-full max-w-[1380px]">
          <div v-if="showDashboardSkeleton" class="flex flex-col gap-8" role="status" :aria-label="t.common.loading" aria-live="polite">
            <div class="portal-card-row portal-card-featured-row flex flex-col items-center justify-center gap-6 lg:flex-row">
              <div
                v-for="item in 2"
                :key="`featured-skeleton-${item}`"
                class="portal-stat-card portal-card-featured portal-card-skeleton h-[214px] w-full rounded-[16px] border border-slate-100 bg-white p-8 shadow-[0_2px_8px_rgba(15,23,42,0.08)] lg:basis-[34%] lg:grow-0 lg:shrink-0"
              >
                <div class="mx-auto h-9 w-9 rounded-full bg-slate-100" />
                <div class="mx-auto mt-6 h-5 w-32 rounded-full bg-slate-100" />
                <div class="mx-auto mt-12 h-9 w-16 rounded-full bg-slate-100" />
                <div class="mx-auto mt-4 h-4 w-28 rounded-full bg-slate-100" />
              </div>
            </div>

            <div class="portal-card-row portal-card-secondary-row flex flex-col items-center justify-center gap-6 lg:flex-row">
              <div
                v-for="item in 3"
                :key="`secondary-skeleton-${item}`"
                class="portal-stat-card portal-card-secondary portal-card-skeleton h-[214px] w-full rounded-[16px] border border-slate-100 bg-white p-8 shadow-[0_2px_8px_rgba(15,23,42,0.08)] lg:basis-[29%] lg:grow-0 lg:shrink-0"
              >
                <div class="mx-auto h-9 w-9 rounded-full bg-slate-100" />
                <div class="mx-auto mt-6 h-5 w-28 rounded-full bg-slate-100" />
                <div class="mx-auto mt-12 h-9 w-14 rounded-full bg-slate-100" />
                <div class="mx-auto mt-4 h-4 w-24 rounded-full bg-slate-100" />
              </div>
            </div>
          </div>

          <div v-else class="flex flex-col gap-8">
            <div class="portal-card-row portal-card-featured-row flex flex-col items-center justify-center gap-6 lg:flex-row">
              <RouterLink
                v-for="card in featuredCards"
                :key="card.key"
                :to="card.href"
                :class="[
                  'portal-stat-card portal-card-featured group relative flex h-[214px] w-full flex-col items-center justify-center overflow-hidden rounded-[16px] border bg-gradient-to-b p-8 text-center shadow-[0_2px_8px_rgba(15,23,42,0.12)] transition-all duration-300 ease-out hover:-translate-y-1 hover:scale-[1.015] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30 lg:basis-[34%] lg:grow-0 lg:shrink-0',
                  cardStyles[card.color].panel,
                  cardStyles[card.color].border,
                  cardStyles[card.color].hoverBorder,
                ]"
                :style="{ '--portal-card-accent': cardStyles[card.color].accent, '--portal-card-glow': cardStyles[card.color].glow }"
              >
                <span class="portal-card-sheen pointer-events-none absolute left-0 top-0 h-1 w-full" />
                <span class="portal-card-orb pointer-events-none absolute -right-10 -top-12 h-32 w-32 rounded-full opacity-0 transition-opacity duration-300 group-hover:opacity-100" />
                <div class="portal-card-icon relative flex h-12 w-12 items-center justify-center rounded-xl bg-white/60 shadow-sm ring-1 ring-white/70 transition-transform duration-300 group-hover:scale-105">
                  <component :is="card.icon" :class="['h-9 w-9', cardStyles[card.color].text]" :stroke-width="2.1" />
                </div>
                <h2 :class="['relative mt-6 text-lg font-semibold', cardStyles[card.color].text]">{{ card.title }}</h2>
                <p :class="['relative mt-12 text-5xl font-bold tracking-tight', cardStyles[card.color].number]">
                  <span>{{ card.value }}</span>
                </p>
                <p :class="['relative mt-3 text-base', cardStyles[card.color].text]">{{ card.action }}</p>
              </RouterLink>
            </div>

            <div class="portal-card-row portal-card-secondary-row flex flex-col items-center justify-center gap-6 lg:flex-row">
              <RouterLink
                v-for="card in secondaryCards"
                :key="card.key"
                :to="card.href"
                :class="[
                  'portal-stat-card portal-card-secondary group relative flex h-[214px] w-full flex-col items-center justify-center overflow-hidden rounded-[16px] border bg-gradient-to-b p-8 text-center shadow-[0_2px_8px_rgba(15,23,42,0.12)] transition-all duration-300 ease-out hover:-translate-y-1 hover:scale-[1.015] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30 lg:basis-[29%] lg:grow-0 lg:shrink-0',
                  cardStyles[card.color].panel,
                  cardStyles[card.color].border,
                  cardStyles[card.color].hoverBorder,
                ]"
                :style="{ '--portal-card-accent': cardStyles[card.color].accent, '--portal-card-glow': cardStyles[card.color].glow }"
              >
                <span class="portal-card-sheen pointer-events-none absolute left-0 top-0 h-1 w-full" />
                <span class="portal-card-orb pointer-events-none absolute -right-10 -top-12 h-32 w-32 rounded-full opacity-0 transition-opacity duration-300 group-hover:opacity-100" />
                <div class="portal-card-icon relative flex h-12 w-12 items-center justify-center rounded-xl bg-white/60 shadow-sm ring-1 ring-white/70 transition-transform duration-300 group-hover:scale-105">
                  <component :is="card.icon" :class="['h-9 w-9', cardStyles[card.color].text]" :stroke-width="2.1" />
                </div>
                <h2 :class="['relative mt-6 text-lg font-semibold', cardStyles[card.color].text]">{{ card.title }}</h2>
                <p :class="['relative mt-12 text-5xl font-bold tracking-tight', cardStyles[card.color].number]">
                  <span>{{ card.value }}</span>
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
  --portal-card-accent: #38bdf8;
  --portal-card-glow: rgba(56, 189, 248, 0.18);
}

.portal-stat-card:hover {
  box-shadow: 0 18px 34px -18px var(--portal-card-glow), 0 12px 28px rgba(15, 23, 42, 0.12);
}

.portal-card-sheen {
  background: linear-gradient(90deg, transparent, var(--portal-card-accent), transparent);
  opacity: 0.78;
  transform: translateX(-105%);
  transition: transform 0.65s ease;
}

.portal-stat-card:hover .portal-card-sheen {
  transform: translateX(105%);
}

.portal-card-orb {
  background: radial-gradient(circle, var(--portal-card-glow), transparent 68%);
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

@media (min-width: 1024px) and (max-width: 1399px) {
  .portal-card-section {
    width: 875px;
    max-width: none;
  }

  .portal-card-row {
    gap: 24px;
  }

  .portal-card-featured {
    flex-basis: calc((875px - 24px) / 2) !important;
  }

  .portal-card-secondary {
    flex-basis: calc((875px - 48px) / 3) !important;
  }
}
</style>
