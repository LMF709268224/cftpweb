<script setup lang="ts">
import { LayoutDashboard, SearchX } from "lucide-vue-next"
import { computed } from "vue"
import { RouterLink, useRoute } from "vue-router"
import { useAdminLanguage } from "@/lib/language"

const route = useRoute()
const { t } = useAdminLanguage()
const copy = computed(() => t.value.notFound)
const requestedPath = computed(() => route.path)
</script>

<template>
  <section class="mx-auto flex min-h-[calc(100vh-3.5rem)] w-full max-w-[1200px] items-center px-5 py-10 md:min-h-screen md:px-10 md:py-16">
    <div class="grid w-full gap-8 border-y border-slate-200 py-10 md:grid-cols-[180px_1fr] md:items-center md:gap-12 md:py-16">
      <div class="flex h-36 w-36 items-center justify-center rounded-3xl bg-sky-100 text-[#0b4ea2] md:h-44 md:w-44" aria-hidden="true">
        <SearchX class="h-16 w-16 md:h-20 md:w-20" />
      </div>

      <div class="min-w-0">
        <p class="text-sm font-black text-[#0b4ea2]">404</p>
        <h1 class="mt-2 text-3xl font-black text-slate-950 md:text-4xl">{{ copy.title }}</h1>
        <p class="mt-4 max-w-2xl text-base leading-7 text-slate-600">{{ copy.description }}</p>
        <div class="mt-5 max-w-2xl border-l-4 border-sky-200 bg-white px-4 py-3 text-sm text-slate-500">
          <span class="font-bold text-slate-700">{{ copy.pathLabel }}</span>
          <code class="ml-2 break-all">{{ requestedPath }}</code>
        </div>
        <RouterLink
          class="mt-7 inline-flex h-11 items-center justify-center gap-2 rounded-xl bg-[#0b4ea2] px-5 text-sm font-bold text-white shadow-sm transition hover:bg-[#083f84]"
          to="/dashboard"
        >
          <LayoutDashboard class="h-4 w-4" />
          {{ copy.backToDashboard }}
        </RouterLink>
      </div>
    </div>
  </section>
</template>
