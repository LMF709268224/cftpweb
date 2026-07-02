<script setup lang="ts">
import { Copy, Loader2, Plus, RefreshCw, Save, Send, Trash2, X } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
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
const { t } = useAdminLanguage()
const copy = computed(() => t.value.pipelineConfigAdmin)

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
  { key: "overview" as const, title: copy.value.layers.overview.title, desc: copy.value.layers.overview.desc, count: selected.value ? 1 : 0 },
  { key: "stages" as const, title: copy.value.layers.stages.title, desc: copy.value.layers.stages.desc, count: stages.value.length },
  { key: "units" as const, title: copy.value.layers.units.title, desc: copy.value.layers.units.desc, count: units.value.length },
  { key: "certs" as const, title: copy.value.layers.certs.title, desc: copy.value.layers.certs.desc, count: certs.value.length },
  { key: "unlock_quals" as const, title: copy.value.layers.unlockQuals.title, desc: copy.value.layers.unlockQuals.desc, count: unlockQuals.value.length },
  { key: "certs_quals" as const, title: copy.value.layers.certQuals.title, desc: copy.value.layers.certQuals.desc, count: certQuals.value.length },
  { key: "raw" as const, title: copy.value.layers.raw.title, desc: copy.value.layers.raw.desc, count: 1 },
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
  return String(pickFirst(pipeline, ["name", "title"]) || copy.value.unnamedPipeline)
}

function pipelineStatus(pipeline: JsonRecord) {
  return pickFirst(pipeline, ["status", "raw_status"])
}

function pipelineStatusLabel(value: unknown) {
  const raw = String(value || "").trim()
  const normalized = raw.toUpperCase().replace(/^PIPELINE_STATUS_/, "")
  const labels: Record<string, string> = {
    ACTIVE: copy.value.status.active,
    PUBLISHED: copy.value.status.published,
    DRAFT: copy.value.status.draft,
    DEPRECATED: copy.value.status.deprecated,
    INACTIVE: copy.value.status.inactive,
    ARCHIVED: copy.value.status.archived,
    PENDING: copy.value.status.pending,
    PENDING_CREATE: copy.value.status.pendingCreate,
    COMPLETED: copy.value.status.completed,
    CANCELLED: copy.value.status.cancelled,
    FAILED: copy.value.status.failed,
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
      toast.error(copy.value.toasts.structureMustObject)
      return null
    }
    return parsed as JsonRecord
  } catch {
    toast.error(copy.value.toasts.structureInvalidJson)
    return null
  }
}

function applyRawStructure() {
  const parsed = parseStructure()
  if (!parsed) return
  setStructure(parsed)
  toast.success(copy.value.toasts.rawApplied)
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
    toast.error(copy.value.toasts.fieldInvalidJson(key))
  }
}

function addStage() {
  if (isStructureLocked()) return
  const list = asMutableArray(structure.value, "stages")
  list.push({ stage_ulid: "", name: copy.value.defaults.stageName, sort_order: list.length + 1, units: [] })
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
  list.push({ unit_ulid: "", name: copy.value.defaults.unitName, sort_order: list.length + 1 })
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
  addGenericItem("certs", { qual_ulid: "", name_hint: copy.value.defaults.certificateName, pdf_template_ulid: "" }, "certs")
  selectedCertIndex.value = certs.value.length - 1
}

function addUnlockQual() {
  addGenericItem("unlock_quals", { qual_ulid: "", name_hint: copy.value.defaults.unlockQualName }, "unlock_quals")
  selectedUnlockQualIndex.value = unlockQuals.value.length - 1
}

function addCertQual() {
  addGenericItem("certs_quals", { qual_ulid: "", name_hint: copy.value.defaults.certQualName }, "certs_quals")
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
    toast.error(copy.value.toasts.loadFailed)
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
    toast.error(copy.value.toasts.requiredCreateFields)
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
    toast.success(copy.value.toasts.created)
    creating.value = false
    await load()
    const id = pipelineUlid(data)
    const created = pipelines.value.find((item) => pipelineUlid(item) === id)
    if (created) await selectPipeline(created)
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.createFailed)
  } finally {
    saving.value = false
  }
}

