<script setup lang="ts">
import { ArrowLeft, Copy, Loader2, Plus, RefreshCw, Save, Send, Trash2 } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { badgeClass, pickFirst } from "@/lib/status"

type PipelineForm = {
  name: string
  category_tips: string
  respath: string
  structure_json: string
}

const emptyForm: PipelineForm = {
  name: "",
  category_tips: "",
  respath: "",
  structure_json: "{\n  \"unlock_quals\": [],\n  \"certs\": [],\n  \"certs_quals\": [],\n  \"stages\": []\n}",
}

const pipelines = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const form = ref<PipelineForm>({ ...emptyForm })
const loading = ref(false)
const saving = ref(false)
const creating = ref(false)
const categoryFilter = ref("")
const onlyCurrent = ref(false)
const offset = ref(0)
const limit = 20

const canPrev = computed(() => offset.value > 0)
const canNext = computed(() => pipelines.value.length >= limit)
const inEditor = computed(() => !!selected.value || creating.value)
const selectedId = computed(() => selected.value ? pipelineUlid(selected.value) : "")
const published = computed(() => {
  const status = String(selected.value?.status || "").toLowerCase()
  return status === "active" || (selected.value?.is_current && status !== "deprecated")
})

function pipelineUlid(pipeline: JsonRecord) {
  return String(pickFirst(pipeline, ["pipeline_ulid", "pipeline_id"]) || "")
}

function pipelineName(pipeline: JsonRecord) {
  return String(pickFirst(pipeline, ["name", "title"]) || "未命名管线")
}

function pipelineStatus(pipeline: JsonRecord) {
  return pickFirst(pipeline, ["status", "raw_status"])
}

function structureFromPipeline(pipeline: JsonRecord | null) {
  if (!pipeline) return JSON.parse(emptyForm.structure_json)
  return {
    unlock_quals: Array.isArray(pipeline.unlock_quals) ? pipeline.unlock_quals : [],
    certs: Array.isArray(pipeline.certs) ? pipeline.certs : [],
    certs_quals: Array.isArray(pipeline.certs_quals) ? pipeline.certs_quals : [],
    stages: Array.isArray(pipeline.stages) ? pipeline.stages : [],
  }
}

function formFromPipeline(pipeline: JsonRecord | null): PipelineForm {
  if (!pipeline) return { ...emptyForm }
  return {
    name: String(pipeline.name || ""),
    category_tips: String(pipeline.category_tips || ""),
    respath: String(pipeline.respath || pipeline.pipeline_gpath || ""),
    structure_json: JSON.stringify(structureFromPipeline(pipeline), null, 2),
  }
}

function parseStructure() {
  try {
    const parsed = JSON.parse(form.value.structure_json || "{}")
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      toast.error("structure_json 必须是对象")
      return null
    }
    return parsed as JsonRecord
  } catch {
    toast.error("structure_json 不是合法 JSON")
    return null
  }
}

async function load() {
  loading.value = true
  try {
    const params = new URLSearchParams({
      limit: String(limit),
      offset: String(offset.value),
    })
    if (categoryFilter.value.trim()) params.set("category_tips", categoryFilter.value.trim())
    if (onlyCurrent.value) params.set("only_current", "true")
    const data = await apiClient<JsonRecord>(`/api/pipelines?${params}`)
    const list = Array.isArray(data.pipelines) ? data.pipelines : []
    pipelines.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
  } catch (err) {
    console.error(err)
    pipelines.value = []
    toast.error("管线配置加载失败")
  } finally {
    loading.value = false
  }
}

async function selectPipeline(pipeline: JsonRecord) {
  const id = pipelineUlid(pipeline)
  selected.value = pipeline
  creating.value = false
  form.value = formFromPipeline(pipeline)
  if (!id) return
  try {
    const detail = await apiClient<JsonRecord>(`/api/pipelines/${encodeURIComponent(id)}`)
    selected.value = detail
    form.value = formFromPipeline(detail)
  } catch {
    form.value = formFromPipeline(pipeline)
  }
}

function newPipeline() {
  selected.value = null
  creating.value = true
  form.value = { ...emptyForm }
}

function back() {
  selected.value = null
  creating.value = false
  form.value = { ...emptyForm }
}

