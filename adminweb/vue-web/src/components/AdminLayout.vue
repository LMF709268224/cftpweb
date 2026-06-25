<script setup lang="ts">
import {
  BookOpen,
  Boxes,
  ChevronLeft,
  ClipboardCheck,
  CreditCard,
  FileBadge,
  FileText,
  GitBranch,
  GraduationCap,
  Mail,
  MessageSquare,
  Receipt,
  ShieldCheck,
  Webhook,
} from "lucide-vue-next"
import { computed, ref } from "vue"
import { RouterLink, RouterView, useRoute, useRouter } from "vue-router"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { clearAuthSession, getUserName } from "@/lib/authStorage"

const router = useRouter()
const route = useRoute()
const collapsed = ref(false)

const navItems = [
  { path: "/lms", label: "课程配置", icon: BookOpen },
  { path: "/pipelines", label: "管线配置", icon: FileBadge },
  { path: "/bundles", label: "商品配置", icon: Boxes },
  { path: "/prog", label: "管线管理", icon: GitBranch },
  { path: "/messages", label: "站内信", icon: MessageSquare },
  { path: "/mails", label: "邮件中心", icon: Mail },
  { path: "/orders", label: "订单管理", icon: CreditCard },
  { path: "/invoices", label: "发票管理", icon: Receipt },
  { path: "/credentials", label: "资格定义", icon: ShieldCheck },
  { path: "/applications", label: "审核中心", icon: ClipboardCheck },
  { path: "/pdf-templates", label: "PDF 模板配置", icon: FileText },
  { path: "/pdf-requests", label: "证书生成流水", icon: FileBadge },
  { path: "/audit/webhooks", label: "Webhook 审计", icon: Webhook },
  { path: "/permissions", label: "考生权限管理", icon: GraduationCap },
]

const userName = computed(() => getUserName())

async function logout() {
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

      <nav class="flex-1 space-y-1 overflow-y-auto px-3 py-5">
        <RouterLink
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="flex items-center gap-3 rounded-xl px-3 py-3 text-[15px] font-semibold text-slate-700 transition hover:bg-slate-100"
          :class="route.path === item.path || route.path.startsWith(`${item.path}/`) ? 'bg-[#0b7bdc] text-white shadow-lg shadow-sky-200 hover:bg-[#0b7bdc]' : ''"
        >
          <component :is="item.icon" class="h-5 w-5 shrink-0" />
          <span v-if="!collapsed">{{ item.label }}</span>
        </RouterLink>
      </nav>

      <div class="border-t border-slate-200 p-4">
        <div class="mb-3 flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-full bg-sky-100 font-bold text-sky-700">
            {{ userName.slice(0, 1).toUpperCase() }}
          </div>
          <div v-if="!collapsed" class="min-w-0">
            <div class="truncate text-sm font-bold">{{ userName }}</div>
            <div class="text-xs text-slate-500">Admin</div>
          </div>
        </div>
        <button
          v-if="!collapsed"
          class="w-full rounded-xl border border-slate-200 px-3 py-2 text-sm font-semibold text-slate-600 hover:bg-slate-50"
          type="button"
          @click="logout"
        >
          退出登录
        </button>
      </div>
    </aside>

    <main class="min-h-screen transition-all duration-200" :class="collapsed ? 'pl-[76px]' : 'pl-[256px]'">
      <RouterView />
    </main>
  </div>
</template>
