<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { RouterLink } from "vue-router"
import { Archive, ChevronRight, PackageOpen, RefreshCw, Search } from "lucide-vue-next"
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
    <section class="overflow-hidden rounded-[28px] bg-gradient-to-br from-[#09302f] via-[#0f4a52] to-[#d87b4a] p-6 text-white shadow-[0_24px_60px_rgba(15,74,82,0.22)]">
      <div class="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <div class="mb-4 inline-flex items-center gap-2 rounded-full bg-white/14 px-3 py-1 text-xs font-semibold">
            <Archive class="h-3.5 w-3.5" />
            {{ filteredPacks.length }} {{ copy.count }}
          </div>
          <h1 class="text-4xl font-black tracking-tight">{{ copy.title }}</h1>
          <p class="mt-3 max-w-2xl text-sm leading-6 text-white/78">{{ copy.subtitle }}</p>
        </div>
        <button class="inline-flex items-center justify-center gap-2 rounded-2xl bg-white px-4 py-2.5 text-sm font-bold text-[#0f4a52] shadow-lg shadow-black/10" @click="loadPacks()">
          <RefreshCw :class="['h-4 w-4', loading ? 'animate-spin' : '']" />
          {{ copy.refresh }}
        </button>
      </div>
    </section>

    <section class="mt-5 rounded-[24px] bg-white p-4 shadow-[0_12px_30px_rgba(15,74,82,0.06)]">
      <div class="relative">
        <Search class="absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
        <input v-model="search" class="input h-12 rounded-2xl pl-11" :placeholder="copy.search" />
      </div>
    </section>

    <section v-if="filteredPacks.length" class="mt-5 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      <RouterLink
        v-for="pack in filteredPacks"
        :key="pack.pack_id"
        class="group rounded-[24px] border border-slate-100 bg-white p-5 shadow-[0_12px_32px_rgba(15,23,42,0.06)] transition hover:-translate-y-1 hover:shadow-[0_18px_44px_rgba(15,74,82,0.12)]"
        :to="`/resource-packs/detail?id=${encodeURIComponent(pack.pack_id || '')}`"
      >
        <div class="mb-5 flex h-12 w-12 items-center justify-center rounded-2xl bg-[#e7f6f1] text-[#0f766e]">
          <PackageOpen class="h-6 w-6" />
        </div>
        <h2 class="line-clamp-2 text-lg font-black text-slate-950">{{ pack.title || pack.pack_id }}</h2>
        <p class="mt-2 line-clamp-3 min-h-[4.5rem] text-sm leading-6 text-slate-500">{{ pack.description || copy.emptyDesc }}</p>
        <div class="mt-5 space-y-2 text-xs text-slate-500">
          <p v-if="pack.respath">{{ copy.path }}: <span class="font-semibold text-slate-700">{{ pack.respath }}</span></p>
          <p v-if="pack.updated_at">{{ copy.updated }}: {{ pack.updated_at }}</p>
        </div>
        <div class="mt-5 inline-flex items-center gap-1 text-sm font-bold text-[#0f766e]">
          {{ copy.open }}
          <ChevronRight class="h-4 w-4 transition group-hover:translate-x-1" />
        </div>
      </RouterLink>
    </section>

    <section v-else class="mt-5 rounded-[24px] border border-dashed border-slate-200 bg-white p-10 text-center">
      <PackageOpen class="mx-auto h-12 w-12 text-slate-300" />
      <h2 class="mt-4 text-lg font-black text-slate-950">{{ loading ? "Loading..." : copy.emptyTitle }}</h2>
      <p class="mt-2 text-sm text-slate-500">{{ copy.emptyDesc }}</p>
    </section>

    <div v-if="nextPageToken" class="mt-5 text-center">
      <button class="btn btn-outline rounded-2xl" :disabled="loading" @click="loadPacks(nextPageToken)">
        {{ loading ? "Loading..." : "Load more" }}
      </button>
    </div>
  </AppShell>
</template>
