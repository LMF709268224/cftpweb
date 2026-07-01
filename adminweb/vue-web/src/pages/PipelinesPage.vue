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

type LayerKey = "overview" | "stages" | "units" | "certs" | "unlock_quals" | "certs_quals" | "raw"

type UnitListItem = {
  stageIndex: number
  unitIndex: number
  path: string
  stage: JsonRecord
  unit: JsonRecord
}

const emptyStructure = () => ({
  unlock_quals: [],
  certs: [],
  certs_quals: [],
  stages: [],
})

const emptyForm: PipelineForm = {
  name: "",
  category_tips: "",
  respath: "",
  structure_json: JSON.stringify(emptyStructure(), null, 2),
}

const pipelines = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const form = ref<PipelineForm>({ ...emptyForm })
const structure = ref<JsonRecord>(emptyStructure())
const loading = ref(false)
const saving = ref(false)
const creating = ref(false)
const categoryFilter = ref("")
const onlyCurrent = ref(false)
const offset = ref(0)
const activeLayer = ref<LayerKey>("overview")
const selectedStageIndex = ref(0)
const selectedUnitPath = ref("")
const selectedCertIndex = ref(0)
const selectedUnlockQualIndex = ref(0)
const selectedCertQualIndex = ref(0)
const limit = 20

const canPrev = computed(() => offset.value > 0)
const canNext = computed(() => pipelines.value.length >= limit)
const inEditor = computed(() => !!selected.value || creating.value)
const selectedId = computed(() => selected.value ? pipelineUlid(selected.value) : "")
const published = computed(() => {
  const status = String(selected.value?.status || "").toLowerCase()
  return status === "active" || (selected.value?.is_current && status !== "deprecated")
})
const structureLocked = computed(() => creating.value || published.value || !selectedId.value)

const stages = computed(() => asArray(structure.value.stages))
const certs = computed(() => asArray(structure.value.certs))
const unlockQuals = computed(() => asArray(structure.value.unlock_quals))
const certQuals = computed(() => asArray(structure.value.certs_quals))
const units = computed<UnitListItem[]>(() => {
  const list: UnitListItem[] = []
  stages.value.forEach((stage, stageIndex) => {
    asArray(stage.units).forEach((unit, unitIndex) => {
      list.push({
        stageIndex,
        unitIndex,
        path: `${stageIndex}:${unitIndex}`,
        stage,
        unit,
      })
    })
  })
  return list
})

const selectedStage = computed(() => stages.value[selectedStageIndex.value] || null)
const selectedUnitItem = computed(() => units.value.find((item) => item.path === selectedUnitPath.value) || units.value[0] || null)
const selectedCert = computed(() => certs.value[selectedCertIndex.value] || null)
const selectedUnlockQual = computed(() => unlockQuals.value[selectedUnlockQualIndex.value] || null)
const selectedCertQual = computed(() => certQuals.value[selectedCertQualIndex.value] || null)

const layerItems = computed(() => [
  { key: "overview" as const, title: "管线基础", desc: "顶层字段、版本和发布状态", count: selected.value ? 1 : 0 },
  { key: "stages" as const, title: "阶段", desc: "Stage 列表和排序", count: stages.value.length },
  { key: "units" as const, title: "课程单元", desc: "属于某个阶段的单元", count: units.value.length },
  { key: "certs" as const, title: "证书", desc: "完成后可签发的证书", count: certs.value.length },
  { key: "unlock_quals" as const, title: "解锁资格", desc: "进入管线前置资格", count: unlockQuals.value.length },
  { key: "certs_quals" as const, title: "结业资格", desc: "证书/结业资格要求", count: certQuals.value.length },
  { key: "raw" as const, title: "完整结构 JSON", desc: "高级编辑和排查", count: 1 },
])

function asArray(value: unknown): JsonRecord[] {
  return Array.isArray(value)
    ? value.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    : []
}

function isStructureLocked() {
  return Boolean(structureLocked.value)
}

function asMutableArray(parent: JsonRecord, key: string) {
  if (!Array.isArray(parent[key])) parent[key] = []
  return parent[key] as JsonRecord[]
}

function pipelineUlid(pipeline: JsonRecord) {
  return String(pickFirst(pipeline, ["pipeline_ulid", "pipeline_id"]) || "")
}

function pipelineName(pipeline: JsonRecord) {
  return String(pickFirst(pipeline, ["name", "title"]) || "未命名管线")
}

function pipelineStatus(pipeline: JsonRecord) {
  return pickFirst(pipeline, ["status", "raw_status"])
}

