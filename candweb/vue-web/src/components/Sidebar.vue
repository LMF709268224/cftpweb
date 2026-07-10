<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue"
import { RouterLink, useRoute } from "vue-router"
import { Award, BookOpen, ChevronUp, ClipboardList, Crown, FileCheck2, Home, Languages, LayoutDashboard, Loader2, LogOut, Menu, MessageSquare, Package, Settings, ShoppingBag, Store, X } from "lucide-vue-next"
import { apiClient } from "@/lib/apiClient"
import { clearAccessToken } from "@/lib/authStorage"
import { fetchUnreadCount, getCachedUnreadCount, onUnreadCountChanged } from "@/lib/unreadCountCache"
import { useTranslation } from "@/lib/language"
import { initializeSidebarCollapse, useSidebarCollapse } from "@/lib/sidebar"
import { useUser } from "@/lib/user"
import { usePolling } from "@/lib/polling"
import brandLogo from "@/assets/favicon.png"

const { t, lang, changeLanguage } = useTranslation()
const { currentUser, fetchUser } = useUser()
const { isSidebarCollapsed } = useSidebarCollapse()
const route = useRoute()
const userName = ref(t.value.common.user)
const userEmail = computed(() => currentUser.value?.email || "")
const unreadCount = ref(0)
const menuOpen = ref(false)
const mobileMenuOpen = ref(false)
const logoutLoading = ref(false)
const menuContainer = ref<HTMLElement | null>(null)
let stopUnreadCountListener: (() => void) | null = null

const navRouteGroups: Record<string, string[]> = {
  "/": ["/"],
  "/certifications": [
    "/certifications",
    "/courses",
    "/pdf-preview/lessons",
  ],
  "/my-certifications": ["/my-certifications"],
  "/exams": ["/exams"],
  "/resource-packs": [
    "/resource-packs",
    "/resource-pack-files",
    "/pdf-preview/resources",
    "/video-preview/resource-pack-files",
  ],
  "/credentials": ["/credentials"],
  "/certificates": ["/certificates"],
  "/membership": ["/membership"],
  "/orders": ["/orders", "/invoice-redirect"],
  "/messages": ["/messages"],
}

function badgeCountLabel(count: number) {
  return count >= 99 ? "99+" : String(count)
}

function isNavItemActive(href: string) {
  const currentPath = route.path
  const groups = navRouteGroups[href] || [href]

  return groups.some((group) => {
    if (group === "/") return currentPath === "/"
    return currentPath === group || currentPath.startsWith(`${group}/`)
  })
}

const navItems = computed(() => [
  { href: "/", label: t.value.sidebar.home, group: "" },
  { href: "/certifications", label: t.value.sidebar.courses, group: t.value.sidebar.groupLearning },
  { href: "/my-certifications", label: t.value.sidebar.myCertifications, group: t.value.sidebar.groupLearning },
  { href: "/exams", label: t.value.sidebar.exams, group: t.value.sidebar.groupLearning },
  { href: "/resource-packs", label: t.value.sidebar.resourcePacks, group: t.value.sidebar.groupLearning },
  { href: "/credentials", label: t.value.sidebar.credentials, group: t.value.sidebar.groupLearning },
  { href: "/certificates", label: t.value.sidebar.certificates, group: t.value.sidebar.groupMine },
  { href: "/membership", label: t.value.sidebar.membership, group: t.value.sidebar.groupMine },
  { href: "/orders", label: t.value.sidebar.orders, group: t.value.sidebar.groupMine },
  { href: "/messages", label: t.value.sidebar.messages, group: t.value.sidebar.groupMine, badge: unreadCount.value > 0 ? badgeCountLabel(unreadCount.value) : undefined },
])

const navIconByHref = {
  "/": LayoutDashboard,
  "/certifications": Store,
  "/my-certifications": BookOpen,
  "/exams": ClipboardList,
  "/resource-packs": Package,
  "/credentials": FileCheck2,
  "/certificates": Award,
  "/membership": Crown,
  "/orders": ShoppingBag,
  "/messages": MessageSquare,
}

function navIconFor(href: string) {
  return navIconByHref[href as keyof typeof navIconByHref] || Home
}

function updateName() {
  userName.value = localStorage.getItem("user_name") || t.value.common.user
}

function handlePointerDown(event: PointerEvent) {
  if (!menuOpen.value) return
  const target = event.target as Node | null
  if (target && menuContainer.value?.contains(target)) return
  menuOpen.value = false
}

function openMobileSidebar() {
  mobileMenuOpen.value = true
}

const unreadCountPolling = usePolling(async () => {
  unreadCount.value = await fetchUnreadCount(true)
})

