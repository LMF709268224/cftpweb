<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { RouterLink, useRoute, useRouter } from "vue-router"
import { toast } from "vue-sonner"
import { ArrowLeft, CalendarDays, Clock, Eye, FileArchive, Play, RefreshCw, Search } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { formatBackendDateOnly } from "@/lib/utils"
import { useTranslation } from "@/lib/language"

type ResourcePackFile = {
  file_id?: string
  pack_id?: string
  title?: string
  description?: string
  file_type?: number | string
  file_name?: string
  file_size?: number
  duration?: string | number
  duration_seconds?: string | number
  duration_min?: string | number
  sort_order?: number
  updated_at?: string
  created_at?: string
  thumbnail_object_key?: string
  thumbnail_url?: string
}

const route = useRoute()
const router = useRouter()
const { t } = useTranslation()
const loading = ref(false)
const openingFileId = ref("")
const search = ref("")
const files = ref<ResourcePackFile[]>([])
const thumbnailUrls = ref<Record<string, string>>({})
const thumbnailLoading = ref<Record<string, boolean>>({})
const nextPageToken = ref("")

const packId = computed(() => String(route.params.packId || route.query.id || ""))
const storedPackTitle = computed(() => (packId.value ? sessionStorage.getItem(`resource-pack-title:${packId.value}`) || "" : ""))
const storedPackRespath = computed(() => (packId.value ? sessionStorage.getItem(`resource-pack-respath:${packId.value}`) || "" : ""))
const isInsightsPack = computed(() =>
  `${storedPackTitle.value} ${storedPackRespath.value}`.toLowerCase().includes("insight"),
)
const isWebinarsPack = computed(() =>
  `${storedPackTitle.value} ${storedPackRespath.value}`.toLowerCase().includes("webinar"),
)
const isReportsPack = computed(() =>
  `${storedPackTitle.value} ${storedPackRespath.value}`.toLowerCase().includes("report"),
)
const copy = computed(() => {
  const page = t.value.resourcePackDetailPage
  return {
    ...page,
    title: isWebinarsPack.value ? page.titleWebinars : isReportsPack.value ? page.titleReports : page.titleDefault,
    subtitle: isInsightsPack.value
      ? page.subtitleInsights
      : isWebinarsPack.value
        ? page.subtitleWebinars
        : isReportsPack.value
          ? page.subtitleReports
        : page.subtitleDefault,
  }
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

const isVideoGrid = computed(() =>
  filteredFiles.value.length > 0 && filteredFiles.value.every((file) => normalizedType(file.file_type) === 1),
)

function normalizedType(type?: number | string) {
  return Number(type)
}

function fileTypeLabel(type?: number | string) {
  const normalized = normalizedType(type)
  const page = t.value.resourcePackDetailPage
  if (normalized === 1) return page.fileTypeWebinar
  if (normalized === 2) return page.fileTypePdf
  if (normalized === 3) return page.fileTypeZip
  return page.fileTypeFile
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

function formatMinuteDuration(minutes: number) {
  if (!Number.isFinite(minutes) || minutes <= 0) return ""
  return `${Math.floor(minutes)} ${t.value.common.minuteUnit}`
}

function formatDuration(file: ResourcePackFile) {
  const rawDuration = file.duration ?? file.duration_seconds
  if (typeof rawDuration === "string" && rawDuration.trim()) {
    const trimmed = rawDuration.trim()
    const numeric = Number(trimmed)
    if (!Number.isFinite(numeric)) {
      const minuteMatch = trimmed.match(/^(\d+(?:\.\d+)?)\s*(min|mins|minute|minutes)$/i)
      if (minuteMatch) return formatMinuteDuration(Number(minuteMatch[1]))
      if (trimmed.toLowerCase() === "unknown") return t.value.common.unknown
      return trimmed
    }
  }

  const seconds = Number(rawDuration || 0)
  if (Number.isFinite(seconds) && seconds > 0) {
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    const restSeconds = Math.floor(seconds % 60)
    if (hours > 0) return `${hours}:${String(minutes).padStart(2, "0")}:${String(restSeconds).padStart(2, "0")}`
    return `${minutes}:${String(restSeconds).padStart(2, "0")}`
  }

  const minutes = Number(file.duration_min || 0)
  if (Number.isFinite(minutes) && minutes > 0) return formatMinuteDuration(minutes)

  return t.value.common.unknown
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
    const resp = await apiClient(`/api/resource-pack-files/${encodeURIComponent(file.file_id)}/thumbnail-url`, {
      suppressErrorToast: true,
    })
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
      if (file.title) sessionStorage.setItem(`resource-pack-file-preview-title:${file.file_id}`, file.title)
      const target = router.resolve(`/resource-pack-files/${encodeURIComponent(file.file_id)}/preview`)
      window.open(target.href, "_blank", "noopener,noreferrer")
      return
    }
    if (normalizedType(file.file_type) === 1) {
      if (file.title) sessionStorage.setItem(`resource-pack-file-preview-title:${file.file_id}`, file.title)
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
            <button class="resource-detail-refresh-btn inline-flex h-9 items-center gap-2 rounded-xl border px-4 text-sm font-semibold" :disabled="!packId || loading" @click="loadFiles()">
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

    <section
      v-if="filteredFiles.length"
      :class="[
        'grid gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4',
        isVideoGrid ? '' : '2xl:grid-cols-5',
      ]"
    >
      <article
        v-for="file in filteredFiles"
        :key="file.file_id"
        :class="[
          'group overflow-hidden transition-all hover:-translate-y-0.5',
          normalizedType(file.file_type) === 2
            ? 'bg-transparent'
            : 'rounded-[18px] border border-border bg-white shadow-[0_10px_24px_rgba(15,23,42,0.08)] hover:shadow-[0_16px_34px_rgba(15,23,42,0.12)]',
        ]"
      >
        <button
          v-if="normalizedType(file.file_type) === 2"
          class="block w-full text-left"
          :disabled="openingFileId === file.file_id"
          @click="openFile(file)"
        >
          <div class="relative aspect-[0.72/1] overflow-hidden border border-border bg-slate-900 shadow-[0_10px_24px_rgba(15,23,42,0.14)]">
            <img
              v-if="thumbnailFor(file)"
              :src="thumbnailFor(file)"
              :alt="file.title || file.file_name || file.file_id"
              class="h-full w-full object-cover transition duration-500 group-hover:scale-105"
              loading="lazy"
            />
            <div v-else :class="['flex h-full w-full flex-col justify-end bg-gradient-to-br p-5 text-white', fallbackCoverClass(file.file_type)]">
              <h2 class="line-clamp-4 text-2xl font-black leading-tight">{{ file.title || file.file_name || file.file_id }}</h2>
            </div>

            <div class="absolute inset-x-0 bottom-0 h-1/2 bg-gradient-to-t from-slate-950/78 via-slate-950/32 to-transparent" />

            <div class="absolute inset-0 flex items-center justify-center opacity-0 transition-opacity duration-200 group-hover:opacity-100">
              <span class="inline-flex h-12 w-12 items-center justify-center rounded-full bg-white/95 text-slate-950 shadow-lg backdrop-blur">
                <Eye class="h-6 w-6" :stroke-width="2.5" />
              </span>
            </div>

            <div class="absolute bottom-6 left-5 right-5 text-white">
              <h2 class="line-clamp-2 text-lg font-black leading-tight text-white">{{ file.title || file.file_name || file.file_id }}</h2>
              <p v-if="file.description || file.file_name" class="mt-2 line-clamp-2 text-sm font-semibold leading-5 text-white">{{ file.description || file.file_name }}</p>
            </div>
          </div>

          <div v-if="file.updated_at" class="mt-2 flex items-center gap-2 text-sm text-muted-foreground">
            <CalendarDays class="h-4 w-4 shrink-0" />
            <span class="truncate">{{ formatBackendDateOnly(file.updated_at) }}</span>
          </div>
        </button>

        <template v-else>
          <button class="block w-full text-left" :disabled="openingFileId === file.file_id" @click="openFile(file)">
            <div class="relative aspect-[2.2/1] overflow-hidden bg-slate-900">
              <img
                v-if="thumbnailFor(file)"
                :src="thumbnailFor(file)"
                :alt="file.title || file.file_name || file.file_id"
                class="h-full w-full object-cover transition duration-500 group-hover:scale-105"
                loading="lazy"
              />
              <div v-else :class="['flex h-full w-full flex-col justify-between bg-gradient-to-br p-6 text-white', fallbackCoverClass(file.file_type)]">
                <div />
                <div>
                  <div class="mb-3 text-xs font-semibold uppercase tracking-[0.2em] text-white/70">{{ t.common.instituteName }}</div>
                  <h2 class="line-clamp-3 text-2xl font-black leading-tight">{{ file.title || file.file_name || file.file_id }}</h2>
                </div>
                <div />
              </div>

              <span class="absolute left-5 top-5 rounded-xl bg-white/95 px-3.5 py-1.5 text-sm font-medium text-slate-700 shadow-sm backdrop-blur">
                {{ fileTypeLabel(file.file_type) }}
              </span>

              <div class="absolute inset-0 flex items-center justify-center bg-slate-950/10 opacity-0 transition-opacity duration-200 group-hover:opacity-100">
                <span
                  :class="[
                    'inline-flex items-center justify-center rounded-full bg-white/95 text-slate-950 shadow-lg backdrop-blur',
                    normalizedType(file.file_type) === 1 ? 'h-9 w-9' : 'h-12 w-12',
                  ]"
                >
                  <Play v-if="normalizedType(file.file_type) === 1" class="h-5 w-5 fill-none" :stroke-width="2.4" />
                  <Eye v-else class="h-6 w-6" :stroke-width="2.4" />
                </span>
              </div>
            </div>
          </button>

          <button class="block w-full px-4 py-4 text-left" :disabled="openingFileId === file.file_id" @click="openFile(file)">
            <h2 class="line-clamp-2 min-h-[2.75rem] text-base font-bold leading-snug text-foreground transition-colors group-hover:text-primary">{{ file.title || file.file_name || file.file_id }}</h2>
            <p v-if="file.description || file.file_name" class="mt-2 line-clamp-2 text-sm leading-5 text-muted-foreground">{{ file.description || file.file_name }}</p>
            <div class="mt-3 flex items-center justify-between gap-4 text-sm text-muted-foreground">
              <span v-if="file.updated_at" class="inline-flex min-w-0 items-center gap-2">
                <CalendarDays class="h-4 w-4 shrink-0" />
                <span class="truncate">{{ formatBackendDateOnly(file.updated_at) }}</span>
              </span>
              <span v-else />
              <span v-if="normalizedType(file.file_type) === 1" class="inline-flex min-w-0 items-center gap-2">
                <Clock class="h-4 w-4 shrink-0" />
                <span class="truncate">{{ formatDuration(file) }}</span>
              </span>
              <span v-else class="truncate">{{ copy.size }}: {{ formatSize(file.file_size) }}</span>
            </div>
          </button>
        </template>
      </article>
    </section>

    <section v-else class="rounded-[16px] bg-white px-4 py-14 text-center shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10">
        <RefreshCw v-if="loading" class="h-8 w-8 animate-spin text-primary" />
        <FileArchive v-else class="h-8 w-8 text-primary" />
      </div>
      <h2 class="mt-4 text-lg font-semibold text-foreground">{{ loading ? copy.loading : search.trim() ? copy.noSearchTitle : copy.emptyTitle }}</h2>
      <p v-if="!loading" class="mx-auto mt-2 max-w-md text-sm leading-6 text-muted-foreground">{{ search.trim() ? copy.noSearchDesc : copy.emptyDesc }}</p>
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

<style scoped>
.resource-detail-refresh-btn {
  border-color: #e2e8f0;
  background: #ffffff;
  color: #334155;
  box-shadow: 0 8px 18px -16px rgba(15, 23, 42, 0.35);
  transition: transform 0.2s ease, border-color 0.2s ease, background-color 0.2s ease, color 0.2s ease, box-shadow 0.2s ease;
}

.resource-detail-refresh-btn:hover:not(:disabled) {
  border-color: rgba(37, 99, 235, 0.28);
  background: rgba(37, 99, 235, 0.08);
  color: #1d4ed8;
  box-shadow: 0 14px 28px -18px rgba(37, 99, 235, 0.42);
  transform: scale(1.02);
}

.resource-detail-refresh-btn:active:not(:disabled) {
  transform: scale(0.98);
}

.resource-detail-refresh-btn:focus-visible {
  outline: none;
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.16), 0 14px 28px -18px rgba(37, 99, 235, 0.42);
}

.resource-detail-refresh-btn:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}
</style>
