<script setup lang="ts">
import { ArrowLeft, FileJson, Loader2, Plus, RefreshCw, Save, Send } from "lucide-vue-next"
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
const creating = ref(false)
const statusFilter = ref("")
const offset = ref(0)
const schemas = ref<JsonRecord | null>(null)
const limit = 20

const canPrev = computed(() => offset.value > 0)
const canNext = computed(() => bundles.value.length >= limit)
const inEditor = computed(() => !!selected.value || creating.value)
const selectedId = computed(() => selected.value ? bundleUlid(selected.value) : "")

function bundleUlid(bundle: JsonRecord) {
  return String(pickFirst(bundle, ["bundle_ulid", "bundle_id"]) || "")
}

function bundleName(bundle: JsonRecord) {
  return String(pickFirst(bundle, ["name", "title"]) || "未命名商品")
}

function bundleStatus(bundle: JsonRecord) {
  return pickFirst(bundle, ["status", "raw_status"])
}

function displayPrice(bundle: JsonRecord) {
  const currency = String(bundle.display_currency || "")
  const min = Number(bundle.display_amount_min || 0) / 100
  const max = Number(bundle.display_amount_max || 0) / 100
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
  } catch (err) {
    console.error(err)
    bundles.value = []
    toast.error("商品配置加载失败")
  } finally {
    loading.value = false
  }
}

async function selectBundle(bundle: JsonRecord) {
  const id = bundleUlid(bundle)
  selected.value = bundle
  creating.value = false
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
  creating.value = true
  form.value = { ...emptyForm }
}

function back() {
  selected.value = null
  creating.value = false
  form.value = { ...emptyForm }
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
    creating.value = false
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
  await apiClient(`/api/mall/bundles/${encodeURIComponent(selectedId.value)}/publish`, { method: "POST" })
  toast.success("商品已发布")
  await load()
}

async function deprecate() {
  if (!selectedId.value) return
  await apiClient(`/api/mall/bundles/${encodeURIComponent(selectedId.value)}/deprecate`, { method: "POST" })
  toast.success("商品已下架")
  await load()
}

async function removeBundle() {
  if (!selectedId.value) return
  if (!window.confirm("确认删除这个商品草稿？已发布商品通常不能删除。")) return
  await apiClient(`/api/mall/bundles/${encodeURIComponent(selectedId.value)}`, { method: "DELETE" })
  toast.success("商品已删除")
  back()
  await load()
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
}