onMounted(async () => {
  initializeSidebarCollapse()
  updateName()
  fetchUser()
  window.addEventListener("open-mobile-sidebar", openMobileSidebar)
  window.addEventListener("storage", updateName)
  window.addEventListener("pointerdown", handlePointerDown)
  stopUnreadCountListener = onUnreadCountChanged((value) => {
    unreadCount.value = value
  })
  try {
    unreadCount.value = await getCachedUnreadCount()
  } catch {
    // Sidebar should never block page rendering.
  }
  unreadCountPolling.start()
})

onBeforeUnmount(() => {
  window.removeEventListener("open-mobile-sidebar", openMobileSidebar)
  window.removeEventListener("storage", updateName)
  window.removeEventListener("pointerdown", handlePointerDown)
  stopUnreadCountListener?.()
})

async function handleLogout() {
  if (logoutLoading.value) return
  logoutLoading.value = true
  try {
    await apiClient("/api/auth/logout", { method: "POST" })
  } catch {
    // apiClient already shows localized errors.
  } finally {
    clearAccessToken()
    localStorage.removeItem("user_name")
    window.location.href = "/login"
  }
}
</script>

<template>
  <header class="hidden">
    <div class="flex w-full items-center justify-between px-4">
      <button class="flex h-10 w-10 cursor-pointer items-center justify-center rounded-lg bg-primary/10 text-primary transition-colors hover:bg-primary/15" @click="mobileMenuOpen = true">
        <Menu class="h-5 w-5" />
      </button>

      <RouterLink to="/" class="flex items-center gap-3">
        <img :src="brandLogo" alt="Global Fintech Institute" class="h-8 w-8 rounded-lg object-contain" />
        <span class="text-[14px] font-semibold text-[#003A70]">Global Fintech Institute</span>
      </RouterLink>

      <button
        class="inline-flex h-10 items-center justify-center gap-1.5 rounded-full border border-border bg-white px-3 text-xs font-semibold text-slate-700 shadow-sm transition-colors hover:border-primary/30 hover:bg-primary/5 hover:text-primary"
        type="button"
        @click="changeLanguage(lang === 'zh' ? 'en' : 'zh')"
      >
        <Languages class="h-4 w-4" />
        <span>{{ t.sidebar.languageCompact }}</span>
      </button>
    </div>
  </header>

  <div v-if="mobileMenuOpen" class="fixed inset-0 z-50 lg:hidden">
    <div class="absolute inset-0 bg-slate-950/35" @click="mobileMenuOpen = false" />
    <aside class="app-side-card absolute left-0 top-0 flex h-full w-[280px] max-w-[82vw] max-h-none flex-col overflow-y-auto rounded-none border-r border-sidebar-border bg-sidebar shadow-2xl shadow-slate-950/20">
      <div class="flex h-20 items-center justify-between px-5">
        <RouterLink to="/" class="flex items-center gap-3" @click="mobileMenuOpen = false">
          <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-white shadow-sm">
            <img :src="brandLogo" alt="Global Fintech Institute" class="h-7 w-7 rounded-md object-contain" />
          </span>
          <span class="text-[14px] font-semibold text-[#003A70]">Global Fintech Institute</span>
        </RouterLink>
        <button class="flex h-9 w-9 cursor-pointer items-center justify-center rounded-lg text-slate-500 transition-colors hover:bg-sidebar-accent hover:text-primary" @click="mobileMenuOpen = false">
          <X class="h-5 w-5" />
        </button>
      </div>

      <nav class="flex-1 space-y-1 px-4 py-4 text-[14px] text-sidebar-foreground">
        <div class="px-1 pb-2 pt-1 text-xs font-medium text-[#5878ad]">Menu</div>
        <div
          v-for="(item, index) in navItems"
          :key="item.href"
        >
          <div
            v-if="item.group && item.group !== navItems[index - 1]?.group"
            class="px-4 pb-1 pt-4 text-xs font-medium text-[#5878ad]"
          >
            {{ item.group }}
          </div>
          <RouterLink
            :to="item.href"
            :class="[
              'group/nav-item flex h-8 items-center justify-between rounded-xl px-4 transition-colors duration-200',
              isNavItemActive(item.href) ? 'bg-sidebar-accent font-medium text-sidebar-accent-foreground' : 'hover:bg-[#bfd4fb] hover:text-sidebar-accent-foreground',
            ]"
            @click="mobileMenuOpen = false"
          >
            <span class="flex min-w-0 items-center gap-3">
              <component :is="navIconFor(item.href)" :class="['h-4 w-4 shrink-0 transition-all duration-200 ease-out', isNavItemActive(item.href) ? 'text-sidebar-accent-foreground' : 'text-[#2f5597] group-hover/nav-item:scale-[1.05] group-hover/nav-item:text-sidebar-accent-foreground']" :stroke-width="1.9" />
              <span class="truncate">{{ item.label }}</span>
            </span>
            <span v-if="item.badge" class="rounded-full bg-primary/10 px-1.5 py-0.5 text-xs font-semibold text-primary">{{ item.badge }}</span>
          </RouterLink>
        </div>
      </nav>

      <div class="px-4 pb-5">
        <button
          class="mb-3 flex h-8 w-full cursor-pointer items-center justify-center gap-2 rounded-full border border-white/70 bg-white/75 px-4 text-sm font-semibold text-[#2f5597] shadow-sm backdrop-blur transition-colors hover:bg-white"
          type="button"
          @click="changeLanguage(lang === 'zh' ? 'en' : 'zh')"
        >
          <Languages class="h-4 w-4" />
          <span>{{ t.sidebar.languageToggle }}</span>
        </button>

        <button
          class="flex h-12 w-full items-center gap-4 rounded-2xl px-4 text-left transition-colors hover:bg-sidebar-accent"
          :class="menuOpen ? 'bg-sidebar-accent' : ''"
          type="button"
          @pointerdown.stop
          @click="menuOpen = !menuOpen"
        >
          <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-[#eeeeee] text-sm font-bold text-black">
            {{ userName.charAt(0).toUpperCase() }}
          </span>
          <span class="min-w-0 flex-1 truncate text-sm font-medium text-sidebar-accent-foreground">{{ userName }}</span>
          <ChevronUp class="h-4 w-4 shrink-0 text-sidebar-accent-foreground transition-transform" :class="menuOpen ? '' : 'rotate-180'" :stroke-width="2" />
        </button>
      </div>
    </aside>
  </div>

  <aside
    :class="[
      'app-side-card fixed left-0 top-0 z-30 hidden h-screen overflow-y-auto border-r border-sidebar-border bg-sidebar transition-[width] duration-300 ease-out lg:flex lg:flex-col',
      isSidebarCollapsed ? 'w-14' : 'w-[280px]',
    ]"
  >
    <RouterLink
      to="/"
      :class="[
        'flex items-center transition-all duration-300',
        isSidebarCollapsed ? 'h-14' : 'h-20',
        isSidebarCollapsed ? 'justify-center px-0' : 'gap-4 px-8',
      ]"
      :title="isSidebarCollapsed ? 'Global Fintech Institute' : undefined"
    >
      <span :class="['flex shrink-0 items-center justify-center rounded-lg bg-white shadow-sm', isSidebarCollapsed ? 'h-8 w-8' : 'h-9 w-9']">
        <img :src="brandLogo" alt="Global Fintech Institute" :class="['rounded-md object-contain', isSidebarCollapsed ? 'h-7 w-7' : 'h-7 w-7']" />
      </span>
      <div v-if="!isSidebarCollapsed" class="min-w-0 flex-1 whitespace-nowrap text-[14px] font-semibold leading-5 text-[#003A70]">
        Global Fintech Institute
      </div>
    </RouterLink>

    <nav :class="['flex-1 space-y-1 text-[14px] text-sidebar-foreground', isSidebarCollapsed ? 'px-0 py-0' : 'px-4 py-3']">
      <div v-if="!isSidebarCollapsed" class="px-1 pb-2 text-xs font-medium text-[#5878ad]">Menu</div>
      <div
        v-for="(item, index) in navItems"
        :key="item.href"
      >
        <div
          v-if="!isSidebarCollapsed && item.group && item.group !== navItems[index - 1]?.group"
          class="px-4 pb-1 pt-4 text-xs font-medium text-[#5878ad]"
        >
          {{ item.group }}
        </div>
        <RouterLink
          :to="item.href"
          :title="isSidebarCollapsed ? item.label : undefined"
          :class="[
            'group/nav-item relative flex h-8 items-center rounded-xl transition-colors duration-200',
            isSidebarCollapsed ? 'mx-auto w-8 justify-center px-0' : 'justify-between px-4',
            isNavItemActive(item.href) ? 'bg-sidebar-accent font-medium text-sidebar-accent-foreground' : 'hover:bg-[#bfd4fb] hover:text-sidebar-accent-foreground',
          ]"
        >
          <span :class="['flex min-w-0 items-center', isSidebarCollapsed ? 'justify-center' : 'gap-3']">
            <component :is="navIconFor(item.href)" :class="['h-4 w-4 shrink-0 transition-all duration-200 ease-out', isNavItemActive(item.href) ? 'text-sidebar-accent-foreground' : 'text-[#2f5597] group-hover/nav-item:scale-[1.05] group-hover/nav-item:text-sidebar-accent-foreground']" :stroke-width="1.9" />
            <span v-if="!isSidebarCollapsed" class="truncate">{{ item.label }}</span>
          </span>
          <span v-if="item.badge && !isSidebarCollapsed" class="rounded-full bg-primary/10 px-1.5 py-0.5 text-xs font-semibold text-primary">{{ item.badge }}</span>
          <span v-if="item.badge && isSidebarCollapsed" class="absolute right-1.5 top-1.5 min-w-4 rounded-full bg-primary px-1 text-center text-[10px] font-bold leading-4 text-white">{{ item.badge }}</span>
        </RouterLink>
      </div>
    </nav>

    <div :class="[isSidebarCollapsed ? 'px-0' : 'px-5', 'pb-6']">
      <button
        :class="[
          'mb-3 flex h-8 w-full cursor-pointer items-center justify-center gap-2 border border-white/70 bg-white/75 text-sm font-semibold text-[#2f5597] shadow-sm backdrop-blur transition-colors hover:bg-white',
          isSidebarCollapsed ? 'mx-auto w-8 rounded-xl px-0' : 'rounded-full px-4',
        ]"
        type="button"
        @click="changeLanguage(lang === 'zh' ? 'en' : 'zh')"
        :title="lang === 'zh' ? t.sidebar.englishTitle : t.sidebar.chineseTitle"
      >
        <Languages class="h-4 w-4" />
        <span>{{ t.sidebar.languageToggle }}</span>
      </button>

      <button
        :class="[
          'flex h-12 w-full items-center rounded-2xl text-left transition-colors hover:bg-sidebar-accent',
          isSidebarCollapsed ? 'mx-auto h-8 w-8 justify-center px-0' : 'gap-4 px-4',
          menuOpen ? 'bg-sidebar-accent' : '',
        ]"
        type="button"
        @pointerdown.stop
        @click="menuOpen = !menuOpen"
        :title="isSidebarCollapsed ? userName : undefined"
      >
        <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-[#eeeeee] text-sm font-bold text-black">
          {{ userName.charAt(0).toUpperCase() }}
        </span>
        <span v-if="!isSidebarCollapsed" class="min-w-0 flex-1 truncate text-sm font-medium text-sidebar-accent-foreground">{{ userName }}</span>
        <ChevronUp v-if="!isSidebarCollapsed" class="h-4 w-4 shrink-0 text-sidebar-accent-foreground transition-transform" :class="menuOpen ? '' : 'rotate-180'" :stroke-width="2" />
      </button>
    </div>
  </aside>

  <div ref="menuContainer" class="fixed bottom-5 left-5 z-50">
    <div v-if="menuOpen" class="mb-2 w-[240px] overflow-hidden rounded-2xl border border-border bg-white shadow-xl shadow-slate-950/10 lg:mb-[60px]">
      <RouterLink
        to="/settings?tab=profile"
        class="flex h-[76px] cursor-pointer items-center gap-4 border-b border-border px-4 text-slate-900 transition-colors hover:bg-slate-50"
        @click="menuOpen = false"
      >
        <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-[#eeeeee] text-sm font-bold text-black">
          {{ userName.charAt(0).toUpperCase() }}
        </span>
        <span class="min-w-0 flex-1">
          <span class="block truncate text-sm font-semibold leading-5">{{ userName }}</span>
          <span v-if="userEmail" class="mt-0.5 block truncate text-xs leading-4 text-slate-500">{{ userEmail }}</span>
        </span>
      </RouterLink>
      <RouterLink
        to="/settings?tab=account"
        class="group/menu-row flex h-12 cursor-pointer items-center px-2 text-sm text-slate-900"
        @click="menuOpen = false"
      >
        <span class="flex h-8 w-full items-center gap-4 rounded-lg px-3 transition-colors group-hover/menu-row:bg-primary group-hover/menu-row:text-white">
          <Settings class="h-5 w-5 text-slate-500 transition-colors group-hover/menu-row:text-white" />
          <span>{{ t.sidebar.settings }}</span>
        </span>
      </RouterLink>
      <button class="group/menu-row flex h-12 w-full cursor-pointer items-center border-t border-border px-2 text-left text-sm text-slate-900 disabled:cursor-not-allowed disabled:opacity-70" :disabled="logoutLoading" @click="handleLogout">
        <span class="flex h-8 w-full items-center gap-4 rounded-lg px-3 transition-colors group-hover/menu-row:bg-primary group-hover/menu-row:text-white">
          <Loader2 v-if="logoutLoading" class="h-5 w-5 animate-spin text-slate-500 transition-colors group-hover/menu-row:text-white" />
          <LogOut v-else class="h-5 w-5 text-slate-500 transition-colors group-hover/menu-row:text-white" />
          <span>{{ t.sidebar.logout }}</span>
        </span>
      </button>
    </div>

    <button class="hidden h-12 w-12 cursor-pointer items-center justify-center rounded-full bg-primary text-lg font-black text-white shadow-lg shadow-primary/25 transition-transform hover:scale-105" @click="menuOpen = !menuOpen">
      {{ userName.charAt(0).toUpperCase() }}
    </button>
  </div>
</template>
