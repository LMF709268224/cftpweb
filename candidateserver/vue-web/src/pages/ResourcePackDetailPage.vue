<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { RouterLink, useRoute, useRouter } from "vue-router"
import { toast } from "vue-sonner"
import { ArrowLeft, CalendarDays, Eye, FileArchive, FileText, Play, RefreshCw, Search } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"

type ResourcePackFile = {
  file_id?: string
  pack_id?: string
  title?: string
  description?: string
  file_type?: number | string
  file_name?: string
  file_size?: number
  sort_order?: number
  updated_at?: string
  created_at?: string
  thumbnail_object_key?: string
  thumbnail_url?: string
}

const route = useRoute()
const router = useRouter()
const { lang } = useTranslation()
const loading = ref(false)
const openingFileId = ref("")
const search = ref("")
const files = ref<ResourcePackFile[]>([])
const thumbnailUrls = ref<Record<string, string>>({})
const thumbnailLoading = ref<Record<string, boolean>>({})
const nextPageToken = ref("")

const packId = computed(() => String(route.params.packId || route.query.id || ""))
const copy = computed(() => lang.value === "zh"
  ? {
      title: "资源包详情",
      subtitle: "浏览你有权限访问的报告、视频和资料，点击卡片即可在线预览。",
      back: "返回资源包",
      search: "搜索资源标题或说明",
      emptyTitle: "这个资源包暂无文件",
      emptyDesc: "管理员添加文件并发布后，会显示在这里。",
      noSearchTitle: "没有匹配的资源",
      noSearchDesc: "换个关键词试试，或清空搜索查看全部资源。",
      clearSearch: "清空搜索",
      preview: "预览",
      size: "文件大小",
      updated: "更新于",
      refresh: "刷新",
      missing: "缺少资源包 ID",
      loading: "加载中...",
      loadingPreview: "正在打开预览...",
      loadMore: "加载更多",
      noViewUrl: "暂时无法打开预览，请稍后再试。",
    }
  : {
      title: "Resource Pack Detail",
      subtitle: "Browse reports, videos, and materials you are allowed to access. Click a card to preview online.",
      back: "Back to packs",
      search: "Search resources",
      emptyTitle: "No files in this pack",
      emptyDesc: "Files will appear here after admins add and publish them.",
      noSearchTitle: "No matching resources",
      noSearchDesc: "Try another keyword or clear the search to view all resources.",
      clearSearch: "Clear search",
      preview: "Preview",
      size: "File size",
      updated: "Updated",
      refresh: "Refresh",
      missing: "Missing resource pack ID",
      loading: "Loading...",
      loadingPreview: "Opening preview...",
      loadMore: "Load more",
      noViewUrl: "Unable to open the preview right now. Please try again later.",
    })

const orderedFiles = computed(() =>
  files.value.slice().sort((a, b) => Number(a.sort_order || 0) - Number(b.sort_order || 0)),
)

const filteredFiles = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  if (!keyword) return orderedFiles.value
  return orderedFiles.value.filter((file) =>
    `${file.title || ""} ${file.description || ""} ${file.file_name || ""}`.toLowerCase().includes(keyword),
  )
})

function normalizedType(type?: number | string) {
  return Number(type)
}

function fileTypeLabel(type?: number | string) {
  const normalized = normalizedType(type)
  if (normalized === 1) return "Video"
  if (normalized === 2) return "PDF"
  if (normalized === 3) return "ZIP"
  return "File"
}

function fileTypeIcon(type?: number | string) {
  const normalized = normalizedType(type)
  if (normalized === 1) return Play
  if (normalized === 2) return FileText
  return FileArchive
}

function fileTypePillClass(type?: number | string) {
  const normalized = normalizedType(type)
  if (normalized === 1) return "border-rose-200 bg-rose-50 text-rose-700"
  if (normalized === 2) return "border-blue-200 bg-blue-50 text-blue-700"
  if (normalized === 3) return "border-amber-200 bg-amber-50 text-amber-700"
  return "border-slate-200 bg-slate-50 text-slate-700"
}

function fallbackCoverClass(type?: number | string) {
  const normalized = normalizedType(type)
  if (normalized === 1) return "from-rose-500 via-orange-400 to-slate-900"
  if (normalized === 2) return "from-[#0b3478] via-[#0c5aa5] to-[#071b42]"
  if (normalized === 3) return "from-amber-500 via-orange-500 to-slate-900"
  return "from-slate-600 via-slate-800 to-slate-950"
}

