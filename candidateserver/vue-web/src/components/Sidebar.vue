<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { RouterLink, useRoute } from "vue-router"
import {
  Award,
  BookOpen,
  ChevronLeft,
  ChevronRight,
  Crown,
  FileText,
  Globe,
  GraduationCap,
  Home,
  LogOut,
  MessageSquare,
  Settings,
  ShoppingCart,
  User,
} from "lucide-vue-next"
import { apiClient } from "@/lib/apiClient"
import { getCachedDashboard } from "@/lib/dashboardCache"
import { useTranslation } from "@/lib/language"

const { t, lang, changeLanguage } = useTranslation()
const route = useRoute()
const collapsed = ref(false)
const userName = ref(t.value.common.user)
const unreadCount = ref(0)
const menuOpen = ref(false)

const navItems = computed(() => [
  { href: "/", icon: Home, label: t.value.sidebar.home },
  { href: "/courses", icon: BookOpen, label: t.value.sidebar.courses },
  { href: "/membership", icon: Crown, label: t.value.sidebar.membership },
  { href: "/exams", icon: FileText, label: t.value.sidebar.exams },
  { href: "/records", icon: GraduationCap, label: t.value.sidebar.records },
  { href: "/credentials", icon: Award, label: t.value.sidebar.credentials },
  { href: "/certificates", icon: Crown, label: t.value.sidebar.certificates },
  { href: "/orders", icon: ShoppingCart, label: t.value.sidebar.orders },
  { href: "/messages", icon: MessageSquare, label: t.value.sidebar.messages, badge: unreadCount.value > 0 ? unreadCount.value : undefined },
])

onMounted(async () => {
  const updateName = () => {
    userName.value = localStorage.getItem("user_name") || t.value.common.user
  }
  updateName()
  window.addEventListener("storage", updateName)
  try {
    const dashboard = await getCachedDashboard()
    if (dashboard?.unread_messages_count !== undefined) unreadCount.value = dashboard.unread_messages_count
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
    localStorage.removeItem("access_token")
    localStorage.removeItem("user_name")
    window.location.href = "/login"
  }
}
</script>

<template>
  <aside
    :class="[
      'fixed left-0 top-0 z-40 flex h-screen flex-col bg-white shadow-[10px_0_30px_rgba(15,74,82,0.05)] transition-all duration-300',
      collapsed ? 'w-[72px]' : 'w-64',
    ]"
  >
    <div class="flex h-20 items-center gap-3 px-4 py-4">
      <div class="flex h-11 w-11 items-center justify-center rounded-[18px] bg-primary text-primary-foreground shadow-[0_8px_18px_rgba(10,122,138,0.18)]">
        <GraduationCap class="h-5 w-5" />
      </div>
      <div v-if="!collapsed" class="flex flex-col">
        <span class="text-xl font-black tracking-tight text-primary">{{ t.sidebar.systemBrand }}</span>
        <span class="text-xs text-muted-foreground">{{ t.sidebar.systemName }}</span>
      </div>
    </div>

    <nav class="flex-1 space-y-1 overflow-y-auto px-3 py-4">
      <RouterLink
        v-for="item in navItems"
        :key="item.href"
        :to="item.href"
        :class="[
          'group relative flex items-center gap-3 rounded-2xl px-3 py-3 text-sm font-medium transition-all duration-200',
          route.path === item.href
            ? 'bg-primary text-primary-foreground shadow-[0_8px_18px_rgba(10,122,138,0.18)]'
            : 'text-sidebar-foreground hover:bg-[#edf8f9] hover:text-primary',
        ]"
      >
        <component :is="item.icon" :class="['h-5 w-5 shrink-0', collapsed && 'mx-auto']" />
        <template v-if="!collapsed">
          <span>{{ item.label }}</span>
          <span
            v-if="item.badge"
            :class="[
              'ml-auto inline-flex h-5 min-w-[20px] items-center justify-center rounded-full px-1.5 text-xs',
              route.path === item.href ? 'bg-primary-foreground/20 text-primary-foreground' : 'bg-primary/10 text-primary',
            ]"
          >
            {{ item.badge }}
          </span>
        </template>
        <span
          v-else-if="item.badge"
          class="absolute -right-1 -top-1 flex h-4 w-4 items-center justify-center rounded-full bg-destructive text-[10px] text-destructive-foreground"
        >
          {{ item.badge }}
        </span>
      </RouterLink>
    </nav>

    <div class="relative bg-[#f7fbfc] p-3">
      <button
        :class="['flex w-full items-center gap-3 rounded-2xl px-3 py-2.5 text-left transition-colors hover:bg-white', collapsed && 'justify-center px-0']"
        @click="menuOpen = !menuOpen"
      >
        <div class="flex h-10 w-10 items-center justify-center rounded-full bg-primary/10 font-semibold text-primary">
          {{ userName.charAt(0) }}
        </div>
        <div v-if="!collapsed" class="flex-1 overflow-hidden">
          <p class="truncate text-sm font-medium text-sidebar-foreground">{{ userName }}</p>
          <p class="truncate text-xs text-muted-foreground">{{ t.common.certifiedMember }}</p>
        </div>
      </button>

      <div v-if="menuOpen && !collapsed" class="absolute bottom-[76px] right-3 z-50 w-56 overflow-hidden rounded-2xl bg-card p-1.5 shadow-lg shadow-slate-900/10">
        <RouterLink to="/settings?tab=profile" class="flex items-center gap-2 rounded-xl px-3 py-2 text-sm hover:bg-muted">
          <User class="h-4 w-4" /> {{ t.sidebar.profile }}
        </RouterLink>
        <RouterLink to="/settings?tab=account" class="flex items-center gap-2 rounded-xl px-3 py-2 text-sm hover:bg-muted">
          <Settings class="h-4 w-4" /> {{ t.sidebar.settings }}
        </RouterLink>
        <button class="flex w-full items-center gap-2 rounded-xl px-3 py-2 text-sm hover:bg-muted" @click="changeLanguage(lang === 'zh' ? 'en' : 'zh')">
          <Globe class="h-4 w-4" /> {{ t.sidebar.switchLang }}
        </button>
        <div class="my-1 h-px bg-border" />
        <button class="flex w-full items-center gap-2 rounded-xl px-3 py-2 text-sm text-destructive hover:bg-muted" @click="handleLogout">
          <LogOut class="h-4 w-4" /> {{ t.sidebar.logout }}
        </button>
      </div>
    </div>

    <button
      class="absolute -right-3 top-20 flex h-6 w-6 items-center justify-center rounded-full bg-white text-muted-foreground shadow-md transition-colors hover:bg-accent hover:text-accent-foreground"
      @click="collapsed = !collapsed"
    >
      <ChevronRight v-if="collapsed" class="h-3 w-3" />
      <ChevronLeft v-else class="h-3 w-3" />
    </button>
  </aside>
</template>
