<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { RouterLink, useRoute } from "vue-router"
import { Languages, LogOut, Menu, Settings, User, X } from "lucide-vue-next"
import { apiClient } from "@/lib/apiClient"
import { clearAccessToken } from "@/lib/authStorage"
import { getCachedUnreadCount } from "@/lib/unreadCountCache"
import { useTranslation } from "@/lib/language"

const { t, lang, changeLanguage } = useTranslation()
const route = useRoute()
const userName = ref(t.value.common.user)
const unreadCount = ref(0)
const menuOpen = ref(false)
const mobileMenuOpen = ref(false)

const navRouteGroups: Record<string, string[]> = {
  "/": ["/"],
  "/certifications": [
    "/certifications",
    "/courses",
    "/pdf-preview/lessons",
  ],
  "/exams": ["/exams"],
  "/resource-packs": [
    "/resource-packs",
    "/resource-pack-files",
    "/pdf-preview/resources",
    "/video-preview/resource-pack-files",
  ],
  "/credentials": ["/credentials"],
  "/certificates": ["/certificates"],
  "/orders": ["/orders", "/invoice-redirect"],
  "/messages": ["/messages"],
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
  { href: "/", label: t.value.sidebar.home },
  { href: "/certifications", label: t.value.sidebar.courses },
  { href: "/exams", label: t.value.sidebar.exams },
  { href: "/resource-packs", label: lang.value === "zh" ? "资源包" : "Resources" },
  { href: "/credentials", label: t.value.sidebar.credentials },
  { href: "/certificates", label: t.value.sidebar.certificates },
  { href: "/orders", label: t.value.sidebar.orders },
  { href: "/messages", label: t.value.sidebar.messages, badge: unreadCount.value > 0 ? unreadCount.value : undefined },
])

onMounted(async () => {
  const updateName = () => {
    userName.value = localStorage.getItem("user_name") || t.value.common.user
  }
  updateName()
  window.addEventListener("storage", updateName)
  try {
    unreadCount.value = await getCachedUnreadCount()
  } catch {
    // Sidebar should never block page rendering.
  }
})

