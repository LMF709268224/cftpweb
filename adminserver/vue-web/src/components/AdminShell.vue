<script setup lang="ts">
import { computed, ref } from "vue"
import { RouterLink, useRoute } from "vue-router"
import { Archive, Database, Home, LogOut, Menu, Settings, ShieldCheck, X } from "lucide-vue-next"
import { apiClient } from "@/lib/apiClient"

const route = useRoute()
const menuOpen = ref(false)

const navItems = [
  { href: "/", label: "总览", icon: Home },
  { href: "/resource-packs", label: "资源包配置", icon: Archive },
  { href: "/whitebox", label: "白盒查看台", icon: Database },
  { href: "/settings", label: "账号设置", icon: Settings },
]

const adminName = computed(() => localStorage.getItem("admin_name") || "Admin")

async function logout() {
  try {
    await apiClient("/api/auth/logout", { method: "POST" })
  } catch {
    // Logout should still clear local state if the server session is already gone.
  } finally {
    localStorage.removeItem("access_token")
    localStorage.removeItem("admin_name")
    window.location.href = "/login"
  }
}
</script>

<template>
  <div class="min-h-screen bg-[var(--page-bg)] text-slate-950">
    <div class="fixed inset-x-0 top-0 z-40 border-b border-white/35 bg-[rgba(245,240,230,0.82)] backdrop-blur-xl">
      <div class="mx-auto flex h-16 max-w-[1440px] items-center justify-between px-4">
        <RouterLink to="/" class="flex items-center gap-3">
          <span class="brand-mark"><ShieldCheck class="h-5 w-5" /></span>
          <span>
            <span class="block text-sm font-black tracking-[0.28em] text-[var(--ink)]">CFTP</span>
            <span class="block text-[11px] font-semibold uppercase tracking-[0.18em] text-slate-500">Admin Console</span>
          </span>
        </RouterLink>

        <div class="hidden items-center gap-3 md:flex">
          <span class="rounded-full bg-white/70 px-3 py-1 text-xs font-bold text-slate-600">{{ adminName }}</span>
          <button class="btn btn-ghost" @click="logout"><LogOut class="h-4 w-4" />退出</button>
        </div>
        <button class="btn btn-ghost md:hidden" @click="menuOpen = true"><Menu class="h-5 w-5" /></button>
      </div>
    </div>

    <aside class="fixed bottom-5 left-5 top-24 z-30 hidden w-64 rounded-[32px] border border-white/55 bg-white/72 p-4 shadow-[0_24px_70px_rgba(62,54,38,0.12)] backdrop-blur-xl lg:block">
      <div class="mb-5 rounded-[24px] bg-[var(--ink)] p-5 text-white">
        <p class="text-xs font-bold uppercase tracking-[0.22em] text-white/55">White Box</p>
        <h2 class="mt-2 text-xl font-black">微服务透明化</h2>
        <p class="mt-2 text-xs leading-5 text-white/68">统一查看资源包、课程、订单、消息、证书、PDF 和流水日志。</p>
      </div>
      <nav class="space-y-2">
        <RouterLink
          v-for="item in navItems"
          :key="item.href"
          :to="item.href"
          :class="['nav-item', route.path === item.href ? 'nav-item-active' : '']"
        >
          <component :is="item.icon" class="h-4 w-4" />
          {{ item.label }}
        </RouterLink>
      </nav>
    </aside>

    <div v-if="menuOpen" class="fixed inset-0 z-50 bg-black/30 p-4 lg:hidden" @click.self="menuOpen = false">
      <div class="ml-auto h-full w-[280px] rounded-[28px] bg-white p-4 shadow-2xl">
        <div class="mb-4 flex items-center justify-between">
          <strong>菜单</strong>
          <button class="btn btn-ghost" @click="menuOpen = false"><X class="h-4 w-4" /></button>
        </div>
        <nav class="space-y-2">
          <RouterLink
            v-for="item in navItems"
            :key="item.href"
            :to="item.href"
            :class="['nav-item', route.path === item.href ? 'nav-item-active' : '']"
            @click="menuOpen = false"
          >
            <component :is="item.icon" class="h-4 w-4" />
            {{ item.label }}
          </RouterLink>
        </nav>
        <button class="btn btn-primary mt-6 w-full" @click="logout"><LogOut class="h-4 w-4" />退出</button>
      </div>
    </div>

    <main class="px-4 pb-8 pt-24 lg:ml-72 lg:pr-8">
      <slot />
    </main>
  </div>
</template>
