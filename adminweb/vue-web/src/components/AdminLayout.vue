<script setup lang="ts">
import {
  BookOpen,
  Boxes,
  BarChart3,
  ChevronDown,
  ChevronLeft,
  ClipboardCheck,
  ClipboardList,
  CreditCard,
  FileBadge,
  FileText,
  GitBranch,
  GraduationCap,
  LogOut,
  Mail,
  MessageSquare,
  Receipt,
  Settings,
  ShieldCheck,
  Webhook,
} from "lucide-vue-next"
import { onMounted, onUnmounted, ref } from "vue"
import { RouterLink, RouterView, useRoute, useRouter } from "vue-router"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { clearAuthSession, getUserName } from "@/lib/authStorage"

const router = useRouter()
const route = useRoute()
const collapsed = ref(false)
const userMenuOpen = ref(false)
const userMenuRef = ref<HTMLElement | null>(null)

const navGroups = [
  {
    label: "课程与资源",
    items: [
      { path: "/dashboard", label: "运营看板", icon: BarChart3 },
      { path: "/lms", label: "课程配置", icon: BookOpen },
      { path: "/resource-packs", label: "资源包配置", icon: FileText },
      { path: "/resource-pack-files", label: "资源文件配置", icon: FileText },
    ],
  },
  {
    label: "认证流程",
    items: [
      { path: "/pipelines", label: "管线配置", icon: FileBadge },
      { path: "/prog", label: "管线管理", icon: GitBranch },
      { path: "/exams", label: "考试管理", icon: ClipboardList },
      { path: "/credentials", label: "资格定义", icon: ShieldCheck },
      { path: "/applications", label: "审核中心", icon: ClipboardCheck },
      { path: "/permissions", label: "考生权限管理", icon: GraduationCap },
    ],
  },
  {
    label: "商品与财务",
    items: [
      { path: "/bundles", label: "商品配置", icon: Boxes },
      { path: "/orders", label: "订单管理", icon: CreditCard },
      { path: "/invoices", label: "发票管理", icon: Receipt },
    ],
  },
  {
    label: "消息通知",
    items: [
      { path: "/messages", label: "站内信", icon: MessageSquare },
      { path: "/mails", label: "邮件中心", icon: Mail },
    ],
  },
  {
    label: "系统运维",
    items: [
      { path: "/pdf-templates", label: "PDF 模板配置", icon: FileText },
      { path: "/pdf-requests", label: "证书生成流水", icon: FileBadge },
      { path: "/audit/webhooks", label: "Webhook 审计", icon: Webhook },
      { path: "/settings", label: "账户设置", icon: Settings },
    ],
  },
]

const userName = ref(getUserName())

function refreshUserName() {
  userName.value = getUserName()
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
    toast.success("已退出登录")
    router.push("/login")
  }
}

onMounted(() => {
  window.addEventListener("storage", refreshUserName)
  document.addEventListener("pointerdown", closeUserMenuOnOutsideClick)
})

onUnmounted(() => {
  window.removeEventListener("storage", refreshUserName)
  document.removeEventListener("pointerdown", closeUserMenuOnOutsideClick)
})
</script>

