<script setup lang="ts">
import { Archive, ArrowRight, Database, Layers3, PackageCheck } from "lucide-vue-next"
import { RouterLink } from "vue-router"
import AdminShell from "@/components/AdminShell.vue"

const cards = [
  {
    title: "资源包配置",
    desc: "创建资源包、配置 respath、维护文件，发布后 candidate 才能按权限看到。",
    href: "/resource-packs",
    icon: Archive,
  },
  {
    title: "白盒查看台",
    desc: "聚合各微服务现有 list/detail 接口，查问题时不用先去数据库翻表。",
    href: "/whitebox",
    icon: Database,
  },
]
</script>

<template>
  <AdminShell>
    <section class="relative overflow-hidden rounded-[36px] bg-[var(--ink)] p-8 text-white shadow-[0_28px_80px_rgba(32,49,45,0.24)]">
      <div class="absolute right-[-60px] top-[-70px] h-64 w-64 rounded-full bg-[var(--clay)]/40 blur-3xl" />
      <div class="absolute bottom-[-80px] left-[30%] h-48 w-48 rounded-full bg-[var(--moss)]/40 blur-3xl" />
      <div class="relative">
        <div class="mb-5 inline-flex rounded-full bg-white/12 px-4 py-2 text-xs font-black uppercase tracking-[0.22em] text-white/70">
          README 前三项
        </div>
        <h1 class="max-w-4xl text-4xl font-black leading-tight tracking-tight md:text-6xl">Admin Server Vue 化与微服务白盒化入口</h1>
        <p class="mt-5 max-w-3xl text-base leading-8 text-white/70">
          当前版本已经把 admin 静态目录切到 `vue-web/dist`，并新增资源包管理、candidate 资源包访问和统一白盒查看台。
        </p>
      </div>
    </section>

    <section class="mt-6 grid gap-4 md:grid-cols-3">
      <div class="card p-6">
        <PackageCheck class="h-8 w-8 text-[var(--clay)]" />
        <p class="mt-5 text-3xl font-black">01</p>
        <p class="mt-2 text-sm font-bold text-slate-600">资源包 admin 配置与 candidate 查看</p>
      </div>
      <div class="card p-6">
        <Layers3 class="h-8 w-8 text-[var(--moss)]" />
        <p class="mt-5 text-3xl font-black">02</p>
        <p class="mt-2 text-sm font-bold text-slate-600">admin 前端从 web 切到 vue-web</p>
      </div>
      <div class="card p-6">
        <Database class="h-8 w-8 text-[var(--ink)]" />
        <p class="mt-5 text-3xl font-black">03</p>
        <p class="mt-2 text-sm font-bold text-slate-600">微服务列表查询白盒化</p>
      </div>
    </section>

    <section class="mt-6 grid gap-4 lg:grid-cols-2">
      <RouterLink
        v-for="card in cards"
        :key="card.href"
        :to="card.href"
        class="card group block p-6 text-inherit no-underline"
      >
        <component :is="card.icon" class="h-9 w-9 text-[var(--clay)]" />
        <h2 class="mt-6 text-2xl font-black">{{ card.title }}</h2>
        <p class="mt-3 min-h-12 text-sm leading-6 text-slate-500">{{ card.desc }}</p>
        <span class="mt-6 inline-flex items-center gap-2 text-sm font-black text-[var(--ink)]">
          进入
          <ArrowRight class="h-4 w-4 transition group-hover:translate-x-1" />
        </span>
      </RouterLink>
    </section>
  </AdminShell>
</template>
