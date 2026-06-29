<script setup lang="ts">
import { FileBox, Loader2, Plus, RefreshCw, Save, Trash2 } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, humanizeKey, type JsonRecord } from "@/lib/display"

const pageSize = 20

const packs = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const loading = ref(false)
const detailLoading = ref(false)
const saving = ref(false)
const mode = ref<"create" | "edit">("edit")
const pageToken = ref("")
const nextPageToken = ref("")
const previousTokens = ref<string[]>([])
let detailRequestId = 0

const form = ref({
  pack_id: "",
  title: "",
  description: "",
  thumbnail_object_key: "",
  thumbnail_file_hash: "",
  respath: "",
  status: "Active",
  version: 0,
  icon: "",
  category: "",
})

const selectedEntries = computed(() => Object.entries(selected.value || {}))
const canPrevious = computed(() => previousTokens.value.length > 0)
const canNext = computed(() => !!nextPageToken.value)

function asRecordList(value: unknown) {
  return Array.isArray(value) ? value.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item)) : []
}

function packId(pack: JsonRecord | null) {
  return String(pack?.pack_id || "")
}

function packTitle(pack: JsonRecord | null) {
  return String(pack?.title || pack?.pack_id || "未命名资源包")
}

function packVersion(pack: JsonRecord | null) {
  return Number(pack?.version || 0)
}

function fillForm(pack: JsonRecord | null) {
  form.value = {
    pack_id: String(pack?.pack_id || ""),
    title: String(pack?.title || ""),
    description: String(pack?.description || ""),
    thumbnail_object_key: String(pack?.thumbnail_object_key || ""),
    thumbnail_file_hash: String(pack?.thumbnail_file_hash || ""),
    respath: String(pack?.respath || ""),
    status: String(pack?.status || "Active"),
    version: Number(pack?.version || 0),
    icon: String(pack?.icon || ""),
    category: String(pack?.category || ""),
  }
}

function mergePackDetail(id: string, detail: JsonRecord) {
  const index = packs.value.findIndex((pack) => packId(pack) === id)
  const base = index >= 0 ? packs.value[index] : selected.value || {}
  const merged = { ...base, ...detail }
  if (index >= 0) {
    packs.value.splice(index, 1, merged)
  }
  if (selected.value && packId(selected.value) === id) {
    selected.value = merged
    fillForm(merged)
  }
}

async function loadPackDetail(pack: JsonRecord | null) {
  if (!pack) return
  const id = packId(pack)
  if (!id) return

  const requestId = ++detailRequestId
  detailLoading.value = true
  try {
    const detail = await apiClient<JsonRecord>(`/api/lms/resource-packs/${encodeURIComponent(id)}`)
    if (requestId !== detailRequestId) return
    if (detail && typeof detail === "object" && !Array.isArray(detail)) {
      mergePackDetail(id, detail)
    }
  } catch (err) {
    console.error(err)
    if (requestId === detailRequestId) {
      toast.error("资源包详情加载失败")
    }
  } finally {
    if (requestId === detailRequestId) {
      detailLoading.value = false
    }
  }
}

async function load() {
  loading.value = true
  try {
    const url = new URL("/api/lms/resource-packs", window.location.origin)
    url.searchParams.set("page_size", String(pageSize))
    if (pageToken.value) url.searchParams.set("page_token", pageToken.value)

    const data = await apiClient<JsonRecord>(`${url.pathname}${url.search}`)
    packs.value = asRecordList(data.packs || data.items)
    nextPageToken.value = String(data.next_page_token || "")
    const nextSelected = packs.value.find((pack) => packId(pack) === form.value.pack_id) || packs.value[0] || null
    selectPack(nextSelected)
  } catch (err) {
    console.error(err)
    packs.value = []
    selected.value = null
    fillForm(null)
    toast.error("资源包列表加载失败")
  } finally {
    loading.value = false
  }
}

function selectPack(pack: JsonRecord | null) {
  selected.value = pack
  mode.value = pack ? "edit" : "create"
  fillForm(pack)
  void loadPackDetail(pack)
}

function startCreate() {
  detailRequestId += 1
  detailLoading.value = false
  selected.value = null
  mode.value = "create"
  fillForm(null)
}

