<script setup lang="ts">
import {
  BarChart3,
  BookOpen,
  Boxes,
  ChevronDown,
  ChevronLeft,
  ClipboardCheck,
  CreditCard,
  FileBadge,
  FileText,
  GitBranch,
  GraduationCap,
  Languages,
  LogOut,
  Mail,
  Menu,
  MessageSquare,
  Receipt,
  Settings,
  ShieldCheck,
  Wrench,
  X,
} from "lucide-vue-next"
import { computed, onMounted, onUnmounted, ref } from "vue"
import { RouterLink, RouterView, useRoute, useRouter } from "vue-router"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { clearAuthSession, getUserName } from "@/lib/authStorage"
import { useAdminLanguage } from "@/lib/language"

const router = useRouter()
const route = useRoute()
const collapsed = ref(false)
const mobileNavOpen = ref(false)
const userMenuOpen = ref(false)
const userMenuRef = ref<HTMLElement | null>(null)
const { lang, t, setAdminLanguage } = useAdminLanguage()
const copy = computed(() => t.value.layout)
const showSidebarText = computed(() => mobileNavOpen.value || !collapsed.value)
const redDots = ref<Record<string, number>>({})

function getDotCount(path: string): number {
  switch (path) {
    case "/applications": return redDots.value.applications || 0
    case "/exams": return redDots.value.exams || 0
    case "/prog": return redDots.value.prog || 0
    case "/orders": return redDots.value.orders || 0
    case "/invoices": return redDots.value.invoices || 0
    case "/messages": return redDots.value.messages || 0
    case "/mails": return redDots.value.messages || 0
    default: return 0
  }
}

async function fetchRedDots() {
  try {
    const res = await apiClient<any>("/api/system/reddots")
    redDots.value = res || {}
  } catch (err) {
    console.error("Failed to fetch red dots", err)
  }
}

const navGroups = computed(() => [
  {
    label: copy.value.groups.workbench,
    items: [
      { path: "/dashboard", label: copy.value.nav.dashboard, icon: BarChart3 },
    ],
  },
  {
    label: copy.value.groups.review,
    items: [
      { path: "/applications", label: copy.value.nav.applications, icon: ClipboardCheck },
      { path: "/exams", label: copy.value.nav.exams, icon: ClipboardCheck },
      { path: "/prog", label: copy.value.nav.prog, icon: GitBranch },
    ],
  },
  {
    label: copy.value.groups.config,
    items: [
      { path: "/lms", label: copy.value.nav.lms, icon: BookOpen },
      { path: "/pipelines", label: copy.value.nav.pipelines, icon: FileBadge },
      { path: "/bundles", label: copy.value.nav.bundles, icon: Boxes },
      { path: "/resource-packs", label: copy.value.nav.resourcePacks, icon: FileText },
      { path: "/resource-pack-files", label: copy.value.nav.resourcePackFiles, icon: FileText },
      { path: "/credentials", label: copy.value.nav.credentials, icon: ShieldCheck },
      { path: "/pdf-templates", label: copy.value.nav.pdfTemplates, icon: FileText },
    ],
  },
  {
    label: copy.value.groups.commerce,
    items: [
      { path: "/orders", label: copy.value.nav.orders, icon: CreditCard },
      { path: "/invoices", label: copy.value.nav.invoices, icon: Receipt },
    ],
  },
  {
    label: copy.value.groups.messages,
    items: [
      { path: "/messages", label: copy.value.nav.messages, icon: MessageSquare },
      { path: "/mails", label: copy.value.nav.mails, icon: Mail },
    ],
  },
  {
    label: copy.value.groups.operations,
    items: [
      { path: "/permissions", label: copy.value.nav.permissions, icon: GraduationCap },
      { path: "/admin-ops", label: copy.value.nav.adminOps, icon: Wrench },
      { path: "/audit/logs", label: copy.value.nav.auditLogs, icon: ShieldCheck },
      { path: "/pdf-requests", label: copy.value.nav.pdfRequests, icon: FileBadge },
    ],
  },
])

const userName = ref(getUserName())

function refreshUserName() {
  userName.value = getUserName()
}

function isActivePath(path: string) {
  return route.path === path || route.path.startsWith(`${path}/`)
}

function toggleLanguage() {
  setAdminLanguage(lang.value === "zh" ? "en" : "zh")
}

