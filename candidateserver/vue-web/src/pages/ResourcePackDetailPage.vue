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
  const resp = await apiClient(`/api/resource-pack-files/${encodeURIComponent(file.file_id)}/view-url`)
  const url = resp?.view_url
  if (!url) {
    toast.error("No view URL")
    return
  }
  window.open(url, "_blank", "noopener,noreferrer")
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
    <section class="rounded-[28px] bg-white p-5 shadow-[0_12px_30px_rgba(15,74,82,0.06)]">
      <RouterLink to="/resource-packs" class="mb-5 inline-flex items-center gap-2 text-sm font-bold text-[#0f766e]">
        <ArrowLeft class="h-4 w-4" />
        {{ copy.back }}
      </RouterLink>
      <div class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <h1 class="text-3xl font-black text-slate-950">{{ copy.title }}</h1>
          <p class="mt-2 max-w-2xl text-sm leading-6 text-slate-500">{{ packId || copy.missing }}</p>
          <p class="mt-2 max-w-2xl text-sm leading-6 text-slate-500">{{ copy.subtitle }}</p>
        </div>
        <button class="btn btn-outline rounded-2xl" :disabled="!packId || loading" @click="loadFiles()">
          <RefreshCw :class="['h-4 w-4', loading ? 'animate-spin' : '']" />
          {{ copy.refresh }}
        </button>
      </div>
    </section>

    <section v-if="orderedFiles.length" class="mt-5 grid gap-4">
      <article
        v-for="file in orderedFiles"
        :key="file.file_id"
        class="flex flex-col gap-4 rounded-[24px] bg-white p-5 shadow-[0_12px_30px_rgba(15,74,82,0.06)] md:flex-row md:items-center md:justify-between"
      >
        <div class="flex gap-4">
          <component :is="fileIcon(file.file_type)" class="mt-1 h-8 w-8 shrink-0 text-[#0f766e]" />
          <div>
            <div class="mb-2 inline-flex rounded-full bg-[#e7f6f1] px-2.5 py-1 text-xs font-bold text-[#0f766e]">{{ fileTypeLabel(file.file_type) }}</div>
            <h2 class="text-lg font-black text-slate-950">{{ file.title || file.file_name || file.file_id }}</h2>
            <p class="mt-1 max-w-3xl text-sm leading-6 text-slate-500">{{ file.description || file.file_name }}</p>
            <div class="mt-3 flex flex-wrap gap-3 text-xs text-slate-500">
              <span>{{ copy.size }}: {{ formatSize(file.file_size) }}</span>
              <span v-if="file.updated_at">{{ copy.updated }}: {{ file.updated_at }}</span>
            </div>
          </div>
        </div>
        <button class="inline-flex items-center justify-center gap-2 rounded-2xl bg-[#0f4a52] px-4 py-2.5 text-sm font-bold text-white" @click="openFile(file)">
          <ExternalLink v-if="Number(file.file_type) === 1" class="h-4 w-4" />
          <Download v-else class="h-4 w-4" />
          {{ Number(file.file_type) === 1 ? copy.open : copy.download }}
        </button>
      </article>
    </section>

    <section v-else class="mt-5 rounded-[24px] border border-dashed border-slate-200 bg-white p-10 text-center">
      <FileArchive class="mx-auto h-12 w-12 text-slate-300" />
      <h2 class="mt-4 text-lg font-black text-slate-950">{{ loading ? "Loading..." : copy.emptyTitle }}</h2>
      <p class="mt-2 text-sm text-slate-500">{{ copy.emptyDesc }}</p>
    </section>

    <div v-if="nextPageToken" class="mt-5 text-center">
      <button class="btn btn-outline rounded-2xl" :disabled="loading" @click="loadFiles(nextPageToken)">
        {{ loading ? "Loading..." : "Load more" }}
      </button>
    </div>
  </AppShell>
</template>