function pipelineStatusLabel(value: unknown) {
  const raw = String(value || "").trim()
  const normalized = raw.toUpperCase().replace(/^PIPELINE_STATUS_/, "")
  const labels: Record<string, string> = {
    ACTIVE: "已发布 / Active",
    PUBLISHED: "已发布 / Published",
    DRAFT: "草稿 / Draft",
    DEPRECATED: "已下架 / Deprecated",
    INACTIVE: "未启用 / Inactive",
    ARCHIVED: "已归档 / Archived",
    PENDING: "待处理 / Pending",
    PENDING_CREATE: "等待创建 / Pending Create",
    COMPLETED: "已完成 / Completed",
    CANCELLED: "已取消 / Cancelled",
    FAILED: "失败 / Failed",
  }
  return labels[normalized] || raw || "-"
}

function itemTitle(item: JsonRecord | null | undefined, fallback: string) {
  if (!item) return fallback
  return String(pickFirst(item, ["name", "title", "unit_name", "stage_name", "name_hint", "qual_name", "qual_ulid", "unit_ulid", "stage_ulid"]) || fallback)
}

function itemId(item: JsonRecord | null | undefined, keys: string[]) {
  if (!item) return ""
  return String(pickFirst(item, keys) || "")
}

function structureFromPipeline(pipeline: JsonRecord | null) {
  if (!pipeline) return emptyStructure()
  return {
    unlock_quals: Array.isArray(pipeline.unlock_quals) ? pipeline.unlock_quals : [],
    certs: Array.isArray(pipeline.certs) ? pipeline.certs : [],
    certs_quals: Array.isArray(pipeline.certs_quals) ? pipeline.certs_quals : [],
    stages: Array.isArray(pipeline.stages) ? pipeline.stages : [],
  }
}

function setStructure(next: JsonRecord) {
  structure.value = {
    unlock_quals: Array.isArray(next.unlock_quals) ? next.unlock_quals : [],
    certs: Array.isArray(next.certs) ? next.certs : [],
    certs_quals: Array.isArray(next.certs_quals) ? next.certs_quals : [],
    stages: Array.isArray(next.stages) ? next.stages : [],
  }
  syncStructureJson()
  ensureSelections()
}

function syncStructureJson() {
  form.value.structure_json = JSON.stringify(structure.value, null, 2)
}