function closeUserMenuOnOutsideClick(event: PointerEvent) {
  if (!userMenuOpen.value) return
  const target = event.target
  if (target instanceof Node && userMenuRef.value?.contains(target)) return
  userMenuOpen.value = false
}

async function logout() {
  userMenuOpen.value = false
  try {
    await apiClient("/api/auth/logout", { method: "POST" })
  } catch {
    // Local logout should still succeed if the server session is already gone.
  } finally {
    clearAuthSession()
    toast.success(copy.value.logoutSuccess)
    router.push("/login")
  }
}

let redDotsTimer: ReturnType<typeof setInterval>
let removeAfterEach: (() => void) | undefined

onMounted(() => {
  window.addEventListener("storage", refreshUserName)
  document.addEventListener("pointerdown", closeUserMenuOnOutsideClick)
  
  fetchRedDots()
  redDotsTimer = setInterval(fetchRedDots, 5 * 60 * 1000)
  removeAfterEach = router.afterEach(() => {
    fetchRedDots()
  })
})

onUnmounted(() => {
  window.removeEventListener("storage", refreshUserName)
  document.removeEventListener("pointerdown", closeUserMenuOnOutsideClick)
  if (redDotsTimer) clearInterval(redDotsTimer)
  removeAfterEach?.()
})
</script>

