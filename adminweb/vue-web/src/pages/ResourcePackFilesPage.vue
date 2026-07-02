<script setup lang="ts">
import { FileText, Loader2, Plus, RefreshCw, Save, Trash2, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, humanizeKey, type JsonRecord } from "@/lib/display"

const fileTypeOptions = [
  { value: 1, label: "视频" },
  { value: 2, label: "PDF 文档" },
  { value: 3, label: "ZIP 压缩包" },
]

const pageSize = 10

const packs = ref<JsonRecord[]>([])
const files = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const loading = ref(false)
const detailLoading = ref(false)
const saving = ref(false)
const detailOpen = ref(false)
const mode = ref<"create" | "edit">("edit")
const pageToken = ref("")
const nextPageToken = ref("")
const previousTokens = ref<string[]>([])
const currentPage = ref(1)
const packFilter = ref("")
let detailRequestId = 0

const form = ref({
  file_id: "",
  pack_id: "",
  title: "",
  description: "",
  thumbnail_object_key: "",
  thumbnail_file_hash: "",
  file_type: 2,
  file_name: "",
  file_size: 0,
  file_hash: "",
  file_object_key: "",
  video_stream_uid: "",
  sort_order: 1,
  version: 0,
})

const selectedEntries = computed(() => Object.entries(selected.value || {}))
const canPrevious = computed(() => previousTokens.value.length > 0)
const canNext = computed(() => !!nextPageToken.value)
const selectedPack = computed(() => packById(form.value.pack_id))

function asRecordList(value: unknown) {
  return Array.isArray(value) ? value.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item)) : []
}

function fileId(file: JsonRecord | null) {
  return String(file?.file_id || "")
}

function fileTitle(file: JsonRecord | null) {
  return String(file?.title || file?.file_name || file?.file_id || "未命名资源文件")
}

function fileVersion(file: JsonRecord | null) {
  return Number(file?.version || 0)
}

function packId(pack: JsonRecord | null) {
  return String(pack?.pack_id || "")
}

function packTitle(pack: JsonRecord | null) {
  return String(pack?.title || pack?.pack_id || "未知资源包")
}

function packById(id: unknown) {
  const value = String(id || "")
  return packs.value.find((pack) => packId(pack) === value) || null
}

function ownerText(file: JsonRecord | null) {
  const pack = packById(file?.pack_id)
  const id = String(file?.pack_id || "-")
  return `${packTitle(pack)}（${id}）`
}

function fileTypeLabel(value: unknown) {
  const numeric = Number(value || 0)
  return fileTypeOptions.find((option) => option.value === numeric)?.label || String(value || "-")
}