async function createPipeline() {
  if (!form.value.name.trim() || !form.value.category_tips.trim() || !form.value.respath.trim()) {
    toast.error("请填写名称、分类提示和 respath")
    return
  }
  saving.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/pipelines", {
      method: "POST",
      body: JSON.stringify({
        name: form.value.name.trim(),
        category_tips: form.value.category_tips.trim(),
        respath: form.value.respath.trim(),
      }),
    })
    toast.success("管线草稿已创建")
    creating.value = false
    await load()
    const id = pipelineUlid(data)
    const created = pipelines.value.find((item) => pipelineUlid(item) === id)
    if (created) await selectPipeline(created)
  } catch (err) {
    console.error(err)
    toast.error("创建失败")
  } finally {
    saving.value = false
  }
}

async function saveMetadata() {
  if (!selectedId.value) return
  if (!form.value.name.trim()) {
    toast.error("请填写名称")
    return
  }
  saving.value = true
  try {
    await apiClient(`/api/pipelines/${encodeURIComponent(selectedId.value)}/metadata`, {
      method: "PUT",
      body: JSON.stringify({ new_name: form.value.name.trim() }),
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

async function saveStructure() {
  if (!selectedId.value) return
  if (published.value) {
    toast.error("已发布管线不建议直接修改结构，请先创建新版本")
    return
  }
  const structure = parseStructure()
  if (!structure) return
  saving.value = true
  try {
    const data = await apiClient<JsonRecord>(`/api/pipelines/${encodeURIComponent(selectedId.value)}/structure`, {
      method: "PUT",
      body: JSON.stringify(structure),
    })
    toast.success("管线结构已保存")
    selected.value = data
    form.value = formFromPipeline(data)
    await load()
  } catch (err) {
    console.error(err)
    toast.error("结构保存失败")
  } finally {
    saving.value = false
  }
}

async function publish() {
  if (!selectedId.value) return
  const structure = parseStructure()
  if (!structure) return
  saving.value = true
  try {
    await apiClient(`/api/pipelines/${encodeURIComponent(selectedId.value)}/publish`, {
      method: "POST",
      body: JSON.stringify({}),
    })
    toast.success("管线已发布")
    await load()
  } catch (err) {
    console.error(err)
    toast.error("发布失败")
  } finally {
    saving.value = false
  }
}

async function deprecate() {
  if (!selectedId.value) return
  if (!window.confirm("确认下架这个管线版本？")) return
  await apiClient(`/api/pipelines/${encodeURIComponent(selectedId.value)}/deprecate`, {
    method: "POST",
    body: JSON.stringify({}),
  })
  toast.success("管线已下架")
  await load()
}

async function removePipeline() {
  if (!selectedId.value) return
  if (published.value) {
    toast.error("已发布管线不能删除，请先下架或创建新版本")
    return
  }
  if (!window.confirm("确认删除这个管线草稿？")) return
  await apiClient(`/api/pipelines/${encodeURIComponent(selectedId.value)}`, { method: "DELETE" })
  toast.success("管线已删除")
  back()
  await load()
}

async function clonePipeline() {
  if (!selected.value) return
  const id = selectedId.value
  if (!id) return
  if (!form.value.respath.trim()) {
    toast.error("请填写 respath")
    return
  }
  saving.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/pipelines", {
      method: "POST",
      body: JSON.stringify({
        name: `${form.value.name.trim()} (Copy)`,
        category_tips: form.value.category_tips.trim(),
        respath: form.value.respath.trim(),
        from_pipeline_guid: selected.value.pipeline_guid,
        from_pipeline_id: id,
      }),
    })
    toast.success("管线副本已创建")
    await load()
    const newId = pipelineUlid(data)
    const created = pipelines.value.find((item) => pipelineUlid(item) === newId)
    if (created) await selectPipeline(created)
  } catch (err) {
    console.error(err)
    toast.error("克隆失败")
  } finally {
    saving.value = false
  }
}

watch([categoryFilter, onlyCurrent, offset], () => load())
onMounted(load)
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">管线配置</h1>
        <p class="mt-2 text-slate-600">维护认证管线、阶段、课程单元、证书和资格要求。</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button v-if="inEditor" class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="back">
          <ArrowLeft class="h-4 w-4" />
          返回列表
        </button>
        <button v-else class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-4 py-3 text-sm font-bold text-white shadow-sm" type="button" @click="newPipeline">
          <Plus class="h-4 w-4" />
          新建管线
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
          <h2 class="text-xl font-black">管线列表</h2>
          <p class="mt-1 text-sm text-slate-500">列表隐藏 ULID，进入详情后可查看和编辑完整配置。</p>
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <input v-model="categoryFilter" class="rounded-xl border border-slate-200 px-4 py-3" placeholder="分类提示，例如 CFtP/CFtP" />
          <label class="inline-flex items-center gap-2 rounded-xl border border-slate-200 px-4 py-3 text-sm font-bold">
            <input v-model="onlyCurrent" type="checkbox" />
            仅当前版本
          </label>
        </div>
      </div>
      <div v-if="loading" class="p-12 text-center text-slate-500">
        <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
        正在加载...
      </div>
      <button
        v-for="pipeline in pipelines"
        v-else
        :key="pipelineUlid(pipeline)"
        class="grid w-full grid-cols-[1fr_auto] gap-4 border-b border-slate-100 px-5 py-5 text-left last:border-b-0 hover:bg-sky-50"
        type="button"
        @click="selectPipeline(pipeline)"
      >
        <div>
          <div class="text-lg font-black">{{ pipelineName(pipeline) }}</div>
          <div class="mt-1 line-clamp-2 text-sm text-slate-500">{{ pipeline.description || "暂无描述" }}</div>
          <div class="mt-2 text-sm text-slate-600">{{ pipeline.category_tips || "-" }}</div>
        </div>
        <div class="text-right">
          <span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(pipelineStatus(pipeline))">{{ pipelineStatus(pipeline) || "-" }}</span>
          <div class="mt-3 text-xs text-slate-500">v{{ pipeline.version || 0 }}</div>
          <div class="mt-1 text-xs text-slate-400">{{ formatDate(String(pipeline.updated_at || pipeline.created_at || "")) }}</div>
        </div>
      </button>
      <div v-if="!loading && !pipelines.length" class="p-12 text-center text-slate-500">暂无管线</div>
      <div class="flex justify-end gap-3 border-t border-slate-200 p-5">
        <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrev" @click="offset = Math.max(0, offset - limit)">上一页</button>
        <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext" @click="offset += limit">下一页</button>
      </div>
    </section>

    <section v-else class="grid gap-6">
      <div class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <div class="mb-5 flex flex-wrap items-center justify-between gap-4">
          <div>
            <h2 class="text-2xl font-black">{{ creating ? "新建管线" : form.name || "管线详情" }}</h2>
            <p class="mt-1 text-sm text-slate-500">基础信息、发布状态和版本操作。</p>
          </div>
          <div v-if="!creating" class="flex flex-wrap gap-2">
            <button class="inline-flex items-center gap-2 rounded-xl border px-4 py-2 font-bold" type="button" @click="clonePipeline">
              <Copy class="h-4 w-4" />
              克隆版本
            </button>
            <button class="rounded-xl border px-4 py-2 font-bold" type="button" @click="publish">发布</button>
            <button class="rounded-xl border px-4 py-2 font-bold" type="button" @click="deprecate">下架</button>
            <button class="inline-flex items-center gap-2 rounded-xl bg-red-600 px-4 py-2 font-bold text-white" type="button" @click="removePipeline">
              <Trash2 class="h-4 w-4" />
              删除
            </button>
          </div>
        </div>
        <div class="grid gap-4 md:grid-cols-2">
          <label class="grid gap-2 text-sm font-bold">
            名称
            <input v-model="form.name" class="rounded-xl border border-slate-200 px-4 py-3" />
          </label>
          <label class="grid gap-2 text-sm font-bold">
            分类提示
            <input v-model="form.category_tips" :disabled="!creating" class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-50" />
          </label>
          <label class="grid gap-2 text-sm font-bold md:col-span-2">
            Respath
            <input v-model="form.respath" :disabled="!creating" class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-50" />
          </label>
        </div>
        <div class="mt-5 flex justify-end gap-3">
          <button v-if="creating" class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving" @click="createPipeline">
            <Plus class="h-4 w-4" />
            创建草稿
          </button>
          <button v-else class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving" @click="saveMetadata">
            <Save class="h-4 w-4" />
            保存基础信息
          </button>
        </div>
      </div>

      <div class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <div class="mb-4">
          <h3 class="text-xl font-black">管线结构 JSON</h3>
          <p class="mt-1 text-sm text-slate-500">结构字段会直接提交到 `/api/pipelines/{id}/structure`，用于适配微服务最新 ULID 字段。</p>
        </div>
        <textarea v-model="form.structure_json" class="min-h-[560px] w-full rounded-xl border border-slate-200 p-4 font-mono text-xs leading-6" />
        <div class="mt-5 flex justify-end">
          <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving || creating" @click="saveStructure">
            <Send class="h-4 w-4" />
            保存结构
          </button>
        </div>
      </div>

      <div v-if="selected" class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <h3 class="mb-4 text-xl font-black">完整详情</h3>
        <pre class="max-h-[520px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selected, null, 2) }}</pre>
      </div>
    </section>
  </section>
</template>
