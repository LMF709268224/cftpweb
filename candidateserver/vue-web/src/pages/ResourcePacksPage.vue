<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { RouterLink } from "vue-router"
import { PackageOpen, RefreshCw, Search } from "lucide-vue-next"
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
      <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ copy.title }}</h1>
      <p class="mt-2 max-w-2xl text-muted-foreground">{{ copy.subtitle }}</p>
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
        class="group relative overflow-hidden rounded-[16px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all hover:-translate-y-0.5 hover:shadow-md hover:shadow-primary/10"
        :to="`/resource-packs/detail?id=${encodeURIComponent(pack.pack_id || '')}`"
      >
        <div class="absolute left-0 top-0 h-full w-1 bg-primary/45" />
        <div class="mb-4 flex h-11 w-11 items-center justify-center rounded-lg bg-primary/10 text-primary transition-transform group-hover:scale-105">
          <PackageOpen class="h-6 w-6" />
        </div>
        <h2 class="line-clamp-2 text-lg font-semibold text-card-foreground transition-colors group-hover:text-primary">{{ pack.title || pack.pack_id }}</h2>
        <p class="mt-2 line-clamp-3 min-h-[4.5rem] text-sm leading-6 text-muted-foreground">{{ pack.description || copy.emptyDesc }}</p>
        <div class="mt-4 space-y-2 text-xs text-muted-foreground">
          <p v-if="pack.respath">{{ copy.path }}: <span class="font-medium text-card-foreground">{{ pack.respath }}</span></p>
          <p v-if="pack.updated_at">{{ copy.updated }}: {{ pack.updated_at }}</p>
        </div>
        <div class="mt-4 flex h-9 w-full items-center justify-center rounded-lg bg-primary px-4 text-sm font-semibold text-white shadow-sm shadow-primary/20 transition-colors group-hover:bg-primary/90">
          {{ copy.open }}
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
      <h2 class="mt-4 text-lg font-semibold text-foreground">{{ copy.emptyTitle }}</h2>
      <p class="mt-2 text-sm text-muted-foreground">{{ copy.emptyDesc }}</p>
    </section>

    <div v-if="nextPageToken" class="mt-4 text-center">
      <button class="btn btn-outline rounded-lg" :disabled="loading" @click="loadPacks(nextPageToken)">
        {{ loading ? copy.loading : copy.loadMore }}
      </button>
    </div>
  </AppShell>
</template>
