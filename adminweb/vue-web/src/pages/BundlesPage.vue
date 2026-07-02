<script setup lang="ts">
import { FileJson, Loader2, Plus, RefreshCw, Save, Send, Trash2, X } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { badgeClass, pickFirst } from "@/lib/status"

type BundleForm = {
  bundle_ulid: string
  bundle_gpath: string
  name: string
  description: string
  items_json: string
  pricing_json: string
  thumbnail_object_key: string
  thumbnail_file_hash: string
}

type DetailTab = "summary" | "meta" | "pricing" | "schema" | "actions" | "raw"
type Mode = "detail" | "create"
type SummaryField = {
  label: string
  value: string
}

const emptyForm: BundleForm = {
  bundle_ulid: "",
  bundle_gpath: "",
  name: "",
  description: "",
  items_json: "[]",
  pricing_json: "{}",
  thumbnail_object_key: "",
  thumbnail_file_hash: "",
}

const bundles = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const form = ref<BundleForm>({ ...emptyForm })
const loading = ref(false)
const saving = ref(false)
const publishing = ref(false)
const deprecating = ref(false)
const deleting = ref(false)
const detailOpen = ref(false)
const statusFilter = ref("")
const offset = ref(0)
const schemas = ref<JsonRecord | null>(null)
const activeTab = ref<DetailTab>("summary")
const mode = ref<Mode>("detail")
const showDeleteConfirm = ref(false)
const limit = 20

const canPrev = computed(() => offset.value > 0)
const canNext = computed(() => bundles.value.length >= limit)
const statusActionBusy = computed(() => publishing.value || deprecating.value || deleting.value)
const selectedId = computed(() => selected.value ? bundleUlid(selected.value) : "")
const selectedFields = computed(() => selected.value || {})
const detailTabs = computed(() => [
  { key: "summary" as const, title: "概览", count: selected.value ? 1 : 0 },
  { key: "meta" as const, title: "基础信息", count: 1 },
  { key: "pricing" as const, title: "结构与价格", count: 2 },
  { key: "schema" as const, title: "Schema", count: schemas.value ? 1 : 0 },
  { key: "actions" as const, title: "状态操作", count: 3 },
  { key: "raw" as const, title: "完整字段", count: 1 },
])
const summaryFields = computed<SummaryField[]>(() => {
  const bundle = selected.value
  if (!bundle) return []
  return [
    { label: "展示价格", value: displayPrice(bundle) },
    { label: "状态", value: String(bundleStatus(bundle) || "-") },
    { label: "版本", value: String(bundle.version ?? "-") },
    { label: "Bundle ULID", value: bundleUlid(bundle) || "-" },
    { label: "Bundle GPath", value: String(bundle.bundle_gpath || "-") },
    { label: "名称", value: bundleName(bundle) },
    { label: "封面 Object Key", value: String(bundle.thumbnail_object_key || "-") },
    { label: "更新时间", value: formatDate(String(pickFirst(bundle, ["updated_at", "updatedAt"]) || "")) || "-" },
  ]
})

function bundleUlid(bundle: JsonRecord | null | undefined) {
  return String(pickFirst(bundle || {}, ["bundle_ulid", "bundle_id"]) || "")
}

function bundleName(bundle: JsonRecord | null | undefined) {
  return String(pickFirst(bundle || {}, ["name", "title"]) || "未命名商品")
}

function bundleStatus(bundle: JsonRecord | null | undefined) {
  return pickFirst(bundle || {}, ["status", "raw_status"])
}

function displayPrice(bundle: JsonRecord | null | undefined) {
  const currency = String(bundle?.display_currency || "")
  const min = Number(bundle?.display_amount_min || 0) / 100
  const max = Number(bundle?.display_amount_max || 0) / 100
  if (!currency || (!min && !max)) return "-"
  if (min === max) return `${currency} ${min.toFixed(2)}`
  return `${currency} ${min.toFixed(2)} - ${max.toFixed(2)}`
}

function parseJson(value: string, field: string) {
  try {
    return JSON.parse(value || "")
  } catch {
    toast.error(`${field} 不是合法 JSON`)
    return null
  }
}

function formFromBundle(bundle: JsonRecord | null): BundleForm {
  if (!bundle) return { ...emptyForm }
  return {
    bundle_ulid: String(bundle.bundle_ulid || bundle.bundle_id || ""),
    bundle_gpath: String(bundle.bundle_gpath || ""),
    name: String(bundle.name || ""),
    description: String(bundle.description || ""),
    items_json: String(bundle.items_json || "[]"),
    pricing_json: String(bundle.pricing_json || "{}"),
    thumbnail_object_key: String(bundle.thumbnail_object_key || ""),
    thumbnail_file_hash: String(bundle.thumbnail_file_hash || ""),
  }
}