async function savePack() {
  if (!form.value.title.trim()) {
    toast.error("资源包标题不能为空")
    return
  }
  if (mode.value === "edit" && (!form.value.pack_id || form.value.version <= 0)) {
    toast.error("更新资源包需要有效的 pack_id 和 version")
    return
  }

  saving.value = true
  try {
    const body: JsonRecord = {
      title: form.value.title.trim(),
      description: form.value.description,
      thumbnail_object_key: form.value.thumbnail_object_key,
      thumbnail_file_hash: form.value.thumbnail_file_hash,
      respath: form.value.respath,
      icon: form.value.icon,
      category: form.value.category,
    }

    if (mode.value === "create") {
      if (form.value.pack_id.trim()) body.pack_id = form.value.pack_id.trim()
      await apiClient("/api/lms/resource-packs", {
        method: "POST",
        body: JSON.stringify(body),
      })
      toast.success("资源包已创建")
    } else {
      body.status = form.value.status
      body.version = form.value.version
      await apiClient(`/api/lms/resource-packs/${encodeURIComponent(form.value.pack_id)}`, {
        method: "PUT",
        body: JSON.stringify(body),
      })
      toast.success("资源包已保存")
    }

    await load()
  } catch (err) {
    console.error(err)
    toast.error("资源包保存失败")
  } finally {
    saving.value = false
  }
}

async function deletePack() {
  if (!selected.value) return
  const id = packId(selected.value)
  const version = packVersion(selected.value)
  if (!id || version <= 0) {
    toast.error("删除资源包需要有效的 pack_id 和 version")
    return
  }
  if (!window.confirm(`确定删除资源包「${packTitle(selected.value)}」吗？`)) return

  saving.value = true
  try {
    await apiClient(`/api/lms/resource-packs/${encodeURIComponent(id)}?version=${version}`, {
      method: "DELETE",
    })
    toast.success("资源包已删除")
    await load()
  } catch (err) {
    console.error(err)
    toast.error("资源包删除失败")
  } finally {
    saving.value = false
  }
}

function previousPage() {
  if (!canPrevious.value) return
  pageToken.value = previousTokens.value.pop() || ""
  void load()
}

function nextPage() {
  if (!canNext.value) return
  previousTokens.value.push(pageToken.value)
  pageToken.value = nextPageToken.value
  void load()
}