function fileSize(value: unknown) {
  const size = Number(value || 0)
  if (!Number.isFinite(size) || size <= 0) return "-"
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

function fillForm(file: JsonRecord | null) {
  form.value = {
    file_id: String(file?.file_id || ""),
    pack_id: String(file?.pack_id || packFilter.value || packs.value[0]?.pack_id || ""),
    title: String(file?.title || ""),
    description: String(file?.description || ""),
    thumbnail_object_key: String(file?.thumbnail_object_key || ""),
    thumbnail_file_hash: String(file?.thumbnail_file_hash || ""),
    file_type: Number(file?.file_type || 2),
    file_name: String(file?.file_name || ""),
    file_size: Number(file?.file_size || 0),
    file_hash: String(file?.file_hash || ""),
    file_object_key: String(file?.file_object_key || ""),
    video_stream_uid: String(file?.video_stream_uid || ""),
    sort_order: Number(file?.sort_order || 1),
    version: Number(file?.version || 0),
  }
}

function mergeFileDetail(id: string, detail: JsonRecord) {
  const index = files.value.findIndex((file) => fileId(file) === id)
  const base = index >= 0 ? files.value[index] : selected.value || {}
  const merged = { ...base, ...detail }
  if (index >= 0) {
    files.value.splice(index, 1, merged)
  }
  if (selected.value && fileId(selected.value) === id) {
    selected.value = merged
    fillForm(merged)
  }
}

async function loadFileDetail(file: JsonRecord | null) {
  if (!file) return
  const id = fileId(file)
  if (!id) return

  const requestId = ++detailRequestId
  detailLoading.value = true
  try {
    const detail = await apiClient<JsonRecord>(`/api/lms/resource-pack-files/${encodeURIComponent(id)}`)
    if (requestId !== detailRequestId) return
    if (detail && typeof detail === "object" && !Array.isArray(detail)) {
      mergeFileDetail(id, detail)
    }
  } catch (err) {
    console.error(err)
    if (requestId === detailRequestId) {
      toast.error("资源文件详情加载失败")
    }
  } finally {
    if (requestId === detailRequestId) {
      detailLoading.value = false
    }
  }
}

async function loadPacks() {
  const data = await apiClient<JsonRecord>("/api/lms/resource-packs?page_size=1000")
  packs.value = asRecordList(data.packs || data.items)
}

async function loadFiles() {
  const url = new URL("/api/lms/resource-pack-files", window.location.origin)
  url.searchParams.set("page_size", String(pageSize))
  if (packFilter.value) url.searchParams.set("pack_id", packFilter.value)
  if (pageToken.value) url.searchParams.set("page_token", pageToken.value)

  const data = await apiClient<JsonRecord>(`${url.pathname}${url.search}`)
  files.value = asRecordList(data.files || data.items)
  nextPageToken.value = String(data.next_page_token || "")
  const nextSelected = files.value.find((file) => fileId(file) === form.value.file_id) || files.value[0] || null
  selectFile(nextSelected)
}

async function load() {
  loading.value = true
  try {
    await loadPacks()
    await loadFiles()
  } catch (err) {
    console.error(err)
    packs.value = []
    files.value = []
    selected.value = null
    fillForm(null)
    toast.error("资源文件加载失败")
  } finally {
    loading.value = false
  }
}

function selectFile(file: JsonRecord | null, openDetail = false) {
  selected.value = file
  mode.value = file ? "edit" : "create"
  fillForm(file)
  void loadFileDetail(file)
  if (openDetail) detailOpen.value = true
}

function openFileDetail(file: JsonRecord) {
  selectFile(file, true)
}

function closeFileDetail() {
  detailOpen.value = false
}

function startCreate() {
  detailRequestId += 1
  detailLoading.value = false
  selected.value = null
  mode.value = "create"
  fillForm(null)
  detailOpen.value = true
}

async function saveFile() {
  if (!form.value.title.trim()) {
    toast.error("资源文件标题不能为空")
    return
  }
  if (!form.value.file_type) {
    toast.error("请选择文件类型")
    return
  }
  if (mode.value === "create" && !form.value.pack_id) {
    toast.error("新增资源文件必须选择所属资源包")
    return
  }
  if (mode.value === "edit" && (!form.value.file_id || form.value.version <= 0)) {
    toast.error("更新资源文件需要有效的 file_id 和 version")
    return
  }

  saving.value = true
  try {
    const body: JsonRecord = {
      title: form.value.title.trim(),
      description: form.value.description,
      thumbnail_object_key: form.value.thumbnail_object_key,
      thumbnail_file_hash: form.value.thumbnail_file_hash,
      file_type: form.value.file_type,
      file_name: form.value.file_name,
      file_size: Number(form.value.file_size || 0),
      file_hash: form.value.file_hash,
      file_object_key: form.value.file_object_key,
      video_stream_uid: form.value.video_stream_uid,
      sort_order: Number(form.value.sort_order || 0),
    }

    if (mode.value === "create") {
      if (form.value.file_id.trim()) body.file_id = form.value.file_id.trim()
      await apiClient(`/api/lms/resource-packs/${encodeURIComponent(form.value.pack_id)}/files`, {
        method: "POST",
        body: JSON.stringify(body),
      })
      toast.success("资源文件已创建")
    } else {
      body.version = form.value.version
      await apiClient(`/api/lms/resource-pack-files/${encodeURIComponent(form.value.file_id)}`, {
        method: "PUT",
        body: JSON.stringify(body),
      })
      toast.success("资源文件已保存")
    }

    await loadFiles()
  } catch (err) {
    console.error(err)
    toast.error("资源文件保存失败")
  } finally {
    saving.value = false
  }
}

async function deleteFile() {
  if (!selected.value) return
  const id = fileId(selected.value)
  const version = fileVersion(selected.value)
  if (!id || version <= 0) {
    toast.error("删除资源文件需要有效的 file_id 和 version")
    return
  }
  if (!window.confirm(`确定删除资源文件「${fileTitle(selected.value)}」吗？`)) return

  saving.value = true
  try {
    await apiClient(`/api/lms/resource-pack-files/${encodeURIComponent(id)}?version=${version}`, {
      method: "DELETE",
    })
    toast.success("资源文件已删除")
    await loadFiles()
  } catch (err) {
    console.error(err)
    toast.error("资源文件删除失败")
  } finally {
    saving.value = false
  }
}

function resetPaging() {
  pageToken.value = ""
  nextPageToken.value = ""
  previousTokens.value = []
  currentPage.value = 1
}

function changePackFilter() {
  resetPaging()
  void loadFiles()
}

function previousPage() {
  if (!canPrevious.value) return
  pageToken.value = previousTokens.value.pop() || ""
  currentPage.value = Math.max(1, currentPage.value - 1)
  void loadFiles()
}

function nextPage() {
  if (!canNext.value) return
  previousTokens.value.push(pageToken.value)
  pageToken.value = nextPageToken.value
  currentPage.value += 1
  void loadFiles()
}

onMounted(load)
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight text-slate-950">资源文件配置</h1>
        <p class="mt-2 text-slate-600">维护资源包内的文件。新增时选择所属资源包，详情中清楚展示归属。</p>
      </div>
      <div class="flex gap-3">
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" :disabled="loading" @click="load">
          <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
          <RefreshCw v-else class="h-4 w-4" />
          刷新
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl bg-blue-700 px-4 py-3 text-sm font-bold text-white shadow-sm" type="button" @click="startCreate">
          <Plus class="h-4 w-4" />
          新增资源文件
        </button>
      </div>
    </header>

    <div class="rounded-2xl border border-slate-200 bg-white px-5 py-4 shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h2 class="text-lg font-black">按资源包筛选</h2>
          <p class="mt-1 text-sm text-slate-500">可查看全部文件，也可以只看某个资源包下的文件。</p>
        </div>
        <select v-model="packFilter" class="h-10 min-w-[360px] rounded-xl border border-slate-200 px-3 text-sm font-bold" @change="changePackFilter">
          <option value="">全部资源包</option>
          <option v-for="pack in packs" :key="packId(pack)" :value="packId(pack)">{{ packTitle(pack) }}（{{ packId(pack) }}）</option>
        </select>
      </div>
    </div>

    <div class="grid gap-6">
      <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
        <div class="flex items-center justify-between border-b border-slate-200 px-5 py-4">
          <div>
            <h2 class="text-xl font-black">资源文件列表</h2>
            <p class="mt-1 text-sm text-slate-500">选择文件后通过弹框查看详情并编辑。</p>
          </div>
          <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-bold text-slate-500">本页 {{ files.length }} / {{ pageSize }} 条</span>
        </div>

        <div v-if="loading" class="px-6 py-10 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          正在加载...
        </div>
        <div v-else-if="!files.length" class="px-6 py-10 text-center text-slate-500">暂无资源文件</div>
        <div v-else class="grid grid-cols-[minmax(0,1fr)_88px_92px_110px] gap-4 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-500">
          <span>资源文件</span>
          <span class="text-center">排序</span>
          <span class="text-right">类型</span>
          <span class="text-right">操作</span>
        </div>
        <button
          v-for="file in files"
          :key="fileId(file)"
          class="grid w-full grid-cols-[minmax(0,1fr)_88px_92px_110px] gap-4 border-b border-slate-100 px-5 py-4 text-left transition last:border-b-0 hover:bg-slate-50"
          :class="fileId(selected) === fileId(file) ? 'bg-sky-50' : ''"
          type="button"
          @click="openFileDetail(file)"
        >
          <div class="min-w-0">
            <div class="truncate text-lg font-black text-slate-950">{{ fileTitle(file) }}</div>
            <div class="mt-1 line-clamp-2 text-sm text-slate-500">{{ file.description || "-" }}</div>
            <div class="mt-2 flex flex-wrap items-center gap-2 text-xs font-bold text-slate-500">
              <span class="max-w-full truncate rounded-full bg-blue-50 px-2 py-1 text-blue-700">所属：{{ ownerText(file) }}</span>
              <span class="rounded-full bg-slate-100 px-2 py-1">Version：{{ file.version || 0 }}</span>
            </div>
          </div>
          <span class="self-center text-center text-sm font-black text-slate-700">{{ file.sort_order || 0 }}</span>
          <span class="h-fit self-center justify-self-end whitespace-nowrap rounded-full border border-slate-200 bg-white px-3 py-1 text-xs font-black text-slate-700">
            {{ fileTypeLabel(file.file_type) }}
          </span>
          <span class="self-center justify-self-end rounded-xl border border-blue-100 bg-blue-50 px-3 py-2 text-sm font-black text-blue-700">查看详情</span>
        </button>

        <div class="flex items-center justify-end gap-3 border-t border-slate-200 px-5 py-4">
          <span class="mr-auto text-sm font-bold text-slate-500">第 {{ currentPage }} 页</span>
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrevious || loading" @click="previousPage">上一页</button>
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext || loading" @click="nextPage">下一页</button>
        </div>
      </section>

      <section v-if="detailOpen" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-6">
        <div class="flex max-h-[88vh] w-full max-w-[1100px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
        <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-5 py-4">
          <div>
            <h2 class="text-xl font-black">{{ mode === "create" ? "新增资源文件" : "资源文件详情" }}</h2>
            <p class="mt-1 text-sm text-slate-500">资源文件归属在创建时选择，创建后不可修改所属资源包。</p>
            <p v-if="detailLoading" class="mt-1 inline-flex items-center gap-2 text-xs font-bold text-blue-600">
              <Loader2 class="h-3.5 w-3.5 animate-spin" />
              正在加载 get 详情...
            </p>
          </div>
          <button class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm hover:bg-slate-50 hover:text-slate-900" type="button" aria-label="关闭" @click="closeFileDetail">
            <X class="h-5 w-5" />
          </button>
        </div>

        <div class="flex-1 overflow-y-auto p-5">
          <div class="mb-5 rounded-2xl border border-blue-100 bg-blue-50/70 p-4">
            <div class="text-xs font-black uppercase text-blue-600">所属资源包</div>
            <div class="mt-1 text-lg font-black text-slate-950">{{ selectedPack ? packTitle(selectedPack) : "请选择资源包" }}</div>
            <div class="mt-1 break-all text-sm font-bold text-blue-700">{{ form.pack_id || "-" }}</div>
          </div>

          <div class="mb-3 text-sm font-black text-slate-700">基础信息</div>
          <div class="grid gap-3 md:grid-cols-2">
          <label class="text-sm font-bold">
            File ID
            <input v-model="form.file_id" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3 disabled:bg-slate-100" :disabled="mode === 'edit'" placeholder="留空则由后台生成" />
          </label>
          <label class="text-sm font-bold">
            所属资源包
            <select v-model="form.pack_id" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3 disabled:bg-slate-100" :disabled="mode === 'edit'">
              <option value="">请选择资源包</option>
              <option v-for="pack in packs" :key="packId(pack)" :value="packId(pack)">{{ packTitle(pack) }}（{{ packId(pack) }}）</option>
            </select>
          </label>
          <label class="text-sm font-bold">
            标题
            <input v-model="form.title" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="资源文件标题" />
          </label>
          <label class="text-sm font-bold">
            文件类型
            <select v-model.number="form.file_type" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3">
              <option v-for="option in fileTypeOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
            </select>
          </label>
          <label class="md:col-span-2 text-sm font-bold">
            描述
            <textarea v-model="form.description" class="mt-2 min-h-20 w-full rounded-xl border border-slate-200 px-3 py-2" placeholder="资源文件描述" />
          </label>
          </div>

          <div class="mb-3 mt-5 border-t border-slate-100 pt-5 text-sm font-black text-slate-700">文件与封面</div>
          <div class="grid gap-3 md:grid-cols-2">
          <label class="text-sm font-bold">
            文件名
            <input v-model="form.file_name" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="example.pdf" />
          </label>
          <label class="text-sm font-bold">
            文件大小（字节）
            <input v-model.number="form.file_size" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" type="number" min="0" />
          </label>
          <label class="text-sm font-bold">
            文件 Object Key
            <input v-model="form.file_object_key" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="resource-packs/.../file.pdf" />
          </label>
          <label class="text-sm font-bold">
            文件 Hash
            <input v-model="form.file_hash" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="SHA256 Hash" />
          </label>
          <label class="text-sm font-bold">
            封面 Object Key
            <input v-model="form.thumbnail_object_key" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="thumbnail.jpg" />
          </label>
          <label class="text-sm font-bold">
            封面 Hash
            <input v-model="form.thumbnail_file_hash" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="SHA256 Hash" />
          </label>
          </div>

          <div class="mb-3 mt-5 border-t border-slate-100 pt-5 text-sm font-black text-slate-700">排序与版本</div>
          <div class="grid gap-3 md:grid-cols-3">
          <label class="text-sm font-bold">
            Video Stream UID
            <input v-model="form.video_stream_uid" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="视频资源可填" />
          </label>
          <label class="text-sm font-bold">
            排序
            <input v-model.number="form.sort_order" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" type="number" min="0" />
          </label>
          <label class="text-sm font-bold">
            Version
            <input v-model.number="form.version" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3 disabled:bg-slate-100" type="number" min="0" :disabled="mode === 'create'" />
          </label>
          </div>

        <div class="mt-5 flex flex-wrap items-center justify-between gap-3 border-t border-slate-100 pt-5">
          <button
            v-if="mode === 'edit'"
            class="inline-flex h-10 items-center gap-2 rounded-xl border border-red-200 bg-red-50 px-4 text-sm font-bold text-red-700 disabled:opacity-50"
            type="button"
            :disabled="saving"
            @click="deleteFile"
          >
            <Trash2 class="h-4 w-4" />
            删除资源文件
          </button>
          <span v-else class="hidden sm:block"></span>
          <button class="inline-flex h-10 min-w-[180px] items-center justify-center gap-2 rounded-xl bg-blue-700 px-4 font-bold text-white disabled:opacity-50" type="button" :disabled="saving" @click="saveFile">
            <Loader2 v-if="saving" class="h-4 w-4 animate-spin" />
            <Save v-else class="h-4 w-4" />
            {{ mode === "create" ? "创建资源文件" : "保存资源文件" }}
          </button>
        </div>

        <div v-if="selected" class="mt-6 rounded-2xl border border-slate-200 bg-slate-50 p-4">
          <div class="mb-3 flex items-center gap-2 text-sm font-black">
            <FileText class="h-4 w-4 text-blue-700" />
            完整字段
          </div>
          <div class="grid gap-3 md:grid-cols-2">
            <div class="rounded-xl bg-white p-3 md:col-span-2">
              <div class="text-[11px] font-black uppercase text-slate-400">Belongs To Resource Pack</div>
              <div class="mt-1 break-words text-sm font-semibold">{{ ownerText(selected) }}</div>
            </div>
            <div v-for="[key, value] in selectedEntries" :key="key" class="rounded-xl bg-white p-3">
              <div class="text-[11px] font-black uppercase text-slate-400">{{ humanizeKey(key) }}</div>
              <div v-if="key === 'file_type'" class="mt-1 break-words text-sm font-semibold">{{ fileTypeLabel(value) }}</div>
              <div v-else-if="key === 'file_size'" class="mt-1 break-words text-sm font-semibold">{{ fileSize(value) }}</div>
              <div v-else-if="typeof value === 'string' && key.endsWith('_at')" class="mt-1 break-words text-sm font-semibold">{{ formatDate(value) }}</div>
              <div v-else class="mt-1 break-words text-sm font-semibold">{{ value ?? "-" }}</div>
            </div>
          </div>
        </div>
        </div>
        </div>
      </section>
    </div>
  </section>
</template>