async function load() {
  loading.value = true
  try {
    const params = new URLSearchParams({
      limit: String(limit),
      offset: String(offset.value),
    })
    if (statusFilter.value) params.set("status", statusFilter.value)
    const data = await apiClient<JsonRecord>(`/api/mall/bundles?${params}`)
    const list = Array.isArray(data.bundles) ? data.bundles : []
    bundles.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    if (!selected.value && bundles.value.length) {
      await selectBundle(bundles.value[0], false)
    }
  } catch (err) {
    console.error(err)
    bundles.value = []
    toast.error("商品配置加载失败")
  } finally {
    loading.value = false
  }
}

async function selectBundle(bundle: JsonRecord, open = true) {
  const id = bundleUlid(bundle)
  selected.value = bundle
  detailOpen.value = open
  mode.value = "detail"
  activeTab.value = "summary"
  showDeleteConfirm.value = false
  form.value = formFromBundle(bundle)
  if (!id) return
  try {
    const detail = await apiClient<JsonRecord>(`/api/mall/bundles/${encodeURIComponent(id)}`)
    const actualBundle = (detail.bundle && typeof detail.bundle === "object" ? detail.bundle : detail) as JsonRecord
    selected.value = actualBundle
    form.value = formFromBundle(actualBundle)
  } catch {
    form.value = formFromBundle(bundle)
  }
}

function newBundle() {
  selected.value = null
  detailOpen.value = true
  mode.value = "create"
  activeTab.value = "meta"
  showDeleteConfirm.value = false
  form.value = { ...emptyForm }
}

function closeDetail() {
  detailOpen.value = false
  if (mode.value === "create") mode.value = "detail"
}

async function createBundle() {
  if (!form.value.bundle_ulid.trim() || !form.value.bundle_gpath.trim() || !form.value.name.trim()) {
    toast.error("请填写 bundle_ulid、bundle_gpath 和名称")
    return
  }
  if (parseJson(form.value.items_json, "items_json") === null || parseJson(form.value.pricing_json, "pricing_json") === null) {
    return
  }

  saving.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/mall/bundles", {
      method: "POST",
      body: JSON.stringify({
        bundle_ulid: form.value.bundle_ulid.trim(),
        bundle_gpath: form.value.bundle_gpath.trim(),
        name: form.value.name.trim(),
        description: form.value.description.trim(),
        items_json: form.value.items_json.trim(),
        pricing_json: form.value.pricing_json.trim(),
        thumbnail_object_key: form.value.thumbnail_object_key.trim(),
        thumbnail_file_hash: form.value.thumbnail_file_hash.trim(),
      }),
    })
    toast.success("商品草稿已创建")
    await load()
    const id = String(data.bundle_ulid || form.value.bundle_ulid)
    const created = bundles.value.find((item) => bundleUlid(item) === id)
    if (created) await selectBundle(created)
  } catch (err) {
    console.error(err)
    toast.error("创建失败")
  } finally {
    saving.value = false
  }
}

async function saveMeta() {
  if (!selectedId.value) return
  saving.value = true
  try {
    await apiClient(`/api/mall/bundles/${encodeURIComponent(selectedId.value)}/meta`, {
      method: "PUT",
      body: JSON.stringify({
        name: form.value.name.trim(),
        description: form.value.description.trim(),
        thumbnail_object_key: form.value.thumbnail_object_key.trim(),
        thumbnail_file_hash: form.value.thumbnail_file_hash.trim(),
      }),
    })
    toast.success("基础信息已保存")
    await load()
  } catch (err) {
    console.error(err)
    toast.error("保存失败")
  } finally {
    saving.value = false
  }
}

async function savePricing() {
  if (!selectedId.value) return
  if (parseJson(form.value.items_json, "items_json") === null || parseJson(form.value.pricing_json, "pricing_json") === null) {
    return
  }
  saving.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/mall/bundles/pricing", {
      method: "PUT",
      body: JSON.stringify({
        bundle_ulid: selectedId.value,
        items_json: form.value.items_json.trim(),
        pricing_json: form.value.pricing_json.trim(),
      }),
    })
    toast.success("商品结构与价格已保存")
    const actualBundle = (data.bundle && typeof data.bundle === "object" ? data.bundle : data) as JsonRecord
    selected.value = actualBundle
    form.value = formFromBundle(actualBundle)
    await load()
  } catch (err) {
    console.error(err)
    toast.error("保存失败")
  } finally {
    saving.value = false
  }
}