async function saveMetadata() {
  if (!selectedId.value) return
  if (!form.value.name.trim()) {
    toast.error(copy.value.toasts.nameRequired)
    return
  }
  saving.value = true
  try {
    await apiClient(`/api/pipelines/${encodeURIComponent(selectedId.value)}/metadata`, {
      method: "PUT",
      body: JSON.stringify({ new_name: form.value.name.trim() }),
    })
    toast.success(copy.value.toasts.metadataSaved)
    await load()
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.saveFailed)
  } finally {
    saving.value = false
  }
}

async function saveStructure() {
  if (!selectedId.value) return
  if (published.value) {
    toast.error(copy.value.toasts.publishedStructureLocked)
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
    toast.success(copy.value.toasts.structureSaved)
    selected.value = data
    form.value = formFromPipeline(data)
    setStructure(structureFromPipeline(data))
    await load()
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.structureSaveFailed)
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
    toast.success(copy.value.toasts.published)
    await load()
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.publishFailed)
  } finally {
    saving.value = false
  }
}

async function deprecate() {
  if (!selectedId.value) return
  if (!window.confirm(copy.value.confirmDeprecate)) return
  await apiClient(`/api/pipelines/${encodeURIComponent(selectedId.value)}/deprecate`, {
    method: "POST",
    body: JSON.stringify({}),
  })
  toast.success(copy.value.toasts.deprecated)
  await load()
}

async function removePipeline() {
  if (!selectedId.value) return
  if (published.value) {
    toast.error(copy.value.toasts.publishedDeleteBlocked)
    return
  }
  if (!window.confirm(copy.value.confirmDeleteDraft)) return
  await apiClient(`/api/pipelines/${encodeURIComponent(selectedId.value)}`, { method: "DELETE" })
  toast.success(copy.value.toasts.deleted)
  back()
  await load()
}