<template>
  <div class="min-h-screen bg-[#f4f8fc] text-slate-950">
    <button
      v-if="!mobileNavOpen"
      class="fixed left-4 top-4 z-30 flex h-11 w-11 items-center justify-center rounded-2xl border border-slate-200 bg-white text-slate-700 shadow-lg shadow-slate-200/70 md:hidden"
      type="button"
      aria-label="打开菜单"
      @click="mobileNavOpen = true"
    >
      <Menu class="h-5 w-5" />
    </button>

    <button
      v-if="mobileNavOpen"
      class="fixed inset-0 z-30 bg-slate-950/45 backdrop-blur-[1px] md:hidden"
      type="button"
      aria-label="关闭菜单"
      @click="mobileNavOpen = false"
    />

    <aside
      class="fixed inset-y-0 left-0 z-40 flex w-[280px] flex-col border-r border-slate-200 bg-blue-50/95 shadow-xl shadow-slate-900/10 transition-all duration-200 md:z-30 md:bg-white/95 md:shadow-sm"
      :class="[
        mobileNavOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0',
        collapsed ? 'md:w-[76px]' : 'md:w-[256px]',
      ]"
    >
      <div class="flex h-20 items-center gap-3 border-b border-slate-200 px-4">
        <div class="flex h-11 w-11 items-center justify-center rounded-2xl bg-[#0b7bdc] text-white shadow-lg shadow-sky-200">
          <GraduationCap class="h-6 w-6" />
        </div>
        <div v-if="showSidebarText" class="leading-tight">
          <div class="text-lg font-black">CFTP</div>
          <div class="text-sm text-slate-500">{{ copy.systemName }}</div>
        </div>
        <button
          class="ml-auto flex h-9 w-9 items-center justify-center rounded-full text-slate-500 transition hover:bg-white/80 hover:text-slate-900 md:hidden"
          type="button"
          aria-label="关闭菜单"
          @click="mobileNavOpen = false"
        >
          <X class="h-5 w-5" />
        </button>
      </div>

      <button
        class="absolute -right-4 top-24 hidden h-8 w-8 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm md:flex"
        type="button"
        @click="collapsed = !collapsed"
      >
        <ChevronLeft class="h-4 w-4 transition-transform" :class="collapsed ? 'rotate-180' : ''" />
      </button>

      <nav class="flex-1 overflow-y-auto px-3 py-5">
        <div v-for="group in navGroups" :key="group.label" class="mb-5 last:mb-0">
          <div v-if="showSidebarText" class="mb-2 px-3 text-[11px] font-black uppercase tracking-wider text-slate-400">
            {{ group.label }}
          </div>
          <div v-else class="mx-auto mb-3 hidden h-px w-8 bg-slate-200 first:hidden md:block" />
          <div class="space-y-1">
            <RouterLink
              v-for="item in group.items"
              :key="item.path"
              :to="item.path"
              class="relative flex h-10 items-center gap-3 rounded-xl px-3 text-[15px] font-semibold text-slate-700 transition hover:bg-slate-100"
              :class="isActivePath(item.path) ? 'bg-blue-200 text-[#0b4ea2] shadow-none hover:bg-blue-200 md:!bg-[#0b4ea2] md:!text-white md:shadow-lg md:shadow-sky-200 md:hover:!bg-[#0b4ea2]' : ''"
              @click="mobileNavOpen = false"
            >
              <component :is="item.icon" class="h-5 w-5 shrink-0" />
              <span v-if="showSidebarText" class="flex-1">{{ item.label }}</span>
              <span v-if="showSidebarText && getDotCount(item.path) > 0" class="flex h-5 min-w-[20px] items-center justify-center rounded-full bg-red-500 px-1.5 text-[11px] font-bold text-white shadow-sm">
                {{ getDotCount(item.path) > 99 ? '99+' : getDotCount(item.path) }}
              </span>
              <div v-else-if="!showSidebarText && getDotCount(item.path) > 0" class="absolute right-2 top-2 h-2.5 w-2.5 rounded-full bg-red-500 shadow-sm border border-white"></div>
            </RouterLink>
          </div>
        </div>
      </nav>

      <div ref="userMenuRef" class="border-t border-slate-200 p-4">
        <button
          class="mb-3 flex w-full items-center justify-center gap-2 rounded-2xl border border-slate-200 bg-white px-3 py-2 text-sm font-bold text-slate-700 shadow-sm transition hover:bg-slate-50"
          type="button"
          :title="copy.language"
          @click="toggleLanguage"
        >
          <Languages class="h-4 w-4 shrink-0 text-slate-500" />
          <span v-if="showSidebarText">{{ copy.language }}</span>
        </button>

        <div v-if="showSidebarText && userMenuOpen" class="mb-3 overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-lg shadow-slate-200/60">
          <div class="flex items-center gap-3 px-4 py-4">
            <div class="flex h-9 w-9 items-center justify-center rounded-full bg-sky-100 font-bold text-sky-700">
              {{ userName.slice(0, 1).toUpperCase() }}
            </div>
            <div class="min-w-0">
              <div class="truncate text-sm font-bold text-slate-950">{{ userName }}</div>
              <div class="truncate text-xs text-slate-500">{{ copy.roleName }}</div>
            </div>
          </div>
          <RouterLink
            class="flex items-center gap-3 border-t border-slate-100 px-4 py-3 text-sm font-semibold text-slate-700 hover:bg-slate-50"
            to="/settings"
            @click="userMenuOpen = false; mobileNavOpen = false"
          >
            <Settings class="h-4 w-4 text-slate-500" />
            {{ copy.accountSettings }}
          </RouterLink>
          <button
            class="flex w-full items-center gap-3 border-t border-slate-100 px-4 py-3 text-left text-sm font-semibold text-slate-700 hover:bg-slate-50"
            type="button"
            @click="logout"
          >
            <LogOut class="h-4 w-4 text-slate-500" />
            {{ copy.logout }}
          </button>
        </div>

        <button
          class="flex w-full items-center gap-3 rounded-2xl px-3 py-3 text-left transition hover:bg-slate-100"
          :class="showSidebarText && userMenuOpen ? 'bg-blue-100 text-[#0b4ea2] md:bg-blue-50' : ''"
          type="button"
          @click="userMenuOpen = !userMenuOpen"
        >
          <div class="flex h-10 w-10 items-center justify-center rounded-full bg-sky-100 font-bold text-sky-700">
            {{ userName.slice(0, 1).toUpperCase() }}
          </div>
          <div v-if="showSidebarText" class="min-w-0">
            <div class="truncate text-sm font-bold">{{ userName }}</div>
            <div class="text-xs text-slate-500">{{ copy.roleName }}</div>
          </div>
          <ChevronDown
            v-if="showSidebarText"
            class="ml-auto h-4 w-4 text-slate-500 transition-transform"
            :class="userMenuOpen ? 'rotate-180' : ''"
          />
        </button>
      </div>
    </aside>

    <main class="min-h-screen pt-14 transition-all duration-200 md:pt-0" :class="collapsed ? 'md:pl-[76px]' : 'md:pl-[256px]'">
      <RouterView />
    </main>
  </div>
</template>