async function publish() {
  if (!selectedId.value) return
  if (statusActionBusy.value) return
  publishing.value = true
  try {
    await apiClient(`/api/mall/bundles/${encodeURIComponent(selectedId.value)}/publish`, { method: "POST" })
    toast.success("商品已发布")
    await load()
  } catch (err) {
    console.error(err)
    toast.error("发布失败")
  } finally {
    publishing.value = false
  }
}

async function deprecate() {
  if (!selectedId.value) return
  if (statusActionBusy.value) return
  deprecating.value = true
  try {
    await apiClient(`/api/mall/bundles/${encodeURIComponent(selectedId.value)}/deprecate`, { method: "POST" })
    toast.success("商品已下架")
    await load()
  } catch (err) {
    console.error(err)
    toast.error("下架失败")
  } finally {
    deprecating.value = false
  }
}

async function removeBundle() {
  if (!selectedId.value) return
  if (statusActionBusy.value) return
  deleting.value = true
  try {
    await apiClient(`/api/mall/bundles/${encodeURIComponent(selectedId.value)}`, { method: "DELETE" })
    toast.success("商品已删除")
    selected.value = null
    detailOpen.value = false
    form.value = { ...emptyForm }
    showDeleteConfirm.value = false
    await load()
  } catch (err) {
    console.error(err)
    toast.error("删除失败")
  } finally {
    deleting.value = false
  }
}

async function syncDisplayPricing() {
  await apiClient("/api/mall/bundles/sync-display-pricing", {
    method: "POST",
    body: JSON.stringify({ bundle_ulid: selectedId.value || undefined }),
  })
  toast.success("展示价格已同步")
  await load()
}

async function loadSchemas() {
  schemas.value = await apiClient<JsonRecord>("/api/mall/bundles/schemas")
  activeTab.value = "schema"
}

