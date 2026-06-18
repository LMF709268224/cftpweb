<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { RouterLink } from "vue-router"
import { ArrowRight, CalendarClock, PackageOpen, RefreshCw, Search } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"

type ResourcePack = {
  pack_id?: string
  title?: string
  description?: string
  respath?: string
  status?: string
  updated_at?: string
}

const { lang } = useTranslation()
const loading = ref(false)
const search = ref("")
const packs = ref<ResourcePack[]>([])
const nextPageToken = ref("")

const copy = computed(() => lang.value === "zh"
  ? {
      title: "资源包",
      subtitle: "查看你当前有权限访问的补充资料包，进入详情后可以打开视频、PDF 或 ZIP 文件。",
      search: "搜索资源包标题或说明",
      refresh: "刷新",
      emptyTitle: "暂无可访问资源包",
      emptyDesc: "当后台为你的资格或课程开放资源包后，会显示在这里。",
      noSearchTitle: "没有匹配的资源包",
      noSearchDesc: "换个关键词再试，或清空搜索查看全部可访问资源包。",
      clearSearch: "清空搜索",
      open: "查看详情",
      path: "权限路径",
      updated: "更新于",
      count: "个资源包",
      loading: "加载中...",
      loadMore: "加载更多",
    }
  : {
      title: "Resource Packs",
      subtitle: "Browse supplemental packs you are allowed to access. Open a pack to view videos, PDFs, or ZIP files.",
      search: "Search resource packs",
      refresh: "Refresh",
      emptyTitle: "No resource packs yet",
      emptyDesc: "Accessible resource packs will appear here once enabled by admins.",
      noSearchTitle: "No matching resource packs",
      noSearchDesc: "Try another keyword or clear the search to view all accessible packs.",
      clearSearch: "Clear search",
      open: "Open",
      path: "Access path",
      updated: "Updated",
      count: "packs",
      loading: "Loading...",
      loadMore: "Load more",
    })

const filteredPacks = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  if (!keyword) return packs.value
  return packs.value.filter((pack) =>
    `${pack.title || ""} ${pack.description || ""} ${pack.respath || ""}`.toLowerCase().includes(keyword),
  )
})

function formatDate(value?: string) {
  if (!value) return ""
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleDateString(lang.value === "zh" ? "zh-CN" : "en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
  })
}

async function loadPacks(pageToken = "") {
  loading.value = true
  try {
    const params = new URLSearchParams({ page_size: "50" })
    if (pageToken) params.set("page_token", pageToken)
    const resp = await apiClient(`/api/resource-packs?${params.toString()}`)
    const list = Array.isArray(resp?.packs) ? resp.packs : []
    packs.value = pageToken ? packs.value.concat(list) : list
    nextPageToken.value = resp?.next_page_token || ""
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void loadPacks()
})
</script>

<template>
  <AppShell content-class="p-4">
    <div class="mb-4 px-1 py-3 md:py-5">
      <div class="flex items-start gap-3">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-accent text-primary">
          <PackageOpen class="h-6 w-6" />
        </div>
        <div>
          <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ copy.title }}</h1>
          <p class="mt-2 max-w-2xl text-muted-foreground">{{ copy.subtitle }}</p>
        </div>
      </div>
    </div>

    <section class="mb-4 flex flex-col gap-4 rounded-[16px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)] sm:flex-row sm:items-center sm:justify-between">
      <div class="relative flex-1 sm:max-w-md">
        <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <input v-model="search" class="input pl-10" :placeholder="copy.search" />
      </div>
      <button class="btn btn-outline rounded-lg bg-white/80 shadow-sm hover:border-primary/25 hover:bg-primary/10 hover:text-primary" @click="loadPacks()">
        <RefreshCw :class="['h-4 w-4', loading ? 'animate-spin' : '']" />
        {{ copy.refresh }}
      </button>
    </section>

    <section v-if="filteredPacks.length" class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      <RouterLink
        v-for="pack in filteredPacks"
        :key="pack.pack_id"
        class="group relative overflow-hidden rounded-[16px] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all hover:-translate-y-0.5 hover:shadow-md hover:shadow-primary/10"
        :to="`/resource-packs/${encodeURIComponent(pack.pack_id || '')}`"
      >
        <div class="relative flex h-24 items-end overflow-hidden px-4 pb-3 text-white bg-[linear-gradient(135deg,rgb(11,31,69)_0%,rgb(27,69,141)_55%,rgb(58,111,192)_100%)]">
          <span class="relative z-10 inline-flex items-center rounded-full bg-white/95 px-3 py-1 text-xs font-semibold text-primary shadow-sm">
            {{ lang === 'zh' ? '资料包' : 'Resource Pack' }}
          </span>
          <PackageOpen class="absolute right-5 top-1/2 h-9 w-9 -translate-y-1/2 text-white/35 transition-transform group-hover:scale-105" />
        </div>
        <div class="p-4">
          <h2 class="line-clamp-2 text-lg font-semibold text-card-foreground transition-colors group-hover:text-primary">{{ pack.title || pack.pack_id }}</h2>
          <p class="mt-2 line-clamp-3 min-h-[4.5rem] text-sm leading-6 text-muted-foreground">{{ pack.description || copy.emptyDesc }}</p>
          <div class="mt-4 space-y-2 text-xs text-muted-foreground">
            <p v-if="pack.respath">{{ copy.path }}: <span class="font-medium text-card-foreground">{{ pack.respath }}</span></p>
            <p v-if="pack.updated_at" class="flex items-center gap-1.5">
              <CalendarClock class="h-3.5 w-3.5" />
              <span>{{ copy.updated }}: {{ formatDate(pack.updated_at) }}</span>
            </p>
          </div>
          <div class="mt-4 flex h-9 w-full items-center justify-center gap-2 rounded-lg bg-primary px-4 text-sm font-semibold text-white shadow-sm shadow-primary/20 transition-colors group-hover:bg-primary/90">
            {{ copy.open }}
            <ArrowRight class="h-4 w-4" />
          </div>
        </div>
      </RouterLink>
    </section>

    <section v-else-if="loading" class="flex items-center justify-center gap-2 rounded-[16px] bg-white py-16 text-muted-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <RefreshCw class="h-5 w-5 animate-spin" />
      <span>{{ copy.loading }}</span>
    </section>

    <section v-else class="rounded-[16px] bg-white px-4 py-14 text-center shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10">
        <PackageOpen class="h-8 w-8 text-primary" />
      </div>
      <h2 class="mt-4 text-lg font-semibold text-foreground">{{ search.trim() ? copy.noSearchTitle : copy.emptyTitle }}</h2>
      <p class="mx-auto mt-2 max-w-md text-sm leading-6 text-muted-foreground">{{ search.trim() ? copy.noSearchDesc : copy.emptyDesc }}</p>
      <button v-if="search.trim()" class="btn btn-primary mt-5 rounded-lg shadow-sm shadow-primary/20" @click="search = ''">
        {{ copy.clearSearch }}
      </button>
    </section>

    <div v-if="nextPageToken" class="mt-4 text-center">
      <button class="btn btn-outline rounded-lg" :disabled="loading" @click="loadPacks(nextPageToken)">
        {{ loading ? copy.loading : copy.loadMore }}
      </button>
    </div>
  </AppShell>
</template>
