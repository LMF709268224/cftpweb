<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { RouterLink } from "vue-router"
import { ArrowRight, CalendarClock, PackageOpen, RefreshCw, Search } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { formatBackendDateOnly } from "@/lib/utils"
import { useTranslation } from "@/lib/language"

type ResourcePack = {
  pack_id?: string
  title?: string
  description?: string
  respath?: string
  status?: string
  updated_at?: string
}

const { t } = useTranslation()
const loading = ref(false)
const search = ref("")
const packs = ref<ResourcePack[]>([])
const nextPageToken = ref("")

const copy = computed(() => t.value.resourcePacksPage)

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

function rememberPack(pack: ResourcePack) {
  if (!pack.pack_id) return
  sessionStorage.setItem(`resource-pack-title:${pack.pack_id}`, pack.title || "")
  sessionStorage.setItem(`resource-pack-respath:${pack.pack_id}`, pack.respath || "")
}

onMounted(() => {
  void loadPacks()
})
</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <PackageOpen class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ copy.title }}</span>
      </header>

      <main class="px-5 py-8 md:px-8 lg:px-10">
        <div class="mb-6">
          <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ copy.title }}</h1>
          <p class="mt-2 max-w-2xl text-muted-foreground">{{ copy.subtitle }}</p>
        </div>

    <section class="mb-4 flex flex-col gap-4 rounded-[16px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)] sm:flex-row sm:items-center sm:justify-between">
      <div class="relative flex-1 sm:max-w-md">
        <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <input v-model="search" class="input pl-10" :placeholder="copy.search" />
      </div>
      <button class="resource-refresh-btn inline-flex h-9 items-center gap-2 rounded-xl border px-4 text-sm font-semibold" @click="loadPacks()">
        <RefreshCw :class="['h-4 w-4', loading ? 'animate-spin' : '']" />
        {{ copy.refresh }}
      </button>
    </section>

    <section v-if="filteredPacks.length" class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      <RouterLink
        v-for="pack in filteredPacks"
        :key="pack.pack_id"
        class="resource-pack-card group relative overflow-hidden rounded-[16px] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all duration-300 ease-out hover:-translate-y-1 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30"
        :to="`/resource-packs/${encodeURIComponent(pack.pack_id || '')}`"
        @click="rememberPack(pack)"
      >
        <span class="resource-pack-sheen pointer-events-none absolute left-0 top-0 z-20 h-1 w-full" />
        <span class="resource-pack-orb pointer-events-none absolute -right-12 -top-12 z-10 h-36 w-36 rounded-full opacity-0 transition-opacity duration-300 group-hover:opacity-100" />
        <div class="relative flex h-24 items-end overflow-hidden px-4 pb-3 text-white bg-[linear-gradient(135deg,rgb(11,31,69)_0%,rgb(27,69,141)_55%,rgb(58,111,192)_100%)]">
          <div class="absolute inset-0 bg-[radial-gradient(circle_at_20%_15%,rgba(255,255,255,0.24),transparent_30%)] opacity-70" />
          <div class="absolute right-4 top-1/2 flex h-12 w-12 -translate-y-1/2 items-center justify-center rounded-xl bg-white/10 ring-1 ring-white/15 transition-transform duration-300 group-hover:scale-105">
            <PackageOpen class="h-7 w-7 text-white/70" />
          </div>
        </div>
        <div class="relative p-4">
          <h2 class="line-clamp-2 text-lg font-semibold text-card-foreground transition-colors group-hover:text-primary">{{ pack.title || pack.pack_id }}</h2>
          <p class="mt-2 line-clamp-3 min-h-[4.5rem] text-sm leading-6 text-muted-foreground">{{ pack.description || copy.emptyDesc }}</p>
          <div class="mt-4 space-y-2 text-xs text-muted-foreground">
            <p v-if="pack.updated_at" class="flex items-center gap-1.5">
              <CalendarClock class="h-3.5 w-3.5" />
              <span>{{ copy.updated }}: {{ formatBackendDateOnly(pack.updated_at) }}</span>
            </p>
          </div>
          <div class="mt-4 flex h-9 w-full items-center justify-center gap-2 rounded-lg bg-primary px-4 text-sm font-semibold text-white shadow-sm shadow-primary/20 transition-all duration-300 group-hover:bg-primary/90 group-hover:shadow-primary/30">
            {{ copy.open }}
            <ArrowRight class="h-4 w-4 transition-transform duration-300 group-hover:translate-x-0.5" />
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
      </main>
    </div>
  </AppShell>
</template>

<style scoped>
.resource-refresh-btn {
  border-color: #e2e8f0;
  background: #ffffff;
  color: #334155;
  box-shadow: 0 8px 18px -16px rgba(15, 23, 42, 0.35);
  transition: transform 0.2s ease, border-color 0.2s ease, background-color 0.2s ease, color 0.2s ease, box-shadow 0.2s ease;
}

.resource-refresh-btn:hover {
  border-color: rgba(37, 99, 235, 0.28);
  background: rgba(37, 99, 235, 0.08);
  color: #1d4ed8;
  box-shadow: 0 14px 28px -18px rgba(37, 99, 235, 0.42);
  transform: scale(1.02);
}

.resource-refresh-btn:active {
  transform: scale(0.98);
}

.resource-refresh-btn:focus-visible {
  outline: none;
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.16), 0 14px 28px -18px rgba(37, 99, 235, 0.42);
}

.resource-pack-card {
  --resource-pack-accent: #38bdf8;
  --resource-pack-glow: rgba(37, 99, 235, 0.2);
}

.resource-pack-card:hover {
  box-shadow: 0 18px 34px -18px var(--resource-pack-glow), 0 12px 28px rgba(15, 23, 42, 0.1);
}

.resource-pack-sheen {
  background: linear-gradient(90deg, transparent, var(--resource-pack-accent), transparent);
  opacity: 0.78;
  transform: translateX(-105%);
  transition: transform 0.65s ease;
}

.resource-pack-card:hover .resource-pack-sheen {
  transform: translateX(105%);
}

.resource-pack-orb {
  background: radial-gradient(circle, rgba(56, 189, 248, 0.2), transparent 68%);
}
</style>