watch([statusFilter, offset], () => {
  selected.value = null
  void load()
})
onMounted(load)
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1580px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">商品配置</h1>
        <p class="mt-2 text-slate-600">配置 gmall Bundle：认证商品、管线引用、价格 JSON、封面和发布状态。</p>
        <p class="mt-2 text-xs font-semibold text-slate-500">
          已确认接口：list/get/create/update meta/update pricing/publish/deprecate/delete/schema/sync display pricing/upload-url。
        </p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-4 py-3 text-sm font-bold text-white shadow-sm" type="button" @click="newBundle">
          <Plus class="h-4 w-4" />
          新建商品
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="syncDisplayPricing">
          <RefreshCw class="h-4 w-4" />
          同步展示价格
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load">
          <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
          刷新
        </button>
      </div>
    </header>

    <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 p-5">
          <div class="flex items-center gap-3">
            <div>
              <h2 class="text-xl font-black">商品列表</h2>
              <p class="mt-1 text-sm text-slate-500">来自 `/api/mall/bundles`，点击行或按钮查看详情。</p>
            </div>
            <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ bundles.length }}</span>
          </div>
          <select v-model="statusFilter" class="h-10 w-full rounded-xl border border-slate-200 px-4 text-sm md:w-64">
            <option value="">全部状态</option>
            <option value="Draft">Draft</option>
            <option value="Active">Active</option>
            <option value="Deprecated">Deprecated</option>
          </select>
        </div>
        <div v-if="loading" class="p-12 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          正在加载...
        </div>
        <template v-else>
          <div class="grid grid-cols-[minmax(0,1fr)_160px_110px_170px_112px] gap-4 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-500">
            <span>商品</span>
            <span class="text-center">状态</span>
            <span class="text-center">版本</span>
            <span class="text-right">更新时间</span>
            <span class="text-right">操作</span>
          </div>
          <div
            v-for="bundle in bundles"
            :key="bundleUlid(bundle)"
            class="grid w-full cursor-pointer grid-cols-[minmax(0,1fr)_160px_110px_170px_112px] gap-4 border-b border-slate-100 px-5 py-4 text-left transition last:border-b-0 hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-200"
            :class="mode === 'detail' && selectedId === bundleUlid(bundle) ? 'bg-sky-50' : ''"
            role="button"
            tabindex="0"
            @click="selectBundle(bundle)"
            @keydown.enter.prevent="selectBundle(bundle)"
            @keydown.space.prevent="selectBundle(bundle)"
          >
            <div class="min-w-0">
              <div class="truncate text-lg font-black">{{ bundleName(bundle) }}</div>
              <div class="mt-1 line-clamp-1 text-sm text-slate-500">{{ bundle.description || "暂无描述" }}</div>
              <div class="mt-2 flex flex-wrap gap-2 text-xs font-semibold text-slate-500">
                <span class="rounded-full bg-slate-100 px-2 py-1">展示价格：{{ displayPrice(bundle) }}</span>
                <span class="rounded-full bg-slate-100 px-2 py-1">ID: {{ bundleUlid(bundle) || "-" }}</span>
              </div>
            </div>
            <span class="self-center justify-self-center rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(bundleStatus(bundle))">{{ bundleStatus(bundle) || "-" }}</span>
            <span class="self-center text-center text-sm font-black text-slate-700">v{{ bundle.version || 0 }}</span>
            <span class="self-center justify-self-end text-sm font-semibold text-slate-500">{{ formatDate(String(bundle.updated_at || bundle.created_at || "")) }}</span>
            <button class="inline-flex h-9 items-center justify-self-end rounded-xl border border-slate-200 bg-white px-3 text-sm font-bold text-blue-700 shadow-sm transition hover:border-blue-200 hover:bg-blue-50" type="button" @click.stop="selectBundle(bundle)">
              查看详情
            </button>
          </div>
        </template>
        <div v-if="!loading && !bundles.length" class="p-12 text-center text-slate-500">暂无商品</div>
        <div class="flex justify-end gap-3 border-t border-slate-200 p-5">
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrev" @click="offset = Math.max(0, offset - limit)">上一页</button>
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext" @click="offset += limit">下一页</button>
        </div>
    </section>

    <Teleport to="body">
      <div v-if="detailOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/45 p-6">
        <section class="flex max-h-[88vh] w-full max-w-[1320px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
        <template v-if="mode === 'create'">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 p-5">
            <div>
              <h2 class="text-2xl font-black">新建商品</h2>
              <p class="mt-1 text-sm text-slate-500">创建草稿需要 bundle_ulid、bundle_gpath、名称，以及合法 JSON。</p>
            </div>
            <button class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900" type="button" aria-label="关闭" @click="closeDetail">
              <X class="h-5 w-5" />
            </button>
          </div>
          <div class="space-y-5 overflow-y-auto p-5">
            <div class="grid gap-4 md:grid-cols-2">
              <label class="grid gap-2 text-sm font-bold">
                Bundle ULID
                <input v-model="form.bundle_ulid" class="rounded-xl border border-slate-200 px-4 py-3" />
              </label>
              <label class="grid gap-2 text-sm font-bold">
                Bundle GPath
                <input v-model="form.bundle_gpath" class="rounded-xl border border-slate-200 px-4 py-3" />
              </label>
              <label class="grid gap-2 text-sm font-bold">
                名称
                <input v-model="form.name" class="rounded-xl border border-slate-200 px-4 py-3" maxlength="160" />
              </label>
              <label class="grid gap-2 text-sm font-bold">
                封面 Object Key
                <input v-model="form.thumbnail_object_key" class="rounded-xl border border-slate-200 px-4 py-3" />
              </label>
              <label class="grid gap-2 text-sm font-bold md:col-span-2">
                描述
                <textarea v-model="form.description" class="min-h-24 rounded-xl border border-slate-200 p-4" maxlength="1200" />
              </label>
              <label class="grid gap-2 text-sm font-bold md:col-span-2">
                Thumbnail File Hash
                <input v-model="form.thumbnail_file_hash" class="rounded-xl border border-slate-200 px-4 py-3" />
              </label>
            </div>
            <div class="grid gap-4 xl:grid-cols-2">
              <label class="grid gap-2 text-sm font-bold">
                items_json
                <textarea v-model="form.items_json" class="min-h-[260px] rounded-xl border border-slate-200 p-4 font-mono text-xs leading-6" />
              </label>
              <label class="grid gap-2 text-sm font-bold">
                pricing_json
                <textarea v-model="form.pricing_json" class="min-h-[260px] rounded-xl border border-slate-200 p-4 font-mono text-xs leading-6" />
              </label>
            </div>
            <div class="flex justify-end">
              <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving" @click="createBundle">
                <Plus class="h-4 w-4" />
                创建草稿
              </button>
            </div>
          </div>
        </template>

        <div v-else-if="!selected" class="flex items-start justify-between gap-4 p-6">
          <div>
            <h2 class="text-2xl font-black">商品详情</h2>
            <p class="mt-1 text-sm text-slate-500">请选择一个商品，或点击新建商品。</p>
          </div>
          <button class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900" type="button" aria-label="关闭" @click="closeDetail">
            <X class="h-5 w-5" />
          </button>
        </div>

        <template v-else>
          <div class="border-b border-slate-200 p-5">
            <div class="flex flex-wrap items-start justify-between gap-4">
              <div>
                <h2 class="text-2xl font-black">{{ bundleName(selected) }}</h2>
                <p class="mt-1 break-all text-sm text-slate-500">{{ selectedId }}</p>
              </div>
              <div class="flex items-center gap-3">
                <span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(bundleStatus(selected))">{{ bundleStatus(selected) || "-" }}</span>
                <button class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900" type="button" aria-label="关闭" @click="closeDetail">
                  <X class="h-5 w-5" />
                </button>
              </div>
            </div>
          </div>

          <div class="grid min-h-0 flex-1 overflow-hidden lg:grid-cols-[260px_minmax(0,1fr)]">
            <aside class="border-b border-slate-200 p-4 lg:border-b-0 lg:border-r">
              <div class="space-y-2">
                <button
                  v-for="tab in detailTabs"
                  :key="tab.key"
                  class="w-full rounded-2xl border px-4 py-3 text-left"
                  :class="activeTab === tab.key ? 'border-sky-200 bg-sky-50' : 'border-slate-100 hover:bg-slate-50'"
                  type="button"
                  @click="activeTab = tab.key"
                >
                  <div class="flex items-center justify-between gap-3">
                    <span class="font-black">{{ tab.title }}</span>
                    <span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-black text-slate-600">{{ tab.count }}</span>
                  </div>
                </button>
              </div>
            </aside>

            <main class="min-w-0 overflow-y-auto p-5">
              <div v-if="activeTab === 'summary'" class="space-y-5">
                <div class="grid gap-4 md:grid-cols-2">
                  <div v-for="field in summaryFields" :key="field.label" class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                    <div class="text-xs font-black uppercase text-slate-400">{{ field.label }}</div>
                    <div class="mt-2 break-all text-sm font-black text-slate-800">{{ field.value }}</div>
                  </div>
                </div>
                <div class="rounded-2xl border border-slate-200 bg-white p-4">
                  <div class="text-xs font-black uppercase text-slate-400">描述</div>
                  <p class="mt-2 whitespace-pre-wrap text-sm font-semibold leading-6 text-slate-700">{{ form.description || "-" }}</p>
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <label v-for="(value, key) in selectedFields" :key="key" class="grid gap-2 text-sm font-bold">
                    {{ key }}
                    <textarea
                      v-if="Array.isArray(value) || (value && typeof value === 'object')"
                      class="min-h-24 rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-600"
                      disabled
                      :value="JSON.stringify(value, null, 2)"
                    />
                    <input v-else class="rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-600" disabled :value="String(value ?? '-')" />
                  </label>
                </div>
              </div>

              <div v-else-if="activeTab === 'meta'" class="space-y-5">
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="grid gap-2 text-sm font-bold">
                    Bundle ULID
                    <input v-model="form.bundle_ulid" disabled class="rounded-xl border border-slate-200 bg-slate-100 px-4 py-3" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    Bundle GPath
                    <input v-model="form.bundle_gpath" disabled class="rounded-xl border border-slate-200 bg-slate-100 px-4 py-3" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    名称
                    <input v-model="form.name" class="rounded-xl border border-slate-200 px-4 py-3" maxlength="160" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    封面 Object Key
                    <input v-model="form.thumbnail_object_key" class="rounded-xl border border-slate-200 px-4 py-3" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold md:col-span-2">
                    描述
                    <textarea v-model="form.description" class="min-h-28 rounded-xl border border-slate-200 p-4" maxlength="1200" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold md:col-span-2">
                    Thumbnail File Hash
                    <input v-model="form.thumbnail_file_hash" class="rounded-xl border border-slate-200 px-4 py-3" />
                  </label>
                </div>
                <div class="flex justify-end">
                  <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving" @click="saveMeta">
                    <Save class="h-4 w-4" />
                    保存基础信息
                  </button>
                </div>
              </div>

              <div v-else-if="activeTab === 'pricing'" class="space-y-5">
                <div class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
                  items_json 和 pricing_json 会直接提交到 gmall；保存前会先校验 JSON 格式。
                </div>
                <div class="grid gap-4 xl:grid-cols-2">
                  <label class="grid gap-2 text-sm font-bold">
                    items_json
                    <textarea v-model="form.items_json" class="min-h-[420px] rounded-xl border border-slate-200 p-4 font-mono text-xs leading-6" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    pricing_json
                    <textarea v-model="form.pricing_json" class="min-h-[420px] rounded-xl border border-slate-200 p-4 font-mono text-xs leading-6" />
                  </label>
                </div>
                <div class="flex justify-end">
                  <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving" @click="savePricing">
                    <Send class="h-4 w-4" />
                    保存结构与价格
                  </button>
                </div>
              </div>

              <div v-else-if="activeTab === 'schema'" class="space-y-4">
                <button class="inline-flex items-center gap-2 rounded-xl border px-4 py-2 text-sm font-bold" type="button" @click="loadSchemas">
                  <FileJson class="h-4 w-4" />
                  加载 Schema
                </button>
                <pre v-if="schemas" class="max-h-[620px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(schemas, null, 2) }}</pre>
                <div v-else class="rounded-2xl border border-dashed border-slate-200 p-10 text-center text-slate-500">暂无 Schema，点击加载。</div>
              </div>

              <div v-else-if="activeTab === 'actions'" class="space-y-5">
                <div class="rounded-2xl border border-slate-200 bg-slate-50 p-5">
                  <h3 class="font-black">状态操作</h3>
                  <p class="mt-1 text-sm text-slate-500">使用 gmall 发布/下架/删除接口。</p>
                  <div class="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                    <button class="inline-flex h-11 items-center justify-center gap-2 rounded-xl border bg-white px-4 text-sm font-bold shadow-sm transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60" type="button" :disabled="statusActionBusy" @click="publish">
                      <Loader2 v-if="publishing" class="h-4 w-4 animate-spin" />
                      {{ publishing ? "发布中..." : "发布" }}
                    </button>
                    <button class="inline-flex h-11 items-center justify-center gap-2 rounded-xl border bg-white px-4 text-sm font-bold shadow-sm transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60" type="button" :disabled="statusActionBusy" @click="deprecate">
                      <Loader2 v-if="deprecating" class="h-4 w-4 animate-spin" />
                      {{ deprecating ? "下架中..." : "下架" }}
                    </button>
                    <button class="inline-flex h-11 items-center justify-center gap-2 rounded-xl bg-red-600 px-4 text-sm font-bold text-white shadow-sm shadow-red-200 transition hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-60" type="button" :disabled="statusActionBusy" @click="showDeleteConfirm = true">
                      <Loader2 v-if="deleting" class="h-4 w-4 animate-spin" />
                      <Trash2 v-else class="h-4 w-4" />
                      {{ deleting ? "删除中..." : "删除" }}
                    </button>
                  </div>
                </div>
              </div>

              <div v-else-if="activeTab === 'raw'" class="space-y-4">
                <div class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
                  完整字段只读展示，避免手工修改不可编辑字段。
                </div>
                <pre class="max-h-[620px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selected, null, 2) }}</pre>
              </div>
            </main>
          </div>
        </template>
        </section>
      </div>
    </Teleport>

    <div v-if="showDeleteConfirm" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-6">
      <div class="w-full max-w-md rounded-3xl bg-white p-6 shadow-2xl">
        <h2 class="text-2xl font-black">确认删除商品</h2>
        <p class="mt-3 text-sm text-slate-600">删除会调用 gmall delete 接口。已发布商品如果微服务不允许删除，会返回错误。</p>
        <div class="mt-5 rounded-2xl bg-slate-50 p-4">
          <div class="font-black">{{ bundleName(selected) }}</div>
          <div class="mt-1 break-all text-xs text-slate-500">{{ selectedId }}</div>
        </div>
        <div class="mt-6 flex justify-end gap-3">
          <button class="rounded-xl border px-5 py-3 font-bold disabled:cursor-not-allowed disabled:opacity-60" type="button" :disabled="deleting" @click="showDeleteConfirm = false">取消</button>
          <button class="inline-flex items-center justify-center gap-2 rounded-xl bg-red-600 px-5 py-3 font-bold text-white disabled:cursor-not-allowed disabled:opacity-60" type="button" :disabled="deleting" @click="removeBundle">
            <Loader2 v-if="deleting" class="h-4 w-4 animate-spin" />
            {{ deleting ? "删除中..." : "确认删除" }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
