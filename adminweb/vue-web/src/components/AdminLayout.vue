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

onMounted(() => {
  window.addEventListener("storage", refreshUserName)
  document.addEventListener("pointerdown", closeUserMenuOnOutsideClick)

  fetchRedDots()
  redDotsTimer = setInterval(fetchRedDots, 5 * 60 * 1000)
  router.afterEach(() => {
    fetchRedDots()
  })
})

onUnmounted(() => {
  window.removeEventListener("storage", refreshUserName)
  document.removeEventListener("pointerdown", closeUserMenuOnOutsideClick)
  if (redDotsTimer) clearInterval(redDotsTimer)
})
</script>

<template>
  <div class="min-h-screen bg-canvas text-ink">
    <button
      v-if="!mobileNavOpen"
      class="fixed left-4 top-4 z-30 flex h-10 w-10 items-center justify-center rounded-[10px] border border-slate-200 bg-white text-slate-600 shadow-sm transition-colors hover:bg-slate-50 md:hidden"
      type="button"
      aria-label="打开菜单"
      @click="mobileNavOpen = true"
    >
      <Menu class="h-5 w-5" />
    </button>

    <Transition name="modal">
      <button
        v-if="mobileNavOpen"
        class="fixed inset-0 z-30 bg-slate-950/40 backdrop-blur-[2px] md:hidden"
        type="button"
        aria-label="关闭菜单"
        @click="mobileNavOpen = false"
      />
    </Transition>

    <aside
      class="fixed inset-y-0 left-0 z-40 flex w-[280px] flex-col border-r border-edge bg-white transition-all duration-200 md:z-30"
      :class="[
        mobileNavOpen ? 'translate-x-0 shadow-2xl' : '-translate-x-full md:translate-x-0',
        collapsed ? 'md:w-[76px]' : 'md:w-[264px]',
      ]"
    >
      <!-- Brand -->
      <div class="flex h-16 shrink-0 items-center gap-3 border-b border-hairline px-4">
        <div
          class="flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px] text-white shadow-sm"
          style="background: linear-gradient(135deg, #6366f1, #8b5cf6)"
        >
          <GraduationCap class="h-5 w-5" />
        </div>
        <div v-if="showSidebarText" class="min-w-0 leading-tight">
          <div class="text-[15px] font-semibold tracking-tight text-ink">CFTP</div>
          <div class="truncate text-xs text-slate-400">{{ copy.systemName }}</div>
        </div>
        <button
          class="ml-auto flex h-8 w-8 items-center justify-center rounded-lg text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600 md:hidden"
          type="button"
          aria-label="关闭菜单"
          @click="mobileNavOpen = false"
        >
          <X class="h-5 w-5" />
        </button>
      </div>

      <!-- Collapse toggle (desktop) -->
      <button
        class="absolute -right-3.5 top-[72px] hidden h-7 w-7 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-400 shadow-sm transition-colors hover:text-slate-600 md:flex"
        type="button"
        aria-label="折叠侧边栏"
        @click="collapsed = !collapsed"
      >
        <ChevronLeft class="h-4 w-4 transition-transform duration-200" :class="collapsed ? 'rotate-180' : ''" />
      </button>

      <!-- Navigation -->
      <nav class="flex-1 overflow-y-auto px-3 py-4">
        <div v-for="group in navGroups" :key="group.label" class="mb-5 last:mb-0">
          <div
            v-if="showSidebarText"
            class="mb-1.5 px-3 text-[11px] font-medium uppercase tracking-wider text-slate-400"
          >
            {{ group.label }}
          </div>
          <div v-else class="mx-auto mb-3 h-px w-8 bg-hairline" />
          <div class="space-y-0.5">
            <RouterLink
              v-for="item in group.items"
              :key="item.path"
              :to="item.path"
              class="group relative flex h-9 items-center gap-2.5 rounded-lg px-3 text-[13.5px] font-medium transition-colors duration-150"
              :class="isActivePath(item.path)
                ? 'bg-primary-50 text-primary-700'
                : 'text-slate-600 hover:bg-slate-100/70 hover:text-slate-900'"
              @click="mobileNavOpen = false"
            >
              <span
                v-if="isActivePath(item.path)"
                class="absolute left-0 top-1/2 h-4 w-[3px] -translate-y-1/2 rounded-full bg-primary-600"
              />
              <component
                :is="item.icon"
                class="h-[18px] w-[18px] shrink-0 transition-colors"
                :class="isActivePath(item.path) ? 'text-primary-600' : 'text-slate-400 group-hover:text-slate-600'"
              />
              <span v-if="showSidebarText" class="min-w-0 flex-1 truncate">{{ item.label }}</span>
              <span
                v-if="showSidebarText && getDotCount(item.path) > 0"
                class="tnum flex h-[18px] min-w-[18px] items-center justify-center rounded-full bg-red-50 px-1.5 text-[10.5px] font-semibold text-red-600"
              >
                {{ getDotCount(item.path) > 99 ? '99+' : getDotCount(item.path) }}
              </span>
              <span
                v-else-if="!showSidebarText && getDotCount(item.path) > 0"
                class="absolute right-1.5 top-1.5 h-2 w-2 rounded-full bg-red-500 ring-2 ring-white"
              />
            </RouterLink>
          </div>
        </div>
      </nav>

      <!-- Footer: language + user -->
      <div ref="userMenuRef" class="relative shrink-0 border-t border-hairline p-3">
        <button
          class="mb-2 flex w-full items-center justify-center gap-2 rounded-[10px] border border-slate-200 bg-white px-3 py-2 text-xs font-medium text-slate-600 transition-colors hover:border-slate-300 hover:bg-slate-50"
          type="button"
          :title="copy.language"
          @click="toggleLanguage"
        >
          <Languages class="h-4 w-4 shrink-0 text-slate-400" />
          <span v-if="showSidebarText">{{ copy.language }}</span>
        </button>

        <Transition name="menu">
          <div
            v-if="showSidebarText && userMenuOpen"
            class="absolute inset-x-3 bottom-full z-10 mb-2 origin-bottom overflow-hidden rounded-2xl border border-edge bg-white shadow-[0_1px_2px_rgb(15_23_42/0.04),0_16px_40px_-12px_rgb(15_23_42/0.18)]"
          >
            <div class="flex items-center gap-3 px-4 py-3.5">
              <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary-50 text-sm font-semibold text-primary-700">
                {{ userName.slice(0, 1).toUpperCase() }}
              </div>
              <div class="min-w-0">
                <div class="truncate text-sm font-semibold text-ink">{{ userName }}</div>
                <div class="truncate text-xs text-slate-400">{{ copy.roleName }}</div>
              </div>
            </div>
            <RouterLink
              class="flex items-center gap-2.5 border-t border-hairline px-4 py-2.5 text-[13px] font-medium text-slate-600 transition-colors hover:bg-slate-50 hover:text-slate-900"
              to="/settings"
              @click="userMenuOpen = false; mobileNavOpen = false"
            >
              <Settings class="h-4 w-4 text-slate-400" />
              {{ copy.accountSettings }}
            </RouterLink>
            <button
              class="flex w-full items-center gap-2.5 border-t border-hairline px-4 py-2.5 text-left text-[13px] font-medium text-slate-600 transition-colors hover:bg-slate-50 hover:text-slate-900"
              type="button"
              @click="logout"
            >
              <LogOut class="h-4 w-4 text-slate-400" />
              {{ copy.logout }}
            </button>
          </div>
        </Transition>

        <button
          class="flex w-full items-center gap-2.5 rounded-[10px] px-2.5 py-2 text-left transition-colors hover:bg-slate-100/70"
          :class="showSidebarText && userMenuOpen ? 'bg-slate-100/70' : ''"
          type="button"
          @click="userMenuOpen = !userMenuOpen"
        >
          <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary-50 text-sm font-semibold text-primary-700">
            {{ userName.slice(0, 1).toUpperCase() }}
          </div>
          <div v-if="showSidebarText" class="min-w-0">
            <div class="truncate text-[13px] font-semibold text-ink">{{ userName }}</div>
            <div class="truncate text-xs text-slate-400">{{ copy.roleName }}</div>
          </div>
          <ChevronDown
            v-if="showSidebarText"
            class="ml-auto h-4 w-4 shrink-0 text-slate-400 transition-transform duration-200"
            :class="userMenuOpen ? 'rotate-180' : ''"
          />
        </button>
      </div>
    </aside>

    <main
      class="min-h-screen pt-14 transition-all duration-200 md:pt-0"
      :class="collapsed ? 'md:pl-[76px]' : 'md:pl-[264px]'"
    >
      <RouterView v-slot="{ Component }">
        <Transition name="page" mode="out-in">
          <component :is="Component" />
        </Transition>
      </RouterView>
    </main>
  </div>
</template>
