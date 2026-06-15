<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { RouterLink, useRoute } from "vue-router"
import { LogOut } from "lucide-vue-next"
import { apiClient } from "@/lib/apiClient"
import { getCachedDashboard } from "@/lib/dashboardCache"
import { useTranslation } from "@/lib/language"

const { t, lang, changeLanguage } = useTranslation()
const route = useRoute()
const userName = ref(t.value.common.user)
const unreadCount = ref(0)
const menuOpen = ref(false)

const activeSettingsTab = computed(() => String(route.query.tab || "profile"))

const navItems = computed(() => [
  { href: "/", label: t.value.sidebar.home },
  { href: "/courses", label: t.value.sidebar.courses },
  { href: "/membership", label: t.value.sidebar.membership },
  { href: "/exams", label: t.value.sidebar.exams },
  { href: "/records", label: t.value.sidebar.records },
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
  <header class="fixed left-0 right-0 top-0 z-40 flex h-[84px] items-center bg-white shadow-[0_1px_0_rgba(15,23,42,0.06)]">
    <div class="mx-auto flex w-full max-w-[1280px] items-center justify-between px-4">
      <RouterLink to="/" class="flex h-9 w-14 items-center justify-center rounded-md bg-primary text-xl font-black tracking-tight text-white shadow-sm">
        CFtP
      </RouterLink>

      <nav class="ml-auto flex h-10 items-center gap-8 text-sm text-slate-500">
        <RouterLink
          to="/settings?tab=profile"
          :class="['relative flex h-10 items-center px-1 transition-colors hover:text-primary', route.path === '/settings' && activeSettingsTab === 'profile' ? 'text-primary' : '']"
        >
          <span>{{ t.sidebar.profile }}</span>
          <span v-if="route.path === '/settings' && activeSettingsTab === 'profile'" class="absolute bottom-1 left-1/2 h-0.5 w-6 -translate-x-1/2 rounded-full bg-primary" />
        </RouterLink>
        <RouterLink
          to="/settings?tab=account"
          :class="['relative flex h-10 items-center px-1 transition-colors hover:text-primary', route.path === '/settings' && activeSettingsTab === 'account' ? 'text-primary' : '']"
        >
          <span>{{ t.sidebar.settings }}</span>
          <span v-if="route.path === '/settings' && activeSettingsTab === 'account'" class="absolute bottom-1 left-1/2 h-0.5 w-6 -translate-x-1/2 rounded-full bg-primary" />
        </RouterLink>
        <button class="flex h-10 cursor-pointer items-center px-1 transition-colors hover:text-primary" @click="changeLanguage(lang === 'zh' ? 'en' : 'zh')">{{ t.sidebar.switchLang }}</button>
      </nav>

      <div class="relative ml-8">
        <button class="flex h-10 w-10 cursor-pointer items-center justify-center rounded-full bg-red-50 text-lg font-black text-primary transition-transform hover:scale-105" @click="menuOpen = !menuOpen">
          {{ userName.charAt(0).toUpperCase() }}
        </button>
        <div v-if="menuOpen" class="absolute right-0 top-12 z-50 w-32 rounded-xl bg-white p-1.5 shadow-lg shadow-slate-900/10">
          <button class="flex w-full cursor-pointer items-center justify-center gap-2 rounded-lg px-3 py-2 text-sm text-primary hover:bg-primary/10" @click="handleLogout">
            <LogOut class="h-4 w-4" />
            {{ t.sidebar.logout }}
          </button>
        </div>
      </div>
    </div>
  </header>

  <aside class="app-side-card fixed top-[106px] z-30 hidden w-[220px] overflow-y-auto rounded-md bg-white lg:block">
    <div class="px-8 pb-9 pt-8 text-center">
      <div class="mx-auto flex h-[72px] w-[72px] items-center justify-center rounded-full bg-red-50 text-3xl font-black text-primary">
        {{ userName.charAt(0).toUpperCase() }}
      </div>
      <h2 class="mt-4 truncate text-base font-bold text-slate-950">{{ userName }}</h2>
      <p class="mt-3 whitespace-nowrap text-sm text-slate-500">
        {{ lang === 'zh' ? '会员：' : 'Member: ' }}{{ t.common.certifiedMember }}
      </p>
    </div>

    <div class="h-px bg-slate-100" />

    <nav class="py-6 text-center text-base text-slate-950">
      <RouterLink
        v-for="item in navItems"
        :key="item.href"
        :to="item.href"
        :class="[
          'relative block py-2.5 transition-colors hover:text-primary',
          route.path === item.href ? 'font-medium text-primary' : '',
        ]"
      >
        <span v-if="route.path === item.href" class="absolute left-0 top-1/2 h-8 w-0.5 -translate-y-1/2 bg-primary" />
        {{ item.label }}
        <span v-if="item.badge" class="ml-1 rounded-full bg-primary/10 px-1.5 py-0.5 text-xs text-primary">{{ item.badge }}</span>
      </RouterLink>
    </nav>
  </aside>
</template>