async function handleLogout() {
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
  <header class="fixed left-0 right-0 top-0 z-40 flex h-20 items-center border-b border-border bg-white lg:hidden">
    <div class="flex w-full items-center justify-between px-4">
      <button class="flex h-10 w-10 cursor-pointer items-center justify-center rounded-lg bg-primary/10 text-primary transition-colors hover:bg-primary/15" @click="mobileMenuOpen = true">
        <Menu class="h-5 w-5" />
      </button>

      <RouterLink to="/" class="flex h-9 w-14 items-center justify-center rounded-md bg-primary text-xl font-black tracking-tight text-white shadow-sm">
        CFtP
      </RouterLink>

      <div class="h-10 w-10" />
    </div>
  </header>

  <div v-if="mobileMenuOpen" class="fixed inset-0 z-50 lg:hidden">
    <div class="absolute inset-0 bg-slate-950/35" @click="mobileMenuOpen = false" />
    <aside class="app-side-card absolute left-0 top-0 h-full w-[248px] max-w-[78vw] max-h-none overflow-y-auto rounded-none border-r border-sidebar-border bg-sidebar shadow-2xl shadow-slate-950/20">
      <div class="flex h-20 items-center justify-between px-5">
        <RouterLink to="/" class="flex h-9 w-14 items-center justify-center rounded-md bg-primary text-xl font-black tracking-tight text-white shadow-sm" @click="mobileMenuOpen = false">
          CFtP
        </RouterLink>
        <button class="flex h-9 w-9 cursor-pointer items-center justify-center rounded-lg text-slate-500 transition-colors hover:bg-sidebar-accent hover:text-primary" @click="mobileMenuOpen = false">
          <X class="h-5 w-5" />
        </button>
      </div>

      <div class="border-y border-sidebar-border px-5 py-6 text-center">
        <div class="mx-auto flex h-[54px] w-[54px] items-center justify-center rounded-full bg-primary/10 text-2xl font-black text-primary">
          {{ userName.charAt(0).toUpperCase() }}
        </div>
        <h2 class="mt-3 truncate text-sm font-bold text-foreground">{{ userName }}</h2>
      </div>

      <nav class="space-y-1 px-3 py-4 text-[15px] text-sidebar-foreground">
        <RouterLink
          v-for="item in navItems"
          :key="item.href"
          :to="item.href"
          :class="[
            'flex items-center justify-between rounded-lg px-4 py-2.5 transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
            isNavItemActive(item.href) ? 'bg-sidebar-accent font-semibold text-sidebar-accent-foreground' : '',
          ]"
          @click="mobileMenuOpen = false"
        >
          <span>{{ item.label }}</span>
          <span v-if="item.badge" class="rounded-full bg-primary/10 px-1.5 py-0.5 text-xs font-semibold text-primary">{{ item.badge }}</span>
        </RouterLink>
      </nav>
    </aside>
  </div>

  <aside class="app-side-card fixed left-0 top-0 z-30 hidden h-screen w-[240px] overflow-y-auto border-r border-sidebar-border bg-sidebar lg:block">
    <RouterLink to="/" class="flex h-24 items-center gap-3 px-5">
      <div class="flex h-9 w-11 items-center justify-center rounded-md bg-primary text-base font-black tracking-tight text-white shadow-sm">
        CFtP
      </div>
      <div class="min-w-0">
        <div class="truncate text-sm font-bold leading-5 text-foreground">CFtP</div>
        <div class="truncate text-xs text-muted-foreground">{{ lang === "zh" ? "培训系统" : "Training Portal" }}</div>
      </div>
    </RouterLink>

    <div class="h-px bg-sidebar-border" />

    <div class="px-5 pb-3 pt-5 text-xs font-bold uppercase tracking-wide text-slate-400">Menu</div>
    <nav class="space-y-1 px-3 text-sm text-sidebar-foreground">
      <RouterLink
        v-for="item in navItems"
        :key="item.href"
        :to="item.href"
        :class="[
          'flex items-center justify-between rounded-lg px-3 py-2.5 transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
          isNavItemActive(item.href) ? 'bg-sidebar-accent font-semibold text-sidebar-accent-foreground' : '',
        ]"
      >
        <span>{{ item.label }}</span>
        <span v-if="item.badge" class="rounded-full bg-primary/10 px-1.5 py-0.5 text-xs font-semibold text-primary">{{ item.badge }}</span>
      </RouterLink>
    </nav>
  </aside>

  <div class="fixed bottom-5 left-5 z-50">
    <div v-if="menuOpen" class="mb-3 w-44 rounded-xl border border-border bg-white p-1.5 shadow-xl shadow-slate-950/10">
      <RouterLink
        to="/settings?tab=profile"
        class="flex cursor-pointer items-center gap-2 rounded-lg px-3 py-2 text-sm text-slate-600 transition-colors hover:bg-sidebar-accent hover:text-primary"
        @click="menuOpen = false"
      >
        <User class="h-4 w-4" />
        {{ t.sidebar.profile }}
      </RouterLink>
      <RouterLink
        to="/settings?tab=account"
        class="flex cursor-pointer items-center gap-2 rounded-lg px-3 py-2 text-sm text-slate-600 transition-colors hover:bg-sidebar-accent hover:text-primary"
        @click="menuOpen = false"
      >
        <Settings class="h-4 w-4" />
        {{ t.sidebar.settings }}
      </RouterLink>
      <button class="flex w-full cursor-pointer items-center gap-2 rounded-lg px-3 py-2 text-sm text-slate-600 transition-colors hover:bg-sidebar-accent hover:text-primary" @click="changeLanguage(lang === 'zh' ? 'en' : 'zh')">
        <Languages class="h-4 w-4" />
        {{ t.sidebar.switchLang }}
      </button>
      <div class="my-1 h-px bg-border" />
      <button class="flex w-full cursor-pointer items-center gap-2 rounded-lg px-3 py-2 text-sm text-red-600 transition-colors hover:bg-red-50" @click="handleLogout">
        <LogOut class="h-4 w-4" />
        {{ t.sidebar.logout }}
      </button>
    </div>

    <button class="flex h-12 w-12 cursor-pointer items-center justify-center rounded-full bg-primary text-lg font-black text-white shadow-lg shadow-primary/25 transition-transform hover:scale-105" @click="menuOpen = !menuOpen">
      {{ userName.charAt(0).toUpperCase() }}
    </button>
  </div>
</template>