watch([statusFilter, offset], () => load())
onMounted(load)
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">商品配置</h1>
        <p class="mt-2 text-slate-600">配置 gmall Bundle：认证商品、管线引用、价格 JSON、封面和发布状态。</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button v-if="inEditor" class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="back">
          <ArrowLeft class="h-4 w-4" />
          返回列表
        </button>
        <button v-else class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-4 py-3 text-sm font-bold text-white shadow-sm" type="button" @click="newBundle">
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

    <section v-if="!inEditor" class="rounded-3xl border border-slate-200 bg-white shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 p-5">
        <div>
          <h2 class="text-xl font-black">商品列表</h2>
          <p class="mt-1 text-sm text-slate-500">列表隐藏 ULID，进入详情后可查看完整配置。</p>
        </div>
        <select v-model="statusFilter" class="rounded-xl border border-slate-200 px-4 py-3">
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
      <button
        v-for="bundle in bundles"
        v-else
        :key="bundleUlid(bundle)"
        class="grid w-full grid-cols-[1fr_auto] gap-4 border-b border-slate-100 px-5 py-5 text-left last:border-b-0 hover:bg-sky-50"
        type="button"
        @click="selectBundle(bundle)"
      >
        <div>
          <div class="text-lg font-black">{{ bundleName(bundle) }}</div>
          <div class="mt-1 line-clamp-2 text-sm text-slate-500">{{ bundle.description || "暂无描述" }}</div>
          <div class="mt-2 text-sm font-bold text-slate-700">展示价格：{{ displayPrice(bundle) }}</div>
        </div>
        <div class="text-right">
          <span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(bundleStatus(bundle))">{{ bundleStatus(bundle) || "-" }}</span>
          <div class="mt-3 text-xs text-slate-500">v{{ bundle.version || 0 }}</div>
          <div class="mt-1 text-xs text-slate-400">{{ formatDate(String(bundle.updated_at || bundle.created_at || "")) }}</div>
        </div>
      </button>
      <div v-if="!loading && !bundles.length" class="p-12 text-center text-slate-500">暂无商品</div>
      <div class="flex justify-end gap-3 border-t border-slate-200 p-5">
        <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrev" @click="offset = Math.max(0, offset - limit)">上一页</button>
        <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext" @click="offset += limit">下一页</button>
      </div>
    </section>

    <section v-else class="grid gap-6">
      <div class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <div class="mb-5 flex flex-wrap items-center justify-between gap-4">
          <div>
            <h2 class="text-2xl font-black">{{ creating ? "新建商品" : form.name || "商品详情" }}</h2>
            <p class="mt-1 text-sm text-slate-500">Bundle 基础信息和价格结构。</p>
          </div>
          <div v-if="!creating" class="flex flex-wrap gap-2">
            <button class="rounded-xl border px-4 py-2 font-bold" type="button" @click="publish">发布</button>
            <button class="rounded-xl border px-4 py-2 font-bold" type="button" @click="deprecate">下架</button>
            <button class="rounded-xl bg-red-600 px-4 py-2 font-bold text-white" type="button" @click="removeBundle">删除</button>
          </div>
        </div>

        <div class="grid gap-4 md:grid-cols-2">
          <label class="grid gap-2 text-sm font-bold">
            Bundle ULID
            <input v-model="form.bundle_ulid" :disabled="!creating" class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-50" />
          </label>
          <label class="grid gap-2 text-sm font-bold">
            Bundle GPath
            <input v-model="form.bundle_gpath" :disabled="!creating" class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-50" />
          </label>
          <label class="grid gap-2 text-sm font-bold">
            名称
            <input v-model="form.name" class="rounded-xl border border-slate-200 px-4 py-3" />
          </label>
          <label class="grid gap-2 text-sm font-bold">
            封面 Object Key
            <input v-model="form.thumbnail_object_key" class="rounded-xl border border-slate-200 px-4 py-3" />
          </label>
          <label class="grid gap-2 text-sm font-bold md:col-span-2">
            描述
            <textarea v-model="form.description" class="min-h-24 rounded-xl border border-slate-200 p-4" />
          </label>
          <label class="grid gap-2 text-sm font-bold md:col-span-2">
            Thumbnail File Hash
            <input v-model="form.thumbnail_file_hash" class="rounded-xl border border-slate-200 px-4 py-3" />
          </label>
        </div>

        <div class="mt-5 flex justify-end gap-3">
          <button v-if="creating" class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving" @click="createBundle">
            <Plus class="h-4 w-4" />
            创建草稿
          </button>
          <button v-else class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving" @click="saveMeta">
            <Save class="h-4 w-4" />
            保存基础信息
          </button>
        </div>
      </div>

      <div class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <div class="mb-4 flex flex-wrap items-center justify-between gap-4">
          <div>
            <h3 class="text-xl font-black">商品结构与价格 JSON</h3>
            <p class="mt-1 text-sm text-slate-500">items_json 和 pricing_json 会直接提交到 gmall。</p>
          </div>
          <button class="inline-flex items-center gap-2 rounded-xl border px-4 py-2 text-sm font-bold" type="button" @click="loadSchemas">
            <FileJson class="h-4 w-4" />
            查看 Schema
          </button>
        </div>
        <div class="grid gap-4 xl:grid-cols-2">
          <label class="grid gap-2 text-sm font-bold">
            items_json
            <textarea v-model="form.items_json" class="min-h-[360px] rounded-xl border border-slate-200 p-4 font-mono text-xs leading-6" />
          </label>
          <label class="grid gap-2 text-sm font-bold">
            pricing_json
            <textarea v-model="form.pricing_json" class="min-h-[360px] rounded-xl border border-slate-200 p-4 font-mono text-xs leading-6" />
          </label>
        </div>
        <div class="mt-5 flex justify-end">
          <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving || creating" @click="savePricing">
            <Send class="h-4 w-4" />
            保存结构与价格
          </button>
        </div>
      </div>

      <div v-if="schemas" class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <h3 class="mb-4 text-xl font-black">Schema</h3>
        <pre class="max-h-[520px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(schemas, null, 2) }}</pre>
      </div>

      <div v-if="selected" class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <h3 class="mb-4 text-xl font-black">完整详情</h3>
        <pre class="max-h-[520px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selected, null, 2) }}</pre>
      </div>
    </section>
  </section>
</template>