function formatSize(size?: number) {
  if (!size) return "-"
  if (size < 1024 * 1024) return `${Math.round(size / 1024)} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

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

function thumbnailFor(file: ResourcePackFile) {
  if (!file.file_id) return file.thumbnail_url || ""
  return thumbnailUrls.value[file.file_id] || file.thumbnail_url || ""
}

async function loadThumbnail(file: ResourcePackFile) {
  if (!file.file_id || thumbnailUrls.value[file.file_id] || thumbnailLoading.value[file.file_id]) return
  if (file.thumbnail_url) {
    thumbnailUrls.value[file.file_id] = file.thumbnail_url
    return
  }
  if (!file.thumbnail_object_key) return

  thumbnailLoading.value[file.file_id] = true
  try {
    const resp = await apiClient(`/api/resource-pack-files/${encodeURIComponent(file.file_id)}/thumbnail-url`)
    if (resp?.url) thumbnailUrls.value[file.file_id] = resp.url
  } catch (err) {
    console.warn("Failed to load resource thumbnail", err)
  } finally {
    thumbnailLoading.value[file.file_id] = false
  }
}

async function loadFiles(pageToken = "") {
  if (!packId.value) return
  loading.value = true
  try {
    const params = new URLSearchParams({ page_size: "100" })
    if (pageToken) params.set("page_token", pageToken)
    const resp = await apiClient(`/api/resource-packs/${encodeURIComponent(packId.value)}/files?${params.toString()}`)
    const list = Array.isArray(resp?.files) ? resp.files : []
    files.value = pageToken ? files.value.concat(list) : list
    nextPageToken.value = resp?.next_page_token || ""
    void Promise.all(list.map((file: ResourcePackFile) => loadThumbnail(file)))
  } finally {
    loading.value = false
  }
}

async function openFile(file: ResourcePackFile) {
  if (!file.file_id) return
  openingFileId.value = file.file_id
  try {
    if (normalizedType(file.file_type) === 2) {
      sessionStorage.setItem(`resource-pack-file-preview-title:${file.file_id}`, file.title || file.file_name || file.file_id)
      const target = router.resolve(`/resource-pack-files/${encodeURIComponent(file.file_id)}/preview`)
      window.open(target.href, "_blank", "noopener,noreferrer")
      return
    }
    if (normalizedType(file.file_type) === 1) {
      sessionStorage.setItem(`resource-pack-file-preview-title:${file.file_id}`, file.title || file.file_name || file.file_id)
      const target = router.resolve(`/video-preview/resource-pack-files/${encodeURIComponent(file.file_id)}`)
      window.open(target.href, "_blank", "noopener,noreferrer")
      return
    }

    const resp = await apiClient(`/api/resource-pack-files/${encodeURIComponent(file.file_id)}/view-url`)
    const viewUrl = String(resp?.view_url || resp?.url || "").trim()
    if (!viewUrl) {
      toast.error(copy.value.noViewUrl)
      return
    }
    window.open(viewUrl, "_blank", "noopener,noreferrer")
  } catch {
    // apiClient already shows localized errors.
  } finally {
    window.setTimeout(() => {
      if (openingFileId.value === file.file_id) openingFileId.value = ""
    }, 800)
  }
}

watch(packId, () => {
  files.value = []
  thumbnailUrls.value = {}
  void loadFiles()
})

onMounted(() => {
  void loadFiles()
})
</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <FileArchive class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ copy.title }}</span>
      </header>

      <main class="px-5 py-8 md:px-8 lg:px-10">
        <section class="mb-6">
          <RouterLink to="/resource-packs" class="mb-6 inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-primary">
            <ArrowLeft class="h-4 w-4" />
            {{ copy.back }}
          </RouterLink>
          <div class="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
            <div>
              <div class="mb-3 inline-flex items-center gap-2 rounded-full bg-primary/10 px-3 py-1 text-xs font-semibold text-primary">
                <FileArchive class="h-3.5 w-3.5" />
                {{ packId || copy.missing }}
              </div>
              <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ copy.title }}</h1>
              <p class="mt-2 max-w-2xl text-muted-foreground">{{ copy.subtitle }}</p>
            </div>
            <button class="btn btn-outline rounded-lg bg-white shadow-sm hover:border-primary/25 hover:bg-primary/10 hover:text-primary" :disabled="!packId || loading" @click="loadFiles()">
              <RefreshCw :class="['h-4 w-4', loading ? 'animate-spin' : '']" />
              {{ copy.refresh }}
            </button>
          </div>
        </section>

    <section class="mb-5 rounded-[16px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <div class="relative">
        <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <input v-model="search" class="input pl-10" :placeholder="copy.search" />
      </div>
    </section>

    <section v-if="filteredFiles.length" class="grid gap-5 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
      <article
        v-for="file in filteredFiles"
        :key="file.file_id"
        class="group overflow-hidden rounded-[18px] bg-white shadow-[0_16px_34px_rgba(15,74,82,0.08)] transition-all hover:-translate-y-1 hover:shadow-[0_24px_48px_rgba(15,74,82,0.14)]"
      >
        <button class="block w-full text-left" :disabled="openingFileId === file.file_id" @click="openFile(file)">
          <div class="relative aspect-[4/5] overflow-hidden bg-slate-900">
            <img
              v-if="thumbnailFor(file)"
              :src="thumbnailFor(file)"
              :alt="file.title || file.file_name || file.file_id"
              class="h-full w-full object-cover transition duration-500 group-hover:scale-105"
              loading="lazy"
            />
            <div v-else :class="['flex h-full w-full flex-col justify-between bg-gradient-to-br p-5 text-white', fallbackCoverClass(file.file_type)]">
              <div class="inline-flex h-12 w-12 items-center justify-center rounded-2xl bg-white/15 backdrop-blur">
                <component :is="fileTypeIcon(file.file_type)" class="h-6 w-6" />
              </div>
              <div>
                <div class="mb-3 text-xs font-semibold uppercase tracking-[0.2em] text-white/70">Global Fintech Institute</div>
                <h2 class="line-clamp-5 text-2xl font-black leading-tight">{{ file.title || file.file_name || file.file_id }}</h2>
              </div>
              <div class="text-xs font-semibold text-white/70">{{ fileTypeLabel(file.file_type) }}</div>
            </div>
            <div class="absolute inset-x-0 bottom-0 bg-gradient-to-t from-slate-950/90 via-slate-950/40 to-transparent p-4 text-white">
              <div class="mb-2 flex items-center justify-between gap-3">
                <span :class="['rounded-full border px-2.5 py-1 text-xs font-bold backdrop-blur', fileTypePillClass(file.file_type)]">{{ fileTypeLabel(file.file_type) }}</span>
                <span class="inline-flex h-9 w-9 items-center justify-center rounded-full bg-white/90 text-slate-900 shadow-lg transition-transform group-hover:scale-110">
                  <Eye class="h-4 w-4" />
                </span>
              </div>
              <h2 class="line-clamp-2 text-base font-bold leading-tight">{{ file.title || file.file_name || file.file_id }}</h2>
              <p v-if="file.description || file.file_name" class="mt-1 line-clamp-2 text-xs leading-5 text-white/80">{{ file.description || file.file_name }}</p>
            </div>
          </div>
        </button>

        <div class="space-y-3 p-4">
          <div class="flex flex-wrap gap-2 text-xs text-muted-foreground">
            <span>{{ copy.size }}: {{ formatSize(file.file_size) }}</span>
            <span v-if="file.updated_at" class="inline-flex items-center gap-1">
              <CalendarDays class="h-3.5 w-3.5" />
              {{ formatDate(file.updated_at) }}
            </span>
          </div>
          <button
            class="btn btn-primary w-full rounded-xl shadow-sm shadow-primary/20"
            :disabled="openingFileId === file.file_id"
            @click="openFile(file)"
          >
            <RefreshCw v-if="openingFileId === file.file_id" class="h-4 w-4 animate-spin" />
            <Eye v-else class="h-4 w-4" />
            {{ openingFileId === file.file_id ? copy.loadingPreview : copy.preview }}
          </button>
        </div>
      </article>
    </section>

    <section v-else class="rounded-[16px] bg-white px-4 py-14 text-center shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10">
        <RefreshCw v-if="loading" class="h-8 w-8 animate-spin text-primary" />
        <FileArchive v-else class="h-8 w-8 text-primary" />
      </div>
      <h2 class="mt-4 text-lg font-semibold text-foreground">{{ loading ? copy.loading : search.trim() ? copy.noSearchTitle : copy.emptyTitle }}</h2>
      <p class="mx-auto mt-2 max-w-md text-sm leading-6 text-muted-foreground">{{ search.trim() ? copy.noSearchDesc : copy.emptyDesc }}</p>
      <button v-if="search.trim()" class="btn btn-primary mt-5 rounded-lg shadow-sm shadow-primary/20" @click="search = ''">
        {{ copy.clearSearch }}
      </button>
    </section>

    <div v-if="nextPageToken" class="mt-4 text-center">
      <button class="btn btn-outline rounded-lg" :disabled="loading" @click="loadFiles(nextPageToken)">
        {{ loading ? copy.loading : copy.loadMore }}
      </button>
    </div>
      </main>
    </div>
  </AppShell>
</template>