onMounted(load)
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight text-slate-950">资源包配置</h1>
        <p class="mt-2 text-slate-600">维护 Resource Packs。已接入 list / get / create / update / delete 接口。</p>
      </div>
      <div class="flex gap-3">
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" :disabled="loading" @click="load">
          <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
          <RefreshCw v-else class="h-4 w-4" />
          刷新
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl bg-blue-700 px-4 py-3 text-sm font-bold text-white shadow-sm" type="button" @click="startCreate">
          <Plus class="h-4 w-4" />
          新增资源包
        </button>
      </div>
    </header>

    <div class="grid gap-6 xl:grid-cols-[0.92fr_1.08fr]">
      <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="flex items-center justify-between border-b border-slate-200 p-5">
          <div>
            <h2 class="text-xl font-black">资源包列表</h2>
            <p class="mt-1 text-sm text-slate-500">左侧选择资源包，右侧查看详情并编辑。</p>
          </div>
          <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-bold text-slate-500">{{ packs.length }} 条</span>
        </div>

        <div v-if="loading" class="p-12 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          正在加载...
        </div>
        <div v-else-if="!packs.length" class="p-12 text-center text-slate-500">暂无资源包</div>
        <button
          v-for="pack in packs"
          v-else
          :key="packId(pack)"
          class="grid w-full grid-cols-[1fr_auto] gap-4 border-b border-slate-100 px-5 py-4 text-left last:border-b-0 hover:bg-sky-50"
          :class="packId(selected) === packId(pack) ? 'bg-sky-50' : ''"
          type="button"
          @click="selectPack(pack)"
        >
          <div class="min-w-0">
            <div class="truncate text-lg font-black text-slate-950">{{ packTitle(pack) }}</div>
            <div class="mt-1 line-clamp-2 text-sm text-slate-500">{{ pack.description || "-" }}</div>
            <div class="mt-2 flex flex-wrap gap-2 text-xs font-bold text-slate-500">
              <span class="rounded-full bg-slate-100 px-2 py-1">ID：{{ pack.pack_id }}</span>
              <span class="rounded-full bg-slate-100 px-2 py-1">Version：{{ pack.version || 0 }}</span>
            </div>
          </div>
          <span class="h-fit rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1 text-xs font-black text-emerald-700">
            {{ pack.status || "Active" }}
          </span>
        </button>

        <div class="flex justify-end gap-3 border-t border-slate-200 p-5">
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrevious || loading" @click="previousPage">上一页</button>
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext || loading" @click="nextPage">下一页</button>
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <div class="mb-5 flex items-start justify-between gap-4">
          <div>
            <h2 class="text-xl font-black">{{ mode === "create" ? "新增资源包" : "资源包详情" }}</h2>
            <p class="mt-1 text-sm text-slate-500">不能修改的系统字段在下方完整字段里只读展示。</p>
            <p v-if="detailLoading" class="mt-1 inline-flex items-center gap-2 text-xs font-bold text-blue-600">
              <Loader2 class="h-3.5 w-3.5 animate-spin" />
              正在加载 get 详情...
            </p>
          </div>
          <button
            v-if="mode === 'edit'"
            class="inline-flex items-center gap-2 rounded-xl border border-red-200 bg-red-50 px-4 py-2 text-sm font-bold text-red-700"
            type="button"
            :disabled="saving"
            @click="deletePack"
          >
            <Trash2 class="h-4 w-4" />
            删除
          </button>
        </div>

        <div class="grid gap-4 md:grid-cols-2">
          <label class="text-sm font-bold">
            Pack ID
            <input v-model="form.pack_id" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2 disabled:bg-slate-100" :disabled="mode === 'edit'" placeholder="留空则由后台生成" />
          </label>
          <label class="text-sm font-bold">
            标题
            <input v-model="form.title" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2" placeholder="资源包标题" />
          </label>
          <label class="md:col-span-2 text-sm font-bold">
            描述
            <textarea v-model="form.description" class="mt-2 min-h-24 w-full rounded-xl border border-slate-200 px-3 py-2" placeholder="资源包描述" />
          </label>
          <label class="text-sm font-bold">
            Respath
            <input v-model="form.respath" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2" placeholder="/res-packages/..." />
          </label>
          <label class="text-sm font-bold">
            状态
            <input v-model="form.status" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2" :disabled="mode === 'create'" placeholder="Active" />
          </label>
          <label class="text-sm font-bold">
            封面 Object Key
            <input v-model="form.thumbnail_object_key" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2" placeholder="resource-packs/.../thumbnail.jpg" />
          </label>
          <label class="text-sm font-bold">
            封面 File Hash
            <input v-model="form.thumbnail_file_hash" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2" placeholder="SHA256 Hash" />
          </label>
          <label class="text-sm font-bold">
            Icon
            <input v-model="form.icon" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2" placeholder="FileBarChart" />
          </label>
          <label class="text-sm font-bold">
            Category
            <input v-model="form.category" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2" placeholder="public / member / ..." />
          </label>
          <label class="text-sm font-bold">
            Version
            <input v-model.number="form.version" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2 disabled:bg-slate-100" type="number" min="0" :disabled="mode === 'create'" />
          </label>
        </div>

        <button class="mt-5 inline-flex w-full items-center justify-center gap-2 rounded-xl bg-blue-700 px-4 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving" @click="savePack">
          <Loader2 v-if="saving" class="h-4 w-4 animate-spin" />
          <Save v-else class="h-4 w-4" />
          {{ mode === "create" ? "创建资源包" : "保存资源包" }}
        </button>

        <div v-if="selected" class="mt-6 rounded-2xl border border-slate-200 bg-slate-50 p-4">
          <div class="mb-3 flex items-center gap-2 text-sm font-black">
            <FileBox class="h-4 w-4 text-blue-700" />
            完整字段
          </div>
          <div class="grid gap-3 md:grid-cols-2">
            <div v-for="[key, value] in selectedEntries" :key="key" class="rounded-xl bg-white p-3">
              <div class="text-[11px] font-black uppercase text-slate-400">{{ humanizeKey(key) }}</div>
              <div v-if="typeof value === 'string' && key.endsWith('_at')" class="mt-1 break-words text-sm font-semibold">{{ formatDate(value) }}</div>
              <div v-else class="mt-1 break-words text-sm font-semibold">{{ value ?? "-" }}</div>
            </div>
          </div>
        </div>
      </section>
    </div>
  </section>
</template>