async function clonePipeline() {
  if (!selected.value) return
  const id = selectedId.value
  if (!id) return
  if (!form.value.respath.trim()) {
    toast.error(copy.value.toasts.respathRequired)
    return
  }
  saving.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/pipelines", {
      method: "POST",
      body: JSON.stringify({
        name: copy.value.copyName(form.value.name.trim()),
        category_tips: form.value.category_tips.trim(),
        respath: form.value.respath.trim(),
        from_pipeline_guid: selected.value.pipeline_guid,
        from_pipeline_id: id,
      }),
    })
    toast.success(copy.value.toasts.cloned)
    await load()
    const newId = pipelineUlid(data)
    const created = pipelines.value.find((item) => pipelineUlid(item) === newId)
    if (created) await selectPipeline(created)
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.cloneFailed)
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
        <h1 class="text-4xl font-black tracking-tight">{{ copy.title }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button v-if="!creating" class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-4 py-3 text-sm font-bold text-white shadow-sm" type="button" @click="newPipeline">
          <Plus class="h-4 w-4" />
          {{ copy.newPipeline }}
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load">
          <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
          {{ copy.refresh }}
        </button>
      </div>
    </header>

    <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 px-5 py-4">
        <div>
          <h2 class="text-xl font-black">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <input v-model="categoryFilter" class="h-10 rounded-xl border border-slate-200 px-3 text-sm" :placeholder="copy.categoryPlaceholder" />
          <label class="inline-flex h-10 items-center gap-2 rounded-xl border border-slate-200 px-3 text-sm font-bold">
            <input v-model="onlyCurrent" type="checkbox" />
            {{ copy.onlyCurrent }}
          </label>
        </div>
      </div>
      <div v-if="loading" class="px-6 py-10 text-center text-slate-500">
        <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
        {{ copy.loading }}
      </div>
      <div v-else-if="!pipelines.length" class="px-6 py-10 text-center text-slate-500">{{ copy.empty }}</div>
      <template v-else>
        <div class="grid grid-cols-[minmax(0,1fr)_150px_72px_170px_112px] gap-4 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-500">
          <span>{{ copy.columns.pipeline }}</span>
          <span class="text-center">{{ copy.columns.status }}</span>
          <span class="text-center">{{ copy.columns.version }}</span>
          <span class="text-right">{{ copy.columns.updatedAt }}</span>
          <span class="text-right">{{ copy.columns.action }}</span>
        </div>
        <div
          v-for="pipeline in pipelines"
          :key="pipelineUlid(pipeline)"
          class="grid w-full cursor-pointer grid-cols-[minmax(0,1fr)_150px_72px_170px_112px] gap-4 border-b border-slate-100 px-5 py-4 text-left transition last:border-b-0 hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-200"
          role="button"
          tabindex="0"
          @click="selectPipeline(pipeline)"
          @keydown.enter.prevent="selectPipeline(pipeline)"
          @keydown.space.prevent="selectPipeline(pipeline)"
        >
          <div class="min-w-0">
            <div class="truncate text-lg font-black">{{ pipelineName(pipeline) }}</div>
            <div class="mt-1 line-clamp-1 text-sm text-slate-500">{{ pipeline.description || copy.noDescription }}</div>
            <div class="mt-2 flex flex-wrap items-center gap-2 text-xs font-semibold text-slate-500">
              <span class="rounded-full bg-slate-100 px-2 py-1 text-slate-600">{{ pipeline.category_tips || "-" }}</span>
              <span class="break-all text-slate-400">ID: {{ pipelineUlid(pipeline) || "-" }}</span>
            </div>
          </div>
          <span class="self-center justify-self-center whitespace-nowrap rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(pipelineStatus(pipeline))">{{ pipelineStatusLabel(pipelineStatus(pipeline)) }}</span>
          <span class="self-center text-center text-sm font-black text-slate-700">v{{ pipeline.version || 0 }}</span>
          <span class="self-center justify-self-end text-sm font-semibold text-slate-500">{{ formatDate(String(pipeline.updated_at || pipeline.created_at || "")) }}</span>
          <button class="inline-flex h-9 items-center justify-self-end rounded-xl border border-slate-200 bg-white px-3 text-sm font-bold text-blue-700 shadow-sm transition hover:border-blue-200 hover:bg-blue-50" type="button" @click.stop="selectPipeline(pipeline)">
            {{ copy.viewDetails }}
          </button>
        </div>
      </template>
      <div class="flex justify-end gap-3 border-t border-slate-200 px-5 py-4">
        <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrev" @click="offset = Math.max(0, offset - limit)">{{ copy.prev }}</button>
        <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext" @click="offset += limit">{{ copy.next }}</button>
      </div>
    </section>

    <Teleport to="body">
      <div v-if="inEditor" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/45 p-6">
        <div class="flex max-h-[88vh] w-full max-w-[1320px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-6 py-5">
            <div class="min-w-0">
              <h2 class="truncate text-2xl font-black">{{ creating ? copy.newPipeline : form.name || copy.detailTitle }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ copy.detailDescription }}</p>
            </div>
            <button class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="back">
              <X class="h-5 w-5" />
            </button>
          </div>

          <section class="grid gap-6 overflow-y-auto p-6">
      <div class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <div class="mb-5 flex flex-wrap items-center justify-between gap-4">
          <div>
            <h2 class="text-2xl font-black">{{ copy.basicTitle }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.basicDescription }}</p>
          </div>
          <div v-if="!creating" class="flex flex-wrap gap-2">
            <button class="inline-flex items-center gap-2 rounded-xl border px-4 py-2 font-bold" type="button" @click="clonePipeline">
              <Copy class="h-4 w-4" />
              {{ copy.cloneVersion }}
            </button>
            <button class="rounded-xl border px-4 py-2 font-bold" type="button" @click="publish">{{ copy.publish }}</button>
            <button class="rounded-xl border px-4 py-2 font-bold" type="button" @click="deprecate">{{ copy.deprecate }}</button>
            <button class="inline-flex items-center gap-2 rounded-xl bg-red-600 px-4 py-2 font-bold text-white" type="button" @click="removePipeline">
              <Trash2 class="h-4 w-4" />
              {{ copy.delete }}
            </button>
          </div>
        </div>
        <div class="grid gap-4 md:grid-cols-2">
          <label class="grid gap-2 text-sm font-bold">
            {{ copy.fields.name }}
            <input v-model="form.name" class="rounded-xl border border-slate-200 px-4 py-3" />
          </label>
          <label class="grid gap-2 text-sm font-bold">
            {{ copy.fields.categoryTips }}
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
            {{ copy.createDraft }}
          </button>
          <button v-else class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving" @click="saveMetadata">
            <Save class="h-4 w-4" />
            {{ copy.saveBasic }}
          </button>
        </div>
      </div>

      <div v-if="!creating" class="grid gap-6 xl:grid-cols-[360px_minmax(0,1fr)]">
        <aside class="space-y-4">
          <div class="rounded-3xl border border-slate-200 bg-white p-4 shadow-sm">
            <h3 class="text-lg font-black">{{ copy.layerTitle }}</h3>
            <p class="mt-1 text-sm text-slate-500">{{ copy.layerDescription }}</p>
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
            {{ copy.structureLockedHint }}
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
              <div class="text-xs font-black uppercase text-slate-400">{{ copy.fields.status }}</div>
              <div class="mt-2"><span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(pipelineStatus(selected || {}))">{{ pipelineStatusLabel(pipelineStatus(selected || {})) }}</span></div>
            </div>
            <div class="rounded-2xl border border-slate-100 bg-slate-50 p-4">
              <div class="text-xs font-black uppercase text-slate-400">{{ copy.fields.version }}</div>
              <div class="mt-2 text-sm font-bold text-slate-950">v{{ selected?.version || 0 }}</div>
            </div>
            <div class="rounded-2xl border border-slate-100 bg-slate-50 p-4">
              <div class="text-xs font-black uppercase text-slate-400">{{ copy.fields.createdAt }}</div>
              <div class="mt-2 text-sm font-bold text-slate-950">{{ formatDate(String(selected?.created_at || "")) }}</div>
            </div>
            <div class="rounded-2xl border border-slate-100 bg-slate-50 p-4">
              <div class="text-xs font-black uppercase text-slate-400">{{ copy.fields.updatedAt }}</div>
              <div class="mt-2 text-sm font-bold text-slate-950">{{ formatDate(String(selected?.updated_at || "")) }}</div>
            </div>
          </div>

          <div v-else-if="activeLayer === 'stages'" class="grid min-h-[620px] lg:grid-cols-[320px_minmax(0,1fr)]">
            <div class="border-b border-slate-200 lg:border-b-0 lg:border-r">
              <div class="flex items-center justify-between gap-3 border-b border-slate-200 p-4">
                <div>
                  <div class="font-black">{{ copy.stageListTitle }}</div>
                  <div class="text-xs text-slate-500">{{ copy.stageListDescription }}</div>
                </div>
                <button class="rounded-xl border px-3 py-2 text-sm font-bold disabled:opacity-40" type="button" :disabled="isStructureLocked()" @click="addStage">{{ copy.newStage }}</button>
              </div>
              <button
                v-for="(stage, index) in stages"
                :key="`${itemId(stage, ['stage_ulid'])}-${index}`"
                class="w-full border-b border-slate-100 p-4 text-left hover:bg-sky-50"
                :class="selectedStageIndex === index ? 'bg-sky-50' : ''"
                type="button"
                @click="selectedStageIndex = index"
              >
                <div class="font-black">{{ itemTitle(stage, copy.stageFallback(index + 1)) }}</div>
                <div class="mt-1 text-sm text-slate-500">{{ copy.stageMeta(stage.sort_order ?? index + 1, asArray(stage.units).length) }}</div>
                <div class="mt-2 break-all text-xs font-semibold text-slate-500">ID: {{ itemId(stage, ['stage_ulid']) || "-" }}</div>
              </button>
              <div v-if="!stages.length" class="p-8 text-center text-sm text-slate-500">{{ copy.noStages }}</div>
            </div>
            <div class="space-y-5 p-5">
              <template v-if="selectedStage">
                <div class="flex items-center justify-between gap-3">
                  <div>
                    <h4 class="text-lg font-black">{{ copy.stageDetailTitle }}</h4>
                    <p class="text-sm text-slate-500">{{ copy.stageDetailDescription }}</p>
                  </div>
                  <button class="rounded-xl border border-red-200 px-4 py-2 font-bold text-red-600 disabled:opacity-40" type="button" :disabled="isStructureLocked()" @click="removeStage()">{{ copy.deleteStage }}</button>
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="grid gap-2 text-sm font-bold">
                    Stage ID
                    <input :value="fieldValue(selectedStage, 'stage_ulid')" disabled class="rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-500" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    {{ copy.fields.sortOrder }}
                    <input :value="numberValue(selectedStage, 'sort_order')" :disabled="isStructureLocked()" type="number" class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100 disabled:text-slate-500" @input="setField(selectedStage, 'sort_order', eventNumber($event))" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold md:col-span-2">
                    {{ copy.fields.name }}
                    <input :value="fieldValue(selectedStage, 'name')" :disabled="isStructureLocked()" class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100 disabled:text-slate-500" @input="setField(selectedStage, 'name', eventValue($event))" />
                  </label>
                </div>
                <pre class="max-h-[360px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selectedStage, null, 2) }}</pre>
              </template>
              <div v-else class="p-12 text-center text-slate-500">{{ copy.selectOrAddStage }}</div>
            </div>
          </div>

          <div v-else-if="activeLayer === 'units'" class="grid min-h-[620px] lg:grid-cols-[360px_minmax(0,1fr)]">
            <div class="border-b border-slate-200 lg:border-b-0 lg:border-r">
              <div class="flex items-center justify-between gap-3 border-b border-slate-200 p-4">
                <div>
                  <div class="font-black">{{ copy.unitListTitle }}</div>
                  <div class="text-xs text-slate-500">{{ copy.unitListDescription }}</div>
                </div>
                <button class="rounded-xl border px-3 py-2 text-sm font-bold disabled:opacity-40" type="button" :disabled="isStructureLocked()" @click="addUnit()">{{ copy.newUnit }}</button>
              </div>
              <button
                v-for="item in units"
                :key="item.path"
                class="w-full border-b border-slate-100 p-4 text-left hover:bg-sky-50"
                :class="selectedUnitPath === item.path ? 'bg-sky-50' : ''"
                type="button"
                @click="selectedUnitPath = item.path"
              >
                <div class="font-black">{{ itemTitle(item.unit, copy.unitFallback(item.unitIndex + 1)) }}</div>
                <div class="mt-1 text-sm text-slate-500">{{ copy.parentStagePrefix }}{{ itemTitle(item.stage, copy.stageFallback(item.stageIndex + 1)) }}</div>
                <div class="mt-2 break-all text-xs font-semibold text-slate-500">ID: {{ itemId(item.unit, ['unit_ulid']) || "-" }}</div>
              </button>
              <div v-if="!units.length" class="p-8 text-center text-sm text-slate-500">{{ copy.noUnits }}</div>
            </div>
            <div class="space-y-5 p-5">
              <template v-if="selectedUnitItem">
                <div class="flex items-center justify-between gap-3">
                  <div>
                    <h4 class="text-lg font-black">{{ copy.unitDetailTitle }}</h4>
                    <p class="text-sm text-slate-500">{{ copy.unitDetailDescription }}</p>
                  </div>
                  <button class="rounded-xl border border-red-200 px-4 py-2 font-bold text-red-600 disabled:opacity-40" type="button" :disabled="isStructureLocked()" @click="removeSelectedUnit">{{ copy.deleteUnit }}</button>
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="grid gap-2 text-sm font-bold">
                    {{ copy.fields.parentStage }}
                    <select :value="selectedUnitItem.stageIndex" :disabled="isStructureLocked()" class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100 disabled:text-slate-500" @change="moveSelectedUnit(eventNumber($event))">
                      <option v-for="(stage, index) in stages" :key="index" :value="index">{{ itemTitle(stage, copy.stageFallback(index + 1)) }}{{ copy.stageOptionId(itemId(stage, ['stage_ulid'])) }}</option>
                    </select>
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    Unit ID
                    <input :value="fieldValue(selectedUnitItem.unit, 'unit_ulid')" disabled class="rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-500" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    {{ copy.fields.name }}
                    <input :value="fieldValue(selectedUnitItem.unit, 'name')" :disabled="isStructureLocked()" class="rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100 disabled:text-slate-500" @input="setField(selectedUnitItem?.unit, 'name', eventValue($event))" />
                  </label>
                  <label class="grid gap-2 text-sm font-bold">
                    {{ copy.fields.sortOrder }}
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
                    {{ copy.fields.allowRetake }}
                  </label>
                  <label class="inline-flex items-center gap-2 text-sm font-bold">
                    <input :checked="boolValue(selectedUnitItem.unit, 'allow_exemption')" :disabled="isStructureLocked()" type="checkbox" @change="setField(selectedUnitItem?.unit, 'allow_exemption', eventChecked($event))" />
                    {{ copy.fields.allowExemption }}
                  </label>
                  <label class="grid gap-2 text-sm font-bold md:col-span-2">
                    exemption_quals JSON
                    <textarea :value="jsonValue(selectedUnitItem.unit, 'exemption_quals')" :disabled="isStructureLocked()" class="min-h-[110px] rounded-xl border border-slate-200 px-4 py-3 font-mono text-xs disabled:bg-slate-100 disabled:text-slate-500" @change="setJsonField(selectedUnitItem?.unit, 'exemption_quals', eventValue($event))" />
                  </label>
                </div>
                <pre class="max-h-[300px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selectedUnitItem.unit, null, 2) }}</pre>
              </template>
              <div v-else class="p-12 text-center text-slate-500">{{ copy.selectOrAddUnit }}</div>
            </div>
          </div>

          <div v-else-if="activeLayer === 'certs'" class="grid min-h-[560px] lg:grid-cols-[320px_minmax(0,1fr)]">
            <div class="border-b border-slate-200 lg:border-b-0 lg:border-r">
              <div class="flex items-center justify-between gap-3 border-b border-slate-200 p-4">
                <div>
                  <div class="font-black">{{ copy.certListTitle }}</div>
                  <div class="text-xs text-slate-500">{{ copy.certListDescription }}</div>
                </div>
                <button class="rounded-xl border px-3 py-2 text-sm font-bold disabled:opacity-40" type="button" :disabled="isStructureLocked()" @click="addCert">{{ copy.newCert }}</button>
              </div>
              <button v-for="(cert, index) in certs" :key="index" class="w-full border-b border-slate-100 p-4 text-left hover:bg-sky-50" :class="selectedCertIndex === index ? 'bg-sky-50' : ''" type="button" @click="selectedCertIndex = index">
                <div class="font-black">{{ itemTitle(cert, copy.certFallback(index + 1)) }}</div>
                <div class="mt-2 break-all text-xs font-semibold text-slate-500">{{ copy.qualIdPrefix }}{{ itemId(cert, ['qual_ulid']) || "-" }}</div>
              </button>
              <div v-if="!certs.length" class="p-8 text-center text-sm text-slate-500">{{ copy.noCerts }}</div>
            </div>
            <div class="space-y-5 p-5">
              <template v-if="selectedCert">
                <div class="flex items-center justify-between gap-3">
                  <h4 class="text-lg font-black">{{ copy.certDetailTitle }}</h4>
                  <button class="rounded-xl border border-red-200 px-4 py-2 font-bold text-red-600 disabled:opacity-40" type="button" :disabled="isStructureLocked()" @click="removeGenericItem('certs', selectedCertIndex)">{{ copy.deleteCert }}</button>
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
              <div v-else class="p-12 text-center text-slate-500">{{ copy.selectOrAddCert }}</div>
            </div>
          </div>

          <div v-else-if="activeLayer === 'unlock_quals' || activeLayer === 'certs_quals'" class="grid min-h-[560px] lg:grid-cols-[320px_minmax(0,1fr)]">
            <div class="border-b border-slate-200 lg:border-b-0 lg:border-r">
              <div class="flex items-center justify-between gap-3 border-b border-slate-200 p-4">
                <div>
                  <div class="font-black">{{ activeLayer === 'unlock_quals' ? copy.unlockQualListTitle : copy.certQualListTitle }}</div>
                  <div class="text-xs text-slate-500">{{ copy.qualListDescription }}</div>
                </div>
                <button class="rounded-xl border px-3 py-2 text-sm font-bold disabled:opacity-40" type="button" :disabled="isStructureLocked()" @click="activeLayer === 'unlock_quals' ? addUnlockQual() : addCertQual()">{{ copy.add }}</button>
              </div>
              <template v-if="activeLayer === 'unlock_quals'">
                <button v-for="(qual, index) in unlockQuals" :key="index" class="w-full border-b border-slate-100 p-4 text-left hover:bg-sky-50" :class="selectedUnlockQualIndex === index ? 'bg-sky-50' : ''" type="button" @click="selectedUnlockQualIndex = index">
                  <div class="font-black">{{ itemTitle(qual, copy.unlockQualFallback(index + 1)) }}</div>
                  <div class="mt-2 break-all text-xs font-semibold text-slate-500">ID: {{ itemId(qual, ['qual_ulid', 'qual_id']) || "-" }}</div>
                </button>
                <div v-if="!unlockQuals.length" class="p-8 text-center text-sm text-slate-500">{{ copy.noUnlockQuals }}</div>
              </template>
              <template v-else>
                <button v-for="(qual, index) in certQuals" :key="index" class="w-full border-b border-slate-100 p-4 text-left hover:bg-sky-50" :class="selectedCertQualIndex === index ? 'bg-sky-50' : ''" type="button" @click="selectedCertQualIndex = index">
                  <div class="font-black">{{ itemTitle(qual, copy.certQualFallback(index + 1)) }}</div>
                  <div class="mt-2 break-all text-xs font-semibold text-slate-500">ID: {{ itemId(qual, ['qual_ulid', 'qual_id']) || "-" }}</div>
                </button>
                <div v-if="!certQuals.length" class="p-8 text-center text-sm text-slate-500">{{ copy.noCertQuals }}</div>
              </template>
            </div>
            <div class="space-y-5 p-5">
              <template v-if="activeLayer === 'unlock_quals' ? selectedUnlockQual : selectedCertQual">
                <div class="flex items-center justify-between gap-3">
                  <h4 class="text-lg font-black">{{ copy.qualDetailTitle }}</h4>
                  <button
                    class="rounded-xl border border-red-200 px-4 py-2 font-bold text-red-600 disabled:opacity-40"
                    type="button"
                    :disabled="isStructureLocked()"
                    @click="activeLayer === 'unlock_quals' ? removeGenericItem('unlock_quals', selectedUnlockQualIndex) : removeGenericItem('certs_quals', selectedCertQualIndex)"
                  >
                    {{ copy.deleteQual }}
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
              <div v-else class="p-12 text-center text-slate-500">{{ copy.selectOrAddQual }}</div>
            </div>
          </div>

          <div v-else-if="activeLayer === 'raw'" class="space-y-5 p-5">
            <div class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
              {{ copy.rawHint }}
            </div>
            <textarea v-model="form.structure_json" :disabled="isStructureLocked()" class="min-h-[560px] w-full rounded-xl border border-slate-200 p-4 font-mono text-xs leading-6 disabled:bg-slate-100 disabled:text-slate-500" />
            <div class="flex flex-wrap justify-end gap-3">
              <button class="rounded-xl border px-5 py-3 font-bold disabled:opacity-40" type="button" :disabled="isStructureLocked()" @click="applyRawStructure">{{ copy.applyRaw }}</button>
              <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving || isStructureLocked()" @click="saveStructure">
                <Send class="h-4 w-4" />
                {{ copy.saveStructure }}
              </button>
            </div>
          </div>

          <div v-if="activeLayer !== 'raw'" class="flex justify-end gap-3 border-t border-slate-200 p-5">
            <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving || isStructureLocked()" @click="saveStructure">
              <Send class="h-4 w-4" />
              {{ copy.saveStructure }}
            </button>
          </div>
        </main>
      </div>

      <div v-if="selected" class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <h3 class="mb-4 text-xl font-black">{{ copy.fullDetails }}</h3>
        <pre class="max-h-[420px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selected, null, 2) }}</pre>
      </div>
          </section>
        </div>
      </div>
    </Teleport>
  </section>
</template>
