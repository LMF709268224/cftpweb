<script setup lang="ts">
import { Loader2, Pencil, Plus, RefreshCw, Save, Search, Trash2, Upload, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"

type FormState = {
  courseware_ulid: string
  name: string
  description: string
  base_url: string
  version: number
}

const { isZh } = useAdminLanguage()
const copy = computed(() => isZh.value ? {
  title: "外部课件与 Token",
  subtitle: "维护第三方课件入口及分配给考生的专属访问 URL。",
  refresh: "刷新",
  search: "搜索课件",
  create: "新建课件",
  empty: "暂无外部课件",
  name: "课件名称",
  description: "说明",
  baseUrl: "基础/兜底 URL",
  baseUrlHint: "专属 URL 池耗尽时，考生将打开这个公共入口。",
  id: "课件 ID",
  updated: "更新时间",
  actions: "操作",
  edit: "编辑",
  remove: "删除",
  save: "保存",
  cancel: "取消",
  required: "请填写课件名称和第三方基础 URL",
  invalidUrl: "基础 URL 必须是 http 或 https 地址",
  saved: "外部课件已保存",
  deleted: "外部课件已删除",
  confirmDelete: "确认删除这个外部课件？已被课时使用的课件无法删除。",
  tokens: "专属访问 URL 资产",
  total: "总数",
  available: "可用",
  allocated: "已分配",
  importTokens: "导入专属 URL",
  importHint: "每行一个完整的 http/https 访问 URL，单次最多 1000 条；重复值由服务端自动忽略。",
  importPlaceholder: "https://partner.example/token/user-001\nhttps://partner.example/token/user-002",
  invalidTokenUrls: "请逐行填写有效的 http 或 https 完整访问 URL",
  import: "导入",
  imported: "成功导入 {count} 条，忽略重复 {duplicates} 条",
  token: "专属访问 URL",
  candidate: "绑定考生",
  allocatedAt: "分配时间",
  unallocated: "未分配",
  noTokens: "暂无专属访问 URL 记录",
  loadMore: "加载更多",
  loadFailed: "外部课件数据加载失败",
  detail: "配置与访问 URL 库",
} : {
  title: "External Courseware & Tokens",
  subtitle: "Manage third-party courseware entry points and dedicated candidate access URLs.",
  refresh: "Refresh",
  search: "Search courseware",
  create: "New Courseware",
  empty: "No external courseware",
  name: "Courseware Name",
  description: "Description",
  baseUrl: "Base / Fallback URL",
  baseUrlHint: "Candidates open this public entry point when the dedicated URL pool is exhausted.",
  id: "Courseware ID",
  updated: "Updated",
  actions: "Actions",
  edit: "Edit",
  remove: "Delete",
  save: "Save",
  cancel: "Cancel",
  required: "Courseware name and base URL are required",
  invalidUrl: "Base URL must use http or https",
  saved: "External courseware saved",
  deleted: "External courseware deleted",
  confirmDelete: "Delete this external courseware? Courseware linked to a lesson cannot be deleted.",
  tokens: "Dedicated Access URL Inventory",
  total: "Total",
  available: "Available",
  allocated: "Allocated",
  importTokens: "Import Access URLs",
  importHint: "One complete HTTP/HTTPS access URL per line, up to 1,000 per import. Duplicates are ignored by the service.",
  importPlaceholder: "https://partner.example/token/user-001\nhttps://partner.example/token/user-002",
  invalidTokenUrls: "Enter one valid HTTP or HTTPS access URL per line",
  import: "Import",
  imported: "Imported {count}; ignored {duplicates} duplicates",
  token: "Dedicated Access URL",
  candidate: "Candidate",
  allocatedAt: "Allocated At",
  unallocated: "Unallocated",
  noTokens: "No dedicated access URL records",
  loadMore: "Load More",
  loadFailed: "Failed to load external courseware",
  detail: "Configuration & Access URL Pool",
})

const items = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const tokens = ref<JsonRecord[]>([])
const stats = ref<JsonRecord>({})
const keyword = ref("")
const loading = ref(false)
const detailLoading = ref(false)
const saving = ref(false)
const importing = ref(false)
const formOpen = ref(false)
const importOpen = ref(false)
const mode = ref<"create" | "edit">("create")
const tokenInput = ref("")
const tokenNextCursor = ref("")
const tokenHasMore = ref(false)
const loadingMoreTokens = ref(false)
const form = ref<FormState>(emptyForm())

function emptyForm(): FormState {
  return { courseware_ulid: "", name: "", description: "", base_url: "", version: 0 }
}

function recordList(value: unknown): JsonRecord[] {
  return Array.isArray(value) ? value.filter((item): item is JsonRecord => Boolean(item) && typeof item === "object" && !Array.isArray(item)) : []
}

function coursewareId(item: JsonRecord | null) {
  return String(item?.courseware_ulid || "")
}

function isSafeBaseUrl(value: string) {
  try {
    const url = new URL(value)
    return url.protocol === "https:" || url.protocol === "http:"
  } catch {
    return false
  }
}

function maskToken(value: unknown) {
  const token = String(value || "")
  if (!token) return "-"
  if (token.length <= 8) return `${token.slice(0, 2)}***${token.slice(-2)}`
  return `${token.slice(0, 4)}...${token.slice(-4)}`
}

async function load() {
  loading.value = true
  try {
    const url = new URL("/api/lms/external-coursewares", window.location.origin)
    url.searchParams.set("page_size", "100")
    if (keyword.value.trim()) url.searchParams.set("keyword", keyword.value.trim())
    const data = await apiClient<JsonRecord>(`${url.pathname}${url.search}`)
    items.value = recordList(data.items)
    if (selected.value) {
      selected.value = items.value.find((item) => coursewareId(item) === coursewareId(selected.value)) || null
    }
    if (!selected.value && items.value.length) await selectItem(items.value[0])
  } catch (error) {
    console.error(error)
    toast.error(apiErrorMessage(error, copy.value.loadFailed))
  } finally {
    loading.value = false
  }
}

async function selectItem(item: JsonRecord) {
  selected.value = item
  const id = coursewareId(item)
  if (!id) return
  detailLoading.value = true
  try {
    const [detail, tokenStats, tokenList] = await Promise.all([
      apiClient<JsonRecord>(`/api/lms/external-coursewares/${encodeURIComponent(id)}`),
      apiClient<JsonRecord>(`/api/lms/external-coursewares/${encodeURIComponent(id)}/token-stats`),
      apiClient<JsonRecord>(`/api/lms/external-coursewares/${encodeURIComponent(id)}/tokens?page_size=100`),
    ])
    if (coursewareId(selected.value) !== id) return
    selected.value = detail
    stats.value = tokenStats
    tokens.value = recordList(tokenList.items)
    tokenNextCursor.value = String(tokenList.next_cursor || "")
    tokenHasMore.value = Boolean(tokenList.has_more && tokenNextCursor.value)
  } catch (error) {
    console.error(error)
    toast.error(apiErrorMessage(error, copy.value.loadFailed))
  } finally {
    detailLoading.value = false
  }
}

async function loadMoreTokens() {
  const id = coursewareId(selected.value)
  if (!id || !tokenHasMore.value || !tokenNextCursor.value || loadingMoreTokens.value) return
  loadingMoreTokens.value = true
  try {
    const data = await apiClient<JsonRecord>(`/api/lms/external-coursewares/${encodeURIComponent(id)}/tokens?page_size=100&cursor=${encodeURIComponent(tokenNextCursor.value)}`)
    if (coursewareId(selected.value) !== id) return
    tokens.value.push(...recordList(data.items))
    tokenNextCursor.value = String(data.next_cursor || "")
    tokenHasMore.value = Boolean(data.has_more && tokenNextCursor.value)
  } catch (error) {
    console.error(error)
    toast.error(apiErrorMessage(error, copy.value.loadFailed))
  } finally {
    loadingMoreTokens.value = false
  }
}

function openCreate() {
  mode.value = "create"
  form.value = emptyForm()
  formOpen.value = true
}

function openEdit() {
  if (!selected.value) return
  mode.value = "edit"
  form.value = {
    courseware_ulid: coursewareId(selected.value),
    name: String(selected.value.name || ""),
    description: String(selected.value.description || ""),
    base_url: String(selected.value.base_url || ""),
    version: Number(selected.value.version || 0),
  }
  formOpen.value = true
}

async function save() {
  if (!form.value.name.trim() || !form.value.base_url.trim()) {
    toast.error(copy.value.required)
    return
  }
  if (!isSafeBaseUrl(form.value.base_url.trim())) {
    toast.error(copy.value.invalidUrl)
    return
  }
  saving.value = true
  try {
    const body = JSON.stringify({
      name: form.value.name.trim(), description: form.value.description.trim(), base_url: form.value.base_url.trim(),
      version: form.value.version,
    })
    const data = mode.value === "create"
      ? await apiClient<JsonRecord>("/api/lms/external-coursewares", { method: "POST", body })
      : await apiClient<JsonRecord>(`/api/lms/external-coursewares/${encodeURIComponent(form.value.courseware_ulid)}`, { method: "PUT", body })
    formOpen.value = false
    toast.success(copy.value.saved)
    await load()
    const saved = items.value.find((item) => coursewareId(item) === coursewareId(data)) || data
    if (coursewareId(saved)) await selectItem(saved)
  } catch (error) {
    console.error(error)
    toast.error(apiErrorMessage(error, copy.value.loadFailed))
  } finally {
    saving.value = false
  }
}

async function removeSelected() {
  const id = coursewareId(selected.value)
  if (!id || !window.confirm(copy.value.confirmDelete)) return
  saving.value = true
  try {
    await apiClient(`/api/lms/external-coursewares/${encodeURIComponent(id)}`, { method: "DELETE" })
    selected.value = null
    tokens.value = []
    stats.value = {}
    toast.success(copy.value.deleted)
    await load()
  } catch (error) {
    console.error(error)
    toast.error(apiErrorMessage(error, copy.value.loadFailed))
  } finally {
    saving.value = false
  }
}

async function importTokens() {
  const id = coursewareId(selected.value)
  const values = tokenInput.value.split(/\r?\n/).map((value) => value.trim()).filter(Boolean)
  if (!id || !values.length) return
  if (values.some((value) => !isSafeBaseUrl(value))) {
    toast.error(copy.value.invalidTokenUrls)
    return
  }
  importing.value = true
  try {
    const result = await apiClient<JsonRecord>(`/api/lms/external-coursewares/${encodeURIComponent(id)}/tokens/import`, {
      method: "POST", body: JSON.stringify({ token_urls: values }),
    })
    toast.success(copy.value.imported.replace("{count}", String(result.imported_count || 0)).replace("{duplicates}", String(result.duplicate_count || 0)))
    tokenInput.value = ""
    importOpen.value = false
    if (selected.value) await selectItem(selected.value)
  } catch (error) {
    console.error(error)
    toast.error(apiErrorMessage(error, copy.value.loadFailed))
  } finally {
    importing.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="mx-auto w-full max-w-[1480px] space-y-6 px-4 py-5 md:px-8 md:py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div><h1 class="text-3xl font-black text-slate-950 md:text-4xl">{{ copy.title }}</h1><p class="mt-2 text-slate-600">{{ copy.subtitle }}</p></div>
      <div class="flex gap-3">
        <button class="inline-flex h-11 items-center gap-2 rounded-lg border border-slate-300 bg-white px-4 font-bold" :disabled="loading" @click="load"><RefreshCw :class="loading ? 'animate-spin' : ''" class="h-4 w-4" />{{ copy.refresh }}</button>
        <button class="inline-flex h-11 items-center gap-2 rounded-lg bg-blue-700 px-4 font-bold text-white" @click="openCreate"><Plus class="h-4 w-4" />{{ copy.create }}</button>
      </div>
    </header>

    <div class="flex h-11 max-w-md items-center gap-2 rounded-lg border border-slate-200 bg-white px-3"><Search class="h-4 w-4 text-slate-400" /><input v-model="keyword" class="min-w-0 flex-1 outline-none" :placeholder="copy.search" @keyup.enter="load" /></div>

    <div class="grid min-h-[560px] overflow-hidden rounded-lg border border-slate-200 bg-white lg:grid-cols-[360px_minmax(0,1fr)]">
      <aside class="border-b border-slate-200 lg:border-b-0 lg:border-r">
        <button v-for="item in items" :key="coursewareId(item)" class="block w-full border-b border-slate-100 px-5 py-4 text-left hover:bg-slate-50" :class="coursewareId(selected) === coursewareId(item) ? 'bg-blue-50' : ''" @click="selectItem(item)">
          <span class="block font-black text-slate-900">{{ item.name || '-' }}</span><span class="mt-1 block break-all text-xs text-slate-500">{{ coursewareId(item) }}</span>
        </button>
        <div v-if="!loading && !items.length" class="p-12 text-center text-slate-500">{{ copy.empty }}</div>
      </aside>

      <main class="min-w-0 p-5 md:p-6">
        <div v-if="selected" class="space-y-6">
          <div class="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-5">
            <div><div class="text-xs font-bold uppercase text-blue-700">{{ copy.detail }}</div><h2 class="mt-1 text-2xl font-black">{{ selected.name }}</h2><p class="mt-1 break-all text-sm text-slate-500">{{ coursewareId(selected) }}</p></div>
            <div class="flex gap-2"><button class="inline-flex h-10 items-center gap-2 rounded-lg border px-3 font-bold" @click="openEdit"><Pencil class="h-4 w-4" />{{ copy.edit }}</button><button class="inline-flex h-10 items-center gap-2 rounded-lg border border-red-200 px-3 font-bold text-red-600" :disabled="saving" @click="removeSelected"><Trash2 class="h-4 w-4" />{{ copy.remove }}</button></div>
          </div>

          <div class="grid gap-4 md:grid-cols-2"><div><div class="text-xs font-bold text-slate-500">{{ copy.baseUrl }}</div><div class="mt-1 break-all font-semibold">{{ selected.base_url }}</div></div><div><div class="text-xs font-bold text-slate-500">{{ copy.updated }}</div><div class="mt-1 font-semibold">{{ formatDate(String(selected.updated_at || '')) || '-' }}</div></div><div class="md:col-span-2"><div class="text-xs font-bold text-slate-500">{{ copy.description }}</div><div class="mt-1 whitespace-pre-wrap font-semibold">{{ selected.description || '-' }}</div></div></div>

          <section class="border-t border-slate-200 pt-5">
            <div class="flex flex-wrap items-center justify-between gap-3"><h3 class="text-lg font-black">{{ copy.tokens }}</h3><button class="inline-flex h-10 items-center gap-2 rounded-lg bg-blue-700 px-4 font-bold text-white" @click="importOpen = true"><Upload class="h-4 w-4" />{{ copy.importTokens }}</button></div>
            <div class="mt-4 grid grid-cols-3 gap-3"><div class="rounded-lg bg-slate-50 p-4"><div class="text-xs font-bold text-slate-500">{{ copy.total }}</div><div class="mt-1 text-2xl font-black">{{ stats.total_count || 0 }}</div></div><div class="rounded-lg bg-emerald-50 p-4"><div class="text-xs font-bold text-emerald-700">{{ copy.available }}</div><div class="mt-1 text-2xl font-black text-emerald-900">{{ stats.available_count || 0 }}</div></div><div class="rounded-lg bg-blue-50 p-4"><div class="text-xs font-bold text-blue-700">{{ copy.allocated }}</div><div class="mt-1 text-2xl font-black text-blue-900">{{ stats.allocated_count || 0 }}</div></div></div>
            <div class="mt-4 overflow-x-auto border-t border-slate-200"><table class="w-full text-left text-sm"><thead><tr class="text-slate-500"><th class="py-3">{{ copy.token }}</th><th class="py-3">{{ copy.candidate }}</th><th class="py-3">{{ copy.allocatedAt }}</th></tr></thead><tbody><tr v-for="item in tokens" :key="String(item.token_id)" class="border-t"><td class="py-3 font-mono" :title="String(item.token_url || '')">{{ maskToken(item.token_url) }}</td><td class="py-3 font-mono text-xs">{{ item.candidate_ulid || copy.unallocated }}</td><td class="py-3">{{ formatDate(String(item.allocated_at || '')) || '-' }}</td></tr></tbody></table><div v-if="!detailLoading && !tokens.length" class="py-10 text-center text-slate-500">{{ copy.noTokens }}</div><div v-if="tokenHasMore" class="border-t py-4 text-center"><button class="inline-flex h-10 items-center gap-2 rounded-lg border px-4 font-bold" :disabled="loadingMoreTokens" @click="loadMoreTokens"><Loader2 v-if="loadingMoreTokens" class="h-4 w-4 animate-spin" />{{ copy.loadMore }}</button></div></div>
          </section>
        </div>
        <div v-else class="flex min-h-[500px] items-center justify-center text-slate-500"><Loader2 v-if="loading" class="h-6 w-6 animate-spin" /><span v-else>{{ copy.empty }}</span></div>
      </main>
    </div>

    <Teleport to="body"><div v-if="formOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4"><form class="w-full max-w-2xl rounded-lg bg-white shadow-2xl" @submit.prevent="save"><header class="flex items-center justify-between border-b p-5"><h2 class="text-xl font-black">{{ mode === 'create' ? copy.create : copy.edit }}</h2><button type="button" class="h-9 w-9 rounded-full border" @click="formOpen = false"><X class="mx-auto h-4 w-4" /></button></header><div class="grid gap-4 p-5"><label><span class="text-sm font-bold">{{ copy.name }}</span><input v-model="form.name" class="mt-2 h-11 w-full rounded-lg border px-3" /></label><label><span class="text-sm font-bold">{{ copy.baseUrl }}</span><input v-model="form.base_url" class="mt-2 h-11 w-full rounded-lg border px-3" placeholder="https://partner.example/courseware" /><small class="mt-1 block text-slate-500">{{ copy.baseUrlHint }}</small></label><label><span class="text-sm font-bold">{{ copy.description }}</span><textarea v-model="form.description" class="mt-2 min-h-24 w-full rounded-lg border p-3" /></label></div><footer class="flex justify-end gap-3 border-t p-5"><button type="button" class="h-10 rounded-lg border px-4 font-bold" @click="formOpen = false">{{ copy.cancel }}</button><button class="inline-flex h-10 items-center gap-2 rounded-lg bg-blue-700 px-4 font-bold text-white" :disabled="saving"><Loader2 v-if="saving" class="h-4 w-4 animate-spin" /><Save v-else class="h-4 w-4" />{{ copy.save }}</button></footer></form></div></Teleport>

    <Teleport to="body"><div v-if="importOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4"><div class="w-full max-w-xl rounded-lg bg-white shadow-2xl"><header class="flex items-center justify-between border-b p-5"><h2 class="text-xl font-black">{{ copy.importTokens }}</h2><button class="h-9 w-9 rounded-full border" @click="importOpen = false"><X class="mx-auto h-4 w-4" /></button></header><div class="p-5"><p class="mb-3 text-sm text-slate-600">{{ copy.importHint }}</p><textarea v-model="tokenInput" class="min-h-64 w-full rounded-lg border p-3 font-mono text-sm" :placeholder="copy.importPlaceholder" /></div><footer class="flex justify-end gap-3 border-t p-5"><button class="h-10 rounded-lg border px-4 font-bold" @click="importOpen = false">{{ copy.cancel }}</button><button class="inline-flex h-10 items-center gap-2 rounded-lg bg-blue-700 px-4 font-bold text-white" :disabled="importing || !tokenInput.trim()" @click="importTokens"><Loader2 v-if="importing" class="h-4 w-4 animate-spin" /><Upload v-else class="h-4 w-4" />{{ copy.import }}</button></footer></div></div></Teleport>
  </section>
</template>