function ensureSelections() {
  if (selectedStageIndex.value >= stages.value.length) selectedStageIndex.value = Math.max(0, stages.value.length - 1)
  if (!selectedUnitPath.value || !units.value.some((item) => item.path === selectedUnitPath.value)) selectedUnitPath.value = units.value[0]?.path || ""
  if (selectedCertIndex.value >= certs.value.length) selectedCertIndex.value = Math.max(0, certs.value.length - 1)
  if (selectedUnlockQualIndex.value >= unlockQuals.value.length) selectedUnlockQualIndex.value = Math.max(0, unlockQuals.value.length - 1)
  if (selectedCertQualIndex.value >= certQuals.value.length) selectedCertQualIndex.value = Math.max(0, certQuals.value.length - 1)
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

function applyRawStructure() {
  const parsed = parseStructure()
  if (!parsed) return
  setStructure(parsed)
  toast.success("已应用到分层视图，保存后才会提交微服务")
}

function eventValue(event: Event) {
  return (event.target as HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement | null)?.value || ""
}

function eventNumber(event: Event) {
  const value = Number(eventValue(event))
  return Number.isFinite(value) ? value : 0
}

function eventChecked(event: Event) {
  return Boolean((event.target as HTMLInputElement | null)?.checked)
}

function fieldValue(item: JsonRecord | null | undefined, key: string) {
  const value = item?.[key]
  return value === undefined || value === null ? "" : String(value)
}

function numberValue(item: JsonRecord | null | undefined, key: string) {
  const value = Number(item?.[key])
  return Number.isFinite(value) ? value : 0
}

function boolValue(item: JsonRecord | null | undefined, key: string) {
  return Boolean(item?.[key])
}

function jsonValue(item: JsonRecord | null | undefined, key: string) {
  const value = item?.[key]
  return JSON.stringify(value ?? [], null, 2)
}

function setField(item: JsonRecord | null | undefined, key: string, value: unknown) {
  if (!item || isStructureLocked()) return
  item[key] = value
  syncStructureJson()
}

function setJsonField(item: JsonRecord | null | undefined, key: string, value: string) {
  if (!item || isStructureLocked()) return
  try {
    item[key] = JSON.parse(value || "[]")
    syncStructureJson()
  } catch {
    toast.error(`${key} 不是合法 JSON`)
  }
}

function addStage() {
  if (isStructureLocked()) return
  const list = asMutableArray(structure.value, "stages")
  list.push({ stage_ulid: "", name: "New Stage", sort_order: list.length + 1, units: [] })
  selectedStageIndex.value = list.length - 1
  activeLayer.value = "stages"
  syncStructureJson()
}

function removeStage(index = selectedStageIndex.value) {
  if (isStructureLocked()) return
  asMutableArray(structure.value, "stages").splice(index, 1)
  ensureSelections()
  syncStructureJson()
}

function addUnit(stageIndex = selectedStageIndex.value) {
  if (isStructureLocked()) return
  if (!stages.value.length) addStage()
  const stage = stages.value[stageIndex] || stages.value[0]
  const list = asMutableArray(stage, "units")
  list.push({ unit_ulid: "", name: "New Unit", sort_order: list.length + 1 })
  selectedUnitPath.value = `${Math.max(0, stageIndex)}:${list.length - 1}`
  activeLayer.value = "units"
  syncStructureJson()
}

function removeSelectedUnit() {
  if (isStructureLocked() || !selectedUnitItem.value) return
  const item = selectedUnitItem.value
  asMutableArray(item.stage, "units").splice(item.unitIndex, 1)
  ensureSelections()
  syncStructureJson()
}

function moveSelectedUnit(targetStageIndex: number) {
  if (isStructureLocked() || !selectedUnitItem.value) return
  const item = selectedUnitItem.value
  if (targetStageIndex === item.stageIndex) return
  const targetStage = stages.value[targetStageIndex]
  if (!targetStage) return
  asMutableArray(item.stage, "units").splice(item.unitIndex, 1)
  const targetUnits = asMutableArray(targetStage, "units")
  targetUnits.push(item.unit)
  selectedUnitPath.value = `${targetStageIndex}:${targetUnits.length - 1}`
  syncStructureJson()
}

function addCert() {
  addGenericItem("certs", { qual_ulid: "", name_hint: "New Certificate", pdf_template_ulid: "" }, "certs")
  selectedCertIndex.value = certs.value.length - 1
}

function addUnlockQual() {
  addGenericItem("unlock_quals", { qual_ulid: "", name_hint: "New Unlock Qualification" }, "unlock_quals")
  selectedUnlockQualIndex.value = unlockQuals.value.length - 1
}

function addCertQual() {
  addGenericItem("certs_quals", { qual_ulid: "", name_hint: "New Completion Qualification" }, "certs_quals")
  selectedCertQualIndex.value = certQuals.value.length - 1
}

function addGenericItem(key: "certs" | "unlock_quals" | "certs_quals", value: JsonRecord, layer: LayerKey) {
  if (isStructureLocked()) return
  asMutableArray(structure.value, key).push(value)
  activeLayer.value = layer
  syncStructureJson()
}

function removeGenericItem(key: "certs" | "unlock_quals" | "certs_quals", index: number) {
  if (isStructureLocked()) return
  asMutableArray(structure.value, key).splice(index, 1)
  ensureSelections()
  syncStructureJson()
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
  setStructure(structureFromPipeline(pipeline))
  activeLayer.value = "overview"
  if (!id) return
  try {
    const detail = await apiClient<JsonRecord>(`/api/pipelines/${encodeURIComponent(id)}`)
    selected.value = detail
    form.value = formFromPipeline(detail)
    setStructure(structureFromPipeline(detail))
  } catch {
    form.value = formFromPipeline(pipeline)
    setStructure(structureFromPipeline(pipeline))
  }
}

function newPipeline() {
  selected.value = null
  creating.value = true
  form.value = { ...emptyForm }
  setStructure(emptyStructure())
  activeLayer.value = "overview"
}

function back() {
  selected.value = null
  creating.value = false
  form.value = { ...emptyForm }
  setStructure(emptyStructure())
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
  const parsed = parseStructure()
  if (!parsed) return
  saving.value = true
  try {
    const data = await apiClient<JsonRecord>(`/api/pipelines/${encodeURIComponent(selectedId.value)}/structure`, {
      method: "PUT",
      body: JSON.stringify(parsed),
    })
    toast.success("管线结构已保存")
    selected.value = data
    form.value = formFromPipeline(data)
    setStructure(structureFromPipeline(data))
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
  const parsed = parseStructure()
  if (!parsed) return
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
  <section class="mx-auto flex min-h-screen w-full max-w-[1520px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">管线配置</h1>
        <p class="mt-2 text-slate-600">按微服务接口能力维护认证管线、阶段、课程单元、证书和资格要求。</p>
        <p class="mt-2 text-xs font-semibold text-slate-500">
          已确认接口：list/get/create draft/duplicate/update metadata/update structure/publish/deprecate/delete。
        </p>
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

    <section v-if="!inEditor" class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 px-5 py-4">
        <div>
          <h2 class="text-xl font-black">管线列表</h2>
          <p class="mt-1 text-sm text-slate-500">列表来自 `/api/pipelines`，进入详情后按层级维护完整配置。</p>
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <input v-model="categoryFilter" class="h-10 rounded-xl border border-slate-200 px-3 text-sm" placeholder="分类提示，例如 CFtP/CFtP" />
          <label class="inline-flex h-10 items-center gap-2 rounded-xl border border-slate-200 px-3 text-sm font-bold">
            <input v-model="onlyCurrent" type="checkbox" />
            仅当前版本
          </label>
        </div>
      </div>
      <div v-if="loading" class="px-6 py-10 text-center text-slate-500">
        <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
        正在加载...
      </div>
      <div v-else-if="!pipelines.length" class="px-6 py-10 text-center text-slate-500">暂无管线</div>
      <template v-else>
        <div class="grid grid-cols-[minmax(0,1fr)_150px_72px_170px] gap-4 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-500">
          <span>管线</span>
          <span class="text-center">状态</span>
          <span class="text-center">版本</span>
          <span class="text-right">更新时间</span>
        </div>
        <button
          v-for="pipeline in pipelines"
          :key="pipelineUlid(pipeline)"
          class="grid w-full grid-cols-[minmax(0,1fr)_150px_72px_170px] gap-4 border-b border-slate-100 px-5 py-4 text-left transition last:border-b-0 hover:bg-slate-50"
          type="button"
          @click="selectPipeline(pipeline)"
        >
          <div class="min-w-0">
            <div class="truncate text-lg font-black">{{ pipelineName(pipeline) }}</div>
            <div class="mt-1 line-clamp-1 text-sm text-slate-500">{{ pipeline.description || "暂无描述" }}</div>
            <div class="mt-2 flex flex-wrap items-center gap-2 text-xs font-semibold text-slate-500">
              <span class="rounded-full bg-slate-100 px-2 py-1 text-slate-600">{{ pipeline.category_tips || "-" }}</span>
              <span class="break-all text-slate-400">ID: {{ pipelineUlid(pipeline) || "-" }}</span>
            </div>
          </div>
          <span class="self-center justify-self-center whitespace-nowrap rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(pipelineStatus(pipeline))">{{ pipelineStatusLabel(pipelineStatus(pipeline)) }}</span>
          <span class="self-center text-center text-sm font-black text-slate-700">v{{ pipeline.version || 0 }}</span>
          <span class="self-center justify-self-end text-sm font-semibold text-slate-500">{{ formatDate(String(pipeline.updated_at || pipeline.created_at || "")) }}</span>
        </button>
      </template>
      <div class="flex justify-end gap-3 border-t border-slate-200 px-5 py-4">
        <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrev" @click="offset = Math.max(0, offset - limit)">上一页</button>
        <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext" @click="offset += limit">下一页</button>
      </div>
    </section>

    <section v-else class="grid gap-6">
      <div class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <div class="mb-5 flex flex-wrap items-center justify-between gap-4">
          <div>
            <h2 class="text-2xl font-black">{{ creating ? "新建管线" : form.name || "管线详情" }}</h2>
            <p class="mt-1 text-sm text-slate-500">
              顶层信息由 metadata/create 接口维护；阶段、课程单元、证书和资格要求由 structure 接口整体保存。
            </p>
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
            <input v-model="form.category_tips" :disabled="!creating" class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100 disabled:text-slate-500" />
          </label>
          <label class="grid gap-2 text-sm font-bold md:col-span-2">
            Respath
            <input v-model="form.respath" :disabled="!creating" class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100 disabled:text-slate-500" />
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

      <div v-if="!creating" class="grid gap-6 xl:grid-cols-[360px_minmax(0,1fr)]">
        <aside class="space-y-4">
          <div class="rounded-3xl border border-slate-200 bg-white p-4 shadow-sm">
            <h3 class="text-lg font-black">配置层级</h3>
            <p class="mt-1 text-sm text-slate-500">左侧选择层级，右侧查看和维护详情。</p>
            <div class="mt-4 space-y-2">
              <button
                v-for="layer in layerItems"
                :key="layer.key"
                class="w-full rounded-2xl border px-4 py-3 text-left transition"
                :class="activeLayer === layer.key ? 'border-sky-200 bg-sky-50 shadow-sm' : 'border-slate-100 bg-white hover:bg-slate-50'"
                type="button"
                @click="activeLayer = layer.key"
              >
                <div class="flex items-center justify-between gap-3">
                  <span class="font-black text-slate-950">{{ layer.title }}</span>
                  <span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-black text-slate-600">{{ layer.count }}</span>
                </div>
                <p class="mt-1 text-xs text-slate-500">{{ layer.desc }}</p>
              </button>
            </div>
          </div>

          <div v-if="structureLocked" class="rounded-3xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
            当前结构不可编辑：新建时请先创建草稿；已发布版本请先克隆新版本再修改。
          </div>
        </aside>

        <main class="min-w-0 rounded-3xl border border-slate-200 bg-white shadow-sm">
          <div class="border-b border-slate-200 p-5">
            <h3 class="text-xl font-black">{{ layerItems.find((layer) => layer.key === activeLayer)?.title }}</h3>
            <p class="mt-1 text-sm text-slate-500">{{ layerItems.find((layer) => layer.key === activeLayer)?.desc }}</p>
          </div>

          <div v-if="activeLayer === 'overview'" class="grid gap-5 p-5 lg:grid-cols-2">
            <div class="rounded-2xl border border-slate-100 bg-slate-50 p-4">
              <div class="text-xs font-black uppercase text-slate-400">Pipeline ID</div>
              <div class="mt-2 break-all text-sm font-bold text-slate-950">{{ selectedId || "-" }}</div>
            </div>
            <div class="rounded-2xl border border-slate-100 bg-slate-50 p-4">
              <div class="text-xs font-black uppercase text-slate-400">Pipeline GUID</div>
              <div class="mt-2 break-all text-sm font-bold text-slate-950">{{ selected?.pipeline_guid || "-" }}</div>
            </div>
            <div class="rounded-2xl border border-slate-100 bg-slate-50 p-4">
              <div class="text-xs font-black uppercase text-slate-400">状态</div>
              <div class="mt-2"><span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(pipelineStatus(selected || {}))">{{ pipelineStatusLabel(pipelineStatus(selected || {})) }}</span></div>
            </div>
            <div class="rounded-2xl border border-slate-100 bg-slate-50 p-4">
              <div class="text-xs font-black uppercase text-slate-400">版本</div>
              <div class="mt-2 text-sm font-bold text-slate-950">v{{ selected?.version || 0 }}</div>
            </div>
            <div class="rounded-2xl border border-slate-100 bg-slate-50 p-4">
              <div class="text-xs font-black uppercase text-slate-400">创建时间</div>
              <div class="mt-2 text-sm font-bold text-slate-950">{{ formatDate(String(selected?.created_at || "")) }}</div>
            </div>
            <div class="rounded-2xl border border-slate-100 bg-slate-50 p-4">
              <div class="text-xs font-black uppercase text-slate-400">更新时间</div>
              <div class="mt-2 text-sm font-bold text-slate-950">{{ formatDate(String(selected?.updated_at || "")) }}</div>
            </div>
          </div>

          <div v-else-if="activeLayer === 'stages'" class="grid min-h-[620px] lg:grid-cols-[320px_minmax(0,1fr)]">
            <div class="border-b border-slate-200 lg:border-b-0 lg:border-r">
              <div class="flex items-center justify-between gap-3 border-b border-slate-200 p-4">
                <div>
                  <div class="font-black">阶段列表</div>
                  <div class="text-xs text-slate-500">选择阶段查看详情</div>
                </div>
                <button class="rounded-xl border px-3 py-2 text-sm font-bold disabled:opacity-40" type="button" :disabled="isStructureLocked()" @click="addStage">新阶段</button>
              </div>
              <button
                v-for="(stage, index) in stages"
                :key="`${itemId(stage, ['stage_ulid'])}-${index}`"
                class="w-full border-b border-slate-100 p-4 text-left hover:bg-sky-50"
                :class="selectedStageIndex === index ? 'bg-sky-50' : ''"
                type="button"
                @click="selectedStageIndex = index"
              >
                <div class="font-black">{{ itemTitle(stage, `阶段 ${index + 1}`) }}</div>
                <div class="mt-1 text-sm text-slate-500">排序 {{ stage.sort_order ?? index + 1 }} · 单元 {{ asArray(stage.units).length }}</div>
                <div class="mt-2 break-all text-xs font-semibold text-slate-500">ID: {{ itemId(stage, ['stage_ulid']) || "-" }}</div>
              </button>
              <div v-if="!stages.length" class="p-8 text-center text-sm text-slate-500">暂无阶段</div>
            </div>
            <div class="space-y-5 p-5">
              <template v-if="selectedStage">
                <div class="flex items-center justify-between gap-3">
                  <div>
                    <h4 class="text-lg font-black">阶段详情</h4>
                    <p class="text-sm text-slate-500">阶段只属于当前管线。</p>
                  </div>
                  <button class="rounded-xl border border-red-200 px-4 py-2 font-bold text-red-600 disabled:opacity-40" type="button" :disabled="isStructureLocked()" @click="removeStage()">删除阶段</button>
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="grid gap-2 text-sm font-bold">
                    Stage ID
                    <input :value="fieldValue(selectedStage, 'stage_ulid')" disabled class="rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-500" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    排序
                    <input :value="numberValue(selectedStage, 'sort_order')" :disabled="isStructureLocked()" type="number" class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100 disabled:text-slate-500" @input="setField(selectedStage, 'sort_order', eventNumber($event))" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold md:col-span-2">
                    名称
                    <input :value="fieldValue(selectedStage, 'name')" :disabled="isStructureLocked()" class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100 disabled:text-slate-500" @input="setField(selectedStage, 'name', eventValue($event))" />
                  </label>
                </div>
                <pre class="max-h-[360px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selectedStage, null, 2) }}</pre>
              </template>
              <div v-else class="p-12 text-center text-slate-500">请选择或新增阶段</div>
            </div>
          </div>

          <div v-else-if="activeLayer === 'units'" class="grid min-h-[620px] lg:grid-cols-[360px_minmax(0,1fr)]">
            <div class="border-b border-slate-200 lg:border-b-0 lg:border-r">
              <div class="flex items-center justify-between gap-3 border-b border-slate-200 p-4">
                <div>
                  <div class="font-black">课程单元列表</div>
                  <div class="text-xs text-slate-500">每个单元显示所属阶段</div>
                </div>
                <button class="rounded-xl border px-3 py-2 text-sm font-bold disabled:opacity-40" type="button" :disabled="isStructureLocked()" @click="addUnit()">新单元</button>
              </div>
              <button
                v-for="item in units"
                :key="item.path"
                class="w-full border-b border-slate-100 p-4 text-left hover:bg-sky-50"
                :class="selectedUnitPath === item.path ? 'bg-sky-50' : ''"
                type="button"
                @click="selectedUnitPath = item.path"
              >
                <div class="font-black">{{ itemTitle(item.unit, `课程单元 ${item.unitIndex + 1}`) }}</div>
                <div class="mt-1 text-sm text-slate-500">所属阶段：{{ itemTitle(item.stage, `阶段 ${item.stageIndex + 1}`) }}</div>
                <div class="mt-2 break-all text-xs font-semibold text-slate-500">ID: {{ itemId(item.unit, ['unit_ulid']) || "-" }}</div>
              </button>
              <div v-if="!units.length" class="p-8 text-center text-sm text-slate-500">暂无课程单元</div>
            </div>
            <div class="space-y-5 p-5">
              <template v-if="selectedUnitItem">
                <div class="flex items-center justify-between gap-3">
                  <div>
                    <h4 class="text-lg font-black">课程单元详情</h4>
                    <p class="text-sm text-slate-500">右侧可调整所属阶段，保存后通过 structure 接口提交。</p>
                  </div>
                  <button class="rounded-xl border border-red-200 px-4 py-2 font-bold text-red-600 disabled:opacity-40" type="button" :disabled="isStructureLocked()" @click="removeSelectedUnit">删除单元</button>
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="grid gap-2 text-sm font-bold">
                    所属阶段
                    <select :value="selectedUnitItem.stageIndex" :disabled="isStructureLocked()" class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100 disabled:text-slate-500" @change="moveSelectedUnit(eventNumber($event))">
                      <option v-for="(stage, index) in stages" :key="index" :value="index">{{ itemTitle(stage, `阶段 ${index + 1}`) }}（{{ itemId(stage, ['stage_ulid']) || '无 ID' }}）</option>
                    </select>
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    Unit ID
                    <input :value="fieldValue(selectedUnitItem.unit, 'unit_ulid')" disabled class="rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-500" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    名称
                    <input :value="fieldValue(selectedUnitItem.unit, 'name')" :disabled="isStructureLocked()" class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100 disabled:text-slate-500" @input="setField(selectedUnitItem?.unit, 'name', eventValue($event))" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    排序
                    <input :value="numberValue(selectedUnitItem.unit, 'sort_order')" :disabled="isStructureLocked()" type="number" class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100 disabled:text-slate-500" @input="setField(selectedUnitItem?.unit, 'sort_order', eventNumber($event))" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    GLMS Course ID
                    <input :value="fieldValue(selectedUnitItem.unit, 'glms_course_id')" :disabled="isStructureLocked()" class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100 disabled:text-slate-500" @input="setField(selectedUnitItem?.unit, 'glms_course_id', eventValue($event))" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    Program
                    <input :value="fieldValue(selectedUnitItem.unit, 'program')" :disabled="isStructureLocked()" class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100 disabled:text-slate-500" @input="setField(selectedUnitItem?.unit, 'program', eventValue($event))" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    Exam ID
                    <input :value="fieldValue(selectedUnitItem.unit, 'exam_id')" :disabled="isStructureLocked()" class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100 disabled:text-slate-500" @input="setField(selectedUnitItem?.unit, 'exam_id', eventValue($event))" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    Form Code
                    <input :value="fieldValue(selectedUnitItem.unit, 'form_code')" :disabled="isStructureLocked()" class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100 disabled:text-slate-500" @input="setField(selectedUnitItem?.unit, 'form_code', eventValue($event))" />
                  </label>
                  <label class="inline-flex items-center gap-2 text-sm font-bold">
                    <input :checked="boolValue(selectedUnitItem.unit, 'allow_retake')" :disabled="isStructureLocked()" type="checkbox" @change="setField(selectedUnitItem?.unit, 'allow_retake', eventChecked($event))" />
                    允许重考
                  </label>
                  <label class="inline-flex items-center gap-2 text-sm font-bold">
                    <input :checked="boolValue(selectedUnitItem.unit, 'allow_exemption')" :disabled="isStructureLocked()" type="checkbox" @change="setField(selectedUnitItem?.unit, 'allow_exemption', eventChecked($event))" />
                    允许免考
                  </label>
                  <label class="grid gap-2 text-sm font-bold md:col-span-2">
                    exemption_quals JSON
                    <textarea :value="jsonValue(selectedUnitItem.unit, 'exemption_quals')" :disabled="isStructureLocked()" class="min-h-[110px] rounded-xl border border-slate-200 px-4 py-3 font-mono text-xs disabled:bg-slate-100 disabled:text-slate-500" @change="setJsonField(selectedUnitItem?.unit, 'exemption_quals', eventValue($event))" />
                  </label>
                </div>
                <pre class="max-h-[300px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selectedUnitItem.unit, null, 2) }}</pre>
              </template>
              <div v-else class="p-12 text-center text-slate-500">请选择或新增课程单元</div>
            </div>
          </div>

          <div v-else-if="activeLayer === 'certs'" class="grid min-h-[560px] lg:grid-cols-[320px_minmax(0,1fr)]">
            <div class="border-b border-slate-200 lg:border-b-0 lg:border-r">
              <div class="flex items-center justify-between gap-3 border-b border-slate-200 p-4">
                <div>
                  <div class="font-black">证书列表</div>
                  <div class="text-xs text-slate-500">管线完成后签发</div>
                </div>
                <button class="rounded-xl border px-3 py-2 text-sm font-bold disabled:opacity-40" type="button" :disabled="isStructureLocked()" @click="addCert">新证书</button>
              </div>
              <button v-for="(cert, index) in certs" :key="index" class="w-full border-b border-slate-100 p-4 text-left hover:bg-sky-50" :class="selectedCertIndex === index ? 'bg-sky-50' : ''" type="button" @click="selectedCertIndex = index">
                <div class="font-black">{{ itemTitle(cert, `证书 ${index + 1}`) }}</div>
                <div class="mt-2 break-all text-xs font-semibold text-slate-500">资格 ID: {{ itemId(cert, ['qual_ulid']) || "-" }}</div>
              </button>
              <div v-if="!certs.length" class="p-8 text-center text-sm text-slate-500">暂无证书</div>
            </div>
            <div class="space-y-5 p-5">
              <template v-if="selectedCert">
                <div class="flex items-center justify-between gap-3">
                  <h4 class="text-lg font-black">证书详情</h4>
                  <button class="rounded-xl border border-red-200 px-4 py-2 font-bold text-red-600 disabled:opacity-40" type="button" :disabled="isStructureLocked()" @click="removeGenericItem('certs', selectedCertIndex)">删除证书</button>
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="grid gap-2 text-sm font-bold">
                    qual_ulid
                    <input :value="fieldValue(selectedCert, 'qual_ulid')" :disabled="isStructureLocked()" class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100 disabled:text-slate-500" @input="setField(selectedCert, 'qual_ulid', eventValue($event))" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    pdf_template_ulid
                    <input :value="fieldValue(selectedCert, 'pdf_template_ulid')" :disabled="isStructureLocked()" class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100 disabled:text-slate-500" @input="setField(selectedCert, 'pdf_template_ulid', eventValue($event))" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold md:col-span-2">
                    name_hint
                    <input :value="fieldValue(selectedCert, 'name_hint')" :disabled="isStructureLocked()" class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100 disabled:text-slate-500" @input="setField(selectedCert, 'name_hint', eventValue($event))" />
                  </label>
                </div>
                <pre class="max-h-[360px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selectedCert, null, 2) }}</pre>
              </template>
              <div v-else class="p-12 text-center text-slate-500">请选择或新增证书</div>
            </div>
          </div>

          <div v-else-if="activeLayer === 'unlock_quals' || activeLayer === 'certs_quals'" class="grid min-h-[560px] lg:grid-cols-[320px_minmax(0,1fr)]">
            <div class="border-b border-slate-200 lg:border-b-0 lg:border-r">
              <div class="flex items-center justify-between gap-3 border-b border-slate-200 p-4">
                <div>
                  <div class="font-black">{{ activeLayer === 'unlock_quals' ? '解锁资格列表' : '结业资格列表' }}</div>
                  <div class="text-xs text-slate-500">资格项来自管线结构字段</div>
                </div>
                <button class="rounded-xl border px-3 py-2 text-sm font-bold disabled:opacity-40" type="button" :disabled="isStructureLocked()" @click="activeLayer === 'unlock_quals' ? addUnlockQual() : addCertQual()">新增</button>
              </div>
              <template v-if="activeLayer === 'unlock_quals'">
                <button v-for="(qual, index) in unlockQuals" :key="index" class="w-full border-b border-slate-100 p-4 text-left hover:bg-sky-50" :class="selectedUnlockQualIndex === index ? 'bg-sky-50' : ''" type="button" @click="selectedUnlockQualIndex = index">
                  <div class="font-black">{{ itemTitle(qual, `解锁资格 ${index + 1}`) }}</div>
                  <div class="mt-2 break-all text-xs font-semibold text-slate-500">ID: {{ itemId(qual, ['qual_ulid', 'qual_id']) || "-" }}</div>
                </button>
                <div v-if="!unlockQuals.length" class="p-8 text-center text-sm text-slate-500">暂无解锁资格</div>
              </template>
              <template v-else>
                <button v-for="(qual, index) in certQuals" :key="index" class="w-full border-b border-slate-100 p-4 text-left hover:bg-sky-50" :class="selectedCertQualIndex === index ? 'bg-sky-50' : ''" type="button" @click="selectedCertQualIndex = index">
                  <div class="font-black">{{ itemTitle(qual, `结业资格 ${index + 1}`) }}</div>
                  <div class="mt-2 break-all text-xs font-semibold text-slate-500">ID: {{ itemId(qual, ['qual_ulid', 'qual_id']) || "-" }}</div>
                </button>
                <div v-if="!certQuals.length" class="p-8 text-center text-sm text-slate-500">暂无结业资格</div>
              </template>
            </div>
            <div class="space-y-5 p-5">
              <template v-if="activeLayer === 'unlock_quals' ? selectedUnlockQual : selectedCertQual">
                <div class="flex items-center justify-between gap-3">
                  <h4 class="text-lg font-black">资格详情</h4>
                  <button
                    class="rounded-xl border border-red-200 px-4 py-2 font-bold text-red-600 disabled:opacity-40"
                    type="button"
                    :disabled="isStructureLocked()"
                    @click="activeLayer === 'unlock_quals' ? removeGenericItem('unlock_quals', selectedUnlockQualIndex) : removeGenericItem('certs_quals', selectedCertQualIndex)"
                  >
                    删除资格
                  </button>
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="grid gap-2 text-sm font-bold">
                    qual_ulid / qual_id
                    <input
                      :value="fieldValue(activeLayer === 'unlock_quals' ? selectedUnlockQual : selectedCertQual, 'qual_ulid') || fieldValue(activeLayer === 'unlock_quals' ? selectedUnlockQual : selectedCertQual, 'qual_id')"
                      :disabled="isStructureLocked()"
                      class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100 disabled:text-slate-500"
                      @input="setField(activeLayer === 'unlock_quals' ? selectedUnlockQual : selectedCertQual, 'qual_ulid', eventValue($event))"
                    />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    name_hint
                    <input
                      :value="fieldValue(activeLayer === 'unlock_quals' ? selectedUnlockQual : selectedCertQual, 'name_hint')"
                      :disabled="isStructureLocked()"
                      class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100 disabled:text-slate-500"
                      @input="setField(activeLayer === 'unlock_quals' ? selectedUnlockQual : selectedCertQual, 'name_hint', eventValue($event))"
                    />
                  </label>
                </div>
                <pre class="max-h-[420px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(activeLayer === 'unlock_quals' ? selectedUnlockQual : selectedCertQual, null, 2) }}</pre>
              </template>
              <div v-else class="p-12 text-center text-slate-500">请选择或新增资格</div>
            </div>
          </div>

          <div v-else-if="activeLayer === 'raw'" class="space-y-5 p-5">
            <div class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
              高级编辑区用于排查或补充暂未单独建表单的字段。点击“应用到分层视图”只更新页面状态，点击“保存结构”才会提交微服务。
            </div>
            <textarea v-model="form.structure_json" :disabled="isStructureLocked()" class="min-h-[560px] w-full rounded-xl border border-slate-200 p-4 font-mono text-xs leading-6 disabled:bg-slate-100 disabled:text-slate-500" />
            <div class="flex flex-wrap justify-end gap-3">
              <button class="rounded-xl border px-5 py-3 font-bold disabled:opacity-40" type="button" :disabled="isStructureLocked()" @click="applyRawStructure">应用到分层视图</button>
              <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving || isStructureLocked()" @click="saveStructure">
                <Send class="h-4 w-4" />
                保存结构
              </button>
            </div>
          </div>

          <div v-if="activeLayer !== 'raw'" class="flex justify-end gap-3 border-t border-slate-200 p-5">
            <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving || isStructureLocked()" @click="saveStructure">
              <Send class="h-4 w-4" />
              保存结构
            </button>
          </div>
        </main>
      </div>

      <div v-if="selected" class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <h3 class="mb-4 text-xl font-black">完整详情</h3>
        <pre class="max-h-[420px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selected, null, 2) }}</pre>
      </div>
    </section>
  </section>
</template>
