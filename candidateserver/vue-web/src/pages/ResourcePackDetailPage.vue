<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { RouterLink, useRoute } from "vue-router"
import { toast } from "vue-sonner"
import { ArrowLeft, Download, ExternalLink, FileArchive, FileText, Play, RefreshCw } from "lucide-vue-next"
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
}

const route = useRoute()
const { lang } = useTranslation()
const loading = ref(false)
const openingFileId = ref("")
const files = ref<ResourcePackFile[]>([])
const nextPageToken = ref("")

const packId = computed(() => String(route.query.id || ""))
const copy = computed(() => lang.value === "zh"
  ? {
      title: "资源包详情",
      subtitle: "打开文件前会向后端申请一次性预览链接，权限和有效期由服务端控制。",
      back: "返回资源包",
      emptyTitle: "这个资源包暂无文件",
      emptyDesc: "管理员添加文件并发布后，会显示在这里。",
      open: "打开",
      download: "下载/预览",
      size: "文件大小",
      updated: "更新于",
      refresh: "刷新",
      missing: "缺少资源包 ID",
      loading: "加载中...",
      loadMore: "加载更多",
      noViewUrl: "暂时无法打开文件，请稍后再试。",
    }
  : {
      title: "Resource Pack Detail",
      subtitle: "A temporary view URL is requested from the backend before opening a file.",
      back: "Back to packs",
      emptyTitle: "No files in this pack",
      emptyDesc: "Files will appear here after admins add and publish them.",
      open: "Open",
      download: "Download / Preview",
      size: "File size",
      updated: "Updated",
      refresh: "Refresh",
      missing: "Missing resource pack ID",
      loading: "Loading...",
      loadMore: "Load more",
      noViewUrl: "Unable to open this file right now. Please try again later.",
    })

const orderedFiles = computed(() =>
  files.value.slice().sort((a, b) => Number(a.sort_order || 0) - Number(b.sort_order || 0)),
)

function fileIcon(type?: number | string) {
  const normalized = Number(type)
  if (normalized === 1) return Play
  if (normalized === 2) return FileText
  return FileArchive
}

function fileTypeLabel(type?: number | string) {
  const normalized = Number(type)
  if (normalized === 1) return "Video"
  if (normalized === 2) return "PDF"
  if (normalized === 3) return "ZIP"
  return "File"
}

function formatSize(size?: number) {
  if (!size) return "-"
  if (size < 1024 * 1024) return `${Math.round(size / 1024)} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
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
  } finally {
    loading.value = false
  }
}

async function openFile(file: ResourcePackFile) {
  if (!file.file_id) return
  openingFileId.value = file.file_id
  try {
    const resp = await apiClient(`/api/resource-pack-files/${encodeURIComponent(file.file_id)}/view-url`)
    const url = resp?.view_url
    if (!url) {
      toast.error(copy.value.noViewUrl)
      return
    }
    window.open(url, "_blank", "noopener,noreferrer")
  } finally {
    openingFileId.value = ""
  }
}

watch(packId, () => {
  files.value = []
  void loadFiles()
})

onMounted(() => {
  void loadFiles()
})
</script>

<template>
  <AppShell content-class="p-4">
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

    <section v-if="orderedFiles.length" class="grid gap-4">
      <article
        v-for="file in orderedFiles"
        :key="file.file_id"
        class="group relative flex flex-col gap-4 overflow-hidden rounded-[14px] bg-white p-5 shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-colors hover:bg-white md:flex-row md:items-center md:justify-between"
      >
        <div class="absolute left-0 top-0 h-full w-1 bg-primary/45" />
        <div class="flex gap-4">
          <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary transition-transform group-hover:scale-105">
            <component :is="fileIcon(file.file_type)" class="h-5 w-5" />
          </div>
          <div>
            <div class="mb-2 inline-flex rounded-full border border-primary/20 bg-primary/10 px-2.5 py-1 text-xs font-semibold text-primary">{{ fileTypeLabel(file.file_type) }}</div>
            <h2 class="text-lg font-semibold text-card-foreground">{{ file.title || file.file_name || file.file_id }}</h2>
            <p class="mt-1 max-w-3xl text-sm leading-6 text-muted-foreground">{{ file.description || file.file_name }}</p>
            <div class="mt-3 flex flex-wrap gap-3 text-xs text-muted-foreground">
              <span>{{ copy.size }}: {{ formatSize(file.file_size) }}</span>
              <span v-if="file.updated_at">{{ copy.updated }}: {{ file.updated_at }}</span>
            </div>
          </div>
        </div>
        <button
          class="btn btn-primary shrink-0 rounded-lg shadow-sm shadow-primary/20"
          :disabled="openingFileId === file.file_id"
          @click="openFile(file)"
        >
          <RefreshCw v-if="openingFileId === file.file_id" class="h-4 w-4 animate-spin" />
          <ExternalLink v-else-if="Number(file.file_type) === 1" class="h-4 w-4" />
          <Download v-else class="h-4 w-4" />
          {{ openingFileId === file.file_id ? copy.loading : Number(file.file_type) === 1 ? copy.open : copy.download }}
        </button>
      </article>
    </section>

    <section v-else class="rounded-[16px] bg-white px-4 py-14 text-center shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10">
        <RefreshCw v-if="loading" class="h-8 w-8 animate-spin text-primary" />
        <FileArchive v-else class="h-8 w-8 text-primary" />
      </div>
      <h2 class="mt-4 text-lg font-semibold text-foreground">{{ loading ? copy.loading : copy.emptyTitle }}</h2>
      <p class="mt-2 text-sm text-muted-foreground">{{ copy.emptyDesc }}</p>
    </section>

    <div v-if="nextPageToken" class="mt-4 text-center">
      <button class="btn btn-outline rounded-lg" :disabled="loading" @click="loadFiles(nextPageToken)">
        {{ loading ? copy.loading : copy.loadMore }}
      </button>
    </div>
  </AppShell>
</template>