<template>
  <div class="min-h-screen bg-[#f4f8fc] text-slate-950">
    <aside
      class="fixed inset-y-0 left-0 z-30 flex flex-col border-r border-slate-200 bg-white/95 shadow-sm transition-all duration-200"
      :class="collapsed ? 'w-[76px]' : 'w-[256px]'"
    >
      <div class="flex h-20 items-center gap-3 border-b border-slate-200 px-4">
        <div class="flex h-11 w-11 items-center justify-center rounded-2xl bg-[#0b7bdc] text-white shadow-lg shadow-sky-200">
          <GraduationCap class="h-6 w-6" />
        </div>
        <div v-if="!collapsed" class="leading-tight">
          <div class="text-lg font-black">CFTP</div>
          <div class="text-sm text-slate-500">管理系统</div>
        </div>
      </div>

      <button
        class="absolute -right-4 top-24 flex h-8 w-8 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm"
        type="button"
        @click="collapsed = !collapsed"
      >
        <ChevronLeft class="h-4 w-4 transition-transform" :class="collapsed ? 'rotate-180' : ''" />
      </button>

      <nav class="flex-1 overflow-y-auto px-3 py-5">
        <div v-for="group in navGroups" :key="group.label" class="mb-5 last:mb-0">
          <div v-if="!collapsed" class="mb-2 px-3 text-[11px] font-black uppercase tracking-wider text-slate-400">
            {{ group.label }}
          </div>
          <div v-else class="mx-auto mb-3 h-px w-8 bg-slate-200 first:hidden" />
          <div class="space-y-1">
            <RouterLink
              v-for="item in group.items"
              :key="item.path"
              :to="item.path"
              class="flex items-center gap-3 rounded-xl px-3 py-3 text-[15px] font-semibold text-slate-700 transition hover:bg-slate-100"
              :class="route.path === item.path || route.path.startsWith(`${item.path}/`) ? '!bg-[#0b4ea2] !text-white shadow-lg shadow-sky-200 hover:!bg-[#0b4ea2]' : ''"
            >
              <component :is="item.icon" class="h-5 w-5 shrink-0" />
              <span v-if="!collapsed">{{ item.label }}</span>
            </RouterLink>
          </div>
        </div>
      </nav>

      <div ref="userMenuRef" class="border-t border-slate-200 p-4">
        <div v-if="!collapsed && userMenuOpen" class="mb-3 overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-lg shadow-slate-200/60">
          <div class="flex items-center gap-3 px-4 py-4">
            <div class="flex h-9 w-9 items-center justify-center rounded-full bg-sky-100 font-bold text-sky-700">
              {{ userName.slice(0, 1).toUpperCase() }}
            </div>
            <div class="min-w-0">
              <div class="truncate text-sm font-bold text-slate-950">{{ userName }}</div>
              <div class="truncate text-xs text-slate-500">Admin</div>
            </div>
          </div>
          <RouterLink
            class="flex items-center gap-3 border-t border-slate-100 px-4 py-3 text-sm font-semibold text-slate-700 hover:bg-slate-50"
            to="/settings"
            @click="userMenuOpen = false"
          >
            <Settings class="h-4 w-4 text-slate-500" />
            账户设置
          </RouterLink>
          <button
            class="flex w-full items-center gap-3 border-t border-slate-100 px-4 py-3 text-left text-sm font-semibold text-slate-700 hover:bg-slate-50"
            type="button"
            @click="logout"
          >
            <LogOut class="h-4 w-4 text-slate-500" />
            退出登录
          </button>
        </div>

        <button
          class="flex w-full items-center gap-3 rounded-2xl px-3 py-3 text-left transition hover:bg-slate-100"
          :class="!collapsed && userMenuOpen ? 'bg-blue-50 text-[#0b4ea2]' : ''"
          type="button"
          @click="userMenuOpen = !userMenuOpen"
        >
          <div class="flex h-10 w-10 items-center justify-center rounded-full bg-sky-100 font-bold text-sky-700">
            {{ userName.slice(0, 1).toUpperCase() }}
          </div>
          <div v-if="!collapsed" class="min-w-0">
            <div class="truncate text-sm font-bold">{{ userName }}</div>
            <div class="text-xs text-slate-500">Admin</div>
          </div>
          <ChevronDown
            v-if="!collapsed"
            class="ml-auto h-4 w-4 text-slate-500 transition-transform"
            :class="userMenuOpen ? 'rotate-180' : ''"
          />
        </button>
      </div>
    </aside>

    <main class="min-h-screen transition-all duration-200" :class="collapsed ? 'pl-[76px]' : 'pl-[256px]'">
      <RouterView />
    </main>
  </div>
</template>
